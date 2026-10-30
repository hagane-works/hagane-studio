import { readFileSync } from "node:fs";
import { describe, expect, it } from "vitest";

const source = readFileSync(
  new URL("../components/workbench/TechSurfacePanel.tsx", import.meta.url),
  "utf8",
);
const technologyPageSource = readFileSync(
  new URL("../app/[locale]/project/[id]/technology/page.tsx", import.meta.url),
  "utf8",
);
const zhMessages = JSON.parse(
  readFileSync(
    new URL("../../messages/zh/techSurface.json", import.meta.url),
    "utf8",
  ),
) as Record<string, string>;

describe("technology surface player mental model", () => {
  it("does not expose implementation paths, local-version setup, or P0 language", () => {
    expect(source).not.toContain("tech-surface-additive-guidance");
    expect(source).not.toContain("common/technologies/{TAG}");
    expect(source).not.toContain("countrytechtreeview.gui");
    expect(source).not.toContain("ModFolderPicker");
    expect(source).not.toContain("tech-source-advanced");
    expect(zhMessages.title).toBe("科技树");
    expect(zhMessages.entrancesTitle).toBe("选择研究页面");
    expect(zhMessages.title).not.toContain("P0");
    expect(zhMessages.description).not.toContain("格位");
  });

  it("asks one DLC-requirement question and only reveals DLC choices when required", () => {
    expect(source).toContain('compatibilityMode === "requires_dlc"');
    expect(source).toContain(
      'compatibilityMode === "all_players" ? EMPTY_DLC_STATE : dlcSelection',
    );
    expect(source).toContain('t("requiredDLCsTitle")');
    expect(zhMessages.environmentTitle).toBe("这个 Mod 需要 DLC 吗？");
    expect(zhMessages.compatibilityAllPlayers).toContain("不需要 DLC");
    expect(zhMessages).not.toHaveProperty("ownedDLCsTitle");
  });

  it("discards unsaved compatibility choices when the player cancels", () => {
    expect(source).toContain("const handleCancelCompatibility = () =>");
    expect(source).toMatch(
      /handleCancelCompatibility[\s\S]*?setDlcDraft\(null\)/,
    );
    expect(source).toMatch(
      /handleCancelCompatibility[\s\S]*?setCompatibilityDraft\(null\)/,
    );
    expect(source).toContain("onClick={handleCancelCompatibility}");
  });

  it("uses the four original-game page tabs and a visible year-position timeline", () => {
    expect(source).toContain("TECH_SURFACE_ENTRANCE_IDS.map");
    expect(source).toContain("aria-pressed={active}");
    expect(source).toContain("target.timeline_years.map");
    expect(source).toContain('t("targetFirstYear"');
    expect(source).toContain('t("yearComesFromPosition")');
    expect(source).toContain(
      "isTechSurfaceTargetSelectable(target, compatibilityMode)",
    );
  });

  it("keeps stable IDs optional and supports deleting a complete technology line", () => {
    expect(source).toContain("generatedTechnologyID(rootName)");
    expect(source).toContain("generatedTechnologyID(childName)");
    expect(source).toContain("technologiesApi.deleteSurfaceBranch");
    expect(source).toContain('t("technicalSettings")');
    expect(source).toContain('t("deleteBranchTitle")');
  });

  it("preserves the existing create and timeline-repair API wiring", () => {
    expect(
      source.match(/technologiesApi\.attachCertifiedSurfaceSource\(/g),
    ).toHaveLength(1);
    expect(
      source.match(/technologiesApi\.confirmSurfaceEnvironment\(/g),
    ).toHaveLength(1);
    expect(
      source.match(/technologiesApi\.createSurfaceBranch\(/g),
    ).toHaveLength(1);
    expect(
      source.match(/technologiesApi\.createSurfaceBranchTechnology\(/g),
    ).toHaveLength(1);
    expect(
      source.match(/technologiesApi\.repairSurfaceTimeline\(/g),
    ).toHaveLength(1);
  });
});

describe("technology entry-point clarity", () => {
  it("keeps Mod import in the global sidebar instead of duplicating it in technology", () => {
    expect(technologyPageSource).not.toContain("useImportButton");
    expect(technologyPageSource).not.toContain("importApi.technologies");
    expect(technologyPageSource).not.toContain("techImport.trigger");
  });

  it("attaches the latest certified baseline without asking for a local game folder", () => {
    expect(source).toContain("technologiesApi.attachCertifiedSurfaceSource");
    expect(source).not.toContain("collectLocalTechnologySource");
    expect(source).not.toContain("markSurfaceSourceConnected");
    expect(zhMessages.latestVersionReady).toContain("最新版 HOI4");
  });
});
