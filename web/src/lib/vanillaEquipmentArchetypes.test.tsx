import { describe, expect, it, vi } from "vitest";
import { renderHook, waitFor } from "@testing-library/react";

const fetchMock = vi.fn();
vi.stubGlobal("fetch", fetchMock);

import {
  detectInstanceArchetype,
  useVanillaEquipmentArchetypes,
} from "./vanillaEquipmentArchetypes";

describe("the game's equipment archetypes", () => {
  it("are empty until the index answers, then known", async () => {
    fetchMock.mockResolvedValue({
      ok: true,
      status: 200,
      json: async () => ({
        total: 2,
        archetypes: ["infantry_equipment", "light_tank_chassis"],
      }),
    });
    const { result } = renderHook(() => useVanillaEquipmentArchetypes());
    expect(result.current.size).toBe(0);

    await waitFor(() => expect(result.current.size).toBe(2));
    expect(fetchMock).toHaveBeenCalledWith(
      "/api/go/vanilla/equipment-archetypes",
      expect.anything(),
    );
    expect(
      detectInstanceArchetype("infantry_equipment_1", result.current),
    ).toBe("infantry_equipment");
    expect(
      detectInstanceArchetype("infantry_equipment", result.current),
    ).toBeNull();
    expect(detectInstanceArchetype("mod_rifle_2", result.current)).toBeNull();
  });
});
