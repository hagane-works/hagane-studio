import { readFileSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";
import { cleanup, render, screen } from "@testing-library/react";
import { IntlProvider } from "use-intl";
import { afterEach, describe, expect, it } from "vitest";
import exportMessages from "../../../messages/en/export.json";
import { ExportContentsCard, type ExportSummary } from "./ExportContentsCard";

const EMPTY: ExportSummary = {
  characters: 0,
  focus_trees: 0,
  focus_nodes: 0,
  technologies: 0,
  ideas: 0,
};

function card(
  pending: "idle" | "running" | null,
  summary: ExportSummary | null,
) {
  render(
    <IntlProvider
      locale="en"
      messages={{ export: exportMessages }}
      timeZone="UTC"
    >
      <ExportContentsCard
        pending={pending}
        summary={summary}
        summaryUnavailable={false}
        assetAudit={null}
        builtFileCount={null}
        builtSizeBytes={null}
        identityDescKey="identityStandaloneDesc"
        legacyMapNote={null}
      />
    </IntlProvider>,
  );
}

afterEach(cleanup);

describe("ExportContentsCard before a check", () => {
  it("asks for the check instead of saying there is nothing to export", () => {
    card("idle", null);
    expect(
      screen.getByText(exportMessages.ui.contents.notChecked),
    ).toBeTruthy();
    expect(screen.queryByText(exportMessages.ui.contents.empty)).toBeNull();
  });

  it("says the check is running during the first run", () => {
    card("running", null);
    expect(
      screen.getByText(exportMessages.ui.stepper.step1Running),
    ).toBeTruthy();
    expect(screen.queryByText(exportMessages.ui.contents.empty)).toBeNull();
  });

  it("still says there is nothing to export once a check found nothing", () => {
    card(null, EMPTY);
    expect(screen.getByText(exportMessages.ui.contents.empty)).toBeTruthy();
  });
});

describe("the Check details entry before a check", () => {
  const page = readFileSync(
    join(
      dirname(fileURLToPath(import.meta.url)),
      "../../app/[locale]/project/[id]/export/page.tsx",
    ),
    "utf8",
  );

  it("names each phase instead of showing Checking… whenever no result is done", () => {
    expect(page).toContain(
      'phase === "idle"\n    ? t("ui.stepper.step1SubIdle")',
    );
    expect(page).toContain(
      'phase === "failed"\n    ? t("ui.stepper.step1Failed")',
    );
    expect(page).not.toContain(
      'phase !== "done"\n    ? t("ui.stepper.step1Running")',
    );
  });
});
