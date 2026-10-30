import type { Locale } from "@/lib/store/localeStore";

export function localizedOptionLabel(label: string, locale: Locale): string {
  const slash = label.match(/^([A-Za-z][A-Za-z0-9 .+-]*?)\s*\/\s*(.+)$/);
  if (slash) {
    const [, en, zh] = slash;
    return locale === "zh" ? zh.trim() : en.trim();
  }

  const paren = label.match(/^(.+?)\s*\(([A-Za-z][^)]*)\)\s*$/);
  if (paren) {
    const [, zh, en] = paren;
    return locale === "zh" ? zh.trim() : en.trim();
  }

  const mid = label.match(/^([A-Za-z_][A-Za-z0-9_]*)\s*·\s*(.*[一-鿿].*)$/);
  if (mid) {
    return locale === "zh" ? label : mid[1];
  }

  const tokenParen = label.match(
    /^([A-Za-z_][A-Za-z0-9_]*)\s*\(([^)]*[一-鿿][^)]*)\)$/,
  );
  if (tokenParen) {
    return locale === "zh" ? label : tokenParen[1];
  }
  return label;
}
