import { describe, it, expect } from "vitest";
import {
  parseNamedBlocks,
  serializeNamedBlocks,
  tokenBlockToText,
  textToTokenBlock,
  sectionToText,
} from "./doctrineBlocks";

describe("parseNamedBlocks", () => {
  it("returns [] for empty / whitespace input", () => {
    expect(parseNamedBlocks("")).toEqual([]);
    expect(parseNamedBlocks("   \n  ")).toEqual([]);
  });

  it("splits top-level named entries (rewards shape)", () => {
    const text =
      "interception_priority = {\n\tair_intercept_efficiency = 0.15\n}\n" +
      "air_defense_grids = {\n\tair_interception_detect_factor = 0.2\n}";
    const out = parseNamedBlocks(text);
    expect(out.map((e) => e.name)).toEqual([
      "interception_priority",
      "air_defense_grids",
    ]);
    expect(out[0].body).toContain("air_intercept_efficiency = 0.15");
    expect(out[1].body).toContain("air_interception_detect_factor = 0.2");
  });

  it("keeps nested blocks with their parent entry (depth-aware)", () => {
    const text =
      "milestone_1 = {\n\tname = ms1\n\tresearch_bonus = {\n\t\tinfantry = 0.5\n\t}\n}";
    const out = parseNamedBlocks(text);
    expect(out).toHaveLength(1);
    expect(out[0].name).toBe("milestone_1");
    expect(out[0].body).toContain("research_bonus = {");
    expect(out[0].body).toContain("infantry = 0.5");
  });

  it("never drops a named entry on round-trip (no silent truncation)", () => {
    const text =
      "milestone_1 = {\n\tarmy_org_factor = 0.02\n}\n" +
      "milestone_2 = {\n\tarmy_org_factor = 0.04\n}\n" +
      "milestone_3 = {\n\tarmy_org_factor = 0.06\n}";
    const out = parseNamedBlocks(text);
    expect(out).toHaveLength(3);
    const re = parseNamedBlocks(serializeNamedBlocks(out));
    expect(re.map((e) => e.name)).toEqual(out.map((e) => e.name));
    expect(re).toHaveLength(3);
  });

  it("surfaces leading non-entry text as __prologue__ (lossless)", () => {
    const text = "multiplier = 8.0\nmilestone_1 = {\n\tx = 1\n}";
    const out = parseNamedBlocks(text);
    expect(out[0].name).toBe("__prologue__");
    expect(out[0].body).toContain("multiplier = 8.0");
    expect(out[1].name).toBe("milestone_1");
  });
});

describe("serializeNamedBlocks", () => {
  it("wraps entries as name = { body }", () => {
    const out = serializeNamedBlocks([
      {
        name: "interception_priority",
        body: "air_intercept_efficiency = 0.15",
      },
    ]);
    expect(out).toContain("interception_priority = {");
    expect(out).toContain("air_intercept_efficiency = 0.15");
    expect(out.trim().endsWith("}")).toBe(true);
  });

  it("emits __prologue__ bare (no wrapping)", () => {
    const out = serializeNamedBlocks([
      { name: "__prologue__", body: "multiplier = 8.0" },
      { name: "milestone_1", body: "x = 1" },
    ]);
    expect(out.startsWith("multiplier = 8.0")).toBe(true);
    expect(out).toContain("milestone_1 = {");
  });

  it("skips fully-empty entries", () => {
    expect(serializeNamedBlocks([{ name: "", body: "" }])).toBe("");
  });
});

describe("token block helpers", () => {
  it("tokenBlockToText handles string / array / null", () => {
    expect(tokenBlockToText("infantry combat_support armor")).toBe(
      "infantry combat_support armor",
    );
    expect(tokenBlockToText("infantry\ncombat_support")).toBe(
      "infantry combat_support",
    );
    expect(tokenBlockToText(["a", "b"])).toBe("a b");
    expect(tokenBlockToText(null)).toBe("");
  });

  it("textToTokenBlock normalizes to newline-joined tokens", () => {
    expect(textToTokenBlock("infantry, combat_support  armor")).toBe(
      "infantry\ncombat_support\narmor",
    );
    expect(textToTokenBlock("   ")).toBe("");
  });
});

describe("sectionToText", () => {
  it("joins list-shaped raw_extras values with newline", () => {
    expect(sectionToText(["a = { x }", "b = { y }"])).toBe(
      "a = { x }\nb = { y }",
    );
    expect(sectionToText("plain")).toBe("plain");
    expect(sectionToText(null)).toBe("");
  });
});
