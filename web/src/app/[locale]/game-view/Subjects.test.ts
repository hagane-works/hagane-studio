import { readFileSync } from "node:fs";
import { join } from "node:path";
import { describe, expect, it } from "vitest";
import type { GuiNode, GuiWindow } from "@/lib/guiRenderApi";
import {
  exilesWindow,
  freedomWidth,
  subjectRows,
  subjectsWindow,
  type GuiSubjects,
} from "./SubjectsLayer";
import { autonomyChanges } from "./AutonomyEditPanel";
import { OPENS_OVER, PANEL_TARGETS, matchNamePath } from "./gameNavigation";

const node = (
  name: string,
  type: string,
  children: GuiNode[] = [],
  extra: Partial<GuiNode> = {},
): GuiNode =>
  ({ name, type, children, position: { x: 0, y: 0 }, ...extra }) as GuiNode;

const view = (): GuiWindow =>
  ({
    name: "manage_subjects_window",
    sprites: {},
    texts: {},
    fonts: {},
    node: node("manage_subjects_window", "containerwindowtype", [
      node(
        "subject_items_window",
        "containerwindowtype",
        [node("subjects_grid", "gridboxtype")],
        {
          position: { x: -356, y: 78 },
          raw: { show_position: { x: 2, y: 37 } },
        },
      ),
    ]),
  }) as unknown as GuiWindow;

describe("the Subjects window (情报结尾 1:26–1:36)", () => {
  it("opens from Manage subjects beside the politics screen", () => {
    expect(PANEL_TARGETS["countrypoliticsview/show_subjects_button"]).toBe(
      "manage_subjects_window",
    );
    expect(OPENS_OVER.manage_subjects_window).toBe("countrypoliticsview");
    const page = readFileSync(join(__dirname, "page.tsx"), "utf8");
    expect(page).toContain("subjectsWindow(win)");
    expect(page).toContain("<SubjectsLayer");
    expect(page).toContain("onEditLevel={onEditAutonomy}");
    expect(
      readFileSync(join(__dirname, "SubjectsLayer.tsx"), "utf8"),
    ).toContain("{onEditLevel && (");
  });

  it("has its list slid in to where it opens", () => {
    const out = subjectsWindow(view());
    expect(
      matchNamePath(out.win.node, "subject_items_window")[0].position,
    ).toEqual({ x: 2, y: 37 });
  });

  it("slides the Governments in Exile list into the visible popup", () => {
    const win = view();
    win.name = "exiles_window";
    win.node = node("exiles_window", "containerwindowtype", [
      node("exile_items_window", "containerwindowtype", [], {
        position: { x: -356, y: 78 },
        raw: { show_position: { x: 2, y: 37 } },
      }),
    ]);

    const out = exilesWindow(win);
    expect(
      matchNamePath(out.win.node, "exile_items_window")[0].position,
    ).toEqual({ x: 2, y: 37 });
  });

  it("fills the bar as far as the freedom_level: Australia a fifth, Canada two", () => {
    expect(freedomWidth(0.2)).toBe(19);
    expect(freedomWidth(0.4)).toBe(37);
    expect(freedomWidth(0)).toBe(0);
    expect(freedomWidth(1.5)).toBe(93);
  });

  it("treats a successful older payload without subjects as an empty list", () => {
    expect(
      subjectRows({ tag: "ENG", sprites: {}, levels: {} } as GuiSubjects),
    ).toEqual([]);
    expect(
      subjectRows({ tag: "ENG", subjects: [], sprites: {}, levels: {} }),
    ).toEqual([]);
  });

  it("stacks the flag between the entry and its frame", () => {
    const layer = readFileSync(join(__dirname, "SubjectsLayer.tsx"), "utf8");
    expect(layer).toContain(
      'const LAYERED = ["autonomy_country_flag", "large_flag_frame", "autonomy_progress"',
    );
    expect(layer.indexOf("aria-label={`flag ${subject.tag}`}")).toBeLessThan(
      layer.indexOf("entry={parts.frame}"),
    );
  });
});

describe("an autonomy level's edit panel", () => {
  const colony = {
    id: "autonomy_colony",
    label: "Colony",
    file: "common/autonomous_states/colony.txt",
    min_freedom_level: 0.6,
    manpower_influence: 0.7,
    rule: "can_not_declare_war = yes",
    modifier: "extra_trade_to_overlord_factor = 0.5",
  };

  it("sends only what changed, keeping the scale between 0 and 1", () => {
    expect(autonomyChanges(colony, {})).toEqual({});
    expect(
      autonomyChanges(colony, { min_freedom_level: "0.6", name: "Colony" }),
    ).toEqual({});
    expect(
      autonomyChanges(colony, {
        min_freedom_level: "0.9",
        name: "Crown Colony",
      }),
    ).toEqual({
      values: { min_freedom_level: 0.9 },
      name: "Crown Colony",
    });

    expect(autonomyChanges(colony, { manpower_influence: "" })).toEqual({
      values: { manpower_influence: null },
    });
    expect(autonomyChanges(colony, { min_freedom_level: "" })).toBeNull();
    expect(autonomyChanges(colony, { min_freedom_level: "1.5" })).toBeNull();
    expect(autonomyChanges(colony, { name: "  " })).toBeNull();
    expect(autonomyChanges(colony, { modifier: "  " })).toEqual({
      modifier: "",
    });
  });

  it("opens from a row's buttons in edit mode and saves the name through the text route", () => {
    const layer = readFileSync(join(__dirname, "SubjectsLayer.tsx"), "utf8");
    expect(layer).toContain(
      'label={tw("hit", { tag: subject.tag, name: subject.level.label })}',
    );
    const page = readFileSync(join(__dirname, "page.tsx"), "utf8");
    expect(page).toContain(
      "{editMode && !countryPanelOpen && panel === SUBJECTS_WINDOW && editedAutonomy && !selection && (",
    );
    const panel = readFileSync(
      join(__dirname, "AutonomyEditPanel.tsx"),
      "utf8",
    );
    expect(panel).toContain("/gui-render/autonomy-save");
    expect(panel).toContain("await guiRenderApi.setText(level.id, name);");
  });
});
