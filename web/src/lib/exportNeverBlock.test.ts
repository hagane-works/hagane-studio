import { readFileSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";
import { describe, expect, it } from "vitest";

const here = dirname(fileURLToPath(import.meta.url));
const frontendRoot = join(here, "..", "..");
const read = (path: string) => readFileSync(join(frontendRoot, path), "utf8");
const exportPage = read("src/app/[locale]/project/[id]/export/page.tsx");
const statusCard = read("src/components/export/ExportStatusCard.tsx");
const executablePage = exportPage
  .replace(/\/\*[\s\S]*?\*\//g, "")
  .replace(/^\s*\/\/.*$/gm, "");

describe("export never-intercept wiring", () => {
  it("the page carries no risk-confirm dialog and no pending-risk state", () => {
    expect(exportPage).not.toContain("ExportRiskConfirmDialog");
    expect(exportPage).not.toContain("pendingRiskExport");
    expect(exportPage).not.toContain("diagnosticBlockersFromExportError");
    expect(exportPage).not.toContain("confirmRiskExport");
  });

  it("every Go export job is started with the diagnostic acknowledgement set", () => {
    expect(executablePage).toMatch(
      /buildArtifact\(\s*projectId,\s*includeImported,\s*rps,\s*true,\s*buildIdentity,?\s*\)/,
    );

    expect(executablePage).not.toMatch(
      /buildArtifact\(\s*projectId,\s*[^)]*,\s*false,\s*buildIdentity/,
    );
  });

  it("the primary download button is disabled only by an in-flight download", () => {
    expect(statusCard).toContain("disabled={downloading}");

    expect(exportPage).not.toContain("validationReady");
    expect(statusCard).not.toContain("hasErrors");
  });

  it("findings surface honestly instead of gating (health line + verbatim fallback)", () => {
    expect(statusCard).toContain("check-incomplete-banner");

    expect(exportPage).toContain("presentCheckIssues");
  });

  it("advanced append/diff flows auto-grant the diagnostic confirmation", () => {
    expect(
      exportPage.match(
        /confirmDiagnosticRisk=\{\(\) => Promise\.resolve\(true\)\}/g,
      ),
    ).toHaveLength(2);
  });

  it("view-type surfaces are product dialogs, never native <dialog> / page expansion", () => {
    for (const file of [
      "src/components/export/CheckDetailsDialog.tsx",
      "src/components/export/InGameVerifyDialog.tsx",
      "src/components/export/ExportSettingsDialog.tsx",
    ]) {
      const src = read(file);
      expect(src).toContain('from "@/components/ui/dialog"');
      expect(src).not.toContain("<dialog");
    }
  });
});

describe("download path is dialog-free for every identity", () => {
  const pageSource = exportPage;
  const settingsSource = read("src/components/export/ExportSettingsDialog.tsx");

  it("has no ambiguous-identity dialog left on the page", () => {
    expect(pageSource).not.toContain("ambiguousOpen");
    expect(pageSource).not.toContain("identityAmbiguousDeltaBtn");
  });

  it("downloads straight through without opening anything", () => {
    const handler = pageSource.slice(
      pageSource.indexOf("const handlePrimaryDownload"),
      pageSource.indexOf("const handleFix"),
    );
    expect(handler).toContain("doDownload(includeImported)");
    expect(handler).not.toMatch(
      /setAmbiguousOpen|setSettingsOpen|Open\(true\)/,
    );
  });

  it("keeps the delta-vs-full choice available in export settings", () => {
    expect(settingsSource).toContain("showSourceModeChoice");
    expect(settingsSource).toContain("identityAmbiguousFullBtn");
    expect(settingsSource).toContain("identityAmbiguousDeltaBtn");
  });

  it("defaults an ambiguous project to the full package", () => {
    expect(pageSource).toMatch(
      /includeImportedForIdentity =\s*\n?\s*identity === "loaded_existing" \|\| identity === "ambiguous"/,
    );
  });
});
