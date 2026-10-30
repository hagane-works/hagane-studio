import { describe, expect, it } from "vitest";
import { editorMarkerRange, type MarkerTextModel } from "./editorMarkers";

function modelFor(text: string): MarkerTextModel {
  const lines = text.split("\n");
  return {
    getLineCount: () => lines.length,
    getLineLength: (lineNumber) => lines[lineNumber - 1]?.length ?? 0,
  };
}

describe("editorMarkerRange", () => {
  it("keeps a backend half-open child span instead of widening to line start", () => {
    const model = modelFor(
      "swap_ideas = { add_idea = good remove_idea = old }\n" +
        "    add_idea = rogue",
    );

    expect(
      editorMarkerRange(
        {
          line: 2,
          startColumn: 5,
          endLine: 2,
          endColumn: 21,
          msg: "child outside swap_ideas",
          severity: "warning",
        },
        model,
      ),
    ).toEqual({
      startLineNumber: 2,
      startColumn: 5,
      endLineNumber: 2,
      endColumn: 21,
    });
  });

  it("preserves whole-line fallback markers and clamps stale positions", () => {
    const model = modelFor("short\nline");
    expect(
      editorMarkerRange(
        { line: 99, msg: "parse failed", severity: "error" },
        model,
      ),
    ).toEqual({
      startLineNumber: 2,
      startColumn: 1,
      endLineNumber: 2,
      endColumn: 5,
    });
  });
});
