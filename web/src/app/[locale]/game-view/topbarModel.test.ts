import { describe, expect, it } from "vitest";
import type {
  GuiCountryStats,
  GuiNode,
  GuiSprite,
  GuiWindow,
} from "@/lib/guiRenderApi";
import { assignIds } from "../gui-renderer/GuiCanvas";
import {
  IDLE_TOPBAR,
  buildTopbarModel,
  topbarExtraSprites,
  type TopbarUi,
} from "./topbarModel";
import { nodesNamed } from "./topbarLayout";

function node(partial: Partial<GuiNode>): GuiNode {
  return {
    type: "icontype",
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

const sprite = (frames = 1): GuiSprite => ({
  resolved: true,
  frames,
  width: 30 * frames,
  height: 30,
  effect: null,
  effect_kind: null,
  cornered: false,
  bordersize: null,
  vanilla_url: "http://store/gfx/interface/x.png",
});

function topbarWindow(): GuiWindow {
  const text = (name: string) =>
    node({ type: "instanttextboxtype", name, text: "999", font: "hoi_18mbs" });
  const btn = (name: string, extra: Partial<GuiNode> = {}) =>
    node({ type: "buttontype", name, ...extra });
  const root = node({
    type: "containerwindowtype",
    name: "topbar",
    size: { width: 1924, height: 1081 },
    children: [
      node({
        type: "containerwindowtype",
        name: "toggle_music_commands_win",
        position: { x: -289, y: 39 },
        raw: { show_position: { x: -450, y: 39 }, animation_time: 450 },
      }),
      text("pol_power"),
      node({ name: "fuel_icon", sprite: "GFX_fuel_state" }),
      text("fuel_value"),
      node({
        name: "fuel_ratio_bar",
        sprite: "GFX_topbar_fuel_bar",
        alwaystransparent: true,
      }),
      node({
        type: "containerwindowtype",
        name: "nukes",
        size: { width: 56, height: 30 },
        children: [text("nukes_value")],
      }),
      btn("threat_button", { sprite: "GFX_world_tension_icon_big_strip" }),
      node({
        type: "containerwindowtype",
        name: "time_control_window",
        size: { width: 253, height: 30 },
        children: [
          btn("pause_button_date", { raw: { shortcut: "SPACE" } }),
          btn("pause_button_date_paused", {
            sprite: "GFX_date_pause_button",
            raw: { shortcut: "SPACE" },
          }),
          btn("button_speeddown", {
            raw: { pdx_tooltip: "SPEED_DECREASE", shortcut: "KP_MINUS" },
          }),
          btn("button_speedstep1"),
          btn("button_speedstep2"),
          btn("button_speedstep3"),
          btn("button_speedstep4"),
          btn("button_speedstep5"),
          btn("button_speedup"),
          text("DateText"),
        ],
      }),
      btn("achievements_button"),
      node({
        type: "containerwindowtype",
        name: "country_menu_window",
        children: [
          btn("faction_button", { raw: { shortcut: "ctrl+q" } }),
          text("initiative_text"),
          node({
            name: "faction_icon",
            sprite: "GFX_faction_logo_generic_democratic_miniature",
            scale: 0.7,
          }),
          btn("decisionview_button", { raw: { shortcut: "shift+q" } }),
          node({ name: "decisionview_amount_to_take_bg", frame: 1 }),
          text("decisionview_amount_to_take_items"),
          btn("diplomacy_button", {
            sprite: "GFX_topbar_diplomacy",
            raw: { shortcut: "e" },
          }),
          btn("construction_button", { raw: { shortcut: "t" } }),
          btn("navy_button", { raw: { shortcut: "p" } }),
        ],
      }),
    ],
  });
  assignIds(root);
  return {
    name: "topbar",
    file: "interface/topbar.gui",
    node: root,
    sprites: {},
    texts: {},
    fonts: {},
    overlap_groups: [],
  };
}

const GERMANY: GuiCountryStats = {
  tag: "GER",
  political_power: 0,
  factories_total: 73,
  fuel: { days: 469, ratio: 0.8, capacity: 371_500, gaining: true },
  supply_ratio: 1,
  world_tension: 42,
  decisions_takeable: [{ id: "GER_cancel_mefos", name: "Cancel MEFO Bills" }],
  decisions_takeable_count: 1,
  faction: {
    name: "Axis",
    icon: "GFX_faction_logo_axis_miniature",
    initiative: 0,
  },
  nukes: 0,
};

const EXTRA = {
  GFX_faction_logo_axis_miniature: sprite(),
  GFX_topbar_international_market: sprite(2),
};

function build(
  stats: GuiCountryStats | undefined,
  ui: Partial<TopbarUi> = {},
  extra: Record<string, GuiSprite> = EXTRA,
) {
  const win = topbarWindow();
  const model = buildTopbarModel({
    win,
    stats,
    date: "1936.1.1.12",
    ui: { ...IDLE_TOPBAR, ...ui },
    loc: {},
    extraSprites: extra,
  });
  const one = (name: string) =>
    nodesNamed(model.win.node, name)[0] as GuiNode & { _id: string };
  return {
    model,
    one,
    isHidden: (name: string) => model.hidden.has(one(name)._id),
  };
}

describe("buildTopbarModel", () => {
  it("leaves the bars, the fuel text and the disabled buttons to TopbarLayer", () => {
    const { isHidden } = build(GERMANY);
    expect(isHidden("fuel_ratio_bar")).toBe(true);
    expect(isHidden("fuel_value")).toBe(true);
    expect(isHidden("achievements_button")).toBe(true);
  });

  it("draws the country's own faction emblem, and nothing outside a faction", () => {
    const member = build(GERMANY);
    expect(member.one("faction_icon").sprite).toBe(
      "GFX_faction_logo_axis_miniature",
    );
    expect(
      member.model.win.sprites.GFX_faction_logo_axis_miniature,
    ).toBeDefined();
    expect(member.isHidden("faction_icon")).toBe(false);
    expect(member.isHidden("initiative_text")).toBe(false);
    const alone = build({ ...GERMANY, faction: null });
    expect(alone.isHidden("faction_icon")).toBe(true);
    expect(alone.isHidden("initiative_text")).toBe(true);

    expect(build(undefined).isHidden("faction_icon")).toBe(true);
  });

  it("swaps the E button to the International Market", () => {
    const { one, model } = build(GERMANY);
    expect(one("diplomacy_button").sprite).toBe(
      "GFX_topbar_international_market",
    );
    expect(model.win.sprites.GFX_topbar_international_market).toBeDefined();

    expect(build(GERMANY, {}, {}).one("diplomacy_button").sprite).toBe(
      "GFX_topbar_diplomacy",
    );
  });

  it("shows the decisions badge only with something to take", () => {
    const some = build(GERMANY);
    expect(some.isHidden("decisionview_amount_to_take_bg")).toBe(false);
    expect(
      some.model.textOverrides[
        some.one("decisionview_amount_to_take_items")._id
      ],
    ).toBe("1");
    const none = build({
      ...GERMANY,
      decisions_takeable: [],
      decisions_takeable_count: 0,
    });
    expect(none.isHidden("decisionview_amount_to_take_bg")).toBe(true);
    expect(none.isHidden("decisionview_amount_to_take_items")).toBe(true);
  });

  it("shows the nuke box only for a country that starts with bombs", () => {
    expect(build(GERMANY).isHidden("nukes")).toBe(true);
    expect(build(undefined).isHidden("nukes")).toBe(true);
    const bombs = build({ ...GERMANY, nukes: 2 });
    expect(bombs.isHidden("nukes")).toBe(false);
    expect(bombs.model.textOverrides[bombs.one("nukes_value")._id]).toBe("2");
  });

  it("picks the run-time frames: rising fuel, the tension globe, the lit pips, the open panel's button", () => {
    const { one } = build(GERMANY, {
      speed: 3,
      litButton: "construction_button",
    });
    expect(one("fuel_icon").frame).toBe(2);
    expect(one("threat_button").frame).toBe(5);
    expect(
      [1, 2, 3, 4, 5].map((pip) => one(`button_speedstep${pip}`).frame),
    ).toEqual([2, 2, 2, 1, 1]);
    expect(one("construction_button").frame).toBe(2);
    expect(one("navy_button").frame).toBeNull();
  });

  it("never lights the globe by swapping its frame: its frames are tension bands", () => {
    const calm = build({ ...GERMANY, world_tension: 0 });
    expect(calm.one("threat_button").frame).toBe(1);
    expect(calm.one("threat_button").type).toBe("icontype");
  });

  it("keeps the clock's buttons and date above the blinking pause box", () => {
    const { model, one, isHidden } = build(GERMANY);
    expect(isHidden("pause_button_date")).toBe(true);
    expect(isHidden("pause_button_date_paused")).toBe(false);

    for (const name of [
      "button_speeddown",
      "button_speedstep1",
      "button_speedup",
      "DateText",
    ]) {
      expect(isHidden(name)).toBe(true);
      expect(model.clock?.hidden.has(one(name)._id)).toBe(false);
    }

    expect(model.clock?.hidden.has(one("pause_button_date_paused")._id)).toBe(
      true,
    );
    expect(model.clock?.hidden.has(one("pol_power")._id)).toBe(true);
    expect(model.clock?.pausedBoxId).toBe(one("pause_button_date_paused")._id);
    expect(model.clock?.sprite).toBe("GFX_date_pause_button");
  });

  it("draws the play box and no blink once the clock runs", () => {
    const { model, isHidden } = build(GERMANY, { paused: false });
    expect(isHidden("pause_button_date")).toBe(false);
    expect(isHidden("pause_button_date_paused")).toBe(true);
    expect(isHidden("DateText")).toBe(false);
    expect(model.clock).toBeNull();
  });

  it("slides the music drawer toward its show_position", () => {
    expect(
      build(GERMANY, { music: 0 }).one("toggle_music_commands_win").position,
    ).toEqual({ x: -289, y: 39 });
    expect(
      build(GERMANY, { music: 1 }).one("toggle_music_commands_win").position,
    ).toEqual({ x: -450, y: 39 });
    expect(
      build(GERMANY, { music: 0.5 }).one("toggle_music_commands_win").position,
    ).toEqual({ x: -369, y: 39 });
  });

  it("writes the numbers and the clock", () => {
    const { model, one } = build(GERMANY);
    expect(model.textOverrides[one("DateText")._id]).toBe("12:00, 1 Jan, 1936");
    expect(model.textOverrides[one("pol_power")._id]).toBe("0");
    expect(model.fuel).toEqual({ text: "1.3 y", color: "G", frame: 2 });
    expect(model.bars.fuel_ratio_bar).toBe(0.8);
  });

  it("lets a .gui pdx_tooltip speak for itself and answers for the rest", () => {
    const win = topbarWindow();
    const model = buildTopbarModel({
      win,
      stats: GERMANY,
      ui: IDLE_TOPBAR,
      loc: {
        TOPBAR_NAVY: "Navy",
        TOPBAR_NAVY_DESC: "See an overview of your fleets.",
        SEPARATION_LINE: "--------------",
        SHORTCUT: "Shortcut: KEY",
      },
      extraSprites: {},
    });
    const find = (name: string) => nodesNamed(model.win.node, name)[0];
    expect(model.tooltipFor(find("button_speeddown"))).toBeUndefined();
    expect(model.tooltipFor(find("navy_button"))).toEqual({
      text: "§HNavy§!\nSee an overview of your fleets.\n--------------\nShortcut: £p£",
      icons: { p: "P" },
    });

    expect(model.tooltipFor(find("construction_button"))).toBeNull();
  });
});

describe("topbarExtraSprites", () => {
  it("asks for the market button and the faction's emblem", () => {
    expect(topbarExtraSprites(GERMANY)).toEqual([
      "GFX_faction_logo_axis_miniature",
      "GFX_topbar_international_market",
    ]);
    expect(topbarExtraSprites(undefined)).toEqual([
      "GFX_topbar_international_market",
    ]);
  });
});
