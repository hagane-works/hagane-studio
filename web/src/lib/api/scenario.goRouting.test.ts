import { describe, expect, it, vi } from "vitest";

vi.mock("./_client", () => ({
  API_BASE: "/api",
  request: vi.fn(async (path: string) => ({ path })),
}));

import {
  bookmarksApi,
  autonomyStatesApi,
  rawFilesApi,
  statesApi,
  superEventsApi,
} from "./scenario";
import { request } from "./_client";

describe("bookmarks Go routing", () => {
  it("routes bookmark CRUD to the Go service", async () => {
    vi.mocked(request).mockClear();
    await bookmarksApi.list(182);
    await bookmarksApi.get(182, 5);
    await bookmarksApi.create(182, { bookmark_id: "x" });
    await bookmarksApi.update(182, 5, {});
    await bookmarksApi.remove(182, 5);
    expect(vi.mocked(request).mock.calls.map(([path]) => path)).toEqual([
      "/go/projects/182/bookmarks",
      "/go/projects/182/bookmarks/5",
      "/go/projects/182/bookmarks",
      "/go/projects/182/bookmarks/5",
      "/go/projects/182/bookmarks/5",
    ]);
  });

  it("exposes bookmark pictures from the Go object-store path", () => {
    expect(bookmarksApi.pictureUrl(182, 5)).toBe(
      "/api/go/projects/182/bookmarks/5/picture",
    );
  });
});

describe("super-events Go routing", () => {
  it("routes list + single GET + CRUD to the Go service", async () => {
    vi.mocked(request).mockClear();
    await superEventsApi.list(182);
    await superEventsApi.getSuperEvent(182, 21);
    await superEventsApi.create(182, { number: 1001 });
    await superEventsApi.update(182, 21, {});
    await superEventsApi.remove(182, 21);
    expect(vi.mocked(request).mock.calls.map(([path]) => path)).toEqual([
      "/go/projects/182/super-events",
      "/go/projects/182/super-events/21",
      "/go/projects/182/super-events",
      "/go/projects/182/super-events/21",
      "/go/projects/182/super-events/21",
    ]);

    expect(vi.mocked(request).mock.calls[1]?.[1]).toBeUndefined();
  });
});

describe("autonomy-states Go routing", () => {
  it("routes autonomy state CRUD to the Go service", async () => {
    vi.mocked(request).mockClear();
    await autonomyStatesApi.list(182);
    await autonomyStatesApi.update(182, 2, {});
    expect(vi.mocked(request).mock.calls.map(([path]) => path)).toEqual([
      "/go/projects/182/autonomy-states",
      "/go/projects/182/autonomy-states/2",
    ]);
  });
});

describe("states static-data Go routing", () => {
  it("routes geometry and bundled vanilla seed to projectsvc", async () => {
    vi.mocked(request).mockClear();
    await statesApi.geometry(182);
    await statesApi.seedFromVanilla(182, { state_ids: [1, 2] });
    expect(vi.mocked(request).mock.calls.map(([path]) => path)).toEqual([
      "/go/projects/182/states-geometry",
      "/go/projects/182/states/seed-from-vanilla",
    ]);
  });
});

describe("states delete Go routing", () => {
  it("routes single delete and bulk-delete to projectsvc", async () => {
    vi.mocked(request).mockClear();
    await statesApi.remove(182, 9);
    await statesApi.bulkDelete(182, { state_ids: [9, 10] });
    expect(vi.mocked(request).mock.calls.map(([path]) => path)).toEqual([
      "/go/projects/182/states/9",
      "/go/projects/182/states/bulk-delete",
    ]);
    expect(vi.mocked(request).mock.calls[0]?.[1]?.method).toBe("DELETE");
    expect(vi.mocked(request).mock.calls[1]?.[1]?.method).toBe("POST");
    expect(vi.mocked(request).mock.calls[1]?.[1]?.body).toBe(
      JSON.stringify({ state_ids: [9, 10] }),
    );
  });
});

describe("raw files Go routing (第 4 期切片 G)", () => {
  it("routes raw-files CRUD + templates to Go", async () => {
    vi.mocked(request).mockClear();
    await rawFilesApi.list(182);
    await rawFilesApi.listTemplates(182);
    await rawFilesApi.create(182, { dest_path: "common/x.txt", content: "a" });
    await rawFilesApi.update(182, 7, { content: "b" });
    await rawFilesApi.remove(182, 7);
    expect(vi.mocked(request).mock.calls.map(([path]) => path)).toEqual([
      "/go/projects/182/raw-files",
      "/go/projects/182/raw-files/templates",
      "/go/projects/182/raw-files",
      "/go/projects/182/raw-files/7",
      "/go/projects/182/raw-files/7",
    ]);
  });
});
