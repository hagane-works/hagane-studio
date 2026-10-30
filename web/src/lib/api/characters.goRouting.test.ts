import { describe, expect, it, vi } from "vitest";

vi.mock("./_client", () => ({
  API_BASE: "/api",
  request: vi.fn(async (path: string) => ({ path })),
}));

import { charactersApi } from "./characters";
import { request } from "./_client";

describe("characters Go routing", () => {
  it("routes CRUD, fork, preview and reverse parse to Go", async () => {
    vi.mocked(request).mockClear();

    await charactersApi.list(182);
    await charactersApi.get(182, 5);
    await charactersApi.create(182, { name: "Boris" });
    await charactersApi.update(182, 5, { skill: 3 });
    await charactersApi.delete(182, 5);
    await charactersApi.fork(182, 5, "GER_boris_copy");
    await charactersApi.preview(182, 5);
    await charactersApi.parsePreview(182, 5, "characters = {}");
    await charactersApi.resetPortrait(182, 5);

    expect(vi.mocked(request).mock.calls.map(([path]) => path)).toEqual([
      "/go/projects/182/characters",
      "/go/projects/182/characters/5",
      "/go/projects/182/characters",
      "/go/projects/182/characters/5",
      "/go/projects/182/characters/5",
      "/go/projects/182/characters/5/fork",
      "/go/projects/182/characters/5/preview",
      "/go/projects/182/characters/5/parse-preview",
      "/go/projects/182/characters/5/portrait",
    ]);
    expect(vi.mocked(request).mock.calls[7]?.[1]).toMatchObject({
      method: "POST",
      body: JSON.stringify({ script: "characters = {}" }),
    });
    expect(vi.mocked(request).mock.calls[8]?.[1]).toMatchObject({
      method: "DELETE",
    });
  });

  it("routes multi-file preview to Go without fallback", async () => {
    vi.mocked(request).mockClear();
    await charactersApi.previewFiles(182, 5);
    expect(vi.mocked(request)).toHaveBeenCalledWith(
      "/go/projects/182/characters/5/preview-files",
    );
  });
});
