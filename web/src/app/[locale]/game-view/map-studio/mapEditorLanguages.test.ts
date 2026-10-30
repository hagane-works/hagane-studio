import { readFileSync } from "node:fs";
import { join } from "node:path";
import { describe, expect, it } from "vitest";

const editor = readFileSync(join(__dirname, "MapEditor.tsx"), "utf8");
const world = readFileSync(join(__dirname, "NewWorldStudio.tsx"), "utf8");

describe("map studio in every language", () => {
  it("widens the tool dock where the words are longer, and lets buttons grow rather than overflow", () => {
    expect(editor).toContain(
      String.raw`const compactLabels = /^(zh|ja|ko)\b/i.test(locale);`,
    );
    expect(editor).toContain('${compactLabels ? "w-64" : "w-80"}');
    expect(editor).toContain(
      '${compactLabels ? "grid-cols-3" : "grid-cols-1"}',
    );
    expect(editor).not.toMatch(
      /ui-focus-ring h-7 rounded-md border px-1 text-xs/,
    );
  });

  it("says every refusal through mapProblemMessage, never the server's sentence", () => {
    for (const source of [editor, world]) {
      expect(source).not.toContain("mapResponseDetail");
      expect(source).toContain("mapProblemMessage(");
    }
    expect(world).toContain("mapFindingMessage(finding, t, locale)");
    expect(world).not.toContain('.join("、")');
  });

  it("marks what a refusal names on the map and takes the view there", () => {
    expect(editor).toContain("provinceFlags: new Set(flaggedIndexes)");
    expect(editor).toContain("focus={focus}");
    expect(editor).toContain('t("problem.marked")');
  });
});
