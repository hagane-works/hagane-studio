import type { Node, Edge } from "@xyflow/react";
import type { FocusNode } from "@/types";
import { API_BASE } from "@/lib/api";
import { gfxUrl } from "@/lib/api/basemod";
import { resolveDisplayName } from "@/lib/i18n/localizedName";
import type { ContentLocale } from "@/lib/store/contentLocaleStore";

export const GRID_CELL_WIDTH = 130;
export const GRID_CELL_HEIGHT = 180;

export const GRID_OFFSET_X = 65;
export const GRID_OFFSET_Y = 40;

export const NODE_WIDTH = 110;
export const NODE_HEIGHT_NOMINAL = 120;

export const GRID_LINE_OFFSET_X =
  GRID_CELL_WIDTH / 2 - GRID_OFFSET_X + (GRID_CELL_WIDTH - NODE_WIDTH) / 2;
export const GRID_LINE_OFFSET_Y =
  GRID_CELL_HEIGHT / 2 -
  GRID_OFFSET_Y +
  (GRID_CELL_HEIGHT - NODE_HEIGHT_NOMINAL) / 2;

export function hoi4ToCellCenter(
  x: number,
  y: number,
): { x: number; y: number } {
  return {
    x: x * GRID_CELL_WIDTH + GRID_OFFSET_X + NODE_WIDTH / 2,
    y: y * GRID_CELL_HEIGHT + GRID_OFFSET_Y + NODE_HEIGHT_NOMINAL / 2,
  };
}

export function hoi4ToPixel(x: number, y: number): { x: number; y: number } {
  return {
    x: x * GRID_CELL_WIDTH + GRID_OFFSET_X,
    y: y * GRID_CELL_HEIGHT + GRID_OFFSET_Y,
  };
}

export function pixelToHoi4(px: number, py: number): { x: number; y: number } {
  return {
    x: Math.round((px - GRID_OFFSET_X) / GRID_CELL_WIDTH),
    y: Math.round((py - GRID_OFFSET_Y) / GRID_CELL_HEIGHT),
  };
}

export function absOccupiedCells(nodes: FocusNode[]): Set<string> {
  const out = new Set<string>();
  for (const n of nodes) {
    const p = resolveAbsolutePosition(n, nodes);
    out.add(`${p.x},${p.y}`);
  }
  return out;
}

export function findFreeGridCell(
  occupied: Set<string>,
  startX: number,
  startY: number,
  limit: number,
): { x: number; y: number } {
  let i = 0;
  while (i < limit * 5) {
    const x = startX + (i % 5);
    const y = startY + Math.floor(i / 5);
    if (!occupied.has(`${x},${y}`)) return { x, y };
    i += 1;
  }
  return { x: startX, y: startY };
}

export function findFreeMutexCell(
  occupied: Set<string>,
  parentX: number,
  parentY: number,
  limit: number,
): { x: number; y: number } {
  const leftCol = parentX - 1;
  const rightCol = parentX + 1;

  for (let dy = 0; dy < limit; dy++) {
    if (dy === 0) {
      if (!occupied.has(`${leftCol},${parentY}`))
        return { x: leftCol, y: parentY };
    }
    if (dy > 0) {
      if (!occupied.has(`${leftCol},${parentY - dy}`)) {
        return { x: leftCol, y: parentY - dy };
      }

      if (!occupied.has(`${leftCol},${parentY + dy}`)) {
        return { x: leftCol, y: parentY + dy };
      }
    }

    if (dy === 0) {
      if (!occupied.has(`${rightCol},${parentY}`))
        return { x: rightCol, y: parentY };
    }
    if (dy > 0) {
      if (!occupied.has(`${rightCol},${parentY - dy}`)) {
        return { x: rightCol, y: parentY - dy };
      }

      if (!occupied.has(`${rightCol},${parentY + dy}`)) {
        return { x: rightCol, y: parentY + dy };
      }
    }
  }

  return { x: leftCol, y: parentY };
}

export function findFreeMutexCellLeft(
  occupied: Set<string>,
  parentX: number,
  parentY: number,
  limit: number,
): { x: number; y: number } {
  const leftCol = parentX - 1;

  for (let dy = 0; dy < limit; dy++) {
    if (dy === 0) {
      if (!occupied.has(`${leftCol},${parentY}`))
        return { x: leftCol, y: parentY };
    }
    if (dy > 0) {
      if (!occupied.has(`${leftCol},${parentY - dy}`)) {
        return { x: leftCol, y: parentY - dy };
      }

      if (!occupied.has(`${leftCol},${parentY + dy}`)) {
        return { x: leftCol, y: parentY + dy };
      }
    }
  }

  return { x: leftCol, y: parentY };
}

export function findFreeMutexCellRight(
  occupied: Set<string>,
  parentX: number,
  parentY: number,
  limit: number,
): { x: number; y: number } {
  const rightCol = parentX + 1;

  for (let dy = 0; dy < limit; dy++) {
    if (dy === 0) {
      if (!occupied.has(`${rightCol},${parentY}`))
        return { x: rightCol, y: parentY };
    }
    if (dy > 0) {
      if (!occupied.has(`${rightCol},${parentY - dy}`)) {
        return { x: rightCol, y: parentY - dy };
      }

      if (!occupied.has(`${rightCol},${parentY + dy}`)) {
        return { x: rightCol, y: parentY + dy };
      }
    }
  }

  return { x: rightCol, y: parentY };
}

