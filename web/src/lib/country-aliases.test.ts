import { describe, expect, it, vi } from "vitest";

const fetchMock = vi.fn();
vi.stubGlobal("fetch", fetchMock);

import { getCountryAliases, loadCountryAliases } from "./country-aliases";
import { filenameMatchesTagOrAlias } from "./fs-scanner";
import { isForeignCountrySpecificTech } from "./techGraph";

describe("the game's country aliases", () => {
  it("are unknown before they arrive, then read once from the game index", async () => {
    expect(getCountryAliases("GER")).toEqual([]);
    expect(filenameMatchesTagOrAlias("germany.txt", "GER")).toBe(false);

    fetchMock.mockResolvedValue({
      ok: true,
      status: 200,
      json: async () => ({
        game_version: "1.16.9",
        aliases: {
          GER: ["germany"],
          PRC: ["china", "china_communist", "chinacommunist"],
        },
      }),
    });
    const aliases = await loadCountryAliases();
    await loadCountryAliases();

    expect(fetchMock).toHaveBeenCalledTimes(1);
    expect(fetchMock).toHaveBeenCalledWith(
      "/api/go/vanilla/country-aliases",
      expect.anything(),
    );
    expect(getCountryAliases("ger")).toEqual(["germany"]);
    expect([...aliases.tags].sort()).toEqual(["GER", "PRC"]);

    expect(filenameMatchesTagOrAlias("germany.txt", "GER")).toBe(true);
    expect(filenameMatchesTagOrAlias("china_communist_sea.txt", "PRC")).toBe(
      true,
    );
    expect(
      isForeignCountrySpecificTech("PRC_guerilla_tech", "GER", aliases.tags),
    ).toBe(true);
    expect(isForeignCountrySpecificTech("ToA_tech", "GER", aliases.tags)).toBe(
      false,
    );
  });
});
