"use client";

import { useState } from "react";
import useSWR from "swr";
import { useTranslations } from "use-intl";
import { readGuiResponse } from "./guiResponse";
import { subUnitsApi } from "@/lib/api";
import type { SubUnit } from "@/types";
import { SubUnitEditor } from "@/app/[locale]/project/[id]/sub-units/SubUnitEditor";
import { GamePanel, NUMBER, PanelSave } from "./GamePanel";
import { fileUndo, pick, recordChange, rowUndo } from "./changeLog";
import { changedKeys, useWorkbenchProject } from "./workbenchProject";
import { withGameLanguage } from "@/lib/guiRenderApi";

export interface GuiSubUnitEdit {
  id: string;
  found: boolean;
  name?: string;
  file?: string;
  stat_order?: string[];
  stats?: Record<string, number>;
  need?: { equipment: string; name: string; amount: number }[];
}

export const SUB_UNIT_STATS: readonly string[] = [
  "combat_width",
  "max_strength",
  "max_organisation",
  "default_morale",
  "manpower",
  "training_time",
  "suppression",
  "weight",
  "supply_consumption",
];

export interface SubUnitDraft {
  stats?: Record<string, string>;
  need?: Record<string, string>;
}

type Changes = {
  stats?: Record<string, number>;
  need?: Record<string, number>;
};

export function subUnitChanges(
  unit: GuiSubUnitEdit,
  draft: SubUnitDraft,
): Changes | null {
  const out: Changes = {};
  for (const [key, text] of Object.entries(draft.stats ?? {})) {
    const value = Number(text);
    if (text.trim() === "" || !Number.isFinite(value) || value < 0) return null;
    if (value !== unit.stats?.[key]) out.stats = { ...out.stats, [key]: value };
  }
  for (const [equipment, text] of Object.entries(draft.need ?? {})) {
    const value = Number(text);
    if (text.trim() === "" || !Number.isInteger(value) || value < 0)
      return null;
    const had = unit.need?.find((item) => item.equipment === equipment)?.amount;
    if (value !== had) out.need = { ...out.need, [equipment]: value };
  }
  return out;
}

export function subUnitRows(
  rows: readonly SubUnit[] | undefined,
  id: string,
): { own: SubUnit | null; imported: boolean } {
  const want = id.toLowerCase();
  const same =
    rows?.filter((row) => row.sub_unit_id.toLowerCase() === want) ?? [];
  return {
    own: same.find((row) => row.source !== "imported") ?? null,
    imported: same.some((row) => row.source === "imported"),
  };
}

export function SubUnitEditPanel({
  projectId,
  id,
  onSaved,
  onClose,
}: {
  projectId: number;
  id: string;
  onSaved: (message: string) => void;
  onClose: () => void;
}) {
  const t = useTranslations("gameViewMilitary.subUnit");
  const tPanel = useTranslations("gameViewEdit.panel");
  const unit = useSWR<GuiSubUnitEdit>(
    ["game-view-sub-unit", projectId, id],
    async () => {
      const response = await fetch(
        withGameLanguage(
          `/api/go/projects/${projectId}/gui-render/sub-unit?${new URLSearchParams({ id })}`,
        ),
      );
      return readGuiResponse<GuiSubUnitEdit>(response);
    },
    { revalidateOnFocus: false },
  );
  const rows = useSWR(
    ["game-view-sub-unit-rows", projectId],
    () => subUnitsApi.list(projectId),
    {
      revalidateOnFocus: false,
    },
  );
  const { own: row, imported } = subUnitRows(rows.data, id);
  const projectReady = useWorkbenchProject(projectId);
  const data = unit.data;
  const title = data?.name ?? id;
  const kind = t("kind", { id });

  const save = async (update: Partial<SubUnit>) => {
    if (!row || Object.keys(update).length === 0) return;
    const rowId = row.id;
    const keys = changedKeys(row, update);
    const before = pick(row, keys);
    const saved = await subUnitsApi.update(projectId, rowId, update);
    recordChange({
      projectId,
      title,
      kind,
      workbenchHref: `/project/${projectId}/sub-units`,
      undo: keys.length
        ? rowUndo({
            before,
            after: pick(saved, keys),
            load: () => subUnitsApi.get(projectId, rowId),
            save: (back) => subUnitsApi.update(projectId, rowId, back),
          })
        : undefined,
    });
    await rows.mutate();
    await unit.mutate();
    onSaved(t("saved", { name: title }));
  };

  return (
    <GamePanel
      title={title}
      kind={kind}
      detail={data?.file}
      source={
        rows.isLoading
          ? undefined
          : row
            ? "项目"
            : imported
              ? "mod 文件"
              : data?.found
                ? "原版"
                : undefined
      }
      workbenchHref={row ? `/project/${projectId}/sub-units` : null}

      wide={row ? "workbench" : false}
      onClose={onClose}
    >
      {rows.isLoading ? (
        <p className="mt-4 text-micro text-muted-foreground">
          {t("loadingRows")}
        </p>
      ) : row ? (
        projectReady ? (
          <div className="-mx-4 mt-2">
            <SubUnitEditor key={row.id} subUnit={row} onSave={save} />
          </div>
        ) : (
          <p className="mt-4 text-micro text-muted-foreground">
            {tPanel("loadingProject")}
          </p>
        )
      ) : (
        <FileNumbers
          projectId={projectId}
          id={id}
          data={data}
          reload={() => unit.mutate()}
          onSaved={onSaved}
        />
      )}
    </GamePanel>
  );
}

