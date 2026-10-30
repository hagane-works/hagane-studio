import type { Node, Edge } from "@xyflow/react";
import type { Technology, TechnologyFolderPlacement } from "@/types";
import type { ContentLocale } from "@/lib/store/contentLocaleStore";
import { API_BASE } from "@/lib/api";
import { gfxUrl } from "@/lib/api/basemod";
import { resolveDisplayName } from "@/lib/i18n/localizedName";
import {
  resolveVisualSection,
  getFolderSubSections,
  type TechSections,
} from "./visualSectionMapping";

export const COLUMN_WIDTH = 130;
export const ROW_HEIGHT_PER_YEAR = 100;

export const PRE_1934_GAP = 200;
export const NODE_OFFSET_X = 65;
export const NODE_OFFSET_Y = 40;
export const TIMELINE_LABEL_WIDTH = 60;

export const YEAR_TICKS = [
  1918, 1934, 1936, 1938, 1939, 1940, 1941, 1942, 1943, 1944, 1945, 1946,
];
export const TIMELINE_START_YEAR = 1918;
export const TIMELINE_END_YEAR = 1946;

export function getTechnologyFolderPlacements(
  tech: Pick<
    Technology,
    "folder" | "position_x" | "position_y" | "folder_placements"
  >,
): TechnologyFolderPlacement[] {
  if (tech.folder_placements && tech.folder_placements.length > 0) {
    return tech.folder_placements.map((placement) => ({ ...placement }));
  }
  return tech.folder
    ? [{ folder: tech.folder, x: tech.position_x, y: tech.position_y }]
    : [];
}

export function resolveTechnologyPlacementIndex(
  tech: Pick<
    Technology,
    "folder" | "position_x" | "position_y" | "folder_placements"
  >,
  selectedIndex: number,
): number {
  const placements = getTechnologyFolderPlacements(tech);
  return placements.length === 0
    ? 0
    : Math.min(Math.max(selectedIndex, 0), placements.length - 1);
}

export function getSelectedTechnologyPlacement(
  tech: Pick<
    Technology,
    "folder" | "position_x" | "position_y" | "folder_placements"
  >,
  selectedIndex: number,
): TechnologyFolderPlacement | null {
  const placements = getTechnologyFolderPlacements(tech);
  return (
    placements[resolveTechnologyPlacementIndex(tech, selectedIndex)] ?? null
  );
}

export function updateTechnologyFolderPlacement(
  tech: Technology,
  selectedIndex: number,
  patch: Partial<TechnologyFolderPlacement>,
): Technology {
  const placements = getTechnologyFolderPlacements(tech);
  if (placements.length === 0) return tech;
  const index = resolveTechnologyPlacementIndex(tech, selectedIndex);
  placements[index] = { ...placements[index], ...patch };
  const first = placements[0];
  return {
    ...tech,
    folder_placements: placements,
    folder: first.folder,
    position_x: first.x,
    position_y: first.y,
  };
}

export function technologyAtSelectedPlacement(
  tech: Technology,
  selectedIndex: number,
): Technology {
  const placement = getSelectedTechnologyPlacement(tech, selectedIndex);
  return placement
    ? {
        ...tech,
        folder: placement.folder,
        position_x: placement.x,
        position_y: placement.y,
      }
    : tech;
}

export type LedgerKey = "army" | "air" | "navy" | "civilian" | "special";

export const LEDGER_LABELS: Record<LedgerKey, { en: string; zh: string }> = {
  army: { en: "Army", zh: "陆军" },
  air: { en: "Air", zh: "空军" },
  navy: { en: "Navy", zh: "海军" },
  civilian: { en: "Civilian", zh: "工业装备" },
  special: { en: "Special", zh: "特别项目" },
};

export const LEDGER_LABEL_I18N_KEYS: Record<LedgerKey, string> = {
  army: "army",
  air: "air",
  navy: "navy",
  civilian: "civilian",
  special: "special",
};

export const LEDGER_ORDER: LedgerKey[] = [
  "army",
  "air",
  "navy",
  "civilian",
  "special",
];

export const LEDGER_BY_TECH_LINE: Record<string, LedgerKey> = {
  infantry: "army",
  support: "army",
  artillery: "army",
  armor: "army",
  nsb_armor: "army",
  special_forces_doctrine: "army",

  air_techs: "air",
  bba_air_techs: "air",

  naval: "navy",
  mtg_naval: "navy",
  naval_support: "navy",

  industry: "civilian",
  electronic_mechanical_engineering: "civilian",
};

export function resolveLedger(techLine: string): LedgerKey {
  return LEDGER_BY_TECH_LINE[techLine] ?? "army";
}

export const TECH_LINE_DEFAULT_LAYOUT: Record<string, TechLayoutMode> = {
  infantry: "vertical",
  support: "vertical",
  artillery: "vertical",
  armor: "vertical",
  nsb_armor: "vertical",
  special_forces_doctrine: "vertical",

  air_techs: "vertical",
  bba_air_techs: "vertical",

  naval: "vertical",
  mtg_naval: "vertical",
  naval_support: "vertical",

  industry: "vertical",
  electronic_mechanical_engineering: "vertical",
};

export function resolveTechLineLayout(techLine: string): TechLayoutMode {
  return TECH_LINE_DEFAULT_LAYOUT[techLine] ?? "vertical";
}

export type TechLayoutClass = "tree" | "doctrine";

export const LAYOUT_CLASS_BY_TECH_LINE: Record<string, TechLayoutClass> = {
  infantry: "tree",
  support: "tree",
  artillery: "tree",
  armor: "tree",
  nsb_armor: "tree",
  air_techs: "tree",
  bba_air_techs: "tree",
  naval: "tree",
  mtg_naval: "tree",
  naval_support: "tree",
  industry: "tree",
  electronic_mechanical_engineering: "tree",

  special_forces_doctrine: "doctrine",
};

export function resolveLayoutClass(techLine: string): TechLayoutClass {
  const mapped = LAYOUT_CLASS_BY_TECH_LINE[techLine];
  if (mapped) return mapped;
  if (/_doctrine$/.test(techLine)) return "doctrine";
  return "tree";
}

export const DLC_TECH_LINE_TO_BASE: Record<string, string> = {
  nsb_armor: "armor",
  mtg_naval: "naval",
  bba_air_techs: "air_techs",
};

export function resolveDisplayTechLine(techLine: string): string {
  return DLC_TECH_LINE_TO_BASE[techLine] ?? techLine;
}

