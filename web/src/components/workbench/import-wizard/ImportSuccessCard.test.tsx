import React from "react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, render, screen } from "@testing-library/react";
import { IntlProvider, useTranslations } from "use-intl";

import unifiedModImport from "../../../../messages/en/unifiedModImport.json";
import type { ScanAndParseResult } from "@/lib/api";
import {
  ImportSuccessCard,
  type ImportSuccessCardProps,
} from "./ImportSuccessCard";

afterEach(() => cleanup());

const keys = Object.assign((key: string) => key, {
  has: () => false,
}) as unknown as ImportSuccessCardProps["t"];

function Card({
  result,
  real,
}: {
  result: Record<string, unknown>;
  real: boolean;
}) {
  const wizard = useTranslations("unifiedModImport");
  return (
    <ImportSuccessCard
      phase="done"
      reImporting={false}
      result={result as unknown as ScanAndParseResult}
      t={real ? wizard : keys}
      tag="GER"
      countryTag="GER"
      router={{ push: vi.fn() } as unknown as ImportSuccessCardProps["router"]}
      projectId={7}
      setOpen={vi.fn()}
      imageStatus=""
      whatsNextDetailOpen
      setWhatsNextDetailOpen={vi.fn()}
      imageExtSwaps={0}
      projectCountries={["GER"]}
      handleAddCountryAndReImport={vi.fn()}
      scanStatus=""
    />
  );
}

function renderCard(result: Record<string, unknown>, real = false) {
  render(
    <IntlProvider
      locale="en"
      messages={{ unifiedModImport }}
      onError={() => {}}
      timeZone="UTC"
    >
      <Card result={result} real={real} />
    </IntlProvider>,
  );
}

const counts = { created: 0, updated: 0, skipped: 0, warnings: null };

function importResult(overrides: Record<string, unknown>) {
  return {
    characters: { ...counts, created: 2 },
    focus_trees: {
      created_trees: 0,
      updated_trees: 0,
      skipped_trees: 0,
      created_nodes: 0,
      warnings: null,
    },
    technologies: { ...counts },
    ideas: { ...counts },
    warnings: null,
    needed_images: null,
    gfx_mappings_count: 0,
    ...overrides,
  };
}

describe("import success card", () => {
  it("renders the summary when the server sends null image and warning lists", () => {
    renderCard(importResult({}));
    expect(screen.getByText(/itemCharacters/)).toBeTruthy();
    expect(screen.queryByText("backendRequestedImages")).toBeNull();
  });

  it("still reports the images the server asked for", () => {
    renderCard(
      importResult({ needed_images: ["gfx/leaders/GER/a.png"], warnings: [] }),
    );
    expect(screen.getByText("backendRequestedImages")).toBeTruthy();
  });

  it("words the program's own coded warnings in the player's language", () => {
    renderCard(
      importResult({
        warnings: [
          '§§not_loaded_by_game§§{"file": "common/national_focus/FOR HOTFIX/a.txt", "folder": "common/national_focus"}§§common/national_focus/FOR HOTFIX/a.txt was not imported: plain English',
        ],
      }),
      true,
    );
    expect(
      screen.getByText(
        "common/national_focus/FOR HOTFIX/a.txt was not imported — the game only reads the files directly in common/national_focus/, not the ones in its subfolders.",
      ),
    ).toBeTruthy();
  });
});
