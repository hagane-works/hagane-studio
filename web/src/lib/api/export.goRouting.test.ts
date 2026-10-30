import { afterEach, describe, expect, it, vi } from "vitest";
import { exportApi } from "./export";

describe("advanced export is permanently Go-owned", () => {
  afterEach(() => vi.unstubAllGlobals());

  it("posts diff preview directly to /api/go", async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValue(
        new Response(
          JSON.stringify({ reports: {}, totals: {}, warnings: [] }),
          { status: 200, headers: { "content-type": "application/json" } },
        ),
      );
    vi.stubGlobal("fetch", fetchMock);
    await exportApi.diffPreview(
      182,
      new File(["zip"], "base.zip", { type: "application/zip" }),
    );
    expect(fetchMock.mock.calls[0][0]).toBe(
      "/api/go/projects/182/export/diff-preview",
    );
  });

  it("loads base installs directly from the Go project service", async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValue(
        new Response(JSON.stringify({ base_installs: [] }), {
          status: 200,
          headers: { "content-type": "application/json" },
        }),
      );
    vi.stubGlobal("fetch", fetchMock);

    await exportApi.baseInstalls(182);

    expect(fetchMock.mock.calls[0][0]).toBe(
      "/api/go/projects/182/export/base-installs",
    );
  });
});