export function filterByDisplayTechLine(
  techs: Technology[],
  displayTechLine: string,
): Technology[] {
  return techs.filter(
    (t) => resolveDisplayTechLine(t.tech_line) === displayTechLine,
  );
}

export function isForeignCountrySpecificTech(
  techId: string,
  activeCountry: string | null | undefined,
  countryTags: ReadonlySet<string>,
): boolean {
  if (!activeCountry) return false;
  const m = techId.match(/^([A-Z][A-Za-z0-9]{1,3})_/);
  if (!m) return false;
  const prefix = m[1].toUpperCase();
  if (!countryTags.has(prefix)) return false;
  return prefix !== activeCountry.toUpperCase();
}

export function isHiddenInGameTech(
  tech: Technology,
  referencedAsPrereq: Set<string>,
): boolean {
  if (tech.source !== "imported") return false;
  if ((tech.categories ?? []).length > 0) return false;
  const prereqGroups = (tech.prerequisites ?? []) as unknown as string[][];
  const hasPrereq = prereqGroups.some((g) => g.length > 0);
  if (hasPrereq) return false;
  if (referencedAsPrereq.has(tech.tech_id)) return false;
  return true;
}

export type TechVisibilityMode = "reachability" | "hide-hidden" | "show-all";

export type ReachabilityMap = Set<string>;

export function applyTechVisibilityMode(
  techs: Technology[],
  opts: {
    activeCountry: string | null | undefined;
    mode: TechVisibilityMode;
    reachable: ReachabilityMap | null;

    countryTags: ReadonlySet<string>;
  },
): Technology[] {
  const afterCountry = opts.activeCountry
    ? techs.filter(
        (t) =>
          !isForeignCountrySpecificTech(
            t.tech_id,
            opts.activeCountry,
            opts.countryTags,
          ),
      )
    : techs;

  if (opts.mode === "show-all") return afterCountry;

  const referenced = new Set<string>();
  for (const t of afterCountry) {
    for (const grp of (t.prerequisites ?? []) as unknown as string[][]) {
      for (const id of grp) referenced.add(id);
    }
  }

  if (opts.mode === "hide-hidden") {
    return afterCountry.filter((t) => !isHiddenInGameTech(t, referenced));
  }

  const reachable = opts.reachable;

  if (!reachable || reachable.size === 0) {
    return afterCountry.filter((t) => !isHiddenInGameTech(t, referenced));
  }
  return afterCountry.filter((t) => {
    if (reachable.has(t.tech_id)) return true;
    return !isHiddenInGameTech(t, referenced);
  });
}

export function yearToPixel(year: number): number {
  if (year <= TIMELINE_START_YEAR) return NODE_OFFSET_Y;
  if (year <= 1934) {
    const ratio = (year - TIMELINE_START_YEAR) / (1934 - TIMELINE_START_YEAR);
    return NODE_OFFSET_Y + ratio * PRE_1934_GAP;
  }
  return NODE_OFFSET_Y + PRE_1934_GAP + (year - 1934) * ROW_HEIGHT_PER_YEAR;
}

export function pixelToYear(py: number): number {
  const adjusted = py - NODE_OFFSET_Y;
  if (adjusted <= 0) return TIMELINE_START_YEAR;
  if (adjusted <= PRE_1934_GAP) {
    const ratio = adjusted / PRE_1934_GAP;
    return Math.round(
      TIMELINE_START_YEAR + ratio * (1934 - TIMELINE_START_YEAR),
    );
  }
  return Math.round(1934 + (adjusted - PRE_1934_GAP) / ROW_HEIGHT_PER_YEAR);
}

export interface YearRowScale {
  years: number[];

  yearToPixel(year: number): number;

  pixelToYear(py: number): number;

  ticks: number[];

  axisLength: number;
}

export function buildYearRowScale(techs: Technology[]): YearRowScale {
  const distinct = new Set<number>();
  for (const t of techs) distinct.add(t.start_year);
  const years =
    distinct.size > 0
      ? Array.from(distinct).sort((a, b) => a - b)
      : [...YEAR_TICKS];

  const rowHeight = ROW_HEIGHT_PER_YEAR;
  const last = years.length - 1;
  const rowPixel = (i: number): number => NODE_OFFSET_Y + i * rowHeight;

  const yearToPixelFn = (year: number): number => {
    if (year <= years[0]) {
      const gap = years.length > 1 ? years[1] - years[0] : 1;
      const off = ((year - years[0]) / gap) * rowHeight;
      return rowPixel(0) + Math.max(-rowHeight, off);
    }
    if (year >= years[last]) {
      const gap = years.length > 1 ? years[last] - years[last - 1] : 1;
      const off = ((year - years[last]) / gap) * rowHeight;
      return rowPixel(last) + Math.min(rowHeight, off);
    }

    let lo = 0;
    let hi = last;
    while (lo + 1 < hi) {
      const mid = (lo + hi) >> 1;
      if (years[mid] <= year) lo = mid;
      else hi = mid;
    }
    const frac = (year - years[lo]) / (years[hi] - years[lo]);
    return rowPixel(lo) + frac * rowHeight;
  };

  const pixelToYearFn = (py: number): number => {
    const idx = Math.round((py - NODE_OFFSET_Y) / rowHeight);
    return years[Math.min(last, Math.max(0, idx))];
  };

  return {
    years,
    yearToPixel: yearToPixelFn,
    pixelToYear: pixelToYearFn,
    ticks: years,

    axisLength: NODE_OFFSET_Y + last * rowHeight + rowHeight,
  };
}

export type TechLayoutMode = "vertical" | "horizontal";

export interface TechNodePosition {
  x: number;
  y: number;
}

export interface LayoutAxes {
  yearAxis: "x" | "y";
  catAxis: "x" | "y";

  timelineSides: ("left" | "right" | "top" | "bottom")[];

  categoryHeaderSide: "top" | "left";
}

export function axesForLayout(layout: TechLayoutMode): LayoutAxes {
  if (layout === "vertical") {
    return {
      yearAxis: "y",
      catAxis: "x",
      timelineSides: ["left", "right"],
      categoryHeaderSide: "top",
    };
  }
  return {
    yearAxis: "x",
    catAxis: "y",
    timelineSides: ["top", "bottom"],
    categoryHeaderSide: "left",
  };
}

