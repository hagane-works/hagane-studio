import { readFileSync } from "node:fs";
import { join } from "node:path";
import { describe, expect, it } from "vitest";

const bookmarksContent = join(
  __dirname,
  "..",
  "workbench",
  "BookmarksContent.tsx",
);
const projectSettingsDialog = join(
  __dirname,
  "..",
  "workbench",
  "ProjectSettingsDialog.tsx",
);

describe("ScenarioSettingsPanel bookmark-workbench seam", () => {
  it("mounts campaign world/end settings beside the bookmark entry and persists only those fields", () => {
    const source = readFileSync(bookmarksContent, "utf8");

    expect(source).toMatch(
      /from ["']@\/components\/project-scenario\/ScenarioSettingsPanel["']/,
    );
    expect(source).toMatch(/<ScenarioSettingsPanel[\s\S]*?onChange=/);
    expect(source).toMatch(/<ScenarioSettingsPanel[\s\S]*?needsEndDate=/);
    expect(source).toMatch(
      /<ScenarioSettingsPanel[\s\S]*?isStandaloneProject=/,
    );
    expect(source).toMatch(
      /saveScenarioProjectWithReadback\(projectId, \{[\s\S]*?end_date:/,
    );
    expect(source).toMatch(/scenario_history_policy:/);
    expect(source).toContain("applyProjectReadback(readback)");
  });

  it("syncs the project opening to the earliest valid saved bookmark", () => {
    const source = readFileSync(bookmarksContent, "utf8");

    expect(source).toContain(
      "const savedBookmark = earliestSavedBookmark(rows)",
    );
    expect(source).toMatch(
      /saveScenarioProjectWithReadback\(projectId, \{[\s\S]*?start_date: openingDate/,
    );
    expect(source).not.toContain("campaignOpening.useBookmark");
  });

  it("keeps campaign settings out of the descriptor dialog", () => {
    const source = readFileSync(projectSettingsDialog, "utf8");

    expect(source).not.toContain("ScenarioSettingsPanel");
    expect(source).not.toContain("scenario_history_policy:");
    expect(source).not.toContain("start_date:");
  });
});
