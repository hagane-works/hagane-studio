import type { Locale } from "@/lib/store/localeStore";
import type { UnitNames } from "@/lib/vanillaUnitNames";

export function subunitOptionLabel(
  token: string,
  locale: Locale,
  names: UnitNames,
): string {
  if (locale !== "en") {
    const gloss = names[token]?.[locale];
    if (gloss) return `${token} · ${gloss}`;
  }
  return token;
}
