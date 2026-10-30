import { readFileSync } from "node:fs";
import { join } from "node:path";
import { describe, expect, it } from "vitest";
import { catalogueItems, type GuiEquipmentItem } from "./ProductionLayer";

const row = (
  name: string,
  extra: Partial<GuiEquipmentItem> = {},
): GuiEquipmentItem => ({
  name,
  archetype: "infantry_equipment",
  category: "interface_category_land",
  group: "infantry",
  year: 1936,
  cost: 0.5,
  ...extra,
});

describe("the production catalogue's rows", () => {
  it("leaves the obsolete ones out until Show Outdated Equipment is ticked (recording 3:06)", () => {
    const rows = [
      row("infantry_equipment_0", {
        label: "Gewehr 98",
        general_name: "Basic Infantry Equipment",
        obsolete: true,
      }),
      row("infantry_equipment_1", {
        label: "Karabiner 98k",
        general_name: "Infantry Equipment I",
      }),
      row("Panzer II", {
        design: true,
        type: "light_tank_chassis_2",
        group: "armor",
        designer: "tank",
      }),
      row("He 51", { design: true, obsolete: true, group: "air" }),
    ];
    expect(
      catalogueItems(rows, ["infantry", "armor", "air"], false).map(
        (item) => item.name,
      ),
    ).toEqual(["infantry_equipment_1", "Panzer II"]);
    expect(
      catalogueItems(rows, ["infantry", "armor", "air"], true),
    ).toHaveLength(4);
    expect(
      catalogueItems(rows, ["armor"], true).map((item) => item.name),
    ).toEqual(["Panzer II"]);
  });

  it("draws a row's general name under its own and wires the outdated checkbox", () => {
    const layer = readFileSync(join(__dirname, "ProductionLayer.tsx"), "utf8");
    expect(layer).toContain("text={item.general_name}");
    expect(layer).toContain('aria-label="outdated_units"');
    expect(layer).toContain("{ checkbox: showOutdated ? 2 : 1 }");
  });
});