export function resolveAbsolutePosition(
  node: FocusNode,
  allNodes: FocusNode[],
  visited: Set<number> = new Set(),
): { x: number; y: number } {
  if (!node.relative_position_id) {
    return { x: node.x, y: node.y };
  }

  if (visited.has(node.id)) {
    return { x: node.x, y: node.y };
  }
  visited.add(node.id);

  const refNode = allNodes.find(
    (n) => n.focus_id === node.relative_position_id,
  );
  if (!refNode) {
    return { x: node.x, y: node.y };
  }

  const refPos = resolveAbsolutePosition(refNode, allNodes, visited);
  return {
    x: refPos.x + node.x,
    y: refPos.y + node.y,
  };
}

export function focusVisualSubtree(
  focusNodes: FocusNode[],
  rootDbId: number,
): number[] {
  const inBatch = new Set(focusNodes.map((n) => n.focus_id));
  const byFocusId = new Map(focusNodes.map((n) => [n.focus_id, n]));
  const anchorOf = (n: FocusNode): string | null => {
    const a = n.relative_position_id;
    return a && inBatch.has(a) && a !== n.focus_id ? a : null;
  };
  const parentsOf = new Map<string, string[]>();
  for (const n of focusNodes) {
    const ps: string[] = [];
    for (const grp of n.prerequisites ?? [])
      for (const r of grp) if (inBatch.has(r) && r !== n.focus_id) ps.push(r);
    const a = anchorOf(n);
    if (a && !ps.includes(a)) ps.push(a);
    parentsOf.set(n.focus_id, ps);
  }

  const depth = new Map<string, number>();
  const remaining = new Map<string, string[]>(parentsOf);
  let guard = 0;
  while (remaining.size > 0 && guard <= focusNodes.length + 2) {
    guard += 1;
    const ready: string[] = [];
    for (const [fid, ps] of remaining) {
      if (ps.every((p) => depth.has(p))) ready.push(fid);
    }
    if (ready.length === 0) {
      for (const fid of remaining.keys()) depth.set(fid, 0);
      break;
    }
    for (const fid of ready) {
      let d = 0;
      for (const p of parentsOf.get(fid)!)
        d = Math.max(d, (depth.get(p) ?? 0) + 1);
      depth.set(fid, d);
      remaining.delete(fid);
    }
  }

  const childrenOf = new Map<string, string[]>();
  for (const n of focusNodes) {
    let primary: string | null = null;
    let best = -1;
    for (const p of parentsOf.get(n.focus_id)!) {
      const d = depth.get(p) ?? -1;
      if (d > best) {
        best = d;
        primary = p;
      }
    }
    if (primary) {
      if (!childrenOf.has(primary)) childrenOf.set(primary, []);
      childrenOf.get(primary)!.push(n.focus_id);
    }
  }
  const root = focusNodes.find((n) => n.id === rootDbId);
  if (!root) return [];
  const out: number[] = [];
  const seen = new Set<string>([root.focus_id]);
  const stack = [root.focus_id];
  while (stack.length > 0) {
    const cur = stack.pop()!;
    for (const c of childrenOf.get(cur) ?? []) {
      if (seen.has(c)) continue;
      seen.add(c);
      out.push(byFocusId.get(c)!.id);
      stack.push(c);
    }
  }
  return out;
}

export interface FocusGraphNodeData {
  focusId: string;
  label: string;
  icon: string;
  iconUrl: string | null;

  baseModIconUrls: string[];

  vanillaIconUrl: string | null;
  cost: number;
  dbId: number;
  isRelative: boolean;

  hasAllowBranch: boolean;
  hasLoadFocusTreeRules: boolean;

  onAddChild?: () => void;

  onAddMutexChild?: (side: "left" | "right") => void;
  [key: string]: unknown;
}

export function buildFocusGraphNode(
  node: FocusNode,
  allNodes: FocusNode[],
  selectedNodeId: number | null,
  availableIcons: Set<string> | undefined,
  projectId: number | undefined,
  iconCacheBust: { gfxName?: string; version: number } | undefined,
  locale: ContentLocale,
  baseModDepKeys: string[],
  onCreateChildNode?: (parentNodeId: number) => void,
  onCreateMutexChildNode?: (
    parentNodeId: number,
    side: "left" | "right",
  ) => void,
): Node<FocusGraphNodeData> {
  const absPos = resolveAbsolutePosition(node, allNodes);
  const pixelPos = hoi4ToPixel(absPos.x, absPos.y);

  let iconUrl: string | null = null;
  if (
    projectId &&
    availableIcons &&
    node.icon &&
    availableIcons.has(node.icon)
  ) {
    const url = `${API_BASE}/go/projects/${projectId}/focus-icons/${encodeURIComponent(node.icon)}`;
    iconUrl =
      iconCacheBust &&
      (!iconCacheBust.gfxName || iconCacheBust.gfxName === node.icon)
        ? `${url}?t=${iconCacheBust.version}`
        : url;
  }
  const vanillaIconUrl: string | null =
    node.icon &&
    node.icon !== "GFX_goal_unknown" &&
    (node.icon.startsWith("GFX_focus_") || node.icon.startsWith("GFX_goal_"))
      ? `${API_BASE}/go/vanilla/focus-icon/${encodeURIComponent(node.icon)}`
      : null;

  const baseModIconUrls: string[] =
    projectId &&
    node.icon &&
    node.icon !== "GFX_goal_unknown" &&
    baseModDepKeys.length
      ? baseModDepKeys.map((dk) => gfxUrl(projectId, dk, node.icon))
      : [];

  const allowAst = node.allow_branch_ast as Record<string, unknown> | undefined;
  const loadRules = node.load_focus_tree_rules as unknown[] | undefined;
  const hasAllowBranch = !!allowAst && Object.keys(allowAst).length > 0;
  const hasLoadFocusTreeRules =
    Array.isArray(loadRules) && loadRules.length > 0;

  return {
    id: String(node.id),
    type: "focusNode",
    position: { x: pixelPos.x, y: pixelPos.y },
    selected: node.id === selectedNodeId,
    data: {
      focusId: node.focus_id,

      label: resolveDisplayName(
        locale,
        node.display_name_locs,
        node.name_en,
        node.name_zh,
        node.name || node.focus_id || "New Focus",
      ),
      icon: node.icon,
      iconUrl,
      baseModIconUrls,
      vanillaIconUrl,
      cost: node.cost,
      dbId: node.id,
      isRelative: !!node.relative_position_id,
      hasAllowBranch,
      hasLoadFocusTreeRules,
      onAddChild: onCreateChildNode
        ? () => onCreateChildNode(node.id)
        : undefined,
      onAddMutexChild: onCreateMutexChildNode
        ? (side: "left" | "right") => onCreateMutexChildNode(node.id, side)
        : undefined,
    },
  };
}

