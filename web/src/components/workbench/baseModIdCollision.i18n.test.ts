import { readFileSync } from "node:fs";
import { join } from "node:path";
import { describe, expect, it } from "vitest";

const LOCALES = ["de", "en", "es", "fr", "ja", "ko", "ru", "zh"] as const;
const KEYS = [
  "message",
  "fixHint",
  "unknownDependency",
  "entityCharacter",
  "entityFocus",
  "entityIdea",
  "entityDecision",
] as const;

function load(locale: string): Record<string, string> {
  const path = join(
    process.cwd(),
    "messages",
    locale,
    "baseModIdCollision.json",
  );
  return JSON.parse(readFileSync(path, "utf8")) as Record<string, string>;
}

describe("baseModIdCollision message catalog", () => {
  it.each(LOCALES)("%s carries every key", (locale) => {
    const messages = load(locale);
    for (const key of KEYS) {
      expect(messages[key], `${locale}.${key}`).toBeTruthy();
    }
  });

  it.each(LOCALES)(
    "%s keeps the ICU placeholders the component passes",
    (locale) => {
      const messages = load(locale);
      for (const placeholder of ["{id}", "{dependency}", "{entityLabel}"]) {
        expect(messages.message).toContain(placeholder);
      }
      expect(messages.fixHint).toContain("{dependency}");
    },
  );

  it.each(LOCALES)("%s does not fall back to the English string", (locale) => {
    if (locale === "en") return;
    const english = load("en");
    expect(load(locale).message).not.toBe(english.message);
  });

  it.each(LOCALES)("%s does not threaten a crash", (locale) => {
    const text =
      `${load(locale).message} ${load(locale).fixHint}`.toLowerCase();
    for (const scare of [
      "crash",
      "ctd",
      "崩溃",
      "闪退",
      "absturz",
      "cierre",
      "plantage",
      "クラッシュ",
      "충돌",
      "вылет",
    ]) {
      expect(text, `${locale} says "${scare}"`).not.toContain(scare);
    }
  });
});
