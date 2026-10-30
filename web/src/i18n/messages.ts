import type { AppLocale } from "./routing";

type Messages = Record<string, unknown>;

const files = import.meta.glob<Messages>("../../messages/*/*.json", {
  import: "default",
});

async function loadLocale(locale: AppLocale): Promise<Messages> {
  const prefix = `../../messages/${locale}/`;
  const entries = await Promise.all(
    Object.entries(files)
      .filter(([path]) => path.startsWith(prefix))
      .map(
        async ([path, load]) =>
          [path.slice(prefix.length, -".json".length), await load()] as const,
      ),
  );
  return Object.fromEntries(entries);
}

export async function loadMessages(locale: AppLocale): Promise<Messages> {
  if (locale === "en") return loadLocale("en");
  const [fallback, localized] = await Promise.all([
    loadLocale("en"),
    loadLocale(locale),
  ]);
  return mergeMessages(fallback, localized);
}

function mergeMessages(fallback: Messages, localized: Messages): Messages {
  const merged: Messages = { ...fallback };
  for (const [key, value] of Object.entries(localized)) {
    const base = fallback[key];
    merged[key] =
      isMessageObject(base) && isMessageObject(value)
        ? mergeMessages(base, value)
        : value;
  }
  return merged;
}

function isMessageObject(value: unknown): value is Messages {
  return Boolean(value) && typeof value === "object" && !Array.isArray(value);
}
