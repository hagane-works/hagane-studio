import { beforeEach, describe, expect, it, vi } from "vitest";

const requestMock = vi.hoisted(() => vi.fn());

vi.mock("./_client", () => ({ request: requestMock }));

import { crossRefRepairsApi } from "./crossRefRepair";

const BASE_REQUEST = {
  entity_kind: "focus" as const,
  entity_id: 42,
  field: "completion_reward",
  target_id: "missing_idea",
  target_start_offset: 15,
  target_end_offset: 27,
  replacement_id: "known_idea",
  delete_assignment: false,
};

describe("crossRefRepairsApi", () => {
  beforeEach(() => requestMock.mockReset());

  it("posts preview evidence to the project-scoped endpoint", async () => {
    requestMock.mockResolvedValue({ applicable: true });
    await crossRefRepairsApi.preview(7, BASE_REQUEST);

    expect(requestMock).toHaveBeenCalledWith(
      "/go/projects/7/cross-ref-repairs/preview",
      { method: "POST", body: JSON.stringify(BASE_REQUEST) },
    );
  });

  it("sends the preview source hash back for stale-safe apply", async () => {
    requestMock.mockResolvedValue({ applied: true });
    const body = { ...BASE_REQUEST, source_sha256: "abc123" };
    await crossRefRepairsApi.apply(7, body);

    expect(requestMock).toHaveBeenCalledWith(
      "/go/projects/7/cross-ref-repairs/apply",
      expect.objectContaining({
        method: "POST",
        body: JSON.stringify(body),
        headers: { "X-Edit-Revisions": "{}" },
        onResponse: expect.any(Function),
      }),
    );
  });
});
