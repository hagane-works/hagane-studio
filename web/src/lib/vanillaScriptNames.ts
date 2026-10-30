import { useEffect, useState } from "react";
import { requestGameIndexed } from "@/lib/api/gameIndex";

export interface VanillaScriptNames {
  triggers: Set<string>;
  effects: Set<string>;
  structuralFields: Set<string>;

  mathFunctions: Set<string>;
  loaded: boolean;
}

const EMPTY: VanillaScriptNames = {
  triggers: new Set(),
  effects: new Set(),
  structuralFields: new Set(),
  mathFunctions: new Set(),
  loaded: false,
};

let cache: VanillaScriptNames | null = null;
let inflight: Promise<VanillaScriptNames> | null = null;

async function fetchVanillaScriptNames(): Promise<VanillaScriptNames> {
  if (cache) return cache;
  if (inflight) return inflight;
  inflight = (async () => {
    const [triggersEffects, structural] = await Promise.all([
      requestGameIndexed<{
        triggers?: string[];
        effects?: string[];
        math_functions?: string[];
      }>("/go/vanilla/script-names").catch(() => null),
      requestGameIndexed<{ fields?: string[] }>(
        "/go/vanilla/structural-fields",
      ).catch(() => null),
    ]);
    const result: VanillaScriptNames = {
      triggers: new Set(triggersEffects?.triggers ?? []),
      effects: new Set(triggersEffects?.effects ?? []),
      structuralFields: new Set(structural?.fields ?? []),

      mathFunctions: new Set(triggersEffects?.math_functions ?? []),
      loaded: true,
    };

    if (triggersEffects && structural) cache = result;
    inflight = null;
    return result;
  })();
  return inflight;
}

export function useVanillaScriptNames(): VanillaScriptNames {
  const [data, setData] = useState<VanillaScriptNames>(() => cache ?? EMPTY);
  useEffect(() => {
    let cancelled = false;
    fetchVanillaScriptNames().then((d) => {
      if (!cancelled) setData(d);
    });
    return () => {
      cancelled = true;
    };
  }, []);
  return data;
}

function _levenshtein(a: string, b: string): number {
  if (a === b) return 0;
  if (!a.length) return b.length;
  if (!b.length) return a.length;
  const m = a.length;
  const n = b.length;

  let prev = new Array(n + 1);
  let curr = new Array(n + 1);
  for (let j = 0; j <= n; j++) prev[j] = j;
  for (let i = 1; i <= m; i++) {
    curr[0] = i;
    for (let j = 1; j <= n; j++) {
      const cost = a[i - 1] === b[j - 1] ? 0 : 1;
      curr[j] = Math.min(curr[j - 1] + 1, prev[j] + 1, prev[j - 1] + cost);
    }
    [prev, curr] = [curr, prev];
  }
  return prev[n];
}

export function suggestVanillaToken(
  token: string,
  vanilla: VanillaScriptNames,
  maxDistance = 3,
): string | null {
  let best: string | null = null;
  let bestDist = maxDistance + 1;
  const all: Iterable<string> = (function* () {
    for (const t of vanilla.triggers) yield t;
    for (const e of vanilla.effects) yield e;
  })();
  for (const candidate of all) {
    const lenDiff = Math.abs(candidate.length - token.length);
    if (lenDiff > maxDistance) continue;
    const d = _levenshtein(token, candidate);
    if (d < bestDist) {
      bestDist = d;
      best = candidate;
      if (d === 0) break;
    }
  }
  return bestDist <= maxDistance ? best : null;
}

export const SCRIPT_WRAP_KEYS = new Set<string>([
  "if",
  "else",
  "else_if",
  "limit",

  "AND",
  "OR",
  "NOT",
  "XOR",
  "and",
  "or",
  "not",
  "xor",
  "any",
  "all",
  "all_owned_state",
  "any_owned_state",
  "any_country",
  "all_country",
  "any_other_country",
  "FROM",
  "ROOT",
  "THIS",
  "PREV",
  "var",
  "value",

  "option",
  "ai_chance",
  "base",
  "modifier",
  "factor",
  "add",
  "trigger",
  "immediate",
  "after",
  "fire_only_once",
  "is_triggered_only",
  "hidden",
  "major",

  "id",
  "title",
  "desc",
  "picture",
  "name",

  "scope",
  "tag",
  "original_tag",
  "owner",
  "controller",

  "random_list",
  "random_country",
  "random_state",
  "global_event_target",
  "event_target",
]);

export interface ScriptTokenIssue {
  line: number;
  column: number;
  token: string;
  suggestion: string | null;
}

export function findUnknownScriptTokens(
  text: string,
  vanilla: VanillaScriptNames,
  customNames: ReadonlySet<string> = new Set(),
): ScriptTokenIssue[] {
  if (!vanilla.loaded) return [];
  const issues: ScriptTokenIssue[] = [];
  const lines = text.split(/\r?\n/);

  const tokenRe = /^(\s*)([A-Za-z_][A-Za-z0-9_]*)\s*=/;
  for (let i = 0; i < lines.length; i++) {
    const line = lines[i];

    const commentIdx = line.indexOf("#");
    const effective = commentIdx >= 0 ? line.slice(0, commentIdx) : line;
    const m = effective.match(tokenRe);
    if (!m) continue;
    const indent = m[1];
    const token = m[2];

    if (token.includes(".")) continue;
    if (SCRIPT_WRAP_KEYS.has(token)) continue;
    if (customNames.has(token)) continue;
    if (vanilla.triggers.has(token) || vanilla.effects.has(token)) continue;

    if (vanilla.structuralFields.has(token)) continue;

    if (vanilla.mathFunctions.has(token)) continue;

    issues.push({
      line: i + 1,
      column: indent.length + 1,
      token,
      suggestion: suggestVanillaToken(token, vanilla),
    });
  }
  return issues;
}

export const _levenshteinForTests = _levenshtein;
