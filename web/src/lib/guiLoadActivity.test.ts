import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import {
  getGuiLoadSnapshot,
  SLOW_AFTER_MS,
  trackGuiRead,
} from "./guiLoadActivity";

function deferred<T>() {
  let resolve!: (value: T) => void;
  let reject!: (reason: unknown) => void;
  const promise = new Promise<T>((ok, fail) => {
    resolve = ok;
    reject = fail;
  });
  return { promise, resolve, reject };
}

describe("guiLoadActivity", () => {
  beforeEach(() => vi.useFakeTimers());
  afterEach(() => vi.useRealTimers());

  it("counts reads in flight and finished, failures included, and passes results through", async () => {
    const start = getGuiLoadSnapshot().finished;
    const ok = deferred<string>();
    const bad = deferred<string>();
    const first = trackGuiRead(ok.promise);
    const second = trackGuiRead(bad.promise).catch(
      (error: Error) => error.message,
    );
    expect(getGuiLoadSnapshot().pending).toBe(2);
    ok.resolve("window");
    bad.reject(new Error("503"));
    await expect(first).resolves.toBe("window");
    await expect(second).resolves.toBe("503");
    expect(getGuiLoadSnapshot().pending).toBe(0);
    expect(getGuiLoadSnapshot().finished - start).toBe(2);
  });

  it("flags a round as slow only after it has run past the threshold, and clears it when idle", async () => {
    const quick = deferred<null>();
    const tracked = trackGuiRead(quick.promise);
    vi.advanceTimersByTime(SLOW_AFTER_MS - 1);
    expect(getGuiLoadSnapshot().slow).toBe(false);
    quick.resolve(null);
    await tracked;
    vi.advanceTimersByTime(SLOW_AFTER_MS);
    expect(getGuiLoadSnapshot().slow).toBe(false);

    const slow = deferred<null>();
    const pending = trackGuiRead(slow.promise);
    vi.advanceTimersByTime(SLOW_AFTER_MS);
    expect(getGuiLoadSnapshot().slow).toBe(true);
    slow.resolve(null);
    await pending;
    expect(getGuiLoadSnapshot()).toMatchObject({ pending: 0, slow: false });
  });
});
