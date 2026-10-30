import { readFileSync } from "node:fs";
import { join } from "node:path";
import { describe, expect, it } from "vitest";
import type { GuiNode, GuiWindow } from "@/lib/guiRenderApi";
import { matchNamePath } from "./gameNavigation";
import { programWindow } from "./ProgramWindowLayer";
import {
  outputWindow,
  projectPageLayout,
  resourcesText,
  sectionsWindow,
  type GuiOutputItem,
  type GuiProjectPage,
} from "./ProjectPageLayer";
import { specialProjectChanges } from "./SpecialProjectEditPanel";
import type { GuiSpecialProject } from "./FacilitiesLayer";

const node = (
  name: string,
  type: string,
  children: GuiNode[] = [],
  extra: Partial<GuiNode> = {},
): GuiNode =>
  ({ name, type, children, position: { x: 0, y: 0 }, ...extra }) as GuiNode;

const item = (name: string, stats: number): GuiOutputItem => ({
  name,
  label: name,
  stats: Array.from({ length: stats }, (_, at) => ({
    stat: `s${at}`,
    label: `S${at}:`,
    value: "1",
  })),
  resources: [],
});

const escort = {
  outputs: [
    "Gain 100.00% discount for 1 design template(s). Can be used for: Escort Carrier",
  ],
  modules: [item("deck", 5)],
  equipment: [item("hull", 12)],
};

describe("a special project's page (情报结尾 1:51–1:53)", () => {
  it("stacks the output list where the recording has it", () => {
    const laid = projectPageLayout(escort, 2);

    expect(
      laid.sections.map((section) => [
        section.kind,
        section.header,
        section.items[0].y,
        section.items[0].rows,
      ]),
    ).toEqual([
      ["modules", 117, 147, 3],
      ["equipment", 357, 387, 6],
    ]);

    expect(laid.height).toBe(387 + 95 + 6 * 30 + 39);
    expect(
      projectPageLayout({ outputs: [], modules: [], equipment: [] }, 0)
        .sections,
    ).toEqual([]);
  });

  it("writes an item's resources as the game does, and keeps only the list's parts", () => {
    expect(
      resourcesText([
        { resource: "steel", amount: 2, frame: 5 },
        { resource: "chromium", amount: 1, frame: 4 },
      ]),
    ).toBe("2£resources_strip|5 1£resources_strip|4");
    const entry = {
      name: "detailed_output_item",
      sprites: {},
      texts: {},
      fonts: {},
      node: node("detailed_output_item", "containerwindowtype", [
        node("header", "instanttextboxtype"),
        node("rewards_header", "instanttextboxtype"),
        node("country_effect_reward_description", "instanttextboxtype"),
        node("unlocked_equipment_modules", "containerwindowtype", [
          node("header", "instanttextboxtype"),
        ]),
        node("unlocked_equipments", "containerwindowtype", [
          node("header", "instanttextboxtype"),
        ]),
      ]),
    } as unknown as GuiWindow;
    const page = { ...escort, modules: [] } as unknown as GuiProjectPage;
    const out = sectionsWindow(
      entry,
      page,
      projectPageLayout(page, 2).sections,
    );
    expect(
      out.node.children.map((child) => [child.name, child.position.y]),
    ).toEqual([
      ["rewards_header", 10],
      ["country_effect_reward_description", 52],
      ["unlocked_equipments", 117],
    ]);
  });

  it("puts the output paper (-20, 35) from its window", () => {
    const entry = {
      name: "output_window",
      sprites: {},
      texts: {},
      fonts: {},
      node: node(
        "output_window",
        "containerwindowtype",
        [node("simple_output_window", "containerwindowtype")],
        {
          background: node("Background", "icontype", [], {
            sprite: "GFX_project_output_bg",
            position: { x: -20, y: 35 },
          }),
        },
      ),
    } as unknown as GuiWindow;
    const out = outputWindow(entry);
    expect(out.node.background).toBeNull();
    expect(
      out.node.children.map((child) => [
        child.name,
        child.sprite,
        child.position,
      ]),
    ).toEqual([
      ["Background", "GFX_project_output_bg", { x: -20, y: 35 }],
      ["simple_output_window", undefined, { x: 0, y: 0 }],
    ]);
  });

  it("takes the list's place in the facility window and goes back to it", () => {
    const win = {
      name: "program_window",
      sprites: {},
      texts: {},
      fonts: {},
      node: node("program_window", "containerwindowtype", [
        node("frame", "containerwindowtype"),
        node("projects", "containerwindowtype"),
      ]),
    } as unknown as GuiWindow;
    expect(
      matchNamePath(programWindow(win, null).win.node, "projects"),
    ).toHaveLength(1);
    expect(
      matchNamePath(programWindow(win, null, true).win.node, "projects"),
    ).toHaveLength(0);
    const layer = readFileSync(
      join(__dirname, "ProgramWindowLayer.tsx"),
      "utf8",
    );
    expect(layer).toContain("onClick={() => onOpenPage(project.id)}");
    expect(layer).toContain("onOpenPage && !editMode");
    expect(layer).toContain("onBack={() => onOpenPage?.(null)}");
    const page = readFileSync(join(__dirname, "page.tsx"), "utf8");
    expect(page).toContain(
      "programWindow(win, facilityPick, Boolean(projectPage))",
    );
    expect(page).toContain("onOpenProjectPage={(id) => {");
    expect(page).toContain("if (id) setProjectEdit(id);");
    const layerPage = readFileSync(
      join(__dirname, "ProjectPageLayer.tsx"),
      "utf8",
    );
    expect(layerPage).toContain("/gui-render/special-project?");
  });
});

