import { afterEach, describe, expect, it, vi } from "vitest";
import { twoStageImportApi } from "@/lib/api/import";
import { focusTreesApi } from "@/lib/api/focus-tech";
import { useLocaleStore } from "@/lib/store/localeStore";
const options = {
  characters: false,
  focus: true,
  tech: false,
  ideas: false,
  events: false,
  decisions: false,
};
const json = (body: unknown, headers = {}) =>
  new Response(JSON.stringify(body), { headers });
afterEach(() => {
  vi.useRealTimers();
  vi.unstubAllGlobals();
});
describe("import commit revision ownership", () => {
  it("keeps the tree queue until the job commits and then advances the queued local write", async () => {
    vi.useFakeTimers();
    const fetcher = vi
      .fn()
      .mockResolvedValueOnce(json({ status: "queued" }))
      .mockResolvedValueOnce(json({ status: "running" }))
      .mockResolvedValueOnce(
        json({
          status: "complete",
          result: { edit_revisions: { "focus_tree:901": "8" } },
        }),
      )
      .mockResolvedValueOnce(
        json({ id: 1 }, { "X-Edit-Revisions": '{"focus_tree:901":"9"}' }),
      );
    vi.stubGlobal("fetch", fetcher);
    const pending = twoStageImportApi.scanAndParseStaged(
      901,
      "fixture",
      options,
      "overwrite",
      undefined,
      undefined,
      { "focus_tree:901": "7" },
    );
    const write = focusTreesApi.updateNode(901, 901, 1, { x: 4 }, "7", true);
    await vi.advanceTimersByTimeAsync(350);
    expect(fetcher).toHaveBeenCalledTimes(2);
    await vi.advanceTimersByTimeAsync(350);
    await Promise.all([pending, write]);
    expect(fetcher.mock.calls[0][1].headers["X-Edit-Revisions"]).toBe(
      '{"focus_tree:901":"7"}',
    );
    expect(fetcher.mock.calls[3][1].headers["X-Edit-Revisions"]).toBe(
      '{"focus_tree:901":"8"}',
    );
  });
  it("preserves a job's conflict identity and never restarts with a fresh token", async () => {
    vi.useFakeTimers();
    useLocaleStore.setState({ locale: "en" });
    const fetcher = vi
      .fn()
      .mockResolvedValueOnce(json({ status: "queued" }))
      .mockResolvedValueOnce(
        json({
          status: "failed",
          error_code: "save_conflict",
          http_status: 409,
        }),
      );
    vi.stubGlobal("fetch", fetcher);
    const pending = twoStageImportApi.scanAndParseStaged(
      902,
      "fixture",
      options,
      "overwrite",
      undefined,
      undefined,
      { "focus_tree:902": "1" },
    );
    const assertion = expect(pending).rejects.toMatchObject({
      code: "save_conflict",
      status: 409,
    });
    await vi.advanceTimersByTimeAsync(350);
    await assertion;
    expect(fetcher).toHaveBeenCalledTimes(2);
  });
});
