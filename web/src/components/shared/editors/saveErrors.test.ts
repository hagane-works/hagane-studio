import { describe, expect, it } from "vitest";

import {
  GENERIC_SAVE_ERROR,
  extractSaveErrorMessage,
  validateRequiredId,
} from "./saveErrors";

describe("extractSaveErrorMessage", () => {
  it("pulls the backend detail off an Error (the api.ts common path)", () => {
    const err = new Error("Event id 'foo.1' already exists. Pick a unique id.");
    expect(extractSaveErrorMessage(err)).toBe(
      "Event id 'foo.1' already exists. Pick a unique id.",
    );
  });

  it("handles a raw string error", () => {
    expect(extractSaveErrorMessage("validation failed: name required")).toBe(
      "validation failed: name required",
    );
  });

  it("handles an axios-shaped object (response.data.detail)", () => {
    const err = { response: { data: { detail: "duplicate id" } } };
    expect(extractSaveErrorMessage(err)).toBe("duplicate id");
  });

  it("handles a plain object carrying detail / message", () => {
    expect(extractSaveErrorMessage({ detail: "boom" })).toBe("boom");
    expect(extractSaveErrorMessage({ message: "kaboom" })).toBe("kaboom");
  });

  it("falls back to the generic message when nothing usable is present", () => {
    expect(extractSaveErrorMessage(null)).toBe(GENERIC_SAVE_ERROR);
    expect(extractSaveErrorMessage(undefined)).toBe(GENERIC_SAVE_ERROR);
    expect(extractSaveErrorMessage(new Error("   "))).toBe(GENERIC_SAVE_ERROR);
    expect(extractSaveErrorMessage("")).toBe(GENERIC_SAVE_ERROR);
    expect(extractSaveErrorMessage({})).toBe(GENERIC_SAVE_ERROR);
    expect(extractSaveErrorMessage(42)).toBe(GENERIC_SAVE_ERROR);
  });

  it("honours a custom fallback", () => {
    expect(extractSaveErrorMessage(null, "Could not fork")).toBe(
      "Could not fork",
    );
  });
});

describe("validateRequiredId", () => {
  it("rejects an empty / whitespace / undefined id (would be auto-renamed)", () => {
    expect(validateRequiredId("", "Event ID")).toMatch(/required/i);
    expect(validateRequiredId("   ", "Event ID")).toMatch(/required/i);
    expect(validateRequiredId(undefined, "Event ID")).toMatch(/required/i);
    expect(validateRequiredId(null, "Event ID")).toMatch(/required/i);
  });

  it("includes the field label and warns about the silent rename", () => {
    const msg = validateRequiredId("", "Idea ID")!;
    expect(msg).toContain("Idea ID");
    expect(msg).toMatch(/renamed/i);
  });

  it("accepts a non-blank id (charset / uniqueness stays the backend's job)", () => {
    expect(validateRequiredId("my_event.1", "Event ID")).toBeNull();
    expect(validateRequiredId("  trimmed_is_fine  ", "Event ID")).toBeNull();
  });
});
