import { readFileSync } from "node:fs";
import { join } from "node:path";
import { describe, expect, it } from "vitest";
import type { GuiNode, GuiWindow } from "@/lib/guiRenderApi";
import {
  NO_INTEL_PICK,
  createdAgencyWindow,
  insigniaWindow,
  intelAgencyWindow,
  intelChoice,
  type GuiIntelAgency,
} from "./IntelAgencyLayer";
import { OPENS_OVER, matchNamePath, restPlacement } from "./gameNavigation";
import {
  INTEL_UPGRADES_WINDOW,
  dotsLeft,
  intelUpgradesWindow,
} from "./IntelUpgradesLayer";
import { insigniaChanges, namesText } from "./IntelAgencyEditPanel";
import { intelUpgradeChanges } from "./IntelUpgradeEditPanel";

const node = (name: string, type: string, children: GuiNode[] = []): GuiNode =>
  ({ name, type, children }) as GuiNode;

const panel = (): GuiWindow =>
  ({
    name: "countryintelligenceagencyview",
    sprites: {},
    texts: {},
    fonts: {},
    node: node("countryintelligenceagencyview", "containerwindowtype", [
      node("name_logo", "containerwindowtype", [
        node("agency_name", "instanttextboxtype"),
      ]),
      node("before_creation_panel", "containerwindowtype", [
        node("creation_button", "buttontype"),
        node("cancel_creation_button", "buttontype"),
        node("creation_progressbar", "containerwindowtype", [
          node("creation_progressbar_content", "icontype"),
          node("creation_progressbar_factories", "instanttextboxtype"),
          node("creation_progressbar_days", "instanttextboxtype"),
        ]),
      ]),
      node("agency_agents", "containerwindowtype", [
        node("total_operative_text", "instanttextboxtype"),
        node("upgrade_progressbar", "containerwindowtype"),
        node("operatives_progressbar", "containerwindowtype"),
        node("operatives_before_agency", "containerwindowtype"),
      ]),
    ]),
  }) as unknown as GuiWindow;

const chooser = (): GuiWindow =>
  ({
    name: "name_logo_agency_selection_window",
    sprites: {},
    texts: {},
    fonts: {},
    node: node("name_logo_agency_selection_window", "containerwindowtype", [
      node("agency_name", "containerwindowtype", [
        node("agency_name_entry", "editboxtype"),
      ]),
      node("logo_list", "containerwindowtype", [
        node("item_grid", "gridboxtype"),
      ]),
    ]),
  }) as unknown as GuiWindow;

const idOf = (n: GuiNode | undefined) =>
  (n as GuiNode & { _id?: string })?._id ?? "";

const germany: GuiIntelAgency = {
  tag: "GER",
  insignia: [
    {
      file: "common/intelligence_agencies/00_intelligence_agencies.txt",
      ordinal: 4,
      picture: "GFX_intelligence_agency_logo_ger",
      names: ["Abwehr"],
      raw_names: ["Abwehr"],
      default: true,
      available_script: "original_tag = GER",
      default_script: "tag = GER",
    },
    {
      file: "common/intelligence_agencies/00_intelligence_agencies.txt",
      ordinal: 61,
      picture: "GFX_intelligence_agency_logo_generic_2",
      names: ["Tower"],
      raw_names: ["Tower"],
      default: false,
      available_script: "always = yes",
      default_script: "",
    },
  ],
  chosen: 0,
  name: "Abwehr",
  branches: [
    {
      id: "branch_intelligence",
      label: "Intelligence",
      upgrades: [
        { id: "upgrade_economy_civilian" },
        { id: "upgrade_army_department" },
      ],
    },
  ],
  creation: { factories: 5, days: 30, days_text: "§Y30§! days" },
  texts: {
    create: "Create Agency",
    forming: "Agency being formed...",
    choose: "Choose Insignia",
  },
  sprites: {},
};

