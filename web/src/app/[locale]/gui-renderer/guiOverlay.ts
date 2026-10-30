import type { GuiNode } from "@/lib/guiRenderApi";

export type EditOp =
  | "set_text"
  | "swap_sprite"
  | "set_layout"
  | "set_orientation"
  | "set_origo"
  | "hide_node"
  | "add_node"
  | "delete_node";

export interface GuiEdit {
  id: string;

  target: string;
  op: EditOp;
  payload: Record<string, unknown>;
}

export interface GuiOverlay {
  schema_version: number;
  project_id: number | null;
  file: string;
  source: string;
  scaffold_hash: string;
  edits: GuiEdit[];
}

interface IdNode extends GuiNode {
  _id?: string;
}

function segment(node: GuiNode, indexFallback: number): string {
  const name = (node.name || "").trim();
  if (name) return name;
  const t = node.type.replace(/type$/i, "");
  return `${t}#${indexFallback}`;
}

export function buildNamePathMap(root: GuiNode): Record<string, string> {
  const map: Record<string, string> = {};
  const walk = (node: IdNode, parentPath: string, index: number) => {
    const seg = segment(node, index);
    const path = parentPath ? `${parentPath}/${seg}` : seg;
    if (node._id) map[node._id] = path;
    (node.children || []).forEach((c, i) => walk(c as IdNode, path, i));
  };

  const r = root as IdNode;
  const rootSeg = segment(r, 0);
  if (r._id) map[r._id] = rootSeg;
  (r.children || []).forEach((c, i) => walk(c as IdNode, rootSeg, i));
  return map;
}

export function editId(target: string, op: EditOp): string {
  return `${op}:${target}`;
}

export function upsertEdit(edits: GuiEdit[], next: GuiEdit): GuiEdit[] {
  const idx = edits.findIndex((e) => e.id === next.id);
  if (idx === -1) return [...edits, next];
  const copy = edits.slice();
  copy[idx] = next;
  return copy;
}

export function removeEdit(edits: GuiEdit[], id: string): GuiEdit[] {
  return edits.filter((e) => e.id !== id);
}

export function removeEditsForTarget(
  edits: GuiEdit[],
  target: string,
): GuiEdit[] {
  return edits.filter((e) => e.target !== target);
}

export function addNodeEdit(
  parentPath: string,
  node: Record<string, unknown>,
  index?: number,
): GuiEdit {
  const rawName = typeof node.name === "string" ? node.name.trim() : "";
  const typeStr = typeof node.type === "string" ? node.type : "node";

  const keySeg = rawName || `${typeStr.replace(/type$/i, "")}#new`;
  const payload: Record<string, unknown> = { node };
  if (typeof index === "number") payload.index = index;
  return {
    id: `add_node:${parentPath}/${keySeg}`,
    target: parentPath,
    op: "add_node",
    payload,
  };
}

export function deleteNodeEdit(target: string): GuiEdit {
  return {
    id: `delete_node:${target}`,
    target,
    op: "delete_node",
    payload: {},
  };
}
