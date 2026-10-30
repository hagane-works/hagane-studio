"use client";

import type { CwtoolRules } from "@/lib/api";
import type { EditorMarker } from "./ScriptEditor";

export const CWTOOL_MARKER_OWNER = "hoi4-cwtool";

type CwtoolLocale = "en" | "zh";
type CwtoolMsgKey =
  | "cwtoolUnmatchedClose"
  | "cwtoolBraceImbalance"
  | "cwtoolTriggerEnumMismatch"
  | "cwtoolFieldEnumMismatch"
  | "cwtoolFieldEnumBlacklistDefault"
  | "cwtoolScopeKeyNotBlock"
  | "cwtoolUnknownModifier"
  | "cwtoolUnknownModifierHint"
  | "unquotedModUnregisteredIdeologyAddIt";

export type CwtoolMessageFormatter = (
  key: CwtoolMsgKey,
  params?: Record<string, string | number>,
) => string;

const CWTOOL_MSG: Record<CwtoolLocale, Record<CwtoolMsgKey, string>> = {
  zh: {
    cwtoolUnmatchedClose: "意外的 `}` · 多余的右括号(没有对应的 `{` 开头)",
    cwtoolBraceImbalance: "括号不平衡 · {depth} 个 `{` 没有匹配的 `}`",
    cwtoolTriggerEnumMismatch:
      "`{key}` 期望 {allowed} · 但读到 `{value}` · HOI4 加载会拒此 trigger",
    cwtoolFieldEnumMismatch: "`{key}` 期望 {allowed} · 但读到 `{value}`",
    cwtoolFieldEnumBlacklistDefault: "`{value}` 是被拒绝的值",
    cwtoolScopeKeyNotBlock:
      "`{key}` 必须是 block 形式 `{key} = { ... }` · 当前是 scalar `{value}` · HOI4 加载会报 `Non assign trigger is not enclosed in {}`",
    cwtoolUnknownModifier: "`{key}` 不是已知 vanilla modifier{hint}",
    cwtoolUnknownModifierHint: " · 是不是想写 `{nearest}`?",
    unquotedModUnregisteredIdeologyAddIt:
      "`{unquoted}` 未在项目意识形态注册表中 · 若是 mod 自定义意识形态,可在「项目设置 → 意识形态注册表」登记以消除此提示",
  },
  en: {
    cwtoolUnmatchedClose:
      "Unexpected `}` · Extra closing brace with no matching `{`",
    cwtoolBraceImbalance:
      "Unbalanced braces · {depth} unmatched `{` without `}`",
    cwtoolTriggerEnumMismatch:
      "`{key}` expects {allowed} · got `{value}` · HOI4 will reject this trigger",
    cwtoolFieldEnumMismatch: "`{key}` expects {allowed} · got `{value}`",
    cwtoolFieldEnumBlacklistDefault: "`{value}` is a rejected value",
    cwtoolScopeKeyNotBlock:
      "`{key}` must be a block `{key} = { ... }` · current value is scalar `{value}` · HOI4 will reject with `Non assign trigger is not enclosed in {}`",
    cwtoolUnknownModifier: "`{key}` is not a known vanilla modifier{hint}",
    cwtoolUnknownModifierHint: " · did you mean `{nearest}`?",
    unquotedModUnregisteredIdeologyAddIt:
      "`{unquoted}` isn't registered in this project's ideology list. If it's a mod's custom ideology, add it under Project Settings → Ideology.",
  },
};

function formatMsg(
  locale: CwtoolLocale,
  key: CwtoolMsgKey,
  params: Record<string, string | number> = {},
  formatter?: CwtoolMessageFormatter,
): string {
  if (formatter) return formatter(key, params);
  const tmpl = CWTOOL_MSG[locale][key];
  return tmpl.replace(/\{(\w+)\}/g, (_m, k: string) =>
    k in params ? String(params[k]) : `{${k}}`,
  );
}

const MODIFIER_BLOCK_KEYS = new Set([
  "modifier",
  "hidden_modifier",
  "targeted_modifier",
]);

const WEIGHT_BLOCK_KEYS = new Set([
  "ai_will_do",
  "ai_chance",
  "weight_modifier",
]);

const LEVENSHTEIN_MAX = 2;

function levenshtein(a: string, b: string, maxDist: number): number {
  const m = a.length;
  const n = b.length;
  if (Math.abs(m - n) > maxDist) return maxDist + 1;
  let prev = new Array<number>(n + 1);
  for (let j = 0; j <= n; j++) prev[j] = j;
  for (let i = 1; i <= m; i++) {
    const curr = new Array<number>(n + 1);
    curr[0] = i;
    let rowMin = i;
    for (let j = 1; j <= n; j++) {
      const cost = a.charCodeAt(i - 1) === b.charCodeAt(j - 1) ? 0 : 1;
      const v = Math.min(curr[j - 1] + 1, prev[j] + 1, prev[j - 1] + cost);
      curr[j] = v;
      if (v < rowMin) rowMin = v;
    }
    if (rowMin > maxDist) return maxDist + 1;
    prev = curr;
  }
  return prev[n];
}

