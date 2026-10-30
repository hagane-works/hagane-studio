export type WarningTranslator = (
  key: string,
  values?: Record<string, string | number>,
) => string;

function paramsOf(segments: string[]): Record<string, string> {
  return Object.fromEntries(
    segments.map((p) => p.split("=", 2) as [string, string]),
  );
}

export function localizeOobWarning(
  warning: string,
  t: WarningTranslator,
): string {
  const body = String(warning ?? "").replace(/^(actionable|info):\s*/, "");
  const [head, ...rest] = body.split("|");
  switch (head) {
    case "no_location":
      return t("warnNoLocation");
    case "unknown_tag": {
      const params = paramsOf(rest);
      return t("warnUnknownTag", { tag: params.tag ?? "?" });
    }
    case "unowned_province": {
      const params = paramsOf(rest);
      return t("warnUnownedProvince", {
        province: params.province ?? "?",
        tag: params.tag ?? "?",
      });
    }
    case "unowned_state": {
      const params = paramsOf(rest);
      return t("warnUnownedState", {
        state: params.state ?? "?",
        tag: params.tag ?? "?",
      });
    }
    default:
      return body;
  }
}

export function oobRowWarnings(row: unknown): string[] {
  const value = (row as { warnings?: unknown } | null | undefined)?.warnings;
  if (!Array.isArray(value)) return [];
  return value.filter(
    (item): item is string => typeof item === "string" && item.length > 0,
  );
}
