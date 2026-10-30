import { readFileSync } from "node:fs";
import { join } from "node:path";
import { describe, expect, it } from "vitest";

function withoutComments(source: string): string {
  return source.replace(/\/\*[\s\S]*?\*\//g, "").replace(/^\s*\/\/.*$/gm, "");
}

describe("ProjectTopBar active-country presence", () => {
  it("keeps the country chip read-only while its shortcut opens the shared palette", () => {
    const topBar = withoutComments(
      readFileSync(join(__dirname, "ProjectTopBar.tsx"), "utf8"),
    );
    const switcher = withoutComments(
      readFileSync(join(__dirname, "CountrySwitcher.tsx"), "utf8"),
    );
    const layout = withoutComments(
      readFileSync(
        join(__dirname, "../../app/[locale]/project/[id]/layout.tsx"),
        "utf8",
      ),
    );

    expect(topBar).toContain('aria-label={`${t("activeCountryLabel")}');
    expect(topBar).toMatch(/<div\s+aria-label=/);
    expect(topBar).toContain("<button");
    expect(topBar).toContain("hoi4-open-country-palette");
    expect(topBar).toContain("openPalette");
    expect(topBar).toContain("shortcutHint");
    expect(topBar).not.toMatch(
      /<button\s+[\s\S]*?aria-label=\{`\$\{t\("activeCountryLabel"\)/,
    );
    expect(switcher).toContain(
      'window.addEventListener("hoi4-open-country-palette", onOpen)',
    );
    expect(switcher).toContain('document.addEventListener("keydown", onKey)');
    expect(layout).toMatch(
      /sidebarCollapsed\s*&&\s*currentProject\s*&&\s*countries\.length\s*>\s*1[\s\S]*?<CountrySwitcher projectId=\{projectId\}/,
    );
  });
});
