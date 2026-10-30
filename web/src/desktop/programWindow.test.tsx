import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { IntlProvider } from "use-intl";
import { afterEach, beforeAll, describe, expect, it, vi } from "vitest";
import confirmDialog from "../../messages/en/confirmDialog.json";
import desktop from "../../messages/en/desktop.json";
import { ConfirmDialogProvider } from "@/components/shared/ConfirmDialogProvider";
import { useEditorDirtyStore } from "@/lib/store/editorDirtyStore";
import {
  ProgramWindowClose,
  hasUnsavedChanges,
  inProgramWindow,
  installProgramWindow,
} from "./programWindow";

const bridge = { open: vi.fn(), close: vi.fn() };
const gameView = {
  closed: false,
  opener: window,
  haganeWorksHasUnsaved: vi.fn(() => false),
};
const browserOpen = vi.fn(
  (..._args: unknown[]) => gameView as unknown as Window,
);

beforeAll(() => {
  window.haganeWorks = bridge;
  window.open = browserOpen;
  installProgramWindow();
});

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
  useEditorDirtyStore.getState().clearAll();
  document.body.replaceChildren();
});

function click(href: string, target = ""): boolean {
  const link = document.createElement("a");
  link.href = href;
  link.target = target;
  document.body.append(link);
  let prevented = false;
  const note = (event: Event) => {
    prevented = event.defaultPrevented;
    event.preventDefault();
  };
  document.addEventListener("click", note);
  fireEvent.click(link);
  document.removeEventListener("click", note);
  return prevented;
}

describe("links in the program's window", () => {
  it("sends other sites to the default browser and leaves the program's own pages alone", () => {
    expect(inProgramWindow()).toBe(true);
    expect(
      click("https://github.com/cwtools/cwtools-hoi4-config", "_blank"),
    ).toBe(true);
    expect(bridge.open).toHaveBeenCalledWith(
      "https://github.com/cwtools/cwtools-hoi4-config",
    );
    expect(window.open("https://example.com/help", "_blank")).toBeNull();
    expect(bridge.open).toHaveBeenLastCalledWith("https://example.com/help");
    expect(browserOpen).not.toHaveBeenCalled();

    expect(click("/en/project/1/start")).toBe(false);
    expect(bridge.open).toHaveBeenCalledTimes(2);
  });

  it("opens the program's own windows itself, and asks them about unsaved changes", () => {
    expect(click("/en/game-view?project_id=1", "hoi4-game-view-1")).toBe(true);
    expect(browserOpen).toHaveBeenCalledWith(
      `${location.origin}/en/game-view?project_id=1`,
      "hoi4-game-view-1",
      undefined,
    );
    expect(hasUnsavedChanges()).toBe(false);
    gameView.haganeWorksHasUnsaved.mockReturnValue(true);
    expect(hasUnsavedChanges()).toBe(true);
    gameView.haganeWorksHasUnsaved.mockReturnValue(false);
  });

  it("shows the browser's right-click menu only on text", () => {
    const input = document.createElement("input");
    const panel = document.createElement("div");
    document.body.append(input, panel);
    expect(fireEvent.contextMenu(input)).toBe(true);
    expect(fireEvent.contextMenu(panel)).toBe(false);
  });
});

describe("closing the program's window", () => {
  function renderClose() {
    render(
      <IntlProvider
        locale="en"
        messages={{ desktop, confirmDialog }}
        timeZone="UTC"
      >
        <ConfirmDialogProvider>
          <ProgramWindowClose />
        </ConfirmDialogProvider>
      </IntlProvider>,
    );
  }

  it("closes at once when nothing is unsaved", () => {
    renderClose();
    window.haganeWorksCloseRequested?.();
    expect(bridge.close.mock.calls).toEqual([["close"]]);
  });

  it("asks the player first while something is unsaved", async () => {
    renderClose();
    useEditorDirtyStore.getState().setDirty("characters", true);

    window.haganeWorksCloseRequested?.();
    expect(bridge.close).toHaveBeenLastCalledWith("asking");
    fireEvent.click(await screen.findByRole("button", { name: "Cancel" }));
    await vi.waitFor(() =>
      expect(bridge.close).toHaveBeenLastCalledWith("stay"),
    );

    window.haganeWorksCloseRequested?.();
    expect(await screen.findByText("Unsaved changes")).toBeTruthy();
    fireEvent.click(
      screen.getByRole("button", { name: "Close without saving" }),
    );
    await vi.waitFor(() =>
      expect(bridge.close).toHaveBeenLastCalledWith("close"),
    );
    expect(bridge.close.mock.calls.map(([answer]) => answer)).toEqual([
      "asking",
      "stay",
      "asking",
      "close",
    ]);
  });
});
