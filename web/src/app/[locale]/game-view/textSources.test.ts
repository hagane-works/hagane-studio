import { describe, expect, it } from "vitest";
import type { GuiNode, GuiWindow } from "@/lib/guiRenderApi";
import { assignIds } from "../gui-renderer/GuiCanvas";
import { createPrintedRegistry, withPrintedOver } from "./editContext";
import type { EditSelection } from "./editModel";
import { matchNamePath } from "./gameNavigation";
import { buildPoliticsModel } from "./politicsModel";
import {
  countryCardPrinted,
  countryPickPrinted,
  countryRowPrinted,
  diplomacyHeaderPrinted,
  eventOptionPrinted,
  eventPrinted,
  fixedTextSources,
  keySource,
  leaderNamePrinted,
  namedPrinted,
  partyNamePrinted,
  personPrinted,
  scenarioPrinted,
  statePrinted,
} from "./textSources";

function node(
  name: string,
  children: GuiNode[] = [],
  text: string | null = null,
): GuiNode {
  return {
    type: children.length ? "containerwindowtype" : "instanttextboxtype",
    name,
    position: { x: 0, y: 0 },
    size: null,
    orientation: null,
    origo: null,
    scale: null,
    sprite: null,
    frame: null,
    text,
    text_source: null,
    font: null,
    format: null,
    maxwidth: null,
    maxheight: null,
    text_color: null,
    centerposition: false,
    alwaystransparent: false,
    background: null,
    children,
  };
}

function windowOf(name: string, children: GuiNode[]): GuiWindow {
  const root = node(name, children);
  assignIds(root);
  return {
    name,
    file: `interface/${name}.gui`,
    node: root,
    sprites: {},
    texts: {},
    fonts: {},
    overlap_groups: [],
  };
}

const idAt = (win: GuiWindow, namePath: string) =>
  (matchNamePath(win.node, namePath)[0] as GuiNode & { _id: string })._id;

describe("where the engine's texts come from", () => {
  it("takes a key the text route accepts, and nothing else", () => {
    expect(keySource("ABK_neutrality_party", "country")).toEqual({
      kind: "key",
      key: "ABK_neutrality_party",
      scope: "country",
    });
    expect(keySource(" fascism ", "allCountries")).toEqual({
      kind: "key",
      key: "fascism",
      scope: "allCountries",
    });

    expect(keySource("Axis Pact", "everywhere")).toBeNull();
    expect(keySource(undefined, "everywhere")).toBeNull();
    expect(keySource("", "everywhere")).toBeNull();
  });

  it("names the fixed keys the engine writes over a .gui's placeholders", () => {
    const win = windowOf("countrypoliticsview", [
      node("active_goal", [node("title", [], "Placeholder")]),
    ]);
    expect(fixedTextSources(win)).toEqual({
      [idAt(win, "active_goal/title")]: {
        kind: "key",
        key: "POLITICS_SELECT_FOCUS",
        scope: "everywhere",
      },
    });
    expect(fixedTextSources(windowOf("nothing_here", [node("x")]))).toEqual({});
  });

  it("names the leader under the box CharacterLayer draws the name over", () => {
    const win = windowOf("countrypoliticsview", [
      node("leader_name", [], "leader_name"),
    ]);
    expect(
      leaderNamePrinted(win, { token: "ABK_boris", name: "波罗斯" }),
    ).toEqual({
      [idAt(win, "leader_name")]: {
        text: "波罗斯",
        source: { kind: "person", token: "ABK_boris" },
      },
    });
    expect(leaderNamePrinted(win, null)).toEqual({});
  });

  it("names a party's line from the country's own key, also for a party it never named", () => {
    const row = node("political_party_info_entry", [
      node("name", [], ""),
      node("color_block"),
    ]);
    assignIds(row);
    const id = (row.children[0] as GuiNode & { _id: string })._id;
    expect(
      partyNamePrinted(row, {
        party: "neutrality",
        label: "阿布哈兹人民大会",
        label_key: "ABK_neutrality_party",
      }),
    ).toEqual({
      [id]: {
        text: "阿布哈兹人民大会",
        source: { kind: "key", key: "ABK_neutrality_party", scope: "country" },
      },
    });

    expect(partyNamePrinted(row, { party: "neutrality" })).toEqual({
      [id]: { text: "neutrality" },
    });
  });

  it("says where the politics screen's lines and faction name come from", () => {
    const win = windowOf("countrypoliticsview", [
      node("ideology", [], "ideology"),
      node("elections", [], "elections"),
      node("faction", [node("in_faction", [node("faction_name", [], "")])]),
    ]);
    const model = buildPoliticsModel({
      win,
      stats: {
        tag: "ABK",
        politics: {
          ruling_party: "neutrality",
          ideology: "中立主义",
          elections: "无选举",
          elections_key: "POLITICS_NOELECTIONS",
        },
        faction: { name: "轴心国", name_key: "Axis", icon: "", initiative: 0 },
      },
      extraSprites: {},
    });
    expect(model.sources[idAt(win, "ideology")]).toEqual({
      kind: "key",
      key: "neutrality",
      scope: "allCountries",
    });
    expect(model.sources[idAt(win, "elections")]).toEqual({
      kind: "key",
      key: "POLITICS_NOELECTIONS",
      scope: "everywhere",
    });
    expect(model.sources[idAt(win, "in_faction/faction_name")]).toEqual({
      kind: "key",
      key: "Axis",
      scope: "everywhere",
    });

    const older = buildPoliticsModel({
      win,
      stats: {
        tag: "ABK",
        politics: { ruling_party: "", ideology: "中立主义" },
      },
      extraSprites: {},
    });
    expect(older.sources).toEqual({});
    expect(older.textOverrides[idAt(win, "ideology")]).toBe("中立主义");
  });
});

