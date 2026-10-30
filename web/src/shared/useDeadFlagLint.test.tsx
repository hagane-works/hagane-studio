import type { ReactNode } from "react";
import { act, renderHook, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { SWRConfig } from "swr";

vi.mock("@/lib/api", () => ({
  lintApi: { deadFlags: vi.fn() },
}));

import { lintApi, type DeadFlagReport } from "@/lib/api";
import { useDeadFlagLint } from "./useDeadFlagLint";

const deadFlagsMock = vi.mocked(lintApi.deadFlags);

function report(): DeadFlagReport {
  return {
    set_total: 3,
    check_total: 5,
    dead_flags: [{ flag_id: "my_dead_flag", set_locations: ["focus:x"] }],
  };
}

function makeWrapper() {
  const cache = new Map();
  return function Wrapper({ children }: { children: ReactNode }) {
    return (
      <SWRConfig value={{ provider: () => cache, shouldRetryOnError: false }}>
        {children}
      </SWRConfig>
    );
  };
}

describe("useDeadFlagLint (SWR-backed)", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("跨挂载共享缓存:重挂(工作台切换)在 dedup 窗口内不重跑 lint", async () => {
    deadFlagsMock.mockResolvedValue(report());
    const wrapper = makeWrapper();

    const first = renderHook(() => useDeadFlagLint(42), { wrapper });
    await waitFor(() => expect(first.result.current.report).not.toBeNull());
    expect(deadFlagsMock).toHaveBeenCalledTimes(1);
    first.unmount();

    const second = renderHook(() => useDeadFlagLint(42), { wrapper });
    await waitFor(() =>
      expect(second.result.current.report?.dead_flags[0].flag_id).toBe(
        "my_dead_flag",
      ),
    );
    expect(deadFlagsMock).toHaveBeenCalledTimes(1);
  });

  it("projectId 为 null(面板未打开)时不发请求", () => {
    const { result } = renderHook(() => useDeadFlagLint(null), {
      wrapper: makeWrapper(),
    });
    expect(result.current.report).toBeNull();
    expect(deadFlagsMock).not.toHaveBeenCalled();
  });

  it("失败不阻塞:report 落 null,不外抛", async () => {
    deadFlagsMock.mockRejectedValue(new Error("backend restarting"));
    const { result } = renderHook(() => useDeadFlagLint(42), {
      wrapper: makeWrapper(),
    });
    await waitFor(() => expect(deadFlagsMock).toHaveBeenCalledTimes(1));
    expect(result.current.report).toBeNull();
  });

  it("refresh() 强制重拉(绕过 dedup)", async () => {
    deadFlagsMock.mockResolvedValue(report());
    const { result } = renderHook(() => useDeadFlagLint(42), {
      wrapper: makeWrapper(),
    });
    await waitFor(() => expect(result.current.report).not.toBeNull());

    await act(async () => {
      await result.current.refresh();
    });

    expect(deadFlagsMock.mock.calls.length).toBeGreaterThanOrEqual(2);
  });
});
