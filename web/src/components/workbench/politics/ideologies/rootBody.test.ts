import { describe, expect, it } from "vitest";
import {
  defaultRootBody,
  rootBodyColor,
  rootBodyDraftError,
  rootBodyWithColor,
} from "./rootBody";

describe("political-group advanced script helpers", () => {
  it("generates a safe default from the visual color", () => {
    expect(defaultRootBody("#2d5f87")).toBe(
      "types = {\n}\n\ncolor = { 45 95 135 }",
    );
  });

  it("reads and updates color without overwriting other advanced rules", () => {
    const body =
      "types = { civic_reformism = { } }\ncolor = { 1 2 3 }\nrules = { can_send_volunteers = yes }";
    const updated = rootBodyWithColor(body, "#2d5f87");
    expect(rootBodyColor(updated)).toBe("#2d5f87");
    expect(updated).toContain("rules = { can_send_volunteers = yes }");
  });

  it("reports recoverable draft errors before save", () => {
    expect(rootBodyDraftError("color = { 1 2 3 }")).toBe("missing_types");
    expect(rootBodyDraftError("types = { }")).toBe("missing_color");
    expect(rootBodyDraftError("types = { }\ncolor = { 1 2 3 }\n}")).toBe(
      "unbalanced",
    );
    expect(rootBodyDraftError(defaultRootBody())).toBeNull();
  });
});
