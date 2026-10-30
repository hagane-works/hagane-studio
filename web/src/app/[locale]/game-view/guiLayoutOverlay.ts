import type { LayoutOverride } from "../gui-renderer/GuiCanvas";
import { editId, upsertEdit, type GuiEdit } from "../gui-renderer/guiOverlay";
import { rewriteOverlay } from "./guiTextOverlay";

export function addressable(namePath: string | undefined): namePath is string {
  return (
    !!namePath &&
    !namePath.split("/").some((segment) => !segment || /#\d+$/.test(segment))
  );
}

export function withGuiLayout(
  edits: readonly GuiEdit[],
  target: string,
  to: LayoutOverride,
): GuiEdit[] {
  const id = editId(target, "set_layout");
  const existing = edits.find(
    (edit) =>
      edit.id === id || (edit.op === "set_layout" && edit.target === target),
  );
  const payload: Record<string, unknown> = { ...(existing?.payload ?? {}) };
  for (const key of ["x", "y", "width", "height"] as const) {
    if (typeof to[key] === "number") payload[key] = Math.round(to[key]!);
  }

  const others = edits.filter(
    (edit) =>
      !(edit.op === "set_layout" && edit.target === target && edit.id !== id),
  );
  return upsertEdit(others, { id, target, op: "set_layout", payload });
}

export function withLayoutBefore(
  edits: readonly GuiEdit[],
  target: string,
  before: GuiEdit | null,
): GuiEdit[] {
  const others = edits.filter(
    (edit) => !(edit.op === "set_layout" && edit.target === target),
  );
  return before ? [...others, before] : others;
}

export async function saveGuiLayout(
  file: string,
  target: string,
  to: LayoutOverride,
): Promise<GuiEdit | null> {
  let before: GuiEdit | null = null;
  await rewriteOverlay(file, (edits) => {
    before =
      edits.find(
        (edit) => edit.op === "set_layout" && edit.target === target,
      ) ?? null;
    return withGuiLayout(edits, target, to);
  });
  return before;
}

export function restoreGuiLayout(
  file: string,
  target: string,
  before: GuiEdit | null,
): Promise<void> {
  return rewriteOverlay(file, (edits) =>
    withLayoutBefore(edits, target, before),
  );
}
