import { describe, expect, it } from "vitest";

import { ApiError } from "@/lib/api/_client";
import {
  characterTagOwners,
  characterTagTaken,
  isCharacterTagConflict,
} from "./characterTagTaken";

describe("characterTagTaken", () => {
  const owners = characterTagOwners([
    { id: 1, tag: "GER_rommel" },
    { id: 2, tag: " GER_guderian " },
    { id: 3, tag: "" },
  ]);

  it("flags a TAG another character of the project holds", () => {
    expect(characterTagTaken(owners, "GER_rommel", 2)).toBe(true);
    expect(characterTagTaken(owners, " GER_guderian", 1)).toBe(true);
  });

  it("lets a character keep its own TAG and use a free one", () => {
    expect(characterTagTaken(owners, "GER_rommel", 1)).toBe(false);
    expect(characterTagTaken(owners, "GER_manstein", 1)).toBe(false);
    expect(characterTagTaken(owners, "", 1)).toBe(false);
  });

  it("compares exactly, as the database does", () => {
    expect(characterTagTaken(owners, "ger_rommel", 2)).toBe(false);
  });

  it("recognises the server's duplicate-TAG answer", () => {
    expect(
      isCharacterTagConflict(
        new ApiError(
          409,
          "tag 'x' already exists in this project. Pick a unique tag.",
        ),
      ),
    ).toBe(true);
    expect(
      isCharacterTagConflict(new ApiError(503, "character store unavailable")),
    ).toBe(false);
    expect(isCharacterTagConflict(new Error("network"))).toBe(false);
  });
});
