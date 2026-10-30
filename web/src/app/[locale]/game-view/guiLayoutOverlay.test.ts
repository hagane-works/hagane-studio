import { readFileSync } from "node:fs";
import { join } from "node:path";
import { describe, expect, it } from "vitest";
import type { GuiEdit } from "../gui-renderer/guiOverlay";
import {
  addressable,
  withGuiLayout,
  withLayoutBefore,
} from "./guiLayoutOverlay";

describe("moving a game window's own element", () => {
  it("moves only an element the backend can find by its name", () => {
    expect(addressable("countrypoliticsview/political_power_text")).toBe(true);
    expect(addressable("countrypoliticsview/iconType#3")).toBe(false);
    expect(addressable(undefined)).toBe(false);
    expect(addressable("countrypoliticsview//x")).toBe(false);
  });

  it("merges a move into the element's layout edit, keeping an earlier stretch", () => {
    const text: GuiEdit = {
      id: "w/title::set_text",
      target: "w/title",
      op: "set_text",
      payload: { lang_texts: { en: "Hi" } },
    };
    const moved = withGuiLayout([text], "w/box", { x: 40.4, y: 120 });
    expect(moved).toContainEqual(text);
    const layout = moved.find((edit) => edit.op === "set_layout");
    expect(layout).toMatchObject({
      target: "w/box",
      payload: { x: 40, y: 120 },
    });
    const stretched = withGuiLayout(moved, "w/box", { width: 300, height: 90 });
    expect(stretched.filter((edit) => edit.op === "set_layout")).toHaveLength(
      1,
    );
    expect(stretched.find((edit) => edit.op === "set_layout")?.payload).toEqual(
      { x: 40, y: 120, width: 300, height: 90 },
    );
  });

  it("puts the element back as it was: its earlier layout edit, or none", () => {
    const earlier: GuiEdit = {
      id: "w/box::set_layout",
      target: "w/box",
      op: "set_layout",
      payload: { x: 10, y: 10 },
    };
    const after = withGuiLayout([earlier], "w/box", { x: 99, y: 99 });
    expect(withLayoutBefore(after, "w/box", earlier)).toEqual([earlier]);
    expect(withLayoutBefore(after, "w/box", null)).toEqual([]);
  });

  it("is wired into UI 编辑's screens, and a wobble only picks", () => {
    const page = readFileSync(join(__dirname, "page.tsx"), "utf8");
    expect(page).toContain("onMoveGuiNode: uiEdit ? moveGuiNode : undefined,");
    expect(page).toContain("dragThreshold={layoutEditable ? 4 : 0}");
    expect(page).toContain(
      "const layoutEditable = !!onMoveGuiNode && !shown.ui_panel && !!shown.file && !!(shown.scaffold_hash || shown.current_scaffold_hash);",
    );
  });
});
