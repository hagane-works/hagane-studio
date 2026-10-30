import { describe, expect, it, vi } from "vitest";
import { ApiError } from "@/lib/api/_client";
import {
  runExportValidationRequests,
  validationIssuesFromRequestError,
} from "@/lib/exportValidationRun";

describe("export validation request isolation", () => {
  it("keeps validate findings and skips a preview that would only repeat the blocker 409", async () => {
    const preview = vi.fn(async () => ({ files: ["must-not-run"] }));
    const result = await runExportValidationRequests(
      async () => ({
        issues: [
          {
            level: "error",
            code: "focus.reward.unsupported_scope",
            message: "wrong scope",
          },
          {
            level: "error",
            code: "focus_effective_default_multiple",
            message: "two defaults",
          },
        ],
        summary: { focus_trees: 1 },
      }),
      preview,
      (error) => `generic ${String(error)}`,
    );

    expect(preview).not.toHaveBeenCalled();
    expect(result.issues.map((issue) => issue.code)).toEqual([
      "focus.reward.unsupported_scope",
      "focus_effective_default_multiple",
    ]);
    expect(result.issues.map((issue) => issue.message)).not.toContain(
      "Request failed: 409",
    );
    expect(result.files).toEqual([]);
    expect(result.structureStatus).toBe("blocked");
  });

  it("recovers structured blockers if preview rejects after validate succeeds", async () => {
    const payload = {
      error: "ctd_blockers",
      message: "Fix the blockers before exporting.",
      blockers: [
        {
          code: "focus.reward.unsupported_scope",
          severity: "blocker",
          message: "Use a supported scope.",
          entity_ref: { type: "focus", id: "focus_a" },
          repair_route: "focus.reward.wrap_supported_scope",
        },
        {
          code: "focus.reward.unsupported_scope",
          severity: "blocker",
          message: "Use a supported scope.",
          entity_ref: { type: "focus", id: "focus_b" },
          repair_route: "focus.reward.wrap_supported_scope",
        },
      ],
    };
    const result = await runExportValidationRequests(
      async () => ({ issues: [], summary: { focus_trees: 1 } }),
      async () => {
        throw new ApiError(409, payload.message, payload);
      },
      () => "generic 409",
    );

    expect(result.issues).toHaveLength(2);
    expect(result.issues.map((issue) => issue.entity_ref?.id)).toEqual([
      "focus_a",
      "focus_b",
    ]);
    expect(result.issues[0]).toEqual(
      expect.objectContaining({
        level: "error",
        code: "focus.reward.unsupported_scope",
        message: "Use a supported scope.",
        repair_route: "focus.reward.wrap_supported_scope",
      }),
    );
    expect(result.structureStatus).toBe("failed");
    expect(result.structureError).toBeInstanceOf(ApiError);
  });

  it("falls back only when the response has no structured diagnostic", () => {
    expect(
      validationIssuesFromRequestError(
        new Error("network"),
        "friendly fallback",
      ),
    ).toEqual([{ level: "error", message: "friendly fallback" }]);
  });
});
