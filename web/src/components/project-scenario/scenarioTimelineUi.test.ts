import { describe, expect, it } from "vitest";

import type { BookmarkRow } from "@/lib/api";
import {
  earliestSavedBookmark,
  hasCustomBookmarkOpening,
  latestSavedBookmark,
  parseHoi4Date,
  sameHoi4Day,
  scenarioNeedsExplicitEndDate,
  validSavedBookmarks,
} from "./scenarioTimelineUi";

function bookmark(id: number, date: string): BookmarkRow {
  return { id, date } as BookmarkRow;
}

describe("scenario timeline UI projection", () => {
  it("uses the earliest valid saved bookmark, never the currently selected row", () => {
    const rows = [
      bookmark(3, "1964.5.1.12"),
      bookmark(1, "not-a-date"),
      bookmark(2, "1956.1.1"),
    ];

    expect(earliestSavedBookmark(rows)?.id).toBe(2);
    expect(latestSavedBookmark(rows)?.id).toBe(3);
    expect(validSavedBookmarks(rows).map((row) => row.id)).toEqual([3, 2]);
  });

  it("shows automatic bookmark replacement only for a valid custom opening", () => {
    expect(hasCustomBookmarkOpening([])).toBe(false);
    expect(hasCustomBookmarkOpening([bookmark(1, "bad")])).toBe(false);
    expect(hasCustomBookmarkOpening([bookmark(1, "1936.1.1")])).toBe(false);
    expect(hasCustomBookmarkOpening([bookmark(1, "1964.5.1")])).toBe(true);
    expect(
      hasCustomBookmarkOpening([
        bookmark(1, "1964.5.1"),
        bookmark(2, "1936.1.1.12"),
      ]),
    ).toBe(false);
  });

  it("matches the timeline planner's date validity and vanilla-end boundary", () => {
    expect(parseHoi4Date("183.1.1.12")).not.toBeNull();
    expect(parseHoi4Date("1964.13.1")).toBeNull();
    expect(sameHoi4Day("1936.1.1", "1936.1.1.12")).toBe(true);
    expect(scenarioNeedsExplicitEndDate("1948.12.31")).toBe(false);
    expect(scenarioNeedsExplicitEndDate("1949.1.1")).toBe(true);
  });
});