export function focusNodesToRFNodes(
  nodes: FocusNode[],
  selectedNodeId: number | null,
  availableIcons?: Set<string>,
  projectId?: number,
  iconCacheBust?: { gfxName?: string; version: number },

  locale: ContentLocale = "en",

  baseModDepKeys: string[] = [],

  onCreateChildNode?: (parentNodeId: number) => void,

  onCreateMutexChildNode?: (
    parentNodeId: number,
    side: "left" | "right",
  ) => void,
): Node<FocusGraphNodeData>[] {
  return nodes.map((node) =>
    buildFocusGraphNode(
      node,
      nodes,
      selectedNodeId,
      availableIcons,
      projectId,
      iconCacheBust,
      locale,
      baseModDepKeys,
      onCreateChildNode,
      onCreateMutexChildNode,
    ),
  );
}

export function focusNodesToRFEdges(nodes: FocusNode[]): Edge[] {
  const edges: Edge[] = [];
  const focusIdToDbId = new Map<string, number>();

  const seenMutexPairs = new Set<string>();

  for (const node of nodes) {
    focusIdToDbId.set(node.focus_id, node.id);
  }

  const mutexInBatch = new Set(nodes.map((n) => n.focus_id));
  const mutexAdj = buildMutexAdjacency(nodes, mutexInBatch);

  const suppressedMutexPairs = new Set<string>();
  for (const comp of mutexComponents(mutexAdj)) {
    if (comp.length < 3) continue;
    const isClique = comp.every((a) =>
      comp.every((b) => a === b || (mutexAdj.get(a)?.has(b) ?? false)),
    );
    if (!isClique) continue;

    const byPos = [...comp].sort((a, b) => {
      const pa = resolveAbsolutePosition(
        nodes.find((n) => n.focus_id === a)!,
        nodes,
      );
      const pb = resolveAbsolutePosition(
        nodes.find((n) => n.focus_id === b)!,
        nodes,
      );
      return pa.y - pb.y || pa.x - pb.x || a.localeCompare(b);
    });
    const chain = new Set<string>();
    for (let i = 0; i + 1 < byPos.length; i += 1) {
      const u = focusIdToDbId.get(byPos[i])!;
      const v = focusIdToDbId.get(byPos[i + 1])!;
      chain.add(`${Math.min(u, v)}-${Math.max(u, v)}`);
    }
    for (const a of comp) {
      for (const b of comp) {
        if (a >= b) continue;
        const u = focusIdToDbId.get(a)!;
        const v = focusIdToDbId.get(b)!;
        const key = `${Math.min(u, v)}-${Math.max(u, v)}`;
        if (!chain.has(key)) suppressedMutexPairs.add(key);
      }
    }
  }

  for (const node of nodes) {
    const prereqs = node.prerequisites || [];
    prereqs.forEach((group, groupIndex) => {
      const isOrGroup = group.length > 1;

      group.forEach((prereqFocusId) => {
        const sourceDbId = focusIdToDbId.get(prereqFocusId);
        if (sourceDbId === undefined) return;

        const edgeId = `prereq-${sourceDbId}-${node.id}-g${groupIndex}`;
        edges.push({
          id: edgeId,
          source: String(sourceDbId),
          target: String(node.id),
          type: "prereqEdge",
          data: {
            groupIndex,
            isOrGroup,
            prereqFocusId,
            targetNodeId: node.id,
          },
          style: { stroke: "var(--color-muted-foreground)", strokeWidth: 2 },
          animated: false,
        });
      });
    });

    const mutexList = node.mutually_exclusive || [];
    mutexList.forEach((otherFocusId) => {
      const otherDbId = focusIdToDbId.get(otherFocusId);
      if (otherDbId === undefined) return;

      if (otherDbId === node.id) return;

      const lowId = Math.min(node.id, otherDbId);
      const highId = Math.max(node.id, otherDbId);
      const pairKey = `${lowId}-${highId}`;
      if (suppressedMutexPairs.has(pairKey)) return;
      if (!seenMutexPairs.has(pairKey)) {
        seenMutexPairs.add(pairKey);
        const otherNode = nodes.find((n) => n.id === otherDbId);
        if (!otherNode) return;

        const posSelf = resolveAbsolutePosition(node, nodes);
        const posOther = resolveAbsolutePosition(otherNode, nodes);
        const selfIsLeft = posSelf.x <= posOther.x;
        const rightNode = selfIsLeft ? otherNode : node;
        const leftNode = selfIsLeft ? node : otherNode;

        const lowNode = node.id === lowId ? node : otherNode;
        const highNode = node.id === lowId ? otherNode : node;

        edges.push({
          id: `mutex-${lowNode.id}-${highNode.id}`,
          source: String(rightNode.id),
          target: String(leftNode.id),
          sourceHandle: "mutex-left",
          targetHandle: "mutex-right",
          type: "mutexEdge",
          data: {
            sourceFocusId: lowNode.focus_id,
            targetFocusId: highNode.focus_id,
          },
          style: {
            stroke: "var(--color-destructive)",
            strokeWidth: 2,
            strokeDasharray: "6 3",
          },
          animated: false,
        });
      }
    });
  }

  return edges;
}

