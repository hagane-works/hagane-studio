import { describe, it, expect } from "vitest";
import { readFileSync } from "node:fs";
import { join } from "node:path";

const source = readFileSync(join(__dirname, "DesignerLayer.tsx"), "utf8");

describe("division designer layer", () => {
  it("places each battalion where the battle order puts it, not where it comes in the list", () => {
    expect(source).toContain("(slot.at?.x ?? slot.x) * slotW");
    expect(source).toContain("(slot.at?.y ?? slot.y) * slotH");
    expect(source).not.toContain("at % columns");
  });

  it("takes each grid's pitch from the .gui and its width from the engine's defines", () => {
    expect(source).toContain(
      'gridGeometry(win.node, "regiments_grid", MAX_DIVISION_BRIGADE_WIDTH)',
    );
    expect(source).toContain(
      'gridGeometry(win.node, "support_grid", MAX_DIVISION_SUPPORT_WIDTH)',
    );
    expect(source).toContain(
      'gridGeometry(win.node, "regiment_support_grid", MAX_DIVISION_BRIGADE_WIDTH)',
    );
    expect(source).toContain("const MAX_DIVISION_BRIGADE_WIDTH = 5;");
    expect(source).toContain("const MAX_DIVISION_SUPPORT_WIDTH = 1;");
    expect(source).not.toContain("Math.floor(width / w)");
    expect(source).not.toContain("cells(data.regiments, regimentRect, 102");
    expect(source).not.toContain("cells(data.support, supportRect, 82");
  });

  it("draws each slot with the box its own entry window declares", () => {
    expect(source).toContain(
      'const REGIMENT_ENTRY = "designer_division_regiment_slot_entry"',
    );
    expect(source).toContain(
      'const SUPPORT_ENTRY = "designer_division_support_slot_entry"',
    );
    expect(source).toContain("slotChrome(regimentEntry.data, CELL_BG)");
    expect(source).toContain("frameStyle(box, chromeOf.at.x, chromeOf.at.y)");

    expect(source).toContain('const CELL_BG = "GFX_div_unit_item_bg_wide"');
  });

  it("hangs each regiment from a spine down to its regimental support slot", () => {
    expect(source).toContain('const CONNECTOR = "rgb(99,106,89)"');
    expect(source).toContain("height: spineTo - rect.y");

    expect(source).toContain(
      'slotView(supportSlots(shownDraft), supportRect, supportGrid, supportChrome, "support")',
    );
  });

  it("draws the empty slots the game draws: the '+', the lock, the doctrine lock", () => {
    expect(source).toContain('add: "GFX_new_regiment_slot"');
    expect(source).toContain('locked: "GFX_locked_icon"');
    expect(source).toContain('blocked: "GFX_blocked_icon"');
    expect(source).toContain("regimentalSupportSlots(shownDraft)");
  });

  it("draws a battalion as its counter alone, where the entry's unit_icon sits", () => {
    expect(source).toContain('find((child) => child.name === "unit_icon")');
    expect(source).toContain("frameStyle(sprite, icon.at.x, icon.at.y, {");
    expect(source).toContain("scale: icon.scale,");
    expect(source).not.toContain("iconStyle(");

    expect(source).toContain(
      'const unnamed = slot.kind === "unit" && !!icons.data && !hasArt;',
    );
    expect(source).not.toContain("sprite?.vanilla_url &&");
  });

  it("always shows the stat lists, with no 3D model selector over them", () => {
    expect(source).not.toContain("toggle_model_selector_button");
    expect(source).not.toContain("modelSelector");
    expect(source).toContain("{statList(statRows(BASE_STATS), baseRect)}");
    expect(source).toContain(
      '<DesignerTerrain projectId={projectId} rect={rects["name:adjuster_stats"]}',
    );
  });

  it("identifies combat width by its box, since two elements share the name", () => {
    expect(source).toContain(
      'rects[hq ? "name:combat_width_view_hq" : "name:combat_width_view"]',
    );
    expect(source).not.toContain('rects["name:combat_width_text"]');
  });

  it("prints the three lists row for row as the game does", () => {
    const order = (list: string) => {
      const body = source.slice(
        source.indexOf(`const ${list}`),
        source.indexOf(";", source.indexOf(`const ${list}`)),
      );
      return [...body.matchAll(/key: "([A-Z_]+)"/g)].map((m) => m[1]);
    };
    expect(order("BASE_STATS")[0]).toBe("STAT_COMMON_MAXIMUM_SPEED");
    expect(order("BASE_STATS").at(-1)).toBe("STAT_ARMY_EXPERIENCE_LOSS_FACTOR");
    expect(order("BASE_STATS")).toContain("STAT_COMMON_RELIABILITY_AVERAGE");
    expect(order("COMBAT_STATS")[0]).toBe("STAT_ARMY_SOFT_ATTACK");
    expect(order("COMBAT_STATS").at(-1)).toBe("DESIGNER_COMBATWIDTH");
    expect(order("COST_STATS")).toEqual([
      "DESIGNER_MANPOWER",
      "DESIGNER_TRAINING_TIME",
      "STAT_ARMY_FUEL_CAPACITY",

      "STAT_COMMON_FUEL_CONSUMPTION",
    ]);

    expect(order("BASE_STATS")).not.toContain("DESIGNER_MANPOWER");
  });

  it("writes numbers the way the game writes them", () => {
    expect(source).toContain("return `${cutFixed(n, 1)} km/h`");
    expect(source).toContain("return `${cutFixed(n * 100, 1)}%`");
    expect(source).toContain("return cutFixed(n, 2)");
    expect(source).toContain('formatStat(header.combat_width, "whole")');
  });
});