export function getNodePosition(
  tech: Technology,
  layout: TechLayoutMode = "vertical",
  scale?: YearRowScale,

  useStoredRow = false,
): TechNodePosition {
  const rowPx = useStoredRow
    ? NODE_OFFSET_Y + Number(tech.position_y ?? 0) * ROW_HEIGHT_PER_YEAR
    : scale
      ? scale.yearToPixel(tech.start_year)
      : yearToPixel(tech.start_year);
  if (layout === "vertical") {
    return {
      x: tech.position_x * COLUMN_WIDTH + NODE_OFFSET_X,
      y: rowPx,
    };
  }

  return {
    x: rowPx,
    y: tech.position_x * COLUMN_WIDTH + NODE_OFFSET_X,
  };
}

export function pixelToRowFields(
  px: number,
  py: number,
  layout: TechLayoutMode = "vertical",
): { position_x: number; position_y: number } {
  const toCol = (p: number) =>
    Math.max(0, Math.round((p - NODE_OFFSET_X) / COLUMN_WIDTH));
  const toRow = (p: number) =>
    Math.max(0, Math.round((p - NODE_OFFSET_Y) / ROW_HEIGHT_PER_YEAR));
  return layout === "vertical"
    ? { position_x: toCol(px), position_y: toRow(py) }
    : { position_x: toCol(py), position_y: toRow(px) };
}

export function pixelToStoreFields(
  px: number,
  py: number,
  layout: TechLayoutMode = "vertical",
  scale?: YearRowScale,
): { start_year: number; position_x: number } {
  const yearOf = (p: number): number =>
    scale ? scale.pixelToYear(p) : pixelToYear(p);
  if (layout === "vertical") {
    return {
      start_year: yearOf(py),
      position_x: Math.max(0, Math.round((px - NODE_OFFSET_X) / COLUMN_WIDTH)),
    };
  }
  return {
    start_year: yearOf(px),
    position_x: Math.max(0, Math.round((py - NODE_OFFSET_X) / COLUMN_WIDTH)),
  };
}

export type TechSource = "self" | "vanilla" | "replaced";

export function inferTechSource(tech: Technology): TechSource {
  const raw = (tech as Technology & { source?: string }).source;
  if (raw === "imported") return "vanilla";
  if (raw === "replaced") return "replaced";
  return "self";
}

export const SOURCE_COLORS: Record<
  TechSource,
  { border: string; bg: string; label: string }
> = {
  self: { border: "#22c55e", bg: "rgba(34, 197, 94, 0.05)", label: "Mod-self" },
  vanilla: {
    border: "#94a3b8",
    bg: "rgba(148, 163, 184, 0.05)",
    label: "Vanilla import",
  },
  replaced: {
    border: "#f59e0b",
    bg: "rgba(245, 158, 11, 0.05)",
    label: "Replaced",
  },
};

export function inferLevelBadge(tech_id: string): string | null {
  const m = tech_id.match(/^(.+?)(\d+)$/);
  if (!m) return null;
  const num = Number(m[2]);
  if (num >= 1 && num <= 4) {
    return ["I", "II", "III", "IV"][num - 1];
  }
  return null;
}

export interface TechGraphNodeData {
  techId: string;
  label: string;
  cost: number;
  startYear: number;
  iconUrl: string | null;

  baseModIconUrls: string[];

  vanillaIconUrl: string | null;
  dbId: number;
  source: TechSource;
  levelBadge: string | null;

  layout: TechLayoutMode;

  layoutMode?: boolean;

  isBanner: boolean;

  categories: string[];
  prerequisites: string[];

  isReachable: boolean;

  reachabilityProjectId: number | null;
  reachabilityCountryTag: string;
  [key: string]: unknown;
}

export function techsToRFNodes(
  techs: Technology[],
  selectedId: number | null,
  projectId?: number,
  iconCacheBust?: number,
  layout: TechLayoutMode = "vertical",

  positionOverrides?: Map<number, TechNodePosition>,

  reachable?: ReachabilityMap | null,

  activeCountryTag?: string | null,

  locale: ContentLocale = "en",

  baseModDepKeys: string[] = [],

  availableTechIcons?: Set<string> | null,
  layoutMode = false,
): Node<TechGraphNodeData>[] {
  const projectIdOrNull = projectId ?? null;
  const tagUpper = (activeCountryTag ?? "").toUpperCase();

  const scale = buildYearRowScale(techs);
  const useStoredRow = techsUseStoredRow(techs);
  return techs.map((tech) => {
    const override = positionOverrides?.get(tech.id);
    const pos = override ?? getNodePosition(tech, layout, scale, useStoredRow);
    let iconUrl: string | null = null;
    if (projectId && availableTechIcons?.has(tech.tech_id)) {
      const url = `${API_BASE}/go/projects/${projectId}/tech-icons/${encodeURIComponent(tech.tech_id)}`;
      iconUrl = iconCacheBust ? `${url}?t=${iconCacheBust}` : url;
    }

    const vanillaIconUrl = `${API_BASE}/go/vanilla/tech-icon/${encodeURIComponent(tech.tech_id)}`;

    const baseModIconUrls: string[] =
      projectId && baseModDepKeys.length
        ? baseModDepKeys.map((dk) =>
            gfxUrl(projectId, dk, `GFX_${tech.tech_id}_medium`),
          )
        : [];

    return {
      id: String(tech.id),
      type: "techNode",
      position: pos,
      selected: tech.id === selectedId,
      data: {
        techId: tech.tech_id,

        label: resolveDisplayName(
          locale,
          tech.display_name_locs,
          tech.name_en,
          tech.name_zh,
          tech.name || tech.tech_id || "New Tech",
        ),
        cost: tech.research_cost,
        startYear: tech.start_year,
        iconUrl,
        baseModIconUrls,
        vanillaIconUrl,
        dbId: tech.id,
        source: inferTechSource(tech),
        layoutMode,
        levelBadge: inferLevelBadge(tech.tech_id),
        layout,
        isBanner: tech.is_banner ?? false,
        categories: tech.categories ?? [],
        prerequisites: tech.prerequisites ?? [],
        isReachable: reachable ? reachable.has(tech.tech_id) : false,
        reachabilityProjectId: projectIdOrNull,
        reachabilityCountryTag: tagUpper,
      },
    };
  });
}

function stripTrailingDigits(techId: string): string {
  return techId.replace(/\d+$/, "");
}

