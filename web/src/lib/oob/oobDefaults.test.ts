import { readFileSync } from "node:fs";
import { dirname, join, resolve } from "node:path";
import { fileURLToPath } from "node:url";
import { describe, expect, it } from "vitest";

import {
  VANILLA_START_YEAR,
  defaultOobFileName,
  normalizeCountryTag,
  oobStarterScript,
  pickDefaultOobFile,
  resolveOobFileName,
  startYearFromStartDate,
  tagFromOob,
} from "./oobDefaults";

const here = dirname(fileURLToPath(import.meta.url));
const repoRoot = resolve(here, "../../..");
const messagesRoot = join(repoRoot, "messages");
const LOCALES = ["de", "en", "es", "fr", "ja", "ko", "ru", "zh"] as const;

const oobRaw = (locale: string): Record<string, unknown> =>
  JSON.parse(readFileSync(join(messagesRoot, locale, "oobRaw.json"), "utf8"));

const PROJECT_COUNTRIES = ["GUN"];

describe("H-3 · default oob_file never invents a country", () => {
  it("REGRESSION: the old literal default decoded to a country no project has", () => {
    expect(tagFromOob("new_1936")).toBe("NEW");
    expect(PROJECT_COUNTRIES).not.toContain(tagFromOob("new_1936"));
  });

  it("a brand-new project's default decodes to a country the project really has", () => {
    const oob = pickDefaultOobFile({
      countryTag: "GUN",
      startDate: "1936.1.1.12",
      existing: [],
    });
    expect(oob).toBe("GUN_1936");
    expect(tagFromOob(oob)).toBe("GUN");
    expect(PROJECT_COUNTRIES).toContain(tagFromOob(oob));
  });

  it("honours a non-vanilla project start year", () => {
    expect(
      pickDefaultOobFile({ countryTag: "ROM", startDate: "1939.9.1.12" }),
    ).toBe("ROM_1939");
    expect(tagFromOob("ROM_1939")).toBe("ROM");
  });

  it("falls back to the vanilla start year when the project pins none", () => {
    expect(pickDefaultOobFile({ countryTag: "ROM", startDate: null })).toBe(
      `ROM_${VANILLA_START_YEAR}`,
    );
    expect(VANILLA_START_YEAR).toBe(1936);
  });

  it("reuses an OOB file this country already owns instead of minting a second one", () => {
    const oob = pickDefaultOobFile({
      countryTag: "GER",
      startDate: "1936.1.1.12",
      existing: ["ROM_1936", "GER_start", "GER_1936"],
    });
    expect(oob).toBe("GER_start");
    expect(tagFromOob(oob)).toBe("GER");
  });

  it("never adopts another country's OOB file", () => {
    const oob = pickDefaultOobFile({
      countryTag: "GUN",
      startDate: "1936.1.1.12",
      existing: ["ROM_1936", "new_1936"],
    });
    expect(oob).toBe("GUN_1936");
    expect(tagFromOob(oob)).not.toBe("ROM");
    expect(tagFromOob(oob)).not.toBe("NEW");
  });

  it("returns an EMPTY field rather than a fake tag when the project has no country", () => {
    expect(
      pickDefaultOobFile({ countryTag: "", startDate: "1936.1.1.12" }),
    ).toBe("");
    expect(pickDefaultOobFile({ countryTag: null })).toBe("");
    expect(defaultOobFileName(undefined, "1936.1.1.12")).toBe("");
  });

  it("still reuses an existing file when there is no country tag at all", () => {
    expect(
      pickDefaultOobFile({ countryTag: "", existing: ["", "  ", "ROM_1936"] }),
    ).toBe("ROM_1936");
  });

  it("round-trips for every plausible tag: tagFromOob(default) === the tag", () => {
    for (const tag of ["GUN", "ROM", "USA", "GER", "D01", "SOV"]) {
      for (const startDate of [null, "1936.1.1.12", "1939.9.1", "183.1.1.12"]) {
        const oob = defaultOobFileName(tag, startDate);
        expect(oob, `${tag} @ ${startDate}`).not.toBe("");
        expect(tagFromOob(oob), `${tag} @ ${startDate}`).toBe(tag);
      }
    }
  });

  it("refuses a tag that could not survive the underscore-prefix round-trip", () => {
    expect(normalizeCountryTag("RO_M")).toBe("");
    expect(defaultOobFileName("RO_M", "1936.1.1.12")).toBe("");
    expect(normalizeCountryTag("  rom ")).toBe("ROM");
    expect(defaultOobFileName(" rom ", "1936.1.1.12")).toBe("ROM_1936");
  });
});

describe("startYearFromStartDate", () => {
  it("reads the year segment of a HOI4 date string", () => {
    expect(startYearFromStartDate("1936.1.1.12")).toBe(1936);
    expect(startYearFromStartDate("1939.9.1")).toBe(1939);
    expect(startYearFromStartDate("183.1.1.12")).toBe(183);
  });

  it("returns null (not 0, not NaN) for anything unusable", () => {
    for (const bad of [
      null,
      undefined,
      "",
      "   ",
      "abcd.1.1",
      "12345.1.1",
      "0.1.1",
    ]) {
      expect(startYearFromStartDate(bad), String(bad)).toBeNull();
    }
  });
});

