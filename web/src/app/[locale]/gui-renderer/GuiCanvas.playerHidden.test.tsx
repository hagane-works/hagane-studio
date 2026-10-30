import { describe, expect, it } from "vitest";
import { renderToStaticMarkup } from "react-dom/server";
import type { GuiFontMeta, GuiNode } from "@/lib/guiRenderApi";
import {
  GuiCanvas,
  assignIds,
  isPlayerHidden,
  measureWindow,
} from "./GuiCanvas";

function node(partial: Partial<GuiNode>): GuiNode {
  return {
    type: "buttontype",
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

const FONT: Record<string, GuiFontMeta> = {
  hoi_18mbs: {
    family: null,
    size: 18,
    outline: false,
    line_ratio: 1,
    bitmap: {
      line: 20,
      base: 16,
      atlas: "atlas.png",
      atlas_w: 64,
      atlas_h: 64,
      tint: "alpha",
      chars: { "83": [0, 0, 10, 20, 0, 0, 10] },
    },
  },
};

function screen(): GuiNode {
  const root = node({
    type: "containerwindowtype",
    name: "window",
    size: { width: 300, height: 100 },
    children: [
      node({
        name: "gone",
        text: "S",
        font: "hoi_18mbs",
        size: { width: 100, height: 34 },
        raw: { gameview_hidden: "yes" },
      }),
      node({
        name: "kept",
        text: "S",
        font: "hoi_18mbs",
        position: { x: 0, y: 50 },
        size: { width: 100, height: 34 },
      }),
    ],
  });
  assignIds(root);
  return root;
}

function draw(editMode: boolean) {
  return renderToStaticMarkup(
    <GuiCanvas
      node={screen()}
      sprites={{}}
      texts={{ S: { display: "S", kind: "loc" } }}
      fonts={FONT}
      versions={{}}
      hidden={new Set()}
      isolateId={null}
      selectedId={null}
      onSelect={() => {}}
      editMode={editMode}
      onActivate={() => {}}
    />,
  );
}

describe("an element the player hid", () => {
  it("is not on the screen and measures as hidden, as in the game", () => {
    const play = draw(false);
    expect(play).not.toContain("data-player-hidden");
    expect(play).not.toContain('aria-label="gone"');
    expect(play).toContain('aria-label="kept"');
    const root = screen();
    const rects = measureWindow(root, {}, 1920, 1080);
    expect(rects["name:gone"]?.hidden).toBe(true);
    expect(rects["name:kept"]?.hidden).toBe(false);
  });

  it("is drawn faint in UI 编辑, where it can be picked and shown again", () => {
    const edit = draw(true);
    expect(edit).toContain("data-player-hidden");
    expect(edit).toContain("opacity:0.3");
  });

  it("is told from the .gui's own hide by its mark", () => {
    expect(isPlayerHidden(node({ raw: { gameview_hidden: "yes" } }))).toBe(
      true,
    );
    expect(isPlayerHidden(node({ raw: { hide: "yes" } }))).toBe(false);
    expect(isPlayerHidden(node({}))).toBe(false);
  });
});
