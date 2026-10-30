import { describe, expect, it } from "vitest";
import {
  mergeRunCheckResults,
  type CtdCheckResult,
  type CtdIssue,
} from "./export";

const baseIssue: CtdIssue = {
  code: "focus_prereq_not_found",
  entity_ref: { type: "focus", id: 11, tree_id: 2 },
  severity: "blocker",
  message: "internal",
  fix_hint: "internal",
};

const base: CtdCheckResult = {
  ok: false,
  blocker_count: 1,
  warning_count: 0,
  blockers: [baseIssue],
  warnings: [],
  issues: [baseIssue],
};

describe("mergeRunCheckResults", () => {
  it("deduplicates Go parity findings while preserving the compatibility gate severity", () => {
    const merged = mergeRunCheckResults(base, [
      { ...baseIssue, severity: "warning" },
    ]);
    expect(merged.issues).toHaveLength(1);
    expect(merged.blockers).toEqual([baseIssue]);
    expect(merged.ok).toBe(false);
  });

  it("adds supplemental Go diagnostics as advisory findings", () => {
    const eventIssue: CtdIssue = {
      code: "event_namespace_mismatch",
      entity_ref: { type: "event", id: 7, namespace_id: 3 },
      severity: "warning",
      message: "",
      fix_hint: "",
    };
    const merged = mergeRunCheckResults(base, [eventIssue]);
    expect(merged.blocker_count).toBe(1);
    expect(merged.warning_count).toBe(1);
    expect(merged.warnings).toEqual([eventIssue]);
  });

  it("keeps distinct missing references on the same focus", () => {
    const clean: CtdCheckResult = {
      ok: true,
      blocker_count: 0,
      warning_count: 0,
      blockers: [],
      warnings: [],
      issues: [],
    };
    const missing = (reference: string): CtdIssue => ({
      code: "focus_prereq_not_found",
      entity_ref: { type: "focus", id: 12, tree_id: 2 },
      severity: "warning",
      message: "",
      fix_hint: "",
      detail: { prereq: reference, reference },
    });

    const merged = mergeRunCheckResults(clean, [missing("A"), missing("B")]);

    expect(merged.warnings).toHaveLength(2);
    expect(merged.warnings.map((item) => item.detail?.reference)).toEqual([
      "A",
      "B",
    ]);
  });
});
