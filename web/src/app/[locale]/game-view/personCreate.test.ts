import { readFileSync } from "node:fs";
import { join } from "node:path";
import { describe, expect, it } from "vitest";
import {
  NEW_ADVISOR_COST,
  newPersonProblem,
  newPersonRow,
  personDeletable,
  personToken,
  type NewPerson,
} from "./personCreate";

const advisor = (extra: Partial<NewPerson> = {}): NewPerson => ({
  tag: "ger",
  name: "Otto Weber",
  kind: "advisor",
  slot: "political_advisor",
  ideology: "",
  ...extra,
});

describe("a new person from the game view", () => {
  it("makes the token from the country and the name", () => {
    expect(personToken("ger", "Otto Weber")).toBe("GER_otto_weber");
    expect(personToken("GER", "Käthe  Kollwitz-Schmidt")).toBe(
      "GER_kathe_kollwitz_schmidt",
    );
    expect(personToken("GER", "毛奇")).toBe("GER_new_person");
  });

  it("creates an advisor in their seat at the usual cost", () => {
    expect(newPersonRow(advisor())).toEqual({
      name: "Otto Weber",
      name_en: "Otto Weber",
      tag: "GER_otto_weber",
      localisation_key: "",
      role_type: "advisor",
      has_advisor_role: true,
      country_tag: "GER",
      advisor_slot: "political_advisor",
      advisor_cost: NEW_ADVISOR_COST,
    });
  });

  it("gives a leader their ideology and a general nothing of an advisor's", () => {
    expect(
      newPersonRow(advisor({ kind: "country_leader", ideology: "nazism" })),
    ).toMatchObject({
      role_type: "country_leader",
      has_advisor_role: false,
      ideology: "nazism",
    });
    const general = newPersonRow(advisor({ kind: "general" }));
    expect(general).not.toHaveProperty("advisor_slot");
    expect(general).not.toHaveProperty("ideology");
  });

  it("refuses what the game would not take", () => {
    expect(newPersonProblem(advisor())).toBeNull();
    expect(newPersonProblem(advisor({ name: " " }))).toBe("nameRequired");
    expect(newPersonProblem(advisor({ name: 'Otto "Iron" Weber' }))).toBe(
      "nameInvalid",
    );
    expect(newPersonProblem(advisor({ slot: "" }))).toBe("slotRequired");
    expect(newPersonProblem(advisor({ kind: "country_leader" }))).toBe(
      "ideologyRequired",
    );
  });

  it("deletes only a person made in the project", () => {
    expect(
      personDeletable({ replaces_vanilla_id: false, source_file: "" }),
    ).toBe(true);
    expect(
      personDeletable({
        replaces_vanilla_id: true,
        source_file: "common/characters/GER.txt",
      }),
    ).toBe(false);
    expect(
      personDeletable({
        replaces_vanilla_id: false,
        source_file: "common/characters/MOD.txt",
      }),
    ).toBe(false);
  });

  it("is offered on the politics screen too, in the edit column like every 新建 card", () => {
    const page = readFileSync(join(__dirname, "page.tsx"), "utf8");
    expect(page).toContain(
      '(base === OFFICER_CORPS || panel === "countrypoliticsview") && tag ? (',
    );
    expect(page).toContain(
      "{editSwitcher && <EditSwitcherDock>{editSwitcher}</EditSwitcherDock>}",
    );
    const panel = readFileSync(
      join(__dirname, "CharacterEditPanel.tsx"),
      "utf8",
    );
    expect(panel).toContain("row && personDeletable(row) && (");
  });
});
