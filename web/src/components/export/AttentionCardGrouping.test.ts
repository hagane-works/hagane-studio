import { readFileSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";
import { describe, expect, it } from "vitest";

const here = dirname(fileURLToPath(import.meta.url));
const frontendRoot = join(here, "..", "..", "..");
const read = (path: string) => readFileSync(join(frontendRoot, path), "utf8");
const card = read("src/components/export/AttentionCard.tsx");
const presentation = read("src/lib/exportCheckPresentation.ts");
const repairQueue = read("src/lib/exportRepairQueue.ts");
const repairPanel = read("src/components/export/repair/RepairFindingPanel.tsx");

describe("A2 · the card lists codes, not findings", () => {
  it("renders from grouped rows and counts rows in the header chip", () => {
    expect(card).toContain("groupAttentionFindings");
    expect(card).toContain(
      "const rows = groups ?? groupAttentionFindings(attention)",
    );
    expect(card).toContain(
      't("ui.attention.countChip", { count: rows.length })',
    );

    expect(card).not.toContain("attention.map(");
  });

  it("sends a multi-entity row's detail to the repair page, never inline", () => {
    expect(card).toContain("/export/repair#${repairHash(finding.issue)}");
    expect(card).toContain('t("ui.attention.viewGroup")');

    expect(card).not.toContain("<details");
    expect(card).not.toContain("useState");
  });

  it("shows the deduped affected count on rows that cover several things", () => {
    expect(card).toContain('data-testid="export-attention-group-count"');
    expect(card).toContain('t("ui.attention.groupAffected", {');
  });
});

describe("A2 · one grouping implementation, two surfaces", () => {
  it("the repair queue groups through the presentation layer's function", () => {
    expect(repairQueue).toContain("groupByFindingCode");
    expect(repairQueue).toContain('from "@/lib/exportCheckPresentation"');

    const groupFn = repairQueue.slice(
      repairQueue.indexOf("export function groupRepairQueue"),
    );
    expect(groupFn).not.toContain("new Map<string");
  });

  it("the deduped count is the entity count, with an honest fallback", () => {
    expect(presentation).toContain("export function countDistinctEntities");

    expect(presentation).toContain("unattributed += 1");
    expect(presentation).toContain("return seen.size + unattributed;");
  });
});

describe("A13 · no entry point is stated, not left blank", () => {
  it("reuses the repair page's wording instead of inventing a second one", () => {
    expect(card).toContain('t("ui.repair.panelNoRoute")');
    expect(card).toContain('t("ui.repair.panelNoRouteBody")');
    expect(repairPanel).toContain('t("ui.repair.panelNoRoute")');
    expect(repairPanel).toContain('t("ui.repair.panelNoRouteBody")');
  });

  it("renders the note whenever the finding has no workbench route", () => {
    expect(card).toContain("const noRoute = target === null;");
    expect(card).toContain('data-testid="export-attention-no-route"');
  });

  it("never fabricates a route to fill the gap", () => {
    expect(card).toContain("findingRepairTarget(projectId, finding.issue)");
    expect(card).not.toMatch(/href=\{`\/project\/\$\{projectId\}\/\$\{/);
  });
});

describe("project-scoped findings open the settings dialog, not a dead link", () => {
  it("renders a button wired to the cross-component settings store", () => {
    expect(card).toContain('data-testid="export-attention-open-settings"');
    expect(card).toContain("useOpenProjectSettings((s) => s.requestOpen)");
    expect(card).toContain("openProjectSettings()");

    expect(card).not.toMatch(/href=\{[^}]*settings/i);
  });
});
