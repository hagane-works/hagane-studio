import { describe, expect, it } from "vitest";
import { getInitialPositionEditorState } from "./initialPositionUi";

describe("focus opening-camera editor state", () => {
  it("keeps a tagged ambiguous legacy value unresolved with its raw value", () => {
    expect(
      getInitialPositionEditorState({
        initial_show_position: { kind: "legacy_ambiguous", raw: "23" },
        initial_show_focus: "",
      }),
    ).toMatchObject({
      mode: "unresolved",
      unresolved: { kind: "legacy_ambiguous", raw: "23" },
    });
  });

  it("keeps invalid importer evidence unresolved and prefers its source text", () => {
    expect(
      getInitialPositionEditorState({
        initial_show_position: {
          kind: "invalid",
          raw: { x: 7 },
          raw_text: "initial_show_position = { x = 7 }",
        },
        initial_show_focus: "",
      }),
    ).toMatchObject({
      mode: "unresolved",
      unresolved: {
        kind: "invalid",
        raw: "initial_show_position = { x = 7 }",
      },
    });
  });

  it("does not guess whether a numeric compatibility value is a focus or x coordinate", () => {
    expect(
      getInitialPositionEditorState({
        initial_show_position: null,
        initial_show_focus: "-12",
      }),
    ).toMatchObject({
      mode: "unresolved",
      unresolved: { kind: "legacy_ambiguous", raw: "-12" },
    });
  });

  it("still materializes an unambiguous compatibility focus reference", () => {
    expect(
      getInitialPositionEditorState({
        initial_show_position: null,
        initial_show_focus: "FIN_industrial_effort",
      }),
    ).toMatchObject({
      mode: "focus",
      focusId: "FIN_industrial_effort",
      unresolved: null,
    });
  });
});
