import { readFileSync } from "node:fs";
import { join } from "node:path";
import { describe, expect, it } from "vitest";
import type { GuiNode, GuiWindow } from "@/lib/guiRenderApi";
import {
  DOCTRINE_RINGS,
  TRACK_W,
  TRACKS_AREA_W,
  costText,
  doctrineWindow,
  insetWindow,
  itemPlace,
  masteryText,
  pickerPlace,
  rewardFrames,
  rewardWindow,
  trackFill,
  trackGrid,
  trackSpot,
  type GuiDoctrineReward,
  type GuiDoctrines,
} from "./DoctrineLayer";
import { PANEL_TARGETS, matchNamePath } from "./gameNavigation";

const node = (
  name: string,
  type: string,
  extra: Partial<GuiNode> = {},
  children: GuiNode[] = [],
): GuiNode => ({ name, type, children, ...extra }) as GuiNode;

const view = (): GuiWindow =>
  ({
    name: "country_doctrine_view",
    sprites: {},
    texts: {},
    fonts: {},
    node: node("country_doctrine_view", "containerwindowtype", {}, [
      node(
        "doctrine_folder_view",
        "containerwindowtype",
        { raw: { margin: { top: 15, left: 15 } } } as Partial<GuiNode>,
        [
          node("no_grand_doctrine_info", "containerwindowtype", {
            position: { x: 0, y: 0 },
          }),
          node(
            "grand_doctrine",
            "containerwindowtype",
            { position: { x: -100, y: 40 } },
            [
              node("bonus_container", "containerwindowtype"),
              node("grand_doctrine_button", "buttontype"),
              node("grand_doctrine_icon", "icontype"),
              node("grand_doctrine_name", "instanttextboxtype"),
            ],
          ),
          node("tracks_container", "containerwindowtype", {
            position: { x: 0, y: 145 },
            size: { width: 100, height: 100 },
          }),
        ],
      ),
    ]),
  }) as unknown as GuiWindow;

const land = (extra: Partial<GuiDoctrines> = {}): GuiDoctrines => ({
  tag: "GER",
  folder: "land",
  folders: [],
  color_frame: 1,
  grand: {
    token: "new_mobile_warfare",
    label: "Mobile Warfare",
    icon: "GFX_doctrine_mobile_warfare_medium",
  },
  tracks: ["infantry", "combat_support", "armor", "operations"].map(
    (token) => ({
      token,
      label: token,
      background: "",
      icon: "",
      icon_frame: "",
      mastery: 0,
      choices: [],
    }),
  ),
  texts: {
    no_grand: "Select a Grand Doctrine",
    mastery: "£doctrine_mastery $VALUE|H0$",
  },
  sprites: {
    GFX_doctrine_mobile_warfare_medium: { width: 64, height: 64 } as never,
  },
  ...extra,
});

const find = (win: GuiWindow, path: string) => matchNamePath(win.node, path)[0];
const idOf = (n: GuiNode | undefined) =>
  (n as GuiNode & { _id?: string })?._id ?? "";

