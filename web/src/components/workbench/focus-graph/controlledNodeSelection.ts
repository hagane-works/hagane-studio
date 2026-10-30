import type { Node, NodeChange } from "@xyflow/react";

type SelectionChange<NodeType extends Node = Node> = Extract<
  NodeChange<NodeType>,
  { type: "select" }
>;

export function splitControlledSelectionChanges<NodeType extends Node>(
  changes: NodeChange<NodeType>[],
): {
  selectionChanges: SelectionChange<NodeType>[];
  renderChanges: NodeChange<NodeType>[];
} {
  const selectionChanges: SelectionChange<NodeType>[] = [];
  const renderChanges: NodeChange<NodeType>[] = [];
  for (const change of changes) {
    if (change.type === "select") {
      selectionChanges.push(change as SelectionChange<NodeType>);
    } else {
      renderChanges.push(change);
    }
  }
  return { selectionChanges, renderChanges };
}

export function selectedNodeIdsAfterChanges<NodeType extends Node>(
  currentIds: readonly number[],
  changes: readonly SelectionChange<NodeType>[],
): number[] {
  const selected = new Set(currentIds);
  for (const change of changes) {
    const id = Number(change.id);
    if (!Number.isFinite(id)) continue;
    if (change.selected) selected.add(id);
    else selected.delete(id);
  }
  return [...selected].sort((a, b) => a - b);
}

export function sameSelectedNodeIds(
  left: readonly number[],
  right: readonly number[],
): boolean {
  if (left.length !== right.length) return false;
  const rightSet = new Set(right);
  return left.every((id) => rightSet.has(id));
}
