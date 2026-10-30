import { describe, expect, it } from "vitest";

import {
  buildWargoalPreview,
  freshWargoalDraft,
  toWargoalDraft,
  toWargoalWrite,
} from "./wargoalForm";

describe("wargoal form helpers", () => {
  it("chooses a non-colliding starter id", () => {
    expect(freshWargoalDraft(["my_wargoal", "my_wargoal_2"]).wargoal_key).toBe(
      "my_wargoal_3",
    );
  });

  it("normalizes a write without discarding script-valued scalars", () => {
    const draft = freshWargoalDraft([]);
    draft.wargoal_key = " custom_goal ";
    draft.safe_scalars = { threat: " @CUSTOM_THREAT ", expire: "" };
    draft.advanced_raw_script =
      " on_add = {\n  ROOT = { add_stability = 0.1 }\n } ";
    expect(toWargoalWrite(draft)).toMatchObject({
      wargoal_key: "custom_goal",
      safe_scalars: { threat: "@CUSTOM_THREAT" },
      advanced_raw_script: "on_add = {\n  ROOT = { add_stability = 0.1 }\n }",
    });
  });

  it("builds the exact top-level wrapper and separates advanced script", () => {
    const draft = freshWargoalDraft([]);
    draft.wargoal_key = "custom_goal";
    draft.war_name = "CUSTOM_WAR_NAME";
    draft.allowed = "always = yes";
    draft.safe_scalars = { threat: "0.5" };
    draft.advanced_raw_script = "custom_flag = yes";
    const preview = buildWargoalPreview(draft);
    expect(preview).toContain("wargoal_types = {");
    expect(preview).toContain("\tcustom_goal = {");
    expect(preview).toContain("\t\tallowed = {");
    expect(preview).toContain("\t\tcustom_flag = yes");
  });

  it("copies response maps instead of sharing mutable form state", () => {
    const row = {
      id: 7,
      project_id: 2,
      wargoal_key: "custom_goal",
      war_name: "",
      safe_scalars: { threat: "1" },
      allowed: "",
      available: "",
      take_states: "",
      puppet: "",
      liberate: "",
      advanced_raw_script: "",
      source_file: "common/wargoals/custom.txt",
      source: "imported",
      created_at: "2026-07-28T00:00:00Z",
      updated_at: "2026-07-28T00:00:00Z",
    };
    const draft = toWargoalDraft(row);
    draft.safe_scalars.threat = "2";
    expect(row.safe_scalars.threat).toBe("1");
  });
});
