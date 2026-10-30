import { describe, it, expect } from "vitest";
import { omitKey, textToParents } from "./MIOTraitsEditor";

describe("textToParents", () => {
  it("splits on commas and whitespace, trimming + dropping blanks", () => {
    expect(textToParents("a, b ,c")).toEqual(["a", "b", "c"]);
    expect(textToParents("a   b\tc")).toEqual(["a", "b", "c"]);
    expect(textToParents("  a ,,  b  ,")).toEqual(["a", "b"]);
  });

  it("returns an empty array for empty / whitespace-only input", () => {
    expect(textToParents("")).toEqual([]);
    expect(textToParents("   ")).toEqual([]);
    expect(textToParents(" , , ")).toEqual([]);
  });
});

describe("omitKey", () => {
  it("removes the key immutably and preserves the rest", () => {
    const src = { a: 1, b: 2, c: 3 };
    const out = omitKey(src, "b");
    expect(out).toEqual({ a: 1, c: 3 });

    expect(src).toEqual({ a: 1, b: 2, c: 3 });
  });

  it("returns the same reference when the key is absent (no-op)", () => {
    const src = { a: 1 };
    expect(omitKey(src, "missing")).toBe(src);
  });

  it("preserves unknown / not-surfaced keys (don't-drop-fields contract)", () => {
    const trait = {
      kind: "add",
      token: "t",
      relative_position_id: "x",
      any_parent: ["p"],
      position: { x: 0, y: 1 },
    };

    expect(omitKey(trait, "position")).toEqual({
      kind: "add",
      token: "t",
      relative_position_id: "x",
      any_parent: ["p"],
    });
  });
});
