import { describe, expect, it, vi } from "vitest";

vi.mock("./_client", () => ({
  API_BASE: "/api",
  request: vi.fn(async (path: string) => ({ path })),
}));

import { focusTreesApi, technologiesApi } from "./focus-tech";
import { request } from "./_client";

describe("focus trees Go routing", () => {
  it("routes tree + node CRUD to the Go service", async () => {
    vi.mocked(request).mockClear();
    await focusTreesApi.list(182);
    await focusTreesApi.get(182, 3);
    await focusTreesApi.create(182, {});
    await focusTreesApi.update(182, 3, {});
    await focusTreesApi.delete(182, 3);
    await focusTreesApi.getNode(182, 3, 11);
    await focusTreesApi.createNode(182, 3, {});
    await focusTreesApi.updateNode(182, 3, 11, {});
    await focusTreesApi.bulkUpdateNodePositions(182, 3, [
      { id: 11, x: 1, y: 2 },
    ]);
    await focusTreesApi.deleteNode(182, 3, 11);
    await focusTreesApi.forkNode(182, 3, 11, "GER_fork");
    expect(vi.mocked(request).mock.calls.map(([path]) => path)).toEqual([
      "/go/projects/182/focus-trees",
      "/go/projects/182/focus-trees/3",
      "/go/projects/182/focus-trees",
      "/go/projects/182/focus-trees/3",
      "/go/projects/182/focus-trees/3",
      "/go/projects/182/focus-trees/3/nodes/11",
      "/go/projects/182/focus-trees/3/nodes",
      "/go/projects/182/focus-trees/3/nodes/11",
      "/go/projects/182/focus-trees/3/node-positions",
      "/go/projects/182/focus-trees/3/nodes/11",
      "/go/projects/182/focus-trees/3/nodes/11/fork",
    ]);
  });

  it("routes preview + parse-preview to the Go service", async () => {
    vi.mocked(request).mockClear();
    await focusTreesApi.preview(182, 3);
    await focusTreesApi.parseNodePreview(182, 3, 11, "focus = {}");
    expect(vi.mocked(request).mock.calls.map(([path]) => path)).toEqual([
      "/go/projects/182/focus-trees/3/preview",
      "/go/projects/182/focus-trees/3/nodes/11/parse-preview",
    ]);

    expect(vi.mocked(request).mock.calls[1]?.[1]).toMatchObject({
      method: "POST",
      body: JSON.stringify({ script: "focus = {}" }),
    });
  });

  it("routes focus and technology asset reads to Go", () => {
    expect(focusTreesApi.nodeIconUrl(182, "GFX_focus_demo")).toBe(
      "/api/go/projects/182/focus-icons/GFX_focus_demo",
    );
    expect(technologiesApi.iconUrl(182, "infantry_weapons")).toBe(
      "/api/go/projects/182/tech-icons/infantry_weapons",
    );
  });
});
