import { describe, expect, it, vi } from "vitest";
import { renderHook, waitFor } from "@testing-library/react";

const fetchMock = vi.fn();
vi.stubGlobal("fetch", fetchMock);

import { useVanillaUnitNames } from "./vanillaUnitNames";

describe("the game's unit names", () => {
  it("are empty until the index answers, then known", async () => {
    fetchMock.mockResolvedValue({
      ok: true,
      status: 200,
      json: async () => ({
        game_version: "1.16.9",
        subunits: { infantry: { en: "Infantry", zh: "步兵" } },
        equipment: {
          infantry_equipment: { en: "Infantry Equipment", ja: "歩兵装備" },
        },
      }),
    });
    const { result } = renderHook(() => useVanillaUnitNames());
    expect(result.current).toEqual({ subunits: {}, equipment: {} });

    await waitFor(() =>
      expect(result.current.subunits.infantry?.zh).toBe("步兵"),
    );
    expect(result.current.equipment.infantry_equipment?.ja).toBe("歩兵装備");
    expect(fetchMock).toHaveBeenCalledWith(
      "/api/go/vanilla/unit-names",
      expect.anything(),
    );
  });
});