export function techsToRFEdges(techs: Technology[]): Edge[] {
  const techIdToDbId = new Map<string, number>();
  const techIdToBase = new Map<string, string>();
  for (const tech of techs) {
    techIdToDbId.set(tech.tech_id, tech.id);
    techIdToBase.set(tech.tech_id, stripTrailingDigits(tech.tech_id));
  }

  const edges: Edge[] = [];
  for (const tech of techs) {
    const targetBase = techIdToBase.get(tech.tech_id) ?? tech.tech_id;
    for (const prereqId of tech.prerequisites || []) {
      const sourceDbId = techIdToDbId.get(prereqId);
      if (sourceDbId === undefined) continue;
      const sourceBase = techIdToBase.get(prereqId) ?? prereqId;
      const isMainChain = sourceBase === targetBase && sourceBase.length > 0;
      edges.push({
        id: `prereq-${sourceDbId}-${tech.id}`,
        source: String(sourceDbId),
        target: String(tech.id),
        type: "techPrereqEdge",

        data: { prereqId, targetDbId: tech.id, isMainChain },
        style: isMainChain
          ? { stroke: "#f59e0b", strokeWidth: 2.5 }
          : { stroke: "#94a3b8", strokeWidth: 1.5, strokeOpacity: 0.55 },

        animated: false,
      });
    }

    for (const otherTechId of tech.xor || []) {
      const otherDbId = techIdToDbId.get(otherTechId);
      if (otherDbId === undefined) continue;
      if (tech.id >= otherDbId) continue;
      edges.push({
        id: `xor-${tech.id}-${otherDbId}`,
        source: String(tech.id),
        target: String(otherDbId),
        type: "techXorEdge",
        deletable: false,
        data: { sourceTechId: tech.tech_id, targetTechId: otherTechId },
        style: {
          stroke: "var(--color-destructive)",
          strokeWidth: 2,
          strokeDasharray: "6 3",
        },
        animated: false,
      });
    }
  }
  return edges;
}

export function parseTechEdgeId(edgeId: string): {
  sourceDbId: number;
  targetDbId: number;
} | null {
  const m = edgeId.match(/^prereq-(\d+)-(\d+)$/);
  if (!m) return null;
  return {
    sourceDbId: Number(m[1]),
    targetDbId: Number(m[2]),
  };
}

export function filterByTechLine(
  techs: Technology[],
  techLine: string,
): Technology[] {
  return techs.filter((t) => t.tech_line === techLine);
}

export function filterByLedger(
  techs: Technology[],
  ledger: LedgerKey,
): Technology[] {
  return techs.filter((t) => resolveLedger(t.tech_line) === ledger);
}

export function searchTechs(techs: Technology[], query: string): Technology[] {
  const q = query.trim().toLowerCase();
  if (!q) return [];
  return techs.filter(
    (t) =>
      t.tech_id.toLowerCase().includes(q) ||
      (t.name_en || "").toLowerCase().includes(q) ||
      (t.name_zh || "").toLowerCase().includes(q) ||
      (t.name || "").toLowerCase().includes(q),
  );
}

const UNCATEGORIZED_KEY = "_uncategorized";

export interface CategoryColumn {
  category: string;
  startCol: number;
  width: number;

  names?: Record<string, string>;
}

export interface AutoLayoutResult {
  positions: Map<number, { position_x: number }>;

  columns: CategoryColumn[];
}

export function autoLayoutTechs(techs: Technology[]): AutoLayoutResult {
  const categories = new Set<string>();
  for (const t of techs) {
    categories.add((t.categories ?? [])[0] ?? UNCATEGORIZED_KEY);
  }
  const categoryOrder = Array.from(categories).sort();

  const grouped = new Map<string, Technology[]>();
  const maxStackPerCat: Record<string, number> = {};
  for (const t of techs) {
    const cat = (t.categories ?? [])[0] ?? UNCATEGORIZED_KEY;
    const key = `${t.start_year}__${cat}`;
    if (!grouped.has(key)) grouped.set(key, []);
    grouped.get(key)!.push(t);
  }
  for (const [key, list] of grouped) {
    const cat = key.split("__", 2)[1];
    const cur = maxStackPerCat[cat] ?? 1;
    if (list.length > cur) maxStackPerCat[cat] = list.length;
  }

  const columns: CategoryColumn[] = [];
  let acc = 0;
  for (const cat of categoryOrder) {
    const width = Math.max(1, maxStackPerCat[cat] ?? 1);
    columns.push({ category: cat, startCol: acc, width });
    acc += width;
  }
  const colInfo = new Map(columns.map((c) => [c.category, c]));

  const positions = new Map<number, { position_x: number }>();
  for (const [, list] of grouped) {
    list.sort((a, b) => a.tech_id.localeCompare(b.tech_id));
    list.forEach((t, i) => {
      const cat = (t.categories ?? [])[0] ?? UNCATEGORIZED_KEY;
      const col = colInfo.get(cat)!;
      positions.set(t.id, { position_x: col.startCol + i });
    });
  }

  return { positions, columns };
}

export function needsAutoLayout(techs: Technology[]): boolean {
  if (techs.length < 2) return false;
  const seen = new Set<string>();
  for (const t of techs) {
    const key = `${t.start_year}_${t.position_x}`;
    if (seen.has(key)) return true;
    seen.add(key);
  }
  return false;
}

export function formatCategoryLabel(category: string): string {
  if (category === UNCATEGORIZED_KEY) return "Uncategorized";
  return category.replace(/_/g, " ").replace(/\b\w/g, (c) => c.toUpperCase());
}

interface BfsLayerResult {
  positions: Map<number, { layer: number; index: number }>;
}

function _bfsLayerTechs(techs: Technology[]): BfsLayerResult {
  if (techs.length === 0) return { positions: new Map() };
  const byId = new Map<string, Technology>(techs.map((t) => [t.tech_id, t]));
  const inBatch = new Set<string>(techs.map((t) => t.tech_id));

  const parentsOf = new Map<string, Set<string>>();
  for (const t of techs) {
    const parents = new Set<string>();
    for (const ref of t.prerequisites ?? []) {
      if (inBatch.has(ref) && ref !== t.tech_id) parents.add(ref);
    }
    parentsOf.set(t.tech_id, parents);
  }

  const depth = new Map<string, number>();
  const remaining = new Map(parentsOf);
  let layerIdx = 0;
  while (remaining.size > 0) {
    const ready: string[] = [];
    for (const [tid, parents] of remaining) {
      let allReady = true;
      for (const p of parents) {
        if (!depth.has(p)) {
          allReady = false;
          break;
        }
      }
      if (allReady) ready.push(tid);
    }
    if (ready.length === 0) break;
    for (const tid of ready) {
      depth.set(tid, layerIdx);
      remaining.delete(tid);
    }
    layerIdx += 1;
  }

  const byDepth = new Map<number, string[]>();
  for (const [tid, d] of depth) {
    if (!byDepth.has(d)) byDepth.set(d, []);
    byDepth.get(d)!.push(tid);
  }

  const out = new Map<number, { layer: number; index: number }>();
  for (const [d, tids] of byDepth) {
    tids.sort();
    tids.forEach((tid, i) => {
      const t = byId.get(tid)!;
      out.set(t.id, { layer: d, index: i });
    });
  }
  return { positions: out };
}

