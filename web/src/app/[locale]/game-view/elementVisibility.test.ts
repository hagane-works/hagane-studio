import { describe, expect, it } from "vitest";
import type { GuiEdit } from "../gui-renderer/guiOverlay";
import {
  addedNodeEdit,
  withElementHidden,
  withElementShown,
  withoutAddedElement,
} from "./elementVisibility";

const text: GuiEdit = {
  id: "set_text:panel/title",
  target: "panel/title",
  op: "set_text",
  payload: { key: "X" },
};
const added: GuiEdit = {
  id: "add_node:panel/badge",
  target: "panel",
  op: "add_node",
  payload: { node: { type: "iconType", name: "badge" } },
};
const ofAdded: GuiEdit = {
  id: "set_layout:panel/badge",
  target: "panel/badge",
  op: "set_layout",
  payload: { x: 4 },
};

describe("hiding a base-game element", () => {
  it("adds one hide_node for it and keeps the file's other edits", () => {
    const once = withElementHidden([text], "panel/close");
    expect(once).toEqual([
      text,
      {
        id: "hide_node:panel/close",
        target: "panel/close",
        op: "hide_node",
        payload: {},
      },
    ]);
    expect(withElementHidden(once, "panel/close")).toEqual(once);
  });

  it("shows it again by taking only its hide out", () => {
    const hidden = withElementHidden(
      withElementHidden([text], "panel/close"),
      "panel/flag",
    );
    expect(
      withElementShown(hidden, "panel/close").map((edit) => edit.id),
    ).toEqual(["set_text:panel/title", "hide_node:panel/flag"]);
  });
});

describe("an element the player added", () => {
  it("is found by its add_node edit, whose target is its parent", () => {
    expect(addedNodeEdit([text, added], "panel/badge")).toBe(added);
    expect(addedNodeEdit([text, added], "panel/title")).toBeUndefined();
  });

  it("is removed with every edit of it, which an undo puts back", () => {
    const { edits, removed } = withoutAddedElement(
      [text, added, ofAdded],
      "panel/badge",
    );
    expect(edits).toEqual([text]);
    expect(removed).toEqual([added, ofAdded]);
  });
});
