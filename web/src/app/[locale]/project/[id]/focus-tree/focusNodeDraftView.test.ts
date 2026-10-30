import { readFileSync } from "node:fs";
import { resolve } from "node:path";
import { describe, expect, it } from "vitest";

import {
  buildDraftPreviewEntities,
  findPersistedSelectedEntity,
} from "./focusNodeDraftView";

type TestNode = {
  id: number;
  x: number;
  y: number;
  name: string;
  completionReward: string;
};

describe("focus-node state ownership", () => {
  it("wires persisted data to the editor and limits draft projection to the canvas", () => {
    const page = readFileSync(
      resolve(
        process.cwd(),
        "src/app/[locale]/project/[id]/focus-tree/page.tsx",
      ),
      "utf8",
    )
      .replace(/\/\*[\s\S]*?\*\//g, "")
      .replace(/(^|[^:])\/\/.*$/gm, "$1");

    expect(page).toContain("selectedNode={selectedNode}");
    expect(page).toContain("allNodes={nodes}");
    expect(page).toContain("nodes={visualNodes}");
    expect(page).not.toContain("selectedNodeVisual");
  });

  it("keeps a same-id external Apply visible to the editor/readback path", () => {
    const applied: TestNode = {
      id: 7,
      x: 4,
      y: 2,
      name: "Persisted name",
      completionReward: "add_stability = 0.1",
    };
    const staleDraft: TestNode = {
      ...applied,
      name: "Old form value",
      completionReward: "",
    };

    const persistedSelection = findPersistedSelectedEntity([applied], 7);
    const canvasProjection = buildDraftPreviewEntities(
      [applied],
      7,
      staleDraft,
    );

    expect(persistedSelection).toBe(applied);
    expect(persistedSelection?.completionReward).toBe("add_stability = 0.1");
    expect(canvasProjection[0].completionReward).toBe("");
  });

  it("preserves an unsaved draft for canvas preview without mutating store data", () => {
    const persisted: TestNode = {
      id: 7,
      x: 8,
      y: 5,
      name: "Server name",
      completionReward: "",
    };
    const dirtyDraft: TestNode = {
      ...persisted,
      x: 1,
      y: 1,
      name: "Unsaved author edit",
    };

    const projected = buildDraftPreviewEntities([persisted], 7, dirtyDraft);

    expect(projected[0]).toMatchObject({
      name: "Unsaved author edit",
      x: 8,
      y: 5,
    });
    expect(persisted).toMatchObject({ name: "Server name", x: 8, y: 5 });
  });

  it("does not leak a previous node's draft across a selection switch", () => {
    const nodes: TestNode[] = [
      { id: 7, x: 0, y: 0, name: "A", completionReward: "" },
      { id: 8, x: 1, y: 0, name: "B", completionReward: "" },
    ];
    const previousDraft = { ...nodes[0], name: "Dirty A" };

    expect(buildDraftPreviewEntities(nodes, 8, previousDraft)).toBe(nodes);
    expect(findPersistedSelectedEntity(nodes, 8)).toBe(nodes[1]);
  });
});