describe("designer runtime-hidden elements", () => {
  it("hides the Army HQ layer the .gui says C++ toggles on for HQ templates only", async () => {
    const { RUNTIME_HIDDEN } = await import("./gameNavigation");
    const hidden = RUNTIME_HIDDEN.countrydivisiondesignerview;
    expect(hidden).toContain("subunits_hq");
    expect(hidden).toContain("hq_overlay_images");
    expect(hidden).toContain("hq_title");
  });

  it("hides the sub-dialogs the engine opens only on demand", async () => {
    const { RUNTIME_HIDDEN } = await import("./gameNavigation");
    const hidden = RUNTIME_HIDDEN.countrydivisiondesignerview;
    for (const name of [
      "designerregimentswindow",
      "div_templ_custom_icon_view",
      "div_template_select_model",
      "division_equipment_view",
      "niche_icon_selector",
    ]) {
      expect(hidden).toContain(name);
    }
  });

  it("hides the division's 3D model and the box that opens its picker", async () => {
    const { RUNTIME_HIDDEN } = await import("./gameNavigation");
    const hidden = RUNTIME_HIDDEN.countrydivisiondesignerview;
    expect(hidden).toContain("select_model");
    expect(hidden).toContain("toggle_model_selector_button");
  });

  it("hides the placeholder values the engine writes over", async () => {
    const { RUNTIME_HIDDEN } = await import("./gameNavigation");
    const hidden = RUNTIME_HIDDEN.countrydivisiondesignerview;

    expect(hidden).toContain("hardness_stat/stat_text");
    expect(hidden).toContain("combat_width_view/combat_width_text");
    expect(hidden).toContain("division_names/name");
    expect(hidden).toContain("non_hq_view/div_templ_symbol_button");
  });

  it("keeps the window's own title, 'Division Designer'", async () => {
    const { RUNTIME_HIDDEN } = await import("./gameNavigation");
    expect(RUNTIME_HIDDEN.countrydivisiondesignerview).not.toContain(
      "division_title",
    );
    expect(source).not.toContain("titleRect");
    expect(source).not.toContain('rects["name:division_title"]');

    expect(source).toContain('const view = hq ? "hq_view" : "non_hq_view";');
    expect(source).toContain("at(`${view}/divisions/name`)");

    expect(source).toMatch(
      /text=\{shownDraft\.name \?\? data\.name\}\s+left=\{nameRect\.x\}/,
    );
  });

  it("shows row 1, which the overlap heuristic folds away", async () => {
    const { RUNTIME_HIDDEN, RUNTIME_SHOWN } = await import("./gameNavigation");
    expect(RUNTIME_SHOWN.countrydivisiondesignerview).toEqual([
      "non_hq_view",
      "template_emblem_slot",
    ]);
    const hidden = RUNTIME_HIDDEN.countrydivisiondesignerview;

    expect(hidden).toContain("hq_view");
    expect(hidden).toContain("save_and_deploy_button");
    expect(hidden).toContain("non_hq_view/equipment_warning");
  });

  it("paints the elements whose frame the template decides", async () => {
    const { RUNTIME_HIDDEN } = await import("./gameNavigation");
    const hidden = RUNTIME_HIDDEN.countrydivisiondesignerview;
    for (const name of [
      "non_hq_view/division_priority_icon",
      "non_hq_view/priority_button_1",
      "non_hq_view/priority_button_2",
      "non_hq_view/priority_button_3",
      "non_hq_view/niche_icon_bg",
      "non_hq_view/niche_button",
      "non_hq_view/niche_icon_disabled",
    ]) {
      expect(hidden).toContain(name);
    }

    expect(source).toContain("paint(node, at + 1 + (priority === at ? 3 : 0))");
    expect(source).toContain("paint(filled.priorityIcon, priority + 1)");

    expect(source).toContain("data.priority ?? 1");

    expect(source).toContain("paint(filled.niche.bg, 2)");
  });

  it("prints each stat box's value at its own stat_text offset, in gold", () => {
    expect(source).toContain("at(`${box.node}/stat_text`)");
    expect(source).toContain("rect.x + Number(at?.position?.x ?? 45)");
    expect(source).not.toContain("left={rect.x + 45}");
    expect(source).toContain('color={fontTextColors("hoi_18mbs").Y}');
  });
});

