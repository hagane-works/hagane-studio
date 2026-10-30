import { guiRenderApi } from "@/lib/guiRenderApi";
import { editId, upsertEdit, type GuiEdit } from "../gui-renderer/guiOverlay";
import { rewriteOverlay } from "./guiTextOverlay";

export function withElementPicture(
  edits: readonly GuiEdit[],
  target: string,
  gfx: string,
): GuiEdit[] {
  const id = editId(target, "swap_sprite");

  const others = edits.filter(
    (edit) =>
      !(edit.op === "swap_sprite" && edit.target === target && edit.id !== id),
  );
  return upsertEdit(others, {
    id,
    target,
    op: "swap_sprite",
    payload: { gfx_ref: gfx },
  });
}

export function withoutElementPicture(
  edits: readonly GuiEdit[],
  target: string,
): GuiEdit[] {
  return edits.filter(
    (edit) => !(edit.op === "swap_sprite" && edit.target === target),
  );
}

export function saveElementPicture(
  at: { file: string; target: string },
  gfx: string,
): Promise<void> {
  return rewriteOverlay(at.file, (edits) =>
    withElementPicture(edits, at.target, gfx),
  );
}

export async function restoreElementPicture(at: {
  file: string;
  target: string;
  gfx: string;
}): Promise<void> {
  await rewriteOverlay(at.file, (edits) =>
    withoutElementPicture(edits, at.target),
  );

  await guiRenderApi.revert(at.gfx).catch(() => undefined);
}
