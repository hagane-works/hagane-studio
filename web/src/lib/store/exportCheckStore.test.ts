import { beforeEach, describe, expect, it, vi } from "vitest";
import {
  checkStaleReason,
  settleExportCheckWithinTimeout,
  useExportCheckStore,
  type ExportCheckRun,
} from "@/lib/store/exportCheckStore";

const RAN_AT = Date.parse("2026-08-07T10:00:00Z");

function run(overrides: Partial<ExportCheckRun> = {}): ExportCheckRun {
  return {
    projectId: 47,
    key: "47|false|",
    issues: [],
    checkFailed: false,
    summary: null,
    assetAudit: null,
    summaryFailed: false,
    ranAt: RAN_AT,
    ...overrides,
  };
}

describe("checkStaleReason", () => {
  it("stays quiet when the result still answers the current question", () => {
    expect(
      checkStaleReason(run(), "47|false|", "2026-08-07T09:00:00Z"),
    ).toBeNull();
  });

  it("reports a settings change", () => {
    expect(checkStaleReason(run(), "47|true|", "2026-08-07T09:00:00Z")).toBe(
      "settings",
    );
  });

  it("reports an edit made after the run", () => {
    expect(checkStaleReason(run(), "47|false|", "2026-08-07T10:00:01Z")).toBe(
      "project-edited",
    );
  });

  it("says nothing when there is no result to be stale", () => {
    expect(
      checkStaleReason(null, "47|false|", "2030-01-01T00:00:00Z"),
    ).toBeNull();
  });

  it("does not invent a warning from a missing or unparsable timestamp", () => {
    expect(checkStaleReason(run(), "47|false|", null)).toBeNull();
    expect(checkStaleReason(run(), "47|false|", undefined)).toBeNull();
    expect(checkStaleReason(run(), "47|false|", "not a date")).toBeNull();
  });

  it("prefers the settings reason when both are true", () => {
    expect(checkStaleReason(run(), "47|true|", "2026-08-07T10:00:01Z")).toBe(
      "settings",
    );
  });
});

describe("useExportCheckStore", () => {
  beforeEach(() => {
    useExportCheckStore.setState({
      runsByProject: {},
      activeByProject: {},
      nextGeneration: 0,
    });
  });

  it("keeps project results isolated while checks overlap", () => {
    const store = useExportCheckStore.getState();
    const projectA = store.startRun(47, "47|false|");
    const projectB = store.startRun(59, "59|false|");
    expect(projectA).not.toBeNull();
    expect(projectB).not.toBeNull();
    expect(store.isRunning(47)).toBe(true);
    expect(store.isRunning(59)).toBe(true);

    store.finishRun(projectB!, run({ projectId: 59, key: "59|false|" }));
    store.finishRun(projectA!, run({ projectId: 47 }));

    expect(store.getRun(47)?.projectId).toBe(47);
    expect(store.getRun(59)?.projectId).toBe(59);
    expect(store.isRunning(47)).toBe(false);
    expect(store.isRunning(59)).toBe(false);
  });

  it("keeps the shown result while re-checking the SAME project", () => {
    const first = useExportCheckStore.getState().startRun(47, "47|false|");
    useExportCheckStore.getState().finishRun(first!, run({ projectId: 47 }));
    const second = useExportCheckStore.getState().startRun(47, "47|false|");
    expect(useExportCheckStore.getState().getRun(47)?.projectId).toBe(47);
    expect(useExportCheckStore.getState().isRunning(47)).toBe(true);
    expect(second).not.toBeNull();
  });

  it("ignores a late completion after the request generation was settled", () => {
    const first = useExportCheckStore.getState().startRun(47, "47|false|");
    useExportCheckStore
      .getState()
      .finishRun(first!, run({ checkFailed: true }));
    const second = useExportCheckStore.getState().startRun(47, "47|true|");
    useExportCheckStore.getState().finishRun(first!, run({ key: "47|false|" }));

    expect(useExportCheckStore.getState().getRun(47)?.checkFailed).toBe(true);
    expect(useExportCheckStore.getState().isRunning(47)).toBe(true);
    expect(second).not.toBeNull();
  });
});

describe("settleExportCheckWithinTimeout", () => {
  it("settles as timed out and cannot be revised by a late response", async () => {
    vi.useFakeTimers();
    try {
      let resolveWork: (value: string) => void = () => undefined;
      const work = new Promise<string>((resolve) => {
        resolveWork = resolve;
      });

      const outcome = settleExportCheckWithinTimeout(work, 25);
      await vi.advanceTimersByTimeAsync(25);
      await expect(outcome).resolves.toEqual({ kind: "timed-out" });
      resolveWork("late");
      await Promise.resolve();
      await expect(outcome).resolves.toEqual({ kind: "timed-out" });
    } finally {
      vi.useRealTimers();
    }
  });
});