describe("what the other screens print over their boxes", () => {
  const person = (token: string) => ({ kind: "person", token }) as const;
  const key = (
    k: string,
    scope: "everywhere" | "country" | "allCountries" = "everywhere",
  ) => ({ kind: "key", key: k, scope }) as const;

  it("names the picked country's details on the selection screen", () => {
    const win = windowOf("gamesetup_interesting_countries_window", [
      node("nation_title"),
      node("leader"),
      node("ideology"),
      node("government"),
      node("party"),
      node("history"),
    ]);
    const printed = countryPickPrinted(
      win,
      {
        tag: "ABK",
        name: "阿布哈兹",
        name_key: "ABK_neutrality",
        ideology: "neutrality",
        ideology_label: "中立主义",
        ruling_party: "neutrality",
        ruling_party_label: "中立主义",
        history: "很久以前…",
        history_key: "ABK_DESC",
        leader: { token: "ABK_boris", name: "波罗斯", roles: [] },
      },
      "中立主义 60%",
    );
    expect(printed[idAt(win, "nation_title")]).toEqual({
      text: "阿布哈兹",
      source: key("ABK_neutrality", "country"),
    });
    expect(printed[idAt(win, "leader")]).toEqual({
      text: "波罗斯",
      source: person("ABK_boris"),
    });
    expect(printed[idAt(win, "ideology")]).toEqual({
      text: "中立主义",
      source: key("neutrality", "allCountries"),
    });
    expect(printed[idAt(win, "party")]).toEqual({
      text: "中立主义 60%",
      source: key("neutrality", "allCountries"),
    });
    expect(printed[idAt(win, "history")]).toEqual({
      text: "很久以前…",
      source: key("ABK_DESC", "country"),
    });
    expect(countryPickPrinted(win, undefined, undefined)).toEqual({});
  });

  it("names a country's card, a list row and the diplomacy header", () => {
    const card = windowOf("country_entry", [node("country_name")]);
    expect(
      countryCardPrinted(card.node, { name: "阿布哈兹", name_key: "ABK" }),
    ).toEqual({
      [idAt(card, "country_name")]: {
        text: "阿布哈兹",
        source: key("ABK", "country"),
      },
    });
    const row = windowOf("diplomacy_row", [node("name")]);
    expect(
      countryRowPrinted(row.node, {
        tag: "ABK",
        name: "阿布哈兹",
        name_key: "ABK",
      })[idAt(row, "name")],
    ).toEqual({
      text: "阿布哈兹",
      source: key("ABK", "country"),
    });
    const header = windowOf("countrydiplomacyview", [
      node("country_name"),
      node("leader_name"),
    ]);
    expect(
      diplomacyHeaderPrinted(
        header,
        { tag: "ABK", name: "阿布哈兹", name_key: "ABK" },
        { name: "波罗斯", token: "ABK_boris" },
      ),
    ).toEqual({
      [idAt(header, "country_name")]: {
        text: "阿布哈兹",
        source: key("ABK", "country"),
      },
      [idAt(header, "leader_name")]: {
        text: "波罗斯",
        source: person("ABK_boris"),
      },
    });
  });

  it("names a scenario's title and text, an event's, and a state's", () => {
    const scenario = windowOf("gamesetup_bookmarks_window", [
      node("bookmark_title"),
      node("bookmark_desc"),
    ]);
    expect(
      scenarioPrinted(scenario, {
        name: "STORM",
        label: "风暴",
        desc: "云",
        desc_key: "STORM_DESC",
        date: "1936.1.1",
        default_country: "GER",
        countries: [],
      }),
    ).toEqual({
      [idAt(scenario, "bookmark_title")]: {
        text: "风暴",
        source: key("STORM"),
      },
      [idAt(scenario, "bookmark_desc")]: {
        text: "云",
        source: key("STORM_DESC"),
      },
    });
    const event = windowOf("EventWindow", [node("Title"), node("Description")]);
    expect(
      eventPrinted(event, {
        title: "阅兵",
        title_key: "e.1.t",
        desc: "队伍走过",
        desc_key: "e.1.d",
      }),
    ).toEqual({
      [idAt(event, "Title")]: { text: "阅兵", source: key("e.1.t") },
      [idAt(event, "Description")]: { text: "队伍走过", source: key("e.1.d") },
    });
    const option = windowOf("event_option_entry", [node("Name")]);
    expect(
      eventOptionPrinted(option.node, { name: "e.1.a", label: "好" })[
        idAt(option, "Name")
      ],
    ).toEqual({ text: "好", source: key("e.1.a") });
    const state = windowOf("countrystateview", [node("state_name")]);
    expect(
      statePrinted(state, { id: 64, name: "勃兰登堡" })[
        idAt(state, "state_name")
      ],
    ).toEqual({ text: "勃兰登堡", source: key("STATE_64") });
  });

  it("names a row's name from a key, or a person, and nothing without a text", () => {
    const row = windowOf("row", [node("name")]);
    expect(
      namedPrinted(row.node, "name", "统一税制", "tax_law", "everywhere")[
        idAt(row, "name")
      ],
    ).toEqual({ text: "统一税制", source: key("tax_law") });
    expect(
      personPrinted(row.node, "name", { token: "GER_x", name: "某人" })[
        idAt(row, "name")
      ],
    ).toEqual({ text: "某人", source: person("GER_x") });
    expect(namedPrinted(row.node, "name", "", "tax_law")).toEqual({});

    expect(
      namedPrinted(row.node, "name", "Axis Pact", "Axis Pact")[
        idAt(row, "name")
      ],
    ).toEqual({ text: "Axis Pact" });
  });
});

