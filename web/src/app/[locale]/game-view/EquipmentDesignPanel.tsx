"use client";

import { useState } from "react";
import { useTranslations } from "use-intl";
import { historyFilesApi } from "@/lib/api/scenario";
import { BUTTON, FIELD, GamePanel, PRIMARY } from "./GamePanel";
import { pick, recordChange, rowUndo } from "./changeLog";
import {
  designDraftOf,
  designProblem,
  designSaveBody,
  draftChanged,
  nameTaken,
  type DesignDraft,
} from "./designEdit";
import { useEquipmentDesigner } from "./EquipmentDesignerLayer";
import { useHistoryRow } from "./startStateEdit";

const SMALL =
  "w-16 rounded border border-input bg-background px-1 py-0.5 text-right text-sm tabular-nums";

async function postJSON(url: string, body: unknown) {
  const response = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });
  const out = await response.json().catch(() => ({}));
  if (!response.ok)
    throw new Error(
      (out as { detail?: string }).detail || `HTTP ${response.status}`,
    );
  return out;
}

export function EquipmentDesignPanel({
  projectId,
  tag,
  date,
  name,
  draft,
  focusSlot,
  onDraft,
  onOpenDesign,
  onSaved,
  onClose,
}: {
  projectId: number;
  tag: string;
  date?: string | null;

  name: string;
  draft: DesignDraft | null;

  focusSlot?: string | null;
  onDraft: (draft: DesignDraft | null) => void;
  onOpenDesign: (name: string) => void;

  onSaved: (message: string, next: string) => void;
  onClose: () => void;
}) {
  const tPanel = useTranslations("gameViewEdit.panel");
  const t = useTranslations("gameViewMilitary");
  const start = useEquipmentDesigner(projectId, tag, date, name, null, true);
  const row = useHistoryRow(projectId, tag, true);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [confirmDelete, setConfirmDelete] = useState(false);

  const design = start.data?.found ? start.data : undefined;
  const current = draft ?? (design ? designDraftOf(design) : null);
  const kind = t(design?.variant ? "design.kindVariant" : "design.kindNew", {
    type: design ? `${design.type_label} · ${design.type}` : name,
  });
  const changed = draftChanged(draft, design);
  const problem = current && design ? designProblem(current, design) : null;
  const problemText = !problem
    ? null
    : problem.key === "slotsMissing"
      ? t("design.problem.slotsMissing", {
          slots: problem.slots.join(t("design.listSeparator")),
        })
      : t(`design.problem.${problem.key}`);
  const canSave = !!row.data && !!design && !!current && !problem && !busy;
  const takenNew = !!design && !!current && nameTaken(current, design, "new");
  const takenUpdate =
    !!design && !!current && nameTaken(current, design, "update");

  const run = async (action: () => Promise<[string, string] | null>) => {
    setBusy(true);
    setError(null);
    try {
      const done = await action();
      if (done) onSaved(done[0], done[1]);
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setBusy(false);
    }
  };
  const save = (mode: "new" | "update" | "delete") =>
    run(async () => {
      if (!design || !current || (mode !== "delete" && problem)) return null;
      const before = row.data;
      await postJSON(
        `/api/go/projects/${projectId}/gui-render/equipment-design-save`,
        designSaveBody(tag, design, current, mode),
      );
      const after = await row.mutate();

      const id = before?.id;
      recordChange({
        projectId,
        title:
          mode === "delete"
            ? t("log.delete", { name: design.name })
            : mode === "new"
              ? t("log.create", { name: current.name.trim() })
              : current.name.trim(),
        kind,
        undo:
          mode !== "new" && before && after && id !== undefined
            ? rowUndo({
                before: pick(before, ["raw_script"]),
                after: pick(after, ["raw_script"]),
                load: () => historyFilesApi.get(projectId, id),
                save: (back) => historyFilesApi.update(projectId, id, back),
              })
            : undefined,
      });
      if (mode === "delete")
        return [t("design.deleted", { name: design.name }), design.type];
      const saved = current.name.trim();
      return [
        mode === "new"
          ? t("design.savedNew", { name: saved })
          : t("design.saved", { name: saved }),
        saved,
      ];
    });

  const set = (next: Partial<DesignDraft>) => {
    if (current) onDraft({ ...current, ...next });
  };

  return (
    <GamePanel
      title={current?.name || name}
      kind={kind}
      source={row.isLoading ? undefined : row.data ? "项目" : "原版"}
      onClose={onClose}
    >
      {!row.isLoading && !row.data && (
        <p className="mt-2 text-micro text-muted-foreground">
          {tPanel("notInProjectHistory")}
        </p>
      )}
      {start.isLoading && (
        <p className="mt-3 text-xs text-muted-foreground">
          {t("common.loading")}
        </p>
      )}
      {design && current && (
        <div className="mt-3 space-y-3">
          <label className="block text-xs">
            {t("common.name")}
            <input
              className={FIELD}
              value={current.name}
              onChange={(e) => set({ name: e.target.value })}
            />
          </label>
          {!!design.designs?.length && (
            <label className="block text-xs">
              {t("design.openExisting")}
              <select
                className={FIELD}
                value=""
                onChange={(e) => e.target.value && onOpenDesign(e.target.value)}
              >
                <option value="">{t("design.choose")}</option>
                {design.designs.map((other) => (
                  <option key={other.name} value={other.name}>
                    {other.name} · {other.type_label}
                  </option>
                ))}
              </select>
            </label>
          )}
          <div>
            <div className="text-xs font-medium">{t("design.modules")}</div>
            <div className="mt-1 space-y-1.5">
              {design.slots.map((slot) => (
                <label
                  key={slot.slot}
                  className={`block rounded px-1 text-micro ${focusSlot === slot.slot ? "ring-1 ring-accent" : ""}`}
                >
                  <span className="text-muted-foreground">
                    {slot.slot_label || slot.slot}
                    {slot.required ? " *" : ""}
                  </span>
                  <select
                    aria-label={t("design.module", { slot: slot.slot })}
                    className={FIELD}
                    value={current.modules[slot.slot] ?? ""}
                    onChange={(e) =>
                      set({
                        modules: {
                          ...current.modules,
                          [slot.slot]: e.target.value,
                        },
                      })
                    }
                  >
                    <option value="">{t("design.empty")}</option>
                    {(slot.allowed ?? []).map((option) => (
                      <option key={option.name} value={option.name}>
                        {option.label}
                      </option>
                    ))}
                  </select>
                </label>
              ))}
            </div>
          </div>
          {design.upgrades.length > 0 && (
            <div>
              <div className="text-xs font-medium">{t("design.upgrades")}</div>
              <div className="mt-1 space-y-1">
                {design.upgrades.map((upgrade) => (
                  <label
                    key={upgrade.name}
                    className="flex items-center justify-between gap-2 text-micro"
                  >
                    <span>{upgrade.label}</span>
                    <input
                      type="number"
                      aria-label={t("design.upgrade", { name: upgrade.name })}
                      className={SMALL}
                      min={0}
                      max={upgrade.max ?? 10}
                      value={current.upgrades[upgrade.name] ?? 0}
                      onChange={(e) => {
                        const level = Math.max(
                          0,
                          Math.min(
                            upgrade.max ?? 10,
                            Math.round(Number(e.target.value) || 0),
                          ),
                        );
                        set({
                          upgrades: {
                            ...current.upgrades,
                            [upgrade.name]: level,
                          },
                        });
                      }}
                    />
                  </label>
                ))}
              </div>
            </div>
          )}
          {problemText && (
            <p className="text-micro text-destructive">{problemText}</p>
          )}
          {!problem && takenNew && (
            <p className="text-micro text-muted-foreground">
              {t(
                design.variant ? "design.nameTakenVariant" : "design.nameTaken",
                { name: current.name.trim() },
              )}
            </p>
          )}
          <div className="flex flex-wrap items-center gap-2">
            {changed && (
              <button
                type="button"
                className={BUTTON}
                disabled={busy}
                onClick={() => onDraft(null)}
              >
                {tPanel("revert")}
              </button>
            )}
            <button
              type="button"
              className={PRIMARY}
              disabled={!canSave || takenNew}
              onClick={() => save("new")}
            >
              {t("design.saveNew")}
            </button>
            {design.variant && (
              <button
                type="button"
                className={BUTTON}
                disabled={!canSave || !changed || takenUpdate}
                onClick={() => save("update")}
              >
                {t("design.saveUpdate")}
              </button>
            )}
            {design.variant && (
              <button
                type="button"
                className={BUTTON}
                disabled={!row.data || busy}
                onClick={() =>
                  confirmDelete ? save("delete") : setConfirmDelete(true)
                }
              >
                {confirmDelete ? t("design.confirmDelete") : t("design.delete")}
              </button>
            )}
          </div>
          <p className="text-micro text-muted-foreground">{t("design.note")}</p>
          {error && <p className="text-micro text-destructive">{error}</p>}
        </div>
      )}
    </GamePanel>
  );
}
