import { describe, expect, it } from "vitest";
import {
  TECH_SURFACE_ENTRANCE_IDS,
  buildTechSurfaceBranchCreateInput,
  buildTechSurfaceEnvironmentConfirmInput,
  isTechSurfaceEditableLayout,
  isTechSurfaceTargetSelectable,
  techSurfaceBranches,
  type TechSurfaceStatus,
  type TechSurfaceProfileTarget,
} from "./techSurface";

describe("isTechSurfaceEditableLayout", () => {
  it("locks the generic canvas while a certified P0 surface owns the branch", () => {
    expect(isTechSurfaceEditableLayout(true, true)).toBe(false);
  });

  it("preserves legacy layout mode for projects without a P0 surface", () => {
    expect(isTechSurfaceEditableLayout(true, false)).toBe(true);
    expect(isTechSurfaceEditableLayout(false, false)).toBe(false);
  });
});

describe("techSurfaceBranches", () => {
  it("treats a legacy null branch collection as an empty surface", () => {
    const status = {
      manifest: { branches: null },
    } as unknown as TechSurfaceStatus;

    expect(techSurfaceBranches(status)).toEqual([]);
    expect(techSurfaceBranches(undefined)).toEqual([]);
  });
});

describe("technology surface profile authoring contract", () => {
  it("uses the four canonical entrance tokens", () => {
    expect(TECH_SURFACE_ENTRANCE_IDS).toEqual([
      "army",
      "air",
      "navy",
      "industry_equipment",
    ]);
    expect(TECH_SURFACE_ENTRANCE_IDS).not.toContain("industrial_equipment");
  });

  it("submits an explicit entry and target without client placement fields", () => {
    const request = buildTechSurfaceBranchCreateInput({
      id: "mod_infantry_extension",
      entryID: "army",
      targetID: "infantry",
      root: { tech_id: "MOD_infantry_extension_1", name: "Test I" },
    });

    expect(request).toEqual({
      id: "mod_infantry_extension",
      entry_id: "army",
      target_id: "infantry",
      root: { tech_id: "MOD_infantry_extension_1", name: "Test I" },
    });
    expect(request).not.toHaveProperty("surface");
    expect(request).not.toHaveProperty("folder");
    expect(request).not.toHaveProperty("slot");
  });

  it("keeps an uncertified armour target visible but unselectable", () => {
    const armour: TechSurfaceProfileTarget = {
      id: "armour",
      folder_id: "armour_folder",
      authoring_enabled: false,
      certification: {
        state: "source_locked_pending_engine",
        evidence_id: "P0-ARMOUR-PENDING",
        detail: "Requires in-game evidence.",
      },
      timeline_years: [1936, 1940],
      all_players_supported: false,
    };

    expect(isTechSurfaceTargetSelectable(armour)).toBe(false);
  });

  it("enables only a target backed by engine-verified geometry", () => {
    const infantry: TechSurfaceProfileTarget = {
      id: "infantry",
      folder_id: "infantry_folder",
      authoring_enabled: true,
      certification: {
        state: "engine_verified_geometry",
        evidence_id: "P0-A-ENGINE-ACCEPTED",
      },
      timeline_years: [1918, 1936],
      all_players_supported: true,
    };

    expect(isTechSurfaceTargetSelectable(infantry)).toBe(true);
  });

  it("confirms only the three relevant DLC switches", () => {
    expect(
      buildTechSurfaceEnvironmentConfirmInput(
        {
          no_step_back: true,
          by_blood_alone: false,
          man_the_guns: true,
        },
        "all_players",
      ),
    ).toEqual({
      dlc: {
        no_step_back: true,
        by_blood_alone: false,
        man_the_guns: true,
      },
      compatibility_mode: "all_players",
    });
  });

  it("blocks a DLC-only target under the all-player policy", () => {
    const navalSupport: TechSurfaceProfileTarget = {
      id: "naval_support",
      folder_id: "mtgnavalsupportfolder",
      authoring_enabled: true,
      certification: {
        state: "engine_verified_geometry",
        evidence_id: "P0-A-ENGINE-ACCEPTED",
      },
      timeline_years: [1922, 1936],
      all_players_supported: false,
    };
    expect(isTechSurfaceTargetSelectable(navalSupport, "all_players")).toBe(
      false,
    );
    expect(isTechSurfaceTargetSelectable(navalSupport, "requires_dlc")).toBe(
      true,
    );
  });
});
