import { readFileSync } from "node:fs";
import path from "node:path";
import { IntlMessageFormat } from "intl-messageformat";
import { describe, expect, it } from "vitest";

const MESSAGES_ROOT = path.resolve(process.cwd(), "messages");
const NON_ENGLISH_LOCALES = ["de", "es", "fr", "ja", "ko", "ru", "zh"] as const;

type FlatMessages = Record<string, string>;

function flatten(
  value: unknown,
  prefix = "",
  out: FlatMessages = {},
): FlatMessages {
  if (value && typeof value === "object" && !Array.isArray(value)) {
    for (const [key, child] of Object.entries(value)) {
      flatten(child, prefix ? `${prefix}.${key}` : key, out);
    }
  } else if (typeof value === "string") {
    out[prefix] = value;
  }
  return out;
}

function loadWorkbench(locale: string): FlatMessages {
  const file = path.join(MESSAGES_ROOT, locale, "countrySettingsDialog.json");
  const messages = JSON.parse(readFileSync(file, "utf8")) as {
    workbench?: unknown;
  };
  return flatten(messages.workbench);
}

function icuArguments(message: string, locale: string): string[] {
  const result = new Set<string>();
  const ast = new IntlMessageFormat(message, locale).getAst() as unknown[];

  const visit = (nodes: unknown[]): void => {
    for (const rawNode of nodes) {
      const node = rawNode as {
        type: number;
        value?: unknown;
        options?: Record<string, { value: unknown[] }>;
        children?: unknown[];
      };
      if (node.type >= 1 && node.type <= 6 && typeof node.value === "string") {
        result.add(node.value);
      }
      for (const option of Object.values(node.options ?? {}))
        visit(option.value);
      if (Array.isArray(node.children)) visit(node.children);
    }
  };

  visit(ast);
  return [...result].sort();
}

describe("country workbench translations", () => {
  const english = loadWorkbench("en");
  const englishKeys = Object.keys(english).sort();

  it("keeps the complete workbench key shape in every AppLocale", () => {
    expect(englishKeys).toHaveLength(80);
    for (const locale of NON_ENGLISH_LOCALES) {
      expect(Object.keys(loadWorkbench(locale)).sort(), locale).toEqual(
        englishKeys,
      );
    }
  });

  it("preserves the exact ICU argument set for every localized message", () => {
    const mismatches: string[] = [];
    for (const locale of NON_ENGLISH_LOCALES) {
      const localized = loadWorkbench(locale);
      for (const key of englishKeys) {
        const expected = icuArguments(english[key], "en");
        const actual = icuArguments(localized[key], locale);
        if (expected.join("\u0000") !== actual.join("\u0000")) {
          mismatches.push(
            `${locale}:${key}: en=[${expected.join(",")}] locale=[${actual.join(",")}]`,
          );
        }
      }
    }
    expect(mismatches).toEqual([]);
  });

  it("does not copy English workbench sentences into non-English locales", () => {
    const copied: string[] = [];
    for (const locale of NON_ENGLISH_LOCALES) {
      const localized = loadWorkbench(locale);
      for (const key of englishKeys) {
        if (localized[key] === english[key]) copied.push(`${locale}:${key}`);
      }
    }

    expect(copied).toEqual([]);
  });
});
