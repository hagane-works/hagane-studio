import { readFileSync } from "node:fs";
import { resolve } from "node:path";
import { IntlMessageFormat } from "intl-messageformat";
import { describe, expect, it } from "vitest";
import {
  GATE_PARITY_FINDINGS,
  gateParityRepairRoute,
} from "./gateParityFindingMetadata";

const locales = ["en", "zh", "ru", "de", "es", "fr", "ja", "ko"];
type Copy = Record<string, Record<string, string>>;
const read = (locale: string): Copy =>
  JSON.parse(
    readFileSync(
      resolve(process.cwd(), "messages", locale, "exportFindings.json"),
      "utf8",
    ),
  );
const english = read("en");
const fields = ["what", "impact", "action", "jump"];
const parameters = (value: string) =>
  [...value.matchAll(/\{(\w+)\}/g)].map((m) => m[1]).sort();

describe("gate parity export copy", () => {
  for (const locale of locales) {
    it(`${locale}: every registered finding has complete, compilable copy with matching parameters`, () => {
      const corpus = read(locale);
      for (const { i18nKey } of Object.values(GATE_PARITY_FINDINGS)) {
        const entry = corpus[i18nKey];
        expect(Object.keys(entry).sort(), i18nKey).toEqual([...fields].sort());
        for (const field of fields) {
          const value = entry[field];
          expect(value.trim().length, `${i18nKey}.${field}`).toBeGreaterThan(0);
          expect(value).not.toContain("\uFFFD");
          expect(parameters(value)).toEqual(
            parameters(english[i18nKey][field]),
          );
          const values = Object.fromEntries(
            parameters(value).map((p) => [
              p,
              p === "default_count" ? 3 : "TEST",
            ]),
          );
          expect(() =>
            new IntlMessageFormat(value, locale).format(values),
          ).not.toThrow();
          if (locale !== "en") expect(value).not.toBe(english[i18nKey][field]);
        }
      }
    });
  }

  it("keeps failed checks separate from author content findings", () => {
    expect(GATE_PARITY_FINDINGS["bookmark.filter_registry_missing"].tier).toBe(
      "health",
    );
    expect(GATE_PARITY_FINDINGS.focus_layout_unknown_mode.tier).toBe("health");
    expect(
      GATE_PARITY_FINDINGS["bookmark.default_country_not_interesting"].tier,
    ).toBe("attn");
  });
});

describe("known collection repair destinations", () => {
  it("routes only the known combinations to existing lists", () => {
    expect(
      gateParityRepairRoute("bookmark.default_multiple", "bookmark_set"),
    ).toBe("bookmarks");
    expect(
      gateParityRepairRoute("focus_layout_relative_cycle", "focus_policy"),
    ).toBe("focus-tree");
    expect(
      gateParityRepairRoute("focus_layout_same_cell", "focus_policy"),
    ).toBe("focus-tree");
    expect(
      gateParityRepairRoute(
        "timeline.base_timeline_source_unregistered",
        "timeline_plan",
      ),
    ).toBe("bookmarks");
    expect(
      gateParityRepairRoute("bookmark.future_rule", "bookmark_set"),
    ).toBeNull();
    expect(
      gateParityRepairRoute("focus_layout_missing_anchor", "gui_widget"),
    ).toBeNull();
    expect(
      gateParityRepairRoute("focus_layout_missing_anchor", "focus"),
    ).toBeNull();
    expect(
      gateParityRepairRoute("focus_layout_future_rule", "focus_policy"),
    ).toBeNull();
  });
});
