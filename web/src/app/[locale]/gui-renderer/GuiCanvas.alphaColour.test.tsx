import { describe, expect, it } from "vitest";
import { renderToStaticMarkup } from "react-dom/server";
import type { GuiFontMeta, GuiNode } from "@/lib/guiRenderApi";
import { GuiCanvas } from "./GuiCanvas";
import { REFERENCE_TABLES } from "@/lib/__fixtures__/referenceTables";
import { setReferenceTables } from "@/lib/referenceTables";

setReferenceTables(REFERENCE_TABLES);

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
  hoi_20b: {
    family: null,
    size: 20,
    outline: false,
    line_ratio: 1,
    bitmap: {
      line: 20,
      base: 16,
      atlas: "atlas.png",
      atlas_w: 64,
      atlas_h: 64,
      tint: "alpha",
      chars: { "84": [0, 0, 10, 20, 0, 0, 10] },
    },
  },
};

function draw(text: Partial<GuiNode>) {
  const root = node({
    type: "containerwindowtype",
    name: "window",
    size: { width: 300, height: 100 },
    children: [
      node({
        name: "label",
        text: "T",
        font: "hoi_20b",
        maxwidth: 200,
        ...text,
      }),
    ],
  });
  return renderToStaticMarkup(
    <GuiCanvas
      node={root}
      sprites={{}}
      texts={{ T: { display: "T", kind: "loc" } }}
      fonts={FONT}
      versions={{}}
      hidden={new Set()}
      isolateId={null}
      selectedId={null}
      onSelect={() => {}}
      editMode={false}
    />,
  );
}

describe("the .gui's own colour and see-through", () => {
  it("draws `text_color_code = b` in the game's black (core.gfx textcolors)", () => {
    expect(draw({ text_color: "b" })).toContain(
      "background-color:rgb(0, 0, 0)",
    );
    expect(draw({})).not.toContain("background-color:rgb(0, 0, 0)");
  });

  it("draws an element with `alpha` see-through", () => {
    expect(draw({ raw: { alpha: 0.42 } } as Partial<GuiNode>)).toContain(
      "opacity:0.42",
    );
    expect(draw({})).not.toContain("opacity:");
  });
});
