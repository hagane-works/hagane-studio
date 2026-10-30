"use client";

import { useMemo } from "react";
import { useTranslations } from "use-intl";
import { ChevronRight } from "lucide-react";

export function RawExtrasNotice({
  rawExtras,
}: {
  rawExtras: Record<string, unknown> | null | undefined;
}) {
  const t = useTranslations("techEditor");
  const keys = useMemo(() => Object.keys(rawExtras ?? {}), [rawExtras]);

  const json = useMemo(
    () => (keys.length > 0 ? JSON.stringify(rawExtras, null, 2) : ""),
    [rawExtras, keys.length],
  );
  if (keys.length === 0) return null;

  return (
    <details className="group rounded-md border border-border bg-muted/30 px-3 py-2">
      <summary className="cursor-pointer select-none flex items-center gap-2 text-xs text-muted-foreground">
        <ChevronRight className="w-3.5 h-3.5 shrink-0 transition-transform group-open:rotate-90" />
        <span className="px-2 py-0.5 rounded-full bg-muted text-muted-foreground border border-border">
          {t("rawExtras.badge")}
        </span>
        <span>{t("rawExtras.count", { count: keys.length })}</span>
      </summary>
      <div className="mt-2 space-y-2">
        <p className="text-xs text-muted-foreground">{t("rawExtras.hint")}</p>
        <div>
          <p className="text-xs text-muted-foreground mb-1">
            {t("rawExtras.keysLabel")}
          </p>
          <div className="flex flex-wrap gap-1">
            {keys.map((k) => (
              <span
                key={k}
                className="font-mono text-xs px-1.5 py-0.5 rounded border border-border bg-background/50 text-foreground"
              >
                {k}
              </span>
            ))}
          </div>
        </div>
        <div>
          <p className="text-xs text-muted-foreground mb-1">
            {t("rawExtras.jsonLabel")}
          </p>
          <pre className="font-mono text-xs whitespace-pre-wrap max-h-64 overflow-auto rounded border border-border bg-background/50 p-2">
            {json}
          </pre>
        </div>
      </div>
    </details>
  );
}
