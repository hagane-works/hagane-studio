"use client";

import { useTranslations } from "use-intl";
import { Trash2 } from "lucide-react";
import {
  toWargoalWrite,
  WARGOAL_BLOCK_KEYS,
  WARGOAL_SCALAR_KEYS,
  type WargoalDraft,
} from "@/app/[locale]/project/[id]/wargoals/wargoalForm";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { useRegisterDirty } from "@/lib/hooks/useUnsavedGuard";

export type WargoalSaveStatus = "idle" | "saving" | "saved" | "error";

export function wargoalUnsaved(
  draft: WargoalDraft | null,
  savedDraft: WargoalDraft | null,
): boolean {
  return (
    draft != null &&
    (savedDraft == null ||
      JSON.stringify(toWargoalWrite(draft)) !==
        JSON.stringify(toWargoalWrite(savedDraft)))
  );
}

export function WarGoalEditor({
  draft,
  savedDraft,
  onChange,
  status,
  error,
  onSave,
  onDelete,
}: {
  draft: WargoalDraft;
  savedDraft: WargoalDraft | null;
  onChange: (draft: WargoalDraft) => void;
  status: WargoalSaveStatus;
  error: string;
  onSave: () => void;
  onDelete?: () => void;
}) {
  const t = useTranslations("wargoals");
  const isUnsaved = wargoalUnsaved(draft, savedDraft);
  useRegisterDirty("wargoals", isUnsaved);

  return (
    <div className="mx-auto max-w-3xl space-y-6 p-6">
      <div>
        <h1 className="text-lg font-semibold tracking-tight">
          {t("editorTitle")}
        </h1>
        <p className="mt-1 text-xs text-muted-foreground">
          {t("editorDescription")} <code>{"common/wargoals/*.txt"}</code>
        </p>
      </div>
      <section className="ui-panel space-y-4 p-6">
        <div>
          <h2 className="text-base font-semibold tracking-tight">
            {t("visualFieldsTitle")}
          </h2>
          <p className="mt-1 text-xs text-muted-foreground">
            {t("visualFieldsDescription")}
          </p>
        </div>
        <div className="grid grid-cols-2 gap-4">
          <div className="space-y-1">
            <Label htmlFor="wargoal-key">{t("idLabel")}</Label>
            <Input
              id="wargoal-key"
              className="font-mono text-xs"
              value={draft.wargoal_key}
              onChange={(event) =>
                onChange({ ...draft, wargoal_key: event.target.value })
              }
            />
          </div>
          <div className="space-y-1">
            <Label htmlFor="war-name">war_name</Label>
            <Input
              id="war-name"
              className="font-mono text-xs"
              placeholder="WAR_NAME"
              value={draft.war_name}
              onChange={(event) =>
                onChange({ ...draft, war_name: event.target.value })
              }
            />
          </div>
        </div>
        <div className="space-y-3">
          <h3 className="text-sm font-semibold">{t("scalarTitle")}</h3>
          <div className="grid grid-cols-2 gap-3">
            {WARGOAL_SCALAR_KEYS.map((key) => (
              <div key={key} className="space-y-1">
                <Label htmlFor={`scalar-${key}`} className="font-mono text-xs">
                  {key}
                </Label>
                <Input
                  id={`scalar-${key}`}
                  className="font-mono text-xs"
                  placeholder={key === "threat" ? "0.5" : "200"}
                  value={draft.safe_scalars[key] ?? ""}
                  onChange={(event) =>
                    onChange({
                      ...draft,
                      safe_scalars: {
                        ...draft.safe_scalars,
                        [key]: event.target.value,
                      },
                    })
                  }
                />
              </div>
            ))}
          </div>
        </div>
        <div className="space-y-4">
          <h3 className="text-sm font-semibold">{t("blocksTitle")}</h3>
          {WARGOAL_BLOCK_KEYS.map((key) => (
            <div key={key} className="space-y-1">
              <Label htmlFor={`block-${key}`} className="font-mono text-xs">
                {key}
              </Label>
              <Textarea
                id={`block-${key}`}
                className="min-h-24 font-mono text-xs"
                placeholder="always = yes"
                value={draft[key]}
                onChange={(event) =>
                  onChange({ ...draft, [key]: event.target.value })
                }
              />
            </div>
          ))}
        </div>
      </section>
      <details className="ui-panel p-6">
        <summary className="ui-interactive ui-focus-ring cursor-pointer rounded-md text-base font-semibold tracking-tight">
          {t("advancedTitle")}
        </summary>
        <p className="mt-2 text-xs text-muted-foreground">
          {t("advancedDescription")}
        </p>
        <Textarea
          className="mt-4 min-h-40 font-mono text-xs"
          value={draft.advanced_raw_script}
          onChange={(event) =>
            onChange({ ...draft, advanced_raw_script: event.target.value })
          }
          aria-label={t("advancedTitle")}
        />
      </details>
      <div className="space-y-1">
        <Label htmlFor="source-file">{t("sourceFileLabel")}</Label>
        <Input
          id="source-file"
          className="font-mono text-xs"
          placeholder="common/wargoals/my_wargoals.txt"
          value={draft.source_file}
          onChange={(event) =>
            onChange({ ...draft, source_file: event.target.value })
          }
        />
        <p className="text-xs text-muted-foreground">
          {t("sourceFileDescription")}
        </p>
      </div>
      {(status !== "idle" || isUnsaved || error) && (
        <div
          className={`flex items-center gap-2 border-y px-4 py-2 text-xs font-medium transition-[opacity,background-color,border-color] duration-300 ${status === "saving" ? "border-border bg-muted/80 text-muted-foreground" : status === "saved" && !isUnsaved ? "border-success/20 bg-success/10 text-success" : status === "error" ? "border-destructive/20 bg-destructive/10 text-destructive" : "border-warning/20 bg-warning/10 text-warning"}`}
        >
          {status === "saving"
            ? t("saving")
            : status === "error"
              ? error || t("saveFailed")
              : t(status === "saved" && !isUnsaved ? "saved" : "unsaved")}
        </div>
      )}
      <div className="flex items-center justify-between border-t border-border pt-4">
        <Button
          onClick={onSave}
          disabled={status === "saving" || !draft.wargoal_key.trim()}
        >
          {t("saveButton")}
        </Button>
        {draft.id != null && onDelete && (
          <Button variant="destructive" onClick={onDelete}>
            <Trash2 aria-hidden />
            {t("deleteButton")}
          </Button>
        )}
      </div>
    </div>
  );
}