export function bfsTreeLayout(
  techs: Technology[],
  folder: string,

  sections: TechSections,
): {
  positions: Map<number, { position_x: number }>;
  columns: CategoryColumn[];
} {
  if (techs.length === 0) return { positions: new Map(), columns: [] };

  const sectionByTechId = new Map<number, string>();
  const sectionLabels = new Map<string, Record<string, string>>();
  for (const t of techs) {
    const section = resolveVisualSection(t, folder, sections);
    sectionByTechId.set(t.id, section.subKey);
    if (!sectionLabels.has(section.subKey)) {
      sectionLabels.set(section.subKey, section.names);
    }
  }

  const knownOrder = getFolderSubSections(folder, sections).map(
    (s) => s.sub_key,
  );
  const seen = new Set<string>();
  const sectionOrder: string[] = [];
  for (const known of knownOrder) {
    if (sectionByTechId.size === 0) break;

    for (const sk of sectionByTechId.values()) {
      if (sk === known) {
        if (!seen.has(known)) {
          seen.add(known);
          sectionOrder.push(known);
        }
        break;
      }
    }
  }

  for (const sk of sectionByTechId.values()) {
    if (!seen.has(sk)) {
      seen.add(sk);
      sectionOrder.push(sk);
    }
  }

  const grouped = new Map<string, Technology[]>();
  const maxStackPerSection: Record<string, number> = {};
  for (const t of techs) {
    const sub_key = sectionByTechId.get(t.id)!;
    const key = `${t.start_year}__${sub_key}`;
    if (!grouped.has(key)) grouped.set(key, []);
    grouped.get(key)!.push(t);
  }
  for (const [key, list] of grouped) {
    const sub_key = key.split("__", 2)[1];
    const cur = maxStackPerSection[sub_key] ?? 1;
    if (list.length > cur) maxStackPerSection[sub_key] = list.length;
  }

  const columns: CategoryColumn[] = [];
  let acc = 0;
  for (const sub_key of sectionOrder) {
    const width = Math.max(1, maxStackPerSection[sub_key] ?? 1);
    columns.push({
      category: sub_key,
      startCol: acc,
      width,
      names: sectionLabels.get(sub_key) ?? {},
    });
    acc += width;
  }
  const colInfo = new Map(columns.map((c) => [c.category, c]));

  const positions = new Map<number, { position_x: number }>();
  for (const [, list] of grouped) {
    list.sort((a, b) => a.tech_id.localeCompare(b.tech_id));
    list.forEach((t, i) => {
      const sub_key = sectionByTechId.get(t.id)!;
      const col = colInfo.get(sub_key);
      if (!col) return;
      positions.set(t.id, { position_x: col.startCol + i });
    });
  }

  return { positions, columns };
}

export function buildTreePositionOverrides(
  techs: Technology[],
  treeLayout: { positions: Map<number, { position_x: number }> },
  layout: TechLayoutMode = "vertical",
  scale?: YearRowScale,
): Map<number, TechNodePosition> {
  const overrides = new Map<number, TechNodePosition>();
  const useStoredRow = techsUseStoredRow(techs);
  for (const tech of techs) {
    if (tech.source !== "imported") continue;
    const colInfo = treeLayout.positions.get(tech.id);
    if (!colInfo) continue;
    const synthetic = { ...tech, position_x: colInfo.position_x } as Technology;
    overrides.set(
      tech.id,
      getNodePosition(synthetic, layout, scale, useStoredRow),
    );
  }
  return overrides;
}

export const DOCTRINE_PANEL_WIDTH = 280;
export const DOCTRINE_PANEL_GAP = 30;
export const DOCTRINE_LAYER_HEIGHT = 100;
export const DOCTRINE_NODE_GAP_X = 75;

export interface DoctrinePanel {
  category: string;

  panelIndex: number;

  maxLayer: number;

  maxLayerWidth: number;
}

export interface DoctrineLayoutResult {
  nodePositions: Map<number, TechNodePosition>;

  panels: DoctrinePanel[];
}

export function bfsDoctrineLayout(techs: Technology[]): DoctrineLayoutResult {
  if (techs.length === 0) {
    return { nodePositions: new Map(), panels: [] };
  }

  const groupsByCat = new Map<string, Technology[]>();
  for (const t of techs) {
    const cat = (t.categories ?? [])[0] ?? UNCATEGORIZED_KEY;
    if (!groupsByCat.has(cat)) groupsByCat.set(cat, []);
    groupsByCat.get(cat)!.push(t);
  }

  const categoryOrder = Array.from(groupsByCat.keys()).sort();

  const nodePositions = new Map<number, TechNodePosition>();
  const panels: DoctrinePanel[] = [];

  categoryOrder.forEach((cat, panelIndex) => {
    const groupTechs = groupsByCat.get(cat) ?? [];
    const { positions: layerPos } = _bfsLayerTechs(groupTechs);

    let maxLayer = 0;
    const layerCounts = new Map<number, number>();
    for (const [, { layer }] of layerPos) {
      maxLayer = Math.max(maxLayer, layer);
      layerCounts.set(layer, (layerCounts.get(layer) ?? 0) + 1);
    }
    const maxLayerWidth = Math.max(0, ...layerCounts.values());
    panels.push({ category: cat, panelIndex, maxLayer, maxLayerWidth });

    const panelStartX =
      panelIndex * (DOCTRINE_PANEL_WIDTH + DOCTRINE_PANEL_GAP);

    for (const [techDbId, { layer, index }] of layerPos) {
      const layerCount = layerCounts.get(layer) ?? 1;

      const panelCenter = panelStartX + DOCTRINE_PANEL_WIDTH / 2;
      const totalWidth = (layerCount - 1) * DOCTRINE_NODE_GAP_X;
      const startX = panelCenter - totalWidth / 2;
      const x = startX + index * DOCTRINE_NODE_GAP_X;

      const y = 60 + layer * DOCTRINE_LAYER_HEIGHT;
      nodePositions.set(techDbId, { x, y });
    }
  });

  return { nodePositions, panels };
}

