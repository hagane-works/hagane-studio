"use client";

import { useState } from "react";
import useSWR from "swr";
import { useLocale, useTranslations } from "use-intl";
import { statesApi } from "@/lib/api/scenario";
import type { State } from "@/types";
import { stateForProvince } from "../project/[id]/deployments/deploymentLocation";
import type { GuiArmyDivision } from "./ArmyOverviewLayer";
import { BUTTON, FIELD, GamePanel, NUMBER, PanelSave } from "./GamePanel";
import { fileUndo, recordChange } from "./changeLog";

export interface DivisionDraft {
  name?: string;
  template?: string;

  experience?: string;
  equipment?: string;

  location?: string;

  state?: number | null;
  officer?: string;
}

type Changes = {
  name?: string;
  template?: string;
  experience?: number;
  equipment?: number;
  location?: number;
  officer?: string;
};

function share(text: string): number | null {
  const value = Number(text);
  if (text.trim() === "" || !Number.isFinite(value) || value < 0 || value > 100)
    return null;
  return Math.round(value * 10) / 1000;
}

export function divisionChanges(
  division: GuiArmyDivision,
  draft: DivisionDraft,
): Changes | null {
  const out: Changes = {};
  if (draft.name !== undefined) {
    const name = draft.name.trim();
    if (!name || /["{}]/.test(name)) return null;
    if (name !== division.name) out.name = name;
  }
  if (draft.template !== undefined && draft.template !== division.template)
    out.template = draft.template;
  for (const key of ["experience", "equipment"] as const) {
    const text = draft[key];
    if (text === undefined) continue;
    const value = share(text);
    if (value === null) return null;
    if (value !== division[key]) out[key] = value;
  }
  if (draft.location !== undefined) {
    const province = Number(draft.location);
    if (!Number.isInteger(province) || province <= 0) return null;
    if (province !== (division.location ?? 0)) out.location = province;
  }
  if (draft.officer !== undefined) {
    const officer = draft.officer.trim();
    if (/["{}]/.test(officer)) return null;
    if (officer !== (division.officer ?? "")) out.officer = officer;
  }
  return out;
}

function refusal(detail: string, t: (key: string) => string): string {
  return /Recruit & Deploy/.test(detail)
    ? t("division.addedElsewhere")
    : detail;
}

const percent = (value: number) => String(Math.round(value * 1000) / 10);

export function DivisionEditPanel({
  projectId,
  tag,
  division,
  templates,
  onSaved,
  onRemoved,
  onAddDivision,
  onClose,
}: {
  projectId: number;

  tag?: string | null;
  division: GuiArmyDivision;
  templates: string[];
  onSaved: (message: string) => void;

  onRemoved?: (message: string) => void;

  onAddDivision?: () => void;
  onClose: () => void;
}) {
  const t = useTranslations("gameViewMilitary");
  const locale = useLocale();
  const kind = t("division.kind", { n: division.ordinal + 1 });
  const [draft, setDraft] = useState<DivisionDraft>({});
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const states = useSWR<State[]>(
    tag ? ["states/by-owner", projectId, tag] : null,
    () => statesApi.byOwner(projectId, tag as string),
  );
  const owned = Array.isArray(states.data) ? states.data : [];
  const location =
    draft.location ?? (division.location ? String(division.location) : "");
  const stateId =
    draft.state !== undefined
      ? draft.state
      : stateForProvince(Number(location) || null, owned);
  const provinces =
    owned.find((state) => state.state_id === stateId)?.provinces ?? [];

  const changes = divisionChanges(division, draft);
  const changed = !!changes && Object.keys(changes).length > 0;
  const send = async (body: Record<string, unknown>) => {
    const response = await fetch(
      `/api/go/projects/${projectId}/gui-render/division-save`,
      {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          oob: division.oob,
          ordinal: division.ordinal,
          ...body,
        }),
      },
    );
    const out = await response.json().catch(() => ({}));
    if (!response.ok)
      throw new Error(
        refusal(
          (out as { detail?: string }).detail || `HTTP ${response.status}`,
          t,
        ),
      );
    return out;
  };
  const remove = async () => {
    setBusy(true);
    setError(null);
    try {
      const out = await send({ remove: true });
      recordChange({
        projectId,
        title: t("division.removedLog", { name: division.name }),
        kind,
        undo: fileUndo(projectId, out),
      });
      (onRemoved ?? onSaved)(t("division.removed", { name: division.name }));
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setBusy(false);
    }
  };
  const save = async () => {
    if (!changes || !changed) return;
    setBusy(true);
    setError(null);
    try {
      const out = await send(changes);
      recordChange({
        projectId,
        title: changes.name ?? division.name,
        kind,
        undo: fileUndo(projectId, out),
      });
      setDraft({});
      onSaved(t("division.saved", { name: changes.name ?? division.name }));
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setBusy(false);
    }
  };

  return (
    <GamePanel
      title={division.name}
      kind={kind}
      detail={`history/units/${division.oob}.txt`}
      wide={false}
      onClose={onClose}
    >
      <div className="mt-3 space-y-2">
        <label className="block text-xs">
          {t("common.name")}
          <input
            aria-label="name"
            className={FIELD}
            value={draft.name ?? division.name}
            onChange={(e) => setDraft({ ...draft, name: e.target.value })}
          />
        </label>
        <label className="block text-xs">
          {t("division.template")}
          <select
            aria-label="template"
            className={FIELD}
            value={draft.template ?? division.template}
            onChange={(e) => setDraft({ ...draft, template: e.target.value })}
          >
            {!templates.includes(division.template) && (
              <option value={division.template}>{division.template}</option>
            )}
            {templates.map((template) => (
              <option key={template} value={template}>
                {template}
              </option>
            ))}
          </select>
        </label>
        <div className="grid grid-cols-2 gap-2">
          <label className="block text-xs">
            {t("division.experience")}
            <input
              aria-label="experience"
              className={NUMBER}
              inputMode="decimal"
              value={draft.experience ?? percent(division.experience)}
              onChange={(e) =>
                setDraft({ ...draft, experience: e.target.value })
              }
            />
          </label>
          <label className="block text-xs">
            {t("division.equipment")}
            <input
              aria-label="equipment"
              className={NUMBER}
              inputMode="decimal"
              value={draft.equipment ?? percent(division.equipment)}
              onChange={(e) =>
                setDraft({ ...draft, equipment: e.target.value })
              }
            />
          </label>
        </div>
        <p className="text-micro text-muted-foreground">
          {t("division.rankNote")}
        </p>
        <div className="text-xs">
          {t("division.location")}
          {owned.length > 0 ? (
            <div className="grid grid-cols-2 gap-2">
              <select
                aria-label="state"
                className={FIELD}
                value={stateId ?? ""}
                onChange={(e) =>
                  setDraft({
                    ...draft,
                    state:
                      e.target.value === "" ? null : Number(e.target.value),
                    location: "",
                  })
                }
              >
                <option value="">{t("division.state")}</option>
                {owned.map((state) => (
                  <option key={state.state_id} value={state.state_id}>
                    {state.state_id} ·{" "}
                    {state.display_name?.[
                      locale.startsWith("zh") ? "zh" : "en"
                    ] ||
                      state.display_name?.en ||
                      state.name ||
                      state.state_category}
                  </option>
                ))}
              </select>
              <select
                aria-label="province"
                className={FIELD}
                value={location}
                onChange={(e) =>
                  setDraft({ ...draft, location: e.target.value })
                }
              >
                <option value="">{t("division.province")}</option>
                {location !== "" && !provinces.includes(Number(location)) && (
                  <option value={location}>
                    {t("division.provinceElsewhere", { id: location })}
                  </option>
                )}
                {provinces.map((province) => (
                  <option key={province} value={province}>
                    {province}
                  </option>
                ))}
              </select>
            </div>
          ) : (
            <input
              aria-label="province"
              className={NUMBER}
              inputMode="numeric"
              value={location}
              onChange={(e) => setDraft({ ...draft, location: e.target.value })}
            />
          )}
        </div>
        <label className="block text-xs">
          {t("division.officer")}
          <input
            aria-label="officer"
            className={FIELD}
            placeholder={t("division.officerNone")}
            value={draft.officer ?? division.officer ?? ""}
            onChange={(e) => setDraft({ ...draft, officer: e.target.value })}
          />
        </label>
      </div>
      <PanelSave
        onSave={save}
        canSave={changed}
        busy={busy}
        onRevert={changed ? () => setDraft({}) : undefined}
        problem={changes === null ? t("division.problem") : null}
        note={t("division.note")}
        error={error}
      />
      <section className="mt-4 space-y-2">
        <button
          type="button"
          disabled={busy}
          className={BUTTON}
          onClick={() => void remove()}
        >
          {t("division.remove")}
        </button>
        {onAddDivision && (
          <div>
            <button type="button" className={BUTTON} onClick={onAddDivision}>
              {t("division.add")}
            </button>
            <p className="mt-1 text-micro text-muted-foreground">
              {t("division.addHint")}
            </p>
          </div>
        )}
      </section>
    </GamePanel>
  );
}
