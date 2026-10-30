import type { Node, Edge } from "@xyflow/react";
import type { CustomTrait } from "@/types";
import { resolveDisplayName } from "@/lib/i18n/localizedName";
import type { ContentLocale } from "@/lib/store/contentLocaleStore";
import type { TraitGraphNodeData } from "./TraitGraphNode";
import type { TraitParentEdgeData } from "./TraitParentEdge";

export const TRAIT_CELL_WIDTH = 150;
export const TRAIT_CELL_HEIGHT = 130;
export const TRAIT_OFFSET_X = 60;
export const TRAIT_OFFSET_Y = 40;

export const TRAIT_NODE_WIDTH = 120;
export const TRAIT_NODE_HEIGHT = 86;

export function gridToPixel(
  column: number,
  row: number,
): { x: number; y: number } {
  return {
    x: column * TRAIT_CELL_WIDTH + TRAIT_OFFSET_X,
    y: row * TRAIT_CELL_HEIGHT + TRAIT_OFFSET_Y,
  };
}

export function pixelToGrid(
  px: number,
  py: number,
): { column: number; row: number } {
  return {
    column: Math.max(0, Math.round((px - TRAIT_OFFSET_X) / TRAIT_CELL_WIDTH)),
    row: Math.max(0, Math.round((py - TRAIT_OFFSET_Y) / TRAIT_CELL_HEIGHT)),
  };
}

export function readRawBlocks(
  rawExtras: Record<string, unknown> | null | undefined,
  key: string,
): string[] {
  const v = rawExtras?.[key];
  if (typeof v === "string") return v.trim() ? [v] : [];
  if (Array.isArray(v))
    return v.filter((x): x is string => typeof x === "string" && !!x.trim());
  return [];
}

export function parseTokenList(text: string): string[] {
  return text
    .split("\n")
    .map((line) => {
      const hash = line.indexOf("#");
      return (hash >= 0 ? line.slice(0, hash) : line).trim();
    })
    .flatMap((line) => (line ? line.split(/\s+/) : []))
    .filter(Boolean);
}

export function serializeTokenList(tokens: string[]): string {
  const cleaned = tokens.map((t) => t.trim()).filter(Boolean);
  if (cleaned.length === 0) return "";
  const body = cleaned.join("\n");
  return body.includes("\n") ? body : body + "\n";
}

export interface ParentBlock {
  traits: string[];

  numParentsNeeded: number | null;
}

export function parseParentBlock(text: string): ParentBlock {
  const stripped = text
    .split("\n")
    .map((line) => {
      const hash = line.indexOf("#");
      return hash >= 0 ? line.slice(0, hash) : line;
    })
    .join("\n");
  const traitsMatch = /traits\s*=\s*\{([^}]*)\}/.exec(stripped);
  const numMatch = /num_parents_needed\s*=\s*(-?\d+)/.exec(stripped);
  const parsedNum = numMatch ? parseInt(numMatch[1], 10) : NaN;
  return {
    traits: traitsMatch ? parseTokenList(traitsMatch[1]) : [],

    numParentsNeeded:
      Number.isFinite(parsedNum) && parsedNum >= 0 ? parsedNum : null,
  };
}

export function withAnyParents(
  rawExtras: Record<string, unknown> | null | undefined,
  nextTokens: string[],
): Record<string, unknown> {
  const current = rawExtras ?? {};
  const existing = readRawBlocks(current, "any_parent");

  const currentTokens = existing.flatMap(parseTokenList);
  const cleaned = nextTokens.map((t) => t.trim()).filter(Boolean);
  const deduped = Array.from(new Set(cleaned));

  if (sameTokens(currentTokens, deduped))
    return current as Record<string, unknown>;

  const next = { ...current };
  if (deduped.length === 0) {
    delete next.any_parent;
  } else {
    next.any_parent = serializeTokenList(deduped);
  }
  return next;
}

export function sameTokens(a: string[], b: string[]): boolean {
  if (a.length !== b.length) return false;
  const sortedA = [...a].sort();
  const sortedB = [...b].sort();
  return sortedA.every((x, i) => x === sortedB[i]);
}

export function autoColumnForTraitType(traitType: string | undefined): number {
  switch ((traitType || "").trim()) {
    case "assignable_trait":
    case "assignable_terrain_trait":
      return 1;
    case "field_marshal":
      return 2;
    default:
      return 0;
  }
}

export interface TraitTreeNodeModel {
  dbId: number;
  traitId: string;
  label: string;

  guiRow: number;

  column: number;

  columnIsAuto: boolean;
  traitType: string;
  anyParents: string[];

  parentBlocks: ParentBlock[];

  allParents: string[];
  mutuallyExclusive: string[];
}

function readIntExtra(
  extras: Record<string, unknown> | undefined,
  key: string,
): number {
  const v = extras?.[key];
  return typeof v === "number" && Number.isFinite(v) ? Math.trunc(v) : -1;
}

export function isInTree(trait: CustomTrait): boolean {
  return (
    readIntExtra(trait.extra_fields as Record<string, unknown>, "gui_row") >= 0
  );
}

