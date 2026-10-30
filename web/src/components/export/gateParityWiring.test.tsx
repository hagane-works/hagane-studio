import { describe, expect, it } from "vitest";
import { EXPORT_FINDINGS } from "@/lib/exportFindingsMap";
import { GATE_PARITY_FINDINGS } from "@/lib/gateParityFindingMetadata";
import { presentCheckIssues } from "@/lib/exportCheckPresentation";
import { countGroupsWithEntryPoint } from "@/lib/exportRepairQueue";

describe("gate parity central display seam", () => {
  it("registers every producer code in the actual export display chain", () => {
    for (const [code, meta] of Object.entries(GATE_PARITY_FINDINGS)) {
      expect(EXPORT_FINDINGS[code], code).toEqual(meta);
    }
  });

  it("keeps collection findings navigable in the repair queue as well as the card", () => {
    const result = presentCheckIssues([
      {
        code: "bookmark.default_multiple",
        severity: "blocker",
        message: "bookmark.default_multiple",
        fix_hint: "bookmarks.default.select_one",
        entity_ref: { type: "bookmark_set", id: "effective" },
        evidence: { default_count: 3, bookmark_ids: ["one", "two", "three"] },
      },
    ]);
    expect(countGroupsWithEntryPoint(result.attentionGroups)).toBe(1);
  });
});
