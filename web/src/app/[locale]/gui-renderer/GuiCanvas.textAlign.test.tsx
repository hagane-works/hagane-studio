import { describe, expect, it } from "vitest";
import { renderToStaticMarkup } from "react-dom/server";
import type { GuiFontMeta, GuiNode } from "@/lib/guiRenderApi";
import { GuiCanvas, textAlignOf } from "./GuiCanvas";

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
  hoi_36header: {
    family: null,
    size: 36,
    outline: false,
    line_ratio: 1,
    bitmap: {
      line: 36,
      base: 28,
      atlas: "atlas.png",
      atlas_w: 64,
      atlas_h: 64,
      tint: "alpha",
      chars: { "84": [0, 0, 10, 36, 0, 0, 10] },
    },
  },
};

function title(format: string) {
  const root = node({
    type: "containerwindowtype",
    name: "countrydivisiondesignerview",
    size: { width: 1200, height: 720 },
    children: [
      node({
        name: "division_title",
        position: { x: 400, y: 4 },
        text: "T",
        font: "hoi_36header",
        format,
        maxwidth: 400,
        maxheight: 20,
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

describe("text alignment", () => {
  it("centres a `format = centre` box, the spelling 259 vanilla texts use", () => {
    const centred = title("centre");
    expect(centred).toContain("left:195px");
    expect(title("center")).toContain("left:195px");
    expect(title("left")).not.toContain("left:195px");
  });

  it("reads either spelling, and keeps the per-kind default", () => {
    expect(textAlignOf("centre")).toBe("center");
    expect(textAlignOf("CENTER")).toBe("center");
    expect(textAlignOf("right")).toBe("right");
    expect(textAlignOf(null)).toBe("left");

    expect(textAlignOf(null, "center")).toBe("center");

    expect(textAlignOf("upper_left")).toBe("left");
  });
});
