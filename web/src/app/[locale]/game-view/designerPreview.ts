import type { GuiDesigner } from "@/lib/guiRenderApi";
import { fillRuntimeSlots } from "../gui-renderer/tooltipText";
import {
  REGIMENTAL_SUPPORT_BATTALIONS,
  type DesignerUnit,
} from "./designerEdit";

const LOWER_IS_BETTER = new Set([
  "weight",
  "supply_use",
  "combat_width",
  "manpower",
  "training_time",
  "fuel_use",
  "experience_loss",
  "production_cost",
]);

const EPSILON = 0.0005;

export function lowerIsBetter(stat: string): boolean {
  return LOWER_IS_BETTER.has(stat);
}

export function changeArrow(
  before: number,
  after: number,
  lowerBetter: boolean,
): string {
  const delta = after - before;
  if (Math.abs(delta) < EPSILON) return "";
  const up = delta > 0;
  const better = up !== lowerBetter;
  if (up) return better ? "£increase_texticon" : "£decrease_reversed_texticon";
  return better ? "£increase_reversed_texticon" : "£decrease_texticon";
}

type DeltaFormat = "2" | "1" | "0" | "percent" | "speed";

const DELTA_ROWS: readonly {
  key: string;
  stat: string;
  format: DeltaFormat;
}[] = [
  { key: "STAT_ARMY_DEFAULT_MORALE", stat: "recovery_rate", format: "2" },
  { key: "STAT_ARMY_DEFENSE", stat: "defense", format: "1" },
  { key: "STAT_ARMY_BREAKTHROUGH", stat: "breakthrough", format: "1" },

  { key: "STAT_ARMY_HARDNESS", stat: "hardness", format: "percent" },
  { key: "STAT_ARMY_SOFT_ATTACK", stat: "soft_attack", format: "1" },
  { key: "STAT_ARMY_HARD_ATTACK", stat: "hard_attack", format: "1" },
  { key: "STAT_AIR_ATTACK", stat: "air_attack", format: "1" },
  { key: "STAT_ARMY_SUPPLY_CONSUMPTION", stat: "supply_use", format: "2" },
  { key: "STAT_ARMY_SUPRESSION", stat: "suppression", format: "1" },
  { key: "STAT_ARMY_FUEL_CAPACITY", stat: "fuel_capacity", format: "2" },
  { key: "STAT_ARMY_RECON", stat: "recon", format: "1" },

  { key: "STAT_ARMY_ENTRENCHMENT", stat: "entrenchment", format: "1" },
  { key: "STAT_COMMON_MAX_ORG", stat: "max_organisation", format: "1" },
  { key: "STAT_COMMON_MAX_STRENGTH", stat: "max_strength", format: "1" },
  { key: "STAT_COMMON_MAXIMUM_SPEED", stat: "speed", format: "speed" },
  { key: "STAT_COMMON_ARMOR", stat: "armor_value", format: "1" },
  { key: "STAT_COMMON_PIERCING", stat: "ap_attack", format: "1" },
  { key: "STAT_COMMON_RELIABILITY", stat: "reliability", format: "percent" },
  { key: "STAT_COMMON_WEIGHT", stat: "weight", format: "1" },
  { key: "STAT_COMMON_FUEL_CONSUMPTION", stat: "fuel_use", format: "2" },
  { key: "STAT_ARMY_INITIATIVE", stat: "initiative", format: "percent" },

  { key: "STAT_COMMON_BUILD_COST_IC", stat: "production_cost", format: "2" },
  { key: "DESIGNER_COMBATWIDTH", stat: "combat_width", format: "1" },
];

export const DELTA_KEYS = [
  ...DELTA_ROWS.map((row) => row.key),
  "DESIGNER_MANPOWER",
  "DESIGNER_TRAINING_TIME",

  "DESIGNER_REMOVE",

  "DESIGNER_SUBUNIT_DEPLOYMENT_COST",
  "DESIGNER_DEPLOYED_LEADER_MODIFIERS",
  "ENABLES_ABILITY",
];

const SLOT = {
  add: "DESIGNER_ADD",
  copyAbove: "DESIGNER_ADD_COPY_ABOVE",
  fillRest: "DESIGNER_ADD_FILL_REGIMENT",
  fillPick: "DESIGNER_ADD_FILL_REGIMENT_FROM_PICK",
  copyLeft: "DESIGNER_ADD_COPY_LEFT_REGIMENT",
  fillLeft: "DESIGNER_ADD_FILL_REGIMENT_FROM_LEFT",
  change: "DESIGNER_CHANGE",
  remove: "DESIGNER_CHANGE_REMOVE",
  removeRegiment: "DESIGNER_REMOVE_REGIMENT",
  lockedSupport: "DESIGNER_BLOCKED_REGIMENTAL_SUPPORT",
  needsBattalions: "DESIGNER_BLOCKED_BY_REGIMENT_BATTALIONS",

  hqRemove: "DESIGNER_HQ_REMOVE",
  hqCopyAbove: "DESIGNER_HQ_ADD_COPY_ABOVE",
} as const;

export const SLOT_KEYS = Object.values(SLOT);

