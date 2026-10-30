import { beforeEach, describe, expect, it, vi } from "vitest";

const countryApi = vi.hoisted(() => ({ takeIn: vi.fn() }));
vi.mock("@/lib/api", async (original) => ({
  ...(await original<typeof import("@/lib/api")>()),
  countriesApi: countryApi,
}));

import { ApiError } from "@/lib/api";
import { recordChange, useChangeLog } from "./changeLog";
import {
  chosenCountry,
  onCountryTakenIn,
  resetCountryTakeIn,
  speakFor,
  takeInCountry,
} from "./countryTakeIn";

beforeEach(() => {
  vi.clearAllMocks();
  resetCountryTakeIn();
  useChangeLog.setState({ entries: [] });
  countryApi.takeIn.mockResolvedValue({
    country: { tag: "GER" },
    created: true,
  });
});

describe("country take-in", () => {
  it("takes the chosen country in on a save, once, and links the workbench to it", async () => {
    const heard: string[] = [];
    onCountryTakenIn((projectId, tag) => heard.push(`${projectId}:${tag}`));
    speakFor(7, "ger");

    recordChange({
      projectId: 7,
      title: "Infanterie-Division",
      workbenchHref: "/project/7/division-templates?t=3",
    });
    recordChange({
      projectId: 7,
      title: "Germany",
      workbenchHref: "/project/7/country-profile",
    });
    await vi.waitFor(() => expect(heard).toEqual(["7:GER"]));

    expect(countryApi.takeIn).toHaveBeenCalledTimes(1);
    expect(countryApi.takeIn).toHaveBeenCalledWith(7, "GER");
    expect(
      useChangeLog.getState().entries.map((entry) => entry.workbenchHref),
    ).toEqual([
      "/project/7/country-profile?country=GER",
      "/project/7/division-templates?t=3&country=GER",
    ]);
  });

  it("takes nothing in while the screens only fall back to a country", () => {
    speakFor(7, null);
    recordChange({
      projectId: 7,
      title: "Main menu",
      workbenchHref: "/project/7/start",
    });

    expect(countryApi.takeIn).not.toHaveBeenCalled();
    expect(useChangeLog.getState().entries[0].workbenchHref).toBe(
      "/project/7/start",
    );
    expect(chosenCountry(7)).toBeNull();
  });

  it("does not ask again for a country the game does not have, but retries other failures", async () => {
    countryApi.takeIn.mockRejectedValueOnce(
      new ApiError(422, "The game has no country ZQX", null),
    );
    expect(await takeInCountry(7, "ZQX")).toBe(false);
    expect(await takeInCountry(7, "ZQX")).toBe(false);
    expect(countryApi.takeIn).toHaveBeenCalledTimes(1);

    countryApi.takeIn.mockRejectedValueOnce(new Error("offline"));
    expect(await takeInCountry(7, "FRA")).toBe(false);
    expect(await takeInCountry(7, "FRA")).toBe(true);
    expect(countryApi.takeIn).toHaveBeenCalledTimes(3);
  });

  it("is quiet about a country the project already had", async () => {
    countryApi.takeIn.mockResolvedValue({
      country: { tag: "GER" },
      created: false,
    });
    const heard = vi.fn();
    onCountryTakenIn(heard);

    expect(await takeInCountry(7, "GER")).toBe(true);
    expect(heard).not.toHaveBeenCalled();
  });
});
