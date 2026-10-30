import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { SWRConfig } from "swr";
import { IntlProvider } from "use-intl";
import { afterEach, describe, expect, it, vi } from "vitest";
import desktop from "../../messages/en/desktop.json";

const fetchMock = vi.fn();
vi.stubGlobal("fetch", fetchMock);

import { GameIndexStatusPanel } from "./GameIndexStatus";

const answer = (body: unknown) => ({
  ok: true,
  status: 200,
  json: async () => body,
});
const status = (over: Record<string, unknown>) => ({
  state: "ready",
  game_version: "1.19.3.0",
  step: "",
  done: 34,
  total: 34,
  error: "",
  built_at: "",
  ...over,
});

function renderPanel() {
  return render(
    <SWRConfig value={{ provider: () => new Map(), dedupingInterval: 0 }}>
      <IntlProvider locale="en" messages={{ desktop }} timeZone="UTC">
        <GameIndexStatusPanel />
      </IntlProvider>
    </SWRConfig>,
  );
}

afterEach(() => {
  cleanup();
  fetchMock.mockReset();
});

describe("the game index panel", () => {
  it("shows how far a build is, without a rebuild button", async () => {
    fetchMock.mockResolvedValue(
      answer(status({ state: "building", step: "country_tags", done: 12 })),
    );
    renderPanel();
    expect(await screen.findByText("Reading your game… 12 of 34")).toBeTruthy();
    expect(screen.getByText("Now reading: country_tags")).toBeTruthy();
    expect(screen.queryByRole("button", { name: "Rebuild" })).toBeNull();
  });

  it("shows a failed build's error and rebuilds on request", async () => {
    fetchMock
      .mockResolvedValueOnce(
        answer(
          status({
            state: "failed",
            error: "country_tags: missing history/countries",
          }),
        ),
      )
      .mockResolvedValue(answer(status({ state: "building", done: 0 })));
    renderPanel();
    expect(
      await screen.findByText("The lists could not be built."),
    ).toBeTruthy();
    expect(
      screen.getByText("country_tags: missing history/countries"),
    ).toBeTruthy();

    fireEvent.click(screen.getByRole("button", { name: "Rebuild" }));
    expect(await screen.findByText("Reading your game… 0 of 34")).toBeTruthy();
    expect(fetchMock).toHaveBeenCalledWith(
      "/api/app/game-index/rebuild",
      expect.objectContaining({ method: "POST" }),
    );
  });
});
