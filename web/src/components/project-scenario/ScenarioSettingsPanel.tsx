"use client";

import { useTranslations } from "use-intl";

import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";

export type ScenarioHistoryPolicy =
  "inherit_base_history" | "project_owns_starting_world";

export interface ScenarioSettingsValue {
  scenarioHistoryPolicy: ScenarioHistoryPolicy;
  endDate: string;
}

export interface ScenarioSettingsPanelProps {
  value: ScenarioSettingsValue;
  onChange: (nextValue: ScenarioSettingsValue) => void;

  isStandaloneProject: boolean;

  needsEndDate: boolean;
}

const WORLD_OPTIONS: readonly ScenarioHistoryPolicy[] = [
  "inherit_base_history",
  "project_owns_starting_world",
];

export function ScenarioSettingsPanel({
  value,
  onChange,
  isStandaloneProject,
  needsEndDate,
}: ScenarioSettingsPanelProps) {
  const t = useTranslations("ScenarioSettings");
  const endDateId = "scenario-end-date";
  const showEndDateNotice = needsEndDate && value.endDate.trim().length === 0;

  const update = (patch: Partial<ScenarioSettingsValue>) => {
    onChange({ ...value, ...patch });
  };

  return (
    <section
      className="ui-panel space-y-5 p-4"
      aria-labelledby="scenario-settings-title"
    >
      <div className="space-y-1">
        <h2
          id="scenario-settings-title"
          className="text-base font-semibold tracking-tight"
        >
          {t("title")}
        </h2>
        <p className="text-xs text-muted-foreground">{t("intro")}</p>
      </div>

      <fieldset className="space-y-2">
        <legend className="text-sm font-medium">{t("worldQuestion")}</legend>
        <div className="grid gap-2">
          {WORLD_OPTIONS.map((policy) => {
            const selected = value.scenarioHistoryPolicy === policy;
            const optionKey =
              policy === "inherit_base_history" ? "inherit" : "own";
            return (
              <label
                key={policy}
                className={
                  "ui-interactive flex cursor-pointer gap-3 rounded-lg border p-3 focus-within:ring-1 focus-within:ring-ring " +
                  (selected
                    ? "border-accent bg-accent/10"
                    : "border-border bg-card hover:bg-muted/50")
                }
              >
                <input
                  type="radio"
                  name="scenario-history-policy"
                  value={policy}
                  checked={selected}
                  onChange={() => update({ scenarioHistoryPolicy: policy })}
                  className="mt-0.5 size-4 accent-accent"
                />
                <span className="space-y-1">
                  <span className="block text-sm font-medium">
                    {t(`worldOptions.${optionKey}.title`)}
                  </span>
                  <span className="block text-xs text-muted-foreground">
                    {t(`worldOptions.${optionKey}.description`)}
                  </span>
                </span>
              </label>
            );
          })}
        </div>
      </fieldset>

      <div className="space-y-1.5">
        <Label htmlFor={endDateId}>{t("endDate.label")}</Label>
        <Input
          id={endDateId}
          value={value.endDate}
          onChange={(event) => update({ endDate: event.target.value })}
          placeholder="1976.12.31"
          autoComplete="off"
          className="font-mono text-xs tech-tabular"
        />
        <p className="text-xs text-muted-foreground">{t("endDate.hint")}</p>
      </div>

      <p className="rounded-md border border-border bg-muted/50 px-3 py-2 text-xs text-muted-foreground">
        {t(
          isStandaloneProject
            ? "timelineBaseline.vanilla"
            : "timelineBaseline.unregistered",
        )}
      </p>
      <p className="text-xs text-muted-foreground">
        {t("timelineBaseline.rawBookmarkCoverage")}
      </p>

      {showEndDateNotice ? (
        <p
          className="rounded-md border border-warning/40 bg-warning/10 px-3 py-2 text-xs text-warning"
          role="status"
        >
          {t("compatibilityNotice")}
        </p>
      ) : null}

      {value.scenarioHistoryPolicy === "project_owns_starting_world" ? (
        <aside className="space-y-1 rounded-md border border-border bg-muted/50 px-3 py-3">
          <h3 className="text-sm font-medium">{t("ownWorldSummary.title")}</h3>
          <p className="text-xs text-muted-foreground">
            {t("ownWorldSummary.description")}
          </p>
        </aside>
      ) : null}

      {!isStandaloneProject ? (
        <p className="rounded-md border border-border bg-muted/50 px-3 py-2 text-xs text-muted-foreground">
          {t("submodNotice")}
        </p>
      ) : null}
    </section>
  );
}
