"use client";

import { useTranslations } from "use-intl";

import { ScriptBlockEditor } from "@/components/shared/editors/ScriptBlockEditor";
import { CONDITION_TEMPLATES } from "@/components/shared/editors/conditionTemplates";
import type { UiPanelControl, UiPanelDerivedControl } from "@/types";

const CLICK_ENABLED_PLACEHOLDER = `has_political_power > 24
check_variable = { EA_legitimacy < 100 }`;

export function ClickEnabledSection({
  ctrl,
  derived,
  onPatch,
}: {
  ctrl: UiPanelControl;
  derived: UiPanelDerivedControl | undefined;
  onPatch: (partial: Partial<UiPanelControl>) => void;
}) {
  const t = useTranslations("uiPanelsInspector");
  const value = ctrl.click_enabled_trigger ?? "";

  return (
    <div className="p-4 border-b border-border space-y-2">
      <ScriptBlockEditor
        label={t("clickEnabledTitle")}
        labelTooltip={t("clickEnabledTooltip")}
        value={value}
        onChange={(click_enabled_trigger) => onPatch({ click_enabled_trigger })}
        placeholder={CLICK_ENABLED_PLACEHOLDER}
        templates={CONDITION_TEMPLATES}
        minHeight="min-h-24"
      />
      {derived && value.trim() !== "" && (
        <p className="text-micro text-muted-foreground font-mono break-all">
          {t("exportAsHint")}{" "}
          {`triggers = { ${derived.click_enabled_key} = { ... } }`}
        </p>
      )}
      {value.trim() === "" && (
        <p className="text-micro text-muted-foreground">
          {t("clickEnabledEmptyHint")}
        </p>
      )}
    </div>
  );
}
