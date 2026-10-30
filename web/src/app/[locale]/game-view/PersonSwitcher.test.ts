import { describe, expect, it } from "vitest";
import type { GuiCharacter } from "@/lib/guiRenderApi";
import { commanderGroups, peopleGroups } from "./PersonSwitcher";

const person = (token: string, name: string, roles: string[]): GuiCharacter =>
  ({ token, name, roles }) as unknown as GuiCharacter;

const commanders = [
  person("GER_heinz_guderian", "Heinz Guderian", [
    "advisor",
    "corps_commander",
  ]),
  person("GER_gerd_von_rundstedt", "Gerd von Rundstedt", [
    "field_marshal",
    "corps_commander",
  ]),
  person("GER_erich_raeder", "Erich Raeder", ["navy_leader", "advisor"]),
  person("GER_albert_kesselring", "Albert Kesselring", [
    "advisor",
    "corps_commander",
  ]),
];

describe("commanderGroups", () => {
  it("files field marshals, corps commanders and admirals apart, each by name, everyone once", () => {
    expect(
      commanderGroups(commanders, "").map((group) => [
        group.group,
        group.people.map((p) => p.name),
      ]),
    ).toEqual([
      ["fieldMarshal", ["Gerd von Rundstedt"]],
      ["general", ["Albert Kesselring", "Heinz Guderian"]],
      ["admiral", ["Erich Raeder"]],
    ]);
  });

  it("finds people by name or token and drops the groups left empty", () => {
    expect(
      commanderGroups(commanders, "guder").map((group) => [
        group.group,
        group.people.map((p) => p.token),
      ]),
    ).toEqual([["general", ["GER_heinz_guderian"]]]);
    expect(commanderGroups(commanders, "nobody")).toEqual([]);
  });
});

describe("peopleGroups", () => {
  const leader = person("GER_adolf_hitler", "Adolf Hitler", ["country_leader"]);
  const advisors = [
    person("GER_hjalmar_schacht", "Hjalmar Schacht", ["advisor"]),
    person("GER_heinz_guderian", "Heinz Guderian", [
      "advisor",
      "corps_commander",
    ]),
    person("GER_fritz_todt", "Fritz Todt", ["advisor"]),
  ];

  it("puts the leader first, the commanders next, then the advisors no group holds", () => {
    expect(
      peopleGroups({ leader, advisors, commanders }, "").map((group) => [
        group.group,
        group.people.map((p) => p.name),
      ]),
    ).toEqual([
      ["leader", ["Adolf Hitler"]],
      ["fieldMarshal", ["Gerd von Rundstedt"]],
      ["general", ["Albert Kesselring", "Heinz Guderian"]],
      ["admiral", ["Erich Raeder"]],
      ["advisor", ["Fritz Todt", "Hjalmar Schacht"]],
    ]);
  });

  it("searches every group", () => {
    expect(
      peopleGroups({ leader, advisors, commanders }, "schacht").map(
        (group) => group.group,
      ),
    ).toEqual(["advisor"]);
    expect(peopleGroups(undefined, "")).toEqual([]);
  });
});
