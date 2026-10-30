import { describe, expect, it } from "vitest";
import { addCommand, doctrinePatch, swapCommand } from "./doctrineEdit";

const GER = {
  raw_script: [
    "capital = 64",
    "set_grand_doctrine = new_mobile_warfare",
    "set_grand_doctrine = new_convoy_raiding",
    "if = {",
    '\tlimit = { has_dlc = "No Step Back" }',
    "\tset_grand_doctrine = new_battlefield_support",
    "}",
    "set_variable = { GER_military_land_doctrine_cost_factor = 0 }",
    "",
  ].join("\n"),
  date_blocks: [
    {
      date: "1939.1.1",
      script:
        "set_sub_doctrine = mobile_infantry\nadd_mastery = { amount = 200 sub_doctrine = mobile_infantry }\n",
    },
  ],
};

describe("the doctrine a country starts with", () => {
  it("swaps a name only where it is the command's own", () => {
    const script =
      "set_sub_doctrine = mobile_infantry\nadd_mastery = { amount = 200 sub_doctrine = mobile_infantry }";
    expect(
      swapCommand(script, "sub_doctrine", "mobile_infantry", "mass_assault"),
    ).toBe(
      "set_sub_doctrine = mobile_infantry\nadd_mastery = { amount = 200 sub_doctrine = mass_assault }",
    );
    expect(
      swapCommand(
        "set_sub_doctrine = mobile_infantry_2",
        "set_sub_doctrine",
        "mobile_infantry",
        "x",
      ),
    ).toBe("set_sub_doctrine = mobile_infantry_2");
  });

  it("adds a line after the last top-level one of its kind, not inside a block", () => {
    const out = addCommand(
      GER.raw_script,
      "set_grand_doctrine",
      "special_forces_quantity",
    );
    expect(out.split("\n").slice(1, 4)).toEqual([
      "set_grand_doctrine = new_mobile_warfare",
      "set_grand_doctrine = new_convoy_raiding",
      "set_grand_doctrine = special_forces_quantity",
    ]);
  });

  it("puts a first subdoctrine under the grand doctrines, or at the end", () => {
    const out = addCommand(
      GER.raw_script,
      "set_sub_doctrine",
      "mobile_infantry",
      ["set_grand_doctrine"],
    );
    expect(out.split("\n")[3]).toBe("set_sub_doctrine = mobile_infantry");
    expect(
      addCommand("capital = 64\n\n", "set_sub_doctrine", "mobile_infantry", [
        "set_grand_doctrine",
      ]),
    ).toBe("capital = 64\nset_sub_doctrine = mobile_infantry\n");
    expect(addCommand("", "set_grand_doctrine", "new_mobile_warfare")).toBe(
      "set_grand_doctrine = new_mobile_warfare\n",
    );
  });

  it("changes a doctrine the country has on every date, mastery and all", () => {
    const patch = doctrinePatch(GER, {
      kind: "sub",
      from: "mobile_infantry",
      to: "mass_assault",
    });
    expect(patch?.raw_script).toBeUndefined();
    expect(patch?.date_blocks?.[0].script).toBe(
      "set_sub_doctrine = mass_assault\nadd_mastery = { amount = 200 sub_doctrine = mass_assault }\n",
    );
    const grand = doctrinePatch(GER, {
      kind: "grand",
      from: "new_mobile_warfare",
      to: "new_superior_firepower",
    });
    expect(grand?.raw_script).toContain(
      "set_grand_doctrine = new_superior_firepower",
    );
    expect(grand?.date_blocks).toBeUndefined();
  });

  it("adds one the country does not have, and leaves the row alone for no change", () => {
    expect(
      doctrinePatch(GER, { kind: "grand", to: "special_forces_quantity" })
        ?.raw_script,
    ).toContain("set_grand_doctrine = special_forces_quantity");
    expect(
      doctrinePatch(GER, {
        kind: "sub",
        from: "mobile_infantry",
        to: "mobile_infantry",
      }),
    ).toBeNull();

    expect(
      doctrinePatch(GER, {
        kind: "sub",
        from: "trench_warfare",
        to: "mass_assault",
      }),
    ).toBeNull();
  });
});
