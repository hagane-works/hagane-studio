import { describe, it, expect } from "vitest";
import type { GuiNode } from "@/lib/guiRenderApi";
import {
  assignIds,
  dragLayoutResult,
  anchorCell,
  cellAnchor,
  type DragState,
} from "./GuiCanvas";
import {
  buildNamePathMap,
  editId,
  upsertEdit,
  removeEditsForTarget,
  addNodeEdit,
  deleteNodeEdit,
  type GuiEdit,
} from "./guiOverlay";

function node(partial: Partial<GuiNode>): GuiNode {
  return {
    type: "containerwindowtype",
    name: null,
    position: { x: 0, y: 0 },
    size: null,
    orientation: null,
    origo: null,
    scale: null,
    sprite: null,
    frame: null,
    text: null,
    text_source: null,
    font: null,
    format: null,
    maxwidth: null,
    maxheight: null,
    text_color: null,
    centerposition: false,
    alwaystransparent: false,
    background: null,
    children: [],
    ...partial,
  };
}

function fixture(): GuiNode {
  const root = node({
    type: "containerwindowtype",
    name: "win",
    children: [
      node({ type: "icontype", name: "icon", sprite: "GFX_a" }),
      node({ type: "buttontype", name: null, sprite: "GFX_b" }),
    ],
  });
  assignIds(root);
  return root;
}

describe("gui overlay NAME-PATH addressing", () => {
  it("maps DFS ids to stable name-paths (named + unnamed fallback)", () => {
    const root = fixture();
    const map = buildNamePathMap(root);

    expect(map["root"]).toBe("win");
    expect(map["root/0"]).toBe("win/icon");

    expect(map["root/1"]).toBe("win/button#1");
  });

  it("set_layout edits round-trip: canvas id -> name-path -> back to id", () => {
    const root = fixture();
    const map = buildNamePathMap(root);
    const reverse: Record<string, string> = {};
    for (const [id, target] of Object.entries(map))
      if (target) reverse[target] = id;

    const target = map["root/0"];
    const edit: GuiEdit = {
      id: editId(target, "set_layout"),
      target,
      op: "set_layout",
      payload: { x: 40, y: 12 },
    };
    const edits = upsertEdit([], edit);

    expect(reverse[edits[0].target]).toBe("root/0");
    expect(edits[0].payload).toEqual({ x: 40, y: 12 });
  });

  it("upsertEdit replaces same target+op (autosave model, no duplicates)", () => {
    const target = "win/icon";
    const a: GuiEdit = {
      id: editId(target, "set_layout"),
      target,
      op: "set_layout",
      payload: { x: 1 },
    };
    const b: GuiEdit = {
      id: editId(target, "set_layout"),
      target,
      op: "set_layout",
      payload: { x: 1, y: 2 },
    };
    const edits = upsertEdit(upsertEdit([], a), b);
    expect(edits).toHaveLength(1);
    expect(edits[0].payload).toEqual({ x: 1, y: 2 });
  });

  it("a set_layout and a set_text edit on the same target coexist", () => {
    const target = "win/icon";
    const layout: GuiEdit = {
      id: editId(target, "set_layout"),
      target,
      op: "set_layout",
      payload: { x: 5 },
    };
    const text: GuiEdit = {
      id: editId(target, "set_text"),
      target,
      op: "set_text",
      payload: { text: "Hi" },
    };
    let edits = upsertEdit([], layout);
    edits = upsertEdit(edits, text);
    expect(edits).toHaveLength(2);

    expect(removeEditsForTarget(edits, target)).toHaveLength(0);
  });
});

describe("dragLayoutResult (drag math — commit must not snap back)", () => {
  const move: DragState = {
    mode: "move",
    startX: 100,
    startY: 100,
    baseX: 40,
    baseY: 20,
  };

  it("a real move produces the moved position (regression: commit != revert)", () => {
    expect(dragLayoutResult(move, 130, 112, 1)).toEqual({ x: 70, y: 32 });
  });

  it("divides the screen-pixel delta by zoom", () => {
    expect(dragLayoutResult(move, 130, 100, 2)).toEqual({ x: 55, y: 20 });
  });

  it("zero net movement is a no-op (null = plain click, keep selection)", () => {
    expect(dragLayoutResult(move, 100, 100, 1)).toBeNull();
  });

  it("resize maps delta to width/height and never goes below 1px", () => {
    const resize: DragState = {
      mode: "resize",
      startX: 200,
      startY: 200,
      baseX: 64,
      baseY: 64,
    };
    expect(dragLayoutResult(resize, 220, 210, 1)).toEqual({
      width: 84,
      height: 74,
    });

    expect(dragLayoutResult(resize, 100, 100, 1)).toEqual({
      width: 1,
      height: 1,
    });
  });

  it("an edge handle resizes one way only, and the other axis's movement is no change", () => {
    const wider: DragState = {
      mode: "resize",
      startX: 200,
      startY: 200,
      baseX: 180,
      baseY: 30,
      axes: "x",
    };
    expect(dragLayoutResult(wider, 240, 230, 1)).toEqual({ width: 220 });
    expect(dragLayoutResult(wider, 200, 260, 1)).toBeNull();
    const taller: DragState = { ...wider, axes: "y" };
    expect(dragLayoutResult(taller, 240, 230, 1)).toEqual({ height: 60 });
  });
});

