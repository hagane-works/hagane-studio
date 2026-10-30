import { describe, expect, it, vi } from "vitest";

vi.mock("./_client", () => ({
  API_BASE: "/api",
  request: vi.fn(async (path: string) => ({ path })),
}));

import { decisionsApi, eventsApi } from "./content";
import { request } from "./_client";

describe("events and decisions preview Go routing", () => {
  it("routes event namespace, event preview, and reverse parse to Go", async () => {
    vi.mocked(request).mockClear();

    await eventsApi.previewNamespace(182, 9);
    await eventsApi.preview(182, 31);
    await eventsApi.parsePreview(182, 31, "add_namespace = test");

    expect(vi.mocked(request).mock.calls.map(([path]) => path)).toEqual([
      "/go/projects/182/events/namespaces/9/preview",
      "/go/projects/182/events/31/preview",
      "/go/projects/182/events/31/parse-preview",
    ]);
    expect(vi.mocked(request).mock.calls[2]?.[1]).toMatchObject({
      method: "POST",
      body: JSON.stringify({ script: "add_namespace = test" }),
    });
  });

  it("routes decision category, decision preview, and reverse parse to Go", async () => {
    vi.mocked(request).mockClear();

    await decisionsApi.previewCategory(182, 11);
    await decisionsApi.preview(182, 41);
    await decisionsApi.parsePreview(182, 41, "war_plans = {}");

    expect(vi.mocked(request).mock.calls.map(([path]) => path)).toEqual([
      "/go/projects/182/decisions/categories/11/preview",
      "/go/projects/182/decisions/41/preview",
      "/go/projects/182/decisions/41/parse-preview",
    ]);
    expect(vi.mocked(request).mock.calls[2]?.[1]).toMatchObject({
      method: "POST",
      body: JSON.stringify({ script: "war_plans = {}" }),
    });
  });

  it("routes binary assets and event lint to Go", async () => {
    vi.mocked(request).mockClear();

    expect(eventsApi.pictureUrl(182, 31)).toBe(
      "/api/go/projects/182/events/31/picture",
    );
    expect(decisionsApi.categoryIconUrl(182, 11)).toBe(
      "/api/go/projects/182/decisions/categories/11/icon",
    );
    expect(decisionsApi.iconUrl(182, 41)).toBe(
      "/api/go/projects/182/decisions/41/icon",
    );

    await eventsApi.lint(182, 31);
    expect(vi.mocked(request)).toHaveBeenCalledWith(
      "/go/projects/182/events/31/lint",
    );
  });
});