describe("the intelligence agency before it exists (情报开头 0:00–0:12)", () => {
  it("shows what creating it takes and none of the agency's own parts", () => {
    const out = intelAgencyWindow(panel(), germany, false);
    const id = (path: string) => idOf(matchNamePath(out.win.node, path)[0]);
    for (const path of [
      "name_logo",
      "creation_progressbar_content",
      "upgrade_progressbar",
      "operatives_progressbar",
      "cancel_creation_button",
    ]) {
      expect(out.hidden).toContain(id(path));
    }
    expect(out.hidden).not.toContain(id("operatives_before_agency"));
    expect(out.texts[id("creation_progressbar_factories")]).toBe("5");
    expect(out.texts[id("creation_progressbar_days")]).toBe("§Y30§! days");
    expect(out.texts[id("creation_button")]).toBe("Create Agency");
    expect(out.texts[id("total_operative_text")]).toBe("0/0");
  });

  it("keeps an older payload without creation metadata out of the render crash path", () => {
    const legacy = {
      ...germany,
      creation: undefined,
      texts: undefined,
    } as unknown as GuiIntelAgency;
    const out = intelAgencyWindow(panel(), legacy, false);
    const id = (path: string) => idOf(matchNamePath(out.win.node, path)[0]);
    expect(out.texts[id("creation_progressbar_factories")]).toBeUndefined();
    expect(out.texts[id("creation_progressbar_days")]).toBeUndefined();
    expect(out.texts[id("total_operative_text")]).toBe("0/0");
  });

  it("reads Agency being formed... with its cross once Create Agency is sent", () => {
    const out = intelAgencyWindow(panel(), germany, true);
    const id = (path: string) => idOf(matchNamePath(out.win.node, path)[0]);
    expect(out.texts[id("creation_button")]).toBe("Agency being formed...");
    expect(out.hidden).not.toContain(id("cancel_creation_button"));
  });

  it("starts Choose Insignia on the country's default and keeps what the player typed", () => {
    expect(intelChoice(germany, NO_INTEL_PICK)).toEqual({
      insignia: 0,
      name: "Abwehr",
    });

    expect(intelChoice(germany, { insignia: 1, name: null })).toEqual({
      insignia: 1,
      name: "Abwehr",
    });
    expect(intelChoice(germany, { insignia: 1, name: "SSS" })).toEqual({
      insignia: 1,
      name: "SSS",
    });
    const out = insigniaWindow(chooser(), germany, {
      insignia: 1,
      name: "SSS",
    });
    expect(
      out.texts[
        idOf(matchNamePath(out.win.node, "agency_name/agency_name_entry")[0])
      ],
    ).toBe("SSS");
  });

  it("opens Choose Insignia over the panel from the banner", () => {
    expect(OPENS_OVER.name_logo_agency_selection_window).toBe(
      "countryintelligenceagencyview",
    );
    const layer = readFileSync(join(__dirname, "IntelAgencyLayer.tsx"), "utf8");
    expect(layer).toContain(
      'hit(panelAt["before_creation_panel/creation_button"], "Create Agency", onOpenInsignia',
    );
    expect(layer).toContain(
      "frames={at === choice.insignia ? logoFrames.chosen : logoFrames.plain}",
    );
    const page = readFileSync(join(__dirname, "page.tsx"), "utf8");
    expect(page).toContain("intelAgencyWindow(win, intel.data, intelForming)");
    expect(page).toContain("setPanel(INSIGNIA_WINDOW);");
  });
});

describe("an insignia's edit panel", () => {
  const abwehr = germany.insignia[0];
  it("sends only what changed, names a line each", () => {
    expect(namesText({ ...abwehr, raw_names: ["MI6", "SIS"] })).toBe(
      "MI6\nSIS",
    );
    expect(insigniaChanges(abwehr, {})).toEqual({});
    expect(insigniaChanges(abwehr, { names: "Abwehr" })).toEqual({});
    expect(
      insigniaChanges(abwehr, { names: "Abwehr\n Amt Ausland \n" }),
    ).toEqual({ names: ["Abwehr", "Amt Ausland"] });

    expect(insigniaChanges(abwehr, { available: "  " })).toEqual({
      available: "",
    });
    expect(insigniaChanges(abwehr, { default: "tag = GER" })).toEqual({});
    expect(insigniaChanges(abwehr, { names: "\n" })).toBeNull();
    expect(insigniaChanges(abwehr, { names: 'Ab"wehr' })).toBeNull();
  });

  it("opens from an insignia in edit mode and saves through its routes", () => {
    const layer = readFileSync(join(__dirname, "IntelAgencyLayer.tsx"), "utf8");
    expect(layer).toContain('label={tLayer("insignia", { name: label })}');
    const page = readFileSync(join(__dirname, "page.tsx"), "utf8");
    expect(page).toContain(
      "{editMode && !countryPanelOpen && tag && panel === INSIGNIA_WINDOW && editedInsignia && !selection && (",
    );
    const panel = readFileSync(
      join(__dirname, "IntelAgencyEditPanel.tsx"),
      "utf8",
    );
    expect(panel).toContain('"intel-agency-save"');
    expect(panel).toContain('"intel-agency-create"');
  });
});

