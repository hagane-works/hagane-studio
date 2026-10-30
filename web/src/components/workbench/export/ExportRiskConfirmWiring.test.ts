import fs from "node:fs";
import path from "node:path";
import { describe, expect, it } from "vitest";

const pageSource = fs.readFileSync(
  path.resolve(process.cwd(), "src/app/[locale]/project/[id]/export/page.tsx"),
  "utf8",
);
const appendSource = fs.readFileSync(
  path.resolve(
    process.cwd(),
    "src/components/workbench/AppendToExistingModDialog.tsx",
  ),
  "utf8",
);
const diffSource = fs.readFileSync(
  path.resolve(
    process.cwd(),
    "src/components/workbench/DiffMergeExportDialog.tsx",
  ),
  "utf8",
);

describe("export diagnostic-risk confirmation wiring (post never-intercept)", () => {
  it("the export page no longer mounts the risk-confirm dialog", () => {
    expect(pageSource).not.toContain("ExportRiskConfirmDialog");
    expect(pageSource).not.toContain("pendingRiskExport");
  });

  it("append and diff exports receive an auto-granting confirmation callback", () => {
    expect(
      pageSource.match(
        /confirmDiagnosticRisk=\{\(\) => Promise\.resolve\(true\)\}/g,
      ),
    ).toHaveLength(2);
  });

  it("the advanced dialog components still thread the confirmation internally", () => {
    for (const source of [appendSource, diffSource]) {
      expect(source).toContain("diagnosticBlockersFromExportError");
      expect(source).toContain("await confirmDiagnosticRisk(finalBlockers)");
    }
    expect(appendSource).toContain("runAppend(true)");
    expect(diffSource).toContain("runDiffExport(true)");
  });
});
