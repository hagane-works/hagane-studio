"use client";

import { useState } from "react";
import useSWR from "swr";
import { useTranslations } from "use-intl";
import { guiRenderApi } from "@/lib/guiRenderApi";
import { intelligenceAgenciesApi } from "@/lib/api";
import type {
  IntelAgencyBranch,
  IntelAgencyUpgrade,
  IntelligenceAgencyRow,
} from "@/lib/api";
import { IntelUpgradeEditor } from "@/components/workbench/IntelUpgradeEditor";
import {
  BUTTON,
  FIELD,
  GamePanel,
  NUMBER,
  PanelSave,
  SCRIPT,
} from "./GamePanel";
import type { GuiIntelUpgrade } from "./IntelAgencyLayer";
import {
  fileUndo,
  recordChange,
  rowUndo,
  textBefore,
  textUndo,
  undoAll,
} from "./changeLog";

export interface IntelUpgradeDraft {
  name?: string;
  factories?: string;
  available?: string;
  levels?: string[];
}

type Changes = {
  name?: string;
  factories?: number;
  available?: string;
  levels?: string[];
};

export function intelUpgradeChanges(
  upgrade: GuiIntelUpgrade,
  draft: IntelUpgradeDraft,
): Changes | null {
  const out: Changes = {};
  if (draft.name !== undefined) {
    const name = draft.name.trim();
    if (!name) return null;
    if (name !== (upgrade.label ?? upgrade.id)) out.name = name;
  }
  if (draft.factories !== undefined) {
    const value = Number(draft.factories);
    if (
      draft.factories.trim() === "" ||
      !Number.isFinite(value) ||
      value < 0 ||
      value > 100
    )
      return null;
    if (value !== (upgrade.factories ?? 0)) out.factories = value;
  }
  if (
    draft.available !== undefined &&
    draft.available.trim() !== (upgrade.available_script ?? "")
  )
    out.available = draft.available.trim();
  if (draft.levels !== undefined) {
    const levels = draft.levels.map((level) => level.trim());
    if (!levels.length || levels.some((level) => !level)) return null;
    const before = upgrade.levels_script ?? [];
    if (
      levels.length !== before.length ||
      levels.some((level, at) => level !== before[at])
    )
      out.levels = levels;
  }
  return out;
}

const sameToken = (a: unknown, b: string) =>
  typeof a === "string" && a.trim().toLowerCase() === b.trim().toLowerCase();

export function upgradeRow(
  rows: readonly IntelligenceAgencyRow[] | undefined,
  token: string,
): { row: IntelligenceAgencyRow; upgrade: IntelAgencyUpgrade } | null {
  for (const row of rows ?? []) {
    if (row.source === "imported") continue;
    for (const branch of row.upgrade_branches ?? []) {
      const upgrade = (branch.upgrades ?? []).find((u) =>
        sameToken(u.token, token),
      );
      if (upgrade) return { row, upgrade };
    }
  }
  return null;
}

export function withUpgrade(
  branches: readonly IntelAgencyBranch[],
  token: string,
  change: (upgrade: IntelAgencyUpgrade) => IntelAgencyUpgrade,
): IntelAgencyBranch[] | null {
  let found = false;
  const out = branches.map((branch) => ({
    ...branch,
    upgrades: (branch.upgrades ?? []).map((upgrade) => {
      if (found || !sameToken(upgrade.token, token)) return upgrade;
      found = true;
      return change(upgrade);
    }),
  }));
  return found ? out : null;
}

async function loadAgency(
  projectId: number,
  id: number,
): Promise<IntelligenceAgencyRow | undefined> {
  return (await intelligenceAgenciesApi.list(projectId)).agencies.find(
    (row) => row.id === id,
  );
}

export function IntelUpgradeEditPanel({
  projectId,
  upgrade,
  onSaved,
  onClose,
}: {
  projectId: number;
  upgrade: GuiIntelUpgrade;
  onSaved: (message: string) => void;
  onClose: () => void;
}) {
  const t = useTranslations("gameViewResearch.upgrade");
  const tIntel = useTranslations("gameViewResearch.intel");
  const rows = useSWR(
    ["game-view-intel-agencies-db", projectId],
    async () => (await intelligenceAgenciesApi.list(projectId)).agencies,
    { revalidateOnFocus: false },
  );

  const [token, setToken] = useState(upgrade.id);
  const found = upgradeRow(rows.data, token);
  const workbenchHref = found ? `/project/${projectId}/intelligence` : null;
  const title = upgrade.label ?? upgrade.id;

  const saveUpgrade = async (next: IntelAgencyUpgrade) => {
    if (!found) return;
    const id = found.row.id;
    const now = await loadAgency(projectId, id);
    if (!now) throw new Error(t("agencyGone"));
    const branches = withUpgrade(now.upgrade_branches ?? [], token, (old) => ({
      ...old,
      token: next.token,
      picture: next.picture,
      levels: next.levels,
    }));
    if (!branches) throw new Error(t("upgradeGone"));
    const saved = await intelligenceAgenciesApi.update(projectId, id, {
      upgrade_branches: branches,
    });
    recordChange({
      projectId,
      title,
      kind: t("kind", { id: next.token || token }),
      workbenchHref,
      undo: rowUndo<IntelligenceAgencyRow>({
        before: { upgrade_branches: now.upgrade_branches },
        after: { upgrade_branches: saved.upgrade_branches },
        load: () => loadAgency(projectId, id),
        save: (back) => intelligenceAgenciesApi.update(projectId, id, back),
      }),
    });
    if (next.token.trim()) setToken(next.token);
    onSaved(t("saved", { name: title }));
    await rows.mutate();
  };

  return (
    <GamePanel
      title={title}
      kind={t("kind", { id: upgrade.id })}
      detail={upgrade.file}
      source={found ? "项目" : undefined}
      workbenchHref={workbenchHref}

      wide={found ? "workbench" : false}
      onClose={onClose}
    >
      {rows.isLoading ? (
        <p className="mt-4 text-micro text-muted-foreground">
          {tIntel("loading")}
        </p>
      ) : found ? (
        <>
          <div className="-mx-4 mt-2">
            <IntelUpgradeEditor
              key={found.row.id}
              upgrade={found.upgrade}
              onSave={saveUpgrade}
            />
          </div>
          <div className="mt-3 border-t border-border pt-2 text-micro text-muted-foreground">
            {t("workbenchLacks")}
          </div>
          <UpgradeGameFields
            projectId={projectId}
            upgrade={upgrade}
            withLevels={false}
            onSaved={onSaved}
          />
        </>
      ) : (
        <UpgradeGameFields
          projectId={projectId}
          upgrade={upgrade}
          withLevels
          onSaved={onSaved}
        />
      )}
    </GamePanel>
  );
}

