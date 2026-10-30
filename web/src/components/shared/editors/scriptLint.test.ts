import { describe, expect, it } from "vitest";

import type { CwtoolRules } from "@/lib/api";
import type { VanillaScriptNames } from "@/lib/vanillaScriptNames";
import { EMPTY_SCRIPT_LINT_RESULT, lintScriptBlock } from "./scriptLint";

const RULES: CwtoolRules = {
  version: 2,
  anti_patterns: [],
  trigger_enums: { has_war: ["yes", "no"] },
  field_enums: {},
  scope_keys_must_be_block: [],
  modifier_framework_keys: [],
  known_effects: [],
  known_triggers: ["has_war", "has_political_power"],
  known_modifiers: [],
  allowed_effects_in_modifier_block: [],
  modifier_scalar_parent_effects: [],
  option_scalar_parent_triggers: [],
};

const VANILLA: VanillaScriptNames = {
  triggers: new Set(["has_war", "has_political_power"]),
  effects: new Set(["add_political_power"]),
  structuralFields: new Set(),
  mathFunctions: new Set(),
  loaded: true,
};

describe("shared ScriptBlockEditor lint", () => {
  it("returns a stable empty result for empty text", () => {
    expect(lintScriptBlock("  \n", RULES, VANILLA, "en")).toBe(
      EMPTY_SCRIPT_LINT_RESULT,
    );
  });

  it("treats a missing legacy script field as an empty block", () => {
    expect(lintScriptBlock(undefined, RULES, VANILLA, "en")).toBe(
      EMPTY_SCRIPT_LINT_RESULT,
    );
    expect(lintScriptBlock(null, RULES, VANILLA, "en")).toBe(
      EMPTY_SCRIPT_LINT_RESULT,
    );
  });

  it("accepts a known trigger and scripted_gui structure without noise", () => {
    expect(
      lintScriptBlock("has_war = yes\nvisible = {\n}", RULES, VANILLA, "en"),
    ).toEqual({ markers: [], unknownTokens: [] });
  });

  it("reports a cwtool value problem and a likely token typo", () => {
    const result = lintScriptBlock(
      "has_war = factor\nhas_political_powr = 10",
      RULES,
      VANILLA,
      "en",
    );

    expect(result.markers).toEqual([
      expect.objectContaining({ line: 1, severity: "warning" }),
    ]);
    expect(result.unknownTokens).toEqual([
      expect.objectContaining({
        line: 2,
        token: "has_political_powr",
        suggestion: "has_political_power",
      }),
    ]);
  });
});
