"use client";

import { useMemo, useState } from "react";
import useSWR from "swr";
import { useTranslations } from "use-intl";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select } from "@/components/ui/select";
import { statesApi } from "@/lib/api/scenario";
import { useRegisterDirty } from "@/lib/hooks/useUnsavedGuard";
import { tagFromOob } from "@/lib/oob/oobDefaults";
import type { DivisionDeployment, DivisionTemplate, State } from "@/types";
import { depToDraft, draftToPayload, type DepDraft } from "./deploymentForm";
import { isProvinceAvailable, stateForProvince } from "./deploymentLocation";
import { localizeDeploymentWarning } from "./deploymentWarnings";

export function DeploymentEditor({
  projectId,
  deployment,
  templates,
  onSave,
}: {
  projectId: number;

  deployment: DivisionDeployment;

  templates: readonly DivisionTemplate[];

  onSave: (payload: Partial<DivisionDeployment>) => Promise<void>;
}) {
  const t = useTranslations("divisionDeployment");

  const [draft, setDraft] = useState<DepDraft>(() => depToDraft(deployment));
  const [savedDraft, setSavedDraft] = useState<DepDraft>(draft);
  const [saving, setSaving] = useState(false);
  const [saveStatus, setSaveStatus] = useState<"idle" | "success" | "error">(
    "idle",
  );
  const [error, setError] = useState<string | null>(null);

  const draftTag = draft.country_tag;
  const { data: ownedStates = [], isLoading: ownedStatesLoading } = useSWR<
    State[]
  >(
    projectId && draftTag ? ["states/by-owner", projectId, draftTag] : null,
    () => statesApi.byOwner(projectId, draftTag),
  );

  const stateId =
    draft.state_id ?? stateForProvince(draft.location, ownedStates);

  const templateNames = useMemo(() => {
    const inOob = templates
      .filter((tp) => tp.oob_file === draft.oob_file)
      .map((tp) => tp.template_name);
    return inOob.length ? inOob : templates.map((tp) => tp.template_name);
  }, [templates, draft.oob_file]);

  const cascadeProvinces = useMemo(() => {
    if (stateId == null) return [];
    const st = ownedStates.find((s) => s.state_id === stateId);
    return st ? st.provinces || [] : [];
  }, [stateId, ownedStates]);

  const unavailableDraftProvince =
    draft.location != null &&
    !ownedStatesLoading &&
    !isProvinceAvailable(draft.location, ownedStates);

  const handleSave = async () => {
    setSaving(true);
    setSaveStatus("idle");
    setError(null);
    try {
      await onSave(draftToPayload(draft));
      setSavedDraft(draft);
      setSaveStatus("success");
      setTimeout(() => setSaveStatus("idle"), 2000);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
      setSaveStatus("error");
    } finally {
      setSaving(false);
    }
  };

  const isUnsaved = JSON.stringify(draft) !== JSON.stringify(savedDraft);
  useRegisterDirty("division-deployments", isUnsaved);

  const actionable = (deployment.warnings || []).filter((w) =>
    w.startsWith("actionable:"),
  );
  const info = (deployment.warnings || []).filter((w) => w.startsWith("info:"));

  const localizeWarning = (w: string): string =>
    localizeDeploymentWarning(w, t);

  return (
    <div className="flex-1 min-w-0 p-4 max-w-2xl space-y-4">
      <p className="text-xs text-muted-foreground">
        {t("landing")}{" "}
        <span className="font-mono">
          history/units/{draft.oob_file || "<oob>"}.txt
        </span>
      </p>

      <div className="grid grid-cols-2 gap-3">
        <div className="space-y-1">
          <Label>{t("oobFileLabel")}</Label>
          <Input
            className="font-mono text-xs"
            placeholder={t("oobFilePlaceholder")}
            value={draft.oob_file}
            onChange={(e) =>
              setDraft({
                ...draft,
                oob_file: e.target.value,
                country_tag: tagFromOob(e.target.value),
              })
            }
          />
        </div>
        <div className="space-y-1">
          <Label>{t("countryTagLabel")}</Label>
          <Input
            className="font-mono text-xs"
            placeholder="ROM"
            value={draft.country_tag}
            onChange={(e) =>
              setDraft({
                ...draft,
                country_tag: e.target.value.toUpperCase(),
                state_id: null,
                location: null,
              })
            }
          />
        </div>

        <div className="space-y-1 col-span-2">
          <Label>{t("templateLabel")}</Label>
          <Select
            className="px-2 text-xs font-mono"
            value={draft.division_template}
            onChange={(e) =>
              setDraft({ ...draft, division_template: e.target.value })
            }
          >
            <option value="">{t("templatePlaceholder")}</option>
            {draft.division_template &&
              !templateNames.includes(draft.division_template) && (
                <option value={draft.division_template}>
                  {draft.division_template} ({t("notDefined")})
                </option>
              )}
            {templateNames.map((n) => (
              <option key={n} value={n}>
                {n}
              </option>
            ))}
          </Select>
        </div>

        <div className="space-y-1">
          <Label>{t("stateLabel")}</Label>
          <Select
            className="px-2 text-xs"
            value={stateId ?? ""}
            onChange={(e) => {
              const sid = e.target.value === "" ? null : Number(e.target.value);
              setDraft({ ...draft, state_id: sid, location: null });
            }}
          >
            <option value="">
              {ownedStates.length ? t("statePlaceholder") : t("noOwnedStates")}
            </option>
            {ownedStates.map((s) => (
              <option key={s.state_id} value={s.state_id}>
                {s.state_id} · {s.name || s.state_category} (
                {t("stateProvinceCount", { count: (s.provinces || []).length })}
                )
              </option>
            ))}
          </Select>
        </div>
        <div className="space-y-1">
          <Label>{t("provinceLabel")}</Label>
          <Select
            className="px-2 text-xs font-mono"
            value={draft.location ?? ""}
            disabled={stateId == null}
            onChange={(e) =>
              setDraft({
                ...draft,
                location: e.target.value === "" ? null : Number(e.target.value),
              })
            }
          >
            <option value="">
              {stateId == null ? t("pickStateFirst") : t("provincePlaceholder")}
            </option>
            {draft.location != null &&
              !cascadeProvinces.includes(draft.location) && (
                <option value={draft.location}>
                  {draft.location} ({t("notInProjectStates")})
                </option>
              )}
            {cascadeProvinces.map((p) => (
              <option key={p} value={p}>
                {p}
              </option>
            ))}
          </Select>
          {unavailableDraftProvince && (
            <p className="text-xs text-warning">
              {t("provincePreservedWarning", {
                province: draft.location ?? "?",
                tag: draft.country_tag || "?",
              })}
            </p>
          )}
        </div>

        <div className="space-y-1">
          <Label>{t("divisionNameLabel")}</Label>
          <Input
            placeholder={t("divisionNamePlaceholder")}
            value={draft.division_name}
            onChange={(e) =>
              setDraft({ ...draft, division_name: e.target.value })
            }
          />
        </div>
        <div className="space-y-1">
          <Label>{t("countLabel")}</Label>
          <Input
            type="number"
            min={1}
            className="font-mono text-xs"
            value={draft.count}
            onChange={(e) => setDraft({ ...draft, count: e.target.value })}
          />
          <p className="text-micro text-muted-foreground">{t("countHint")}</p>
        </div>

        <div className="space-y-1">
          <Label>{t("expFactorLabel")}</Label>
          <Input
            type="number"
            step="0.1"
            min={0}
            max={1}
            className="font-mono text-xs"
            placeholder="0.0 - 1.0"
            value={draft.start_experience_factor}
            onChange={(e) =>
              setDraft({ ...draft, start_experience_factor: e.target.value })
            }
          />
        </div>
        <div className="space-y-1">
          <Label>{t("eqFactorLabel")}</Label>
          <Input
            type="number"
            step="0.1"
            min={0}
            className="font-mono text-xs"
            placeholder="0.0 - 1.0"
            value={draft.start_equipment_factor}
            onChange={(e) =>
              setDraft({ ...draft, start_equipment_factor: e.target.value })
            }
          />
        </div>
      </div>

      {actionable.length > 0 && (
        <div className="rounded-md border border-warning/30 bg-warning/10 px-3 py-2 space-y-1">
          {actionable.map((w, i) => (
            <p key={i} className="text-xs text-warning">
              ⚠ {localizeWarning(w)}
            </p>
          ))}
        </div>
      )}
      {info.length > 0 && (
        <div className="rounded-md border border-border bg-muted/40 px-3 py-2 space-y-1">
          {info.map((w, i) => (
            <p key={i} className="text-xs text-muted-foreground">
              {localizeWarning(w)}
            </p>
          ))}
        </div>
      )}

      {(saving || saveStatus !== "idle" || isUnsaved) && (
        <div
          className={`-mx-4 px-4 py-1.5 text-xs font-medium flex items-center gap-2 transition-[opacity,background-color,border-color] duration-300 ${
            saving
              ? "bg-muted/80 text-muted-foreground border-y border-border"
              : saveStatus === "success"
                ? "bg-success/10 text-success border-y border-success/20"
                : saveStatus === "error"
                  ? "bg-destructive/10 text-destructive border-y border-destructive/20"
                  : isUnsaved
                    ? "bg-warning/10 text-warning border-y border-warning/20"
                    : ""
          }`}
        >
          <span>
            {saving
              ? t("saving")
              : saveStatus === "success"
                ? t("saved")
                : saveStatus === "error"
                  ? (error ?? t("saveFailed"))
                  : isUnsaved
                    ? t("unsavedChanges")
                    : ""}
          </span>
        </div>
      )}
      <div className="flex items-center gap-2 pt-2 border-t border-border">
        <Button onClick={handleSave} disabled={saving} size="sm">
          {t("save")}
        </Button>
      </div>
    </div>
  );
}
