import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import type { CtdIssue } from "@/lib/api/export";
import { ApiError } from "@/lib/api/_client";
import {
  diagnosticBlockersFromExportError,
  ExportRiskConfirmDialog,
} from "./ExportRiskConfirmDialog";

const copy: Record<string, string> = {
  riskConfirmTitle: "Review before downloading",
  riskConfirmBody: "There are {count} risks in this mod.",
  riskConfirmImpact: "The ZIP may fail to load or have missing content.",
  riskConfirmCancel: "Go back and fix",
  riskConfirmProceed: "I understand the risk, export anyway",
  riskConfirmReminder:
    "This applies only to this download. We will check again next time.",
  kind_idea: "idea",
  kind_event: "event",
  kind_other: "content",
  summary_idea_no_modifier: "{count} ideas do not change anything in game",
  groupIssueCount: "{count} {kind} problems need attention",
  genericIssueMessage: "This {kind} may not work as expected",
  genericIssueHint: "Open this {kind} and complete the missing content",
  impactBlocker: "The mod may fail to load or this content may be missing.",
  impactWarning: "This content may behave differently in game.",
  impactInfo: "This does not stop the mod from loading.",
};

vi.mock("use-intl", () => ({
  useTranslations:
    () => (key: string, values?: Record<string, string | number>) => {
      let value = copy[key] ?? key;
      for (const [name, replacement] of Object.entries(values ?? {})) {
        value = value.replaceAll(`{${name}}`, String(replacement));
      }
      return value;
    },
}));

const ideaBlocker = (id: number): CtdIssue => ({
  code: "idea_no_modifier",
  severity: "blocker",
  entity_ref: { type: "idea", id },
  message: `RAW_BACKEND_IDEA_${id}`,
  fix_hint: "INTERNAL_FIX_HINT",
});

describe("ExportRiskConfirmDialog", () => {
  afterEach(() => {
    cleanup();
  });

  it("groups player-facing problems and never exposes internal diagnostics", () => {
    render(
      <ExportRiskConfirmDialog
        open
        blockerCount={3}
        blockers={[
          ideaBlocker(1),
          ideaBlocker(2),
          {
            code: "future_internal_rule",
            severity: "blocker",
            entity_ref: { type: "event", id: 8 },
            message: "RAW_BACKEND_EVENT",
            fix_hint: "SECRET_ROUTE",
          },
        ]}
        onCancel={vi.fn()}
        onConfirm={vi.fn()}
      />,
    );

    expect(screen.getByText("There are 3 risks in this mod.")).toBeTruthy();
    expect(
      screen.getByText("2 ideas do not change anything in game"),
    ).toBeTruthy();
    expect(
      screen.getByText("This event may not work as expected"),
    ).toBeTruthy();
    expect(document.body.textContent).not.toContain("idea_no_modifier");
    expect(document.body.textContent).not.toContain("future_internal_rule");
    expect(document.body.textContent).not.toContain("RAW_BACKEND");
    expect(document.body.textContent).not.toContain("INTERNAL_FIX_HINT");
  });

  it("has separate cancel and explicit risk-acceptance actions", () => {
    const onCancel = vi.fn();
    const onConfirm = vi.fn();
    render(
      <ExportRiskConfirmDialog
        open
        blockerCount={1}
        blockers={[ideaBlocker(1)]}
        onCancel={onCancel}
        onConfirm={onConfirm}
      />,
    );

    fireEvent.click(screen.getByRole("button", { name: "Go back and fix" }));
    expect(onCancel).toHaveBeenCalledTimes(1);
    expect(onConfirm).not.toHaveBeenCalled();

    fireEvent.click(
      screen.getByRole("button", {
        name: "I understand the risk, export anyway",
      }),
    );
    expect(onConfirm).toHaveBeenCalledTimes(1);
  });

  it("recovers final-artifact 409 blockers without retaining raw backend prose", () => {
    const blockers = diagnosticBlockersFromExportError(
      new ApiError(409, "RAW TOP LEVEL", {
        message: "RAW TOP LEVEL",
        blockers: [
          {
            code: "focus.reward.unsupported_scope",
            severity: "blocker",
            entity_ref: { type: "focus", id: "focus_a" },
            message: "RAW BLOCKER MESSAGE",
            fix_hint: "RAW FIX HINT",
            detail: { field: "completion_reward" },
          },
        ],
      }),
    );

    expect(blockers).toHaveLength(1);
    expect(blockers[0]).toMatchObject({
      code: "focus.reward.unsupported_scope",
      severity: "blocker",
      entity_ref: { type: "focus", id: "focus_a" },
      message: "",
      fix_hint: "",
      detail: { field: "completion_reward" },
    });
    expect(
      diagnosticBlockersFromExportError(
        new ApiError(500, "server failed", { blockers: [ideaBlocker(1)] }),
      ),
    ).toEqual([]);
  });
});