describe("an agency the history has set up (情报结尾 0:06, Germany 1939)", () => {
  const created = (): GuiWindow =>
    ({
      name: "countryintelligenceagencyview",
      sprites: {},
      texts: {},
      fonts: {},
      node: node("countryintelligenceagencyview", "containerwindowtype", [
        node("name_logo", "containerwindowtype", [
          node("agency_logo", "buttontype"),
          node("agency_name", "instanttextboxtype"),
          node("agency_defense_level", "instanttextboxtype"),
          node("spymaster", "containerwindowtype", [
            node("spymaster_icon", "icontype"),
            node("spymaster_button", "buttontype"),
          ]),
        ]),
        node("before_creation_panel", "containerwindowtype", [
          node("creation_button", "buttontype"),
        ]),
        node("agency_agents", "containerwindowtype", [
          node("total_operative_text", "instanttextboxtype"),
          node("upgrade_progressbar", "containerwindowtype", [
            node("progressbar_content", "icontype"),
            node("progressbar_desc", "instanttextboxtype"),
            node("progressbar_days", "instanttextboxtype"),
            node("cancel_upgrade", "buttontype"),
          ]),
          node("operatives_progressbar", "containerwindowtype", [
            node("progressbar_content", "icontype"),
            node("progressbar_desc", "instanttextboxtype"),
            node("progressbar_days", "instanttextboxtype"),
          ]),
          node("operatives_before_agency", "containerwindowtype"),
        ]),
      ]),
    }) as unknown as GuiWindow;
  const abwehr: GuiIntelAgency = {
    ...germany,
    created: true,
    operative_slots: 1,
    texts: {
      ...germany.texts,
      defense_level: "Counter Intelligence: §H0.00§!",
      no_upgrade: "No upgrade in progress.",
      recruiting: "Recruiting operative",
      recruit_days: "30 days",
    },
    sprites: { GFX_intelligence_agency_logo_ger: { resolved: true } as never },
  };

  it("opens with its insignia and name, nothing under way, the first operative being recruited", () => {
    const out = intelAgencyWindow(created(), abwehr, false);
    const at = (path: string) => matchNamePath(out.win.node, path)[0];
    const text = (path: string) => out.texts[idOf(at(path))];
    expect(at("name_logo/agency_logo").sprite).toBe(
      "GFX_intelligence_agency_logo_ger",
    );
    expect(out.win.sprites.GFX_intelligence_agency_logo_ger).toBeDefined();
    expect(text("agency_name")).toBe("Abwehr");
    expect(text("agency_defense_level")).toBe("Counter Intelligence: §H0.00§!");
    expect(text("upgrade_progressbar/progressbar_desc")).toBe(
      "No upgrade in progress.",
    );
    expect(text("operatives_progressbar/progressbar_desc")).toBe(
      "Recruiting operative",
    );
    expect(text("operatives_progressbar/progressbar_days")).toBe("30 days");
    expect(text("total_operative_text")).toBe("0/1");
    for (const path of [
      "before_creation_panel",
      "operatives_before_agency",
      "spymaster/spymaster_button",
      "upgrade_progressbar/cancel_upgrade",
      "operatives_progressbar/progressbar_content",
    ]) {
      expect(out.hidden).toContain(idOf(at(path)));
    }
    expect(out.hidden).not.toContain(idOf(at("name_logo")));
    expect(out.hidden).not.toContain(idOf(at("spymaster/spymaster_icon")));
    expect(createdAgencyWindow(created(), abwehr).texts).toEqual(out.texts);
  });

  it("draws a pip on each branch card for each of its upgrades", () => {
    const layer = readFileSync(join(__dirname, "IntelAgencyLayer.tsx"), "utf8");
    expect(layer).toContain(
      "(Array.isArray(branch.upgrades) ? branch.upgrades : []).map((upgrade, index) => (",
    );
    expect(layer).toContain(
      "{!agency.created && veil && veilAt && veilHeight > 0 && (",
    );
  });
});

