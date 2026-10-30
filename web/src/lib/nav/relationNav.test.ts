import { afterEach, describe, expect, it, vi } from "vitest";
import type { AppRouter as AppRouterInstance } from "@/platform/navigation";

import { jumpToTarget, jumpToSource, canJump } from "./relationNav";
import { useIdeaStore } from "@/lib/store/ideaStore";
import { useCharacterStore } from "@/lib/store/characterStore";

function makeRouter(): AppRouterInstance & { push: ReturnType<typeof vi.fn> } {
  return {
    push: vi.fn(),
    replace: vi.fn(),
    prefetch: vi.fn(),
    back: vi.fn(),
    forward: vi.fn(),
    refresh: vi.fn(),
  } as unknown as AppRouterInstance & { push: ReturnType<typeof vi.fn> };
}

function idea(id: number, ideaId: string) {
  return { id, idea_id: ideaId } as unknown as ReturnType<
    typeof useIdeaStore.getState
  >["ideas"][number];
}
function character(id: number, tag: string) {
  return { id, tag } as unknown as ReturnType<
    typeof useCharacterStore.getState
  >["characters"][number];
}

afterEach(() => {
  useIdeaStore.setState({ ideas: [], selectedId: null });
  useCharacterStore.setState({ characters: [], selectedId: null });
  vi.useRealTimers();
});

describe("relationNav · canJump", () => {
  it("is true for editor-backed types, false for editor-less ones", () => {
    expect(canJump("idea")).toBe(true);
    expect(canJump("focus")).toBe(true);

    expect(canJump("trait")).toBe(false);
    expect(canJump("mio")).toBe(false);
  });
});

describe("relationNav · jumpToTarget fast path (store already warm)", () => {
  it("selects the row synchronously when the target id is already loaded", () => {
    useIdeaStore.setState({
      ideas: [idea(7, "USA_isolation")],
      selectedId: null,
    });
    const router = makeRouter();

    const ok = jumpToTarget(router, 42, "idea", "USA_isolation");

    expect(ok).toBe(true);
    expect(router.push).toHaveBeenCalledWith("/project/42/ideas");

    expect(useIdeaStore.getState().selectedId).toBe(7);
  });

  it("returns false (no nav) for an editor-less target type", () => {
    const router = makeRouter();
    expect(jumpToTarget(router, 1, "opinion_modifier", "x")).toBe(false);
    expect(router.push).not.toHaveBeenCalled();
  });
});

describe("relationNav · cross-workbench race (data arrives AFTER navigation)", () => {
  it("jumpToTarget selects the row once the target store loads it", () => {
    useIdeaStore.setState({ ideas: [], selectedId: null });
    const router = makeRouter();

    const ok = jumpToTarget(router, 42, "idea", "GER_alpha");
    expect(ok).toBe(true);
    expect(router.push).toHaveBeenCalledWith("/project/42/ideas");

    expect(useIdeaStore.getState().selectedId).toBeNull();

    useIdeaStore.setState({
      ideas: [idea(99, "GER_alpha"), idea(100, "GER_beta")],
    });

    expect(useIdeaStore.getState().selectedId).toBe(99);
  });

  it("jumpToSource (incoming edge, by PK) selects once the store loads", () => {
    useCharacterStore.setState({ characters: [], selectedId: null });
    const router = makeRouter();

    const ok = jumpToSource(router, 5, "character", 314);
    expect(ok).toBe(true);
    expect(router.push).toHaveBeenCalledWith("/project/5/characters");
    expect(useCharacterStore.getState().selectedId).toBeNull();

    useCharacterStore.setState({ characters: [character(314, "GER_hitler")] });

    expect(useCharacterStore.getState().selectedId).toBe(314);
  });

  it("stops retrying after the timeout window (dangling reference stays on list)", () => {
    vi.useFakeTimers();
    useIdeaStore.setState({ ideas: [], selectedId: null });
    const router = makeRouter();

    jumpToTarget(router, 1, "idea", "NEVER_EXISTS");

    vi.advanceTimersByTime(6001);

    useIdeaStore.setState({ ideas: [idea(1, "SOMETHING_ELSE")] });
    expect(useIdeaStore.getState().selectedId).toBeNull();
  });
});
