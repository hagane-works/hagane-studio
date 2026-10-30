export type WarningTranslator = (
  key: string,
  values?: Record<string, string | number>,
) => string;

function paramsOf(segments: string[]): Record<string, string> {
  return Object.fromEntries(
    segments.map((p) => p.split("=", 2) as [string, string]),
  );
}

export function localizeDeploymentWarning(
  warning: string,
  t: WarningTranslator,
): string {
  const body = warning.replace(/^(actionable|info):\s*/, "");
  const [head, ...rest] = body.split("|");
  switch (head) {
    case "no_location":
      return t("warnNoLocation");
    case "unowned_province": {
      const params = paramsOf(rest);
      return t("warnUnownedProvince", {
        province: params.province ?? "?",
        tag: params.tag ?? "?",
      });
    }
    case "unknown_tag": {
      const params = paramsOf(rest);
      return t("warnUnknownTag", { tag: params.tag ?? "?" });
    }
    case "missing_template":
      return t("warnMissingTemplate", {
        template: rest.join("|").replace(/^template=/, ""),
      });
    case "no_template":
      return t("warnNoTemplate");
    default:
      return body;
  }
}
