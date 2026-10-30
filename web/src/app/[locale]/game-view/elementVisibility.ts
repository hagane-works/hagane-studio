import { editId, upsertEdit, type GuiEdit } from "../gui-renderer/guiOverlay";
import { rewriteOverlay } from "./guiTextOverlay";

export function addedNodeEdit(
  edits: readonly GuiEdit[],
  target: string,
): GuiEdit | undefined {
  return edits.find((edit) => {
    if (edit.op !== "add_node") return false;
    const node = edit.payload?.node as { name?: unknown } | undefined;
    return (
      typeof node?.name === "string" && `${edit.target}/${node.name}` === target
    );
  });
}

export function withElementHidden(
  edits: readonly GuiEdit[],
  target: string,
): GuiEdit[] {
  return upsertEdit([...edits], {
    id: editId(target, "hide_node"),
    target,
    op: "hide_node",
    payload: {},
  });
}

export function withElementShown(
  edits: readonly GuiEdit[],
  target: string,
): GuiEdit[] {
  return edits.filter(
    (edit) => !(edit.op === "hide_node" && edit.target === target),
  );
}

export function withoutAddedElement(
  edits: readonly GuiEdit[],
  target: string,
): { edits: GuiEdit[]; removed: GuiEdit[] } {
  const added = addedNodeEdit(edits, target);
  const gone = (edit: GuiEdit) =>
    edit === added ||
    edit.target === target ||
    edit.target.startsWith(`${target}/`);
  return {
    edits: edits.filter((edit) => !gone(edit)),
    removed: edits.filter(gone),
  };
}

export type Removal = "hidden" | "removed";

export async function removeElement(at: {
  file: string;
  target: string;
}): Promise<{ kind: Removal; undo: () => Promise<void> }> {
  let kind = "hidden" as Removal;
  let removed: GuiEdit[] = [];
  await rewriteOverlay(at.file, (edits) => {
    if (addedNodeEdit(edits, at.target)) {
      const out = withoutAddedElement(edits, at.target);
      kind = "removed";
      removed = out.removed;
      return out.edits;
    }
    kind = "hidden";
    return withElementHidden(edits, at.target);
  });
  const undo =
    kind === "removed"
      ? () =>
          rewriteOverlay(at.file, (edits) => [
            ...edits,
            ...removed.filter((edit) => !edits.some((e) => e.id === edit.id)),
          ])
      : () => showElement(at);
  return { kind, undo };
}

export function showElement(at: {
  file: string;
  target: string;
}): Promise<void> {
  return rewriteOverlay(at.file, (edits) => withElementShown(edits, at.target));
}
