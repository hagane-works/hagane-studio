import { describe, expect, it } from "vitest";
import { renderToStaticMarkup } from "react-dom/server";
import type { GuiNode } from "@/lib/guiRenderApi";
import { GuiCanvas, assignIds, measureWindow } from "./GuiCanvas";
import {
  clipRects,
  containerScroll,
  withContainerScroll,
} from "./containerScroll";

function node(partial: Partial<GuiNode>): GuiNode {
  return {
    type: "containerwindowtype",
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

const page = () =>
  node({
    name: "window",
    size: { width: 800, height: 600 },
    children: [
      node({
        name: "tabs",
        position: { x: 0, y: 0 },
        size: { width: 800, height: 40 },
      }),
      node({
        name: "page_content",
        position: { x: 20, y: 50 },
        size: { width: 400, height: 300 },
        children: [
          node({ name: "strip", size: { width: 400, height: 900 } }),
          node({
            type: "icontype",
            name: "deep_icon",
            sprite: "GFX_deep",
            position: { x: 10, y: 700 },
            size: { width: 30, height: 30 },
          }),
        ],
      }),
    ],
  });

const isPage = (n: GuiNode) => n.name === "page_content";

describe("a scrolled container", () => {
  it("is marked on the node, and only where asked", () => {
    const root = withContainerScroll(page(), isPage, { x: 0, y: 120 });
    expect(containerScroll(root.children[1])).toEqual({ x: 0, y: 120 });
    expect(containerScroll(root)).toBeNull();
    expect(containerScroll(root.children[0])).toBeNull();
  });

  it("copies only the path down to it", () => {
    const original = page();
    const root = withContainerScroll(original, isPage, { x: 0, y: 120 });
    expect(root).not.toBe(original);

    expect(root.children[0]).toBe(original.children[0]);
    expect(root.children[1].children[0]).toBe(original.children[1].children[0]);

    expect(containerScroll(original.children[1])).toBeNull();

    expect(withContainerScroll(original, () => false, { x: 0, y: 120 })).toBe(
      original,
    );
  });

  it("moves what it holds, not itself, and keeps every id", () => {
    const at = measureWindow(page(), {}, 800, 600);
    const scrolled = measureWindow(
      withContainerScroll(page(), isPage, { x: 5, y: 120 }),
      {},
      800,
      600,
    );
    expect(scrolled["name:page_content"]).toMatchObject({
      x: 20,
      y: 50,
      w: 400,
      h: 300,
    });
    expect(scrolled["name:deep_icon"]).toMatchObject({
      x: at["name:deep_icon"].x - 5,
      y: at["name:deep_icon"].y - 120,
    });
    expect(Object.keys(scrolled).sort()).toEqual(Object.keys(at).sort());
  });

  it("is drawn cut at its own box", () => {
    const root = withContainerScroll(page(), isPage, { x: 0, y: 120 });
    assignIds(root);
    const html = renderToStaticMarkup(
      <GuiCanvas
        node={root}
        sprites={{}}
        texts={{}}
        fonts={{}}
        versions={{}}
        hidden={new Set()}
        isolateId={null}
        selectedId={null}
        onSelect={() => {}}
        editMode={false}
        showMissingSprites
        baseW={800}
        baseH={600}
      />,
    );
    expect(html).toContain(
      "left:20px;top:50px;width:400px;height:300px;overflow:hidden",
    );

    expect(html).toContain("left:30px;top:630px");
  });
});

describe("cutting rects to a box", () => {
  const box = { x: 0, y: 100, w: 200, h: 100 };
  it("keeps what is inside, cuts what crosses, drops what is outside", () => {
    const inside = { x: 10, y: 110, w: 20, h: 20, id: "a" };
    const crossing = { x: 190, y: 90, w: 40, h: 40, id: "b" };
    const outside = { x: 10, y: 10, w: 20, h: 20, id: "c" };
    const out = clipRects([inside, crossing, outside], box);
    expect(out[0]).toBe(inside);
    expect(out[1]).toEqual({ x: 190, y: 100, w: 10, h: 30, id: "b" });
    expect(out).toHaveLength(2);
  });
});
