"use client";

import { useState } from "react";
import { useTranslations } from "use-intl";
import { useConfirm } from "@/components/shared/ConfirmDialogProvider";
import type {
  DecisionEdits,
  DecisionPick,
  GuiDecision,
} from "./DecisionsLayer";
import { decisionIconSprite } from "./decisionEdit";
import { FIELD, NUMBER, PanelSave } from "./GamePanel";
import { fileUndo, recordChange } from "./changeLog";

export type DecisionOverrideDraft = {
  cost: string;
  icon: string;
  days_remove: string;
  days_re_enable: string;
  fire_only_once: boolean;
};

export function decisionOverrideDraft(
  shown: GuiDecision,
): DecisionOverrideDraft {
  return {
    cost: shown.cost === undefined ? "" : String(shown.cost),
    icon: shown.icon_token ?? "",
    days_remove:
      shown.days_remove === undefined ? "" : String(shown.days_remove),
    days_re_enable:
      shown.days_re_enable === undefined ? "" : String(shown.days_re_enable),
    fire_only_once: !!shown.fire_only_once,
  };
}

export type DecisionOverridePatch = {
  cost?: number;
  icon?: string;
  days_remove?: number;
  days_re_enable?: number;
  fire_only_once?: boolean;
};

export type DecisionOverrideProblem = "cost" | "icon" | "days";

export function decisionOverridePatch(
  start: DecisionOverrideDraft,
  draft: DecisionOverrideDraft,
): { patch: DecisionOverridePatch } | { problem: DecisionOverrideProblem } {
  const patch: DecisionOverridePatch = {};
  if (draft.cost !== start.cost) {
    const cost = Number(draft.cost);
    if (draft.cost.trim() === "" || !Number.isFinite(cost) || cost < 0)
      return { problem: "cost" };
    patch.cost = cost;
  }
  if (draft.icon !== start.icon) {
    const icon = draft.icon.trim();
    if (!/^[A-Za-z0-9_.-]+$/.test(icon)) return { problem: "icon" };
    patch.icon = icon;
  }
  for (const key of ["days_remove", "days_re_enable"] as const) {
    if (draft[key] === start[key]) continue;
    const days = Number(draft[key]);
    if (draft[key].trim() === "" || !Number.isInteger(days) || days < 0)
      return { problem: "days" };
    patch[key] = days;
  }
  if (draft.fire_only_once !== start.fire_only_once)
    patch.fire_only_once = draft.fire_only_once;
  return { patch };
}

export function DecisionOverride({
  projectId,
  pick,
  shown,
  onSaved,
}: {
  projectId: number;
  pick: Extract<DecisionPick, { kind: "decision" }>;
  shown: GuiDecision;
  onSaved: (pick: DecisionPick, change: DecisionEdits[string]) => void;
}) {
  const t = useTranslations("gameViewContent.override");
  const tContent = useTranslations("gameViewContent");
  const confirm = useConfirm();
  const start = decisionOverrideDraft(shown);
  const [draft, setDraft] = useState(start);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [done, setDone] = useState<string | null>(null);
  const checked = decisionOverridePatch(start, draft);
  const changed = "patch" in checked && Object.keys(checked.patch).length > 0;
  const file = shown.file ?? t("fileFallback");

  const save = async () => {
    if (!("patch" in checked) || !changed) return;
    if (
      shown.origin === "vanilla" &&
      !(await confirm({
        title: t("confirmTitle"),
        body: t("confirmBody", { file }),
        confirmLabel: t("confirmLabel"),
      }))
    ) {
      return;
    }
    setBusy(true);
    setError(null);
    try {
      const response = await fetch(
        `/api/go/projects/${projectId}/gui-render/decision-save`,
        {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({
            category: pick.category,
            name: pick.name,
            ...checked.patch,
          }),
        },
      );
      const answer = (await response.json().catch(() => ({}))) as {
        detail?: string;
        file?: string;
        imported?: boolean;
      };
      if (!response.ok)
        throw new Error(answer.detail || `HTTP ${response.status}`);
      recordChange({
        projectId,
        title: shown.label || pick.name,
        kind: tContent("decision.kind", { id: pick.name }),
        undo: fileUndo(projectId, answer),
      });
      setDone(
        t(answer.imported ? "importedDone" : "done", {
          file: answer.file ?? file,
        }),
      );
      onSaved(pick, {
        cost: checked.patch.cost,
        icon: checked.patch.icon
          ? decisionIconSprite(checked.patch.icon)
          : undefined,
      });
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setBusy(false);
    }
  };

  const number = (
    key: "cost" | "days_remove" | "days_re_enable",
    label: string,
  ) => (
    <label className="block">
      <span className="text-xs">{label}</span>
      <input
        aria-label={label}
        inputMode="numeric"
        value={draft[key]}
        disabled={busy}
        placeholder={t("placeholder")}
        onChange={(e) => setDraft({ ...draft, [key]: e.target.value })}
        className={NUMBER}
      />
    </label>
  );

  return (
    <section className="mt-4 space-y-3 border-t border-border pt-3">
      <div className="text-micro font-medium text-muted-foreground">
        {t("heading")}
      </div>
      <p className="text-xs text-muted-foreground">
        {shown.origin === "vanilla"
          ? t("vanillaNote", { file })
          : t("modNote", { file })}
      </p>
      <div className="grid grid-cols-2 gap-2">
        {number("cost", t("cost"))}
        <label className="block">
          <span className="text-xs">{t("icon")}</span>
          <input
            aria-label={t("icon")}
            value={draft.icon}
            disabled={busy}
            placeholder={t("placeholder")}
            onChange={(e) => setDraft({ ...draft, icon: e.target.value })}
            className={FIELD}
          />
        </label>
        {number("days_remove", t("daysRemove"))}
        {number("days_re_enable", t("daysReEnable"))}
      </div>
      <label className="flex items-center gap-2 text-xs">
        <input
          type="checkbox"
          checked={draft.fire_only_once}
          disabled={busy}
          onChange={(e) =>
            setDraft({ ...draft, fire_only_once: e.target.checked })
          }
        />
        {t("fireOnce")}
      </label>
      <PanelSave
        onSave={() => void save()}
        canSave={changed}
        busy={busy}
        problem={
          "problem" in checked ? t(`problem.${checked.problem}`) : undefined
        }
        onRevert={changed ? () => setDraft(start) : undefined}
        note={done ?? t("note")}
        error={error}
      />
    </section>
  );
}
