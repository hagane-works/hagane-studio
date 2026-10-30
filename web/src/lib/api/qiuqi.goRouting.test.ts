import { beforeEach, describe, expect, it, vi } from "vitest";

const fetchMock = vi.fn();
vi.stubGlobal("fetch", fetchMock);

import { qiuqiApi } from "./tooling";

describe("QIUQI Go routing", () => {
  beforeEach(() => {
    fetchMock.mockReset();
    fetchMock.mockResolvedValue({
      ok: true,
      status: 200,
      json: async () => ({ version: 1, count: 0, translations: {} }),
    });
  });

  it("uses the fixed Go modifier catalog route", async () => {
    await qiuqiApi.modifiers();
    expect(fetchMock).toHaveBeenCalledWith(
      "/api/go/qiuqi/modifiers",
      expect.objectContaining({ credentials: "include" }),
    );
  });
});
