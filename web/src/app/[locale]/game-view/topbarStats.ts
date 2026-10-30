import {
  gameLocale,
  type GuiCountryStats,
  type GuiNode,
  type GuiWindow,
} from "@/lib/guiRenderApi";
import { matchNamePath } from "./gameNavigation";

export const SHORT_MONTHS = [
  "Jan",
  "Feb",
  "Mar",
  "Apr",
  "May",
  "Jun",
  "Jul",
  "Aug",
  "Sep",
  "Oct",
  "Nov",
  "Dec",
] as const;

export function abbreviate(value: number): string {
  if (!Number.isFinite(value)) return "0";
  if (Math.abs(value) >= 1_000_000) return `${(value / 1_000_000).toFixed(2)}M`;
  if (Math.abs(value) >= 1_000) return `${(value / 1_000).toFixed(2)}K`;
  return String(Math.round(value));
}

export function formatGameDate(
  raw: string | null | undefined,
  months: readonly string[] = SHORT_MONTHS,
  locale: string = gameLocale(),
): string {
  const parts = (raw ?? "").split(".");
  const year = Number(parts[0]);
  const month = Number(parts[1]);
  const day = Number(parts[2]);
  const hour = Number(parts[3] ?? 12);
  if (!year || !month || !day || month > 12) return raw ?? "";
  const name = months[month - 1] ?? SHORT_MONTHS[month - 1];
  if (/^(zh|ja)/i.test(locale))
    return `${year}年${/月$/.test(name) ? name : `${month}月`}${day}日 ${hour}:00`;
  if (/^ko/i.test(locale))
    return `${year}년 ${/월$/.test(name) ? name : `${month}월`} ${day}일 ${hour}:00`;
  return `${hour}:00, ${day} ${name}, ${year}`;
}

function percent(value: number | undefined): string {
  return `${Math.round((value ?? 0) * 100)}%`;
}

function whole(value: number | undefined): string {
  return String(Math.floor(value ?? 0));
}

export function factoriesTotal(stats: GuiCountryStats): number {
  return (
    stats.factories_total ??
    (stats.civilian_factories ?? 0) +
      (stats.military_factories ?? 0) +
      (stats.dockyards ?? 0)
  );
}

export function decisionsTakeable(stats: GuiCountryStats | undefined): number {
  if (!stats) return 0;
  return (
    stats.decisions_takeable_count ?? stats.decisions_takeable?.length ?? 0
  );
}

export function threatFrame(tension: number): number {
  return Math.min(10, Math.max(1, Math.floor(tension / 10) + 1));
}

export function topbarValues(
  stats: GuiCountryStats,
  date: string | null | undefined,
  months: readonly string[] = SHORT_MONTHS,
): Record<string, string> {
  const out: Record<string, string> = {
    pol_power: whole(stats.political_power),
    command_power_value: whole(stats.command_power),
    manpower: abbreviate(stats.manpower ?? 0),
    stability: percent(stats.stability),
    war_support: percent(stats.war_support),
    industrial_capacity: String(factoriesTotal(stats)),

    convoys_count: String(stats.convoys ?? 0),
    DateText: formatGameDate(date, months),

    experience_value: whole(stats.army_experience),
    navy_experience_value: whole(stats.navy_experience),
    air_experience_value: whole(stats.air_experience),

    nukes_value: whole(stats.nukes),

    initiative_text: stats.faction
      ? String(Math.floor(stats.faction.initiative ?? 0))
      : "",
  };

  if (stats.supply_ratio !== undefined)
    out.supply_value = percent(stats.supply_ratio);
  if (stats.world_tension !== undefined)
    out.threat_value = `${Math.round(stats.world_tension)}%`;
  const takeable = decisionsTakeable(stats);
  if (takeable > 0) out.decisionview_amount_to_take_items = String(takeable);
  return out;
}

export function topbarBars(stats: GuiCountryStats): Record<string, number> {
  const clamp = (v: number) => Math.min(1, Math.max(0, v));
  return {
    industrial_ratio_bar: 1,
    fuel_ratio_bar: clamp(stats.fuel?.ratio ?? 0),
    supply_ratio_bar: clamp(stats.supply_ratio ?? 1),
    convoys_ratio_bar: 1,
  };
}

export interface FuelReadout {
  text: string;
  color: "G" | "R" | null;
  frame: number;
}

export function fuelReadout(
  stats: GuiCountryStats,
  loc: Readonly<Record<string, string>> = {},
): FuelReadout | null {
  const fuel = stats.fuel;
  if (!fuel) return null;

  if (fuel.ratio >= 1)
    return {
      text: loc.FUEL_STOCKPILE_MAX_SHORT || "Full",
      color: "G",
      frame: 1,
    };
  if (fuel.gaining && fuel.days !== null && fuel.days !== undefined) {
    if (fuel.days >= 365)
      return {
        text: `${(fuel.days / 365).toFixed(1)} ${loc.YEAR_UNIT_SHORT || "y"}`,
        color: "G",
        frame: 2,
      };
    return {
      text: `${fuel.days} ${loc.DAY_UNIT_SHORT || "d"}`,
      color: "G",
      frame: 2,
    };
  }

  if (fuel.ratio <= 0)
    return {
      text: loc.FUEL_STOCKPILE_EMPTY_SHORT || "0",
      color: "R",
      frame: 3,
    };
  return { text: "", color: null, frame: 1 };
}

function idOf(node: GuiNode): string {
  return (node as GuiNode & { _id?: string })._id ?? "";
}

export function topbarTextOverrides(
  win: GuiWindow,
  stats: GuiCountryStats | undefined,
  date: string | null | undefined,
  months: readonly string[] = SHORT_MONTHS,
): Record<string, string> {
  if (!stats) return {};
  const out: Record<string, string> = {};
  for (const [name, value] of Object.entries(
    topbarValues(stats, date, months),
  )) {
    for (const node of matchNamePath(win.node, name)) {
      const id = idOf(node);
      if (id) out[id] = value;
    }
  }
  return out;
}
