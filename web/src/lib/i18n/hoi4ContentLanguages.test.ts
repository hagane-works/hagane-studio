import { describe, expect, it } from "vitest";
import {
  contentLocaleFromLocalisationPath,
  detectedContentLocales,
  mergeProjectContentLanguages,
  shouldImportLocalisationPath,
} from "./hoi4ContentLanguages";

describe("HOI4 content language paths", () => {
  it.each([
    ["localisation/english/mod_l_english.yml", "en"],
    ["localisation/simp_chinese/mod_l_simp_chinese.yml", "zh"],
    ["localisation/german/mod_l_german.yml", "de"],
    ["localisation/russian/mod_l_russian.yml", "ru"],
    ["localisation/french/mod_l_french.yml", "fr"],
    ["localisation/spanish/mod_l_spanish.yml", "es"],
    ["localisation/japanese/mod_l_japanese.yml", "ja"],
    ["localisation/korean/mod_l_korean.yml", "ko"],
  ])("maps %s to %s", (path, expected) => {
    expect(contentLocaleFromLocalisationPath(path)).toBe(expected);
  });

  it("detects each language once in stable HOI4 order", () => {
    expect(
      detectedContentLocales([
        "localisation/german/a_l_german.yml",
        "localisation/english/a_l_english.yml",
        "localisation/german/b_l_german.yml",
      ]),
    ).toEqual(["en", "de"]);
  });

  it("imports only selected known languages and preserves unknown layouts", () => {
    expect(
      shouldImportLocalisationPath("localisation/german/a_l_german.yml", [
        "en",
      ]),
    ).toBe(false);
    expect(
      shouldImportLocalisationPath("localisation/english/a_l_english.yml", [
        "en",
      ]),
    ).toBe(true);
    expect(
      shouldImportLocalisationPath("localisation/custom.yml", ["en"]),
    ).toBe(true);
  });

  it("keeps English first and preserves existing plus newly selected languages", () => {
    expect(mergeProjectContentLanguages(["fr", "en"], ["de", "fr"])).toEqual([
      "en",
      "fr",
      "de",
    ]);
    expect(mergeProjectContentLanguages(undefined, ["ko"])).toEqual([
      "en",
      "ko",
    ]);
  });
});
