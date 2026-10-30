import { existsSync, readFileSync } from "node:fs";
import { join } from "node:path";
import { describe, expect, it } from "vitest";

describe("country-domain central seam", () => {
  it("stays red in strict seam mode until central routes mount both private pages", () => {
    if (process.env.HOI4_REQUIRE_POLITICS_WORKBENCH_SEAMS !== "1") {
      expect(true).toBe(true);
      return;
    }

    const layout = readFileSync(
      join(__dirname, "../../../app/[locale]/project/[id]/layout.tsx"),
      "utf8",
    );
    const politicsPath = join(
      __dirname,
      "../../../app/[locale]/project/[id]/country-politics/page.tsx",
    );
    const profilePath = join(
      __dirname,
      "../../../app/[locale]/project/[id]/country-profile/page.tsx",
    );
    expect(existsSync(politicsPath), "create the country-politics route").toBe(
      true,
    );
    expect(existsSync(profilePath), "create the country-profile route").toBe(
      true,
    );
    if (existsSync(politicsPath)) {
      expect(
        readFileSync(politicsPath, "utf8").includes("CountryPoliticsPage"),
      ).toBe(true);
    }
    if (existsSync(profilePath)) {
      expect(
        readFileSync(profilePath, "utf8").includes("CountryProfilePage"),
      ).toBe(true);
    }
    expect(
      layout.includes("useActiveCountry"),
      "keep the central country context",
    ).toBe(true);
    expect(layout).toContain("country-politics");
    expect(layout).toContain("country-profile");
  });

  it("uses an edit affordance for every country-profile entry instead of a settings gear", () => {
    const layout = readFileSync(
      join(__dirname, "../../../app/[locale]/project/[id]/layout.tsx"),
      "utf8",
    );
    const profileLinks =
      layout.match(
        /<Link\s+href=\{`\/project\/\$\{projectId\}\/country-profile`\}[\s\S]*?<\/Link>/g,
      ) ?? [];

    expect(profileLinks).toHaveLength(3);
    for (const link of profileLinks) {
      expect(link).toContain("<PencilLine");
      expect(link).not.toContain("<Settings");
      expect(link).toContain("countryProfileActive");
      expect(link).toContain("countrySettingsTitle");
      expect(link).toContain("border-accent/25 bg-accent/5 text-accent");
    }
    expect(layout).toContain('currentTab === "country-profile"');
    expect(layout).toContain(
      '`${t("countrySettingsTitle")} (${activeCountry})`',
    );
  });
});
