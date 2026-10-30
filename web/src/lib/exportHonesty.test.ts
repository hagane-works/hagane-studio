import { readFileSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";
import { describe, expect, it } from "vitest";
import { UNSUPPORTED_OR_LOSSY_EXPORT_PATHS } from "@/generated/exportCoverage";

const here = dirname(fileURLToPath(import.meta.url));
const frontendRoot = join(here, "..", "..");
const read = (path: string) => readFileSync(join(frontendRoot, path), "utf8");
const exportPage = read("src/app/[locale]/project/[id]/export/page.tsx");

const settingsDialog = read("src/components/export/ExportSettingsDialog.tsx");
const rawOverrideManager = read(
  "src/components/workbench/RawOverrideFilesManager.tsx",
);
const enExport = JSON.parse(read("messages/en/export.json")) as Record<
  string,
  string
>;
const zhExport = JSON.parse(read("messages/zh/export.json")) as Record<
  string,
  string
>;

describe("export honesty wiring", () => {
  it("keeps raw files and directory replacement reachable for standalone projects", () => {
    expect(exportPage).not.toContain("showReplacePathSection");
    expect(exportPage).toContain("getSelectedReplacePaths(replacePaths)");

    const rawManager = settingsDialog.indexOf("<RawOverrideFilesManager");
    expect(rawManager).toBeGreaterThan(-1);
    expect(settingsDialog).not.toContain("showReplacePathSection");
    expect(settingsDialog).not.toContain("showOverrideSection");
    expect(settingsDialog).toContain("allowReplacePath");
    expect(settingsDialog).not.toContain("allowReplacePath={hasDependencies}");
    expect(rawOverrideManager).toContain("allowReplacePath = false");
    expect(
      rawOverrideManager.match(/allowReplacePath\s*&&\s*\(\s*<label/g),
    ).toHaveLength(3);
    expect(
      rawOverrideManager.match(/generate_replace_path: allowReplacePath &&/g),
    ).toHaveLength(3);
  });

  it("renders unsupported and lossy paths from the generated coverage projection", () => {
    expect(UNSUPPORTED_OR_LOSSY_EXPORT_PATHS.length).toBeGreaterThan(0);
    expect(settingsDialog).toContain(
      'import { UNSUPPORTED_OR_LOSSY_EXPORT_PATHS } from "@/generated/exportCoverage";',
    );
    expect(settingsDialog).toContain(
      "UNSUPPORTED_OR_LOSSY_EXPORT_PATHS.map((path) =>",
    );

    expect(exportPage).toContain('identity === "loaded_existing"');
    expect(exportPage).toContain("showCoverageNote={showCoverageNote}");

    for (const messages of [enExport, zhExport]) {
      expect(messages.identityLoadedUnmodeledNote).toContain("{count}");
      expect(messages.identityLoadedUnmodeledPaths).toContain("{count}");
      expect(messages).not.toHaveProperty("step3StandaloneLossyInline");
      expect(messages).not.toHaveProperty("step3LossyWarningBody");
    }
    expect(enExport.checksSectionIntro).not.toMatch(/crash-safety/i);
    expect(zhExport.checksSectionIntro).not.toContain("崩溃安全检查");
  });

  it("keeps an honest scope note on the redesigned check-details dialog", () => {
    const detailsDialog = read("src/components/export/CheckDetailsDialog.tsx");
    expect(detailsDialog).toContain('t("ui.dialogCheck.scopeNote")');
    const enUi = (
      JSON.parse(read("messages/en/export.json")) as {
        ui: { dialogCheck: { scopeNote: string } };
      }
    ).ui;
    const zhUi = (
      JSON.parse(read("messages/zh/export.json")) as {
        ui: { dialogCheck: { scopeNote: string } };
      }
    ).ui;
    expect(enUi.dialogCheck.scopeNote).toMatch(/Raw files/i);
    expect(zhUi.dialogCheck.scopeNote).toContain("Raw 文件");
  });
});