function FileNumbers({
  projectId,
  id,
  data,
  reload,
  onSaved,
}: {
  projectId: number;
  id: string;
  data: GuiSubUnitEdit | undefined;
  reload: () => Promise<unknown>;
  onSaved: (message: string) => void;
}) {
  const t = useTranslations("gameViewMilitary.subUnit");
  const [draft, setDraft] = useState<SubUnitDraft>({});
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const changes = data?.found ? subUnitChanges(data, draft) : null;
  const changed = !!changes && Object.keys(changes).length > 0;
  const save = async () => {
    if (!data || !changes || !changed) return;
    setBusy(true);
    setError(null);
    try {
      const response = await fetch(
        `/api/go/projects/${projectId}/gui-render/sub-unit-save`,
        {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ id: data.id, ...changes }),
        },
      );
      const out = await response.json().catch(() => ({}));
      if (!response.ok)
        throw new Error(
          (out as { detail?: string }).detail || `HTTP ${response.status}`,
        );
      recordChange({
        projectId,
        title: data.name ?? data.id,
        kind: t("kind", { id }),
        undo: fileUndo(projectId, out),
      });
      setDraft({});
      await reload();
      onSaved(t("saved", { name: data.name ?? data.id }));
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setBusy(false);
    }
  };

  if (!data)
    return <div className="mt-3 h-40 animate-pulse rounded bg-muted/60" />;
  if (!data.found)
    return (
      <p className="mt-3 text-xs text-muted-foreground">{t("notFound")}</p>
    );
  return (
    <>
      <div className="mt-3 space-y-2">
        <div className="grid grid-cols-2 gap-2">
          {(data.stat_order ?? []).map((key) => (
            <label key={key} className="block text-xs">
              {SUB_UNIT_STATS.includes(key) ? t(`stat.${key}`) : key}
              <input
                aria-label={key}
                className={NUMBER}
                inputMode="decimal"
                value={draft.stats?.[key] ?? String(data.stats?.[key] ?? 0)}
                onChange={(e) =>
                  setDraft({
                    ...draft,
                    stats: { ...draft.stats, [key]: e.target.value },
                  })
                }
              />
            </label>
          ))}
        </div>
        {(data.need ?? []).length > 0 && (
          <div>
            <div className="text-xs text-muted-foreground">{t("need")}</div>
            {(data.need ?? []).map((item) => (
              <label key={item.equipment} className="mt-1 block text-xs">
                {item.name}
                <input
                  aria-label={`need ${item.equipment}`}
                  className={NUMBER}
                  inputMode="numeric"
                  value={draft.need?.[item.equipment] ?? String(item.amount)}
                  onChange={(e) =>
                    setDraft({
                      ...draft,
                      need: { ...draft.need, [item.equipment]: e.target.value },
                    })
                  }
                />
              </label>
            ))}
          </div>
        )}
        <p className="text-micro text-muted-foreground">{t("affectsAll")}</p>
      </div>
      <PanelSave
        onSave={save}
        canSave={changed}
        busy={busy}
        onRevert={changed ? () => setDraft({}) : undefined}
        problem={changes === null ? t("problem") : null}
        note={t("note")}
        error={error}
      />
    </>
  );
}
