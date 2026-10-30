import { afterEach, describe, expect, it, vi } from "vitest";

afterEach(async () => {
  const { guiRenderApi } = await import("./guiRenderApi");
  guiRenderApi.setProjectId(null);
  vi.unstubAllGlobals();
});

describe("saveOverlay", () => {
  it("keeps the HTTP status on a refused save, so a revision conflict can reload", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue({
        ok: false,
        status: 409,
        json: async () => ({
          detail: "GUI overlay revision conflict; re-read before writing",
        }),
      }),
    );
    const { guiRenderApi } = await import("./guiRenderApi");
    guiRenderApi.setProjectId(7);
    await expect(
      guiRenderApi.saveOverlay({
        file: "demo.gui",
        edits: [],
        scaffold_hash: "a",
        expected_revision: "0",
        expected_scaffold_hash: "a",
      }),
    ).rejects.toMatchObject({
      status: 409,
      message: "GUI overlay revision conflict; re-read before writing",
    });
  });
});
