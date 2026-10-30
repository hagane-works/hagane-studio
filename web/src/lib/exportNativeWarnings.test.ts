import { describe, expect, it } from "vitest";
import {
  NATIVE_WARNING_CODE,
  nativeWarningIssues,
} from "@/lib/exportNativeWarnings";
import { presentCheckIssues } from "@/lib/exportCheckPresentation";

describe("nativeWarningIssues", () => {
  it("maps each non-empty note to one info finding, keeping the text", () => {
    const issues = nativeWarningIssues(
      ["a dropped picture", "  ", "", "loc back-filled from English (3 keys)"],
      "While building the ZIP",
    );
    expect(issues).toHaveLength(2);
    expect(issues.map((i) => i.code)).toEqual([
      NATIVE_WARNING_CODE,
      NATIVE_WARNING_CODE,
    ]);
    expect(issues.every((i) => i.severity === "info")).toBe(true);
    expect(issues[0].message).toBe("While building the ZIP: a dropped picture");
    expect(issues[1].detail).toEqual({
      native_warning: "loc back-filled from English (3 keys)",
    });
  });

  it("returns [] for undefined / null / empty input", () => {
    expect(nativeWarningIssues(undefined)).toEqual([]);
    expect(nativeWarningIssues(null)).toEqual([]);
    expect(nativeWarningIssues([])).toEqual([]);
  });

  it("two notes ⇒ two auto rows in the four-tier presentation, zero attention rows", () => {
    const presentation = presentCheckIssues(
      nativeWarningIssues(["note one", "note two"], "L"),
    );
    expect(presentation.auto).toHaveLength(2);
    expect(presentation.attention).toHaveLength(0);
    expect(presentation.attentionGroups).toHaveLength(0);
    expect(presentation.health).toHaveLength(0);
    expect(presentation.auto.map((f) => f.issue.message)).toEqual([
      "L: note one",
      "L: note two",
    ]);
  });
});