describe("the doctrine screen", () => {
  it("puts the grand doctrine on the banner in its folder's colour and centres the tracks", () => {
    const out = doctrineWindow(view(), land({ color_frame: 2 }));
    expect(find(out.win, "grand_doctrine_button").frame).toBe(2);
    expect(find(out.win, "grand_doctrine_icon").sprite).toBe(
      "GFX_doctrine_mobile_warfare_medium",
    );
    expect(out.texts[idOf(find(out.win, "grand_doctrine_name"))]).toBe(
      "Mobile Warfare",
    );
    const tracks = find(out.win, "tracks_container");
    expect(tracks.size).toEqual({ width: 4 * TRACK_W, height: 764 });

    expect(tracks.position).toEqual({ x: -2 * TRACK_W, y: 160 });
    expect(find(out.win, "grand_doctrine").position).toEqual({
      x: -100,
      y: 55,
    });
    expect(out.hidden).toContain(idOf(find(out.win, "no_grand_doctrine_info")));
    expect(out.hidden).toContain(idOf(find(out.win, "bonus_container")));
  });

  it("keeps the empty banner with its prompt when the folder has no grand doctrine", () => {
    const out = doctrineWindow(
      view(),
      land({ grand: undefined, tracks: undefined, color_frame: 4 }),
    );
    expect(out.texts[idOf(find(out.win, "grand_doctrine_name"))]).toBe(
      "Select a Grand Doctrine",
    );
    expect(out.hidden).toContain(idOf(find(out.win, "grand_doctrine_icon")));
    expect(out.hidden).toContain(idOf(find(out.win, "tracks_container")));
    expect(out.hidden).not.toContain(
      idOf(find(out.win, "no_grand_doctrine_info")),
    );
    expect(find(out.win, "grand_doctrine_button").frame).toBe(4);
  });

  it("writes the mastery with its icon, the value highlighted", () => {
    expect(masteryText("£doctrine_mastery $VALUE|H0$", 0)).toBe(
      "£doctrine_mastery §H0§!",
    );
    expect(masteryText("£doctrine_mastery $VALUE$", 150)).toBe(
      "£doctrine_mastery 150",
    );
    expect(masteryText(undefined, 200)).toBe("200");
  });

  it("fills a track's column: slot in the folder's colour, emblem dark, no subdoctrine picture yet", () => {
    const track = land().tracks![0];
    const empty = trackFill(track, 3, "£doctrine_mastery $VALUE|H0$");
    expect(empty.frames).toEqual({
      sub_doctrine_button: 3,
      milestone_icon: 2,
      milestone_bg_icon: 2,
    });
    expect(empty.hide).toContain("sub_doctrine/sub_doctrine_icon");
    expect(empty.hide).toContain("milestone/preview_container");
    const taken = trackFill(
      {
        ...track,
        sub: {
          token: "mobile_infantry",
          label: "Mobile Infantry",
          icon: "GFX_x",
        },
        mastery: 200,
      },
      1,
    );
    expect(taken.hide).not.toContain("sub_doctrine/sub_doctrine_icon");
    expect(taken.texts["sub_doctrine/sub_doctrine_name"]).toBe(
      "Mobile Infantry",
    );
  });

  it("opens from the officer corps' rings, on their folder, over the alert rows", () => {
    for (const [ring, folder] of Object.entries(DOCTRINE_RINGS)) {
      expect(PANEL_TARGETS[`countryofficercorpview/${ring}`]).toBe(
        "country_doctrine_view",
      );
      expect(ring).toBe(`${folder}_doctrine_ring`);
    }
    const corps = readFileSync(
      join(__dirname, "OfficerCorpsLayer.tsx"),
      "utf8",
    );
    expect(corps).toContain("button.name = `${folder}_doctrine_ring`");
    const page = readFileSync(join(__dirname, "page.tsx"), "utf8");
    expect(page).toContain(
      "new Set([DOCTRINE_VIEW, TECH_TREE, DEPLOYMENT_VIEW])",
    );
    expect(page).toContain(
      'setDoctrineFolder(DOCTRINE_RINGS[node.name ?? ""] ?? null)',
    );
  });
});

describe("the doctrine picker", () => {
  it("writes the cost in red when the country has less of that experience", () => {
    const doctrines = land({
      texts: {
        army_cost: "£army_experience $COST|H$",
        navy_cost: "£navy_experience $COST$",
      },
      experience: { army: 0, navy: 120 },
    });
    expect(
      costText(doctrines, { token: "a", label: "A", icon: "", xp_cost: 100 }),
    ).toBe("£army_experience §R100§!");
    expect(
      costText(doctrines, {
        token: "b",
        label: "B",
        icon: "",
        xp_cost: 100,
        xp_type: "navy",
      }),
    ).toBe("£navy_experience 100");
  });

  it("opens right of a track, or left of it where the right would run off the screen", () => {
    expect(pickerPlace(112, 200)).toEqual({ x: 497, y: 410 });
    expect(pickerPlace(1384, 200)).toEqual({ x: 884, y: 410 });
  });

  it("keeps a negative size's inset on both sides, as the engine does", () => {
    const win = {
      name: "doctrine_selection_list",
      sprites: {},
      texts: {},
      fonts: {},
      node: node(
        "doctrine_selection_list",
        "containerwindowtype",
        { size: { width: 560, height: 576 } },
        [
          node("description_view", "containerwindowtype", {
            position: { x: 30, y: 65 },
            size: { width: -30, height: 98 },
          }),
          node(
            "list",
            "containerwindowtype",
            { position: { x: 15, y: 170 }, size: { width: -15, height: -60 } },
            [
              node("list_container", "containerwindowtype", {
                position: { x: 5, y: 0 },
                size: { width: -5, height: "100%%" },
              }),
            ],
          ),
        ],
      ),
    } as unknown as GuiWindow;
    const out = insetWindow(win, 560, 576);
    expect(find(out, "description_view").size).toEqual({
      width: 500,
      height: 98,
    });
    expect(find(out, "list").size).toEqual({ width: 530, height: 346 });
    expect(find(out, "list/list_container").size).toEqual({
      width: 520,
      height: "100%%",
    });

    expect(find(win, "list").size).toEqual({ width: -15, height: -60 });
  });

  it("lays the entries two to a row from the list's corner", () => {
    expect(itemPlace(0)).toEqual({ left: 13, top: 15 });
    expect(itemPlace(1)).toEqual({ left: 263, top: 15 });
    expect(itemPlace(4)).toEqual({ left: 13, top: 165 });
  });

  it("in edit mode writes the choice into the country's history", () => {
    const page = readFileSync(join(__dirname, "page.tsx"), "utf8");
    expect(page).toContain("const patch = doctrinePatch(row, choice);");
    expect(page).toContain(
      "onChoose={editMode ? (choice) => void chooseDoctrine(choice) : undefined}",
    );
  });
});

