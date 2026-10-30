import { describe, expect, it } from "vitest";

import { visibleCharacterName } from "./characterDisplayName";

describe("visibleCharacterName", () => {
  it("suppresses the historical blank-character placeholder", () => {
    expect(visibleCharacterName("New Character")).toBe("");
    expect(visibleCharacterName("  NEW CHARACTER  ")).toBe("");
  });

  it("keeps and trims authored character names", () => {
    expect(visibleCharacterName("  Maurice Keyaerts  ")).toBe(
      "Maurice Keyaerts",
    );
    expect(visibleCharacterName(null)).toBe("");
  });
});
