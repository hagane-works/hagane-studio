import { afterEach, describe, expect, it, vi } from "vitest";
import { ApiError, request } from "./_client";

describe("request() structured error payload", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("preserves FastAPI object detail and uses its message", async () => {
    const detail = {
      error: "ctd_blockers",
      message: "Fix 2 blockers before exporting.",
      blockers: [{ code: "focus_default", severity: "blocker", message: "x" }],
    };
    vi.stubGlobal(
      "fetch",
      vi.fn(async () => ({
        ok: false,
        status: 409,
        json: async () => ({ detail }),
      })) as unknown as typeof fetch,
    );

    await expect(
      request("/projects/1/export/preview-structure"),
    ).rejects.toMatchObject({
      name: "ApiError",
      status: 409,
      detail: detail.message,
      payload: detail,
    } satisfies Partial<ApiError>);
  });

  it("preserves native Go top-level code and authorization fields", async () => {
    const body = {
      code: "politics_delete_blocked",
      detail: "politics delete blocked",
      authorization: {
        authorized: false,
        impacts: [{ entity_kind: "character", entity_id: "leader_1" }],
      },
    };
    vi.stubGlobal(
      "fetch",
      vi.fn(async () => ({
        ok: false,
        status: 409,
        json: async () => body,
      })) as unknown as typeof fetch,
    );

    await expect(
      request("/go/projects/1/politics/delete-apply"),
    ).rejects.toMatchObject({
      name: "ApiError",
      status: 409,
      detail: body.detail,
      payload: body,
    } satisfies Partial<ApiError>);
  });
});
