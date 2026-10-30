import { readFileSync } from "node:fs";
import { resolve } from "node:path";
import { describe, expect, it } from "vitest";

const read = (path: string) => readFileSync(resolve(__dirname, path), "utf8");
const LOCALES = ["en", "zh", "de", "es", "fr", "ja", "ko", "ru"];

describe("地图创作 brush size", () => {
  const src = read("MapEditor.tsx");

  it("reaches 1 everywhere but new land, which stops at 2", () => {
    expect(src).toContain("const LAND_MIN_RADIUS = 2;");
    expect(src).toContain("min={landMode ? LAND_MIN_RADIUS : 1}");
    expect(src).toContain(
      'Math.max(modeRef.current === "land" ? LAND_MIN_RADIUS : 1, r <= 3 ? r - 1 : Math.round(r / 1.25))',
    );
    expect(src).not.toContain("min={3}");
  });

  it("paints new land with at least the land minimum, whatever the shared setting", () => {
    expect(src).toContain(
      'radiusRef.current = mode === "land" ? Math.max(LAND_MIN_RADIUS, radius) : radius;',
    );
    expect(src).toContain(
      'radius: sized ? mode === "land" ? Math.max(LAND_MIN_RADIUS, radius) : radius : 1',
    );
  });

  it("warns about a small land brush in every language", () => {
    expect(src).toContain(
      '{landMode && radius <= LAND_MIN_RADIUS && (<span className="mt-1 block leading-5 text-warning">{t("landBrushSmall")}</span>)}',
    );
    for (const locale of LOCALES) {
      const messages = JSON.parse(
        read(`../../../../../messages/${locale}/mapStudio.json`),
      ) as Record<string, string>;
      expect(messages.landBrushSmall, locale).toMatch(/8/);
    }
  });
});
