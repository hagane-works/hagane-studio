import {
  guiRenderApi,
  type GuiOverlayEdit,
  type TextKind,
} from "@/lib/guiRenderApi";
import { editId, upsertEdit, type GuiEdit } from "../gui-renderer/guiOverlay";
import { buildSetTextPayload } from "@/shared/gui-editor/overlayPayloads";

export interface GuiTextTarget {
  file: string;

  target: string;

  originalText: string;
  originalKind: TextKind | null;
}

export function withGuiText(
  edits: readonly GuiEdit[],
  at: GuiTextTarget,
  lang: string,
  text: string,
): GuiEdit[] {
  const id = editId(at.target, "set_text");
  const existing = edits.find(
    (edit) =>
      edit.id === id || (edit.op === "set_text" && edit.target === at.target),
  );
  const langTexts: Record<string, string> = {};
  const before = existing?.payload?.lang_texts;
  if (before && typeof before === "object" && !Array.isArray(before)) {
    for (const [code, value] of Object.entries(before))
      if (typeof value === "string") langTexts[code] = value;
  }
  langTexts[lang] = text;

  const others = edits.filter(
    (edit) =>
      !(edit.op === "set_text" && edit.target === at.target && edit.id !== id),
  );
  return upsertEdit(others, {
    id,
    target: at.target,
    op: "set_text",
    payload: buildSetTextPayload({
      target: at.target,
      originalText: at.originalText,
      originalKind: at.originalKind,
      langTexts,
      existingPayload: existing?.payload,
    }),
  });
}

export function withoutGuiText(
  edits: readonly GuiEdit[],
  target: string,
): GuiEdit[] {
  return edits.filter(
    (edit) => !(edit.op === "set_text" && edit.target === target),
  );
}

export async function rewriteOverlay(
  file: string,
  change: (edits: GuiEdit[]) => GuiEdit[],
): Promise<void> {
  for (let attempt = 0; ; attempt++) {
    const row = await guiRenderApi.loadOverlay(file);
    if (row.materializable === false || !row.scaffold_hash)
      throw new GuiTextNotExportable();
    try {
      await guiRenderApi.saveOverlay({
        file,
        edits: change((row.edits ?? []) as GuiEdit[]) as GuiOverlayEdit[],
        scaffold_hash: row.scaffold_hash,
        expected_revision: row.revision || "0",
        expected_scaffold_hash: row.scaffold_hash,
        source: row.source || undefined,
        schema_version: 1,
      });
      return;
    } catch (err) {
      if ((err as { status?: unknown } | null)?.status === 409 && attempt === 0)
        continue;
      throw err;
    }
  }
}

export function saveGuiText(
  at: GuiTextTarget,
  lang: string,
  text: string,
): Promise<void> {
  return rewriteOverlay(at.file, (edits) => withGuiText(edits, at, lang, text));
}

export function restoreGuiText(
  at: Pick<GuiTextTarget, "file" | "target">,
): Promise<void> {
  return rewriteOverlay(at.file, (edits) => withoutGuiText(edits, at.target));
}

export class GuiTextNotExportable extends Error {
  constructor() {
    super("gui file not exportable");
    this.name = "GuiTextNotExportable";
  }
}
