import type { CSSProperties } from "react";

export function gameStageGeometry(
  viewportWidth: number,
  viewportHeight: number,
  baseWidth = 1920,
  baseHeight = 1080,
): { centerX: number; scale: number } {
  const availableWidth = Math.max(1, viewportWidth);
  return {
    centerX: availableWidth / 2,
    scale: Math.max(
      0.01,
      Math.min(
        availableWidth / baseWidth,
        Math.max(1, viewportHeight) / baseHeight,
      ),
    ),
  };
}

const MAP_MODES_CLEARANCE = 96;

export function clearOfMapModes(px: number): CSSProperties {
  const scale = "min(100vw / 1920, 100vh / 1080)";
  const letterbox = "(100vw - min(100vw, 100vh * 1920 / 1080)) / 2";
  return {
    right: `calc(${px}px + ${letterbox} + ${scale} * ${MAP_MODES_CLEARANCE})`,
  };
}

export function isGameDataKey(key: unknown): boolean {
  return (
    Array.isArray(key) &&
    typeof key[0] === "string" &&
    key[0].startsWith("game-view-") &&
    key[0] !== "game-view-window"
  );
}

export function isGameViewRefreshKey(
  key: unknown,
  includeWorldMap: boolean,
): boolean {
  return (
    Array.isArray(key) &&
    typeof key[0] === "string" &&
    (key[0].startsWith("game-view-") ||
      (includeWorldMap && key[0].startsWith("world-map")))
  );
}

export function canSkipGameViewRefresh(
  quiet: boolean,
  version: string | null,
  checkedVersion: string | null,
): boolean {
  return quiet && version !== null && version === checkedVersion;
}