export function parseEdgeId(edgeId: string): {
  type: "prereq" | "mutex";
  sourceDbId: number;
  targetDbId: number;
  groupIndex?: number;
} | null {
  const prereqMatch = edgeId.match(/^prereq-(\d+)-(\d+)-g(\d+)$/);
  if (prereqMatch) {
    return {
      type: "prereq",
      sourceDbId: Number(prereqMatch[1]),
      targetDbId: Number(prereqMatch[2]),
      groupIndex: Number(prereqMatch[3]),
    };
  }

  const mutexMatch = edgeId.match(/^mutex-(\d+)-(\d+)$/);
  if (mutexMatch) {
    return {
      type: "mutex",
      sourceDbId: Number(mutexMatch[1]),
      targetDbId: Number(mutexMatch[2]),
    };
  }

  return null;
}

const LAYOUT_HORIZ_SPACING = 4;

function buildMutexAdjacency(
  focusNodes: FocusNode[],
  inBatch: Set<string>,
): Map<string, Set<string>> {
  const adj = new Map<string, Set<string>>();
  const link = (a: string, b: string) => {
    if (!adj.has(a)) adj.set(a, new Set());
    adj.get(a)!.add(b);
  };
  for (const n of focusNodes) {
    for (const other of n.mutually_exclusive ?? []) {
      if (!inBatch.has(other) || other === n.focus_id) continue;
      link(n.focus_id, other);
      link(other, n.focus_id);
    }
  }
  return adj;
}

function mutexComponents(adj: Map<string, Set<string>>): string[][] {
  const out: string[][] = [];
  const visited = new Set<string>();
  for (const fid of [...adj.keys()].sort()) {
    if (visited.has(fid)) continue;
    const comp: string[] = [];
    const stack = [fid];
    visited.add(fid);
    while (stack.length > 0) {
      const cur = stack.pop()!;
      comp.push(cur);
      for (const nb of adj.get(cur) ?? []) {
        if (!visited.has(nb)) {
          visited.add(nb);
          stack.push(nb);
        }
      }
    }
    comp.sort();
    out.push(comp);
  }
  return out;
}

export interface LayoutRepairChange {
  nodeId: number;
  focusId: string;

  reason:
    | "overlap"
    | "orphan"
    | "mutex-row"
    | "broken-anchor"
    | "child-above-parent"
    | "tree-shift"
    | "tidy";
  from: { x: number; y: number };
  to: { x: number; y: number };
}

export interface LayoutUnaligned {
  focusIds: string[];

  reason: "mutex-conflicts-prereq" | "prereq-cycle";
}

export interface LayoutRepairResult {
  positions: Map<number, { x: number; y: number }>;
  changes: LayoutRepairChange[];

  needsFullLayout: boolean;

  unaligned: LayoutUnaligned[];
}

export function logicalComponents(focusNodes: FocusNode[]): string[][] {
  const inBatch = new Set(focusNodes.map((n) => n.focus_id));
  const parent = new Map<string, string>();
  const find = (a: string): string => {
    let r = a;
    while (parent.get(r) !== r) r = parent.get(r)!;
    while (parent.get(a) !== r) {
      const nxt = parent.get(a)!;
      parent.set(a, r);
      a = nxt;
    }
    return r;
  };
  const union = (a: string, b: string) => {
    const ra = find(a);
    const rb = find(b);
    if (ra !== rb) parent.set(ra, rb);
  };
  for (const n of focusNodes) parent.set(n.focus_id, n.focus_id);
  for (const n of focusNodes) {
    for (const grp of n.prerequisites ?? []) {
      for (const ref of grp) {
        if (inBatch.has(ref) && ref !== n.focus_id) union(n.focus_id, ref);
      }
    }
    for (const m of n.mutually_exclusive ?? []) {
      if (inBatch.has(m) && m !== n.focus_id) union(n.focus_id, m);
    }
  }
  const groups = new Map<string, string[]>();
  for (const n of focusNodes) {
    const r = find(n.focus_id);
    if (!groups.has(r)) groups.set(r, []);
    groups.get(r)!.push(n.focus_id);
  }

  return [...groups.values()].sort(
    (a, b) => b.length - a.length || a[0].localeCompare(b[0]),
  );
}

