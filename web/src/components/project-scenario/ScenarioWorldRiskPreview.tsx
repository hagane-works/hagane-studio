"use client";

import { useTranslations } from "use-intl";

export type ScenarioWorldRiskPreviewModel =
  | Readonly<{
      state: "inherit_base_world";
    }>
  | Readonly<{
      state: "scenario_world_source_unverified";
    }>;

export interface ScenarioWorldRiskPreviewProps {
  preview: ScenarioWorldRiskPreviewModel;
}

export function ScenarioWorldRiskPreview({
  preview,
}: ScenarioWorldRiskPreviewProps) {
  const t = useTranslations("ScenarioSettings");

  if (preview.state === "inherit_base_world") {
    return (
      <section
        className="space-y-2 rounded-lg border border-border bg-muted/50 p-3"
        aria-labelledby="scenario-world-risk-preview-title"
        data-state={preview.state}
      >
        <h3
          id="scenario-world-risk-preview-title"
          className="text-sm font-medium"
        >
          {t("riskPreview.title")}
        </h3>
        <p className="text-xs text-muted-foreground">
          {t("riskPreview.inherit")}
        </p>
      </section>
    );
  }

  return (
    <section
      className="space-y-2 rounded-lg border border-border bg-muted/50 p-3"
      aria-labelledby="scenario-world-risk-preview-title"
      data-state={preview.state}
    >
      <h3
        id="scenario-world-risk-preview-title"
        className="text-sm font-medium"
      >
        {t("riskPreview.title")}
      </h3>
      <p className="text-xs text-muted-foreground">
        {t("riskPreview.unverified")}
      </p>
    </section>
  );
}