export interface GridGeometry {
  name: string;

  originX: number;
  originY: number;

  slotWidth: number;
  slotHeight: number;

  sizeWidth: number;
  sizeHeight: number;

  format: string;
}

export interface GuiYearLabel {
  name: string;

  text: string;

  x: number;
  y: number;

  borderX: number;
  borderY: number;

  maxWidth: number;

  align: "left" | "center" | "right";
}

export interface FolderGeometry {
  folder: string;

  grids: GridGeometry[];

  yearLabels: GuiYearLabel[];
}

export function isColumnMajorFormat(
  format: string | null | undefined,
): boolean {
  return (
    String(format ?? "")
      .trim()
      .toLowerCase() === "left"
  );
}

const FORMAT_ALIGN: Record<string, { fx: number; fy: number }> = {
  left: { fx: 0, fy: 0.5 },
  up: { fx: 0.5, fy: 0 },
  right: { fx: 1, fy: 0.5 },
  down: { fx: 0.5, fy: 1 },
  center: { fx: 0.5, fy: 0.5 },
  upper_left: { fx: 0, fy: 0 },
};

export function gridOriginOffset(grid: GridGeometry): {
  dx: number;
  dy: number;
} {
  const key = String(grid.format ?? "")
    .trim()
    .toLowerCase();
  const { fx, fy } = FORMAT_ALIGN[key] ?? { fx: 0, fy: 0 };

  return {
    dx: fx * ((grid.sizeWidth || 0) - grid.slotWidth) + 0,
    dy: fy * ((grid.sizeHeight || 0) - grid.slotHeight) + 0,
  };
}

export function cellToPixel(
  cell: { x: number; y: number },
  grid: GridGeometry,
): TechNodePosition {
  const { dx, dy } = gridOriginOffset(grid);
  const ox = grid.originX + dx;
  const oy = grid.originY + dy;

  const cx = Math.trunc(cell.x);
  const cy = Math.trunc(cell.y);
  if (isColumnMajorFormat(grid.format)) {
    return { x: ox + cy * grid.slotWidth, y: oy + cx * grid.slotHeight };
  }
  return { x: ox + cx * grid.slotWidth, y: oy + cy * grid.slotHeight };
}

export function pixelToCell(
  px: number,
  py: number,
  grid: GridGeometry,
): { x: number; y: number } {
  const w = grid.slotWidth || 1;
  const h = grid.slotHeight || 1;
  const { dx, dy } = gridOriginOffset(grid);
  const alongX = Math.round((px - grid.originX - dx) / w);
  const alongY = Math.round((py - grid.originY - dy) / h);

  return isColumnMajorFormat(grid.format)
    ? { x: alongY, y: alongX }
    : { x: alongX, y: alongY };
}

export function pixelToCellFields(
  px: number,
  py: number,
  grid: GridGeometry,
): { position_x: number; position_y: number } {
  const cell = pixelToCell(px, py, grid);
  return { position_x: cell.x, position_y: cell.y };
}

export interface GuiNode {
  type?: string | null;
  name?: string | null;
  position?: { x?: number; y?: number } | null;

  size?: { width?: number; height?: number } | null;
  format?: string | null;
  text?: string | null;

  maxwidth?: number | null;

  orientation?: string | null;
  children?: GuiNode[] | null;
  raw?: Record<string, unknown> | null;
}

function numOr0(v: unknown): number {
  return typeof v === "number" && Number.isFinite(v) ? v : 0;
}

function anchorOrigin(
  orientation: string | null | undefined,
  parentX: number,
  parentY: number,
  parentW: number,
  parentH: number,
): [number, number] {
  const o = String(orientation ?? "UPPER_LEFT").toUpperCase();
  let ax = parentX;
  let ay = parentY;
  if (o.includes("RIGHT")) ax = parentX + parentW;
  else if (o.includes("LEFT")) ax = parentX;
  else if (o.includes("CENTER")) ax = parentX + Math.floor(parentW / 2);
  const vTop =
    o.includes("UPPER") || o.includes("TOP") || /(^|_)UP($|_)/.test(o);
  if (o.includes("LOWER") || o.includes("DOWN") || o.includes("BOTTOM"))
    ay = parentY + parentH;
  else if (vTop) ay = parentY;
  else if (o.includes("CENTER")) ay = parentY + Math.floor(parentH / 2);
  return [ax, ay];
}

function readSlotSize(value: unknown): { w: number; h: number } | null {
  if (!value || typeof value !== "object") return null;
  const rec = value as Record<string, unknown>;
  const w = rec.width;
  const h = rec.height;
  if (typeof w !== "number" || typeof h !== "number") return null;
  return { w, h };
}

export function findFolderContainer(
  root: GuiNode | null,
  folder: string,
): GuiNode | null {
  if (!root || !folder) return null;
  const want = folder.trim().toLowerCase();
  const stack: GuiNode[] = [root];
  while (stack.length > 0) {
    const node = stack.pop()!;
    if (
      String(node.name ?? "")
        .trim()
        .toLowerCase() === want
    )
      return node;
    for (const child of node.children ?? []) stack.push(child);
  }
  return null;
}

const YEAR_LABEL_NAME = /_year\d+$/i;

