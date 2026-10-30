import { readFileSync } from "node:fs";
import { join } from "node:path";
import { describe, expect, it } from "vitest";
import type { BookmarkRow } from "@/lib/api/scenario";
import type { GuiBookmark } from "@/lib/guiRenderApi";
import { bookmarkRowFor, bookmarkTitle, dateProblem } from "./bookmarkEdit";

const row = (extra: Partial<BookmarkRow> = {}): BookmarkRow =>
  ({
    id: 2,
    bookmark_id: "gathering_storm",
    name_en: "",
    display_name_locs: { en: "Gathering Storm", zh: "风起云涌" },
    description_locs: {},
    date: "1936.1.1.12",
    default_country: "GER",
    ...extra,
  }) as unknown as BookmarkRow;

const shown = {
  name: "GATHERING_STORM_NAME",
  label: "Gathering Storm",
  desc: "Dark times are coming.",
  date: "1936.1.1.12",
  default_country: "GER",
  countries: [],
} as unknown as GuiBookmark;

describe("the scenario edit panel", () => {
  it("finds the project's row by the scenario's start date", () => {
    expect(
      bookmarkRowFor([row({ id: 1, date: "1939.8.14.12" }), row()], shown)?.id,
    ).toBe(2);
    expect(bookmarkRowFor([row({ date: "1939.8.14.12" })], shown)).toBeNull();
  });

  it("names it by the row's English name, the screen's where the row has none", () => {
    expect(bookmarkTitle(row(), shown, "剧本")).toBe("Gathering Storm");
    expect(
      bookmarkTitle(
        row({ display_name_locs: {}, name_en: "The Storm" }),
        shown,
        "剧本",
      ),
    ).toBe("The Storm");
    expect(
      bookmarkTitle(
        row({ display_name_locs: {} }),
        { ...shown, label: "Storm Clouds" },
        "剧本",
      ),
    ).toBe("Storm Clouds");

    expect(
      bookmarkTitle(row({ display_name_locs: {} }), undefined, "Signet"),
    ).toBe("Signet");
  });

  it("wants a start date the game can read", () => {
    expect(dateProblem("1936.1.1.12")).toBeNull();
    expect(dateProblem("1936.1.1")).toBeNull();
    expect(dateProblem("1 Jan 1936")).toBe("date");
  });

  it("follows the tile picked on the scenario screen", () => {
    const page = readFileSync(join(__dirname, "page.tsx"), "utf8");
    expect(page).toContain("onPicked={onScenarioTile}");
    expect(page).toContain("shown={bookmarks.data?.bookmarks?.[scenarioTile]}");
  });
});
