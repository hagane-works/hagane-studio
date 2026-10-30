import { readFileSync } from "node:fs";
import { resolve } from "node:path";
import { describe, expect, it } from "vitest";
import type { NodeChange } from "@xyflow/react";

import {
  sameSelectedNodeIds,
  selectedNodeIdsAfterChanges,
  splitControlledSelectionChanges,
} from "./controlledNodeSelection";

describe("controlled React Flow selection", () => {
  it("has no second onNodeClick writer in the production graph", () => {
    const graph = readFileSync(
      resolve(
        process.cwd(),
        "src/components/workbench/focus-graph/FocusTreeGraph.tsx",
      ),
      "utf8",
    )
      .replace(/\/\*[\s\S]*?\*\//g, "")
      .replace(/(^|[^:])\/\/.*$/gm, "$1");

    expect(graph).not.toContain("onNodeClick=");
    expect(graph).toContain("applyNodeChanges(renderChanges, nds)");
    expect(graph).not.toContain("applyNodeChanges(changes, nds)");
  });

  it("routes select changes away from applyNodeChanges", () => {
    const changes: NodeChange[] = [
      { type: "select", id: "7", selected: true },
      { type: "position", id: "7", position: { x: 10, y: 20 }, dragging: true },
      { type: "dimensions", id: "8", dimensions: { width: 100, height: 50 } },
    ];
    const split = splitControlledSelectionChanges(changes);
    expect(split.selectionChanges).toEqual([
      { type: "select", id: "7", selected: true },
    ]);
    expect(split.renderChanges.map((change) => change.type)).toEqual([
      "position",
      "dimensions",
    ]);
  });

  it("derives single and multi-selection in Zustand without a click writer", () => {
    const single = selectedNodeIdsAfterChanges(
      [],
      [{ type: "select", id: "7", selected: true }],
    );
    const multi = selectedNodeIdsAfterChanges(single, [
      { type: "select", id: "8", selected: true },
    ]);
    const switched = selectedNodeIdsAfterChanges(multi, [
      { type: "select", id: "7", selected: false },
    ]);
    expect(single).toEqual([7]);
    expect(multi).toEqual([7, 8]);
    expect(switched).toEqual([8]);
  });

  it("recognizes redundant feedback regardless of array order", () => {
    expect(sameSelectedNodeIds([7, 8], [8, 7])).toBe(true);
    expect(sameSelectedNodeIds([7], [8])).toBe(false);
  });

  it("clears a stale edge when the page accepts a node selection", () => {
    const page = readFileSync(
      resolve(
        process.cwd(),
        "src/app/[locale]/project/[id]/focus-tree/page.tsx",
      ),
      "utf8",
    )
      .replace(/\/\*[\s\S]*?\*\//g, "")
      .replace(/(^|[^:])\/\/.*$/gm, "$1");

    expect(page).toContain("onSelectNodes={handleSelectNodes}");
    expect(page).toMatch(
      /setSelectedNodeIds\(nodeIds\);\s*if \(nodeIds\.length > 0\) selectEdge\(null\);/,
    );
  });
});
