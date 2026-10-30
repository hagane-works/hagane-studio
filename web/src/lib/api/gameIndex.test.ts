import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

const fetchMock = vi.fn();
vi.stubGlobal("fetch", fetchMock);

import { ApiError } from "./_client";
import {
  isGameFolderMissing,
  isGameIndexBuilding,
  requestGameIndexed,
  sessionGameData,
} from "./gameIndex";

const answer = (status: number, body: unknown) => ({
  ok: status < 400,
  status,
  statusText: "",
  json: async () => body,
});
const building = () =>
  answer(503, {
    detail: "the game index is still being built",
    code: "game_index_building",
  });

beforeEach(() => {
  fetchMock.mockReset();
  vi.useFakeTimers();
});
afterEach(() => {
  vi.useRealTimers();
});

describe("routes that read the game index", () => {
  it("waits while the index is being built, then answers", async () => {
    fetchMock
      .mockResolvedValueOnce(building())
      .mockResolvedValueOnce(building())
      .mockResolvedValueOnce(answer(200, { tags: ["GER"] }));
    const done = requestGameIndexed<{ tags: string[] }>("/go/tags?q=");
    await vi.runAllTimersAsync();
    await expect(done).resolves.toEqual({ tags: ["GER"] });
    expect(fetchMock).toHaveBeenCalledTimes(3);
    expect(fetchMock).toHaveBeenCalledWith(
      "/api/go/tags?q=",
      expect.objectContaining({ credentials: "include" }),
    );
  });

  it("does not wait for an index that could not be built", async () => {
    fetchMock.mockResolvedValue(
      answer(503, { detail: "boom", code: "game_index_failed" }),
    );
    const failed = requestGameIndexed("/go/vanilla/ideas/index").catch(
      (reason: unknown) => reason,
    );
    const reason = await failed;
    expect(reason).toBeInstanceOf(ApiError);
    expect(isGameIndexBuilding(reason)).toBe(false);
    expect(fetchMock).toHaveBeenCalledTimes(1);
  });

  it("tells the building answer apart", () => {
    expect(
      isGameIndexBuilding(
        new ApiError(503, "x", { detail: "x", code: "game_index_building" }),
      ),
    ).toBe(true);
    expect(isGameIndexBuilding(new ApiError(503, "x", "x"))).toBe(false);
    expect(isGameIndexBuilding(new Error("x"))).toBe(false);
  });

  it("tells a refusal for want of a game folder apart", async () => {
    fetchMock.mockResolvedValue(
      answer(409, {
        detail: "Set the Hearts of Iron IV folder in Settings first",
        code: "game_folder_missing",
      }),
    );
    const { focusTreesApi } = await import("./focus-tech");
    const reason = await focusTreesApi
      .create(7)
      .catch((error: unknown) => error);
    expect(isGameFolderMissing(reason)).toBe(true);
    expect(
      isGameFolderMissing(
        new ApiError(409, "x", { detail: "x", code: "other" }),
      ),
    ).toBe(false);
    expect(isGameFolderMissing(new Error("x"))).toBe(false);
  });
});

describe("data kept for the session", () => {
  it("loads once, tells subscribers, and keeps the data", async () => {
    const fetchData = vi.fn(async () => ["GER", "SOV"]);
    const empty: string[] = [];
    const data = sessionGameData(fetchData, empty);
    const heard = vi.fn();
    expect(data.current()).toBe(empty);

    const unsubscribe = data.subscribe(heard);
    await expect(data.load()).resolves.toEqual(["GER", "SOV"]);
    expect(heard).toHaveBeenCalledTimes(1);
    expect(data.current()).toEqual(["GER", "SOV"]);

    unsubscribe();
    await data.load();
    expect(fetchData).toHaveBeenCalledTimes(1);
  });

  it("does not keep a failure: the next caller asks again", async () => {
    const fetchData = vi
      .fn()
      .mockRejectedValueOnce(new Error("offline"))
      .mockResolvedValueOnce(["GER"]);
    const empty: string[] = [];
    const data = sessionGameData<string[]>(fetchData, empty);

    await expect(data.load()).rejects.toThrow("offline");
    expect(data.current()).toBe(empty);
    await expect(data.load()).resolves.toEqual(["GER"]);
    expect(fetchData).toHaveBeenCalledTimes(2);
  });
});