function compactHorizontallyInPlace(
  focusNodes: FocusNode[],
  abs: Map<string, { x: number; y: number }>,
  prereqsOf: (n: FocusNode) => string[],
  subtreeOf: (fid: string) => string[],
): Set<string> {
  const key = (p: { x: number; y: number }) => `${p.x},${p.y}`;
  const occupied = new Map<string, string>();
  for (const [f, p] of abs) occupied.set(key(p), f);
  const moved = new Set<string>();

  const order = [...focusNodes].sort(
    (a, b) =>
      abs.get(a.focus_id)!.y - abs.get(b.focus_id)!.y ||
      abs.get(a.focus_id)!.x - abs.get(b.focus_id)!.x,
  );
  for (let pass = 0; pass < 8; pass += 1) {
    let changed = false;
    for (const n of order) {
      const ps = prereqsOf(n);
      if (ps.length === 0) continue;
      const xs = ps.map((p) => abs.get(p)!.x).sort((a, b) => a - b);
      const want = xs[Math.floor(xs.length / 2)];
      const group = [n.focus_id, ...subtreeOf(n.focus_id)];
      const inGroup = new Set(group);
      while (abs.get(n.focus_id)!.x !== want) {
        const step = want > abs.get(n.focus_id)!.x ? 1 : -1;
        const blocked = group.some((g) => {
          const p = abs.get(g)!;
          const owner = occupied.get(key({ x: p.x + step, y: p.y }));
          return owner !== undefined && !inGroup.has(owner);
        });
        if (blocked) break;
        for (const g of group) occupied.delete(key(abs.get(g)!));
        for (const g of group) {
          const p = abs.get(g)!;
          const np = { x: p.x + step, y: p.y };
          abs.set(g, np);
          occupied.set(key(np), g);
          moved.add(g);
        }
        changed = true;
      }
    }
    if (!changed) break;
  }
  return moved;
}

function detectMutexPrereqConflicts(
  focusNodes: FocusNode[],
): LayoutUnaligned[] {
  const inBatch = new Set(focusNodes.map((n) => n.focus_id));
  const prereqAdj = new Map<string, string[]>();
  for (const n of focusNodes) {
    const ps: string[] = [];
    for (const grp of n.prerequisites ?? [])
      for (const ref of grp)
        if (inBatch.has(ref) && ref !== n.focus_id) ps.push(ref);
    prereqAdj.set(n.focus_id, ps);
  }
  const dependsOn = (from: string, to: string): boolean => {
    const stack = [from];
    const seen = new Set<string>([from]);
    while (stack.length > 0) {
      const cur = stack.pop()!;
      if (cur === to && cur !== from) return true;
      for (const p of prereqAdj.get(cur) ?? [])
        if (!seen.has(p)) {
          seen.add(p);
          stack.push(p);
        }
    }
    return false;
  };
  const out: LayoutUnaligned[] = [];

  for (const comp of mutexComponents(
    buildMutexAdjacency(focusNodes, inBatch),
  )) {
    if (comp.length < 2) continue;
    const conflict = comp.some((a) =>
      comp.some((b) => a !== b && dependsOn(a, b)),
    );
    if (conflict)
      out.push({ focusIds: comp, reason: "mutex-conflicts-prereq" });
  }

  const state = new Map<string, 0 | 1 | 2>();
  const cyc = new Set<string>();
  const visit = (start: string) => {
    const stack: Array<{ u: string; i: number }> = [{ u: start, i: 0 }];
    state.set(start, 1);
    while (stack.length > 0) {
      const top = stack[stack.length - 1];
      const ps = prereqAdj.get(top.u) ?? [];
      if (top.i < ps.length) {
        const v = ps[top.i];
        top.i += 1;
        const st = state.get(v) ?? 0;
        if (st === 1) {
          cyc.add(v);
          cyc.add(top.u);
        } else if (st === 0) {
          state.set(v, 1);
          stack.push({ u: v, i: 0 });
        }
      } else {
        state.set(top.u, 2);
        stack.pop();
      }
    }
  };
  for (const n of focusNodes)
    if ((state.get(n.focus_id) ?? 0) === 0) visit(n.focus_id);
  if (cyc.size > 0)
    out.push({ focusIds: [...cyc].sort(), reason: "prereq-cycle" });
  return out;
}

const TIDY_HORIZ_SPACING = 2;

export function repairFocusLayout(focusNodes: FocusNode[]): LayoutRepairResult {
  const positions = new Map<number, { x: number; y: number }>();
  const changes: LayoutRepairChange[] = [];
  if (focusNodes.length === 0) {
    return { positions, changes, needsFullLayout: false, unaligned: [] };
  }

  const laid = bfsLayoutFocusNodes(focusNodes, TIDY_HORIZ_SPACING);
  for (const n of focusNodes) {
    const to = laid.get(n.id);
    if (!to) continue;
    if (to.x !== n.x || to.y !== n.y) {
      positions.set(n.id, to);
      changes.push({
        nodeId: n.id,
        focusId: n.focus_id,
        reason: "tidy",
        from: { x: n.x, y: n.y },
        to,
      });
    }
  }

  return {
    positions,
    changes,
    needsFullLayout: false,
    unaligned: detectMutexPrereqConflicts(focusNodes),
  };
}

export interface LayoutCompactResult {
  positions: Map<number, { x: number; y: number }>;
  changes: number;
}

