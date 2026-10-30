import { describe, expect, it, vi } from "vitest";

vi.mock("./_client", () => ({
  API_BASE: "/api",
  request: vi.fn(async (path: string) => ({ path })),
}));

import { technologiesApi } from "./focus-tech";
import { request } from "./_client";

describe("technology Go preview routing", () => {
  it("routes line/single preview and reverse parse to Go", async () => {
    vi.mocked(request).mockClear();
    await technologiesApi.preview(182, "infantry");
    await technologiesApi.preview(182, "infantry", 11);
    await technologiesApi.parsePreview(182, 11, "technologies = {}");

    expect(vi.mocked(request).mock.calls.map(([path]) => path)).toEqual([
      "/go/projects/182/technologies/preview/infantry",
      "/go/projects/182/technologies/preview/infantry?tech_id=11",
      "/go/projects/182/technologies/11/parse-preview",
    ]);
  });
});
