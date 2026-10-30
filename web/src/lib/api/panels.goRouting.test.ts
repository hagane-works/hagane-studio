import { describe, expect, it, vi } from "vitest";

vi.mock("./_client", () => ({
  request: vi.fn(async (path: string) => ({ path })),
}));

import { request } from "./_client";
import { uiPanelsApi } from "./panels";

describe("UI panels Go routing", () => {
  it("routes CRUD and export-backed file preview to projectsvc", async () => {
    vi.mocked(request).mockClear();
    await uiPanelsApi.list(182);
    await uiPanelsApi.get(182, 5);
    await uiPanelsApi.create(182, { token: "new_panel" });
    await uiPanelsApi.update(182, 5, { name_en: "Power" });
    await uiPanelsApi.delete(182, 5);
    await uiPanelsApi.files(182, 5);

    expect(vi.mocked(request).mock.calls.map(([path]) => path)).toEqual([
      "/go/projects/182/ui-panels",
      "/go/projects/182/ui-panels/5",
      "/go/projects/182/ui-panels",
      "/go/projects/182/ui-panels/5",
      "/go/projects/182/ui-panels/5",
      "/go/projects/182/ui-panels/5/files",
    ]);
  });
});
