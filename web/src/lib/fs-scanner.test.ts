import { describe, it, expect } from "vitest";
import { readFileSync, readdirSync } from "node:fs";
import { resolve } from "node:path";
import {
  filenameMatchesTag,
  filenameMatchesTagOrAlias,
  countDeclaredCountryTags,
} from "./fs-scanner";

describe("filenameMatchesTag", () => {
  it("matches vanilla-style TAG.txt", () => {
    expect(filenameMatchesTag("DSR.txt", "DSR")).toBe(true);
  });

  it("matches dash-separated TAG-Name.txt", () => {
    expect(filenameMatchesTag("DSR-German Socialist Republic.txt", "DSR")).toBe(
      true,
    );
  });

  it("matches space-dash-space TAG - Name.txt (TNO/KR alt-hist convention) — regression case", () => {
    expect(
      filenameMatchesTag("DSR - German Socialist Republic.txt", "DSR"),
    ).toBe(true);
  });

  it("matches underscore TAG_decisions.txt", () => {
    expect(filenameMatchesTag("DSR_decisions.txt", "DSR")).toBe(true);
  });

  it("matches reversed decisions_TAG.txt", () => {
    expect(filenameMatchesTag("decisions_DSR.txt", "DSR")).toBe(true);
  });

  it("rejects a different TAG-Name file", () => {
    expect(filenameMatchesTag("GER-Germany.txt", "DSR")).toBe(false);
  });

  it("rejects space-dash-space file for a different TAG", () => {
    expect(
      filenameMatchesTag("GCF - République communiste française.txt", "DSR"),
    ).toBe(false);
  });

  it("rejects substring match (AFRICA.txt for FRA) — anti-substring guard still holds", () => {
    expect(filenameMatchesTag("AFRICA.txt", "FRA")).toBe(false);
  });

  it("is case-insensitive on both filename and tag", () => {
    expect(filenameMatchesTag("dsr - german.txt", "dsr")).toBe(true);
    expect(filenameMatchesTag("DSR - German.txt", "dsr")).toBe(true);
    expect(filenameMatchesTag("dsr - german.txt", "DSR")).toBe(true);
  });

  it("collapses multiple consecutive whitespace separators", () => {
    expect(
      filenameMatchesTag("DSR    German Socialist Republic.txt", "DSR"),
    ).toBe(true);
  });
});

describe("filenameMatchesTagOrAlias", () => {
  it("flows TAG match through for space-dash-space TNO convention", () => {
    expect(
      filenameMatchesTagOrAlias("DSR - German Socialist Republic.txt", "DSR"),
    ).toBe(true);
  });

  it("still rejects unrelated TAG in space-dash-space file", () => {
    expect(
      filenameMatchesTagOrAlias(
        "GCF - République communiste française.txt",
        "DSR",
      ),
    ).toBe(false);
  });
});

describe("countDeclaredCountryTags", () => {
  it("counts TAG declarations, not lines/files; skips comments + dynamic flag; dedups across files", () => {
    const fileA = [
      "# header comment",
      'GER = "countries/Germany.txt"',
      'ENG = "countries/United Kingdom.txt"   # trailing comment',
      "",
      "dynamic_tags = yes",
      'D01 = "countries/D01.txt"',
    ].join("\n");
    const fileB = [
      'GER = "countries/Germany.txt"',
      'FRA = "countries/France.txt"',
    ].join("\n");

    expect(countDeclaredCountryTags([fileA, fileB])).toBe(4);
  });

  it("strips UTF-8 BOM on the first line", () => {
    expect(countDeclaredCountryTags(['﻿GER = "countries/Germany.txt"'])).toBe(
      1,
    );
  });

  const LS = String.fromCharCode(0x2028);
  const PS = String.fromCharCode(0x2029);

  const SEP_DRIFT_INPUT =
    'GER = "countries/Germany.txt"' +
    "\r" +
    'ENG = "countries/UK.txt"' +
    "\x0b" +
    'FRA = "countries/France.txt"' +
    "\x85" +
    "dynamic_tags = yes" +
    "\x1c" +
    'D01 = "countries/D01.txt"' +
    LS +
    'USSR = "countries/Soviet.txt"' +
    PS +
    'USSR2 = "countries/Soviet2.txt"' +
    "\x1e" +
    'SP = "countries/Spain.txt"' +
    "\x0c" +
    'GER = "countries/Germany.txt"';

  const SEP_DRIFT_EXPECTED = 7;
  it(`counts ${SEP_DRIFT_EXPECTED} TAGs across exotic line separators (CR/VT/NEL/FS/LS/PS/FF, == backend; old /\\r?\\n/ would count 0)`, () => {
    expect(countDeclaredCountryTags([SEP_DRIFT_INPUT])).toBe(
      SEP_DRIFT_EXPECTED,
    );
  });

  it("regression: a naive LF/CRLF-only split would count 0 on bare-CR-delimited input", () => {
    const naive = SEP_DRIFT_INPUT.split(/\r?\n/);

    const matched = naive.filter((l) =>
      /^\s*([A-Za-z0-9]{2,4})\s*=\s*"([^"]+)"\s*(?:#.*)?$/.test(l),
    ).length;
    expect(matched).toBeLessThan(SEP_DRIFT_EXPECTED);
  });

  const EXPECTED_TFR_TAGS = 610;
  it(`counts ${EXPECTED_TFR_TAGS} distinct TAGs on the real TFR fixture (== backend, was 3 file-count)`, () => {
    const fixtureDir = resolve(__dirname, "__fixtures__/tfr_country_tags");
    const contents = readdirSync(fixtureDir)
      .filter((f) => f.endsWith(".txt"))
      .sort()
      .map((f) => readFileSync(resolve(fixtureDir, f), "utf-8"));
    expect(contents.length).toBe(3);
    expect(countDeclaredCountryTags(contents)).toBe(EXPECTED_TFR_TAGS);
  });
});
