import { afterEach, expect, it, vi } from "vitest";
import { crossRefRepairsApi } from "@/lib/api/crossRefRepair";
import { focusTreesApi } from "@/lib/api/focus-tech";
const body = {
  entity_kind: "focus" as const,
  entity_id: 12,
  field: "completion_reward",
  target_id: "MISSING",
  target_start_offset: 0,
  target_end_offset: 7,
  replacement_id: "FOUND",
};
afterEach(() => vi.unstubAllGlobals());
it("uses the repair preview snapshot after a manual save and never fetches a replacement token", async () => {
  const fetcher = vi
    .fn()
    .mockResolvedValueOnce(
      new Response(
        JSON.stringify({
          source_sha256: "old-hash",
          edit_revisions: { "focus_tree:4401": "1" },
        }),
      ),
    )
    .mockResolvedValueOnce(
      new Response(JSON.stringify({ id: 12 }), {
        headers: { "X-Edit-Revisions": '{"focus_tree:4401":"2"}' },
      }),
    )
    .mockResolvedValueOnce(
      new Response(JSON.stringify({ detail: { code: "save_conflict" } }), {
        status: 409,
      }),
    );
  vi.stubGlobal("fetch", fetcher);
  const preview = await crossRefRepairsApi.preview(4400, body);
  await focusTreesApi.updateNode(4400, 4401, 12, {
    x: 9,
    tree_edit_revision: "1",
  });
  await expect(
    crossRefRepairsApi.apply(
      4400,
      { ...body, source_sha256: preview.source_sha256 },
      preview.edit_revisions,
    ),
  ).rejects.toMatchObject({ code: "save_conflict", status: 409 });
  expect(fetcher.mock.calls[2][1].headers["X-Edit-Revisions"]).toBe(
    '{"focus_tree:4401":"1"}',
  );
  expect(JSON.parse(fetcher.mock.calls[2][1].body)).toMatchObject({
    source_sha256: "old-hash",
  });
  expect(fetcher).toHaveBeenCalledTimes(3);
});
it("leaves legacy missing preview tokens empty and surfaces 428 without retry", async () => {
  const fetcher = vi
    .fn()
    .mockResolvedValue(
      new Response(
        JSON.stringify({ detail: { code: "save_precondition_required" } }),
        { status: 428 },
      ),
    );
  vi.stubGlobal("fetch", fetcher);
  await expect(
    crossRefRepairsApi.apply(4402, { ...body, source_sha256: "old-hash" }),
  ).rejects.toMatchObject({ code: "save_precondition_required", status: 428 });
  expect(fetcher.mock.calls[0][1].headers["X-Edit-Revisions"]).toBe("{}");
  expect(fetcher).toHaveBeenCalledTimes(1);
});
