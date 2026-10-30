import { describe, expect, it, vi } from "vitest";

const fetchMock = vi.fn();
vi.stubGlobal("fetch", fetchMock);

import {
  loadVanillaCountryTags,
  parseTagTokens,
  vanillaTagName,
} from "./vanillaCountryTags";

describe("the game's country list", () => {
  it("is read once from the game index and looks tags up in any case", async () => {
    fetchMock.mockResolvedValue({
      ok: true,
      status: 200,
      json: async () => ({
        game_version: "1.16.9",
        total: 2,
        tags: [
          {
            tag: "GER",
            name_en: "German Reich",
            names: { en: "German Reich", de: "Deutsches Reich" },
          },
          { tag: "ZZZ", name_en: "ZZZ", names: {} },
        ],
      }),
    });
    const list = await loadVanillaCountryTags();
    await loadVanillaCountryTags();

    expect(fetchMock).toHaveBeenCalledTimes(1);
    expect(fetchMock).toHaveBeenCalledWith(
      "/api/go/vanilla/country-tags",
      expect.anything(),
    );
    expect(list.tags.map((country) => country.tag)).toEqual(["GER", "ZZZ"]);
    expect(list.lookup("ger")?.tag).toBe("GER");
    expect(list.lookup("FRA")).toBeNull();
    expect(list.lookup(null)).toBeNull();

    const ger = list.lookup("GER")!;
    expect(vanillaTagName(ger, "de")).toBe("Deutsches Reich");
    expect(vanillaTagName(ger, "ja")).toBe("German Reich");
    expect(vanillaTagName(list.lookup("ZZZ")!, "zh")).toBe("ZZZ");
  });

  it("names [TAG:NameWithFlag] tokens from the list, and leaves unknown tags bare", () => {
    const lookup = (tag: string | null | undefined) =>
      tag === "GER"
        ? {
            tag: "GER",
            name_en: "German Reich",
            names: { fr: "Reich allemand" },
          }
        : null;
    const tokens = parseTagTokens(
      "[GER:NameWithFlag] vs [FRA:NameWithFlag]",
      lookup,
      undefined,
      "fr",
    );
    expect(
      tokens.map((token) => [token.tag, token.display, token.flag]),
    ).toEqual([
      ["GER", "Reich allemand", ""],
      ["FRA", "FRA", ""],
    ]);
  });
});
