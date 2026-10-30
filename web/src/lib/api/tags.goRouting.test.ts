import { beforeEach, describe, expect, it, vi } from "vitest";

const fetchMock = vi.fn();
vi.stubGlobal("fetch", fetchMock);

import { tagsApi } from "./project";

describe("country tag Go routing", () => {
  beforeEach(() => {
    fetchMock.mockReset();
    fetchMock.mockResolvedValue({
      ok: true,
      status: 200,
      json: async () => ({ tags: [], total: 0 }),
    });
  });

  it("uses the fixed Go catalog route", async () => {
    await tagsApi.search("United Kingdom");
    expect(fetchMock).toHaveBeenCalledWith(
      "/api/go/tags?q=United%20Kingdom",
      expect.objectContaining({ credentials: "include" }),
    );
  });
});