describe("the registry of texts layers print over a screen", () => {
  const win = windowOf("countrypoliticsview", [
    node("leader_name", [], "leader_name"),
    node("ideology"),
  ]);
  const pick = (
    id: string,
    extra: Partial<EditSelection> = {},
  ): EditSelection => ({
    owner: "panel",
    win,
    id,
    node: matchNamePath(win.node, "leader_name")[0],
    ...extra,
  });

  it("fills a pick of a box a layer draws over, and leaves the canvas's own texts alone", () => {
    const registry = createPrintedRegistry();
    const leader = idAt(win, "leader_name");
    registry.publish(
      "countrypoliticsview",
      "characters",
      leaderNamePrinted(win, { token: "ABK_boris", name: "波罗斯" }),
    );
    expect(withPrintedOver(pick(leader), registry)).toMatchObject({
      shown: "波罗斯",
      source: { kind: "person", token: "ABK_boris" },
    });

    expect(
      withPrintedOver(pick(leader, { shown: "其他" }), registry),
    ).toMatchObject({ shown: "其他" });
    expect(
      withPrintedOver(
        pick(leader, { stack: [{ id: leader, node: win.node }] }),
        registry,
      ).stack?.[0],
    ).toMatchObject({ shown: "波罗斯" });

    expect(
      withPrintedOver(
        { ...pick(leader), win: { ...win, name: "other" } },
        registry,
      ).shown,
    ).toBeUndefined();

    registry.publish("countrypoliticsview", "characters", null);
    expect(withPrintedOver(pick(leader), registry).shown).toBeUndefined();
  });
});
