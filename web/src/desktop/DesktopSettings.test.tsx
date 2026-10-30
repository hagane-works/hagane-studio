import {
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
} from "@testing-library/react";
import { SWRConfig } from "swr";
import { IntlProvider } from "use-intl";
import { afterEach, describe, expect, it, vi } from "vitest";
import confirmDialog from "../../messages/en/confirmDialog.json";
import desktop from "../../messages/en/desktop.json";
import { ConfirmDialogProvider } from "@/components/shared/ConfirmDialogProvider";
import { useEditorDirtyStore } from "@/lib/store/editorDirtyStore";
import { useAvatarStore } from "./avatar";

const fetchMock = vi.fn();
vi.stubGlobal("fetch", fetchMock);

import { DesktopSettingsButton } from "./DesktopSettings";

const answer = (body: unknown) => ({
  ok: true,
  status: 200,
  json: async () => body,
});
const status = (cwtoolsDir: string, cwtools: Record<string, unknown>) => ({
  version: "0.1.0",
  settings: {
    projects_dir: "C:\\Mods",
    game_dir: "C:\\HOI4",
    cwtools_dir: cwtoolsDir,
  },
  game: { dir: "C:\\HOI4", version: "1.19.3" },
  mod_dir: "C:\\Mods\\mod",
  cwtools: { dir: "", modifiers: 0, error: "", ...cwtools },
  port: 37330,
  configured_port: 37330,
  cpu: "100%",
  cpu_cores: 8,
  cores: 8,
  ready: true,
});
const INDEX = {
  state: "ready",
  game_version: "1.19.3",
  step: "",
  done: 34,
  total: 34,
  error: "",
  built_at: "",
};

function openSettings(first: unknown, afterSave?: unknown) {
  fetchMock.mockImplementation((url: string, init?: RequestInit) => {
    if (url === "/api/app/game-index") return Promise.resolve(answer(INDEX));
    if (url === "/api/app/settings" && init?.method === "PUT")
      return Promise.resolve(answer(afterSave));
    return Promise.resolve(answer(first));
  });
  render(
    <SWRConfig value={{ provider: () => new Map(), dedupingInterval: 0 }}>
      <IntlProvider
        locale="en"
        messages={{ desktop, confirmDialog }}
        timeZone="UTC"
      >
        <ConfirmDialogProvider>
          <DesktopSettingsButton />
        </ConfirmDialogProvider>
      </IntlProvider>
    </SWRConfig>,
  );
  fireEvent.click(screen.getByRole("button", { name: "Settings" }));
}

afterEach(() => {
  cleanup();
  fetchMock.mockReset();
  useEditorDirtyStore.getState().clearAll();
  useAvatarStore.getState().choosePet("field_engineer_dog");
});

describe("the local avatar in the settings", () => {
  it("switches the settings button to the chosen companion and remembers it locally", async () => {
    openSettings(status("", {}));
    const settingsButton = document.querySelector<HTMLButtonElement>(
      'button[aria-label="Settings"]',
    );
    expect(settingsButton).toBeTruthy();

    fireEvent.click(await screen.findByRole("button", { name: "Admiral Penguin" }));

    expect(settingsButton?.querySelector("img")?.getAttribute("src")).toBe(
      "/avatars/pets/admiral_penguin.png",
    );
    expect(localStorage.getItem("hagane-works-avatar-v1")).toContain(
      "admiral_penguin",
    );
  });

  it("rejects unsupported custom-image files", async () => {
    openSettings(status("", {}));
    const input = await screen.findByLabelText("Use your own image");

    fireEvent.change(input, {
      target: { files: [new File(["text"], "avatar.txt", { type: "text/plain" })] },
    });

    expect(
      await screen.findByText("Choose a PNG, JPEG or WebP image."),
    ).toBeTruthy();
  });
});

