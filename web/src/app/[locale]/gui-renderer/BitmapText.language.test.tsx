import { describe, expect, it } from "vitest";
import { isValidElement, type ReactElement } from "react";
import type { GuiBitmapFont } from "@/lib/guiRenderApi";
import {
  FALLBACK_FONT_FAMILY,
  fallbackFontSize,
  fitLines,
  glyphRun,
  measureLine,
} from "./BitmapText";
import { bitmapMeasure, layoutTooltip } from "./tooltipText";

const chars: Record<string, number[]> = {};
for (let code = 32; code < 127; code++)
  chars[String(code)] = [0, 0, 10, 18, 0, 0, 10];
const FONT: GuiBitmapFont = {
  line: 18,
  base: 15,
  atlas: "atlas.png",
  atlas_w: 64,
  atlas_h: 64,
  tint: "alpha",
  chars,
};

type Styled = ReactElement<{
  style: Record<string, unknown>;
  children?: unknown;
}>;

describe("characters the game font lacks", () => {
  it("are drawn in the web font at the game's size for the language, one run each", () => {
    expect(fallbackFontSize(FONT)).toBe(14);
    const { nodes, pen } = glyphRun(FONT, "AB德国C", 5, 2, undefined, "k");
    const spans = nodes.filter(
      (node): node is Styled => isValidElement(node) && node.type === "span",
    );
    expect(spans).toHaveLength(1);
    expect(spans[0].props.children).toBe("德国");
    expect(spans[0].props.style).toMatchObject({
      left: 25,
      top: 2,
      fontSize: 14,
      fontFamily: FALLBACK_FONT_FAMILY,
      color: "#ffffff",
    });

    expect(pen).toBe(5 + 10 + 10 + 14 + 14 + 10);

    expect(
      nodes.filter((node) => isValidElement(node) && node.type === "div"),
    ).toHaveLength(3);
  });

  it("wrap between wide characters, which have no spaces between words", () => {
    expect(fitLines(FONT, "德意志国", 30, 9)).toEqual(["德意", "志国"]);

    expect(fitLines(FONT, "Inter War", 50, 9)).toEqual(["Inter", "War"]);

    expect(fitLines(FONT, "1936年", 45, 9)).toEqual(["1936", "年"]);
    expect(measureLine(FONT, "1936年")).toBe(54);
  });

  it("wrap in tooltips too", () => {
    const layout = layoutTooltip(
      "德意志国",
      bitmapMeasure(FONT),
      () => null,
      30,
    );
    expect(layout.lines.length).toBe(2);
  });
});
