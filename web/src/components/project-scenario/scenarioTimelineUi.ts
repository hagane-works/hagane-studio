import type { BookmarkRow } from "@/lib/api";

export const VANILLA_SCENARIO_START = "1936.1.1.12";

interface ParsedHoi4Date {
  year: number;
  month: number;
  day: number;
  hour: number;
}

export function parseHoi4Date(
  value: string | null | undefined,
): ParsedHoi4Date | null {
  const pieces = (value || "").trim().split(".");
  if (pieces.length !== 3 && pieces.length !== 4) return null;
  const values = pieces.map((piece) => Number(piece));
  if (!values.every(Number.isInteger)) return null;
  const [year, month, day] = values;
  const hour = values.length === 4 ? values[3] : 12;
  if (
    year <= 0 ||
    month < 1 ||
    month > 12 ||
    day < 1 ||
    day > 31 ||
    hour < 0 ||
    hour > 23
  ) {
    return null;
  }
  return { year, month, day, hour };
}

function compareHoi4Dates(left: ParsedHoi4Date, right: ParsedHoi4Date): number {
  for (const key of ["year", "month", "day", "hour"] as const) {
    if (left[key] !== right[key]) return left[key] - right[key];
  }
  return 0;
}

export function sameHoi4Day(
  left: string | null | undefined,
  right: string | null | undefined,
): boolean {
  const a = parseHoi4Date(left);
  const b = parseHoi4Date(right);
  return Boolean(
    a && b && a.year === b.year && a.month === b.month && a.day === b.day,
  );
}

export function validSavedBookmarks(
  rows: readonly BookmarkRow[],
): BookmarkRow[] {
  return rows.filter((row) => parseHoi4Date(row.date) !== null);
}

export function earliestSavedBookmark(
  rows: readonly BookmarkRow[],
): BookmarkRow | null {
  let earliest: BookmarkRow | null = null;
  let earliestDate: ParsedHoi4Date | null = null;
  for (const row of rows) {
    const parsed = parseHoi4Date(row.date);
    if (!parsed) continue;
    if (
      !earliest ||
      !earliestDate ||
      compareHoi4Dates(parsed, earliestDate) < 0
    ) {
      earliest = row;
      earliestDate = parsed;
    }
  }
  return earliest;
}

export function latestSavedBookmark(
  rows: readonly BookmarkRow[],
): BookmarkRow | null {
  let latest: BookmarkRow | null = null;
  let latestDate: ParsedHoi4Date | null = null;
  for (const row of rows) {
    const parsed = parseHoi4Date(row.date);
    if (!parsed) continue;
    if (!latest || !latestDate || compareHoi4Dates(parsed, latestDate) > 0) {
      latest = row;
      latestDate = parsed;
    }
  }
  return latest;
}

export function hasCustomBookmarkOpening(
  rows: readonly BookmarkRow[],
): boolean {
  const earliest = earliestSavedBookmark(rows);
  return Boolean(
    earliest && !sameHoi4Day(earliest.date, VANILLA_SCENARIO_START),
  );
}

export function scenarioNeedsExplicitEndDate(startDate: string): boolean {
  const parsed = parseHoi4Date(startDate);
  if (!parsed) return false;
  return (
    parsed.year > 1949 ||
    (parsed.year === 1949 &&
      (parsed.month > 1 || (parsed.month === 1 && parsed.day >= 1)))
  );
}
