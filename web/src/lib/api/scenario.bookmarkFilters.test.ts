import { describe, expect, it } from "vitest";
import {
  BOOKMARK_FILTER_CHOICES_1_19,
  updateBookmarkFilterChoice119,
} from "./scenario";

describe("HOI4 1.19 bookmark filter choices", () => {
  it("offers only the versioned choices for new writes", () => {
    expect(BOOKMARK_FILTER_CHOICES_1_19).toEqual([
      "new_content",
      "continent",
      "label",
    ]);
  });

  it("toggles known choices while preserving legacy tokens exactly", () => {
    const imported = ["canada", "continent", "europe", "asia"];
    expect(updateBookmarkFilterChoice119(imported, "label", true)).toEqual([
      "continent",
      "label",
      "canada",
      "europe",
      "asia",
    ]);
    expect(updateBookmarkFilterChoice119(imported, "continent", false)).toEqual(
      ["canada", "europe", "asia"],
    );
  });
});