export function slotHints(
  slot: { kind: string; x: number; y: number },
  grid: "regiment" | "support" | "regimental",
  hasLeft: boolean,
  texts: Record<string, string>,
): string[] {
  const line = (key: string) => texts[key] ?? "";
  let keys: string[] = [];
  if (slot.kind === "unit") {
    keys =
      grid === "regiment"
        ? [SLOT.change, SLOT.remove, SLOT.removeRegiment]
        : [SLOT.change, SLOT.remove];
  } else if (slot.kind === "add") {
    if (grid !== "regiment") keys = [SLOT.add];
    else if (slot.y > 0) keys = [SLOT.add, SLOT.copyAbove, SLOT.fillRest];
    else if (hasLeft)
      keys = [SLOT.add, SLOT.fillPick, SLOT.copyLeft, SLOT.fillLeft];
    else keys = [SLOT.add, SLOT.fillPick];
  } else if (slot.kind === "locked" && grid === "regimental") {
    return [
      line(SLOT.lockedSupport),
      fillRuntimeSlots(line(SLOT.needsBattalions), {
        NUM_BATTALIONS: REGIMENTAL_SUPPORT_BATTALIONS,
      }),
    ].filter(Boolean);
  }
  return keys.map(line).filter(Boolean);
}

export function slotTooltip(
  name: string | null,
  hints: readonly string[],
  alone: GuiDesigner | undefined,
  texts: Record<string, string>,
  lead: readonly string[] = [],
): string {
  const head = [...(name ? [`§Y${name}§!`] : []), ...hints].join("\n");
  const empty = { stats: {}, equipment: [] } as unknown as GuiDesigner;
  const stats = alone
    ? deltaTooltip(name ?? "", empty, alone, texts)
        .split("\n")
        .slice(1)
    : [];
  const body = [...lead, ...stats];
  return body.length ? `${head}\n\n${body.join("\n")}` : head;
}

export function deploymentLine(
  unit: DesignerUnit | undefined,
  texts: Record<string, string>,
): string | null {
  const cost = unit?.deployment_cost;
  const markup = texts.DESIGNER_SUBUNIT_DEPLOYMENT_COST;
  return cost === undefined || !markup
    ? null
    : fillRuntimeSlots(markup, { COST: cost });
}

export function hqPickerTooltip(
  tooltip: string,
  unit: DesignerUnit,
  texts: Record<string, string>,
): string {
  const lines = tooltip.split("\n");
  const deploy = deploymentLine(unit, texts);
  if (deploy) lines.splice(1, 0, deploy);
  if (unit.commander_bonuses?.length) {
    lines.push(
      "",
      texts.DESIGNER_DEPLOYED_LEADER_MODIFIERS ?? "§HCommander bonuses:§!",
      ...unit.commander_bonuses,
    );
  }
  if (unit.abilities?.length) {
    const header = (texts.ENABLES_ABILITY ?? "Enables Ability: $NAME$")
      .replace(/\$NAME[^$]*\$/, "")
      .trim();
    lines.push(
      "",
      `§H${header}§!`,
      ...unit.abilities.map((name) => `§G+${name}§!`),
    );
  }
  return lines.join("\n");
}

function signed(delta: number, format: DeltaFormat): string {
  const sign = delta > 0 ? "+" : "-";
  const size = Math.abs(delta);
  switch (format) {
    case "percent":
      return `${sign}${(size * 100).toFixed(1)}%`;
    case "speed":
    case "1":
      return `${sign}${size.toFixed(1)}`;
    case "2":
      return `${sign}${size.toFixed(2)}`;
    default:
      return `${sign}${Math.round(size)}`;
  }
}

function label(texts: Record<string, string>, key: string): string {
  const text = (texts[key] ?? key).trimEnd();
  return `${text}  `;
}

export function deltaTooltip(
  title: string,
  before: GuiDesigner,
  after: GuiDesigner,
  texts: Record<string, string>,
): string {
  const lines = [`§Y${title}§!`];
  const stat = (data: GuiDesigner, name: string) =>
    Number(data.stats?.[name] ?? 0);
  for (const row of DELTA_ROWS) {
    const delta = stat(after, row.stat) - stat(before, row.stat);
    if (Math.abs(delta) < EPSILON) continue;
    const good = delta > 0 !== lowerIsBetter(row.stat);
    const value = `§${good ? "G" : "R"}${signed(delta, row.format)}§!`;
    lines.push(
      `${label(texts, row.key)}${value}${row.format === "speed" ? " km/h" : ""}`,
    );
  }

  const manpower = stat(after, "manpower") - stat(before, "manpower");
  if (Math.abs(manpower) >= EPSILON)
    lines.push(
      `${label(texts, "DESIGNER_MANPOWER")}§Y${Math.round(manpower)}§!`,
    );
  const training = stat(after, "training_time") - stat(before, "training_time");
  if (Math.abs(training) >= EPSILON) {
    lines.push(
      `${label(texts, "DESIGNER_TRAINING_TIME")}§${training > 0 ? "R" : "G"}${signed(training, "0")}§!`,
    );
  }

  const amounts = (data: GuiDesigner) =>
    new Map(data.equipment.map((item) => [item.archetype, item]));
  const was = amounts(before);
  const now = amounts(after);
  for (const archetype of new Set([...now.keys(), ...was.keys()])) {
    const item = now.get(archetype) ?? was.get(archetype);
    const delta =
      (now.get(archetype)?.amount ?? 0) - (was.get(archetype)?.amount ?? 0);
    if (!item || Math.abs(delta) < EPSILON) continue;
    const name = item.short_label || item.label || archetype;
    lines.push(`${name}:  §Y${Math.round(delta)}§!`);
  }
  return lines.join("\n");
}
