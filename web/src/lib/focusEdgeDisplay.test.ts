import { describe, it, expect } from "vitest";
import type { Edge } from "@xyflow/react";
import { applyEdgeDisplayState } from "./focusEdgeDisplay";

function prereq(
  id: string,
  source: string,
  target: string,
  groupIndex = 0,
): Edge {
  return { id, source, target, type: "prereqEdge", data: { groupIndex } };
}
function mutex(id: string, source: string, target: string): Edge {
  return { id, source, target, type: "mutexEdge", data: {} };
}

describe("applyEdgeDisplayState", () => {
  it("无选中无悬停时全部中性、不淡化", () => {
    const edges = [prereq("e1", "1", "2"), mutex("e2", "3", "4")];
    const out = applyEdgeDisplayState(edges, {
      selectedNodeId: null,
      selectedEdgeId: null,
      hoveredEdgeId: null,
    });
    expect(out[0].style?.stroke).toBe("var(--color-muted-foreground)");
    expect(out[0].style?.opacity).toBe(1);
    expect(out[1].style?.stroke).toBe("var(--color-destructive)");
  });

  it("选中节点作为 target 时,入边按 groupIndex 上色", () => {
    const edges = [prereq("e1", "1", "9", 0), prereq("e2", "2", "9", 1)];
    const out = applyEdgeDisplayState(edges, {
      selectedNodeId: 9,
      selectedEdgeId: null,
      hoveredEdgeId: null,
    });
    expect(out[0].style?.stroke).toBe("var(--color-focus-edge-group-1)");
    expect(out[1].style?.stroke).toBe("var(--color-focus-edge-group-2)");
  });

  it("与选中节点无关的边被淡化", () => {
    const edges = [prereq("e1", "1", "9"), prereq("e2", "5", "6")];
    const out = applyEdgeDisplayState(edges, {
      selectedNodeId: 9,
      selectedEdgeId: null,
      hoveredEdgeId: null,
    });
    expect(out[1].style?.opacity).toBe(0.2);
    expect(out[1].data?.dimmed).toBe(true);
  });

  it("选中节点作为 source 时,出边用单一高亮色(不按 groupIndex)", () => {
    const edges = [prereq("e1", "9", "2", 3)];
    const out = applyEdgeDisplayState(edges, {
      selectedNodeId: 9,
      selectedEdgeId: null,
      hoveredEdgeId: null,
    });
    expect(out[0].style?.stroke).toBe("var(--color-foreground)");
  });

  it("与选中节点相连的互斥边保持红色、不淡化", () => {
    const edges = [mutex("e1", "9", "3")];
    const out = applyEdgeDisplayState(edges, {
      selectedNodeId: 9,
      selectedEdgeId: null,
      hoveredEdgeId: null,
    });
    expect(out[0].style?.stroke).toBe("var(--color-destructive)");
    expect(out[0].style?.opacity).toBe(1);
  });

  it("悬停的边加粗到 3.5 且不淡化", () => {
    const edges = [prereq("e1", "1", "2")];
    const out = applyEdgeDisplayState(edges, {
      selectedNodeId: 99,
      selectedEdgeId: null,
      hoveredEdgeId: "e1",
    });
    expect(out[0].style?.strokeWidth).toBe(3.5);
    expect(out[0].style?.opacity).toBe(1);
  });

  it("groupIndex 超过调色板长度时循环取色", () => {
    const edges = [prereq("e1", "1", "9", 4)];
    const out = applyEdgeDisplayState(edges, {
      selectedNodeId: 9,
      selectedEdgeId: null,
      hoveredEdgeId: null,
    });
    expect(out[0].style?.stroke).toBe("var(--color-focus-edge-group-1)");
  });

  it("悬停节点:相连边不淡化、无关边淡化", () => {
    const edges = [prereq("e1", "5", "2"), prereq("e2", "8", "9")];
    const out = applyEdgeDisplayState(edges, {
      selectedNodeId: null,
      selectedEdgeId: null,
      hoveredEdgeId: null,
      hoveredNodeId: 5,
    });
    expect(out[0].style?.opacity).toBe(1);
    expect(out[1].style?.opacity).toBe(0.2);
  });

  it("悬停节点优先于选中节点做高亮", () => {
    const edges = [prereq("e1", "5", "2"), prereq("e2", "8", "9")];

    const out = applyEdgeDisplayState(edges, {
      selectedNodeId: 8,
      selectedEdgeId: null,
      hoveredEdgeId: null,
      hoveredNodeId: 5,
    });
    expect(out[0].style?.opacity).toBe(1);
    expect(out[1].style?.opacity).toBe(0.2);
  });

  it("没有悬停节点时,回落到选中节点的高亮", () => {
    const edges = [prereq("e1", "5", "2")];
    const out = applyEdgeDisplayState(edges, {
      selectedNodeId: 5,
      selectedEdgeId: null,
      hoveredEdgeId: null,
      hoveredNodeId: null,
    });
    expect(out[0].style?.opacity).toBe(1);
  });
});
