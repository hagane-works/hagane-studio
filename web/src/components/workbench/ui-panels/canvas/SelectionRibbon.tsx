"use client";

import { useTranslations } from "use-intl";

import type { ControlEntry } from "./nodeControlMap";

export function SelectionRibbon({
  entry,
  isGhost,
  mockEnabled,
}: {
  entry: ControlEntry | null;

  isGhost: boolean;

  mockEnabled: boolean;
}) {
  const t = useTranslations("uiPanelsCanvas");
  const frameVar = entry?.ctrl.frame_var?.trim();
  const imageExpr = entry?.ctrl.image_expr?.trim();
  const visibleTrigger = entry?.ctrl.visible_trigger?.trim();
  return (
    <div className="flex items-center gap-3 px-3 py-1.5 border-b border-border bg-card font-mono text-xs whitespace-nowrap overflow-x-auto">
      {entry ? (
        <>
          <span className="text-accent font-medium">
            {entry.derivedName ?? entry.ctrl.name}
          </span>
          <span className="text-micro uppercase tracking-widest text-muted-foreground">
            {entry.ctrl.kind}
          </span>
          {frameVar && (
            <span className="text-muted-foreground">
              frame = <span className="text-foreground">{frameVar}</span>
            </span>
          )}
          {imageExpr && (
            <span className="text-muted-foreground">
              image = <span className="text-foreground">{imageExpr}</span>
            </span>
          )}
          {visibleTrigger && (
            <span className="text-muted-foreground" title={visibleTrigger}>
              visible ✓
            </span>
          )}
          {isGhost && (
            <span className="text-accent font-sans">{t("ribbonGhost")}</span>
          )}
        </>
      ) : (
        <span className="text-muted-foreground font-sans">
          {t("ribbonEmpty")}
        </span>
      )}
      {mockEnabled && (
        <span className="ml-auto text-micro text-muted-foreground font-sans">
          {t("mockPreviewOn")}
        </span>
      )}
    </div>
  );
}
