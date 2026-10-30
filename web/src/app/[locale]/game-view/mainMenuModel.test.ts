import { readFileSync } from "node:fs";
import { join } from "node:path";
import { describe, expect, it } from "vitest";
import type { GuiNode, GuiSprite, GuiWindow } from "@/lib/guiRenderApi";
import { matchNamePath } from "./gameNavigation";
import {
  LOADING_STEPS,
  MENU_DLCS,
  MENU_DLC_COLUMN,
  MENU_DLC_ITEM,
  MENU_MAIN_PANEL,
  defaultBackground,
  dlcColumnSprites,
  dlcColumnWindow,
  loadScreenWindow,
  loadingStepAt,
  mainMenuWindow,
  mainPanelSize,
  menuActionOf,
  menuBackgroundWindow,
  pickTip,
  socialWindow,
} from "./mainMenuModel";

const node = (name: string, extra: Partial<GuiNode> = {}): GuiNode =>
  ({ type: "buttontype", name, children: [], ...extra }) as unknown as GuiNode;
const win = (
  name: string,
  root: GuiNode,
  sprites: Record<string, Partial<GuiSprite>> = {},
): GuiWindow =>
  ({
    name,
    file: "frontendmainview.gui",
    node: root,
    sprites,
    texts: {},
    fonts: {},
    overlap_groups: [],
  }) as unknown as GuiWindow;

describe("the main menu", () => {
  it("does what the game's buttons do, and says what is not built", () => {
    expect(menuActionOf(node("single_player_button")).kind).toBe(
      "single-player",
    );
    expect(menuActionOf(node("sp_new_button")).kind).toBe("new-game");
    expect(menuActionOf(node("sp_continue_button")).kind).toBe("continue");
    expect(menuActionOf(node("sp_back_button")).kind).toBe("back");
    expect(menuActionOf(node("exit_button")).kind).toBe("exit");
    expect(menuActionOf(node("multi_player_button"))).toEqual({
      kind: "not-built",
      button: "multi_player_button",
      named: true,
    });
  });

  it("stands on the newest DLC's picture for the language, not the base game's last", () => {
    const pictures = [
      { name: "load_taog_ast", url: "u/taog", width: 1920, height: 1440 },
      {
        name: "load_ncns_chi",
        url: "u/chi",
        width: 1920,
        height: 1440,
        locale: "simp_chinese",
      },
      { name: "load_9", url: "u/9", width: 1920, height: 1440 },
    ];
    expect(defaultBackground(pictures)?.name).toBe("load_taog_ast");
    expect(defaultBackground(pictures.slice(1))?.name).toBe("load_9");

    expect(defaultBackground(pictures, "simp_chinese")?.name).toBe(
      "load_ncns_chi",
    );
    expect(defaultBackground(pictures, "russian")?.name).toBe("load_taog_ast");
    const background = win(
      "frontend_background",
      node("frontend_background", {
        type: "containerwindowtype",
        background: { sprite: "GFX_frontend_bg" } as GuiNode,
      }),
      {
        GFX_frontend_bg: {
          vanilla_url: "u/load_5",
          width: 1920,
          height: 1440,
          resolved: true,
        },
      },
    );

    const shown = menuBackgroundWindow(background, pictures);
    expect(shown.node.background?.sprite).toBe("GFX_load_taog_ast");
    expect(shown.sprites.GFX_load_taog_ast.vanilla_url).toBe("u/taog");
    expect(shown.sprites.GFX_load_taog_ast.overridden).toBe(false);

    const newPicture = [
      {
        ...pictures[0],
        url: "/api/go/projects/2/gui-render/sprite?name=GFX_load_taog_ast&v=1",
        overridden: true,
      },
    ];
    expect(
      menuBackgroundWindow(background, newPicture).sprites.GFX_load_taog_ast
        .overridden,
    ).toBe(true);

    const replaced = {
      ...background,
      sprites: {
        GFX_frontend_bg: {
          ...background.sprites.GFX_frontend_bg,
          overridden: true,
        },
      },
    } as GuiWindow;
    expect(
      menuBackgroundWindow(replaced, pictures).sprites.GFX_frontend_bg
        .vanilla_url,
    ).toBe("u/load_5");
  });

  it("grows the main panel to its subscription size and prints the version where the .gui says", () => {
    const panel = win(
      MENU_MAIN_PANEL,
      node(MENU_MAIN_PANEL, {
        type: "containerwindowtype",
        size: { width: 320, height: 450 },
        children: [
          node("subscription_size", {
            type: "positiontype",
            position: { x: 320, y: 575 },
          }),
          node("nudge_button"),
          node("single_player_button_warn", { type: "icontype" }),
          node("version_label", {
            type: "instanttextboxtype",
            text: "ALPHA",
            position: { x: 0, y: 411 },
          }),
          node("version_label_subscription_position", {
            type: "positiontype",
            position: { x: 0, y: 539 },
          }),
        ],
      }),
    );
    expect(mainPanelSize(panel)).toEqual({ width: 320, height: 575 });
    const shown = mainMenuWindow(panel, "1.19.3.0");
    const names = shown.node.children?.map((child) => child.name);
    expect(names).toContain("nudge_button");
    expect(names).not.toContain("single_player_button_warn");
    const label = shown.node.children?.find(
      (child) => child.name === "version_label",
    );
    expect(label?.position).toEqual({ x: 0, y: 539 });
    expect(shown.texts[label?.text ?? ""]?.display).toBe("v1.19.3.0");
  });

  it("puts the social buttons inside the panel, under the subscription widget", () => {
    const social = win(
      "mainmenu_social_buttons",
      node("mainmenu_social_buttons", {
        type: "containerwindowtype",
        orientation: "CENTER",
        origo: "CENTER",
        position: { x: -50, y: 25 },
        children: [
          node("social_view_interface_window", { type: "containerwindowtype" }),
          node("first_row", { type: "containerwindowtype" }),
        ],
      }),
    );
    const placed = socialWindow(
      social,
      { width: 320, height: 575 },
      { width: 1920, height: 1080 },
    );

    expect(placed.node.position).toEqual({ x: 836, y: 730 });
    expect(placed.node.orientation).toBe("UPPER_LEFT");
    expect(placed.node.origo).toBeNull();
    expect(placed.node.children?.map((child) => child.name)).toEqual([
      "first_row",
    ]);
  });
});

