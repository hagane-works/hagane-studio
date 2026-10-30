import fs from "node:fs";
import path from "node:path";
import { describe, expect, it } from "vitest";

const HERE = path.dirname(
  new URL(import.meta.url).pathname.replace(/^\/([A-Za-z]:)/, "$1"),
);

function read(relativePath: string): string {
  return fs.readFileSync(path.resolve(HERE, relativePath), "utf8");
}

describe("politics workbench shell", () => {
  it("uses recoverable project routes for politics and country identity", () => {
    const layout = read("../../../app/[locale]/project/[id]/layout.tsx");
    expect(layout).toContain("/country-politics`}");
    expect(layout).toContain("/country-profile`}");
    expect(layout).toMatch(
      /currentTab === "country-politics" \|\| currentTab === "ideologies"/,
    );
  });

  it("keeps the country selector in the project shell, not in the tab bar", () => {
    const layout = read("../../../app/[locale]/project/[id]/layout.tsx");
    const tabs = read("./PoliticsWorkspaceNav.tsx");
    expect(layout).toContain("<CountrySwitcher projectId={projectId}");
    expect(tabs).not.toContain("CountrySwitcher");
    expect(tabs).toContain("countryPoliticsScope");
    expect(tabs).toContain("ideologyLibraryScope");
  });
});
