import { describe, expect, it, vi } from "vitest";

vi.mock("./_client", () => ({
  API_BASE: "/api",
  request: vi.fn(async (path: string) => ({ path })),
}));

import {
  scriptedSnippetsApi,
  scriptedLocalisationsApi,
  dynamicModifiersApi,
  cwtoolApi,
} from "./tooling";
import { request } from "./_client";

describe("tooling Go routing", () => {
  it("routes snippet + localisation + dynamic-modifier CRUD to Go", async () => {
    vi.mocked(request).mockClear();
    await scriptedSnippetsApi.list(182, "effect");
    await scriptedSnippetsApi.update(182, 3, {});
    await scriptedLocalisationsApi.list(182);
    await scriptedLocalisationsApi.update(182, 4, {});
    await dynamicModifiersApi.list(182);
    await dynamicModifiersApi.update(182, 1, {});
    expect(vi.mocked(request).mock.calls.map(([path]) => path)).toEqual([
      "/go/projects/182/scripted-snippets?kind=effect",
      "/go/projects/182/scripted-snippets/3",
      "/go/projects/182/scripted-localisations",
      "/go/projects/182/scripted-localisations/4",
      "/go/projects/182/dynamic-modifiers",
      "/go/projects/182/dynamic-modifiers/1",
    ]);
  });

  it("routes cwtool whitelist to Go (第 2 期切片 C)", async () => {
    vi.mocked(request).mockClear();
    await cwtoolApi.whitelist();
    expect(vi.mocked(request).mock.calls.map(([path]) => path)).toEqual([
      "/go/cwtool/whitelist",
    ]);
  });
});
