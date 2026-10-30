"use client";

import { useTranslations } from "use-intl";
import { X } from "lucide-react";

function LineSwatch({ color, dashed }: { color: string; dashed?: boolean }) {
  return (
    <svg width="28" height="8" className="shrink-0">
      <line
        x1="1"
        y1="4"
        x2="27"
        y2="4"
        stroke={color}
        strokeWidth="2"
        strokeDasharray={dashed ? "4 2" : undefined}
      />
    </svg>
  );
}

function Row({ swatch, text }: { swatch: React.ReactNode; text: string }) {
  return (
    <div className="flex items-start gap-2">
      <div className="mt-1">{swatch}</div>
      <span className="text-muted-foreground">{text}</span>
    </div>
  );
}

export function FocusGraphLegend({ onClose }: { onClose: () => void }) {
  const t = useTranslations("focusTreeGraph");

  return (
    <div className="ui-glass w-[280px] space-y-2 rounded-md p-3 text-xs">
      <div className="flex items-center justify-between">
        <span className="font-semibold">{t("legendTitle")}</span>
        <button onClick={onClose} className="ui-interactive" aria-label="close">
          <X className="h-3.5 w-3.5" />
        </button>
      </div>
      <Row
        swatch={<LineSwatch color="var(--color-muted-foreground)" />}
        text={t("legendPrereq")}
      />
      <Row
        swatch={
          <span className="rounded bg-background/80 px-1 text-micro font-semibold text-accent">
            OR
          </span>
        }
        text={t("legendOr")}
      />
      <Row
        swatch={<LineSwatch color="var(--color-focus-edge-group-1)" />}
        text={t("legendAndBlocks")}
      />
      <Row
        swatch={<LineSwatch color="var(--color-destructive)" dashed />}
        text={t("legendMutex")}
      />
    </div>
  );
}
