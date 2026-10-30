import {
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
} from "@testing-library/react";
import { IntlProvider } from "use-intl";
import { afterEach, describe, expect, it, vi } from "vitest";

const { previewFiles } = vi.hoisted(() => ({
  previewFiles: vi.fn().mockResolvedValue({
    files: {
      script: { path: "common/characters/GER_GER.txt", content: "script" },
      loc_en: {
        path: "localisation/english/replace/GER_characters_l_english.yml",
        content: 'l_english:\n GER_author:0 "English source"',
      },
      loc_de: {
        path: "localisation/german/replace/GER_characters_l_german.yml",
        content: 'l_german:\n GER_author:0 "Deutscher Inhalt"',
      },
      gfx: {
        path: "interface/GER_characters.gfx",
        content: "spriteTypes = {}",
      },
    },
  }),
}));

vi.mock("@/lib/api", () => ({
  charactersApi: { previewFiles },
}));

vi.mock("@/lib/store/projectStore", () => ({
  useProjectStore: (selector: (state: unknown) => unknown) =>
    selector({ currentProject: { export_languages: ["en", "de"] } }),
}));

import { MultiFilePreviewTabs } from "./MultiFilePreviewTabs";

afterEach(() => {
  cleanup();
  previewFiles.mockClear();
});

describe("MultiFilePreviewTabs content-language tabs", () => {
  it("shows every project-enabled language and loads its matching preview file", async () => {
    render(
      <IntlProvider
        locale="de"
        messages={{
          multiFilePreviewTabs: {
            loading: "Wird geladen…",
            previewFailed: "Vorschau fehlgeschlagen: {error}",
            noContentYet: "Noch kein Inhalt.",
            selectCharacterHint: "Figur auswählen.",
          },
        }}
        timeZone="UTC"
      >
        <MultiFilePreviewTabs
          scriptEditor={<div>Editor</div>}
          projectId={7}
          characterId={11}
        />
      </IntlProvider>,
    );

    expect(screen.getByRole("button", { name: "Loc EN" })).toBeTruthy();
    fireEvent.click(screen.getByRole("button", { name: "Loc DE" }));

    await waitFor(() => {
      expect(previewFiles).toHaveBeenCalledWith(7, 11);
      expect(screen.getByText(/Deutscher Inhalt/)).toBeTruthy();
    });
    expect(screen.queryByRole("button", { name: "Loc ZH" })).toBeNull();
  });
});
