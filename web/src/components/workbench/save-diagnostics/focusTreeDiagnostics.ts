import type { FocusNode, FocusTree } from "@/types";
import {
  currentSaveDiagnostics,
  readSaveDiagnostics,
  type SaveDiagnostic,
} from "@/lib/saveDiagnostics";

export type FocusTreeDiagnosticResponse = FocusNode & {
  diagnostics?: readonly SaveDiagnostic[];
  tree_diagnostics?: readonly SaveDiagnostic[];
};

export function applyFocusTreeDiagnostics<T extends FocusTree>(
  tree: T,
  updated: FocusTreeDiagnosticResponse,
): T {
  if (tree.id !== updated.focus_tree_id) return tree;
  const assessment = readSaveDiagnostics({
    diagnostics: updated.tree_diagnostics,
  });
  if (assessment === undefined) return tree;
  const current = currentSaveDiagnostics(assessment);
  const receipt = readSaveDiagnostics(updated);

  const treeWide: number[] = [];
  const byNode = new Map<string, number[]>();
  current.forEach((item, at) => {
    if (
      item.entity_ref.type === "focus_tree" &&
      item.entity_ref.id === String(tree.id)
    )
      treeWide.push(at);
    else if (item.entity_ref.type === "focus_node") {
      const list = byNode.get(item.entity_ref.id);
      if (list) list.push(at);
      else byNode.set(item.entity_ref.id, [at]);
    }
  });

  const findingsOf = (nodeID: number): SaveDiagnostic[] => {
    const own = byNode.get(String(nodeID)) ?? [];
    const out: SaveDiagnostic[] = [];
    let i = 0,
      j = 0;
    while (i < own.length || j < treeWide.length) {
      out.push(
        current[
          j >= treeWide.length || (i < own.length && own[i] < treeWide[j])
            ? own[i++]
            : treeWide[j++]
        ],
      );
    }
    return out;
  };
  return {
    ...tree,
    diagnostics: current,
    nodes: tree.nodes.map((row) => ({
      ...row,

      diagnostics:
        row.id === updated.id && receipt !== undefined
          ? receipt
          : findingsOf(row.id),
    })),
  };
}
