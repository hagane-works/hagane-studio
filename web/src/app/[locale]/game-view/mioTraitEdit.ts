import type { MIORow, MIOTraitEntry } from "@/lib/api/military";
import { stableDraftFingerprint } from "@/lib/workbench-state/draftSession";
import type { GuiIndustrialOrgTrait } from "./IndustrialOrgDetailLayer";

const BONUS_FIELDS = [
  "equipment_bonus",
  "production_bonus",
  "organization_modifier",
] as const;
const SCRIPT_FIELDS = [
  "available",
  "visible",
  "on_complete",
  "ai_will_do",
] as const;

const LIFTED_FIELDS = [
  "position",
  "all_parents",
  ...BONUS_FIELDS,
  ...SCRIPT_FIELDS,
] as const;

const isBonus = (field: string) =>
  (BONUS_FIELDS as readonly string[]).includes(field);
const same = (a: unknown, b: unknown) =>
  stableDraftFingerprint(a ?? null) === stableDraftFingerprint(b ?? null);

export function traitEntryIndex(
  row: Pick<MIORow, "traits">,
  token: string,
): { index: number; own: boolean } {
  const traits = row.traits ?? [];
  const own = traits.findIndex(
    (t) =>
      t.kind !== "override" &&
      t.kind !== "initial" &&
      (t.token ?? t.name) === token,
  );
  if (own >= 0) return { index: own, own: true };
  return {
    index: traits.findIndex((t) => t.kind === "override" && t.token === token),
    own: false,
  };
}

const LINE = /^([A-Za-z0-9_.@]+)\s*=\s*(-?\d+(?:\.\d+)?)$/;

export function parseBonus(text: string): Record<string, number> | null {
  const out: Record<string, number> = {};
  for (const raw of text.split("\n")) {
    const line = raw.trim();
    if (!line) continue;
    const match = LINE.exec(line);
    if (!match) return null;
    out[match[1]] = Number(match[2]);
  }
  return out;
}

const AXIS = /^([xy])\s*=\s*(-?\d+)$/;

export function liftField(field: string, text: string): unknown {
  if (field === "position") {
    const at: Record<string, number> = {};
    for (const raw of text.split("\n")) {
      const line = raw.trim();
      if (!line) continue;
      const match = AXIS.exec(line);
      if (!match) return undefined;
      at[match[1]] = Number(match[2]);
    }
    return { x: at.x ?? 0, y: at.y ?? 0 };
  }
  if (field === "all_parents")
    return /[={}<>]/.test(text) ? undefined : text.split(/\s+/).filter(Boolean);
  if (isBonus(field)) return parseBonus(text) ?? undefined;
  if ((SCRIPT_FIELDS as readonly string[]).includes(field)) return text.trim();
  return undefined;
}

export function fieldText(field: string, value: unknown): string {
  if (field === "position") {
    const at = value as { x: number; y: number };
    return `x = ${at.x}\ny = ${at.y}`;
  }
  if (field === "all_parents") return `${(value as string[]).join("\n")}\n`;
  if (isBonus(field)) {
    return (
      Object.entries(value as Record<string, number>)
        .map(([key, number]) => `${key} = ${number}`)
        .join("\n") || "\n"
    );
  }
  return String(value);
}

export interface TraitStart {
  trait: MIOTraitEntry;

  entry?: MIOTraitEntry;

  lifted: Record<string, string>;

  inherited: Record<string, unknown>;
}

export function traitStart(
  entry: MIOTraitEntry | undefined,
  token: string,
  shown: GuiIndustrialOrgTrait | undefined,
  anchor: { x: number; y: number } | null,
): TraitStart {
  const trait: MIOTraitEntry = entry
    ? { ...entry }
    : { kind: "override", token };
  const extras: Record<string, unknown> = { ...(entry?.raw_extras ?? {}) };
  const lifted: Record<string, string> = {};
  for (const field of LIFTED_FIELDS as readonly string[]) {
    const text = extras[field];
    if (typeof text !== "string" || trait[field] !== undefined) continue;
    const value = liftField(field, text);
    if (value === undefined) continue;
    lifted[field] = text;
    trait[field] = value;
    delete extras[field];
  }
  if (Object.keys(lifted).length) trait.raw_extras = extras;

  const inherited: Record<string, unknown> = {};
  if (trait.kind === "override" && shown) {
    const named = (field: string) =>
      trait[field] !== undefined || field in extras;
    if (!named("position")) {
      const at = shown.relative_to
        ? anchor
          ? { x: shown.x - anchor.x, y: shown.y - anchor.y }
          : null
        : { x: shown.x, y: shown.y };
      if (at) inherited.position = at;
    }
    for (const field of BONUS_FIELDS) {
      const values = shown.values?.[field];
      if (!named(field) && values?.length)
        inherited[field] = Object.fromEntries(
          values.map((v) => [v.key, v.value]),
        );
    }
    Object.assign(trait, inherited);
  }
  return { trait, entry, lifted, inherited };
}

export function storedTrait(
  edited: MIOTraitEntry,
  start: TraitStart,
): MIOTraitEntry {
  const out: MIOTraitEntry = { ...edited };
  const editedExtras = (edited.raw_extras ?? {}) as Record<string, unknown>;
  const extras: Record<string, unknown> = {};
  const lower = (field: string, text: string) => {
    const value = edited[field];
    delete out[field];
    if (value === undefined) {
      if (field in editedExtras) extras[field] = editedExtras[field];
      else if (isBonus(field)) extras[field] = "\n";
      return;
    }
    extras[field] = same(value, liftField(field, text))
      ? text
      : fieldText(field, value);
  };

  for (const key of Object.keys(start.entry?.raw_extras ?? {})) {
    if (key in start.lifted) lower(key, start.lifted[key]);
    else if (key in editedExtras) extras[key] = editedExtras[key];
  }
  for (const key of Object.keys(editedExtras))
    if (!(key in extras)) extras[key] = editedExtras[key];

  for (const [field, value] of Object.entries(start.inherited)) {
    const now = edited[field];
    if (now !== undefined && !same(now, value)) continue;
    delete out[field];
    if (now === undefined && isBonus(field) && !(field in extras))
      extras[field] = "\n";
  }

  if (start.entry?.raw_extras !== undefined || Object.keys(extras).length)
    out.raw_extras = extras;
  else delete out.raw_extras;
  return out;
}

function emptyOverride(trait: MIOTraitEntry): boolean {
  if (trait.kind !== "override") return false;
  const named = Object.keys(trait).filter(
    (key) => key !== "kind" && key !== "token" && key !== "raw_extras",
  );
  return named.length === 0 && Object.keys(trait.raw_extras ?? {}).length === 0;
}

export function traitsWithEntry(
  row: Pick<MIORow, "traits">,
  token: string,
  stored: MIOTraitEntry,
): MIOTraitEntry[] {
  const traits = row.traits ?? [];
  const { index } = traitEntryIndex(row, token);
  if (index >= 0) {
    if (same(traits[index], stored)) return traits;
    return traits.map((trait, at) => (at === index ? stored : trait));
  }
  return emptyOverride(stored) ? traits : [...traits, stored];
}
