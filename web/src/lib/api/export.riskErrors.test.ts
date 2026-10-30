import { afterEach, describe, expect, it, vi } from "vitest";
import { ApiError } from "./_client";
import { exportApi } from "./export";

const BLOCKER = {
  code: "final_artifact_quality",
  severity: "blocker",
  entity_ref: null,
};

function structuredConflict(): Response {
  return new Response(
    JSON.stringify({
      detail: { message: "internal prose", blockers: [BLOCKER] },
    }),
    { status: 409, headers: { "content-type": "application/json" } },
  );
}

describe("advanced export structured risk errors", () => {
  afterEach(() => vi.unstubAllGlobals());

  it("preserves append blockers for the player-facing confirmation", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(structuredConflict()));
    const request = exportApi.appendToExisting(
      7,
      new File(["zip"], "base.zip", { type: "application/zip" }),
    );
    await expect(request).rejects.toMatchObject({
      name: "ApiError",
      status: 409,
      payload: { blockers: [BLOCKER] },
    } satisfies Partial<ApiError>);
  });

  it("preserves diff blockers for the player-facing confirmation", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(structuredConflict()));
    const request = exportApi.diffExportZip(
      7,
      new File(["zip"], "base.zip", { type: "application/zip" }),
      {},
      {},
    );
    await expect(request).rejects.toMatchObject({
      name: "ApiError",
      status: 409,
      payload: { blockers: [BLOCKER] },
    } satisfies Partial<ApiError>);
  });
});
