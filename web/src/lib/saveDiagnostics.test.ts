import { describe, expect, it } from "vitest";
import {
  readSaveDiagnostics,
  mergeSaveDiagnostics,
  withoutSaveDiagnostics,
  type SaveDiagnostic,
} from "./saveDiagnostics";

const issue: SaveDiagnostic = {
  key: "bookmark:1:default",
  code: "bookmark.default_country_not_interesting",
  severity: "warning",
  entity_ref: { type: "bookmark", id: "1" },
  evidence: { default_country: "USA" },
  affected: ["USA"],
  repair_route: "bookmarks",
  change: "introduced",
};
describe("save diagnostics transport compatibility", () => {
  it("keeps absent and explicitly assessed empty distinct without reading old warnings", () => {
    expect(readSaveDiagnostics({ warnings: ["old warning"] })).toBeUndefined();
    expect(readSaveDiagnostics({ warnings: [issue] })).toBeUndefined();
    expect(readSaveDiagnostics({ diagnostics: [] })).toEqual([]);
    expect(mergeSaveDiagnostics([issue], undefined)).toEqual([issue]);
    expect(mergeSaveDiagnostics([issue], [])).toEqual([]);
  });
  it("handles success/direct fetch, nested detail and both ApiError payload layouts", () => {
    for (const body of [
      { diagnostics: [issue] },
      { detail: { diagnostics: [issue] } },
      { payload: { diagnostics: [issue] } },
      { payload: { code: "invalid", detail: { diagnostics: [issue] } } },
    ])
      expect(readSaveDiagnostics(body)).toEqual([issue]);
  });
  it("malformed assessments cannot erase old findings; cycles and primitive responses are safe", () => {
    expect(
      readSaveDiagnostics({ diagnostics: [{ code: issue.code }] }),
    ).toBeUndefined();
    const cyclic: Record<string, unknown> = {};
    cyclic.payload = cyclic;
    for (const body of [cyclic, [], null, "error", undefined])
      expect(readSaveDiagnostics(body)).toBeUndefined();
  });
  it("resolved rows clear current findings without leaking into a write payload", () => {
    expect(
      mergeSaveDiagnostics([issue], [{ ...issue, change: "resolved" }]),
    ).toEqual([]);
    const row = {
      id: 1,
      diagnostics: [issue],
      warnings: ["legacy"],
      name: "New name",
    };
    expect(withoutSaveDiagnostics(row)).toEqual({
      id: 1,
      warnings: ["legacy"],
      name: "New name",
    });
    expect(row.diagnostics).toEqual([issue]);
  });
});
