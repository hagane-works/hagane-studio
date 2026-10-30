"use client";

import { useState } from "react";
import { useTranslations } from "use-intl";

interface EventScopeBannerProps {
  ownCount: number;
  sharedCount: number;
  projectId: number;
  countryTag: string | null;

  hiddenSharedCount?: number | null;

  onOpenVanillaTab?: () => void;
}

const STORAGE_KEY_PREFIX = "event_scope_banner_dismissed_v1_p";

function dismissedSnapshot(projectId: number): string | null {
  if (typeof window === "undefined") return null;
  try {
    return localStorage.getItem(`${STORAGE_KEY_PREFIX}${projectId}`);
  } catch {
    return null;
  }
}

function setDismissedSnapshot(projectId: number, snapshot: string) {
  if (typeof window === "undefined") return;
  try {
    localStorage.setItem(`${STORAGE_KEY_PREFIX}${projectId}`, snapshot);
  } catch {}
}

export function EventScopeBanner({
  ownCount,
  sharedCount,
  projectId,
  countryTag,
  hiddenSharedCount = null,
  onOpenVanillaTab,
}: EventScopeBannerProps) {
  const t = useTranslations("eventScopeBanner");
  const currentSnapshot = `${ownCount}:${sharedCount}`;

  const [overrideForSnapshot, setOverrideForSnapshot] = useState<string | null>(
    null,
  );
  const persistedSnapshot = dismissedSnapshot(projectId);
  const dismissed =
    persistedSnapshot === currentSnapshot ||
    overrideForSnapshot === currentSnapshot;
  if (dismissed) return null;

  const handleDismiss = () => {
    setDismissedSnapshot(projectId, currentSnapshot);
    setOverrideForSnapshot(currentSnapshot);
  };

  return (
    <div className="rounded-md border border-border bg-muted/30 text-xs leading-snug">
      <div className="flex items-start gap-2 px-3 py-2">
        <span className="text-accent shrink-0" aria-hidden>
          ★
        </span>
        <div className="flex-1 min-w-0">
          <p className="text-foreground">
            <span className="font-medium">
              {t("ownCount", { ownCount, tag: countryTag || "own" })}
            </span>
            <span className="text-muted-foreground">
              {t("sharedSuffix", { sharedCount })}
            </span>
          </p>

          {hiddenSharedCount !== null && hiddenSharedCount > 0 && (
            <p className="text-muted-foreground mt-1">
              {t.rich("hiddenHint", {
                hiddenCount: hiddenSharedCount,
                emph: (c) => (
                  <span className="font-medium text-foreground">{c}</span>
                ),
              })}
            </p>
          )}
        </div>
        <button
          type="button"
          onClick={handleDismiss}
          aria-label={t("dismiss")}
          className="text-muted-foreground hover:text-foreground ui-interactive shrink-0"
          title={t("dontShowAgain")}
        >
          ×
        </button>
      </div>

      <div className="border-t border-border/60 px-3 py-2">
        <p className="text-muted-foreground">
          {t("filteredNote", { tag: countryTag || "own" })}
        </p>

        {onOpenVanillaTab && (
          <button
            type="button"
            onClick={onOpenVanillaTab}
            className="ui-interactive -ml-1.5 mt-1.5 inline-flex items-center gap-1 rounded-md px-1.5 py-1 font-medium text-accent outline-none hover:bg-accent/10 focus-visible:ring-1 focus-visible:ring-ring"
          >
            {t("vanillaCta")}
            <span aria-hidden>→</span>
          </button>
        )}
      </div>
    </div>
  );
}
