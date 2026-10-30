import { describe, it, expect } from "vitest";

import {
  HOI4_SUPPORTED_LANGS,
  LANG_DISPLAY,
  availableLangsToAdd,
  isLegacyFallback,
  localisationValueStatus,
  orderLanguages,
  resolveActiveValue,
} from "./displayNameLocsHelpers";

describe("orderLanguages · force-EN rule (PRD decision 2 Z)", () => {
  it("puts EN first even if user export_languages omits it", () => {
    expect(orderLanguages(["zh", "ru"])).toEqual(["en", "zh", "ru"]);
  });

  it("preserves declared order for non-EN langs", () => {
    expect(orderLanguages(["en", "ru", "zh"])).toEqual(["en", "ru", "zh"]);
  });

  it("dedupes EN if user listed it explicitly", () => {
    expect(orderLanguages(["en", "zh"])).toEqual(["en", "zh"]);
  });

  it("appends extra value-keys (round-trip support) after declared tabs", () => {
    expect(orderLanguages(["en", "zh"], ["en", "zh", "fr", "de"])).toEqual([
      "en",
      "zh",
      "fr",
      "de",
    ]);
  });

  it("works with empty exportLanguages (degenerate · still shows EN)", () => {
    expect(orderLanguages([])).toEqual(["en"]);
  });
});

describe("resolveActiveValue · legacy fallback chain", () => {
  it("returns display_name_locs value when present", () => {
    expect(resolveActiveValue({ en: "Mao Zedong", zh: "毛泽东" }, "en")).toBe(
      "Mao Zedong",
    );
  });

  it("falls back to legacy name_en when display_name_locs empty", () => {
    expect(resolveActiveValue({}, "en", { en: "Erwin Rommel" })).toBe(
      "Erwin Rommel",
    );
  });

  it("falls back to legacy when display_name_locs lang is empty string", () => {
    expect(
      resolveActiveValue({ en: "", zh: "毛泽东" }, "en", { en: "Mao" }),
    ).toBe("Mao");
  });

  it("returns empty string when both miss", () => {
    expect(resolveActiveValue({}, "fr", { en: "x" })).toBe("");
  });

  it("tolerates null / undefined value", () => {
    expect(resolveActiveValue(null, "en", { en: "X" })).toBe("X");
    expect(resolveActiveValue(undefined, "en", { en: "X" })).toBe("X");
  });

  it("tolerates missing legacyFallback prop", () => {
    expect(resolveActiveValue({}, "en")).toBe("");
  });
});

describe("availableLangsToAdd · dropdown filter", () => {
  it("returns langs not in current tabs", () => {
    const result = availableLangsToAdd(["en", "zh"]);
    expect(result).toContain("ru");
    expect(result).toContain("fr");
    expect(result).not.toContain("en");
    expect(result).not.toContain("zh");
  });

  it("returns empty when all HOI4 langs already shown", () => {
    expect(availableLangsToAdd([...HOI4_SUPPORTED_LANGS])).toEqual([]);
  });

  it("preserves HOI4_SUPPORTED_LANGS ordering", () => {
    const result = availableLangsToAdd(["en"]);

    expect(result[0]).toBe("zh");
  });
});

describe("isLegacyFallback · visual cue helper", () => {
  it("true when value missing but legacy has it", () => {
    expect(isLegacyFallback({}, "en", { en: "X" })).toBe(true);
  });

  it("false when value has it", () => {
    expect(isLegacyFallback({ en: "X" }, "en", { en: "Y" })).toBe(false);
  });

  it("false when neither has it", () => {
    expect(isLegacyFallback({}, "en", {})).toBe(false);
    expect(isLegacyFallback({}, "en")).toBe(false);
  });
});

describe("localisationValueStatus · per-language provenance", () => {
  it("distinguishes imported canonical values from saved values", () => {
    expect(localisationValueStatus({ de: "Name" }, "de", {}, "imported")).toBe(
      "imported",
    );
    expect(localisationValueStatus({ de: "Name" }, "de", {}, "saved")).toBe(
      "saved",
    );
  });

  it("surfaces legacy and genuinely missing slots without cross-language fallback", () => {
    expect(localisationValueStatus({}, "en", { en: "Legacy" })).toBe("legacy");
    expect(
      localisationValueStatus({ en: "English" }, "de", { en: "Legacy" }),
    ).toBe("missing");
  });
});

describe("LANG_DISPLAY · tab label map", () => {
  it("uses CJK glyph for zh (mod authors expect this)", () => {
    expect(LANG_DISPLAY.zh).toBe("中");
  });

  it("uses ISO upper-case for the rest", () => {
    expect(LANG_DISPLAY.en).toBe("EN");
    expect(LANG_DISPLAY.ru).toBe("RU");
  });

  it("covers every HOI4_SUPPORTED_LANGS entry", () => {
    for (const lang of HOI4_SUPPORTED_LANGS) {
      expect(LANG_DISPLAY[lang]).toBeDefined();
    }
  });
});
