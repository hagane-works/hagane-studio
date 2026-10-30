import { readFileSync } from "node:fs";
import { join } from "node:path";
import { describe, expect, it } from "vitest";

const profilePage = readFileSync(
  join(__dirname, "CountryProfilePage.tsx"),
  "utf8",
);
const politicsPage = readFileSync(
  join(__dirname, "../politics/country-politics/CountryPoliticsPage.tsx"),
  "utf8",
);
const profileWorkbench = readFileSync(
  join(__dirname, "CountryProfileWorkbench.tsx"),
  "utf8",
);
const profileEditor = readFileSync(
  join(__dirname, "CountryProfileEditor.tsx"),
  "utf8",
);
const politicsWorkbench = readFileSync(
  join(__dirname, "../politics/country-politics/CountryPoliticsWorkbench.tsx"),
  "utf8",
);

describe("country-domain central context contract", () => {
  it("consumes one active country and does not render a second selector", () => {
    for (const source of [profilePage, politicsPage]) {
      expect(source).toContain("useActiveCountry()");
      expect(source).toContain("effectiveCountryTag");
      expect(source).not.toMatch(/<select[^>]+country/i);
      expect(source).not.toContain("setActiveCountry");
    }
  });

  it("remounts country drafts on canonical country identity", () => {
    expect(profilePage).toContain("key={`${projectId}-${data.id}`}");
    expect(politicsPage).toContain("key={`${projectId}-${data.id}`}");
  });

  it("registers route-owned drafts with the shared unsaved-change guard", () => {
    expect(profileEditor).toContain(
      'useRegisterDirty("country-profile", isDirty)',
    );
    expect(profileWorkbench).toContain("<CountryProfileEditor");
    expect(politicsWorkbench).toContain(
      'useRegisterDirty("country-politics", isDirty)',
    );
  });
});
