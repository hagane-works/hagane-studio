import { describe, expect, it } from "vitest";
import type { GuiBitmapFont, GuiNode, GuiWindow } from "@/lib/guiRenderApi";
import { assignIds } from "../gui-renderer/GuiCanvas";
import { OPENS_OVER, RUNTIME_HIDDEN, matchNamePath } from "./gameNavigation";
import {
  LEADER_DETAILS,
  fitLine,
  leaderDetailsWindow,
  leaderExclusivePairs,
  leaderTreeLines,
  leftColumn,
  type GuiLeaderDetails,
  type GuiLeaderTreeTrait,
} from "./LeaderDetailsLayer";

function tree(
  token: string,
  column: number,
  row: number,
  parents: string[] = [],
  exclusive: string[] = [],
): GuiLeaderTreeTrait {
  return {
    token,
    label: token,
    icon: `GFX_trait_${token}`,
    column,
    row,
    parents,
    exclusive,
    held: false,
  };
}

const RUNDSTEDT: GuiLeaderDetails = {
  tag: "GER",
  token: "GER_gerd_von_rundstedt",
  name: "Gerd von Rundstedt",
  rank_label: "General",
  skills: {
    skill: 4,
    attack_skill: 5,
    defense_skill: 3,
    planning_skill: 5,
    logistics_skill: 2,
  },
  advisor: {
    token: "army_regrouping_2",
    label: "Army Regrouping(Expert)",
    sprite: 5,
    effects: ["Division Recovery Rate: +8.0%"],
  },
  background: [
    {
      token: "old_guard",
      label: "Old Guard",
      icon: "GFX_trait_old_guard",
      effects: ["a", "b"],
    },
    {
      token: "brilliant_strategist",
      label: "Brilliant Strategist",
      icon: "GFX_trait_brilliant_strategist",
      effects: ["a", "b"],
    },
    {
      token: "trait_cautious",
      label: "Cautious",
      icon: "GFX_trait_trait_cautious",
      effects: ["a", "b", "c"],
    },
  ],
  traits: [],
  pick_slots: 2,
};

describe("the Leader Details window", () => {
  it("opens over the commander roster, without the parts the recording does not show", () => {
    expect(OPENS_OVER[LEADER_DETAILS]).toBe("armyleaderwindow");
    expect(RUNTIME_HIDDEN[LEADER_DETAILS]).toEqual(
      expect.arrayContaining([
        "advisor_traits_window",
        "preferred_tactic_tiling_container",
        "wiki_button",
      ]),
    );
  });

  it("stacks the left column as the recording does", () => {
    const column = leftColumn(
      RUNDSTEDT,
      (effects) => effects.length + (effects.length === 3 ? 2 : 0),
    );
    const at = column.map(
      (item) => `${item.name ?? item.trait?.token}@${item.top}`,
    );

    expect(at).toEqual([
      "officer_corps_title@6",
      "army_regrouping_2@27",
      "personal_traits_title@70",
      "old_guard@96",
      "brilliant_strategist@155",
      "trait_cautious@214",
      "traits_title@321",
      "add_trait_0@339",
      "add_trait_1@381",
    ]);
  });

  it("offers the officer corps role when the person has none, and no Background without traits", () => {
    const names = leftColumn(
      { ...RUNDSTEDT, advisor: undefined, background: [], pick_slots: 0 },
      () => 0,
    ).map((item) => item.name);
    expect(names).toEqual([
      "officer_corps_title",
      "add_advisor_role",
      "traits_title",
    ]);
  });

  it("fills the frame: the Traits tab and page, the person, the column parts hidden", () => {
    const child = (
      name: string,
      children: GuiNode[] = [],
      extra: Partial<GuiNode> = {},
    ) =>
      ({
        type: "containerwindowtype",
        name,
        position: { x: 0, y: 0 },
        children,
        raw: {},
        ...extra,
      }) as unknown as GuiNode;
    const node = child(LEADER_DETAILS, [
      child("traits_tabs", [
        child("leader_traits_tab", [], { type: "buttontype" }),
      ]),
      child("unit_leader", [], {
        type: "icontype",
        sprite: "GFX_leader_unknown",
      }),
      child("name", [], { type: "instanttextboxtype" }),
      child("active_traits", [
        child("officer_corps_title"),
        child("add_trait_0"),
        child("lines"),
      ]),
      child("regular_traits_window"),
    ]);
    assignIds(node);
    const win = {
      name: LEADER_DETAILS,
      node,
      sprites: {},
      texts: {},
      fonts: {},
    } as unknown as GuiWindow;
    const filled = leaderDetailsWindow(win, {
      ...RUNDSTEDT,
      portrait: "GFX_portrait_rundstedt",
      portrait_sprite: { width: 1, height: 1 } as never,
    });
    const find = (path: string) =>
      matchNamePath(filled.win.node, path)[0] as GuiNode & { _id: string };
    expect(find("traits_tabs/leader_traits_tab").frame).toBe(2);
    expect(find("unit_leader").sprite).toBe("GFX_portrait_rundstedt");
    expect(filled.texts[find("name")._id]).toBe("Gerd von Rundstedt");
    expect(filled.hidden).toEqual([
      find("active_traits/officer_corps_title")._id,
      find("active_traits/add_trait_0")._id,
    ]);
    expect(filled.shown).toEqual([find("regular_traits_window")._id]);
  });
});