describe("the DLC column", () => {
  const column = win(
    MENU_DLC_COLUMN,
    node(MENU_DLC_COLUMN, {
      type: "containerwindowtype",
      orientation: "LOWER_LEFT",
      origo: "LOWER_LEFT",
      position: { x: 5, y: 0 },
      size: { width: 370, height: 684 },
      children: [
        node("potential_dlc_list", {
          type: "gridboxtype",
          position: { x: 0, y: 0 },
        }),
      ],
    }),
  );
  const item = win(
    MENU_DLC_ITEM,
    node(MENU_DLC_ITEM, {
      type: "containerwindowtype",
      size: { width: 50, height: 50 },
      children: [
        node("dlc_button", {
          sprite: "GFX_frontend_banner_dlc018",
          position: { x: -230, y: 0 },
        }),
        node("glowup", { type: "icontype", sprite: "GFX_spirit_glow" }),
        node("owned_tick", {
          sprite: "GFX_BPT_area_defense_setting_tick",
          scale: 0.56,
        }),
        node("enabled_tick", {
          sprite: "GFX_general_check",
          orientation: "center_left",
          position: { x: 28, y: 24 },
        }),
      ],
    }),
    { GFX_spirit_glow: { resolved: true } },
  );
  const sprites = Object.fromEntries(
    dlcColumnSprites().map((name) => [name, { resolved: true } as GuiSprite]),
  );

  it("is a row a DLC, banner and medal its own, stacked up from the bottom as the recording", () => {
    const shown = dlcColumnWindow(column, item, sprites);
    const rows =
      shown.node.children?.filter((child) => child.name === MENU_DLC_ITEM) ??
      [];
    expect(rows).toHaveLength(MENU_DLCS.length);
    expect(
      shown.node.children?.some((child) => child.type === "gridboxtype"),
    ).toBe(false);

    expect(rows[0].position).toEqual({ x: 0, y: 123 });
    expect(rows[1].position?.y).toBe(174);
    expect(rows.at(-1)?.position?.y).toBe(684 - 51);

    expect(
      rows[0].children?.find((part) => part.name === "enabled_tick")
        ?.orientation,
    ).toBe("UPPER_LEFT");
    const first = rows[0].children ?? [];
    expect(first.find((part) => part.name === "owned_tick")?.sprite).toBe(
      "GFX_frontend_medal_dlc028",
    );
    expect(first.find((part) => part.name === "dlc_button")?.sprite).toBe(
      "GFX_frontend_banner_dlc028",
    );
    expect(
      rows.at(-1)?.children?.find((part) => part.name === "owned_tick")?.sprite,
    ).toBe("GFX_frontend_medal_dlc052");
    expect(shown.sprites.GFX_frontend_medal_dlc052).toBeDefined();
    expect(shown.sprites.GFX_spirit_glow).toBeDefined();

    expect(
      new Set(rows.map((row) => (row as GuiNode & { _id?: string })._id)).size,
    ).toBe(rows.length);
  });

  it("leaves out a DLC whose medal the game does not have", () => {
    const shown = dlcColumnWindow(column, item, {
      ...sprites,
      GFX_frontend_medal_dlc051: { resolved: false } as GuiSprite,
    });
    expect(
      shown.node.children?.filter((child) => child.name === MENU_DLC_ITEM),
    ).toHaveLength(MENU_DLCS.length - 1);
  });
});