describe("a subdoctrine's rewards", () => {
  const reward = (
    extra: Partial<GuiDoctrineReward> = {},
  ): GuiDoctrineReward => ({
    token: "cavalry_charge",
    label: "Modernized Cavalry",
    level: 1,
    mastery: 100,
    unlocked: true,
    progress: 1,
    ...extra,
  });

  it("lights a won reward in its folder's colour and greys the rest", () => {
    expect(rewardFrames(reward(), 1)).toEqual({
      reward_frame: 1,
      reward_level_icon: 1,
      level: 1,
      mastery_text_bg: 2,
    });
    expect(
      rewardFrames(reward({ level: 3, unlocked: false, progress: 0 }), 1),
    ).toEqual({
      reward_frame: 2,
      reward_level_icon: 8,
      level: 3,
      mastery_text_bg: 1,
    });

    expect(rewardFrames(reward(), 4).reward_frame).toBe(7);
  });

  it("puts the medallion and grade on the reward's side, the mastery under its bar", () => {
    const win = {
      name: "reward_item",
      sprites: {},
      texts: {},
      fonts: {},
      node: node("reward_item", "containerwindowtype", {}, [
        node("reward_container", "containerwindowtype", {
          position: { x: 56, y: 44 },
        }),
        node("level", "icontype", { position: { x: 36, y: 71 } }),
        node("mastery_text", "instanttextboxtype", {
          position: { x: -40, y: 48 },
        }),
      ]),
    } as unknown as GuiWindow;
    const right = rewardWindow(win, "right");
    expect(find(right, "reward_container").position).toEqual({ x: 60, y: 20 });
    expect(find(right, "level").position).toEqual({ x: 125, y: 64 });
    expect(find(right, "mastery_text").position).toEqual({ x: 14, y: 68 });
    const left = rewardWindow(win, "left");
    expect(find(left, "reward_container").position).toEqual({ x: -94, y: 20 });
    expect(find(left, "level").position).toEqual({ x: -98, y: 64 });
  });

  it("lights the track's emblem once every reward is won", () => {
    const track = land().tracks![0];
    expect(trackFill({ ...track, complete: true }, 1).frames).toMatchObject({
      milestone_icon: 1,
      milestone_bg_icon: 1,
    });
    expect(trackFill({ ...track, complete: false }, 1).frames).toMatchObject({
      milestone_icon: 2,
      milestone_bg_icon: 2,
    });
  });

  it("lays out a track's content 384 wide, its subdoctrine in the frame's middle", () => {
    const win = {
      name: "doctrine_track_item",
      sprites: {},
      texts: {},
      fonts: {},
      node: node(
        "doctrine_track_item",
        "containerwindowtype",
        { size: { width: 424, height: 764 } },
        [
          node(
            "content",
            "containerwindowtype",
            {
              position: { x: 20, y: 80 },
              size: { width: -20, height: "100%%" },
            },
            [
              node("rewards_container", "containerwindowtype", {
                position: { x: 18, y: 180 },
                size: { width: -15, height: -12 },
              }),
            ],
          ),
        ],
      ),
    } as unknown as GuiWindow;
    const out = insetWindow(win, TRACK_W, 764);
    expect(find(out, "content").size).toEqual({ width: 384, height: "100%%" });

    expect(find(out, "content/rewards_container").size).toEqual({
      width: 351,
      height: 572,
    });
  });
});

describe("the tracks' rows", () => {
  it("fits four to a row on the 1920 stage, as the recording shows mobile warfare", () => {
    expect(trackGrid(4, TRACKS_AREA_W)).toEqual({ columns: 4, rows: 1 });

    expect(trackSpot(0, { columns: 4 }, TRACKS_AREA_W)).toEqual({
      x: (TRACKS_AREA_W - 4 * TRACK_W) / 2,
      y: 0,
    });
  });

  it("wraps what a row cannot hold onto the rows below", () => {
    expect(trackGrid(6, TRACKS_AREA_W)).toEqual({ columns: 4, rows: 2 });
    expect(trackSpot(4, { columns: 4 }, TRACKS_AREA_W)).toEqual({
      x: (TRACKS_AREA_W - 4 * TRACK_W) / 2,
      y: 764,
    });

    expect(trackGrid(4, 1600)).toEqual({ columns: 3, rows: 2 });
  });

  it("follows what the grand doctrine scripts", () => {
    expect(trackGrid(4, TRACKS_AREA_W, { max_track_columns: 2 })).toEqual({
      columns: 2,
      rows: 2,
    });
    expect(trackGrid(6, TRACKS_AREA_W, { max_track_rows: 1 })).toEqual({
      columns: 6,
      rows: 1,
    });
    expect(trackGrid(0, TRACKS_AREA_W)).toEqual({ columns: 0, rows: 0 });
  });
});