describe("designer save button", () => {
  it("fills the Save button's experience cost: 0 for the template as opened", async () => {
    expect(source).toContain('matchNamePath(win.node, "save_button")');
    expect(source).toContain("fillRuntimeSlots(markup, { VALUE: cost })");
    expect(source).toMatch(
      /export function designerTextOverrides\(\s*win: GuiWindow,\s*cost = 0,\s*experience = 0,/,
    );

    expect(source).toContain("`£army_experience ${experience}`");
  });
});

describe("designer bottom bar", () => {
  const TEXTS = {
    DIVISION_PRODUCTION_COST_VAL:
      "Estimated Production Cost: £production_cost£ $MINVALUE|0$ to £production_cost£ $MAXVALUE|0$",
    DIVISION_PRODUCTION_COST_SINGLE_VAL:
      "Estimated Production Cost: £production_cost£$VALUE|0$",
  };
  const data = (min: number, max: number, equipment = 1) =>
    ({
      equipment: Array.from({ length: equipment }, () => ({
        archetype: "infantry_equipment",
        amount: 100,
      })),
      production_cost: { min, max },
      resources: [
        { resource: "steel", amount: 3, frame: 5 },
        { resource: "rubber", amount: 1, frame: 3 },
      ],
    }) as unknown as import("@/lib/guiRenderApi").GuiDesigner;

  it("prints the range cut to whole numbers, and the resource icons biggest first (designe 0:21, 1:01)", async () => {
    const { designerBarTexts } = await import("./DesignerLayer");
    const bar = designerBarTexts(data(237.7, 247.5), TEXTS);
    expect(bar.value).toBe(
      "Estimated Production Cost: £production_cost£ 237 to £production_cost£ 247",
    );
    expect(bar.resources).toBe("£resources_strip|5 £resources_strip|3");
  });

  it("prints one number when both ends meet, nothing for an empty template (0:31, 0:07)", async () => {
    const { designerBarTexts } = await import("./DesignerLayer");
    expect(designerBarTexts(data(180, 180.4), TEXTS).value).toBe(
      "Estimated Production Cost: £production_cost£180",
    );
    expect(designerBarTexts(data(0, 0, 0), TEXTS)).toEqual({
      value: "",
      resources: "",
    });
  });
});

describe("designer header dropdowns", () => {
  const TEXTS = {
    DIVISION_NAMES_GROUP_LIST_HEADER: "§HExample names:§!",
    DIVISION_NAMES_GROUP_LIST_ENTRY: "\n  $NAME$",
    DIVISION_NAMES_GROUP_LIST_ENTRY_ETC: "\n  ...",
    DIVISION_NAMES_GROUP_LIST_ENTRY_FALLBACK:
      "\n\n§HFallback name:§!\n  $NAME$",
  };

  it("writes a names group's tooltip the way the game does (designe 0:11)", async () => {
    const { namesGroupTooltip } = await import("./DesignerLayer");
    const light = {
      key: "GER_LTARM_01",
      name: "Light Divisions",
      examples: ["4. Leichte-Division", "1. Leichte-Division"],
      fallback: "123. Leichte-Division",
    };
    expect(namesGroupTooltip(light, TEXTS)).toBe(
      "§HExample names:§!\n  4. Leichte-Division\n  1. Leichte-Division\n\n§HFallback name:§!\n  123. Leichte-Division",
    );
    expect(
      namesGroupTooltip({ ...light, more: true, fallback: undefined }, TEXTS),
    ).toMatch(/Leichte-Division\n {2}\.\.\.$/);
  });

  it("names each priority button the way its tooltip does (designe 0:08)", async () => {
    const { priorityTooltip } = await import("./DesignerLayer");
    const texts = {
      TEMPLATE_PRIO_0: "Reserves",
      TEMPLATE_PRIO_0_DESC:
        "Units with this template will receive equipment last.",
    };
    expect(priorityTooltip(0, texts)).toBe(
      "§YReserves§!\nUnits with this template will receive equipment last.",
    );
    expect(priorityTooltip(1, texts)).toBe("");
  });

  it("opens row 1's list on the templates and row 2's on the names groups", () => {
    expect(source).toContain('entryName="designer_division_entry"');
    expect(source).toContain('entryName="designer_div_name_group_entry"');

    expect(source).toMatch(
      /key: NEW_TEMPLATE,\s*label: listText\[LIST_TEXTS\.create\]/,
    );

    expect(source).toContain(
      "onDraft?.({ ...shownDraft, namesGroup: key || null })",
    );

    expect(source).toContain(
      "const header = empty && data.header_stats ? data.header_stats : data.stats;",
    );
  });
});

describe("designer header boxes", () => {
  it("title a box's tooltip with its stat and explain it (designe 0:21)", async () => {
    const { statBoxTooltip } = await import("./DesignerLayer");
    const texts = {
      STAT_COMMON_MAX_ORG: "Organization: ",
      STAT_COMMON_MAX_ORG_DESC:
        "Organization indicates combat readiness and how organized a unit is.",
    };
    expect(statBoxTooltip("STAT_COMMON_MAX_ORG", texts)).toBe(
      "§YOrganization§!\nOrganization indicates combat readiness and how organized a unit is.",
    );
    expect(statBoxTooltip("STAT_ARMY_HARDNESS", texts)).toBe("");
  });
});

describe("the Army HQ Designer's bottom bar", () => {
  it("keeps the days and command power the template's own while an entry is previewed (designe 1:58)", async () => {
    const { designerTextOverrides } = await import("./DesignerLayer");
    const node = (name: string, id: string) => ({
      type: "instanttextboxtype",
      name,
      _id: id,
      children: [],
    });
    const win = {
      name: "countrydivisiondesignerview",
      node: {
        type: "windowtype",
        name: "countrydivisiondesignerview",
        children: [node("hq_deploy_time", "t"), node("hq_deploy_cost", "c")],
      },
      texts: {},
    } as unknown as import("@/lib/guiRenderApi").GuiWindow;
    const hq = (days: number, power: number) =>
      ({
        is_army_hq: true,
        deploy_days: days,
        command_power: power,
        equipment: [],
      }) as unknown as import("@/lib/guiRenderApi").GuiDesigner;
    const out = designerTextOverrides(win, 50, 0, {
      data: hq(6, 16),
      current: hq(5, 15),
      texts: { HQ_TEMPLATE_DEPLOY_DAYS_SHORT: "§Y$DAYS$§! days" },
    });
    expect(out.t).toBe("§Y5§! days");
    expect(out.c).toBe("15");
  });
});

describe("the designer's numbers", () => {
  it("cut rather than round, as the game prints them (designe 0:24, 1:06, 1:41)", async () => {
    const { cutFixed } = await import("./DesignerLayer");
    expect(cutFixed(6.2675, 1)).toBe("6.2");
    expect(cutFixed(13.599, 1)).toBe("13.5");
    expect(cutFixed(50.947, 1)).toBe("50.9");
    expect(cutFixed(46.6, 1)).toBe("46.6");
    expect(cutFixed(0.3, 2)).toBe("0.30");
    expect(cutFixed(-0.5, 1)).toBe("-0.5");
    expect(cutFixed(0, 1)).toBe("0.0");
  });
});

describe("the designer's Save", () => {
  it("types a new name into row 1's box, and a name back to the template's own is no change", () => {
    expect(source).toContain('aria-label="rename_template"');
    expect(source).toContain("name: next === data.name ? undefined : next");
    expect(source).toContain("text={shownDraft.name ?? data.name}");
  });

  it("saves through the project on the Save button and reopens the template under its saved name", async () => {
    const { readFileSync } = await import("node:fs");
    const { join } = await import("node:path");
    const page = readFileSync(join(__dirname, "page.tsx"), "utf8");
    expect(page).toContain('node.name === "save_button"');
    expect(page).toContain("guiRenderApi.saveDesigner(input)");

    expect(page).toContain(
      "(designHQ ? onEditHQTemplate : onEditTemplate)?.(result.template.template_name)",
    );
    expect(page).toContain("editing: true,");
  });
});
