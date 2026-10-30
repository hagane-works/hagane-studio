import { describe, expect, it } from "vitest";
import { formatBookmarkDate } from "./ScenarioSelectLayer";
import { formatGameDate, SHORT_MONTHS } from "./topbarStats";

const CHINESE_MONTHS = [
  "1月",
  "2月",
  "3月",
  "4月",
  "5月",
  "6月",
  "7月",
  "8月",
  "9月",
  "10月",
  "11月",
  "12月",
];

describe("dates in the game view's language", () => {
  it("keep the game's clock order in English and the other Latin languages", () => {
    expect(formatGameDate("1939.8.14.12", SHORT_MONTHS, "en")).toBe(
      "12:00, 14 Aug, 1939",
    );
    expect(
      formatGameDate(
        "1939.8.14.12",
        [
          "Jan",
          "Feb",
          "Mär",
          "Apr",
          "Mai",
          "Jun",
          "Jul",
          "Aug",
          "Sep",
          "Okt",
          "Nov",
          "Dez",
        ],
        "de",
      ),
    ).toBe("12:00, 14 Aug, 1939");
  });

  it("read year first in Chinese, Japanese and Korean", () => {
    expect(formatGameDate("1936.1.1.12", CHINESE_MONTHS, "zh")).toBe(
      "1936年1月1日 12:00",
    );
    expect(formatGameDate("1936.1.1.12", SHORT_MONTHS, "ja")).toBe(
      "1936年1月1日 12:00",
    );
    expect(formatGameDate("1936.1.1.12", SHORT_MONTHS, "ko")).toBe(
      "1936년 1월 1일 12:00",
    );
  });

  it("print bookmark dates the way the language writes them", () => {
    expect(formatBookmarkDate("1936.1.1.12", "en")).toBe("1 January 1936");
    expect(formatBookmarkDate("1936.1.1.12", "zh")).toBe("1936年1月1日");
    expect(formatBookmarkDate("1939.8.14.12", "de")).toBe("14. August 1939");
  });
});
