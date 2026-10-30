import { readFileSync } from "node:fs";
import { join } from "node:path";
import { describe, expect, it } from "vitest";
import { CHECKED, equipmentRows } from "./DesignerEquipment";

describe("the designer's Equipment panel (recording designe 2:02–2:04)", () => {
  it("stacks a header per kind of equipment and its types under it, with their design counts", () => {
    const rows = equipmentRows(
      [
        {
          archetype: "infantry_equipment",
          label: "Infantry Eq.",
          types: [
            {
              type: "infantry_equipment_1",
              label: "Infantry Equipment I",
              allowed: 1,
              total: 1,
            },
            {
              type: "infantry_equipment_0",
              label: "Basic Infantry Equipment",
              allowed: 1,
              total: 1,
            },
          ],
        },
        {
          archetype: "light_tank_chassis",
          label: "Light Tank",
          types: [
            {
              type: "light_tank_chassis_2",
              label: "Improved Light Tank",
              allowed: 2,
              total: 2,
            },
          ],
        },
      ],
      { archetype: 30, type: 33 },
    );
    expect(
      rows.map(
        (row) =>
          `${row.top} ${row.label}${row.counter ? ` ${row.counter}` : ""}`,
      ),
    ).toEqual([
      "0 Infantry Eq.",
      "30 Infantry Equipment I 1/1",
      "63 Basic Infantry Equipment 1/1",
      "96 Light Tank",
      "126 Improved Light Tank 2/2",
    ]);
  });

  it("ticks the boxes with the checkbox's second frame", () => {
    expect(CHECKED).toBe(2);
  });

  it("opens from the Equipment button and closes from the panel's own close button", () => {
    const page = readFileSync(join(__dirname, "page.tsx"), "utf8");
    expect(page).toContain('node.name === "division_equipment_btn"');
    expect(page).toContain('"division_equipment_view/close_button"');
    expect(page).toContain(
      '...(equipmentView ? ["division_equipment_view", "division_equipment_view/division_title"] : [])',
    );

    expect(page).toContain("...(leftCovered ? DESIGNER_PICKER_HIDDEN : [])");
  });
});
