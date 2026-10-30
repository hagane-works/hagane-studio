import { cleanup, fireEvent, render } from "@testing-library/react";
import { ReactFlow, Position, type Node } from "@xyflow/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { FocusGraphNodeData } from "@/lib/focusTreeGraph";
import { FocusGraphNode } from "./FocusGraphNode";

vi.mock("use-intl", () => ({ useTranslations: () => (key: string) => key }));

const nodeTypes = { focusNode: FocusGraphNode };

function node(id: string, x: number): Node<FocusGraphNodeData> {
  return {
    id,
    type: "focusNode",
    position: { x, y: 100 },
    initialWidth: 110,
    initialHeight: 130,
    handles: [
      {
        type: "source",
        id: "mutex-left",
        position: Position.Left,
        x: 0,
        y: 65,
      },
      {
        type: "target",
        id: "mutex-right",
        position: Position.Right,
        x: 110,
        y: 65,
      },
    ],
    data: {
      dbId: Number(id),
      focusId: `focus_${id}`,
      label: "Focus",
      icon: "",
      iconUrl: null,
      vanillaIconUrl: null,
      baseModIconUrls: [],
      cost: 10,
      isRelative: false,
      hasAllowBranch: false,
      hasLoadFocusTreeRules: false,
    },
  };
}

beforeEach(() => {
  vi.stubGlobal(
    "ResizeObserver",
    class {
      observe() {}
      unobserve() {}
      disconnect() {}
    },
  );

  Object.defineProperty(document, "elementFromPoint", {
    configurable: true,
    value: () => null,
  });
});
afterEach(() => {
  cleanup();
  vi.unstubAllGlobals();
});

describe("focus mutual exclusion handles", () => {
  it.each(["left-to-right", "right-to-left"])(
    "connects %s with React Flow's strict handle validation",
    (direction) => {
      const onConnect = vi.fn();
      const { container } = render(
        <ReactFlow
          nodes={[node("1", 100), node("2", 400)]}
          nodeTypes={nodeTypes}
          onConnect={onConnect}
          width={800}
          height={600}
        />,
      );
      const facingLeft = container.querySelectorAll(
        '[data-nodeid="1"][data-handlepos="right"]',
      );
      const facingRight = container.querySelectorAll(
        '[data-nodeid="2"][data-handlepos="left"]',
      );

      expect(facingLeft).toHaveLength(1);
      expect(facingRight).toHaveLength(1);
      const [start, end] =
        direction === "left-to-right"
          ? [facingLeft[0], facingRight[0]]
          : [facingRight[0], facingLeft[0]];
      const startX = direction === "left-to-right" ? 210 : 400;
      const endX = direction === "left-to-right" ? 400 : 210;
      Object.defineProperty(document, "elementFromPoint", {
        configurable: true,
        value: () => end,
      });
      fireEvent.mouseDown(start, { button: 0, clientX: startX, clientY: 165 });
      fireEvent.mouseMove(document, { clientX: endX, clientY: 165 });
      fireEvent.mouseUp(document, { clientX: endX, clientY: 165 });
      expect(onConnect).toHaveBeenCalledExactlyOnceWith({
        source: "2",
        sourceHandle: "mutex-left",
        target: "1",
        targetHandle: "mutex-right",
      });
    },
  );

  it.each(["left", "right"] as const)(
    "keeps drag-to-empty and click-to-create on the %s side",
    (side) => {
      const onConnect = vi.fn();
      const onConnectStart = vi.fn();
      const onConnectEnd = vi.fn();
      const onAddMutexChild = vi.fn();
      const focus = node("1", 100);
      focus.data.onAddMutexChild = onAddMutexChild;
      const { container } = render(
        <ReactFlow
          nodes={[focus]}
          nodeTypes={nodeTypes}
          width={800}
          height={600}
          onConnect={onConnect}
          onConnectStart={onConnectStart}
          onConnectEnd={onConnectEnd}
        />,
      );
      const handle = container.querySelector(
        `[data-nodeid="1"][data-handlepos="${side}"]`,
      )!;
      const startX = side === "left" ? 100 : 210;
      fireEvent.mouseDown(handle, { button: 0, clientX: startX, clientY: 165 });
      fireEvent.mouseMove(document, { clientX: 650, clientY: 400 });
      fireEvent.mouseUp(document, { clientX: 650, clientY: 400 });
      expect(onConnectStart).toHaveBeenCalledWith(expect.anything(), {
        nodeId: "1",
        handleId: `mutex-${side}`,
        handleType: side === "left" ? "source" : "target",
      });
      expect(onConnectEnd).toHaveBeenCalledWith(
        expect.anything(),
        expect.objectContaining({
          toNode: null,
          fromNode: expect.objectContaining({ id: "1" }),
        }),
      );
      expect(onConnect).not.toHaveBeenCalled();
      expect(onAddMutexChild).not.toHaveBeenCalled();

      fireEvent.mouseDown(handle, { button: 0, clientX: startX, clientY: 165 });
      fireEvent.mouseUp(document, { clientX: startX, clientY: 165 });
      expect(onAddMutexChild).toHaveBeenCalledExactlyOnceWith(side);
    },
  );
});