function findNearest(
  target: string,
  candidates: string[],
  maxDist: number,
): string | null {
  let best: string | null = null;
  let bestDist = maxDist + 1;
  for (const c of candidates) {
    if (Math.abs(c.length - target.length) > maxDist) continue;
    const d = levenshtein(target, c, bestDist);
    if (d < bestDist) {
      bestDist = d;
      best = c;
    }
  }
  return best;
}

function inStatModifierBlock(stack: string[]): boolean {
  if (stack.length === 0) return false;
  const top = stack[stack.length - 1];
  if (!MODIFIER_BLOCK_KEYS.has(top)) return false;
  for (let i = 0; i < stack.length - 1; i++) {
    if (WEIGHT_BLOCK_KEYS.has(stack[i])) return false;
  }
  return true;
}

function updateBlockStack(stack: string[], strippedLine: string): void {
  const openMatch = strippedLine.match(/^\s*([A-Za-z_]\w*)\s*=\s*\{/);
  const totalOpens = (strippedLine.match(/\{/g) || []).length;
  const totalCloses = (strippedLine.match(/\}/g) || []).length;
  if (openMatch) {
    stack.push(openMatch[1]);
    for (let extra = 1; extra < totalOpens; extra++) stack.push("");
  } else {
    for (let o = 0; o < totalOpens; o++) stack.push("");
  }
  for (let c = 0; c < totalCloses; c++) stack.pop();
}

export function cwtoolValidate(
  script: string,
  rules: CwtoolRules | null,
  locale: CwtoolLocale = "en",

  opts?: {
    knownIdeologyIds?: ReadonlySet<string>;
    formatMessage?: CwtoolMessageFormatter;
  },
): EditorMarker[] {
  if (!rules || !script) return [];
  const knownIdeologyIds = opts?.knownIdeologyIds;
  const markers: EditorMarker[] = [];
  const lines = script.split("\n");

  for (const ap of rules.anti_patterns) {
    let regex: RegExp;
    try {
      regex = new RegExp(ap.regex);
    } catch {
      continue;
    }
    const severity = (
      ap.severity === "error" ||
      ap.severity === "warning" ||
      ap.severity === "info"
        ? ap.severity
        : "warning"
    ) as EditorMarker["severity"];
    for (let i = 0; i < lines.length; i++) {
      if (regex.test(lines[i])) {
        markers.push({
          line: i + 1,
          msg: ap.message,
          severity,
        });
      }
    }
  }

  let depth = 0;
  let firstUnmatchedOpenLine = -1;
  let firstUnmatchedCloseLine = -1;
  for (let i = 0; i < lines.length; i++) {
    const lineNo = i + 1;
    const stripped = lines[i].replace(/#.*$/, "").replace(/"[^"]*"/g, '""');
    for (const ch of stripped) {
      if (ch === "{") {
        if (depth === 0 && firstUnmatchedOpenLine < 0) {
          firstUnmatchedOpenLine = lineNo;
        }
        depth++;
      } else if (ch === "}") {
        depth--;
        if (depth < 0 && firstUnmatchedCloseLine < 0) {
          firstUnmatchedCloseLine = lineNo;

          depth = 0;
        } else if (depth === 0) {
          firstUnmatchedOpenLine = -1;
        }
      }
    }
  }
  if (firstUnmatchedCloseLine > 0) {
    markers.push({
      line: firstUnmatchedCloseLine,
      msg: formatMsg(locale, "cwtoolUnmatchedClose", {}, opts?.formatMessage),
      severity: "error",
    });
  }
  if (depth > 0) {
    markers.push({
      line: firstUnmatchedOpenLine > 0 ? firstUnmatchedOpenLine : lines.length,
      msg: formatMsg(
        locale,
        "cwtoolBraceImbalance",
        { depth },
        opts?.formatMessage,
      ),
      severity: "error",
    });
  }

  const KV_LINE = /^\s*([A-Za-z_]\w*)\s*=\s*([^\s#]+)/;
  const scopeKeySet = new Set(rules.scope_keys_must_be_block);
  const frameworkSet = new Set(rules.modifier_framework_keys);
  const knownModifiersSet = new Set(rules.known_modifiers ?? []);
  const modifierCandidates = rules.known_modifiers ?? [];

  const knownTriggersSet = new Set(rules.known_triggers ?? []);
  const allowedEffectsInModifierBlockSet = new Set(
    rules.allowed_effects_in_modifier_block ?? [],
  );

  const modifierScalarParentEffectsSet = new Set(
    rules.modifier_scalar_parent_effects ?? [],
  );

  const optionScalarParentTriggersSet = new Set(
    rules.option_scalar_parent_triggers ?? [],
  );

  const blockStack: string[] = [];

  for (let i = 0; i < lines.length; i++) {
    const stripped = lines[i].replace(/#.*$/, "");
    const m = stripped.match(KV_LINE);

    if (m) {
      const key = m[1];
      const value = m[2];
      const unquoted = value.replace(/^"(.*)"$/, "$1");

      const triggerAllowed = rules.trigger_enums[key];
      if (triggerAllowed) {
        if (
          !triggerAllowed.includes(value) &&
          !triggerAllowed.includes(unquoted)
        ) {
          markers.push({
            line: i + 1,
            msg: formatMsg(
              locale,
              "cwtoolTriggerEnumMismatch",
              {
                key,
                allowed: triggerAllowed.join(" / "),
                value,
              },
              opts?.formatMessage,
            ),
            severity: "warning",
          });
        }
      }

      const fieldEnum = rules.field_enums[key];
      if (fieldEnum) {
        let skipByAncestor = false;
        if (fieldEnum.skip_ancestors && fieldEnum.skip_ancestors.length > 0) {
          for (const anc of fieldEnum.skip_ancestors) {
            if (blockStack.includes(anc)) {
              skipByAncestor = true;
              break;
            }
          }
        }
        if (!skipByAncestor) {
          if (fieldEnum.whitelist) {
            if (
              !fieldEnum.whitelist.includes(value) &&
              !fieldEnum.whitelist.includes(unquoted)
            ) {
              markers.push({
                line: i + 1,
                msg: formatMsg(
                  locale,
                  "cwtoolFieldEnumMismatch",
                  {
                    key,
                    allowed: fieldEnum.whitelist.join(" / "),
                    value,
                  },
                  opts?.formatMessage,
                ),
                severity: "warning",
              });
            }
          } else if (fieldEnum.blacklist) {
            if (
              fieldEnum.blacklist.includes(value) ||
              fieldEnum.blacklist.includes(unquoted)
            ) {
              const tmpl =
                fieldEnum.blacklist_message ||
                formatMsg(
                  locale,
                  "cwtoolFieldEnumBlacklistDefault",
                  {
                    value: unquoted,
                  },
                  opts?.formatMessage,
                );
              markers.push({
                line: i + 1,
                msg: tmpl.replace(/\{value\}/g, unquoted),
                severity: "warning",
              });
            } else if (
              key === "ideology" &&
              knownIdeologyIds &&
              unquoted &&
              !knownIdeologyIds.has(unquoted) &&
              !knownIdeologyIds.has(value)
            ) {
              markers.push({
                line: i + 1,
                msg: formatMsg(
                  locale,
                  "unquotedModUnregisteredIdeologyAddIt",
                  { unquoted },
                  opts?.formatMessage,
                ),
                severity: "info",
              });
            }
          }
        }
      }

      if (scopeKeySet.has(key) && value[0] !== "{") {
        const parent =
          blockStack.length > 0 ? blockStack[blockStack.length - 1] : "";
        const skipModifierScalar =
          key === "modifier" && modifierScalarParentEffectsSet.has(parent);
        const skipOptionScalar =
          key === "option" && optionScalarParentTriggersSet.has(parent);
        if (!skipModifierScalar && !skipOptionScalar) {
          markers.push({
            line: i + 1,
            msg: formatMsg(
              locale,
              "cwtoolScopeKeyNotBlock",
              { key, value },
              opts?.formatMessage,
            ),
            severity: "error",
          });
        }
      }

      if (
        modifierCandidates.length > 0 &&
        !scopeKeySet.has(key) &&
        !frameworkSet.has(key) &&
        !knownModifiersSet.has(key) &&
        !knownTriggersSet.has(key) &&
        !allowedEffectsInModifierBlockSet.has(key) &&
        inStatModifierBlock(blockStack)
      ) {
        const nearest = findNearest(key, modifierCandidates, LEVENSHTEIN_MAX);
        const hint = nearest
          ? formatMsg(
              locale,
              "cwtoolUnknownModifierHint",
              { nearest },
              opts?.formatMessage,
            )
          : "";
        markers.push({
          line: i + 1,
          msg: formatMsg(
            locale,
            "cwtoolUnknownModifier",
            { key, hint },
            opts?.formatMessage,
          ),
          severity: "warning",
        });
      }
    }

    updateBlockStack(blockStack, stripped);
  }

  return markers;
}
