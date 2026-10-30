import { describe, expect, it, vi } from "vitest";
import { renderHook, waitFor } from "@testing-library/react";

const fetchMock = vi.fn();
vi.stubGlobal("fetch", fetchMock);

import {
  EMPTY_VANILLA_STATE_INDEX,
  useVanillaStateIndex,
} from "./vanillaStateIndex";

const answer = (status: number, body: unknown) => ({
  ok: status < 400,
  status,
  statusText: "",
  json: async () => body,
});

describe("the game's state index", () => {
  it("is empty until both routes answer, waiting while the index is built", async () => {
    let contentAsked = 0;
    fetchMock.mockImplementation(async (url: string) => {
      if (url === "/api/go/vanilla/state-content") {
        contentAsked += 1;
        if (contentAsked === 1)
          return answer(503, {
            detail: "building",
            code: "game_index_building",
          });
        return answer(200, {
          game_version: "1.16.9",
          states: {
            "608": { navalBases: 1, landmarks: ["landmark_forbidden_city"] },
          },
        });
      }
      if (url === "/api/go/vanilla/dlc-states") {
        return answer(200, {
          game_version: "1.16.9",
          dlc_names: ["Waking the Tiger"],
          states: { "1040": ["Waking the Tiger"] },
        });
      }
      return answer(404, { detail: "not found" });
    });

    const { result } = renderHook(() => useVanillaStateIndex());
    expect(result.current).toBe(EMPTY_VANILLA_STATE_INDEX);

    await waitFor(
      () => expect(result.current).not.toBe(EMPTY_VANILLA_STATE_INDEX),
      { timeout: 3000 },
    );
    expect(contentAsked).toBe(2);
    expect(result.current.content[608]).toEqual({
      navalBases: 1,
      landmarks: ["landmark_forbidden_city"],
    });
    expect(result.current.dlc[1040]).toEqual(["Waking the Tiger"]);
    expect(result.current.content[1]).toBeUndefined();
  });
});
