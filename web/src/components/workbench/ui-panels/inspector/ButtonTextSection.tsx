"use client";

import { useTranslations } from "use-intl";

import { DescriptionLocsTabs } from "@/components/shared/editors/DescriptionLocsTabs";
import { DEFAULT_EXPORT_LANGUAGES } from "@/components/shared/editors/displayNameLocsHelpers";
import { FontField } from "./FontField";
import { useProjectStore } from "@/lib/store/projectStore";
import type { UiPanelControl, UiPanelDerivedControl } from "@/types";

const EMPTY_LOCS: Record<string, string> = {};

export function ButtonTextSection({
  entityKey,
  ctrl,
  derived,
  onPatch,
}: {
  entityKey: string;
  ctrl: UiPanelControl;
  derived: UiPanelDerivedControl | undefined;
  onPatch: (partial: Partial<UiPanelControl>) => void;
}) {
  const t = useTranslations("uiPanelsInspector");

  const exportLanguages = useProjectStore(
    (s) => s.currentProject?.export_languages ?? DEFAULT_EXPORT_LANGUAGES,
  );

  const tooltipPresent =
    (ctrl.tooltip ?? "").trim() !== "" ||
    Object.values(ctrl.tooltip_locs ?? EMPTY_LOCS).some(
      (v) => (v ?? "").trim() !== "",
    );

  return (
    <div className="p-4 border-b border-border space-y-3">
      <span className="block text-micro uppercase tracking-widest font-semibold text-muted-foreground">
        {t("buttonTextTitle")}
      </span>
      <DescriptionLocsTabs
        entityKey={`${entityKey}:text`}
        value={ctrl.text_locs ?? EMPTY_LOCS}
        exportLanguages={exportLanguages}
        onChange={(next) =>
          onPatch({
            text_locs: next,

            text: next.en !== undefined ? next.en : (ctrl.text ?? ""),
          })
        }
        legacyFallback={{ en: ctrl.text }}
        inputComponent="input"
        placeholder={t("buttonTextPlaceholder")}
        ariaLabel={t("buttonTextTitle")}
      />
      {derived && (
        <p className="text-micro text-muted-foreground font-mono break-all">
          {t("textLocKeyHint")} {derived.text_loc_key}
        </p>
      )}
      <FontField ctrl={ctrl} onPatch={onPatch} />

      <span className="block text-micro uppercase tracking-widest font-semibold text-muted-foreground">
        {t("buttonTooltipTitle")}
      </span>
      <DescriptionLocsTabs
        entityKey={`${entityKey}:tooltip`}
        value={ctrl.tooltip_locs ?? EMPTY_LOCS}
        exportLanguages={exportLanguages}
        onChange={(next) =>
          onPatch({
            tooltip_locs: next,
            tooltip: next.en !== undefined ? next.en : (ctrl.tooltip ?? ""),
          })
        }
        legacyFallback={{ en: ctrl.tooltip }}
        inputComponent="textarea"
        rows={2}
        placeholder={t("buttonTooltipPlaceholder")}
        ariaLabel={t("buttonTooltipTitle")}
      />
      {tooltipPresent && derived ? (
        <p className="text-micro text-muted-foreground font-mono break-all">
          {t("exportAsHint")} {`pdx_tooltip = ${derived.tooltip_loc_key}`}
        </p>
      ) : (
        <p className="text-micro text-muted-foreground">
          {t("buttonTooltipEmptyHint")}
        </p>
      )}
    </div>
  );
}
