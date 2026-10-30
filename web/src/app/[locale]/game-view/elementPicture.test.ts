import { describe, expect, it } from "vitest";
import type { GuiEdit } from "../gui-renderer/guiOverlay";
import { withElementPicture, withoutElementPicture } from "./elementPicture";

const text: GuiEdit = {
  id: "set_text:politics/faction",
  target: "politics/faction",
  op: "set_text",
  payload: { key: "X" },
};

describe("withElementPicture", () => {
  it("points the element at its own picture, keeping the file's other edits", () => {
    const edits = withElementPicture(
      [text],
      "politics/faction",
      "GFX_gv_faction_1",
    );
    expect(edits).toEqual([
      text,
      {
        id: "swap_sprite:politics/faction",
        target: "politics/faction",
        op: "swap_sprite",
        payload: { gfx_ref: "GFX_gv_faction_1" },
      },
    ]);
  });

  it("replaces an earlier swap of the same element, under any id", () => {
    const older: GuiEdit = {
      id: "legacy",
      target: "politics/faction",
      op: "swap_sprite",
      payload: { gfx_ref: "GFX_old" },
    };
    const edits = withElementPicture(
      [older, text],
      "politics/faction",
      "GFX_gv_faction_1",
    );
    expect(edits.filter((edit) => edit.op === "swap_sprite")).toEqual([
      {
        id: "swap_sprite:politics/faction",
        target: "politics/faction",
        op: "swap_sprite",
        payload: { gfx_ref: "GFX_gv_faction_1" },
      },
    ]);
  });
});

describe("withoutElementPicture", () => {
  it("drops only the element's swap", () => {
    const swap = withElementPicture(
      [text],
      "politics/faction",
      "GFX_gv_faction_1",
    );
    const other = withElementPicture(swap, "politics/close", "GFX_gv_close_1");
    expect(withoutElementPicture(other, "politics/faction")).toEqual([
      text,
      {
        id: "swap_sprite:politics/close",
        target: "politics/close",
        op: "swap_sprite",
        payload: { gfx_ref: "GFX_gv_close_1" },
      },
    ]);
  });
});