function UpgradeGameFields({
  projectId,
  upgrade,
  withLevels,
  onSaved,
}: {
  projectId: number;
  upgrade: GuiIntelUpgrade;

  withLevels: boolean;
  onSaved: (message: string) => void;
}) {
  const t = useTranslations("gameViewResearch.upgrade");
  const tCommon = useTranslations("gameViewResearch.common");
  const [draft, setDraft] = useState<IntelUpgradeDraft>({});
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const changes = intelUpgradeChanges(upgrade, draft);
  const changed = !!changes && Object.keys(changes).length > 0;
  const levels = draft.levels ?? upgrade.levels_script ?? [];
  const setLevels = (next: string[]) => setDraft({ ...draft, levels: next });
  const save = async () => {
    if (!changes || !changed) return;
    setBusy(true);
    setError(null);
    try {
      const { name, ...definition } = changes;
      let answer: unknown = null;
      if (Object.keys(definition).length) {
        const response = await fetch(
          `/api/go/projects/${projectId}/gui-render/intel-upgrade-save`,
          {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({ id: upgrade.id, ...definition }),
          },
        );
        const out = await response.json().catch(() => ({}));
        if (!response.ok)
          throw new Error(
            (out as { detail?: string }).detail || `HTTP ${response.status}`,
          );
        answer = out;
      }

      const ways = answer !== undefined ? [fileUndo(projectId, answer)] : [];
      if (name !== undefined) {
        guiRenderApi.setProjectId(projectId);
        const before = await textBefore(upgrade.id);
        await guiRenderApi.setText(upgrade.id, name);
        ways.push(textUndo(upgrade.id, name, before));
      }
      recordChange({
        projectId,
        title: name ?? upgrade.label ?? upgrade.id,
        kind: t("kind", { id: upgrade.id }),
        undo: undoAll(...ways),
      });
      setDraft({});
      onSaved(t("saved", { name: name ?? upgrade.label ?? upgrade.id }));
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setBusy(false);
    }
  };

  return (
    <>
      <div className="mt-3 space-y-2">
        <label className="block text-xs">
          {tCommon("name")}
          <input
            aria-label="name"
            className={FIELD}
            value={draft.name ?? upgrade.label ?? upgrade.id}
            onChange={(e) => setDraft({ ...draft, name: e.target.value })}
          />
        </label>
        <label className="block text-xs">
          {t("factories")}
          <input
            aria-label="factories"
            className={NUMBER}
            inputMode="numeric"
            value={draft.factories ?? String(upgrade.factories ?? 0)}
            onChange={(e) => setDraft({ ...draft, factories: e.target.value })}
          />
        </label>
        <label className="block text-xs">
          {t("available")}
          <textarea
            aria-label="available"
            className={SCRIPT}
            rows={2}
            placeholder={tCommon("anytime")}
            value={draft.available ?? upgrade.available_script ?? ""}
            onChange={(e) => setDraft({ ...draft, available: e.target.value })}
          />
        </label>
        {withLevels && (
          <div className="text-xs">
            {t("levels")}
            {levels.map((level, at) => (
              <div key={at} className="mt-1">
                <div className="flex items-center justify-between text-micro text-muted-foreground">
                  {t("level", { n: at + 1 })}
                  {levels.length > 1 && (
                    <button
                      type="button"
                      className={BUTTON}
                      aria-label={`remove level ${at + 1}`}
                      onClick={() =>
                        setLevels(levels.filter((_, i) => i !== at))
                      }
                    >
                      {tCommon("remove")}
                    </button>
                  )}
                </div>
                <textarea
                  aria-label={`level ${at + 1}`}
                  className={SCRIPT}
                  rows={3}
                  value={level}
                  onChange={(e) =>
                    setLevels(
                      levels.map((item, i) =>
                        i === at ? e.target.value : item,
                      ),
                    )
                  }
                />
              </div>
            ))}
            <button
              type="button"
              className={`${BUTTON} mt-1`}
              onClick={() =>
                setLevels([...levels, levels[levels.length - 1] ?? ""])
              }
            >
              {t("addLevel")}
            </button>
          </div>
        )}
      </div>
      <PanelSave
        onSave={save}
        canSave={changed}
        busy={busy}
        onRevert={changed ? () => setDraft({}) : undefined}
        problem={
          changes === null
            ? withLevels
              ? t("problemLevels")
              : t("problem")
            : null
        }
        note={withLevels ? t("noteLevels") : t("note")}
        error={error}
      />
    </>
  );
}