export function parseFolderGeometry(
  container: GuiNode | null,
  folder: string,
): FolderGeometry | null {
  if (!container) return null;
  const grids: GridGeometry[] = [];
  const yearLabels: GuiYearLabel[] = [];

  const walk = (
    node: GuiNode,
    parentX = 0,
    parentY = 0,
    parentW = 0,
    parentH = 0,
  ) => {
    const [anchorX, anchorY] = anchorOrigin(
      node.orientation,
      parentX,
      parentY,
      parentW,
      parentH,
    );
    const absoluteX = anchorX + (node.position?.x ?? 0);
    const absoluteY = anchorY + (node.position?.y ?? 0);
    const type = String(node.type ?? "").toLowerCase();
    if (type === "gridboxtype") {
      const slot = readSlotSize(node.raw?.slotsize);

      const size = readSlotSize(node.size);
      grids.push({
        name: String(node.name ?? ""),
        originX: absoluteX,
        originY: absoluteY,

        slotWidth: slot?.w ?? 70,
        slotHeight: slot?.h ?? 70,
        sizeWidth: size?.w ?? 0,
        sizeHeight: size?.h ?? 0,
        format: String(node.format ?? ""),
      });
    } else if (
      type === "instanttextboxtype" &&
      YEAR_LABEL_NAME.test(String(node.name ?? ""))
    ) {
      const rawAlign = String(node.format ?? "")
        .trim()
        .toLowerCase();
      yearLabels.push({
        name: String(node.name ?? ""),
        text: String(node.text ?? ""),
        x: absoluteX,
        y: absoluteY,

        borderX: numOr0(
          (node.raw?.bordersize as Record<string, unknown> | undefined)?.x,
        ),
        borderY: numOr0(
          (node.raw?.bordersize as Record<string, unknown> | undefined)?.y,
        ),
        maxWidth: numOr0(node.maxwidth),

        align:
          rawAlign === "center" || rawAlign === "centre"
            ? "center"
            : rawAlign === "right"
              ? "right"
              : "left",
      });
    }

    const own = readSlotSize(node.size);
    for (const child of node.children ?? [])
      walk(child, absoluteX, absoluteY, own?.w ?? 0, own?.h ?? 0);
  };

  for (const child of container.children ?? []) walk(child);

  if (grids.length === 0) return null;

  const columnMajorCount = grids.filter((g) =>
    isColumnMajorFormat(g.format),
  ).length;
  const columnMajor = columnMajorCount * 2 > grids.length;
  yearLabels.sort((a, b) => (columnMajor ? a.x - b.x : a.y - b.y));
  return { folder, grids, yearLabels };
}

export function assignGridsToTechs(
  techs: Technology[],
  geometry: FolderGeometry,
  opts?: { assignFallback?: boolean },
): Map<number, GridGeometry> {
  const out = new Map<number, GridGeometry>();
  if (geometry.grids.length === 0) return out;
  const fallback = geometry.grids[0];
  const shouldAssignFallback = opts?.assignFallback !== false;

  const byTechId = new Map<string, Technology>();
  for (const t of techs) byTechId.set(t.tech_id, t);

  const childrenOf = new Map<string, string[]>();
  for (const t of techs) {
    for (const raw of t.prerequisites ?? []) {
      const list = Array.isArray(raw) ? raw : [raw];
      for (const p of list) {
        const key = String(p);
        if (!childrenOf.has(key)) childrenOf.set(key, []);
        childrenOf.get(key)!.push(t.tech_id);
      }
    }
  }

  for (const grid of geometry.grids) {
    const rootId = grid.name.replace(/_tree$/i, "");
    const root = byTechId.get(rootId);
    if (!root) continue;
    const queue = [root.tech_id];
    const seen = new Set<string>();
    while (queue.length > 0) {
      const cur = queue.shift()!;
      if (seen.has(cur)) continue;
      seen.add(cur);
      const tech = byTechId.get(cur);
      if (!tech || out.has(tech.id)) continue;
      out.set(tech.id, grid);
      for (const next of childrenOf.get(cur) ?? []) queue.push(next);
    }
  }

  if (shouldAssignFallback) {
    for (const t of techs) {
      if (!out.has(t.id)) out.set(t.id, fallback);
    }
  }
  return out;
}

function legacyTechsUseStoredRow(techs: Technology[]): boolean {
  return techs.some(
    (t) =>
      (t.position_y_is_explicit === null ||
        t.position_y_is_explicit === undefined) &&
      Number(t.position_y ?? 0) !== 0,
  );
}

export function techsUseStoredRow(techs: Technology[]): boolean {
  return (
    techs.some((t) => t.position_y_is_explicit === true) ||
    legacyTechsUseStoredRow(techs)
  );
}

export function techUsesStoredRow(
  tech: Technology,
  techs: Technology[],
): boolean {
  if (tech.position_y_is_explicit === true) return true;
  if (tech.position_y_is_explicit === false) return false;
  return legacyTechsUseStoredRow(techs);
}

export function buildStoredRowTicks(
  techs: Technology[],
): Array<{ row: number; year: number }> {
  const minYearOfRow = new Map<number, number>();
  for (const t of techs) {
    const row = Number(t.position_y ?? 0);
    const cur = minYearOfRow.get(row);
    if (cur == null || t.start_year < cur) minYearOfRow.set(row, t.start_year);
  }
  return [...minYearOfRow.entries()]
    .map(([row, year]) => ({ row, year }))
    .sort((a, b) => a.row - b.row);
}

export function buildRowTicks(
  techs: Technology[],
): Array<{ row: number; years: number[] }> {
  const yearsOfRow = new Map<number, Set<number>>();
  for (const t of techs) {
    const row = Number(t.position_y ?? 0);
    let set = yearsOfRow.get(row);
    if (!set) yearsOfRow.set(row, (set = new Set()));
    set.add(t.start_year);
  }
  return [...yearsOfRow.entries()]
    .map(([row, years]) => ({ row, years: [...years].sort((a, b) => a - b) }))
    .sort((a, b) => a.row - b.row);
}

export function buildTimelineTicks(
  techs: Technology[],
  scale: YearRowScale,
): Array<{ key: string; text: string; pixel: number }> {
  if (techsUseStoredRow(techs)) return [];
  return scale.ticks.map((year) => ({
    key: String(year),
    text: String(year),
    pixel: scale.yearToPixel(year),
  }));
}

export function buildRowBands(
  techs: Technology[],
): Array<{ key: string; start: number; size: number; shade: boolean }> {
  const ticks = buildRowTicks(techs);
  return ticks.map(({ row }, i) => {
    const nextRow = i + 1 < ticks.length ? ticks[i + 1].row : row + 2;
    return {
      key: `r${row}`,
      start: NODE_OFFSET_Y + (row - 0.5) * ROW_HEIGHT_PER_YEAR,
      size: Math.max(1, nextRow - row) * ROW_HEIGHT_PER_YEAR,
      shade: i % 2 === 0,
    };
  });
}

export function parseYearRowTable(script: string): Array<[string, number]> {
  const rows: Array<[string, number]> = [];

  const re = /^\s*(@\d{3,4})\s*=\s*(-?\d+)\s*$/gm;
  let m: RegExpExecArray | null;
  while ((m = re.exec(script)) !== null) {
    rows.push([m[1], Number(m[2])]);
  }
  return rows.sort((a, b) => a[1] - b[1]);
}

export function yearToRowFromTable(
  year: number,
  rows: Array<[string, number]>,
): number {
  for (let i = rows.length - 1; i >= 0; i--) {
    const yr = Number(rows[i][0].slice(1));
    if (!Number.isFinite(yr)) continue;
    if (year >= yr) return rows[i][1];
  }
  return 0;
}