export function compactFocusLayoutHorizontally(
  focusNodes: FocusNode[],
): LayoutCompactResult {
  const positions = new Map<number, { x: number; y: number }>();
  if (focusNodes.length === 0) return { positions, changes: 0 };

  const inBatch = new Set(focusNodes.map((n) => n.focus_id));
  const anchorOf = (n: FocusNode): string | null => {
    const a = n.relative_position_id;
    return a && inBatch.has(a) && a !== n.focus_id ? a : null;
  };
  const prereqsOf = (n: FocusNode): string[] => {
    const out: string[] = [];
    for (const grp of n.prerequisites ?? []) {
      for (const ref of grp)
        if (inBatch.has(ref) && ref !== n.focus_id) out.push(ref);
    }
    return out;
  };

  const abs = new Map<string, { x: number; y: number }>();
  for (const n of focusNodes)
    abs.set(n.focus_id, resolveAbsolutePosition(n, focusNodes));

  const anchorKids = new Map<string, string[]>();
  for (const n of focusNodes) {
    const a = anchorOf(n);
    if (!a) continue;
    if (!anchorKids.has(a)) anchorKids.set(a, []);
    anchorKids.get(a)!.push(n.focus_id);
  }
  const subtreeOf = (fid: string): string[] => {
    const out: string[] = [];
    const seen = new Set<string>([fid]);
    const stack = [fid];
    while (stack.length > 0) {
      const cur = stack.pop()!;
      for (const k of anchorKids.get(cur) ?? []) {
        if (seen.has(k)) continue;
        seen.add(k);
        out.push(k);
        stack.push(k);
      }
    }
    return out;
  };

  const moved = compactHorizontallyInPlace(
    focusNodes,
    abs,
    prereqsOf,
    subtreeOf,
  );

  for (const n of focusNodes) {
    if (!moved.has(n.focus_id)) continue;
    const to = abs.get(n.focus_id)!;
    const anchorFid = anchorOf(n);
    if (anchorFid) {
      const a = abs.get(anchorFid)!;
      positions.set(n.id, { x: to.x - a.x, y: to.y - a.y });
    } else {
      positions.set(n.id, to);
    }
  }
  return { positions, changes: positions.size };
}

function tidyTreeColumns(
  roots: string[],
  childrenOf: Map<string, string[]>,
  distance: number,
): Map<string, number> {
  const prelim = new Map<string, number>();
  const mod = new Map<string, number>();
  const thread = new Map<string, string | null>();
  const ancestorOf = new Map<string, string>();
  const change = new Map<string, number>();
  const shiftV = new Map<string, number>();
  const num = new Map<string, number>();
  const parentOf = new Map<string, string>();
  for (const [p, kids] of childrenOf) for (const c of kids) parentOf.set(c, p);

  const ch = (v: string) => childrenOf.get(v) ?? [];
  const isLeaf = (v: string) => ch(v).length === 0;
  const leftSibling = (v: string): string | null => {
    const p = parentOf.get(v);
    if (!p) return null;
    const kids = ch(p);
    const i = kids.indexOf(v);
    return i > 0 ? kids[i - 1] : null;
  };
  const leftmostSibling = (v: string): string | null => {
    const p = parentOf.get(v);
    if (!p) return null;
    const kids = ch(p);
    return kids[0] !== v ? kids[0] : null;
  };
  const nextLeft = (v: string): string | null =>
    isLeaf(v) ? (thread.get(v) ?? null) : ch(v)[0];
  const nextRight = (v: string): string | null => {
    if (!isLeaf(v)) {
      const k = ch(v);
      return k[k.length - 1];
    }
    return thread.get(v) ?? null;
  };

  const all = new Set<string>();
  const collect = (v: string) => {
    all.add(v);
    for (const c of ch(v)) collect(c);
  };
  for (const r of roots) collect(r);
  for (const v of all) {
    prelim.set(v, 0);
    mod.set(v, 0);
    thread.set(v, null);
    ancestorOf.set(v, v);
    change.set(v, 0);
    shiftV.set(v, 0);
  }

  const moveSubtree = (wMinus: string, wPlus: string, sh: number) => {
    const subtrees = num.get(wPlus)! - num.get(wMinus)!;
    change.set(wPlus, change.get(wPlus)! - sh / subtrees);
    shiftV.set(wPlus, shiftV.get(wPlus)! + sh);
    change.set(wMinus, change.get(wMinus)! + sh / subtrees);
    prelim.set(wPlus, prelim.get(wPlus)! + sh);
    mod.set(wPlus, mod.get(wPlus)! + sh);
  };
  const anc = (vim: string, v: string, defaultA: string): string => {
    const a = ancestorOf.get(vim)!;
    return parentOf.get(a) === parentOf.get(v) ? a : defaultA;
  };
  const executeShifts = (v: string) => {
    let sh = 0,
      ch2 = 0;
    const kids = ch(v);
    for (let i = kids.length - 1; i >= 0; i -= 1) {
      const w = kids[i];
      prelim.set(w, prelim.get(w)! + sh);
      mod.set(w, mod.get(w)! + sh);
      ch2 += change.get(w)!;
      sh += shiftV.get(w)! + ch2;
    }
  };
  const apportion = (v: string, defaultA: string): string => {
    const w = leftSibling(v);
    if (!w) return defaultA;
    let vip = v,
      vop = v,
      vim = w,
      vom = leftmostSibling(v)!;
    let sip = mod.get(vip)!,
      sop = mod.get(vop)!,
      sim = mod.get(vim)!,
      som = mod.get(vom)!;
    while (nextRight(vim) && nextLeft(vip)) {
      vim = nextRight(vim)!;
      vip = nextLeft(vip)!;
      vom = nextLeft(vom)!;
      vop = nextRight(vop)!;
      ancestorOf.set(vop, v);
      const sh = prelim.get(vim)! + sim - (prelim.get(vip)! + sip) + distance;
      if (sh > 0) {
        moveSubtree(anc(vim, v, defaultA), v, sh);
        sip += sh;
        sop += sh;
      }
      sim += mod.get(vim)!;
      sip += mod.get(vip)!;
      som += mod.get(vom)!;
      sop += mod.get(vop)!;
    }
    if (nextRight(vim) && !nextRight(vop)) {
      thread.set(vop, nextRight(vim));
      mod.set(vop, mod.get(vop)! + sim - sop);
    }
    if (nextLeft(vip) && !nextLeft(vom)) {
      thread.set(vom, nextLeft(vip));
      mod.set(vom, mod.get(vom)! + sip - som);
      return v;
    }
    return defaultA;
  };
  const firstWalk = (v: string) => {
    const kids = ch(v);
    if (kids.length === 0) {
      const w = leftSibling(v);
      prelim.set(v, w ? prelim.get(w)! + distance : 0);
    } else {
      let defaultA = kids[0];
      kids.forEach((w, i) => num.set(w, i + 1));
      for (const w of kids) {
        firstWalk(w);
        defaultA = apportion(w, defaultA);
      }
      executeShifts(v);
      const midpoint =
        (prelim.get(kids[0])! + prelim.get(kids[kids.length - 1])!) / 2;
      const w = leftSibling(v);
      if (w) {
        prelim.set(v, prelim.get(w)! + distance);
        mod.set(v, prelim.get(v)! - midpoint);
      } else {
        prelim.set(v, midpoint);
      }
    }
  };
  const col = new Map<string, number>();
  const secondWalk = (v: string, m: number) => {
    col.set(v, prelim.get(v)! + m);
    for (const w of ch(v)) secondWalk(w, m + mod.get(v)!);
  };

  let offset = 0;
  for (const r of roots) {
    firstWalk(r);
    secondWalk(r, 0);
    const sub: string[] = [];
    const gather = (v: string) => {
      sub.push(v);
      for (const c of ch(v)) gather(c);
    };
    gather(r);
    let minC = Infinity,
      maxC = -Infinity;
    for (const v of sub) {
      const c = col.get(v)!;
      if (c < minC) minC = c;
      if (c > maxC) maxC = c;
    }
    const delta = offset - minC;
    for (const v of sub) col.set(v, col.get(v)! + delta);
    offset = maxC + delta + distance;
  }
  return col;
}

