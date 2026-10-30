import { describe, expect, it } from "vitest";
import type { CountryRow } from "@/lib/api";

import {
  countryProfilePatch,
  countryToProfileDraft,
  profileCountryPatchMatches,
  profileCountryReadbackMatches,
  visibleCountryName,
} from "./model";

const profileDraft = {
  nameEn: "Romania",
  nameZh: "罗马尼亚",
  displayNameLocs: { en: "Romania", zh: "罗马尼亚" },
  adjectiveEn: "Romanian",
  adjectiveZh: "罗马尼亚的",
  adjectiveLocs: { en: "Romanian", zh: "罗马尼亚的" },
  ideologyNameLocs: {
    democratic: { en: "Kingdom of Romania", zh: "罗马尼亚王国" },
    fascism: { en: "Legionary Romania", zh: "军团罗马尼亚" },
  },
  color: [180, 154, 108],
  colorUi: [200, 170, 120],
  graphicalCulture: "western_european_gfx",
  capital: 46,
};

function profileCountry(overrides: Record<string, unknown> = {}): CountryRow {
  return {
    id: 1,
    project_id: 2,
    tag: "ROM",
    name: "Romania",
    ideology: "neutrality",
    flag_image_path: "",
    name_en: profileDraft.nameEn,
    name_zh: profileDraft.nameZh,
    display_name_locs: profileDraft.displayNameLocs,
    adjective_en: profileDraft.adjectiveEn,
    adjective_zh: profileDraft.adjectiveZh,
    adjective_locs: profileDraft.adjectiveLocs,
    ideology_name_locs: profileDraft.ideologyNameLocs,
    color: profileDraft.color,
    color_ui: profileDraft.colorUi,
    graphical_culture: profileDraft.graphicalCulture,
    ...overrides,
  } as CountryRow;
}

describe("country profile dynamic ROOT fallback", () => {
  const country = {
    id: 1,
    project_id: 2,
    tag: "ROM",
    name: "Romania",
  } as never;

  const draft = {
    nameEn: "Romania",
    nameZh: "罗马尼亚",
    displayNameLocs: { en: "Romania" },
    adjectiveEn: "Romanian",
    adjectiveZh: "罗马尼亚的",
    adjectiveLocs: {},
    ideologyNameLocs: {
      military_government: { en: "Romanian Military Directorate" },
    },
    color: [180, 154, 108],
    colorUi: [],
    graphicalCulture: "",
    capital: 46,
  };

  it("uses a fifth ROOT dedicated name", () => {
    expect(
      visibleCountryName(country, draft, "military_government", "en"),
    ).toEqual({
      value: "Romanian Military Directorate",
      fallback: false,
    });
  });

  it("falls back to the generic country name when a ROOT has no override", () => {
    expect(visibleCountryName(country, draft, "democratic", "en")).toEqual({
      value: "Romania",
      fallback: true,
    });
  });
});

describe("country profile semantic readback", () => {
  it("ignores reverse insertion order for localization maps", () => {
    const readback = profileCountry({
      display_name_locs: { zh: "罗马尼亚", en: "Romania" },
      adjective_locs: { zh: "罗马尼亚的", en: "Romanian" },
    });

    expect(profileCountryReadbackMatches(readback, profileDraft)).toBe(true);
  });

  it("ignores insertion order at every nested object level", () => {
    const readback = profileCountry({
      ideology_name_locs: {
        fascism: { zh: "军团罗马尼亚", en: "Legionary Romania" },
        democratic: { zh: "罗马尼亚王国", en: "Kingdom of Romania" },
      },
    });

    expect(profileCountryReadbackMatches(readback, profileDraft)).toBe(true);
  });

  it("keeps array order significant", () => {
    const readback = profileCountry({ color: [108, 154, 180] });

    expect(profileCountryReadbackMatches(readback, profileDraft)).toBe(false);
  });

  it("rejects real nested value differences", () => {
    const readback = profileCountry({
      ideology_name_locs: {
        democratic: { en: "Romanian Republic", zh: "罗马尼亚王国" },
        fascism: { en: "Legionary Romania", zh: "军团罗马尼亚" },
      },
    });

    expect(profileCountryReadbackMatches(readback, profileDraft)).toBe(false);
  });
});

describe("country profile effective fallbacks", () => {
  const sparseCountry = profileCountry({
    tag: "KOR",
    name: "KOR",
    name_en: "KOR",
    name_zh: "",
    display_name_locs: {},
    adjective_en: "",
    adjective_zh: "",
    adjective_locs: {},
    ideology_name_locs: {},
    graphical_culture: "",
    graphical_culture_2d: "",
  });

  it("shows effective game values without materializing untouched inheritance", () => {
    const initial = countryToProfileDraft(sparseCountry, null, {
      name: "Korea",
      adjective: "Korean",
      ideologyNames: { democratic: "Republic of Korea" },
      graphicalCulture: "asian_gfx",
      capital: 525,
    });

    expect(initial).toMatchObject({
      nameEn: "Korea",
      displayNameLocs: { en: "Korea" },
      adjectiveEn: "Korean",
      adjectiveLocs: { en: "Korean" },
      ideologyNameLocs: { democratic: { en: "Republic of Korea" } },
      graphicalCulture: "asian_gfx",
      capital: 525,
    });

    const patch = countryProfilePatch(sparseCountry, initial, {
      ...initial,
      displayNameLocs: { ...initial.displayNameLocs, zh: "韩国" },
    });
    expect(patch).toEqual({ display_name_locs: { zh: "韩国" } });
    expect(
      profileCountryPatchMatches(
        { ...sparseCountry, display_name_locs: { zh: "韩国" } },
        patch,
      ),
    ).toBe(true);
    expect(
      profileCountryPatchMatches(
        { ...sparseCountry, display_name_locs: { zh: "朝鲜" } },
        patch,
      ),
    ).toBe(false);
  });

  it("keeps the 2D art set paired when graphical culture is intentionally changed", () => {
    const initial = countryToProfileDraft(sparseCountry, null, {
      graphicalCulture: "asian_gfx",
    });
    expect(
      countryProfilePatch(sparseCountry, initial, {
        ...initial,
        graphicalCulture: "middle_eastern_gfx",
      }),
    ).toEqual({
      graphical_culture: "middle_eastern_gfx",
      graphical_culture_2d: "middle_eastern_2d",
    });
  });
});
