import { cleanup, render, screen } from "@testing-library/react";
import { SWRConfig } from "swr";
import { IntlProvider } from "use-intl";
import { afterEach, describe, expect, it, vi } from "vitest";
import desktop from "../../../messages/en/desktop.json";
import unifiedModImport from "../../../messages/en/unifiedModImport.json";

const fetchMock = vi.fn();
vi.stubGlobal("fetch", fetchMock);

import { ImportProgressPanel } from "./ImportProgressPanel";

const answer = (body: unknown) => ({
  ok: true,
  status: 200,
  json: async () => body,
});

afterEach(() => {
  cleanup();
  fetchMock.mockReset();
});

describe("the import progress while the game index is built", () => {
  it("says the import waits for the reference lists from the game, and how far they are", async () => {
    fetchMock.mockImplementation((url: string) => {
      if (url === "/api/app/game-index") {
        return Promise.resolve(
          answer({
            state: "building",
            game_version: "1.19.3",
            step: "country_aliases",
            done: 12,
            total: 39,
            error: "",
            built_at: "",
          }),
        );
      }
      return Promise.resolve(
        answer({
          status: "running",
          phase: "game_index",
          files_done: 0,
          files_total: 0,
          percent: 55,
        }),
      );
    });
    render(
      <SWRConfig value={{ provider: () => new Map(), dedupingInterval: 0 }}>
        <IntlProvider
          locale="en"
          messages={{ desktop, unifiedModImport }}
          timeZone="UTC"
        >
          <ImportProgressPanel
            projectId={7}
            sessionId="s1"
            localPhase="parsing"
          />
        </IntlProvider>
      </SWRConfig>,
    );
    expect(
      await screen.findByText(
        "Waiting for the reference lists from your game…",
        { exact: false },
      ),
    ).toBeTruthy();
    expect(await screen.findByText("12/39")).toBeTruthy();
    expect(fetchMock).toHaveBeenCalledWith(
      "/api/go/import/jobs/7/s1",
      expect.anything(),
    );
  });
});
