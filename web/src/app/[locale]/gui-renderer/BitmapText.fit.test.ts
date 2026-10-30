import { describe, expect, it } from "vitest";
import type { GuiBitmapFont } from "@/lib/guiRenderApi";
import { fitLines } from "./BitmapText";

const chars: Record<string, number[]> = {};
for (let code = 32; code < 127; code++)
  chars[String(code)] = [0, 0, 10, 20, 0, 0, 10];
const FONT: GuiBitmapFont = {
  line: 20,
  base: 16,
  atlas: "atlas.png",
  atlas_w: 64,
  atlas_h: 64,
  tint: "alpha",
  chars,
};

describe("fitLines", () => {
  it("cuts a fixedsize box's one line short with dots, mid-word", () => {
    expect(fitLines(FONT, "Inter-War Small Airframe", 100, 1, true)).toEqual([
      "Inter-W...",
    ]);
  });

  it("keeps the lines that fit and cuts the last one", () => {
    expect(fitLines(FONT, "Inter-War Small Airframe", 100, 2, true)).toEqual([
      "Inter-War",
      "Small A...",
    ]);
  });

  it("cuts where the dots' pixels, not the pen, would cross the box", () => {
    const wide = {
      ...chars,
      "46": [0, 0, 6, 20, 0, 0, 4] as [
        number,
        number,
        number,
        number,
        number,
        number,
        number,
      ],
    };
    const font: GuiBitmapFont = { ...FONT, chars: wide };

    expect(fitLines(font, "abcde fgh", 52, 1, true)).toEqual(["abc..."]);
    expect(fitLines(font, "abcde fgh", 54, 1, true)).toEqual(["abcd..."]);
  });

  it("leaves text that fits alone", () => {
    expect(fitLines(FONT, "Ju 52", 100, 1, true)).toEqual(["Ju 52"]);
  });

  it("only drops the extra lines of a box that is not fixedsize", () => {
    expect(fitLines(FONT, "Inter-War Small Airframe", 100, 1)).toEqual([
      "Inter-War",
    ]);
  });

  it("renders a missing runtime label as blank instead of crashing GameView", () => {
    expect(fitLines(FONT, undefined as unknown as string, 100, 1)).toEqual([
      "",
    ]);
  });
});
