import { readFileSync } from "node:fs";
import path from "node:path";
import { describe, expect, it } from "vitest";

const locales = ["en", "zh", "ru", "de", "es", "fr", "ja", "ko"] as const;

function flatten(value: unknown, prefix = ""): string[] {
  if (!value || typeof value !== "object" || Array.isArray(value))
    return [prefix];
  return Object.entries(value as Record<string, unknown>).flatMap(
    ([key, child]) => flatten(child, prefix ? `${prefix}.${key}` : key),
  );
}

function placeholders(value: unknown): string[] {
  if (typeof value !== "string") return [];
  return [...value.matchAll(/\{([a-zA-Z0-9_]+)\}/g)]
    .map((match) => match[1])
    .sort();
}

describe("politicsPanel messages", () => {
  it("keeps the same keys and ICU parameters in all eight app locales", () => {
    const bundles = Object.fromEntries(
      locales.map((locale) => {
        const file = path.resolve(
          __dirname,
          `../../../messages/${locale}/politicsPanel.json`,
        );
        return [
          locale,
          JSON.parse(readFileSync(file, "utf8")) as Record<string, unknown>,
        ];
      }),
    );
    const englishKeys = flatten(bundles.en).sort();

    for (const locale of locales) {
      expect(flatten(bundles[locale]).sort(), locale).toEqual(englishKeys);
      for (const key of englishKeys) {
        const read = (root: Record<string, unknown>) =>
          key
            .split(".")
            .reduce<unknown>(
              (current, part) => (current as Record<string, unknown>)[part],
              root,
            );
        expect(placeholders(read(bundles[locale])), `${locale}:${key}`).toEqual(
          placeholders(read(bundles.en)),
        );
      }
    }
  });
});
