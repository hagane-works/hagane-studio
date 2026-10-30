import { cleanup, render, screen } from "@testing-library/react";
import { SWRConfig } from "swr";
import { IntlProvider } from "use-intl";
import { afterEach, describe, expect, it, vi } from "vitest";
import desktop from "../../messages/en/desktop.json";
import { REFERENCE_TABLES } from "@/lib/__fixtures__/referenceTables";
import { textColors } from "@/lib/referenceTables";

const fetchMock = vi.fn();
vi.stubGlobal("fetch", fetchMock);

import { SetupGate } from "./SetupGate";

const answer = (body: unknown) => ({
  ok: true,
  status: 200,
  json: async () => body,
});
const STATUS = {
  version: "0.1.0",
  settings: { projects_dir: "C:\\Mods", game_dir: "C:\\HOI4" },
  game: { dir: "C:\\HOI4", version: "1.19.3" },
  mod_dir: "C:\\Mods\\mod",
  cwtools: { dir: "", modifiers: 0, error: "" },
  ready: true,
};

function renderGate() {
  return render(
    <SWRConfig
      value={{
        provider: () => new Map(),
        dedupingInterval: 0,
        shouldRetryOnError: false,
      }}
    >
      <IntlProvider locale="en" messages={{ desktop }} timeZone="UTC">
        <SetupGate>
          <p>the app</p>
        </SetupGate>
      </IntlProvider>
    </SWRConfig>,
  );
}

afterEach(() => {
  cleanup();
  fetchMock.mockReset();
});

describe("the setup gate", () => {
  it("renders the app once both the status and the reference tables are in", async () => {
    let tables: (value: unknown) => void = () => {};
    fetchMock.mockImplementation((url: string) =>
      url === "/api/app/reference-tables"
        ? new Promise((resolve) => (tables = resolve))
        : Promise.resolve(answer(STATUS)),
    );
    renderGate();
    await vi.waitFor(() =>
      expect(fetchMock).toHaveBeenCalledWith(
        "/api/app/status",
        expect.anything(),
      ),
    );
    expect(screen.queryByText("the app")).toBeNull();

    tables(answer(REFERENCE_TABLES));
    expect(await screen.findByText("the app")).toBeTruthy();
    expect(textColors().G).toBe("rgb(0, 159, 3)");
  });

  it("shows why when the reference tables cannot be read", async () => {
    fetchMock.mockImplementation((url: string) =>
      Promise.resolve(
        url === "/api/app/reference-tables"
          ? {
              ok: false,
              status: 500,
              statusText: "Internal Server Error",
              json: async () => ({ detail: "tables unreadable" }),
            }
          : answer(STATUS),
      ),
    );
    renderGate();
    expect(await screen.findByText("tables unreadable")).toBeTruthy();
    expect(screen.queryByText("the app")).toBeNull();
  });
});
