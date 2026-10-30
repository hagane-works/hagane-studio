import { describe, expect, it } from "vitest";
import { renderToStaticMarkup } from "react-dom/server";
import type { GuiFontMeta, GuiNode } from "@/lib/guiRenderApi";
import { GuiCanvas, assignIds } from "./GuiCanvas";

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

function draw(disabled?: Set<string>) {
  const root = node({
    type: "containerwindowtype",
    name: "window",
    size: { width: 300, height: 100 },
    children: [
      node({
        name: "save_button",
        text: "S",
        font: "hoi_18mbs",
        size: { width: 221, height: 34 },
      }),
    ],
  });

  assignIds(root);
  return renderToStaticMarkup(
    <GuiCanvas
      node={root}
      sprites={{}}
      texts={{ S: { display: "S", kind: "loc" } }}
      fonts={FONT}
      versions={{}}
      hidden={new Set()}
      isolateId={null}
      selectedId={null}
      onSelect={() => {}}
      editMode={false}
      onActivate={() => {}}
      disabled={disabled}
    />,
  );
}

describe("a switched-off button", () => {
  it("is drawn grey and takes no click, as buttonstate.shader's Disable draws Save ★5 (designe 0:21)", () => {
    const off = draw(new Set(["root/0"]));
    expect(off).toContain("grayscale(1)");
    expect(off).not.toContain('aria-label="save_button"');
  });

  it("is left alone when on", () => {
    const on = draw();
    expect(on).not.toContain("grayscale(1)");
    expect(on).toContain('aria-label="save_button"');
  });
});
