import { readFileSync } from "node:fs";
import { join } from "node:path";
import { describe, expect, it } from "vitest";
import { eventFocusTarget } from "./EventEditPanel";
import { changedKeys } from "./workbenchProject";

const flat = (file: string) =>
  readFileSync(join(__dirname, file), "utf8").replace(/\s+/g, " ");

describe("the workbench editors in the game view's panels", () => {
  it("are the workbenches' own components, with the project loaded first", () => {
    expect(flat("CharacterEditPanel.tsx")).toContain("<CharacterEditor");
    expect(flat("DecisionEditPanel.tsx")).toContain("<DecisionEditor");
    expect(flat("EventEditPanel.tsx")).toContain("<EventEditor");
    expect(flat("FocusEditPanel.tsx")).toContain("<FocusNodeEditor");
    expect(flat("TechEditPanel.tsx")).toContain("<TechEditor");

    expect(flat("SpiritEditPanel.tsx")).toContain("<IdeaEditor");
    expect(flat("SpiritEditPanel.tsx")).toContain("<DynamicModifierEditor");

    expect(flat("../../../components/workbench/IdeaEditor.tsx")).toContain(
      '${embedded ? "" : "overflow-y-auto max-h-[calc(100vh-120px)]"}',
    );

    expect(flat("TechEditPanel.tsx")).not.toContain("onDelete={");
    for (const file of [
      "CharacterEditPanel.tsx",
      "DecisionEditPanel.tsx",
      "EventEditPanel.tsx",
      "FocusEditPanel.tsx",
      "TechEditPanel.tsx",
    ]) {
      expect(flat(file), file).toContain("useWorkbenchProject(projectId)");
      expect(flat(file), file).toContain("wide={");
    }
  });

  it("leave swapping pictures to UI 编辑", () => {
    for (const file of [
      "DecisionEditPanel.tsx",
      "EventEditPanel.tsx",
      "FocusEditPanel.tsx",
      "TechEditPanel.tsx",
      "CharacterEditPanel.tsx",
      "page.tsx",
    ]) {
      expect(flat(file), file).not.toContain("<SpriteReplace");
      expect(flat(file), file).not.toContain("ArtInspector");
    }
    expect(flat("EditPanel.tsx")).toContain("<SpriteReplace");
    const layer = flat("CharacterLayer.tsx");
    expect(layer).toContain(
      "onClick={() => onEdit?.(person.token, person.name)}",
    );
    expect(layer).not.toContain("onPick");
  });

  it("record only what a save changed: the editors send the whole form", () => {
    expect(
      changedKeys({ a: 1, b: "x", c: [1] }, { a: 1, b: "y", c: [1] }),
    ).toEqual(["b"]);
    expect(changedKeys({ a: null as number | null }, { a: undefined })).toEqual(
      [],
    );
  });

  it("take a click on the event window to the editor's section", () => {
    expect(eventFocusTarget("title")).toEqual({
      target: "title",
      option: null,
    });
    expect(eventFocusTarget("desc")).toEqual({
      target: "description",
      option: null,
    });
    expect(eventFocusTarget("option:2")).toEqual({
      target: "options",
      option: 2,
    });
    expect(eventFocusTarget("something")).toBeNull();
  });
});
