import { describe, it, expect } from "vitest";
import type { Edge } from "@xyflow/react";
import {
  applyTechEdgeDisplayState,
  TECH_EDGE_DIM_OPACITY,
  TECH_EDGE_EMPHASIS_WIDTH,
} from "./techEdgeDisplay";

function baseEdges(): Edge[] {
  return [
    {
      id: "prereq-1-2",
      source: "1",
      target: "2",
      type: "techPrereqEdge",
      style: { stroke: "#f59e0b", strokeWidth: 2.5 },
      animated: false,
      data: { isMainChain: true },
    },
    {
      id: "prereq-2-3",
      source: "2",
      target: "3",
      type: "techPrereqEdge",
      style: { stroke: "#94a3b8", strokeWidth: 1.5, strokeOpacity: 0.55 },
      animated: false,
      data: { isMainChain: false },
    },
    {
      id: "prereq-4-5",
      source: "4",
      target: "5",
      type: "techPrereqEdge",
      style: { stroke: "#94a3b8", strokeWidth: 1.5 },
      animated: false,
      data: { isMainChain: false },
    },
  ];
}

describe("applyTechEdgeDisplayState", () => {
  it("无选中无悬停:全部回基线态(opacity 1 · 不流动)", () => {
    const out = applyTechEdgeDisplayState(baseEdges(), {
      selectedTechDbId: null,
      hoveredEdgeId: null,
    });
    for (const e of out) {
      expect(e.style?.opacity).toBe(1);
      expect(e.animated).toBe(false);
    }
  });

  it("选中节点 2:相连边点亮+流动+加粗,无关边淡到 0.2", () => {
    const out = applyTechEdgeDisplayState(baseEdges(), {
      selectedTechDbId: 2,
      hoveredEdgeId: null,
    });
    const e12 = out.find((e) => e.id === "prereq-1-2")!;
    const e23 = out.find((e) => e.id === "prereq-2-3")!;
    const e45 = out.find((e) => e.id === "prereq-4-5")!;

    for (const e of [e12, e23]) {
      expect(e.style?.opacity).toBe(1);
      expect(e.animated).toBe(true);
      expect(e.style?.strokeWidth).toBeGreaterThanOrEqual(
        TECH_EDGE_EMPHASIS_WIDTH,
      );
      expect(e.data?.highlighted).toBe(true);
    }
    expect(e45.style?.opacity).toBe(TECH_EDGE_DIM_OPACITY);
    expect(e45.animated).toBe(false);
    expect(e45.data?.dimmed).toBe(true);
  });

  it("悬停边:加粗+不淡化+流动,即使无选中", () => {
    const out = applyTechEdgeDisplayState(baseEdges(), {
      selectedTechDbId: null,
      hoveredEdgeId: "prereq-2-3",
    });
    const hovered = out.find((e) => e.id === "prereq-2-3")!;
    expect(hovered.style?.opacity).toBe(1);
    expect(hovered.style?.strokeWidth).toBeGreaterThanOrEqual(
      TECH_EDGE_EMPHASIS_WIDTH,
    );
    expect(hovered.animated).toBe(true);
  });

  it("选中态下悬停一条无关边:该边仍恢复全亮(hover 优先于淡化)", () => {
    const out = applyTechEdgeDisplayState(baseEdges(), {
      selectedTechDbId: 2,
      hoveredEdgeId: "prereq-4-5",
    });
    const e45 = out.find((e) => e.id === "prereq-4-5")!;
    expect(e45.style?.opacity).toBe(1);
  });

  it("不改基线 stroke 颜色(只叠加 opacity/width/animated)", () => {
    const out = applyTechEdgeDisplayState(baseEdges(), {
      selectedTechDbId: 2,
      hoveredEdgeId: null,
    });
    expect(out.find((e) => e.id === "prereq-1-2")!.style?.stroke).toBe(
      "#f59e0b",
    );
    expect(out.find((e) => e.id === "prereq-2-3")!.style?.stroke).toBe(
      "#94a3b8",
    );
  });
});