describe("the trait trees", () => {
  it("runs a line out of each parent's bar, down the bus left of the child's column, into the child", () => {
    const lines = leaderTreeLines([
      tree("panzer_leader", 0, 1),
      tree("panzer_expert", 1, 1, ["panzer_leader"]),
    ]);

    expect(lines).toEqual([
      { x: 160, y: 70, length: 25, vertical: false, dotted: false },
      { x: 185, y: 70, length: 25, vertical: false, dotted: false },
    ]);
  });

  it("dots the lines into a trait that needs any one of several parents", () => {
    const lines = leaderTreeLines([
      tree("panzer_leader", 0, 1),
      tree("cavalry_leader", 0, 3),
      tree("combined_arms_expert", 1, 2, ["cavalry_leader", "panzer_leader"]),
    ]);
    expect(lines.every((line) => line.dotted)).toBe(true);

    expect(
      lines
        .filter((line) => line.vertical)
        .map((line) => [line.x, line.y, line.length]),
    ).toEqual([
      [185, 125, 55],
      [185, 70, 55],
    ]);
  });

  it("marks neighbours in a column that rule each other out, whichever names the other", () => {
    const pairs = leaderExclusivePairs([
      tree("panzer_expert", 1, 1, [], ["cavalry_expert"]),
      tree("combined_arms_expert", 1, 2, [], ["panzer_expert"]),
      tree("cavalry_expert", 1, 3, [], ["combined_arms_expert"]),
    ]);

    expect(
      pairs.map(([upper, lower]) => `${upper.token}/${lower.token}`),
    ).toEqual([
      "panzer_expert/combined_arms_expert",
      "combined_arms_expert/cavalry_expert",
    ]);
  });
});

describe("fitLine", () => {
  const chars: Record<string, number[]> = {};
  for (const ch of "Army Regouping(Expet).")
    chars[String(ch.codePointAt(0))] = [0, 0, 10, 20, 0, 0, 10];
  const font = {
    line: 20,
    base: 16,
    atlas: "",
    atlas_w: 1,
    atlas_h: 1,
    tint: "alpha",
    chars,
  } as GuiBitmapFont;

  it("cuts a name that does not fit with ...", () => {
    expect(fitLine(font, "Army", 170)).toBe("Army");
    expect(fitLine(font, "Army Regrouping(Expert)", 170)).toBe(
      "Army Regroupin...",
    );
  });
});
