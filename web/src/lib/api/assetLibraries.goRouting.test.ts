import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

vi.mock("./_client", () => ({
  API_BASE: "/api",
  request: vi.fn(async (path: string) => ({ path })),
}));

import { request } from "./_client";
import { focusIconsApi, technologiesApi } from "./focus-tech";
import { projectsApi } from "./project";
import { workbenchIconsApi } from "./tooling";

describe("asset library and cover Go routing", () => {
  beforeEach(() => {
    vi.mocked(request).mockClear();
    vi.stubGlobal(
      "fetch",
      vi.fn(async () => ({
        ok: true,
        status: 200,
        statusText: "OK",
        json: async () => ({
          gfx_name: "GFX_focus_demo",
          saved: true,
          filename: "thumbnail.png",
        }),
      })),
    );
  });

  afterEach(() => vi.unstubAllGlobals());

  it("routes focus/tech manifests, library and reachability to Go", async () => {
    await focusIconsApi.list(182);
    await focusIconsApi.library(182, {
      source: "project",
      q: "demo",
      limit: 10,
      offset: 2,
    });
    await technologiesApi.listIcons(182);
    await technologiesApi.reachability(182, "GER");
    await technologiesApi.reachabilityPaths(182, "GER", "infantry_weapons");
    await workbenchIconsApi.techIconSize(182, "infantry weapons");
    expect(vi.mocked(request).mock.calls.map(([path]) => path)).toEqual([
      "/go/projects/182/focus-icons",
      "/go/projects/182/focus-icons/library?source=project&q=demo&limit=10&offset=2",
      "/go/projects/182/tech-icons",
      "/go/projects/182/technologies/reachability/GER",
      "/go/projects/182/technologies/reachability/GER/infantry_weapons/paths",
      "/go/projects/182/tech-icons/infantry%20weapons/size",
    ]);
  });

  it("uses fixed Go URLs for icon reads and named uploads", async () => {
    expect(focusIconsApi.getUrl(182, "GFX_focus_demo")).toBe(
      "/api/go/projects/182/focus-icons/GFX_focus_demo",
    );
    expect(workbenchIconsApi.techIconUrl(182, "infantry weapons")).toBe(
      "/api/go/projects/182/tech-icons/infantry%20weapons",
    );
    expect(workbenchIconsApi.decisionIconUrl(182, "GFX_decision demo")).toBe(
      "/api/go/projects/182/decision-icons/GFX_decision%20demo",
    );
    await focusIconsApi.uploadNamed(
      182,
      "GFX_focus_demo",
      new File(["x"], "x.png"),
    );
    expect(vi.mocked(fetch).mock.calls[0]?.[0]).toBe(
      "/api/go/projects/182/focus-icons/upload-named",
    );
  });

  it("uses fixed Go URLs for project cover upload/read/delete", async () => {
    await projectsApi.uploadCover(182, new File(["x"], "cover.png"));
    expect(vi.mocked(fetch).mock.calls[0]?.[0]).toBe(
      "/api/go/projects/182/cover",
    );
    expect(projectsApi.coverUrl(182, 7)).toBe("/api/go/projects/182/cover?v=7");
    await projectsApi.deleteCover(182);
    expect(vi.mocked(request).mock.calls.at(-1)?.[0]).toBe(
      "/go/projects/182/cover",
    );
  });
});