export function bfsLayoutFocusNodes(
  focusNodes: FocusNode[],
  horizSpacing: number = LAYOUT_HORIZ_SPACING,
): Map<number, { x: number; y: number }> {
  const out = new Map<number, { x: number; y: number }>();
  if (focusNodes.length === 0) return out;

  const order = new Map<string, number>(
    focusNodes.map((n, i) => [n.focus_id, i]),
  );
  const inBatch = new Set<string>(focusNodes.map((n) => n.focus_id));

  const anchorOf = (n: FocusNode): string | null => {
    const a = n.relative_position_id;
    return a && inBatch.has(a) && a !== n.focus_id ? a : null;
  };

  const parentsOf = new Map<string, Set<string>>();
  for (const n of focusNodes) {
    const parents = new Set<string>();
    for (const grp of n.prerequisites ?? []) {
      for (const ref of grp) {
        if (inBatch.has(ref) && ref !== n.focus_id) {
          parents.add(ref);
        }
      }
    }
    const anchor = anchorOf(n);
    if (anchor) parents.add(anchor);
    parentsOf.set(n.focus_id, parents);
  }

  const depth = new Map<string, number>();
  const remaining = new Map(parentsOf);
  let layerIdx = 0;
  while (remaining.size > 0) {
    const ready: string[] = [];
    for (const [fid, parents] of remaining) {
      let allReady = true;
      for (const p of parents) {
        if (!depth.has(p)) {
          allReady = false;
          break;
        }
      }
      if (allReady) ready.push(fid);
    }
    if (ready.length === 0) {
      for (const fid of remaining.keys()) depth.set(fid, layerIdx);
      break;
    }

    ready.sort();
    for (const fid of ready) {
      depth.set(fid, layerIdx);
      remaining.delete(fid);
    }
    layerIdx += 1;
  }

  const mutexAdj = buildMutexAdjacency(focusNodes, inBatch);

  const dependsOn = (from: string, to: string): boolean => {
    const stack = [from];
    const seen = new Set<string>([from]);
    while (stack.length > 0) {
      const cur = stack.pop()!;
      if (cur === to && cur !== from) return true;
      for (const p of parentsOf.get(cur) ?? []) {
        if (!seen.has(p)) {
          seen.add(p);
          stack.push(p);
        }
      }
    }
    return false;
  };

  const alignable: string[][] = [];
  for (const comp of mutexComponents(mutexAdj)) {
    const selfDependent = comp.some((x) =>
      comp.some((y) => x !== y && dependsOn(x, y)),
    );
    if (!selfDependent) alignable.push(comp);
  }

  const depthBeforeAlign = new Map(depth);
  let converged = false;
  for (let pass = 0; pass < focusNodes.length + 1; pass += 1) {
    let changed = false;

    for (const [fid, parents] of parentsOf) {
      let lowestParent = -1;
      for (const p of parents)
        lowestParent = Math.max(lowestParent, depth.get(p) ?? 0);
      if (lowestParent >= 0 && (depth.get(fid) ?? 0) < lowestParent + 1) {
        depth.set(fid, lowestParent + 1);
        changed = true;
      }
    }

    for (const comp of alignable) {
      const target = Math.max(...comp.map((m) => depth.get(m) ?? 0));
      for (const m of comp) {
        if ((depth.get(m) ?? 0) !== target) {
          depth.set(m, target);
          changed = true;
        }
      }
    }
    if (!changed) {
      converged = true;
      break;
    }
  }
  if (!converged) {
    depth.clear();
    for (const [fid, d] of depthBeforeAlign) depth.set(fid, d);
  }

  const childrenOf = new Map<string, string[]>();
  const roots: string[] = [];
  for (const n of focusNodes) {
    if (!depth.has(n.focus_id)) continue;
    let primary: string | null = null;
    let bestDepth = -1;
    for (const p of parentsOf.get(n.focus_id) ?? []) {
      const d = depth.get(p);
      if (d === undefined) continue;
      if (
        d > bestDepth ||
        (d === bestDepth && primary !== null && p < primary)
      ) {
        bestDepth = d;
        primary = p;
      }
    }
    if (primary === null) {
      roots.push(n.focus_id);
    } else {
      if (!childrenOf.has(primary)) childrenOf.set(primary, []);
      childrenOf.get(primary)!.push(n.focus_id);
    }
  }
  const byInputOrder = (a: string, b: string) =>
    (order.get(a) ?? 0) - (order.get(b) ?? 0);
  roots.sort(byInputOrder);
  for (const kids of childrenOf.values()) kids.sort(byInputOrder);

  const col = tidyTreeColumns(roots, childrenOf, horizSpacing);

  const emitted = focusNodes.filter(
    (n) => depth.has(n.focus_id) && !n.relative_position_id,
  );
  const byDepth = new Map<number, FocusNode[]>();
  for (const n of emitted) {
    const d = depth.get(n.focus_id)!;
    if (!byDepth.has(d)) byDepth.set(d, []);
    byDepth.get(d)!.push(n);
  }

  const finalCol = new Map<string, number>();
  for (const [, rowNodes] of byDepth) {
    rowNodes.sort((a, b) => {
      const ca = col.get(a.focus_id) ?? 0;
      const cb = col.get(b.focus_id) ?? 0;
      return ca === cb ? byInputOrder(a.focus_id, b.focus_id) : ca - cb;
    });
    let cursor = -Infinity;
    for (const n of rowNodes) {
      const wantGrid = Math.round(col.get(n.focus_id) ?? 0);
      const got =
        cursor === -Infinity
          ? wantGrid
          : Math.max(wantGrid, cursor + horizSpacing);
      finalCol.set(n.focus_id, got);
      cursor = got;
    }
  }

  let minCol = Infinity;
  for (const c of finalCol.values()) if (c < minCol) minCol = c;
  if (!Number.isFinite(minCol)) minCol = 0;

  const absCell = new Map<string, { x: number; y: number }>();

  const taken = new Set<string>();
  const claim = (fid: string, x: number, y: number) => {
    absCell.set(fid, { x, y });
    taken.add(`${x},${y}`);
  };

  const rowClear = (x: number, y: number): boolean => {
    for (let d = 1; d < horizSpacing; d += 1) {
      if (taken.has(`${x - d},${y}`) || taken.has(`${x + d},${y}`))
        return false;
    }
    return !taken.has(`${x},${y}`);
  };

  for (const n of emitted) {
    const x = finalCol.get(n.focus_id)! - minCol;
    const y = depth.get(n.focus_id)!;
    claim(n.focus_id, x, y);
  }

  const anchored = focusNodes
    .filter((n) => depth.has(n.focus_id) && anchorOf(n))
    .sort(
      (a, b) =>
        (depth.get(a.focus_id) ?? 0) - (depth.get(b.focus_id) ?? 0) ||
        a.focus_id.localeCompare(b.focus_id),
    );
  for (const n of anchored) {
    const anchor = absCell.get(anchorOf(n)!);
    if (!anchor) continue;
    const y = depth.get(n.focus_id) ?? anchor.y + 1;
    let x = anchor.x;
    let step = 0;
    while (!rowClear(x, y)) {
      step += 1;
      x =
        anchor.x +
        (step % 2 === 1 ? 1 : -1) * Math.ceil(step / 2) * horizSpacing;
    }
    claim(n.focus_id, x, y);
  }

  for (const n of focusNodes) {
    if (absCell.has(n.focus_id)) continue;
    const y = depth.get(n.focus_id) ?? 0;
    let x = 0;
    while (!rowClear(x, y)) x += horizSpacing;
    claim(n.focus_id, x, y);
  }

  for (const n of focusNodes) {
    const cell = absCell.get(n.focus_id);
    if (!cell) continue;
    const anchorFid = anchorOf(n);
    const a = anchorFid ? absCell.get(anchorFid) : undefined;
    out.set(n.id, a ? { x: cell.x - a.x, y: cell.y - a.y } : cell);
  }

  return out;
}
