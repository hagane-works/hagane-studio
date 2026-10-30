"use client";

import { useTranslations } from "use-intl";

import { ScriptBlockEditor } from "@/components/shared/editors/ScriptBlockEditor";
import { EFFECT_TEMPLATES } from "@/components/shared/editors/effectTemplates";
import { EventReferenceDropdown } from "@/components/workbench/EventReferenceDropdown";
import { IdeaReferenceDropdown } from "@/components/workbench/IdeaReferenceDropdown";
import { DecisionReferenceDropdown } from "@/components/workbench/DecisionReferenceDropdown";
import type { UiPanelControl, UiPanelDerivedControl } from "@/types";

const CLICK_EFFECT_PLACEHOLDER = `add_political_power = -25
add_to_variable = { EA_legitimacy = 10 }`;

export function ClickEffectSection({
  projectId,
  countryTag,
  ctrl,
  derived,
  onPatch,
}: {
  projectId: number;

  countryTag: string;
  ctrl: UiPanelControl;
  derived: UiPanelDerivedControl | undefined;
  onPatch: (partial: Partial<UiPanelControl>) => void;
}) {
  const t = useTranslations("uiPanelsInspector");
  const value = ctrl.click_effect ?? "";

  return (
    <div className="p-4 border-b border-border space-y-2">
      <ScriptBlockEditor
        label={t("clickEffectTitle")}
        labelTooltip={t("clickEffectTooltip")}
        value={value}
        onChange={(click_effect) => onPatch({ click_effect })}
        placeholder={CLICK_EFFECT_PLACEHOLDER}
        templates={EFFECT_TEMPLATES}
        minHeight="min-h-24"
      >
        {(insertCode) => (
          <>
            <EventReferenceDropdown
              projectId={projectId}
              effectType="country_event"
              buttonLabel={t("refTriggerEvent")}
              onSelect={insertCode}
            />
            <IdeaReferenceDropdown
              projectId={projectId}
              countryTag={countryTag}
              effectType="add_ideas"
              buttonLabel={t("refAddIdea")}
              onSelect={insertCode}
            />
            <DecisionReferenceDropdown
              projectId={projectId}
              effectType="activate_decision"
              buttonLabel={t("refActivateDecision")}
              onSelect={insertCode}
            />
          </>
        )}
      </ScriptBlockEditor>
      {derived && (
        <p className="text-micro text-muted-foreground font-mono break-all">
          {t("exportAsHint")} {`effects = { ${derived.click_key} = { ... } }`}
        </p>
      )}
    </div>
  );
}