describe("anchor 3×3 picker mapping (orientation/origo)", () => {
  it("maps anchor strings to the matching [col,row] cell", () => {
    expect(anchorCell("UPPER_LEFT")).toEqual([0, 0]);
    expect(anchorCell("UPPER_RIGHT")).toEqual([2, 0]);
    expect(anchorCell("CENTER")).toEqual([1, 1]);
    expect(anchorCell("LOWER_RIGHT")).toEqual([2, 2]);

    expect(anchorCell("RIGHT")).toEqual([2, 0]);

    expect(anchorCell("CENTER_LEFT")).toEqual([0, 1]);
    expect(anchorCell("CENTER_DOWN")).toEqual([1, 2]);

    expect(anchorCell("")).toEqual([0, 0]);
    expect(anchorCell(null)).toEqual([0, 0]);
  });

  it("cellAnchor is the inverse of anchorCell for canonical cells", () => {
    for (let row = 0; row < 3; row++) {
      for (let col = 0; col < 3; col++) {
        const s = cellAnchor(col, row);
        expect(anchorCell(s)).toEqual([col, row]);
      }
    }
  });

  it("set_orientation / set_origo edits round-trip through upsert (one per op)", () => {
    const target = "win/icon";
    const orient: GuiEdit = {
      id: editId(target, "set_orientation"),
      target,
      op: "set_orientation",
      payload: { value: "CENTER" },
    };
    const origo: GuiEdit = {
      id: editId(target, "set_origo"),
      target,
      op: "set_origo",
      payload: { value: "" },
    };
    let edits = upsertEdit([], orient);
    edits = upsertEdit(edits, origo);

    const layout: GuiEdit = {
      id: editId(target, "set_layout"),
      target,
      op: "set_layout",
      payload: { x: 5 },
    };
    edits = upsertEdit(edits, layout);
    expect(edits).toHaveLength(3);

    edits = upsertEdit(edits, { ...orient, payload: { value: "RIGHT" } });
    expect(edits.filter((e) => e.op === "set_orientation")).toHaveLength(1);
    expect(edits.find((e) => e.op === "set_orientation")?.payload).toEqual({
      value: "RIGHT",
    });

    expect(removeEditsForTarget(edits, target)).toHaveLength(0);
  });
});

describe("structural edits (add_node / delete_node)", () => {
  it("addNodeEdit targets the PARENT path and carries the node in payload", () => {
    const edit = addNodeEdit("win", {
      type: "icontype",
      name: "new_icon",
      sprite: "GFX_x",
    });
    expect(edit.op).toBe("add_node");

    expect(edit.target).toBe("win");
    expect(edit.payload.node).toEqual({
      type: "icontype",
      name: "new_icon",
      sprite: "GFX_x",
    });

    expect("index" in edit.payload).toBe(false);
  });

  it("addNodeEdit threads an explicit insert index", () => {
    const edit = addNodeEdit("win", { type: "icontype", name: "first" }, 0);
    expect(edit.payload.index).toBe(0);
  });

  it("several adds under ONE parent do NOT collapse under upsertEdit", () => {
    const a = addNodeEdit("win", { type: "icontype", name: "icon_a" });
    const b = addNodeEdit("win", { type: "icontype", name: "icon_b" });
    let edits = upsertEdit([], a);
    edits = upsertEdit(edits, b);
    expect(edits).toHaveLength(2);

    const aAgain = addNodeEdit("win", {
      type: "icontype",
      name: "icon_a",
      sprite: "GFX_y",
    });
    edits = upsertEdit(edits, aAgain);
    expect(edits).toHaveLength(2);
    expect(edits.find((e) => e.id === a.id)?.payload.node).toEqual({
      type: "icontype",
      name: "icon_a",
      sprite: "GFX_y",
    });
  });

  it("deleteNodeEdit targets the node path and is idempotent under upsert", () => {
    const d = deleteNodeEdit("win/icon");
    expect(d.op).toBe("delete_node");
    expect(d.target).toBe("win/icon");
    let edits = upsertEdit([], d);
    edits = upsertEdit(edits, deleteNodeEdit("win/icon"));
    expect(edits).toHaveLength(1);

    expect(removeEditsForTarget(edits, "win/icon")).toHaveLength(0);
  });
});

describe("removeEditsForTarget over mixed ops", () => {
  it("clears every op for a target", () => {
    const target = "win/icon";
    const layout: GuiEdit = {
      id: editId(target, "set_layout"),
      target,
      op: "set_layout",
      payload: { x: 5 },
    };
    const text: GuiEdit = {
      id: editId(target, "set_text"),
      target,
      op: "set_text",
      payload: { text: "Hi" },
    };
    let edits = upsertEdit([], layout);
    edits = upsertEdit(edits, text);
    expect(edits).toHaveLength(2);

    expect(removeEditsForTarget(edits, target)).toHaveLength(0);
  });
});
