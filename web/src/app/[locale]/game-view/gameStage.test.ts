import { describe, expect, it } from "vitest";
import {
  canSkipGameViewRefresh,
  clearOfMapModes,
  gameStageGeometry,
  isGameDataKey,
  isGameViewRefreshKey,
} from "./gameStage";

describe("GameView stage geometry", () => {
  it("positions game controls from the full viewport", () => {
    const right = String(clearOfMapModes(12).right);
    expect(right).toContain("min(100vw / 1920, 100vh / 1080) * 96");
    expect(right).toContain("(100vw - min(100vw, 100vh * 1920 / 1080)) / 2");
  });

  it("always fits and centres the game in the complete viewport", () => {
    expect(gameStageGeometry(1920, 1080)).toEqual({ centerX: 960, scale: 1 });
    expect(gameStageGeometry(1280, 720)).toEqual({
      centerX: 640,
      scale: 2 / 3,
    });
    expect(gameStageGeometry(1920, 800).centerX).toBe(960);
    expect(gameStageGeometry(1920, 800).scale).toBeCloseTo(800 / 1080);
  });
});

describe("isGameDataKey", () => {
  it("reloads the screens' data but not their .gui windows", () => {
    expect(isGameDataKey(["game-view-characters", 2, "GER"])).toBe(true);
    expect(isGameDataKey(["game-view-focus-tree", 2, null, "GER"])).toBe(true);
    expect(isGameDataKey(["game-view-window", 2, "topbar"])).toBe(false);
    expect(isGameDataKey(["project", 2])).toBe(false);
    expect(isGameDataKey("game-view-characters")).toBe(false);
  });
});

describe("canSkipGameViewRefresh", () => {
  it("only skips a quiet refresh with the same known version", () => {
    expect(canSkipGameViewRefresh(true, "v2", "v2")).toBe(true);
    expect(canSkipGameViewRefresh(false, "v2", "v2")).toBe(false);
    expect(canSkipGameViewRefresh(true, null, null)).toBe(false);
    expect(canSkipGameViewRefresh(true, "v3", "v2")).toBe(false);
  });
});

describe("isGameViewRefreshKey", () => {
  it("adds world-map caches only for an explicit full refresh", () => {
    expect(isGameViewRefreshKey(["game-view-stats", 2], false)).toBe(true);
    expect(isGameViewRefreshKey(["world-map", 2, "1936.1.1"], false)).toBe(
      false,
    );
    expect(isGameViewRefreshKey(["world-map", 2, "1936.1.1"], true)).toBe(true);
    expect(isGameViewRefreshKey(["world-map-rivers", 2], true)).toBe(true);
    expect(isGameViewRefreshKey(["project", 2], true)).toBe(false);
  });
});
