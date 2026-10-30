"use client";

import { useTranslations } from "use-intl";

import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import type { UiPanelControl, UiPanelDerivedControl } from "@/types";

export function BindingSection({
  ctrl,
  derived,
  onPatch,
}: {
  ctrl: UiPanelControl;
  derived: UiPanelDerivedControl | undefined;
  onPatch: (partial: Partial<UiPanelControl>) => void;
}) {
  const t = useTranslations("uiPanelsInspector");
  const frameVar = ctrl.frame_var ?? "";

  const exportName = derived?.control_name ?? ctrl.name;

  return (
    <div className="p-4 border-b border-border space-y-3">
      <span className="block text-micro uppercase tracking-widest font-semibold text-muted-foreground">
        {t("bindingTitle")}
      </span>
      <div className="space-y-1">
        <Label>{t("frameVarLabel")}</Label>
        <Input
          value={frameVar}
          placeholder="legitimacy_var"
          className="h-8 text-xs font-mono"
          onChange={(e) => onPatch({ frame_var: e.target.value })}
        />
        <p className="text-micro text-muted-foreground">{t("frameVarHint")}</p>
        {frameVar.trim() !== "" && (
          <p className="text-micro text-muted-foreground font-mono break-all">
            {t("exportAsHint")}{" "}
            {`properties = { ${exportName} = { frame = ${frameVar.trim()} } }`}
          </p>
        )}
      </div>
      {ctrl.kind === "icon" && (
        <div className="space-y-1">
          <Label>{t("imageExprLabel")}</Label>
          <Input
            value={ctrl.image_expr ?? ""}
            placeholder="[GetRulerGFX]"
            className="h-8 text-xs font-mono"
            onChange={(e) => onPatch({ image_expr: e.target.value })}
          />
          <p className="text-micro text-muted-foreground">
            {t("imageExprHint")}
          </p>
        </div>
      )}
    </div>
  );
}
