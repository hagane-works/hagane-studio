"use client";

import { useTranslations } from "use-intl";

import { ScriptBlockEditor } from "@/components/shared/editors/ScriptBlockEditor";
import { CONDITION_TEMPLATES } from "@/components/shared/editors/conditionTemplates";
import type { UiPanelControl, UiPanelDerivedControl } from "@/types";

const VISIBLE_PLACEHOLDER = `check_variable = { legitimacy > 20 }
has_country_flag = my_flag`;

export function VisibleTriggerSection({
  ctrl,
  derived,
  onPatch,
}: {
  ctrl: UiPanelControl;
  derived: UiPanelDerivedControl | undefined;
  onPatch: (partial: Partial<UiPanelControl>) => void;
}) {
  const t = useTranslations("uiPanelsInspector");
  const value = ctrl.visible_trigger ?? "";

  return (
    <div className="p-4 border-b border-border space-y-2">
      <ScriptBlockEditor
        label={t("visibleTitle")}
        labelTooltip={t("visibleTooltip")}
        value={value}
        onChange={(visible_trigger) => onPatch({ visible_trigger })}
        placeholder={VISIBLE_PLACEHOLDER}
        templates={CONDITION_TEMPLATES}
        minHeight="min-h-24"
      />
      {derived && value.trim() !== "" && (
        <p className="text-micro text-muted-foreground font-mono break-all">
          {t("exportAsHint")}{" "}
          {`triggers = { ${derived.visible_trigger_key} = { ... } }`}
        </p>
      )}
    </div>
  );
}
