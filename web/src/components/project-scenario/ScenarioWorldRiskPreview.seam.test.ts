import { readFileSync } from "node:fs";
import { join } from "node:path";
import { describe, expect, it } from "vitest";

const bookmarksContent = join(
  __dirname,
  "..",
  "workbench",
  "BookmarksContent.tsx",
);

describe("ScenarioWorldRiskPreview bookmark-workbench seam", () => {
  it("mounts the read-only preview beside campaign opening settings", () => {
    const source = readFileSync(bookmarksContent, "utf8");

    expect(source).toMatch(
      /from ["']@\/components\/project-scenario\/ScenarioWorldRiskPreview["']/,
    );
    expect(source).toMatch(/<ScenarioWorldRiskPreview[\s\S]*?preview=/);
  });
});
