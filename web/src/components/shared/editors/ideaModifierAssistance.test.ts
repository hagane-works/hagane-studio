import { describe, expect, it } from "vitest";

import {
  isUnknownModifierKey,
  knownModifierOptions,
  modifierEntriesToRecord,
  modifierValueFeedback,
  updateModifierEntry,
} from "./ideaModifierAssistance";

describe("idea modifier assistance", () => {
  it("offers deduplicated known modifier keys for searchable suggestions", () => {
    expect(
      knownModifierOptions([
        "war_support_factor",
        "consumer_goods_factor",
        "consumer_goods_factor",
      ]),
    ).toEqual(["consumer_goods_factor", "war_support_factor"]);
  });

  it("preserves an unknown custom key in the existing modifier record shape", () => {
    expect(modifierEntriesToRecord([[" my_custom_modifier ", "1.5"]])).toEqual({
      my_custom_modifier: 1.5,
    });
    expect(
      isUnknownModifierKey(
        "my_custom_modifier",
        new Set(["consumer_goods_factor"]),
      ),
    ).toBe(true);
  });

  it("updates only the requested key or value", () => {
    const entries: [string, string][] = [
      ["consumer_goods_factor", "-0.1"],
      ["war_support_factor", "0.2"],
    ];

    expect(updateModifierEntry(entries, 1, 0, "stability_factor")).toEqual([
      ["consumer_goods_factor", "-0.1"],
      ["stability_factor", "0.2"],
    ]);
    expect(updateModifierEntry(entries, 0, 1, "-0.2")).toEqual([
      ["consumer_goods_factor", "-0.2"],
      ["war_support_factor", "0.2"],
    ]);
  });

  it("explains text and parseFloat truncation without changing save semantics", () => {
    expect(modifierValueFeedback("@my_variable")).toEqual({ kind: "text" });
    expect(modifierValueFeedback("1.5oops")).toEqual({
      kind: "truncated",
      savedValue: 1.5,
    });
    expect(
      modifierEntriesToRecord([["consumer_goods_factor", "1.5oops"]]),
    ).toEqual({ consumer_goods_factor: 1.5 });
  });
});
