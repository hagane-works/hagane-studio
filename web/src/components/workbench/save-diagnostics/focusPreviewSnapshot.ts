export interface FocusScriptPreview {
  script: string;
  tree_id?: number;
  node_id?: number;
  tree_edit_revision?: string;
}

export interface FocusPreviewSnapshot {
  projectId: number;
  treeId: number;
  nodeId: number | null;
  editRevision?: string;
}

export function captureFocusPreview(
  response: FocusScriptPreview,
  projectId: number,
  treeId: number,
  nodeId?: number,
): FocusPreviewSnapshot | null {
  if (
    response.tree_id !== treeId ||
    (response.node_id ?? null) !== (nodeId ?? null)
  )
    return null;
  return {
    projectId,
    treeId,
    nodeId: nodeId ?? null,
    editRevision:
      typeof response.tree_edit_revision === "string" &&
      /^[1-9]\d*$/.test(response.tree_edit_revision)
        ? response.tree_edit_revision
        : undefined,
  };
}

export function focusPreviewForSelection(
  script: string,
  snapshot: FocusPreviewSnapshot | null,
  projectId: number,
  treeId: number | null,
  nodeId: number | null,
) {
  const previewReady =
    snapshot?.projectId === projectId &&
    snapshot.treeId === treeId &&
    snapshot.nodeId === nodeId;
  return {
    scriptPreview: previewReady ? script : "",
    editRevision: previewReady ? snapshot.editRevision : undefined,
    previewReady,
  };
}
