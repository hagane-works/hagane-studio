import { describe, expect, it, vi } from "vitest";

vi.mock("./_client", () => ({
  API_BASE: "/api",
  request: vi.fn(async (path: string) => ({ path })),
}));

import { ideasApi } from "./content";
import { request } from "./_client";

describe("ideas preview Go routing", () => {
  it("routes complete + single preview and reverse parse to Go", async () => {
    vi.mocked(request).mockClear();

    await ideasApi.preview(182);
    await ideasApi.preview(182, 31);
    await ideasApi.parsePreview(182, 31, "ideas = {}");

    expect(vi.mocked(request).mock.calls.map(([path]) => path)).toEqual([
      "/go/projects/182/ideas/preview",
      "/go/projects/182/ideas/preview?idea_id=31",
      "/go/projects/182/ideas/31/parse-preview",
    ]);
    expect(vi.mocked(request).mock.calls[2]?.[1]).toMatchObject({
      method: "POST",
      body: JSON.stringify({ script: "ideas = {}" }),
    });
  });

  it("routes the picture stream to the Go object-store path", () => {
    expect(ideasApi.pictureUrl(182, 31)).toBe(
      "/api/go/projects/182/ideas/31/picture",
    );
  });
});
