"use client";

import { memo } from "react";
import { useTranslations } from "use-intl";

import type { ScriptLintResult } from "./scriptLint";

const SEVERITY_CLASS: Record<string, string> = {
  error: "text-destructive",
  warning: "text-warning",
  info: "text-muted-foreground",
};

export const ScriptLintMessages = memo(function ScriptLintMessages({
  result,
}: {
  result: ScriptLintResult;
}) {
  const t = useTranslations("uiPanelsInspector");

  if (result.markers.length === 0 && result.unknownTokens.length === 0) {
    return null;
  }

  return (
    <ul className="space-y-0.5" aria-live="polite">
      {result.markers.map((marker, index) => (
        <li
          key={`cwtool-${marker.line}-${index}`}
          className={`text-xs font-mono break-all ${SEVERITY_CLASS[marker.severity] ?? "text-muted-foreground"}`}
        >
          L{marker.line} · {marker.msg}
        </li>
      ))}
      {result.unknownTokens.map((issue, index) => (
        <li
          key={`token-${issue.line}-${index}`}
          className="text-xs font-mono break-all text-muted-foreground"
        >
          L{issue.line} · `{issue.token}` {t("lintUnknownToken")}
          {issue.suggestion
            ? ` · ${t("lintDidYouMean")} \`${issue.suggestion}\`?`
            : null}
        </li>
      ))}
    </ul>
  );
});