describe("Agency Branch Upgrades (情报结尾 0:08)", () => {
  const window = (): GuiWindow =>
    ({
      name: INTEL_UPGRADES_WINDOW,
      sprites: {},
      texts: {},
      fonts: {},
      node: {
        ...node(INTEL_UPGRADES_WINDOW, "containerwindowtype", [
          {
            ...node("max_window_size", "positiontype"),
            position: { x: 1092, y: 789 },
          } as GuiNode,
          node("upgrade_duration_title", "instanttextboxtype"),
          {
            ...node("branches", "containerwindowtype", [
              node("branches_grid", "gridboxtype"),
            ]),
            position: { x: 4, y: 75 },
            size: { width: 1084, height: -1 },
          } as GuiNode,
        ]),
        size: { width: 1092, height: "90%" },
        orientation: "center",
        origo: "center",
        position: { x: 0, y: 0 },
      } as GuiNode,
    }) as unknown as GuiWindow;

  it("is 90% of the screen high but no more than 789, its list to 1 above the bottom, with the duration line", () => {
    const out = intelUpgradesWindow(window(), {
      ...germany,
      texts: {
        ...germany.texts,
        upgrades_duration: "Branch Upgrades take §H30§! days to complete.",
      },
    });
    expect(out.win.node.size?.height).toBe(789);
    expect(matchNamePath(out.win.node, "branches")[0].size?.height).toBe(
      789 - 75 - 1,
    );
    expect(
      out.texts[idOf(matchNamePath(out.win.node, "upgrade_duration_title")[0])],
    ).toBe("Branch Upgrades take §H30§! days to complete.");

    expect(restPlacement(window().node).y).toBeCloseTo(540 - 789 / 2);
    expect(restPlacement(window().node).x).toBeCloseTo(960 - 1092 / 2);
    expect(OPENS_OVER[INTEL_UPGRADES_WINDOW]).toBe(
      "countryintelligenceagencyview",
    );
  });

  it("centres an upgrade's level dots in their box", () => {
    expect(dotsLeft(4)).toBe(37 + 25);
    expect(dotsLeft(2)).toBe(37 + 38);
  });

  it("opens from a branch card of an agency that exists, and darkens what may not start", () => {
    const layer = readFileSync(join(__dirname, "IntelAgencyLayer.tsx"), "utf8");
    expect(layer).toContain("{agency.created && onOpenUpgrades && hit(");
    const upgrades = readFileSync(
      join(__dirname, "IntelUpgradesLayer.tsx"),
      "utf8",
    );
    expect(upgrades).toContain(
      "upgrade_background: item.available === false ? LOCKED_BUTTON : undefined,",
    );
    const page = readFileSync(join(__dirname, "page.tsx"), "utf8");
    expect(page).toContain(
      "onOpenIntelUpgrades={() => setPanel(INTEL_UPGRADES_WINDOW)}",
    );
  });
});

describe("an agency upgrade's edit panel", () => {
  const passive = {
    id: "upgrade_passive_defense",
    label: "Passive defense",
    factories: 5,
    available_script: "",
    levels_script: ["modifier = { a = 1 }", "modifier = { a = 2 }"],
  };

  it("sends only what changed: the factories, the condition, every level", () => {
    expect(
      intelUpgradeChanges(passive, {
        name: "Passive defense",
        factories: "5",
        available: " ",
      }),
    ).toEqual({});
    expect(
      intelUpgradeChanges(passive, {
        factories: "8",
        available: "has_tech = radio",
      }),
    ).toEqual({ factories: 8, available: "has_tech = radio" });
    expect(
      intelUpgradeChanges(passive, { levels: ["modifier = { a = 1 }"] }),
    ).toEqual({ levels: ["modifier = { a = 1 }"] });
    expect(
      intelUpgradeChanges(passive, {
        levels: ["modifier = { a = 1 }", " modifier = { a = 2 } "],
      }),
    ).toEqual({});
    for (const wrong of [
      { name: " " },
      { factories: "" },
      { factories: "120" },
      { levels: [] },
      { levels: ["x = 1", " "] },
    ]) {
      expect(intelUpgradeChanges(passive, wrong)).toBeNull();
    }
  });

  it("opens from an upgrade in edit mode and saves the name under the upgrade's id", () => {
    const layer = readFileSync(
      join(__dirname, "IntelUpgradesLayer.tsx"),
      "utf8",
    );
    expect(layer).toContain(
      'label={tLayer("upgrade", { name: item.label ?? item.id })}',
    );
    const page = readFileSync(join(__dirname, "page.tsx"), "utf8");
    expect(page).toContain(
      "{editMode && !countryPanelOpen && panel === INTEL_UPGRADES_WINDOW && editedUpgrade && !selection && (",
    );
    const panel = readFileSync(
      join(__dirname, "IntelUpgradeEditPanel.tsx"),
      "utf8",
    );
    expect(panel).toContain("await guiRenderApi.setText(upgrade.id, name);");
    expect(panel).toContain("/gui-render/intel-upgrade-save");
  });
});
