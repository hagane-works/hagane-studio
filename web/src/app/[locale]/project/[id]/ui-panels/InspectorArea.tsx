"use client";

import { useTranslations } from "use-intl";

import { TooltipProvider } from "@/components/ui/tooltip";
import type { UiPanel, UiPanelControl } from "@/types";
import {
  controlNameSlug,
  getControlAtPath,
  removeControlAtPath,
  uniqueControlName,
  updateControlAtPath,
  useUiPanelSelection,
} from "@/components/workbench/ui-panels/selectionStore";
import { LayoutSection } from "@/components/workbench/ui-panels/inspector/LayoutSection";
import { VisibleTriggerSection } from "@/components/workbench/ui-panels/inspector/VisibleTriggerSection";
import { BindingSection } from "@/components/workbench/ui-panels/inspector/BindingSection";
import { TextSection } from "@/components/workbench/ui-panels/inspector/TextSection";
import { ClickEnabledSection } from "@/components/workbench/ui-panels/inspector/ClickEnabledSection";
import { ClickEffectSection } from "@/components/workbench/ui-panels/inspector/ClickEffectSection";
import { ButtonTextSection } from "@/components/workbench/ui-panels/inspector/ButtonTextSection";

export interface InspectorAreaProps {
  projectId: number;
  panel: UiPanel;
  controls: UiPanelControl[];
  onControlsChange: (next: UiPanelControl[]) => void;
}

export function InspectorArea({
  projectId,
  panel,
  controls,
  onControlsChange,
}: InspectorAreaProps) {
  const t = useTranslations("uiPanelsInspector");
  const selectedPath = useUiPanelSelection((s) => s.selectedPath);
  const setSelectedPath = useUiPanelSelection((s) => s.setSelectedPath);
  const ctrl = getControlAtPath(controls, selectedPath);

  if (!ctrl || !selectedPath) {
    return (
      <aside className="w-[372px] shrink-0 bg-card p-4">
        <p className="text-xs text-muted-foreground">{t("empty")}</p>
      </aside>
    );
  }

  const derived = panel.derived?.controls?.[ctrl.name];
  const patch = (partial: Partial<UiPanelControl>) =>
    onControlsChange(updateControlAtPath(controls, selectedPath, partial));

  const finalizeName = (raw: string) => {
    if (!controlNameSlug(raw)) return ctrl.name;
    return uniqueControlName(controls, raw, selectedPath);
  };

  return (
    <TooltipProvider>
      <aside className="w-[372px] shrink-0 bg-card overflow-y-auto">
        <div className="p-4 border-b border-border space-y-1.5">
          <div className="flex items-center gap-2 flex-wrap">
            <span className="text-sm font-mono font-semibold break-all">
              {derived?.control_name ?? ctrl.name}
            </span>
            <span className="text-micro uppercase tracking-widest font-semibold px-2 py-0.5 rounded-full border border-accent/30 bg-accent/15 text-accent">
              {ctrl.kind}
            </span>
          </div>
          <p className="text-xs text-muted-foreground">{t("subtitle")}</p>
        </div>

        <LayoutSection
          ctrl={ctrl}
          onPatch={patch}
          finalizeName={finalizeName}
        />

        <VisibleTriggerSection ctrl={ctrl} derived={derived} onPatch={patch} />

        {(ctrl.kind === "icon" ||
          ctrl.kind === "progressbar" ||
          ctrl.kind === "button") && (
          <BindingSection ctrl={ctrl} derived={derived} onPatch={patch} />
        )}
        {ctrl.kind === "text" && (
          <TextSection
            entityKey={`ui-panel:${panel.id}:control:${selectedPath}`}
            ctrl={ctrl}
            derived={derived}
            onPatch={patch}
          />
        )}
        {ctrl.kind === "button" && (
          <>
            <ClickEnabledSection
              ctrl={ctrl}
              derived={derived}
              onPatch={patch}
            />
            <ClickEffectSection
              projectId={projectId}
              countryTag={panel.country_tag ?? ""}
              ctrl={ctrl}
              derived={derived}
              onPatch={patch}
            />
            <ButtonTextSection
              entityKey={`ui-panel:${panel.id}:control:${selectedPath}`}
              ctrl={ctrl}
              derived={derived}
              onPatch={patch}
            />
          </>
        )}

        <div className="p-4 space-y-2">
          <button
            type="button"
            onClick={() => {
              onControlsChange(removeControlAtPath(controls, selectedPath));
              setSelectedPath(null);
            }}
            className="h-8 px-3 text-xs rounded-md border border-destructive/40 text-destructive hover:bg-destructive/10 ui-interactive"
          >
            {t("removeControl")}
          </button>
        </div>
      </aside>
    </TooltipProvider>
  );
}
