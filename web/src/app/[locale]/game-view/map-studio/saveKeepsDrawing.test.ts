import { readFileSync } from "node:fs";
import { resolve } from "node:path";
import { describe, expect, it } from "vitest";

const read = (path: string) => readFileSync(resolve(__dirname, path), "utf8");

function at(text: string, needle: string, from = 0): number {
  const i = text.indexOf(needle, from);
  expect(i, `missing: ${needle}`).toBeGreaterThanOrEqual(0);
  return i;
}

describe("地图创作 keeps the drawing up until the map shows the save", () => {
  it("new land: the painting goes only after the map picture was read again", () => {
    const src = read("MapEditor.tsx");
    const start = at(src, "const saveLand = async");
    const end = at(src, "const saveOwners = async", start);
    const body = src.slice(start, end);
    expect(at(body, 'settle("land")')).toBeGreaterThan(
      at(body, "await refreshMapPicture()"),
    );
    expect(at(body, 'settle("land")')).toBeGreaterThan(
      at(body, "await globalMutate(worldMapKey)"),
    );
  });

  it("undo of new land drops the old province index without waiting for the picture", () => {
    const src = read("MapEditor.tsx");
    const start = at(src, "const undoSave = async");
    const body = src.slice(start, at(src, "} catch (err) {", start));
    const land = body.slice(
      at(body, "if (lastSave.land) {"),
      at(body, "} else if (lastSave.regions) {"),
    );

    expect(land).toContain("void refreshMapPicture();");
    expect(land).not.toContain("onMapChanged?.()");

    const refresh = src.slice(
      at(src, "const refreshMapPicture = async"),
      at(src, "const waitForOwners = async"),
    );
    expect(at(refresh, "indexRef.current = null")).toBeLessThan(
      at(refresh, "await new Promise"),
    );
  });

  it("the map canvas reports no province for numbers the map does not have", () => {
    const src = read(
      "../../../../components/workbench/world-map/WorldMapCanvas.tsx",
    );
    const body = src.slice(
      at(src, "const provinceAt = ("),
      at(src, "const mapPoint = ("),
    );
    expect(body).toContain(
      "return province < mapRef.current.provinces.length ? province : 0;",
    );
  });

  it("new world: the studio closes only after the map drew the new picture", () => {
    const src = read("../page.tsx");
    const start = at(src, "onWorldSaved={async (states) => {");
    const end = at(
      src,
      "}}",
      at(src, 'toast.success(t("mapStudioSaved")', start),
    );
    const body = src.slice(start, end);
    const close = at(body, "setMapStudio(false)");
    expect(close).toBeGreaterThan(at(body, "mapShownWaiter.current = waiter"));
    expect(close).toBeGreaterThan(at(body, "await globalMutate("));

    expect(src).toContain(
      "if (waiter && shown > waiter.after) waiter.finish();",
    );
  });

  it("new world: the studio keeps what was shown while it waits", () => {
    const src = read("NewWorldStudio.tsx");
    const start = at(src, "if (save) {");
    expect(at(src, "setPreview(null)", start)).toBeGreaterThan(
      at(src, "await onSaved(", start),
    );
  });

  it("the game map reports the version of the picture it drew", () => {
    expect(read("../GameWorldMap.tsx")).toContain(
      "onIndex={onPictureShown ? () => onPictureShown(version) : undefined}",
    );
  });
});
