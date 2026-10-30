import { renderHook, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

const nav = vi.hoisted(() => ({ query: "bm=7" }));

vi.mock("@/platform/navigation", () => ({
  useSearchParams: () => new URLSearchParams(nav.query),
}));

import { useLinkedRowSelection } from "./useLinkedRowSelection";

type Row = { id: number };

describe("useLinkedRowSelection", () => {
  it("waits for the list, then selects the named row once through the workbench's own select", async () => {
    nav.query = "bm=7";
    const select = vi.fn();
    const { rerender } = renderHook(
      ({ rows }: { rows: Row[] }) => useLinkedRowSelection("bm", rows, select),
      {
        initialProps: { rows: [] as Row[] },
      },
    );
    expect(select).not.toHaveBeenCalled();

    rerender({ rows: [{ id: 3 }, { id: 7 }] });

    expect(select).not.toHaveBeenCalled();
    await waitFor(() => expect(select).toHaveBeenCalledTimes(1));
    expect(select).toHaveBeenCalledWith({ id: 7 });

    rerender({ rows: [{ id: 3 }, { id: 7 }, { id: 9 }] });
    await Promise.resolve();
    expect(select).toHaveBeenCalledTimes(1);
  });

  it("does nothing without the parameter or when the row is not in the list", async () => {
    nav.query = "";
    const none = vi.fn();
    renderHook(() => useLinkedRowSelection("bm", [{ id: 7 }], none));
    expect(none).not.toHaveBeenCalled();

    nav.query = "bm=42";
    const missing = vi.fn();
    renderHook(() => useLinkedRowSelection("bm", [{ id: 7 }], missing));
    await Promise.resolve();
    expect(none).not.toHaveBeenCalled();
    expect(missing).not.toHaveBeenCalled();
  });
});
