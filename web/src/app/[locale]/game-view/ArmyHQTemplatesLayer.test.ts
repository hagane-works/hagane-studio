import { describe, expect, it } from "vitest";
import { OPENS_OVER, PANEL_TARGETS, opensOverWith } from "./gameNavigation";
import { ARMY_HQ_TEMPLATES, hqRowTexts } from "./ArmyHQTemplatesLayer";
import { REFERENCE_TABLES } from "@/lib/__fixtures__/referenceTables";
import { setReferenceTables } from "@/lib/referenceTables";

setReferenceTables(REFERENCE_TABLES);

describe("the Army HQ Templates list", () => {
  it("opens from the officer corps' ribbon, beside it, and opens the designer", () => {
    expect(
      PANEL_TARGETS["countryofficercorpview/button_army_hq_templates_toggle"],
    ).toBe(ARMY_HQ_TEMPLATES);
    expect(OPENS_OVER[ARMY_HQ_TEMPLATES]).toBe("countryofficercorpview");
    expect(PANEL_TARGETS[`${ARMY_HQ_TEMPLATES}/open_designer_button`]).toBe(
      "countrydivisiondesignerview",
    );

    expect(
      opensOverWith(null, ARMY_HQ_TEMPLATES).countrydivisiondesignerview,
    ).toBe(ARMY_HQ_TEMPLATES);
    expect(opensOverWith(null).countrydivisiondesignerview).toBe(
      "countrydeploymentview",
    );
  });

  it("writes a row's days and command power the way the game does", () => {
    const texts = {
      HQ_TEMPLATE_DEPLOY_DAYS_SHORT: "§Y$DAYS$§! days",
      HQ_DEPLOY_AND_WITHDRAW_DAYS:
        "Deployment and withdrawal will take §Y$DAYS$§! days.",
      HQ_DEPLOY_AND_WITHDRAW_DAYS_SINGULAR:
        "Deployment and withdrawal will take §Y$DAYS$§! day.",
      HQ_DEPLOY_CP_COST: "Deployment will reserve £command_power $COST|H0$.",
    };
    const row = hqRowTexts(
      { name: "AOK", deploy_days: 3, command_power: 9 },
      texts,
    );
    expect(row.days).toBe("§Y3§! days");
    expect(row.daysTip).toBe("Deployment and withdrawal will take §Y3§! days.");
    expect(row.cost).toBe("9");
    expect(row.costTip).toBe("Deployment will reserve £command_power §H9§!.");
    expect(hqRowTexts({ name: "X", deploy_days: 1 }, texts).daysTip).toBe(
      "Deployment and withdrawal will take §Y1§! day.",
    );
  });
});
