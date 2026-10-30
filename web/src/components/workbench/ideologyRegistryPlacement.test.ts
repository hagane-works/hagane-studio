import fs from "node:fs";
import path from "node:path";
import { describe, expect, it } from "vitest";

const HERE = path.dirname(
  new URL(import.meta.url).pathname.replace(/^\/([A-Za-z]:)/, "$1"),
);

function src(file: string): string {
  return fs
    .readFileSync(path.join(HERE, file), "utf8")
    .replace(/\/\*[\s\S]*?\*\//g, "")
    .replace(/(^|[^:])\/\/.*$/gm, "$1");
}

describe("ideology registry · placement", () => {
  it("is replaced by the dedicated ideology library route", () => {
    const country = src("CountrySettingsDialog.tsx");
    expect(country).not.toMatch(/<IdeologyRegistrySection/);
    const politics = src("politics/ideologies/IdeologyLibrary.tsx");
    expect(politics).toMatch(/politicsApi\.references/);
    expect(politics).toMatch(/politicsApi\.createSubtype/);
  });

  it("drives country flag slots from registered ROOTs, not the vanilla icon constant", () => {
    const s = src("country-profile/CountryProfileEditor.tsx");
    expect(s).toMatch(/roots\.map\(\(root\) =>/);
    expect(s).toMatch(/draft\.ideologyNameLocs\[root\.value\]/);
    expect(s).toMatch(/root=\{root\.value\}/);
  });

  it("keeps shared political-system icons out of country identity settings", () => {
    const country = src("country-profile/CountryProfileWorkbench.tsx");
    expect(country).not.toMatch(/ideologyIconsApi/);
    expect(country).not.toMatch(/IDEOLOGY_ICON_GROUPS/);
    expect(country).toMatch(/onOpenPolitics/);
  });

  it("is NOT in the Workshop descriptor dialog any more", () => {
    const s = src("ProjectSettingsDialog.tsx");
    expect(s).not.toMatch(/<IdeologyRegistrySection/);
    expect(s).not.toMatch(/function IdeologyRegistrySection/);

    expect(s).not.toMatch(/ideologyPresets/);
    expect(s).not.toMatch(/VANILLA_IDEOLOGY_REGISTRY/);
  });

  it("REGRESSION · the ENTRY POINT follows the section, not just the section", () => {
    const character = src("CharacterEditor.tsx");
    expect(character).toMatch(
      /href=\{`\/project\/\$\{projectId\}\/ideologies`\}/,
    );
    expect(character).toMatch(/confirmDiscardIfDirty\(\)/);
    expect(character).not.toMatch(/openProjectSettings\("ideology"\)/);

    const layout = fs
      .readFileSync(
        path.join(HERE, "../../app/[locale]/project/[id]/layout.tsx"),
        "utf8",
      )
      .replace(/\/\*[\s\S]*?\*\//g, "")
      .replace(/(^|[^:])\/\/.*$/gm, "$1");
    expect(layout).toMatch(
      /href=\{`\/project\/\$\{projectId\}\/country-politics`\}/,
    );
    expect(layout).toMatch(
      /currentTab === "country-politics" \|\| currentTab === "ideologies"/,
    );
  });
});

describe("project navigation · politics and country ownership", () => {
  const layout = () =>
    fs
      .readFileSync(
        path.join(HERE, "../../app/[locale]/project/[id]/layout.tsx"),
        "utf8",
      )
      .replace(/\/\*[\s\S]*?\*\//g, "")
      .replace(/(^|[^:])\/\/.*$/gm, "$1");

  it("shows politics in the always-visible Creation section after Characters", () => {
    const body = layout();
    const createStart = body.indexOf('label={t("sectionCreate")}');
    const characters = body.indexOf('item.path === "characters"', createStart);
    const politics = body.indexOf(
      "href={`/project/${projectId}/country-politics`}",
      characters,
    );
    const deepenStart = body.indexOf('label={t("sectionDeepen")}', politics);
    expect(createStart).toBeGreaterThan(-1);
    expect(characters).toBeGreaterThan(createStart);
    expect(politics).toBeGreaterThan(characters);
    expect(deepenStart).toBeGreaterThan(politics);
    expect(body.slice(createStart, deepenStart)).not.toMatch(
      /toolGroupCollapsed/,
    );
  });

  it("does not hide politics or country identity in Misc", () => {
    const body = layout();
    const miscStart = body.indexOf('label={t("toolsGroupMisc")}');
    const finishStart = body.indexOf('label={t("sectionFinish")}', miscStart);
    const misc = body.slice(miscStart, finishStart);
    expect(miscStart).toBeGreaterThan(-1);
    expect(finishStart).toBeGreaterThan(miscStart);
    expect(misc).not.toMatch(/\/country-politics/);
    expect(misc).not.toMatch(/\/country-profile/);
  });

  it("puts country identity controls beside both single and multi-country selectors", () => {
    const body = layout();
    const singleStart = body.indexOf("countries.length === 1");
    const multiStart = body.indexOf("countries.length > 1", singleStart);
    const allCountries = body.indexOf("<AllCountriesToggle", multiStart);
    expect(body.slice(singleStart, multiStart)).toMatch(/\/country-profile/);
    expect(body.slice(multiStart, allCountries)).toMatch(/\/country-profile/);
    expect(body).toMatch(
      /href=\{`\/project\/\$\{projectId\}\/country-politics`\}/,
    );
  });
});
