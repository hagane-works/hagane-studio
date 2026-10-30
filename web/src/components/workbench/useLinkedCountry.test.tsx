import React from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, render, waitFor } from "@testing-library/react";

const countryApi = vi.hoisted(() => ({ takeIn: vi.fn() }));
vi.mock("@/lib/api/scenario", async (original) => ({
  ...(await original<typeof import("@/lib/api/scenario")>()),
  countriesApi: countryApi,
}));

import { useProjectStore } from "@/lib/store/projectStore";
import { useLinkedCountry } from "./useLinkedCountry";

function Probe() {
  useLinkedCountry(
    7,
    useProjectStore((s) => s.currentProject?.id === 7),
  );
  return null;
}

const project = (tags: string[]) =>
  ({ id: 7, country_tag: tags[0] ?? "", country_tags: tags }) as never;

beforeEach(() => {
  vi.clearAllMocks();
  window.history.replaceState(
    null,
    "",
    "/zh/project/7/division-templates?t=3&country=GER",
  );
});

afterEach(cleanup);

describe("useLinkedCountry", () => {
  it("takes the linked country in, switches to it and drops the param", async () => {
    const fetchProject = vi.fn(async () => {
      useProjectStore.setState({ currentProject: project(["GER"]) });
    });
    useProjectStore.setState({
      currentProject: project([]),
      activeCountryTag: null,
      fetchProject,
    });
    countryApi.takeIn.mockResolvedValue({
      country: { tag: "GER" },
      created: true,
    });

    render(<Probe />);

    await waitFor(() =>
      expect(useProjectStore.getState().activeCountryTag).toBe("GER"),
    );
    expect(countryApi.takeIn).toHaveBeenCalledWith(7, "GER");
    expect(fetchProject).toHaveBeenCalledWith(7);
    expect(window.location.pathname + window.location.search).toBe(
      "/zh/project/7/division-templates?t=3",
    );
  });

  it("only switches to a country the project already has", async () => {
    useProjectStore.setState({
      currentProject: project(["ENG", "GER"]),
      activeCountryTag: "ENG",
    });

    render(<Probe />);

    await waitFor(() =>
      expect(useProjectStore.getState().activeCountryTag).toBe("GER"),
    );
    expect(countryApi.takeIn).not.toHaveBeenCalled();
  });

  it("stays on the current country when the game has no such country", async () => {
    window.history.replaceState(null, "", "/zh/project/7/start?country=ZQX");
    useProjectStore.setState({
      currentProject: project(["ENG"]),
      activeCountryTag: "ENG",
    });
    countryApi.takeIn.mockRejectedValue(
      new Error("The game has no country ZQX"),
    );

    render(<Probe />);

    await waitFor(() => expect(window.location.search).toBe(""));
    expect(useProjectStore.getState().activeCountryTag).toBe("ENG");
  });
});
