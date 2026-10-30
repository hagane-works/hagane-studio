import { describe, expect, it, vi } from "vitest";

vi.mock("./_client", () => ({
  API_BASE: "/api",
  request: vi.fn(async (path: string) => ({ path })),
}));

import {
  projectsApi,
  projectUsageApi,
  basemodTemplatesApi,
  relationsApi,
} from "./project";
import { request } from "./_client";

describe("project Go routing", () => {
  it("routes the full project CRUD to the Go service", async () => {
    await projectsApi.list();
    await projectsApi.get(182);
    await projectsApi.create({ name: "x" });
    await projectsApi.update(182, { name: "y" });
    await projectsApi.delete(182);
    expect(vi.mocked(request).mock.calls.map(([path]) => path)).toEqual([
      "/go/projects",
      "/go/projects/182",
      "/go/projects",
      "/go/projects/182",
      "/go/projects/182",
    ]);
  });

  it("routes the project sub-resources to the Go service", async () => {
    vi.mocked(request).mockClear();
    await projectsApi.baseModVocabulary(182);
    await projectsApi.flipAll(182, { dry_run: true });
    await projectUsageApi.usage(182);
    await projectUsageApi.auditLog(182, 10, 20);
    expect(vi.mocked(request).mock.calls.map(([path]) => path)).toEqual([
      "/go/projects/182/base-mod-vocabulary",
      "/go/projects/182/flip-all",
      "/go/projects/182/usage",
      "/go/projects/182/audit-log?limit=10&offset=20",
    ]);
  });
});

describe("relations Go routing (第 2 期切片 E)", () => {
  it("routes entity/country relations to Go", async () => {
    vi.mocked(request).mockClear();
    await relationsApi.entity(182, "focus", 7);
    await relationsApi.country(182, "GER");
    expect(vi.mocked(request).mock.calls.map(([path]) => path)).toEqual([
      "/go/projects/182/relations/entity/focus/7",
      "/go/projects/182/relations/country/GER",
    ]);
  });
});

describe("basemod templates Go routing (第 2 期切片 C)", () => {
  it("routes the shelf listing to Go", async () => {
    vi.mocked(request).mockClear();
    await basemodTemplatesApi.list();
    expect(vi.mocked(request).mock.calls.map(([path]) => path)).toEqual([
      "/go/basemod/templates",
    ]);
  });
});
