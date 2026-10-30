import type { GuiNode } from "@/lib/guiRenderApi";
import { anchorCell } from "../gui-renderer/GuiCanvas";

type SizeValue = number | string | undefined;

function idOf(node: GuiNode): string {
  return (node as GuiNode & { _id?: string })._id ?? "";
}

function resolvePct(value: SizeValue, base: number): number {
  if (typeof value === "string" && value.trim().endsWith("%")) {
    const n = parseFloat(value);
    return Math.round((base * (Number.isNaN(n) ? 100 : n)) / 100);
  }
  const n = Number(value);
  return Number.isNaN(n) ? 0 : Math.round(n);
}

function resolveSize(value: SizeValue, base: number): number {
  const n = resolvePct(value, base);
  return n < 0 ? Math.max(0, base + n) : n;
}

function isNegative(value: SizeValue): boolean {
  return typeof value === "number"
    ? value < 0
    : typeof value === "string" && /^\s*-\d/.test(value);
}

function isFullPercent(value: SizeValue): boolean {
  return (
    typeof value === "string" &&
    value.trim().endsWith("%") &&
    parseFloat(value) === 100
  );
}

function sizeKeys(
  size: NonNullable<GuiNode["size"]>,
): ["width" | "x", "height" | "y"] {
  return [
    size.width !== undefined ? "width" : "x",
    size.height !== undefined ? "height" : "y",
  ];
}

export function pinFullSizes(
  root: GuiNode,
  baseW: number,
  baseH: number,
  { negative = false }: { negative?: boolean } = {},
): GuiNode {
  const pin = (node: GuiNode, pw: number, ph: number): GuiNode => {
    let next = node;
    let boxW = pw;
    let boxH = ph;
    if (node.size) {
      const [wKey, hKey] = sizeKeys(node.size);
      const wValue = node.size[wKey];
      const hValue = node.size[hKey];
      const [col, row] = anchorCell(node.orientation);
      let w = resolveSize(wValue, pw);
      let h = resolveSize(hValue, ph);
      const size = { ...node.size };
      let changed = false;
      if (isFullPercent(wValue) && col === 0) {
        w = Math.max(0, pw - resolvePct(node.position?.x, pw));
        size[wKey] = w;
        changed = true;
      }
      if (isFullPercent(hValue) && row === 0) {
        h = Math.max(0, ph - resolvePct(node.position?.y, ph));
        size[hKey] = h;
        changed = true;
      }
      if (negative && isNegative(wValue) && col === 0) {
        w = Math.max(0, pw - resolvePct(node.position?.x, pw) + Number(wValue));
        size[wKey] = w;
        changed = true;
      }
      if (negative && isNegative(hValue) && row === 0) {
        h = Math.max(0, ph - resolvePct(node.position?.y, ph) + Number(hValue));
        size[hKey] = h;
        changed = true;
      }
      if (changed) next = { ...node, size };

      boxW = w || pw;
      boxH = h || ph;
    } else if (node.type.endsWith("windowtype")) {
      boxW = 0;
      boxH = 0;
    }
    const children = node.children ?? [];
    const pinned = children.map((child) => pin(child, boxW, boxH));
    if (pinned.some((child, i) => child !== children[i]))
      next = { ...next, children: pinned };
    return next;
  };
  return pin(root, baseW, baseH);
}

export function patchTree(
  root: GuiNode,
  patches: Readonly<Record<string, Partial<GuiNode>>>,
): GuiNode {
  if (!Object.keys(patches).length) return root;
  const walk = (node: GuiNode): GuiNode => {
    const own = patches[idOf(node)];
    const children = node.children ?? [];
    const next = children.map(walk);
    const childrenChanged = next.some((child, i) => child !== children[i]);
    if (!own && !childrenChanged) return node;
    return { ...node, ...own, children: childrenChanged ? next : children };
  };
  return walk(root);
}

export function hiddenExcept(
  root: GuiNode,
  keep: ReadonlySet<string>,
): Set<string> {
  const hidden = new Set<string>();
  const holds = (node: GuiNode): boolean =>
    keep.has(idOf(node)) || (node.children ?? []).some(holds);
  const walk = (node: GuiNode) => {
    for (const child of node.children ?? []) {
      const id = idOf(child);
      if (keep.has(id)) continue;
      if (holds(child)) walk(child);
      else hidden.add(id);
    }
  };
  walk(root);

  if (root.background) hidden.add(idOf(root.background));
  return hidden;
}

export function hiddenExceptNamed(
  root: GuiNode,
  names: readonly string[],
): Set<string> {
  const keep = new Set(
    names.flatMap((name) => nodesNamed(root, name).map(idOf)).filter(Boolean),
  );
  const hidden = hiddenExcept(root, keep);
  const walk = (node: GuiNode) => {
    if (keep.has(idOf(node))) return;
    if (node.background) hidden.add(idOf(node.background));
    for (const child of node.children ?? []) walk(child);
  };
  walk(root);
  return hidden;
}

export function nodesNamed(root: GuiNode, name: string): GuiNode[] {
  const out: GuiNode[] = [];
  const walk = (node: GuiNode) => {
    if (node.name === name) out.push(node);
    for (const child of node.children ?? []) walk(child);
  };
  walk(root);
  return out;
}

export function findWithParent(
  root: GuiNode,
  name: string,
): { node: GuiNode; parent: GuiNode | null } | null {
  const walk = (
    node: GuiNode,
    parent: GuiNode | null,
  ): { node: GuiNode; parent: GuiNode | null } | null => {
    if (node.name === name) return { node, parent };
    for (const child of node.children ?? []) {
      const hit = walk(child, node);
      if (hit) return hit;
    }
    return null;
  };
  return walk(root, null);
}
