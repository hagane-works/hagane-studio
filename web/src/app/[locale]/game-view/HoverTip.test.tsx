import { readFileSync } from "node:fs";
import { join } from "node:path";
import { afterEach, describe, expect, it } from "vitest";
import { cleanup, render, screen } from "@testing-library/react";
import { HoverTip } from "./HoverTip";

afterEach(cleanup);

describe("a hover tip", () => {
  it("describes its control, unseen until pointed at or focused", () => {
    render(
      <HoverTip tip="点界面上任何东西改它的图和字">
        {(id) => (
          <button type="button" aria-describedby={id}>
            UI 编辑
          </button>
        )}
      </HoverTip>,
    );
    const button = screen.getByRole("button", { name: "UI 编辑" });
    const tip = screen.getByRole("tooltip");
    expect(button.getAttribute("aria-describedby")).toBe(tip.id);
    expect(tip.textContent).toBe("点界面上任何东西改它的图和字");
    expect(tip.className).toContain("opacity-0");
    expect(tip.className).toContain("group-hover/tip:opacity-100");
    expect(tip.className).toContain("group-focus-within/tip:opacity-100");
    expect(tip.className).toContain("pointer-events-none");

    expect(tip.className).toContain("bottom-full");
    expect(tip.className).toContain("right-0");
  });

  it("leaves its control alone when there is nothing to tell", () => {
    render(
      <HoverTip tip={null}>
        {(id) => (
          <button type="button" aria-describedby={id}>
            退出 UI 编辑
          </button>
        )}
      </HoverTip>,
    );
    expect(screen.queryByRole("tooltip")).toBeNull();
    expect(
      screen
        .getByRole("button", { name: "退出 UI 编辑" })
        .hasAttribute("aria-describedby"),
    ).toBe(false);
  });

  it("can open under its control and from its left edge", () => {
    render(
      <HoverTip tip="拖到界面上" side="bottom" align="left">
        {(id) => <span aria-describedby={id}>组件</span>}
      </HoverTip>,
    );
    const tip = screen.getByRole("tooltip");
    expect(tip.className).toContain("top-full");
    expect(tip.className).toContain("left-0");
  });

  it("can stand left of its control, level with it", () => {
    render(
      <HoverTip tip="拖到界面上" side="left">
        {(id) => <span aria-describedby={id}>组件</span>}
      </HoverTip>,
    );
    const tip = screen.getByRole("tooltip");
    expect(tip.className).toContain("right-full");
    expect(tip.className).toContain("-translate-y-1/2");
  });

  it("holds the UI 编辑 and 文本编辑 help in GameView's bar, not a line beside the buttons", () => {
    const page = readFileSync(join(__dirname, "page.tsx"), "utf8");

    expect(page).toContain('<HoverTip tip={uiEdit ? null : t("uiEditHint")}>');
    expect(page).toContain(
      '<HoverTip tip={editMode ? null : t("textEditHint")}>',
    );
    expect(page).not.toContain('t("mode.uiEditing")');

    const flat = page.replace(/\s+/g, " ");
    expect(flat).toContain(
      '{mapStudio && ( <span className="rounded-md border border-ingame-edge/65 bg-ingame-panel px-2 py-1 text-xs text-ingame-text shadow-lg"> {t("mode.mapStudio")} </span> )}',
    );
  });
});
