"use client";

import { useTranslations } from "use-intl";

import { Progress } from "@/components/ui/progress";

export function GameLoadingOverlay({
  done,
  total,
}: {
  done: number;
  total: number;
}) {
  const t = useTranslations("gameViewControls");
  const value = total > 0 ? (done / total) * 100 : null;
  return (
    <div
      role="status"
      aria-live="polite"
      className="absolute inset-0 z-[80] flex items-center justify-center bg-ingame-panel-deep"
    >
      <div className="flex w-80 flex-col items-center gap-4 text-center">
        <span
          className="h-10 w-10 rounded-full border-2 border-ingame-gold animate-pulse-ring"
          aria-hidden
        />
        <p className="text-sm text-ingame-title">{t("loadingTitle")}</p>
        <Progress
          value={value}
          className="w-full"
          ariaLabel={t("loadingTitle")}
        />
        <p className="text-xs tabular-nums text-ingame-text">
          {total > 0
            ? t("loadingProgress", { done, total })
            : t("loadingStarting")}
        </p>
      </div>
    </div>
  );
}

export function GameLoadingPill() {
  const t = useTranslations("gameViewControls");
  return (
    <div
      role="status"
      aria-live="polite"
      className="pointer-events-none absolute left-1/2 top-3 z-[80] flex -translate-x-1/2 items-center gap-2 rounded-full border border-ingame-edge bg-ingame-panel px-3 py-1 text-xs text-ingame-text"
    >
      <span
        className="h-2 w-2 rounded-full bg-ingame-gold animate-pulse-ring"
        aria-hidden
      />
      {t("loadingBusy")}
    </div>
  );
}
