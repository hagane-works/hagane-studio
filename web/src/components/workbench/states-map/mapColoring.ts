import type { State } from "@/types";

import type { VanillaStateIndex } from "@/lib/vanillaStateIndex";

export type ColorView =
  | "political"
  | "industry"
  | "category"
  | "resources"
  | "edited"
  | "membership"
  | "vanillaContent";

export const COLOR_VIEWS: ColorView[] = [
  "political",
  "industry",
  "category",
  "resources",
  "edited",
  "membership",

  "vanillaContent",
];

export const RESOURCE_KEYS = [
  "oil",
  "steel",
  "tungsten",
  "aluminium",
  "chromium",
  "rubber",
] as const;
export type ResourceKey = (typeof RESOURCE_KEYS)[number];

export const NEUTRAL_COLOR = "hsl(35, 6%, 78%)";

export const EDITED_CREATED_COLOR = "hsl(28, 82%, 55%)";

export const MEMBERSHIP_CHANGED_COLOR = "hsl(28, 88%, 52%)";

export const VANILLA_CONTENT_MISSING_COLOR = "hsl(28, 84%, 58%)";

const CATEGORY_PALETTE = [
  "hsl(210, 55%, 62%)",
  "hsl(145, 45%, 50%)",
  "hsl(35, 80%, 58%)",
  "hsl(280, 45%, 63%)",
  "hsl(2, 62%, 62%)",
  "hsl(190, 50%, 50%)",
  "hsl(320, 45%, 63%)",
  "hsl(95, 42%, 50%)",
  "hsl(255, 45%, 66%)",
  "hsl(18, 68%, 58%)",
  "hsl(168, 44%, 46%)",
  "hsl(50, 68%, 54%)",
];

function hashString(s: string): number {
  let h = 2166136261;
  for (let i = 0; i < s.length; i++) {
    h ^= s.charCodeAt(i);
    h = Math.imul(h, 16777619);
  }
  return h >>> 0;
}

export function politicalColor(tag: string | null | undefined): string {
  if (!tag) return NEUTRAL_COLOR;
  const hue = (hashString(tag) * 137.508) % 360;
  return `hsl(${hue.toFixed(1)}, 45%, 60%)`;
}

export function heatColor(t: number): string {
  const c = Math.max(0, Math.min(1, Number.isFinite(t) ? t : 0));
  const h = 46 - 20 * c;
  const s = 28 + 60 * c;
  const l = 90 - 42 * c;
  return `hsl(${h.toFixed(1)}, ${s.toFixed(1)}%, ${l.toFixed(1)}%)`;
}

export function industryValue(st: State): number {
  const b = st.buildings ?? {};
  return (b.industrial_complex ?? 0) + (b.arms_factory ?? 0);
}

export interface ColorContext {
  view: ColorView;
  resourceKey: ResourceKey;

  industryMax: number;

  resourceMax: number;

  categoryColors: Record<string, string>;

  categoryOrder: string[];
  membershipChangedCount: number;
  membershipUnchangedCount: number;
  membershipUnknownCount: number;

  vanillaContentMissing: number;

  vanillaContent: VanillaStateIndex["content"];
}

export function buildColorContext(
  states: State[],
  view: ColorView,
  resourceKey: ResourceKey,
  vanillaContent: VanillaStateIndex["content"],
): ColorContext {
  let industryMax = 0;
  let resourceMax = 0;
  const catSet = new Set<string>();
  let membershipChangedCount = 0;
  let membershipUnchangedCount = 0;
  let membershipUnknownCount = 0;
  let vanillaContentMissing = 0;
  for (const st of states) {
    if (view === "industry") {
      const v = industryValue(st);
      if (v > industryMax) industryMax = v;
    } else if (view === "resources") {
      const v = st.resources?.[resourceKey] ?? 0;
      if (v > resourceMax) resourceMax = v;
    } else if (view === "category") {
      catSet.add(st.state_category ?? "");
    } else if (view === "vanillaContent") {
      if (
        vanillaContent[st.state_id] &&
        Object.keys(st.raw_extras ?? {}).length === 0
      ) {
        vanillaContentMissing += 1;
      }
    } else if (view === "membership") {
      const evidence = st.membership_provenance;
      if (evidence?.status !== "known" || evidence.membership_changed == null) {
        membershipUnknownCount += 1;
      } else if (evidence.membership_changed) {
        membershipChangedCount += 1;
      } else {
        membershipUnchangedCount += 1;
      }
    }
  }
  const categoryOrder = Array.from(catSet).sort((a, b) => {
    if (a === "") return 1;
    if (b === "") return -1;
    return a.localeCompare(b);
  });
  const categoryColors: Record<string, string> = {};
  let paletteIdx = 0;
  for (const cat of categoryOrder) {
    categoryColors[cat] =
      cat === ""
        ? NEUTRAL_COLOR
        : CATEGORY_PALETTE[paletteIdx++ % CATEGORY_PALETTE.length];
  }
  return {
    view,
    resourceKey,
    industryMax,
    resourceMax,
    categoryColors,
    categoryOrder,
    membershipChangedCount,
    membershipUnchangedCount,
    membershipUnknownCount,
    vanillaContentMissing,
    vanillaContent,
  };
}

