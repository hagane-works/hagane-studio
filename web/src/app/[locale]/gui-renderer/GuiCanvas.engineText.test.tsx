import { describe, expect, it } from "vitest";
import { renderToStaticMarkup } from "react-dom/server";
import type { GuiFontMeta, GuiNode } from "@/lib/guiRenderApi";
import { GuiCanvas, assignIds } from "./GuiCanvas";

function node(partial: Partial<GuiNode>): GuiNode {
  return {
    type: "instanttextboxtype",
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

function draw(overrides: Record<string, string>) {
  const root = node({
    type: "containerwindowtype",
    name: "window",
    size: { width: 400, height: 100 },

    children: [
      node({
        name: "production_value",
        font: "hoi_18mbs",
        maxwidth: 350,
        maxheight: 25,
      }),
    ],
  });
  assignIds(root);
  return renderToStaticMarkup(
    <GuiCanvas
      node={root}
      sprites={{}}
      texts={{}}
      fonts={FONT}
      versions={{}}
      hidden={new Set()}
      isolateId={null}
      selectedId={null}
      onSelect={() => {}}
      editMode={false}
      textOverrides={overrides}
    />,
  );
}

describe("a box the engine fills", () => {
  it("prints the caller's text though the .gui names none (designer production_value)", () => {
    expect(draw({ "root/0": "S" })).toContain("background-image");
    expect(draw({})).not.toContain("background-image");
  });
});
