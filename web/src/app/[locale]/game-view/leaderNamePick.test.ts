import { readFileSync } from "node:fs";
import { join } from "node:path";
import { describe, expect, it } from "vitest";
import type { GuiCountryStats, GuiNode, GuiWindow } from "@/lib/guiRenderApi";
import { assignIds } from "../gui-renderer/GuiCanvas";
import { createPrintedRegistry, withPrintedOver } from "./editContext";
import { editTargetOf, type EditSelection } from "./editModel";
import { matchNamePath } from "./gameNavigation";
import { buildPoliticsModel, POLITICS } from "./politicsModel";
import { leaderNamePrinted, pickedText } from "./textSources";

function box(
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

const idOf = (node: GuiNode) => (node as GuiNode & { _id: string })._id;

function politicsScreen(): GuiWindow {
  const root = box(POLITICS, [
    box("leader_frame", [box("leader_name", [], "leader_name")]),
    box("ideology", [], "ideology"),
    box("faction", [
      box("no_faction"),
      box("in_faction", [box("faction_name")]),
    ]),
  ]);
  assignIds(root);
  return {
    name: POLITICS,
    file: "interface/countrypoliticsview.gui",
    node: root,
    sprites: {},
    texts: { leader_name: { display: "leader_name", kind: "key" } },
    fonts: {},
    overlap_groups: [],
  };
}

describe("a pick of the leader's name on the politics screen", () => {
  const stats = {
    faction: { name: "同盟国", name_key: "allies" },
  } as unknown as GuiCountryStats;
  const shown = buildPoliticsModel({
    win: politicsScreen(),
    stats,
    extraSprites: {},
  }).win;
  const leader = leaderNamePrinted(shown, {
    token: "CHI_chiang_kaishek",
    name: "蒋介石",
  });
  const node = matchNamePath(shown.node, "leader_name")[0];
  const id = idOf(node);

  it("rewrites the leader's name with nothing registered by the layers", () => {
    const picked: EditSelection = {
      owner: POLITICS,
      win: shown,
      id,
      node,
      ...pickedText(id, leader, {}, {}),
      stack: [{ id, node, ...pickedText(id, leader, {}, {}) }],
    };

    const selection = withPrintedOver(picked, createPrintedRegistry());
    const person = {
      editable: true,
      via: "person",
      token: "CHI_chiang_kaishek",
      current: "蒋介石",
    };
    expect(editTargetOf(selection, selection.shown, {}, {}).text).toEqual(
      person,
    );

    expect(selection.stack?.[0]).toMatchObject({
      shown: "蒋介石",
      source: { kind: "person", token: "CHI_chiang_kaishek" },
    });
  });

  it("leaves the other boxes to the run-time texts and their keys", () => {
    const ideology = idOf(matchNamePath(shown.node, "ideology")[0]);
    const source = {
      kind: "key",
      key: "neutrality",
      scope: "allCountries",
    } as const;
    expect(
      pickedText(
        ideology,
        leader,
        { [ideology]: "中立主义" },
        { [ideology]: source },
      ),
    ).toEqual({ shown: "中立主义", source });
    expect(pickedText(ideology, leader, {}, {})).toEqual({
      shown: undefined,
      source: undefined,
    });
  });

  it("has no leader to name before the country's people arrive", () => {
    expect(leaderNamePrinted(shown, undefined)).toEqual({});
  });

  it("is read where the screen's canvas takes the click", () => {
    const page = readFileSync(join(__dirname, "page.tsx"), "utf8");

    expect(page).toContain(
      "useCharacters(projectId, win.name === POLITICS && tag ? tag : null, date ?? null)",
    );
    expect(page).toContain(
      "leaderNamePrinted(shown, politicsPeople.data?.leader)",
    );
    expect(page).toContain(
      "...pickedText(id, leaderPrinted, runtimeOverrides, runtimeSources),",
    );
    expect(page).toContain(
      "...pickedText(entry.id, leaderPrinted, runtimeOverrides, runtimeSources),",
    );
    const layer = readFileSync(join(__dirname, "CharacterLayer.tsx"), "utf8");
    expect(layer).toContain(
      "useCharacters(projectId, wanted ? tag : null, date ?? null)",
    );
  });
});
