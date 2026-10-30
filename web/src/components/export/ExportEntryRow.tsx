"use client";

import { Gamepad2, Search, Settings } from "lucide-react";
import { useTranslations } from "use-intl";

export function ExportEntryRow({
  checkSubtitle,
  verifyUnlocked,
  onOpenDetails,
  onOpenVerify,
  onOpenSettings,
}: {
  checkSubtitle: string;
  verifyUnlocked: boolean;
  onOpenDetails: () => void;
  onOpenVerify: () => void;
  onOpenSettings: () => void;
}) {
  const t = useTranslations("export");

  const entryClass =
    "flex items-center gap-2.5 rounded-lg border border-border bg-card px-3.5 py-3 text-left ui-interactive ui-focus-ring hover:bg-muted/40 disabled:cursor-not-allowed disabled:opacity-55 disabled:hover:bg-card";

  return (
    <section
      aria-label={t("ui.entries.aria")}
      className="grid grid-cols-1 gap-2.5 md:grid-cols-3"
    >
      <button type="button" onClick={onOpenDetails} className={entryClass}>
        <Search
          className="h-4 w-4 shrink-0 text-muted-foreground"
          aria-hidden
        />
        <span className="min-w-0 flex-1">
          <span className="block text-xs font-semibold text-foreground">
            {t("ui.entries.checkTitle")}
          </span>
          <span className="block truncate text-micro text-muted-foreground">
            {checkSubtitle}
          </span>
        </span>
        <span aria-hidden className="text-xs text-muted-foreground">
          ⧉
        </span>
      </button>

      <button
        type="button"
        onClick={onOpenVerify}
        disabled={!verifyUnlocked}
        title={verifyUnlocked ? undefined : t("ui.entries.verifyLockedTip")}
        className={entryClass}
      >
        <Gamepad2
          className="h-4 w-4 shrink-0 text-muted-foreground"
          aria-hidden
        />
        <span className="min-w-0 flex-1">
          <span className="block text-xs font-semibold text-foreground">
            {t("ui.entries.verifyTitle")}
          </span>
          <span className="block truncate text-micro text-muted-foreground">
            {verifyUnlocked
              ? t("ui.entries.verifySubReady")
              : t("ui.entries.verifySubLocked")}
          </span>
        </span>
        <span aria-hidden className="text-xs text-muted-foreground">
          ⧉
        </span>
      </button>

      <button type="button" onClick={onOpenSettings} className={entryClass}>
        <Settings
          className="h-4 w-4 shrink-0 text-muted-foreground"
          aria-hidden
        />
        <span className="min-w-0 flex-1">
          <span className="block text-xs font-semibold text-foreground">
            {t("ui.entries.settingsTitle")}
          </span>
          <span className="block truncate text-micro text-muted-foreground">
            {t("ui.entries.settingsSub")}
          </span>
        </span>
        <span aria-hidden className="text-xs text-muted-foreground">
          ⧉
        </span>
      </button>
    </section>
  );
}
