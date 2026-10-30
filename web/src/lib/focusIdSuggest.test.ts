import { readFileSync } from "node:fs";
import path from "node:path";
import { describe, expect, it } from "vitest";

import { detectFocusPrefix, suggestNewFocusId } from "./focusIdSuggest";

describe("suggestNewFocusId", () => {
  const treeA = [{ focus_id: "ROM_a" }, { focus_id: "ROM_b" }];
  const treeB = [{ focus_id: "ROM_new_focus" }, { focus_id: "ROM_New_Focus" }];

  it("only-current-tree input reproduces the collision (documents the trap)", () => {
    expect(suggestNewFocusId("ROM", treeA)).toBe("ROM_new_focus");
  });

  it("project-wide input skips the id that lives in the other tree", () => {
    expect(suggestNewFocusId("ROM", [...treeA, ...treeB])).toBe(
      "ROM_new_focus_2",
    );
  });

  it("walks _2, _3 … until free; empty prefix → empty", () => {
    const taken = [{ focus_id: "X_new_focus" }, { focus_id: "X_new_focus_2" }];
    expect(suggestNewFocusId("X", taken)).toBe("X_new_focus_3");
    expect(suggestNewFocusId("", taken)).toBe("");
  });
});

describe("detectFocusPrefix", () => {
  it("majority 2-3 letter ALL-CAPS prefix wins; none → empty", () => {
    expect(
      detectFocusPrefix([
        { focus_id: "ROM_a" },
        { focus_id: "ROM_b" },
        { focus_id: "GER_x" },
      ]),
    ).toBe("ROM");
    expect(detectFocusPrefix([{ focus_id: "lowercase_only" }])).toBe("");
    expect(detectFocusPrefix([])).toBe("");
  });
});

describe("focus-tree page wiring", () => {
  const src = readFileSync(
    path.resolve(__dirname, "../app/[locale]/project/[id]/focus-tree/page.tsx"),
    "utf-8",
  );

  it("suggests against project-wide nodes, never the current tree only", () => {
    expect(src).toMatch(
      /suggestNewFocusId\(detectedPrefix,\s*projectFocusNodes\)/,
    );
    expect(src).not.toMatch(/suggestNewFocusId\(detectedPrefix,\s*nodes\)/);
  });

  it("machine-suggested ids are submitted with on_conflict=suffix", () => {
    expect(src).toMatch(/on_conflict:\s*"suffix"/);
  });

  it("every create path goes through withSuggestedFocusId (4 call sites)", () => {
    const calls = src.match(/withSuggestedFocusId\(/g) ?? [];

    expect(calls.length).toBe(4);
  });
});
