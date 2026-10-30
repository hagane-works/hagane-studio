import { describe, expect, it } from "vitest";

import {
  COMMON_RULE_DIRS,
  COUNTRY_AUDIT_PREFIXES,
  emptyScan,
  selectCountryAuditFiles,
} from "./scan";

function walkFile(path: string) {
  return {
    path,
    file: { name: path.split("/").at(-1) ?? path } as File,
  };
}

describe("import scanner contract projection", () => {
  it("keeps common/countries recursive and available to the audit payload", () => {
    const projectedDirs =
      process.env.Q1_SCANNER_NEGATIVE_CONTROL === "drop-countries"
        ? COMMON_RULE_DIRS.filter((entry) => entry !== "countries")
        : COMMON_RULE_DIRS;

    expect(projectedDirs).toContain("countries");
    expect(COUNTRY_AUDIT_PREFIXES).toEqual([
      "common/country_tags/",
      "common/countries/",
    ]);

    const scan = emptyScan();
    scan.country_tags.files = [walkFile("common/country_tags/00_tags.txt")];
    scan.common_rules.files = [
      walkFile("common/countries/colors.txt"),
      walkFile("common/countries/nested/Germany.txt"),
      walkFile("common/on_actions/00_hooks.txt"),
    ];

    expect(selectCountryAuditFiles(scan).map((entry) => entry.path)).toEqual([
      "common/country_tags/00_tags.txt",
      "common/countries/colors.txt",
      "common/countries/nested/Germany.txt",
    ]);
  });
});
