import { describe, expect, it } from "vitest";
import type { GuiNode, GuiWindow } from "@/lib/guiRenderApi";
import {
  IDEA_WINDOW,
  advisorChoices,
  ideaWindowTexts,
  type GuiIdeaChoices,
} from "./IdeaPickerLayer";

const node = (name: string, id: string, children: GuiNode[] = []): GuiNode =>
  ({ name, _id: id, children }) as unknown as GuiNode;

const win = (name: string): GuiWindow =>
  ({
    name,
    node: node(name, "root", [
      node("title", "t1"),
      node("switch", "s1"),
      node("ideas", "i1"),
    ]),
    sprites: {},
    texts: {},
    fonts: {},
  }) as unknown as GuiWindow;

const choices: GuiIdeaChoices = {
  group: "mobilization_laws",
  title: "Conscription Law",
  subtitle: "Replace §YLimited Conscription?§!",
  current: "limited_conscription",
  current_label: "Limited Conscription",
  ideas: [],
};

describe("ideaWindowTexts", () => {
  it("writes the slot's name and the service's line over the window's placeholders", () => {
    expect(ideaWindowTexts(win(IDEA_WINDOW), choices)).toEqual({
      t1: "Conscription Law",
      s1: "Replace §YLimited Conscription?§!",
    });
  });

  it("makes no English line of its own when the service sends none", () => {
    const withoutLine = {
      ...choices,
      subtitle: undefined,
      title: "Loi de conscription",
      current_label: "Conscription limitée",
    };
    expect(ideaWindowTexts(win(IDEA_WINDOW), withoutLine)).toEqual({
      t1: "Loi de conscription",
      s1: "",
    });
  });

  it("leaves other windows, and a picker still loading, alone", () => {
    expect(ideaWindowTexts(win("countrypoliticsview"), choices)).toEqual({});
    expect(ideaWindowTexts(win(IDEA_WINDOW), undefined)).toEqual({});
  });
});

describe("advisorChoices", () => {
  const person = (name: string, extra: Record<string, unknown> = {}) =>
    ({
      token: name.toLowerCase(),
      name,
      roles: ["advisor"],
      slot: "political_advisor",
      ...extra,
    }) as never;
  const people = [
    person("Walther Funk", { shown: true, available: true }),
    person("August Thalheimer", { shown: true, available: false }),
    person("Franz Seldte", { shown: true, available: true }),
    person("Fritz Todt", { shown: false }),
    person("Gerd von Rundstedt", {
      slot: "high_command",
      shown: true,
      available: true,
    }),
  ];

  it("lists the seat's people, hireable first, each group by name", () => {
    const choices = advisorChoices(
      people,
      "political_advisor",
      "Political Advisor",
    );
    expect(choices.people?.map((p) => p.name)).toEqual([
      "Franz Seldte",
      "Walther Funk",
      "August Thalheimer",
    ]);
    expect(choices.subtitle).toBe("§YPolitical Advisor§!");
  });

  it("takes the seat's line in the game's language from the service", () => {
    const choices = advisorChoices(
      people,
      "political_advisor",
      "Conseiller politique",
      false,
      "Sélectionner §YConseiller politique§!",
    );
    expect(choices.subtitle).toBe("Sélectionner §YConseiller politique§!");
  });

  it("carries each advisor's hover text to their card", () => {
    const tip = "§YWalther Funk§!\n§HCaptain of Industry§!";
    const choices = advisorChoices(
      [person("Walther Funk", { shown: true, available: true, tooltip: tip })],
      "political_advisor",
      "Political Advisor",
    );
    expect(choices.people?.[0].tooltip).toBe(tip);
  });

  it("adds the hidden ones last, faint, in edit mode", () => {
    const choices = advisorChoices(
      people,
      "political_advisor",
      "Political Advisor",
      true,
    );
    expect(choices.people?.map((p) => [p.name, !!p.dim])).toEqual([
      ["Franz Seldte", false],
      ["Walther Funk", false],
      ["August Thalheimer", false],
      ["Fritz Todt", true],
    ]);
  });
});
