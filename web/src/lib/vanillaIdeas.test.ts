import { describe, expect, it, vi } from "vitest";

const fetchMock = vi.fn();
vi.stubGlobal("fetch", fetchMock);

import {
  getVanillaIdeasByGroup,
  getVanillaIdeasForCountry,
  isVanillaIdea,
  isVanillaIdeasLoaded,
  loadVanillaIdeas,
} from "./vanillaIdeas";

describe("the game's idea list", () => {
  it("answers nothing before it is loaded, then reads the index once", async () => {
    expect(isVanillaIdeasLoaded()).toBe(false);
    expect(isVanillaIdea("GER_example")).toBe(false);
    expect(getVanillaIdeasForCountry("GER")).toEqual([]);

    fetchMock.mockResolvedValue({
      ok: true,
      status: 200,
      json: async () => ({
        meta: {
          total: 2,
          source: "common/ideas",
          groups: { country_spirits: 1, laws: 1 },
        },
        by_group: {
          country_spirits: [["GER_example", "country", "GER"]],
          laws: [["civilian_economy", "economy", "shared"]],
        },
        by_country: {
          GER: [["GER_example", "country", "country_spirits"]],
          shared: [["civilian_economy", "economy", "laws"]],
        },
        all_ids: ["GER_example", "civilian_economy"],
      }),
    });
    await Promise.all([loadVanillaIdeas(), loadVanillaIdeas()]);
    await loadVanillaIdeas();

    expect(fetchMock).toHaveBeenCalledTimes(1);
    expect(fetchMock).toHaveBeenCalledWith(
      "/api/go/vanilla/ideas/index",
      expect.anything(),
    );
    expect(isVanillaIdeasLoaded()).toBe(true);
    expect(isVanillaIdea("GER_example")).toBe(true);
    expect(getVanillaIdeasForCountry("GER").map((idea) => idea.id)).toEqual([
      "GER_example",
    ]);
    expect(
      getVanillaIdeasByGroup("laws").map((idea) => [idea.id, idea.country]),
    ).toEqual([["civilian_economy", "shared"]]);
  });
});