describe("the loading screen", () => {
  const screen = win(
    "load_screen",
    node("load_screen", {
      type: "windowtype",
      size: { x: 1024, y: 768 } as unknown as GuiNode["size"],
      children: [
        node("status", {
          type: "windowtype",
          children: [
            node("text", { type: "instanttextboxtype", text: "" }),
            node("progressbar", {
              type: "icontype",
              sprite: "GFX_loadingstatus_progress",
            }),
          ],
        }),
        node("tip", {
          type: "windowtype",
          children: [node("text", { type: "instanttextboxtype", text: "" })],
        }),
      ],
    }),
  );

  it("is full screen, reads its lines from the game's keys and leaves the bar to the page", () => {
    const { win: shown, barId } = loadScreenWindow(
      screen,
      { key: "LOADING_GUI", text: "Loading GUI" },
      { key: "LOADING_TIP_3", text: "A tip" },
      { width: 1920, height: 1080 },
    );
    expect(shown.node.size).toEqual({ width: 1920, height: 1080 });
    expect(matchNamePath(shown.node, "status/text")[0].text).toBe(
      "LOADING_GUI",
    );
    expect(matchNamePath(shown.node, "tip/text")[0].text).toBe("LOADING_TIP_3");
    expect(shown.texts.LOADING_TIP_3).toEqual({
      kind: "loc",
      display: "A tip",
    });
    expect(
      matchNamePath(shown.node, "status/progressbar")[0].sprite,
    ).toBeNull();
    expect(barId).not.toBe("");
  });

  it("runs through the engine's steps and picks one of the game's tips", () => {
    expect(loadingStepAt(0)).toBe(LOADING_STEPS[0]);
    expect(loadingStepAt(1)).toBe(LOADING_STEPS[LOADING_STEPS.length - 1]);
    const loc = {
      LOADING_TIP_0: "first",
      LOADING_TIP_1: "second",
      LOADING_TIP_2: "LOADING_TIP_2",
    };
    expect(pickTip(loc, 0)).toEqual({ key: "LOADING_TIP_0", text: "first" });
    expect(pickTip(loc, 0.99)).toEqual({
      key: "LOADING_TIP_1",
      text: "second",
    });
    expect(pickTip({}, 0.5)).toEqual({ key: "", text: "" });
  });

  it("is wired into the page: Play and Continue load, the in-game menu's Quit leads to the menu", () => {
    const page = readFileSync(join(__dirname, "page.tsx"), "utf8");
    expect(page).toContain('startLoading("countrypoliticsview");');
    expect(page).toContain('if (node.name === "menu_quit_button") {');
    expect(page).toContain(
      "const frontendBackdrop = onMainMenu || panel === SCENARIO_PICKER || panel === COUNTRY_PICKER;",
    );
    expect(page).toContain("{dateKnown && !frontendBackdrop && (");
  });
});
