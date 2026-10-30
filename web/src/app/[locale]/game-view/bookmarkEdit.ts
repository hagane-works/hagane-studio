import type { GuiBookmark } from "@/lib/guiRenderApi";
import type { BookmarkRow } from "@/lib/api/scenario";

export function bookmarkRowFor(
  rows: readonly BookmarkRow[] | undefined,
  shown: GuiBookmark | undefined,
): BookmarkRow | null {
  if (!rows || !shown) return null;
  return rows.find((row) => row.date === shown.date) ?? null;
}

export type BookmarkProblem = "date";

export function dateProblem(date: string): BookmarkProblem | null {
  return /^\d{3,4}\.\d{1,2}\.\d{1,2}(\.\d{1,2})?$/.test(date.trim())
    ? null
    : "date";
}

export function bookmarkTitle(
  row: BookmarkRow,
  shown: GuiBookmark | undefined,
  fallback: string,
): string {
  return (
    row.display_name_locs?.en ||
    row.name_en ||
    shown?.label ||
    shown?.name ||
    fallback
  );
}