export function computeRowBackfill(
  techs: Technology[],
  scale: YearRowScale,
  yearRows?: Array<[string, number]> | null,
): Array<{ id: number; position_y: number }> {
  const denseRowOf = new Map<number, number>();
  scale.years.forEach((y, i) => denseRowOf.set(y, i));
  const useTable = yearRows != null && yearRows.length > 0;
  const plan: Array<{ id: number; position_y: number }> = [];
  for (const t of techs) {
    const row = useTable
      ? yearToRowFromTable(t.start_year, yearRows)
      : (denseRowOf.get(t.start_year) ?? 0);
    if (Number(t.position_y ?? 0) === row) continue;
    plan.push({ id: t.id, position_y: row });
  }
  return plan;
}

export function buildGeometryPositions(
  techs: Technology[],
  geometry: FolderGeometry,
  scale: YearRowScale,
): Map<number, TechNodePosition> {
  const gridOf = assignGridsToTechs(techs, geometry, { assignFallback: false });

  const rowOf = new Map<number, number>();
  scale.years.forEach((y, i) => rowOf.set(y, i));

  const out = new Map<number, TechNodePosition>();
  for (const t of techs) {
    const grid = gridOf.get(t.id);
    if (!grid) continue;
    const cellY = techUsesStoredRow(t, techs)
      ? Number(t.position_y ?? 0)
      : (rowOf.get(t.start_year) ?? 0);

    out.set(t.id, cellToPixel({ x: t.position_x, y: cellY }, grid));
  }
  return out;
}

export type OrphanKind = "no_folder" | "unmatched_grid";

export interface OrphanGroup {
  kind: OrphanKind;
  techs: Technology[];
}

export function classifyOrphans(
  orphans: Technology[],
  geometries: Map<string, FolderGeometry>,
): OrphanGroup[] {
  void geometries;
  const noFolder: Technology[] = [];
  const unmatched: Technology[] = [];
  for (const t of orphans) {
    (String(t.folder ?? "") === "" ? noFolder : unmatched).push(t);
  }
  const out: OrphanGroup[] = [];
  if (noFolder.length > 0) out.push({ kind: "no_folder", techs: noFolder });
  if (unmatched.length > 0)
    out.push({ kind: "unmatched_grid", techs: unmatched });
  return out;
}

export interface FolderGroupLayout {
  folder: string;
  geometry: FolderGeometry;

  offsetX: number;
  offsetY: number;

  minX: number;
  minY: number;
  maxX: number;
  maxY: number;
}

export const FOLDER_GROUP_GAP = 240;

export function buildFolderGroupPositions(
  techs: Technology[],
  geometries: Map<string, FolderGeometry>,
  scale: YearRowScale,
  primaryFolder: string,
): {
  positions: Map<number, TechNodePosition>;

  offsets: Map<number, number>;
  groups: FolderGroupLayout[];
} {
  const byFolder = new Map<string, Technology[]>();
  for (const t of techs) {
    const f = String(t.folder ?? "");
    if (!geometries.has(f)) continue;
    const list = byFolder.get(f);
    if (list) list.push(t);
    else byFolder.set(f, [t]);
  }

  const folders = [...byFolder.keys()].sort((a, b) => {
    if (a === primaryFolder) return -1;
    if (b === primaryFolder) return 1;
    const d = (byFolder.get(b)?.length ?? 0) - (byFolder.get(a)?.length ?? 0);
    return d !== 0 ? d : a.localeCompare(b);
  });

  const positions = new Map<number, TechNodePosition>();
  const offsets = new Map<number, number>();
  const groups: FolderGroupLayout[] = [];
  let cursorY = 0;

  for (const folder of folders) {
    const geo = geometries.get(folder)!;
    const groupTechs = byFolder.get(folder)!;
    const local = buildGeometryPositions(groupTechs, geo, scale);
    if (local.size === 0) continue;

    let minX = Infinity;
    let minY = Infinity;
    let maxX = -Infinity;
    let maxY = -Infinity;
    for (const p of local.values()) {
      if (p.x < minX) minX = p.x;
      if (p.y < minY) minY = p.y;
      if (p.x > maxX) maxX = p.x;
      if (p.y > maxY) maxY = p.y;
    }

    const offsetY = folder === primaryFolder ? 0 : cursorY - minY;
    for (const [id, p] of local) {
      positions.set(id, { x: p.x, y: p.y + offsetY });
      offsets.set(id, offsetY);
    }
    groups.push({
      folder,
      geometry: geo,
      offsetX: 0,
      offsetY,
      minX,
      minY: minY + offsetY,
      maxX,
      maxY: maxY + offsetY,
    });
    cursorY = maxY + offsetY + FOLDER_GROUP_GAP;
  }

  return { positions, offsets, groups };
}

export interface FitBounds {
  x: number;
  y: number;
  width: number;
  height: number;
}

export function computeFitBounds(
  positions: Iterable<TechNodePosition>,
  nodeWidth = 60,
  nodeHeight = 60,
): FitBounds | null {
  let minX = Infinity;
  let minY = Infinity;
  let maxX = -Infinity;
  let maxY = -Infinity;
  let count = 0;
  for (const p of positions) {
    count += 1;
    if (p.x < minX) minX = p.x;
    if (p.y < minY) minY = p.y;
    if (p.x + nodeWidth > maxX) maxX = p.x + nodeWidth;
    if (p.y + nodeHeight > maxY) maxY = p.y + nodeHeight;
  }
  if (count === 0) return null;
  return { x: minX, y: minY, width: maxX - minX, height: maxY - minY };
}

export function boundsFitInViewport(
  bounds: FitBounds | null,
  viewportWidth: number,
  viewportHeight: number,
  zoom: number,
): boolean {
  if (!bounds) return true;
  return (
    bounds.width * zoom <= viewportWidth &&
    bounds.height * zoom <= viewportHeight
  );
}

export interface AutoLayoutImpact {
  moved: number;

  importedMoved: number;

  flippedToCreated: number;

  destructive: boolean;
}

export function computeAutoLayoutImpact(
  techs: Technology[],
  plan: { positions: Map<number, { position_x: number }> },
): AutoLayoutImpact {
  let moved = 0;
  let importedMoved = 0;
  for (const t of techs) {
    const next = plan.positions.get(t.id);
    if (!next || next.position_x === t.position_x) continue;
    moved += 1;
    if (t.source === "imported") importedMoved += 1;
  }
  return {
    moved,
    importedMoved,
    flippedToCreated: importedMoved,
    destructive: importedMoved > 0,
  };
}
