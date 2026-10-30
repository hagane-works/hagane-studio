import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, fireEvent, render } from "@testing-library/react";
import type { GuiNode } from "@/lib/guiRenderApi";
import { GuiCanvas, assignIds } from "./GuiCanvas";

if (typeof window !== "undefined" && !window.PointerEvent) {
  class PointerEventStandIn extends MouseEvent {
    pointerId: number;
    constructor(type: string, init: PointerEventInit = {}) {
      super(type, init);
      this.pointerId = init.pointerId ?? 0;
    }
  }
  window.PointerEvent = PointerEventStandIn as unknown as typeof PointerEvent;
}

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

function draw(containerHits: boolean, selectedId: string | null = null) {
  const root = node({
    name: "panel",
    size: { width: 400, height: 300 },
    children: [
      node({
        name: "GER_ui_box",
        position: { x: 20, y: 30 },
        size: { width: 220, height: 120 },
        background: node({
          type: "background",
          name: "Background",
          sprite: "GFX_tiled_window_1b_border",
        }),
        children: [
          node({
            type: "instanttextboxtype",
            name: "GER_ui_label",
            text: "HI",
            position: { x: 8, y: 8 },
            maxwidth: 100,
            maxheight: 24,
          }),
        ],
      }),
      node({
        name: "GER_ui_group",
        position: { x: 260, y: 30 },
        size: { width: 100, height: 80 },
      }),
      node({
        name: "GER_ui_list_content",
        position: { x: 0, y: 200 },
        size: { width: 100, height: 80 },
      }),
    ],
  });
  assignIds(root);
  const onEditNode = vi.fn();
  const view = render(
    <GuiCanvas
      node={root}
      sprites={{}}
      texts={{ HI: { display: "Hello", kind: "loc" } }}
      fonts={{}}
      versions={{}}
      hidden={new Set()}
      isolateId={null}
      selectedId={selectedId}
      onSelect={() => {}}
      editMode
      containerHits={containerHits}
      onEditNode={onEditNode}
    />,
  );
  const hit = (name: string) =>
    [...view.container.querySelectorAll("[data-hit]")].find((el) =>
      el.getAttribute("title")?.endsWith(` ${name}`),
    );
  return {
    ...view,
    hit,
    onEditNode,
    ids: {
      box: root.children![0] as GuiNode & { _id: string },
      label: root.children![0].children![0] as GuiNode & { _id: string },
    },
  };
}

afterEach(cleanup);

describe("a project panel's containers", () => {
  it("are picked as themselves — box and group, not the box's background art, not a list's inner window", () => {
    const { hit } = draw(true);
    expect(hit("GER_ui_box")).toBeTruthy();
    expect(hit("GER_ui_group")).toBeTruthy();
    expect(hit("Background")).toBeUndefined();
    expect(hit("GER_ui_list_content")).toBeUndefined();

    expect((hit("GER_ui_box") as HTMLElement).style.cursor).toBe("move");
  });

  it("show their resize handle once picked", () => {
    const first = draw(true);
    const id = first.ids.box._id;
    cleanup();
    const { container } = draw(true, id);
    expect(container.querySelector("[data-resize-handle]")).toBeTruthy();
  });

  it("leave the base game's screens as they were: the background picked, not dragged", () => {
    const { hit } = draw(false);
    expect(hit("GER_ui_box")).toBeUndefined();
    expect(hit("Background")).toBeTruthy();
    expect(hit("GER_ui_group")).toBeUndefined();
  });

  it("tell a double-click on an element, to edit its words where they show", () => {
    const { hit, onEditNode, ids } = draw(true);
    fireEvent.doubleClick(hit("GER_ui_label") as HTMLElement);
    expect(onEditNode).toHaveBeenCalledWith(
      ids.label._id,
      expect.objectContaining({ name: "GER_ui_label" }),
    );
  });

  it("drags only past dragThreshold, when given", () => {
    const root = node({
      name: "screen",
      size: { width: 400, height: 300 },
      children: [
        node({
          type: "instanttextboxtype",
          name: "medal",
          text: "HI",
          position: { x: 50, y: 60 },
          maxwidth: 100,
          maxheight: 24,
        }),
      ],
    });
    assignIds(root);
    const onLayoutCommit = vi.fn();
    const onLayoutCancel = vi.fn();
    const view = render(
      <GuiCanvas
        node={root}
        sprites={{}}
        texts={{ HI: { display: "Hello", kind: "loc" } }}
        fonts={{}}
        versions={{}}
        hidden={new Set()}
        isolateId={null}
        selectedId={null}
        onSelect={() => {}}
        editMode
        dragThreshold={4}
        onLayoutPreview={() => {}}
        onLayoutCommit={onLayoutCommit}
        onLayoutCancel={onLayoutCancel}
      />,
    );
    const medal = [...view.container.querySelectorAll("[data-hit]")].find(
      (el) => el.getAttribute("title")?.endsWith(" medal"),
    ) as HTMLElement;
    fireEvent.pointerDown(medal, {
      pointerId: 1,
      button: 0,
      clientX: 100,
      clientY: 100,
    });
    fireEvent.pointerMove(medal, { pointerId: 1, clientX: 102, clientY: 101 });
    fireEvent.pointerUp(medal, { pointerId: 1, clientX: 102, clientY: 101 });
    expect(onLayoutCommit).not.toHaveBeenCalled();
    expect(onLayoutCancel).toHaveBeenCalled();
    fireEvent.pointerDown(medal, {
      pointerId: 2,
      button: 0,
      clientX: 100,
      clientY: 100,
    });
    fireEvent.pointerMove(medal, { pointerId: 2, clientX: 130, clientY: 110 });
    fireEvent.pointerUp(medal, { pointerId: 2, clientX: 130, clientY: 110 });
    expect(onLayoutCommit).toHaveBeenCalledWith(expect.any(String), {
      x: 80,
      y: 70,
    });
  });
});
