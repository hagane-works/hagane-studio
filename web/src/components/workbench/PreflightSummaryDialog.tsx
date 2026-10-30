"use client";

import { useMemo } from "react";
import { useTranslations } from "use-intl";
import { Card, CardContent } from "@/components/ui/card";

export interface PreflightStat {
  key:
    | "countries"
    | "characters"
    | "ideas"
    | "focus_trees"
    | "events"
    | "decisions"
    | "dynamic_modifiers"
    | "superevents";

  count: number | undefined;

  subline?: string;
}

export interface PreflightSummaryContentProps {
  stats: PreflightStat[];

  totalSizeBytes: number;

  totalFileCount: number;

  onBackgroundToggle?: (value: boolean) => void;

  backgroundMode?: boolean;
}

function estimateEta(totalBytes: number): {
  minutes: number | undefined;
  seconds: number | undefined;
  mb: number;
} {
  if (totalBytes <= 0) {
    return { minutes: undefined, seconds: undefined, mb: 0 };
  }
  const mb = totalBytes / (1024 * 1024);

  const baseMinutes = (mb / 50) * 1.6;
  if (mb < 30 || baseMinutes < 1) {
    const seconds = Math.max(5, Math.round(baseMinutes * 60));
    return { minutes: undefined, seconds, mb: Math.round(mb * 10) / 10 };
  }
  return {
    minutes: Math.max(1, Math.round(baseMinutes)),
    seconds: undefined,
    mb: Math.round(mb * 10) / 10,
  };
}

export function PreflightSummaryContent({
  stats,
  totalSizeBytes,
  totalFileCount,
  onBackgroundToggle,
  backgroundMode = false,
}: PreflightSummaryContentProps) {
  const t = useTranslations("preflightDialog");

  const eta = useMemo(() => estimateEta(totalSizeBytes), [totalSizeBytes]);

  return (
    <div className="space-y-3">
      <div>
        <h3 className="text-base font-semibold tracking-tight">{t("title")}</h3>
        <p className="mt-0.5 text-xs text-muted-foreground">{t("subtitle")}</p>
      </div>

      <div className="grid grid-cols-2 sm:grid-cols-4 gap-2">
        {stats.map((stat) => {
          const isAvailable = typeof stat.count === "number";
          return (
            <Card
              key={stat.key}
              size="sm"
              data-testid={`preflight-stat-${stat.key}`}
              className={isAvailable ? "px-3 py-2" : "px-3 py-2 opacity-60"}
            >
              <CardContent className="px-0 space-y-0.5">
                <p className="text-2xl font-semibold tabular-nums leading-none">
                  {isAvailable
                    ? stat.count!.toLocaleString()
                    : t("statFallback")}
                </p>
                <p className="text-xs text-muted-foreground leading-tight">
                  {t(`stat.${stat.key}`)}
                </p>
                {stat.subline && (
                  <p className="text-micro text-muted-foreground leading-tight pt-0.5">
                    {stat.subline}
                  </p>
                )}
              </CardContent>
            </Card>
          );
        })}
      </div>

      <div className="rounded-md border border-border bg-muted/40 p-3 text-xs space-y-1">
        <div className="flex items-center justify-between">
          <span className="font-medium">{t("etaLabel")}</span>
          <span className="font-mono tabular-nums">
            {totalSizeBytes > 0
              ? eta.minutes !== undefined
                ? t("etaUnitMinutes", { minutes: eta.minutes })
                : t("etaUnitSeconds", { seconds: eta.seconds ?? 0 })
              : t("etaFallback")}
          </span>
        </div>
        {totalSizeBytes > 0 && (
          <p className="text-muted-foreground">
            {t("etaUnit", {
              mb: eta.mb,
              files: totalFileCount.toLocaleString(),
            })}
          </p>
        )}
        <p className="text-muted-foreground/80 italic">{t("etaHint")}</p>
      </div>

      {onBackgroundToggle && (
        <label className="flex items-start gap-2 rounded-md border border-border bg-muted/30 p-3 text-xs cursor-pointer">
          <input
            type="checkbox"
            checked={backgroundMode}
            onChange={(e) => onBackgroundToggle(e.target.checked)}
            className="mt-0.5 h-4 w-4 shrink-0 accent-primary"
            data-testid="preflight-background-toggle"
          />
          <div>
            <p className="font-medium">{t("backgroundToggleLabel")}</p>
            <p className="text-muted-foreground">{t("backgroundToggleHint")}</p>
          </div>
        </label>
      )}

      {stats.some((s) => s.count === undefined) && (
        <p className="text-xs text-muted-foreground">{t("notAvailableHint")}</p>
      )}
    </div>
  );
}
