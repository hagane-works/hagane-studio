import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { request } from "./_client";

function mockFetchBody(body: unknown) {
  vi.stubGlobal(
    "fetch",
    vi.fn(async () => ({
      ok: true,
      status: 200,
      json: async () => body,
    })) as unknown as typeof fetch,
  );
}

describe("request() backend-warnings dispatch guard", () => {
  let dispatched: CustomEvent[];
  let spy: ReturnType<typeof vi.spyOn>;

  beforeEach(() => {
    dispatched = [];
    spy = vi.spyOn(window, "dispatchEvent").mockImplementation((e: Event) => {
      if (e.type === "api-backend-warnings") dispatched.push(e as CustomEvent);
      return true;
    });
  });

  afterEach(() => {
    spy.mockRestore();
    vi.unstubAllGlobals();
  });

  it("does NOT dispatch for an OBJECT warnings array (CtdCheckResult shape)", async () => {
    mockFetchBody({
      ok: false,
      blocker_count: 1,
      warning_count: 1,
      blockers: [],
      warnings: [
        {
          code: "content_check",
          entity_ref: null,
          severity: "warning",
          message: "x",
          fix_hint: "",
        },
      ],
      issues: [],
    });

    await request("/projects/1/export/check", { method: "POST" });
    expect(dispatched).toHaveLength(0);
  });

  it("STILL dispatches for a real string[] warnings array (the intended path)", async () => {
    mockFetchBody({
      id: 1,
      warnings: ["event_id 'PRC222.1' collided · auto-suffixed to 'PRC222.2'."],
    });

    await request("/technologies", { method: "POST" });
    expect(dispatched).toHaveLength(1);
    expect((dispatched[0].detail as { warnings: string[] }).warnings).toEqual([
      "event_id 'PRC222.1' collided · auto-suffixed to 'PRC222.2'.",
    ]);
  });

  it("does NOT dispatch for a MIXED array (any non-string entry disqualifies)", async () => {
    mockFetchBody({ warnings: ["ok string", { code: "x" }] });
    await request("/whatever", { method: "POST" });
    expect(dispatched).toHaveLength(0);
  });
});
