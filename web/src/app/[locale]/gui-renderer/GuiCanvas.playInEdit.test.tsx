import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, fireEvent, render } from "@testing-library/react";
import type { GuiNode } from "@/lib/guiRenderApi";
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

function draw(playInEdit?: (n: GuiNode) => boolean) {
  const root = node({
    type: "containerwindowtype",
    name: "window",
    size: { width: 300, height: 100 },
    children: [
      node({ name: "go_button", text: "GO", size: { width: 200, height: 40 } }),
      node({
        type: "instanttextboxtype",
        name: "label",
        text: "PICK",
        size: { width: 200, height: 40 },
      }),
    ],
  });
  assignIds(root);
  const onActivate = vi.fn();
  const onSelect = vi.fn();
  const view = render(
    <GuiCanvas
      node={root}
      sprites={{}}
      texts={{
        GO: { display: "Go", kind: "loc" },
        PICK: { display: "Select a National Focus", kind: "loc" },
      }}
      fonts={{}}
      versions={{}}
      hidden={new Set()}
      isolateId={null}
      selectedId={null}
      onSelect={onSelect}
      editMode
      onActivate={onActivate}
      playInEdit={playInEdit}

      playInEditHint="Right-click to change its picture and words"
    />,
  );
  return { ...view, onActivate, onSelect };
}

const press = (c: HTMLElement) =>
  c.querySelector(
    '[title$="go_button · Right-click to change its picture and words"]',
  ) as HTMLElement | null;

afterEach(() => {
  cleanup();
  vi.restoreAllMocks();
});

describe("edit mode keeps the screen-switching buttons working", () => {
  it("a click presses the button, label and all, and picks nothing", () => {
    const { container, onActivate, onSelect } = draw(
      (n) => n.name === "go_button",
    );
    const over = press(container);
    expect(over).not.toBeNull();
    fireEvent.click(over!);
    expect(onActivate).toHaveBeenCalledTimes(1);
    expect(onActivate.mock.calls[0][0].name).toBe("go_button");
    expect(onSelect).not.toHaveBeenCalled();
  });

  it("a right-click picks what is drawn at that spot: the label over the button, else the button", () => {
    const { container, onActivate, onSelect } = draw(
      (n) => n.name === "go_button",
    );
    const label = container.querySelector(
      '[title^="instanttextboxtype label"]',
    ) as HTMLElement;
    const elementFromPoint = vi.fn<(x: number, y: number) => Element | null>(
      () => label,
    );
    Object.defineProperty(document, "elementFromPoint", {
      value: elementFromPoint,
      configurable: true,
    });
    fireEvent.contextMenu(press(container)!);
    expect(onSelect.mock.calls[0][1].name).toBe("label");

    elementFromPoint.mockReturnValue(null);
    fireEvent.contextMenu(press(container)!);
    expect(onSelect.mock.calls[1][1].name).toBe("go_button");
    expect(onActivate).not.toHaveBeenCalled();
  });

  it("without the rule every element is only picked, as before", () => {
    const { container, onActivate, onSelect } = draw();
    expect(press(container)).toBeNull();
    fireEvent.click(
      container.querySelector('[title^="buttontype go_button"]')!,
    );
    expect(onActivate).not.toHaveBeenCalled();
    fireEvent.pointerDown(
      container.querySelector('[title^="buttontype go_button"]')!,
    );
    expect(onSelect.mock.calls[0][1].name).toBe("go_button");
  });
});
