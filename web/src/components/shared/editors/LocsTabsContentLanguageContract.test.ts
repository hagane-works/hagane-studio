import { readFileSync } from "node:fs";
import { join } from "node:path";
import { describe, expect, it } from "vitest";

const componentNames = ["DisplayNameLocsTabs", "DescriptionLocsTabs"] as const;
const appLocales = ["en", "zh", "ru", "de", "es", "fr", "ja", "ko"] as const;
const namespaces = ["displayNameLocsTabs", "descriptionLocsTabs"] as const;

function source(name: (typeof componentNames)[number]): string {
  return readFileSync(join(__dirname, `${name}.tsx`), "utf8")
    .replace(/\/\*[\s\S]*?\*\//g, "")
    .replace(/(^|[^:])\/\/.*$/gm, "$1");
}

function message(
  locale: (typeof appLocales)[number],
  namespace: (typeof namespaces)[number],
): Record<string, unknown> {
  return JSON.parse(
    readFileSync(
      join(__dirname, "../../../..", "messages", locale, `${namespace}.json`),
      "utf8",
    ),
  ) as Record<string, unknown>;
}

function icuParameters(value: unknown): string[] {
  if (typeof value !== "string") return [];
  return Array.from(
    value.matchAll(/\{([A-Za-z][A-Za-z0-9_]*)\}/g),
    (match) => match[1],
  ).sort();
}

describe("shared localisation tabs project-language contract", () => {
  it.each(componentNames)(
    "%s consumes the shared action inside an entity-keyed state boundary",
    (name) => {
      const component = source(name);

      expect(component).toMatch(
        /useProjectStore\(\s*\(state\) => state\.setContentLanguageEnabled,?\s*\)/,
      );
      expect(component).toMatch(
        /await setContentLanguageEnabled\(lang, true\);[\s\S]*?onChangeRef\.current\(nextDict\)/,
      );
      expect(component).toMatch(/entityKey:\s*React\.Key/);
      expect(component).toMatch(
        new RegExp(
          `return <${name}ForEntity key=\\{props\\.entityKey\\} \\{\\.\\.\\.props\\} \\/>;`,
        ),
      );
      expect(component).toMatch(
        /await setContentLanguageEnabled\(lang, true\);[\s\S]*?if \(!mountedRef\.current\) return;/,
      );
      expect(component).not.toMatch(/\bprojectsApi\b|\bupdateProject\b/);
    },
  );

  it.each(namespaces)(
    "%s keeps all eight AppLocale files aligned without English-copy placeholders",
    (namespace) => {
      const english = message("en", namespace);
      for (const locale of appLocales) {
        const translated = message(locale, namespace);
        expect(Object.keys(translated).sort()).toEqual(
          Object.keys(english).sort(),
        );
        for (const key of ["addingLanguage", "addLanguageError"] as const) {
          expect(icuParameters(translated[key])).toEqual(["language"]);
          if (locale !== "en") expect(translated[key]).not.toBe(english[key]);
        }
      }
    },
  );
});