export function colorForState(
  view: ColorView,
  st: State,
  ctx: ColorContext,
): string {
  switch (view) {
    case "political":
      return politicalColor(st.owner);
    case "industry":
      return heatColor(
        ctx.industryMax > 0 ? industryValue(st) / ctx.industryMax : 0,
      );
    case "category":
      return ctx.categoryColors[st.state_category ?? ""] ?? NEUTRAL_COLOR;
    case "resources":
      return heatColor(
        ctx.resourceMax > 0
          ? (st.resources?.[ctx.resourceKey] ?? 0) / ctx.resourceMax
          : 0,
      );
    case "edited":
      return st.source === "created" ? EDITED_CREATED_COLOR : NEUTRAL_COLOR;
    case "vanillaContent":
      return ctx.vanillaContent[st.state_id] &&
        Object.keys(st.raw_extras ?? {}).length === 0
        ? VANILLA_CONTENT_MISSING_COLOR
        : NEUTRAL_COLOR;
    case "membership":
      return st.membership_provenance?.status === "known" &&
        st.membership_provenance.membership_changed === true
        ? MEMBERSHIP_CHANGED_COLOR
        : NEUTRAL_COLOR;
    default:
      return NEUTRAL_COLOR;
  }
}

export type Legend =
  | { kind: "political" }
  | { kind: "edited" }
  | { kind: "vanillaContent"; missingCount: number }
  | {
      kind: "membership";
      changedCount: number;
      unchangedCount: number;
      unknownCount: number;
    }
  | { kind: "categorical"; entries: { color: string; label: string }[] }
  | { kind: "gradient"; from: string; to: string; min: number; max: number };

export function buildLegend(view: ColorView, ctx: ColorContext): Legend {
  switch (view) {
    case "political":
      return { kind: "political" };
    case "edited":
      return { kind: "edited" };
    case "vanillaContent":
      return {
        kind: "vanillaContent",
        missingCount: ctx.vanillaContentMissing,
      };
    case "membership":
      return {
        kind: "membership",
        changedCount: ctx.membershipChangedCount,
        unchangedCount: ctx.membershipUnchangedCount,
        unknownCount: ctx.membershipUnknownCount,
      };
    case "category":
      return {
        kind: "categorical",

        entries: ctx.categoryOrder.map((c) => ({
          color: ctx.categoryColors[c] ?? NEUTRAL_COLOR,
          label: c || "—",
        })),
      };
    case "industry":
      return {
        kind: "gradient",
        from: heatColor(0),
        to: heatColor(1),
        min: 0,
        max: ctx.industryMax,
      };
    case "resources":
      return {
        kind: "gradient",
        from: heatColor(0),
        to: heatColor(1),
        min: 0,
        max: ctx.resourceMax,
      };
    default:
      return { kind: "political" };
  }
}

export function ringsToPath(rings: number[][][]): string {
  let d = "";
  for (const ring of rings) {
    if (ring.length === 0) continue;
    for (let i = 0; i < ring.length; i++) {
      const [x, y] = ring[i];
      d += (i === 0 ? "M" : "L") + x + "," + y;
    }
    d += "Z";
  }
  return d;
}