describe("a special project's page in edit mode", () => {
  const card = {
    id: "sp_naval_super_heavy_battleship",
    label: "Jellicoe Class Battleship",
    label_key: "ENG_sp_naval_super_heavy_battleship",
    breakthrough_cost: [],
    prototype_time: "sp_time.prototype.medium",
    complexity: "sp_complexity.medium",
    allowed_script: "",
    available_script: "",
  } as unknown as GuiSpecialProject;
  const page = {
    description: "The naval arms race may have diversified in recent years…",
    discounts: [
      {
        name: "sp_naval_super_heavy_battleship_free_design",
        equipment: "ship_hull_super_heavy_1",
        label: "Super Heavy Battleship",
        uses: 1,
        cost_factor: 1,
      },
    ],
  };
  const bonus = "sp_naval_super_heavy_battleship_free_design";

  it("sends the description and the discount only when they change", () => {
    expect(
      specialProjectChanges(
        card,
        {
          description: page.description,
          bonus: { [bonus]: { uses: "1", off: "100" } },
        },
        page,
      ),
    ).toEqual({});
    expect(
      specialProjectChanges(
        card,
        {
          description: "A bigger battleship.",
          bonus: { [bonus]: { uses: "2", off: "50" } },
        },
        page,
      ),
    ).toEqual({
      description: "A bigger battleship.",
      design_bonus: { name: bonus, uses: 2, cost_factor: 0.5 },
    });

    expect(
      specialProjectChanges(
        card,
        { bonus: { [bonus]: { off: "12.5" } } },
        page,
      ),
    ).toEqual({
      design_bonus: { name: bonus, uses: 1, cost_factor: 0.125 },
    });
    for (const wrong of [
      { description: " " },
      { bonus: { [bonus]: { uses: "0" } } },
      { bonus: { [bonus]: { off: "120" } } },
      { bonus: { [bonus]: { off: "" } } },
    ]) {
      expect(specialProjectChanges(card, wrong, page)).toBeNull();
    }

    expect(specialProjectChanges(card, { description: "x" })).toEqual({});
  });

  it("opens the page with the panel, and saves the description under the page's key", () => {
    const view = readFileSync(join(__dirname, "page.tsx"), "utf8");
    expect(view).toMatch(/setProjectEdit\(id\);\s+setProjectPage\(id\);/);
    expect(view).toContain(
      "page={editedPage.data?.found ? editedPage.data : null}",
    );
    const panel = readFileSync(
      join(__dirname, "SpecialProjectEditPanel.tsx"),
      "utf8",
    );
    expect(panel).toContain(
      "await guiRenderApi.setText(page.description_key, description);",
    );
  });
});