describe("the CWTools rules in the settings", () => {
  it("shows the folder in use, how many modifier names it gave and where the rules come from", async () => {
    openSettings(
      status("", { dir: "C:\\Tools\\cwtools-hoi4-config", modifiers: 4718 }),
    );
    expect(
      await screen.findByText("C:\\Tools\\cwtools-hoi4-config"),
    ).toBeTruthy();
    expect(screen.getByText("4,718 modifier names read")).toBeTruthy();
    expect(
      screen
        .getByRole("link", { name: "github.com/cwtools/cwtools-hoi4-config" })
        .getAttribute("href"),
    ).toBe("https://github.com/cwtools/cwtools-hoi4-config");

    expect(
      screen.queryByRole("button", { name: "Find automatically" }),
    ).toBeNull();
  });

  it("says why there are none, and goes back to finding them automatically", async () => {
    openSettings(
      status("D:\\old", { error: "D:\\old has no Config/modifiers.cwt" }),
      status("", { error: "no CWTools rules beside the program" }),
    );
    expect(await screen.findByText("Not found")).toBeTruthy();
    expect(
      screen.getByText("D:\\old has no Config/modifiers.cwt"),
    ).toBeTruthy();

    fireEvent.click(screen.getByRole("button", { name: "Find automatically" }));
    expect(
      await screen.findByText("no CWTools rules beside the program"),
    ).toBeTruthy();
    expect(fetchMock).toHaveBeenCalledWith(
      "/api/app/settings",
      expect.objectContaining({
        method: "PUT",
        body: JSON.stringify({ cwtools_dir: "" }),
      }),
    );
  });
});

describe("the port in the settings", () => {
  it("shows the port in use and saves another one for the next start", async () => {
    const before = status("", {});
    openSettings(before, {
      ...before,
      settings: { ...before.settings, port: 40000 },
      configured_port: 40000,
    });
    const field = (await screen.findByLabelText("Port")) as HTMLInputElement;
    const save = screen.getByRole("button", {
      name: "Save",
    }) as HTMLButtonElement;
    expect(field.value).toBe("37330");
    expect(screen.getByText("In use now: 37330")).toBeTruthy();
    expect(save.disabled).toBe(true);

    fireEvent.change(field, { target: { value: "80" } });
    expect(save.disabled).toBe(true);
    fireEvent.change(field, { target: { value: "40000" } });
    fireEvent.click(save);
    await waitFor(() => expect(save.disabled).toBe(true));
    expect(fetchMock).toHaveBeenCalledWith(
      "/api/app/settings",
      expect.objectContaining({
        method: "PUT",
        body: JSON.stringify({ port: 40000 }),
      }),
    );

    expect(field.value).toBe("40000");
    expect(screen.getByText("In use now: 37330")).toBeTruthy();
  });
});

describe("quitting from the settings", () => {
  it("asks first while something is unsaved, and quits only when the player agrees", async () => {
    const close = vi.spyOn(window, "close").mockImplementation(() => undefined);
    openSettings(status("", {}));
    useEditorDirtyStore.getState().setDirty("characters", true);
    const quit = await screen.findByRole("button", {
      name: "Quit Hagane Works",
    });

    fireEvent.click(quit);
    fireEvent.click(await screen.findByRole("button", { name: "Cancel" }));
    await waitFor(() =>
      expect(screen.queryByText("Unsaved changes")).toBeNull(),
    );
    expect(fetchMock).not.toHaveBeenCalledWith(
      "/api/app/quit",
      expect.anything(),
    );

    fireEvent.click(quit);
    expect(
      await screen.findByText(
        "Some changes are not saved yet. Close Hagane Works anyway? They will be lost.",
      ),
    ).toBeTruthy();
    fireEvent.click(
      screen.getByRole("button", { name: "Close without saving" }),
    );
    await waitFor(() => expect(close).toHaveBeenCalled());
    expect(fetchMock).toHaveBeenCalledWith(
      "/api/app/quit",
      expect.objectContaining({ method: "POST" }),
    );
    close.mockRestore();
  });
});

describe("the CPU use in the settings", () => {
  it("offers one core, half and all of them with their core counts, and saves the choice", async () => {
    openSettings(
      { ...status("", {}), cpu: "50%", cpu_cores: 4 },
      {
        ...status("", {}),
        settings: { ...status("", {}).settings, cpu: "1" },
        cpu: "1",
        cpu_cores: 1,
      },
    );
    const select = await screen.findByRole("combobox", { name: "CPU use" });
    expect(select.textContent).toContain("50% (4 cores)");
    expect(screen.getByText(/How many processor cores/)).toBeTruthy();

    fireEvent.click(select);
    expect(
      await screen.findByRole("option", { name: "100% (8 cores)" }),
    ).toBeTruthy();

    const one = screen.getByRole("option", { name: "1 core" });
    fireEvent.mouseMove(one);
    fireEvent.click(one);
    await waitFor(() =>
      expect(fetchMock).toHaveBeenCalledWith(
        "/api/app/settings",
        expect.objectContaining({
          method: "PUT",
          body: JSON.stringify({ cpu: "1" }),
        }),
      ),
    );
    await waitFor(() =>
      expect(
        screen.getByRole("combobox", { name: "CPU use" }).textContent,
      ).toContain("1 core"),
    );
  });
});
