import { readFileSync } from "node:fs";
import { join } from "node:path";
import { describe, expect, it } from "vitest";

const dialogSource = readFileSync(
  join(__dirname, "country-profile/CountryProfileEditor.tsx"),
  "utf8",
);
const apiSource = readFileSync(
  join(__dirname, "../../lib/api/scenario.ts"),
  "utf8",
);

describe("country adjective authoring wiring", () => {
  it("keeps the adjective fields on the shared country API contract", () => {
    expect(apiSource).toContain("adjective_en: string;");
    expect(apiSource).toContain("adjective_zh: string;");
    expect(apiSource).toContain("adjective_locs: Record<string, string>;");
    expect(apiSource).toContain(
      "ideology_name_locs: Record<string, Record<string, string>>;",
    );
  });

  it("hydrates, edits, and saves canonical multi-language TAG_ADJ values", () => {
    for (const token of [
      "adjective_en: draft.adjectiveEn",
      "adjective_zh: draft.adjectiveZh",
      "adjective_locs: draft.adjectiveLocs",
      "ideology_name_locs: draft.ideologyNameLocs",
      "value={draft.adjectiveLocs}",
      'updateDraft("adjectiveLocs", value)',
    ]) {
      expect(dialogSource).toContain(token);
    }
  });
});