export function buildTraitTreeNodes(
  traits: CustomTrait[],
  locale: ContentLocale = "en",
): TraitTreeNodeModel[] {
  return traits.filter(isInTree).map((trait) => {
    const extras = trait.extra_fields as Record<string, unknown> | undefined;
    const raw = trait.raw_extras as Record<string, unknown> | undefined;
    const guiRow = readIntExtra(extras, "gui_row");
    const guiColumn = readIntExtra(extras, "gui_column");
    const traitType =
      typeof extras?.trait_type === "string" ? extras.trait_type : "";
    const columnIsAuto = guiColumn < 0;
    const mutexRaw =
      typeof extras?.mutually_exclusive === "string"
        ? extras.mutually_exclusive
        : "";

    return {
      dbId: trait.id,
      traitId: trait.trait_id,
      label: resolveDisplayName(
        locale,
        trait.display_name_locs,
        trait.name_en,
        trait.name_zh,
        trait.trait_id,
      ),
      guiRow,
      column: columnIsAuto ? autoColumnForTraitType(traitType) : guiColumn,
      columnIsAuto,
      traitType,
      anyParents: readRawBlocks(raw, "any_parent").flatMap(parseTokenList),
      parentBlocks: readRawBlocks(raw, "parent").map(parseParentBlock),
      allParents: readRawBlocks(raw, "all_parents").flatMap(parseTokenList),
      mutuallyExclusive: mutexRaw ? parseTokenList(mutexRaw) : [],
    };
  });
}

export type TraitEdgeKind = "any" | "all" | "and" | "mutex";

export interface TraitEdgeModel {
  id: string;
  sourceTraitId: string;
  targetTraitId: string;
  kind: TraitEdgeKind;

  numParentsNeeded?: number | null;
}

export function buildTraitTreeEdges(
  nodes: TraitTreeNodeModel[],
): TraitEdgeModel[] {
  const present = new Set(nodes.map((n) => n.traitId));
  const edges: TraitEdgeModel[] = [];
  const seen = new Set<string>();

  const push = (
    source: string,
    target: string,
    kind: TraitEdgeKind,
    numParentsNeeded?: number | null,
  ) => {
    if (!present.has(source) || !present.has(target) || source === target)
      return;
    const id = `${kind}:${source}->${target}`;
    if (seen.has(id)) return;
    seen.add(id);
    edges.push({
      id,
      sourceTraitId: source,
      targetTraitId: target,
      kind,
      numParentsNeeded,
    });
  };

  for (const node of nodes) {
    for (const p of node.anyParents) push(p, node.traitId, "any");
    for (const p of node.allParents) push(p, node.traitId, "all");
    for (const block of node.parentBlocks) {
      for (const p of block.traits)
        push(p, node.traitId, "and", block.numParentsNeeded);
    }
    for (const m of node.mutuallyExclusive) {
      const [a, b] = [node.traitId, m].sort();
      push(a, b, "mutex");
    }
  }
  return edges;
}

export function externalParentsOf(
  node: TraitTreeNodeModel,
  nodes: TraitTreeNodeModel[],
): string[] {
  const present = new Set(nodes.map((n) => n.traitId));
  const all = [
    ...node.anyParents,
    ...node.allParents,
    ...node.parentBlocks.flatMap((b) => b.traits),
  ];
  return Array.from(new Set(all.filter((t) => !present.has(t))));
}

export function traitsToRFNodes(
  nodes: TraitTreeNodeModel[],
  selectedDbId: number | null,
): Node<TraitGraphNodeData>[] {
  return nodes.map((n) => {
    const pos = gridToPixel(n.column, n.guiRow);
    return {
      id: String(n.dbId),
      type: "traitNode",
      position: pos,
      selected: n.dbId === selectedDbId,
      data: {
        traitId: n.traitId,
        label: n.label,
        dbId: n.dbId,
        guiRow: n.guiRow,
        column: n.column,
        columnIsAuto: n.columnIsAuto,
        traitType: n.traitType,
        externalParents: externalParentsOf(n, nodes),
        hasAdvancedParents:
          n.parentBlocks.length > 0 || n.allParents.length > 0,
      },
    };
  });
}

export function traitEdgesToRFEdges(
  edges: TraitEdgeModel[],
  nodes: TraitTreeNodeModel[],
): Edge[] {
  const dbIdByTraitId = new Map(nodes.map((n) => [n.traitId, n.dbId]));
  return edges.flatMap<Edge>((e) => {
    const source = dbIdByTraitId.get(e.sourceTraitId);
    const target = dbIdByTraitId.get(e.targetTraitId);
    if (source === undefined || target === undefined) return [];
    if (e.kind === "mutex") {
      const mutex: Edge = {
        id: e.id,
        source: String(source),
        target: String(target),
        type: "traitMutexEdge",

        deletable: false,
      };
      return [mutex];
    }
    const data: TraitParentEdgeData = {
      kind: e.kind,
      numParentsNeeded: e.numParentsNeeded ?? null,

      readOnly: e.kind !== "any",
    };
    const parentEdge: Edge = {
      id: e.id,
      source: String(source),
      target: String(target),
      type: "traitParentEdge",
      deletable: e.kind === "any",
      data: data as unknown as Record<string, unknown>,
    };
    return [parentEdge];
  });
}
