import { afterEach, describe, expect, it, vi } from "vitest";
import { guiRenderApi, type GuiOverlayResponse } from "@/lib/guiRenderApi";
import type { GuiEdit } from "../gui-renderer/guiOverlay";
import {
  GuiTextNotExportable,
  restoreGuiText,
  saveGuiText,
  withGuiText,
  withoutGuiText,
} from "./guiTextOverlay";

const TITLE = {
  file: "interface/countrypoliticsview.gui",
  target: "countrypoliticsview/title",
  originalText: "Politics",
  originalKind: "literal" as const,
};

const layout: GuiEdit = {
  id: "set_layout:countrypoliticsview/title",
  target: "countrypoliticsview/title",
  op: "set_layout",
  payload: { x: 4 },
};

describe("a .gui's words in its overlay", () => {
  it("gives the element a key of its own, in the language written", () => {
    const edits = withGuiText([layout], TITLE, "zh", "政治2222");
    expect(edits[0]).toBe(layout);
    expect(edits[1]).toMatchObject({
      id: "set_text:countrypoliticsview/title",
      target: "countrypoliticsview/title",
      op: "set_text",
      payload: { is_new: true, lang_texts: { zh: "政治2222" } },
    });
    expect(String(edits[1].payload.key)).toMatch(
      /^GUI_COUNTRYPOLITICSVIEW_TITLE_[0-9A-F]{8}$/,
    );
  });

  it("keeps the key and the other languages of an earlier rewrite", () => {
    const first = withGuiText([], TITLE, "en", "Politics 2");
    const key = first[0].payload.key;
    const second = withGuiText(
      first,
      { ...TITLE, originalText: `GUISUB_${key}`, originalKind: "loc" },
      "zh",
      "政治2222",
    );
    expect(second).toHaveLength(1);
    expect(second[0].payload).toEqual({
      key,
      is_new: true,
      lang_texts: { en: "Politics 2", zh: "政治2222" },
    });
  });

  it("drops an older edit for the element under another id, so it does not export twice", () => {
    const legacy: GuiEdit = {
      id: "legacy",
      target: TITLE.target,
      op: "set_text",
      payload: { text: "old" },
    };
    const edits = withGuiText([legacy], TITLE, "zh", "新");
    expect(edits.map((edit) => edit.id)).toEqual([
      "set_text:countrypoliticsview/title",
    ]);
  });

  it("gives the words back without touching the element's other edits", () => {
    const edits = withGuiText([layout], TITLE, "zh", "政治2222");
    expect(withoutGuiText(edits, TITLE.target)).toEqual([layout]);
  });
});

describe("saving a .gui's words", () => {
  afterEach(() => vi.restoreAllMocks());

  const row = (over: Partial<GuiOverlayResponse> = {}): GuiOverlayResponse => ({
    ok: true,
    project_id: 7,
    file: TITLE.file,
    source: "vanilla",
    scaffold_hash: "hash-1",
    schema_version: 1,
    edits: [layout],
    revision: "rev-1",
    materializable: true,
    ...over,
  });

  it("writes the whole row back against its revision", async () => {
    vi.spyOn(guiRenderApi, "loadOverlay").mockResolvedValue(row());
    const save = vi.spyOn(guiRenderApi, "saveOverlay").mockResolvedValue(row());
    await saveGuiText(TITLE, "zh", "政治2222");
    expect(save).toHaveBeenCalledTimes(1);
    const sent = save.mock.calls[0][0];
    expect(sent).toMatchObject({
      file: TITLE.file,
      scaffold_hash: "hash-1",
      expected_revision: "rev-1",
      expected_scaffold_hash: "hash-1",
      source: "vanilla",
      schema_version: 1,
    });
    expect(sent.edits.map((edit) => edit.op)).toEqual([
      "set_layout",
      "set_text",
    ]);
  });

  it("tries once more on the fresh row when something else saved the file in between", async () => {
    vi.spyOn(guiRenderApi, "loadOverlay")
      .mockResolvedValueOnce(row())
      .mockResolvedValueOnce(row({ revision: "rev-2", edits: [] }));
    const save = vi
      .spyOn(guiRenderApi, "saveOverlay")
      .mockRejectedValueOnce(
        Object.assign(new Error("conflict"), { status: 409 }),
      )
      .mockResolvedValueOnce(row());
    await saveGuiText(TITLE, "zh", "政治2222");
    expect(save).toHaveBeenCalledTimes(2);
    expect(save.mock.calls[1][0].expected_revision).toBe("rev-2");
    expect(save.mock.calls[1][0].edits.map((edit) => edit.op)).toEqual([
      "set_text",
    ]);
  });

  it("gives up after a second conflict", async () => {
    vi.spyOn(guiRenderApi, "loadOverlay").mockResolvedValue(row());
    vi.spyOn(guiRenderApi, "saveOverlay").mockRejectedValue(
      Object.assign(new Error("conflict"), { status: 409 }),
    );
    await expect(saveGuiText(TITLE, "zh", "x")).rejects.toThrow("conflict");
  });

  it("says so when the export has no file to put the rewrite into", async () => {
    vi.spyOn(guiRenderApi, "loadOverlay").mockResolvedValue(
      row({ materializable: false }),
    );
    const save = vi.spyOn(guiRenderApi, "saveOverlay");
    await expect(saveGuiText(TITLE, "zh", "x")).rejects.toBeInstanceOf(
      GuiTextNotExportable,
    );
    expect(save).not.toHaveBeenCalled();
  });

  it("puts the words back by dropping the element's text edit", async () => {
    const edits = withGuiText([layout], TITLE, "zh", "政治2222");
    vi.spyOn(guiRenderApi, "loadOverlay").mockResolvedValue(row({ edits }));
    const save = vi.spyOn(guiRenderApi, "saveOverlay").mockResolvedValue(row());
    await restoreGuiText(TITLE);
    expect(save.mock.calls[0][0].edits).toEqual([layout]);
  });
});
