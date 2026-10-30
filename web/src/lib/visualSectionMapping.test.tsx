import { describe, expect, it, vi } from "vitest";
import { renderHook, waitFor } from "@testing-library/react";

const fetchMock = vi.fn();
vi.stubGlobal("fetch", fetchMock);

import {
  resolveVisualSection,
  useTechSections,
  type TechSections,
} from "./visualSectionMapping";

const SECTIONS: TechSections = {
  armour_folder: {
    count: 2,
    sub_sections: [
      {
        name: "armortech_subtitle_light",
        sub_key: "light",
        y: 0,
        text_key: "ARMOR_TITLE_LIGHT",
        names: { en: "Light Tank", de: "Leichter Panzer" },
      },
      {
        name: "armortech_subtitle_heavy",
        sub_key: "heavy",
        y: 100,
        text_key: "ARMOR_TITLE_HEAVY",
        names: { en: "Heavy Tank" },
      },
    ],
  },
};

describe("tech-tree sub-sections", () => {
  it("are read once from the game index (GET /api/go/vanilla/tech-sections)", async () => {
    fetchMock.mockResolvedValue({
      ok: true,
      status: 200,
      json: async () => ({ game_version: "1.16.9", folders: SECTIONS }),
    });
    const { result } = renderHook(() => useTechSections());
    expect(result.current).toEqual({});
    await waitFor(() => expect(result.current).toEqual(SECTIONS));
    expect(fetchMock).toHaveBeenCalledTimes(1);
    expect(fetchMock).toHaveBeenCalledWith(
      "/api/go/vanilla/tech-sections",
      expect.anything(),
    );
  });

  it("group a tech by its y band and carry the game's names", () => {
    const heavy = resolveVisualSection(
      { tech_id: "basic_heavy_tank", position_y: 120, categories: [] },
      "armour_folder",
      SECTIONS,
    );
    expect(heavy).toEqual({
      subKey: "heavy",
      names: { en: "Heavy Tank" },
      via: "y-band",
    });

    const early = resolveVisualSection(
      {
        tech_id: "basic_light_tank",
        position_y: 0,
        categories: ["cat_light_armor"],
      },
      "armour_folder",
      {},
    );
    expect(early).toEqual({ subKey: "light", names: {}, via: "category" });
  });
});
