import { describe, expect, it, vi } from "vitest";

vi.mock("./_client", () => ({
  API_BASE: "/api",
  request: vi.fn(async (path: string) => ({ path })),
}));

import { wargoalsApi } from "./wargoals";
import { request } from "./_client";

describe("wargoals Go routing", () => {
  it("routes wargoal CRUD to the Go service", async () => {
    vi.mocked(request).mockClear();
    await wargoalsApi.list(182);
    await wargoalsApi.create(182, {
      wargoal_key: "x",
      war_name: "",
      safe_scalars: {},
      allowed: "",
      available: "",
      take_states: "",
      puppet: "",
      liberate: "",
      advanced_raw_script: "",
      source_file: "",
    });
    await wargoalsApi.update(182, 6, {
      wargoal_key: "x",
      war_name: "",
      safe_scalars: {},
      allowed: "",
      available: "",
      take_states: "",
      puppet: "",
      liberate: "",
      advanced_raw_script: "",
      source_file: "",
    });
    await wargoalsApi.delete(182, 6);
    expect(vi.mocked(request).mock.calls.map(([path]) => path)).toEqual([
      "/go/projects/182/wargoals",
      "/go/projects/182/wargoals",
      "/go/projects/182/wargoals/6",
      "/go/projects/182/wargoals/6",
    ]);
  });
});
