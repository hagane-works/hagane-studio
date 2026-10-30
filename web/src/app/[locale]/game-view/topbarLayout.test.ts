import { readFileSync } from "node:fs";
import { join } from "node:path";
import { describe, expect, it } from "vitest";
import type { GuiNode } from "@/lib/guiRenderApi";
import { assignIds, measureWindow } from "../gui-renderer/GuiCanvas";
import {
  hiddenExcept,
  hiddenExceptNamed,
  patchTree,
  pinFullSizes,
} from "./topbarLayout";

function node(partial: Partial<GuiNode>): GuiNode {
  return {
    type: "icontype",
    name: null,
    position: { x: 0, y: 0 },
    size: null,
    orientation: null,
    origo: null,
    scale: null,
    sprite: null,
    frame: null,
    text: null,
    text_source: null,
    font: null,
    format: null,
    maxwidth: null,
    maxheight: null,
    text_color: null,
    centerposition: false,
    alwaystransparent: false,
    background: null,
    children: [],
    ...partial,
  };
}

const button = (
  name: string,
  x: number,
  y: number,
  orientation: string | null = "UPPER_RIGHT",
) =>
  node({
    type: "buttontype",
    name,
    position: { x, y },
    orientation,
    size: { width: 38, height: 38 },
  });

function topbar(): GuiNode {
  const root = node({
    type: "containerwindowtype",
    name: "topbar",
    position: { x: -4, y: -1 },
    size: { width: "100%%", height: "100%%" },
    children: [
      button("threat_button", -96, 20),
      node({
        type: "containerwindowtype",
        name: "country_menu_window",
        position: { x: 217, y: 36 },
        size: { width: "100%%", height: 30 },
        children: [
          button("army_button", -275, 8),
          button("navy_button", -226, 8),
          button("air_button", -180, 8),
        ],
      }),
    ],
  });
  assignIds(root);
  return root;
}

describe("pinFullSizes", () => {
  it("runs a 100% size from the element's own offset to the parent's far edge", () => {
    const pinned = pinFullSizes(topbar(), 1920, 1080);

    expect(pinned.size).toEqual({ width: 1924, height: 1081 });
    expect(pinned.children[1].size).toEqual({ width: 1707, height: 30 });
  });

  it("puts Army/Navy/Air into their sockets and the globe where game_0136 has them", () => {
    const pinned = pinFullSizes(topbar(), 1920, 1080);
    const rects = measureWindow(pinned, {}, 1920, 1080);

    expect(rects["name:army_button"].x - 4).toBe(1645);
    expect(rects["name:navy_button"].x - 4).toBe(1694);
    expect(rects["name:air_button"].x - 4).toBe(1740);
    expect(rects["name:army_button"].y - 1).toBe(43);
    expect(rects["name:threat_button"].x - 4).toBe(1824);
  });

  it("leaves every other percentage and far-edge axis as GuiCanvas reads it", () => {
    const root = node({
      type: "containerwindowtype",
      size: { width: 400, height: 400 },
      children: [
        node({
          type: "containerwindowtype",
          name: "half",
          position: { x: 20, y: 0 },
          size: { width: "50%%", height: 10 },
        }),
        node({
          type: "containerwindowtype",
          name: "right",
          position: { x: 20, y: 0 },
          orientation: "UPPER_RIGHT",
          size: { width: "100%%", height: 10 },
        }),
      ],
    });
    const pinned = pinFullSizes(root, 1920, 1080);
    expect(pinned.children[0].size?.width).toBe("50%%");
    expect(pinned.children[1].size?.width).toBe("100%%");
  });

  it("ends a negative size that far short of the parent's far edge, from its own offset, when asked", () => {
    const root = node({
      type: "containerwindowtype",
      size: { width: 1280, height: 720 },
      children: [
        node({
          type: "containerwindowtype",
          name: "history",
          position: { x: 320, y: 90 },
          size: { width: -5, height: -5 },
        }),
      ],
    });
    expect(
      pinFullSizes(root, 1920, 1080, { negative: true }).children[0].size,
    ).toEqual({ width: 955, height: 625 });

    expect(pinFullSizes(root, 1920, 1080).children[0].size).toEqual({
      width: -5,
      height: -5,
    });
  });

  it("returns the tree itself when nothing is pinned", () => {
    const plain = node({
      type: "containerwindowtype",
      size: { width: 100, height: 100 },
      children: [node({ name: "a" })],
    });
    expect(pinFullSizes(plain, 1920, 1080)).toBe(plain);
  });
});

describe("patchTree", () => {
  it("copies only the patched path and keeps the ids", () => {
    const root = topbar();
    const army = root.children[1].children[0] as GuiNode & { _id: string };
    const patched = patchTree(root, { [army._id]: { frame: 2 } });
    expect(patched).not.toBe(root);
    expect(patched.children[1].children[0].frame).toBe(2);
    expect(
      (patched.children[1].children[0] as GuiNode & { _id: string })._id,
    ).toBe(army._id);

    expect(patched.children[0]).toBe(root.children[0]);
    expect(patched.children[1].children[1]).toBe(root.children[1].children[1]);
    expect(army.frame).toBeNull();
  });

  it("is the tree itself when there is nothing to patch", () => {
    const root = topbar();
    expect(patchTree(root, {})).toBe(root);
  });
});

describe("hiddenExcept", () => {
  it("hides everything but the kept elements, keeping their ancestors shown", () => {
    const root = topbar();
    const navy = (root.children[1].children[1] as GuiNode & { _id: string })
      ._id;
    const hidden = hiddenExcept(root, new Set([navy]));
    const id = (n: GuiNode) => (n as GuiNode & { _id: string })._id;
    expect(hidden.has(id(root.children[0]))).toBe(true);
    expect(hidden.has(id(root.children[1]))).toBe(false);
    expect(hidden.has(id(root.children[1].children[0]))).toBe(true);
    expect(hidden.has(navy)).toBe(false);
    expect(hidden.has(id(root.children[1].children[2]))).toBe(true);
  });
});

describe("hiddenExceptNamed", () => {
  it("keeps the named elements and hides their containers' backgrounds", () => {
    const root = topbar();
    const menu = root.children[1];
    menu.background = node({
      type: "background",
      name: "Background",
      sprite: "GFX_topbar_menu_bg",
    });
    root.background = node({
      type: "background",
      name: "Background",
      sprite: "GFX_topbar_bg",
    });
    assignIds(root);
    const id = (n: GuiNode) => (n as GuiNode & { _id: string })._id;
    const hidden = hiddenExceptNamed(root, ["navy_button"]);
    expect(hidden.has(id(menu.children[1]))).toBe(false);
    expect(hidden.has(id(menu.children[0]))).toBe(true);
    expect(hidden.has(id(root.children[0]))).toBe(true);
    expect(hidden.has(id(menu.background))).toBe(true);
    expect(hidden.has(id(root.background))).toBe(true);
  });
});

describe("the topbar in UI 编辑", () => {
  const layer = readFileSync(join(__dirname, "TopbarLayer.tsx"), "utf8");
  it("lets the clock canvas pick its elements", () => {
    expect(layer).toContain(
      'const editProps = edit("topbar:clock", win, win.node, win.sprites, model.textOverrides);',
    );
    expect(layer).not.toContain("editMode={false}");
  });
  it("draws the trophy and the ! on a pickable canvas", () => {
    expect(layer.replace(/\s+/g, " ")).toContain(
      "{uiEdit ? ( <DisabledButtonsEdit win={win} /> ) : (",
    );
    expect(layer).toContain("hiddenExceptNamed(win.node, DISABLED_BUTTONS)");
  });
});