describe("resolveOobFileName", () => {
  it("prefers an OOB file the country already references", () => {
    expect(resolveOobFileName("  GUN_units  ", "GUN", "1936.1.1.12")).toBe(
      "GUN_units",
    );
  });

  it("derives <TAG>_<year> when there is no existing reference", () => {
    expect(resolveOobFileName("", "GUN", "1936.1.1.12")).toBe("GUN_1936");
    expect(resolveOobFileName(null, null, "1936.1.1.12")).toBe("");
  });
});

describe("H-2 · OOB starter snippet never emits a placeholder file name", () => {
  const en = oobRaw("en") as {
    starterComment: string;
    starterNeedsTag: string;
  };

  it("REGRESSION: the literal TAG_1936 is gone from every produced snippet", () => {
    const snippets = [
      oobStarterScript({
        comment: en.starterComment,
        unknownTagComment: en.starterNeedsTag,
        countryTag: "GUN",
        startDate: "1936.1.1.12",
      }),
      oobStarterScript({
        comment: en.starterComment,
        unknownTagComment: en.starterNeedsTag,
        countryTag: "",
        startDate: null,
      }),
      oobStarterScript({
        comment: en.starterComment,
        unknownTagComment: en.starterNeedsTag,
        countryTag: "GUN",
        oobFile: "GUN_units",
        startDate: "1936.1.1.12",
      }),
    ];
    for (const snippet of snippets) {
      expect(snippet).not.toContain("TAG_1936");
    }
  });

  it("points set_oob at <TAG>_<year> built from the file's own country", () => {
    const snippet = oobStarterScript({
      comment: en.starterComment,
      unknownTagComment: en.starterNeedsTag,
      countryTag: "GUN",
      startDate: "1939.9.1.12",
    });
    expect(snippet).toContain('set_oob = "GUN_1939"');

    const active = snippet
      .split("\n")
      .filter((line) => line.trim() && !line.trim().startsWith("#"));
    expect(active).toEqual(['set_oob = "GUN_1939"']);
  });

  it("reuses the OOB file this history file already references", () => {
    const snippet = oobStarterScript({
      comment: en.starterComment,
      unknownTagComment: en.starterNeedsTag,
      countryTag: "GUN",
      oobFile: "GUN_1937_reserve",
      startDate: "1936.1.1.12",
    });
    expect(snippet).toContain('set_oob = "GUN_1937_reserve"');
    expect(snippet).not.toContain('"GUN_1936"');
  });

  it("emits NOTHING active when the country tag is unknown", () => {
    const snippet = oobStarterScript({
      comment: en.starterComment,
      unknownTagComment: en.starterNeedsTag,
      countryTag: "",
      startDate: null,
    });
    const active = snippet
      .split("\n")
      .filter((line) => line.trim() && !line.trim().startsWith("#"));
    expect(active).toEqual([]);
    expect(snippet).toContain(en.starterNeedsTag);

    expect(/(^|\n)\s*set_oob\b/.test(snippet)).toBe(false);
  });

  it("the detector DOES light up once a real set_oob is emitted", () => {
    const snippet = oobStarterScript({
      comment: en.starterComment,
      unknownTagComment: en.starterNeedsTag,
      countryTag: "GUN",
      startDate: "1936.1.1.12",
    });
    expect(/(^|\n)\s*set_oob\b/.test(snippet)).toBe(true);
  });

  it("prefixes a localized comment line with # in every locale", () => {
    for (const locale of LOCALES) {
      const bundle = oobRaw(locale) as {
        starterComment: string;
        starterNeedsTag: string;
      };
      const snippet = oobStarterScript({
        comment: bundle.starterComment,
        unknownTagComment: bundle.starterNeedsTag,
        countryTag: "",
      });
      for (const line of snippet.split("\n")) {
        expect(line.startsWith("#"), `${locale}: ${line}`).toBe(true);
      }
    }
  });
});

describe("the workbenches actually use the shared default", () => {
  const read = (p: string) => readFileSync(join(repoRoot, "src", p), "utf8");
  const deployments = read("app/[locale]/project/[id]/deployments/page.tsx");
  const templates = read(
    "app/[locale]/project/[id]/division-templates/page.tsx",
  );
  const panel = read("components/workbench/OobPassthroughPanel.tsx");

  it("neither workbench hard-codes a default OOB file name any more", () => {
    for (const [name, src] of [
      ["deployments", deployments],
      ["division-templates", templates],
    ] as const) {
      expect(src, name).toContain("pickDefaultOobFile(");
      expect(src.replace(/^\s*\/\/.*$/gm, ""), name).not.toContain(
        '"new_1936"',
      );
    }
  });

  it("the starter button builds its snippet through oobStarterScript", () => {
    expect(panel).toContain("oobStarterScript(");
    expect(panel).not.toContain("TAG_1936");
  });
});
