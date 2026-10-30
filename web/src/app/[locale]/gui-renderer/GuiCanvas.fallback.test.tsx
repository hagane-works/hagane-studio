import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, fireEvent, render } from "@testing-library/react";
import type { GuiNode } from "@/lib/guiRenderApi";
import {
  GUI_UNHANDLED_ACTIVATE_EVENT,
  GuiCanvas,
  assignIds,
} from "./GuiCanvas";

function windowNode(): GuiNode {
  const button = {
    type: "buttontype",
    name: "go_button",
    position: { x: 0, y: 0 },
    size: { width: 100, height: 30 },
    orientation: null,
    origo: null,
    scale: null,
    sprite: null,
    frame: null,
    text: "GO",
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
  } as GuiNode;
  const root = {
    ...button,
    type: "containerwindowtype",
    name: "window",
    size: { width: 120, height: 50 },
    text: null,
    children: [button],
  } as GuiNode;
  assignIds(root);
  return root;
}

function canvas(
  node: GuiNode,
  onActivate?: (pressed: GuiNode) => boolean | void,
) {
  return (
    <GuiCanvas
      node={node}
      sprites={{}}
      texts={{ GO: { display: "Go", kind: "loc" } }}
      fonts={{}}
      versions={{}}
      hidden={new Set()}
      isolateId={null}
      selectedId={null}
      onSelect={() => {}}
      editMode={false}
      onActivate={onActivate}
    />
  );
}

afterEach(cleanup);

describe("view-mode fallback buttons", () => {
  it("announces a standalone engine button that has no preview action", () => {
    const announced = vi.fn();
    window.addEventListener(GUI_UNHANDLED_ACTIVATE_EVENT, announced, {
      once: true,
    });
    const view = render(canvas(windowNode()));

    fireEvent.click(
      view.container.querySelector('[data-node-name="go_button"]')!,
    );

    expect(announced).toHaveBeenCalledTimes(1);
  });

  it("lets a semantic wrapper own a template's decorative button", () => {
    const parent = vi.fn();
    const announced = vi.fn();
    window.addEventListener(GUI_UNHANDLED_ACTIVATE_EVENT, announced, {
      once: true,
    });
    const view = render(
      <button type="button" onClick={parent}>
        {canvas(windowNode())}
      </button>,
    );

    fireEvent.click(
      view.container.querySelector('[data-node-name="go_button"]')!,
    );

    expect(parent).toHaveBeenCalledTimes(1);
    expect(announced).not.toHaveBeenCalled();
    window.removeEventListener(GUI_UNHANDLED_ACTIVATE_EVENT, announced);
  });

  it("announces a button rejected by a partial specialised handler", () => {
    const announced = vi.fn();
    const onActivate = vi.fn(() => false);
    window.addEventListener(GUI_UNHANDLED_ACTIVATE_EVENT, announced, {
      once: true,
    });
    const view = render(canvas(windowNode(), onActivate));

    fireEvent.click(
      view.container.querySelector('[data-node-name="go_button"]')!,
    );

    expect(onActivate).toHaveBeenCalledTimes(1);
    expect(announced).toHaveBeenCalledTimes(1);
  });
});
