"use client";

import { useState } from "react";
import useSWR from "swr";
import { useTranslations } from "use-intl";
import { guiRenderApi, withGameLanguage } from "@/lib/guiRenderApi";
import { BUTTON, FIELD, PanelSave } from "./GamePanel";
import { recordChange, textUndo } from "./changeLog";

export type ContentOrigin = "vanilla" | "mod";

export function originNote(
  origin?: string,
  file?: string,
): {
  key:
    | "originVanilla"
    | "originVanillaIn"
    | "originMod"
    | "originModIn"
    | "originUnknown";
  file?: string;
} {
  if (origin === "vanilla")
    return file ? { key: "originVanillaIn", file } : { key: "originVanilla" };
  if (origin === "mod")
    return file ? { key: "originModIn", file } : { key: "originMod" };
  return { key: "originUnknown" };
}

export type BaseTextField = {
  field: "name" | "description" | "title" | "body" | "option" | "cardLabel";
  key: string;

  n?: number;
};

export function BaseTextEdit({
  projectId,
  origin,
  file,
  title,
  fields,
  logicNote = "logicNote",
  onSaved,
}: {
  projectId: number;
  origin?: string;

  file?: string;

  title: string;
  fields: BaseTextField[];

  logicNote?: "logicNote" | "logicNoteDecision";

  onSaved?: (key: string, value: string | null) => void;
}) {
  const t = useTranslations("gameViewEdit.baseText");
  const keys = fields.map((field) => field.key);
  const where = originNote(origin, file);
  const texts = useSWR(
    keys.length ? ["game-view-base-text", projectId, keys.join(",")] : null,
    async () => {
      guiRenderApi.setProjectId(projectId);
      const [current, overrides] = await Promise.all([
        guiRenderApi.loc(keys),
        guiRenderApi.textOverrides(),
      ]);
      return { current, overrides };
    },
    { revalidateOnFocus: false },
  );
  return (
    <section className="mt-4 space-y-3">
      <p className="text-micro text-muted-foreground">
        {t(where.key, { file: where.file ?? "" })}
      </p>
      <div className="text-micro font-medium text-muted-foreground">
        {t("heading")}
      </div>
      {texts.isLoading || !texts.data ? (
        <p className="text-micro text-muted-foreground">{t("loading")}</p>
      ) : (
        fields.map((field) => (
          <TextFieldEdit
            key={`${field.key}/${texts.data?.current[field.key] ?? ""}`}
            projectId={projectId}
            title={title}
            field={field}
            current={texts.data?.current[field.key] ?? ""}
            rewritten={Object.prototype.hasOwnProperty.call(
              texts.data?.overrides ?? {},
              field.key,
            )}
            onSaved={(key, value) => {
              void texts.mutate();
              onSaved?.(key, value);
            }}
          />
        ))
      )}
      <p className="text-xs text-muted-foreground">
        {t("note")}
        {origin === "vanilla" ? t("noteShared") : ""}
        {t(logicNote)}
      </p>
    </section>
  );
}

export function BasePersonText({
  projectId,
  token,
  name,
  onSaved,
}: {
  projectId: number;
  token: string;

  name?: string;
  onSaved?: () => void;
}) {
  const t = useTranslations("gameViewEdit.baseText");
  const found = useSWR(
    ["game-view-character-key", projectId, token],
    async () => {
      const response = await fetch(
        withGameLanguage(
          `/api/go/projects/${projectId}/gui-render/character?token=${encodeURIComponent(token)}`,
        ),
      );
      return response.ok
        ? ((await response.json()) as { name_key?: string; origin?: string })
        : null;
    },
    { revalidateOnFocus: false },
  );
  if (found.isLoading)
    return (
      <p className="mt-4 text-micro text-muted-foreground">
        {t("loadingPerson")}
      </p>
    );
  return (
    <BaseTextEdit
      projectId={projectId}
      origin={found.data?.origin}
      title={name || token}
      fields={[{ field: "name", key: found.data?.name_key || token }]}
      onSaved={() => onSaved?.()}
    />
  );
}

function TextFieldEdit({
  projectId,
  title,
  field,
  current,
  rewritten,
  onSaved,
}: {
  projectId: number;
  title: string;
  field: BaseTextField;
  current: string;
  rewritten: boolean;
  onSaved: (key: string, value: string | null) => void;
}) {
  const t = useTranslations("gameViewEdit.baseText");
  const label = t(`field.${field.field}`, { n: field.n ?? 0 });
  const [draft, setDraft] = useState(current);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const log = (wrote: string | null, before: string | null) =>
    recordChange({
      projectId,
      title,
      kind: t("logKind", { key: field.key }),
      undo: textUndo(field.key, wrote, before, {
        set: (key, value) => onSaved(key, value),
        cleared: (key) => onSaved(key, null),
      }),
    });
  const run = async (work: () => Promise<void>) => {
    setBusy(true);
    setError(null);
    try {
      await work();
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setBusy(false);
    }
  };
  return (
    <div>
      <div className="flex items-baseline justify-between gap-2">
        <span className="text-xs">{label}</span>
        <span className="font-mono text-micro text-muted-foreground">
          {field.key}
        </span>
      </div>
      <textarea
        aria-label={label}
        value={draft}
        disabled={busy}
        placeholder={current ? undefined : t("emptyPlaceholder")}
        rows={Math.min(6, Math.max(1, draft.split("\n").length))}
        onChange={(e) => setDraft(e.target.value)}
        className={FIELD}
      />
      <PanelSave
        onSave={() =>
          void run(async () => {
            guiRenderApi.setProjectId(projectId);
            await guiRenderApi.setText(field.key, draft);
            log(draft, rewritten ? current : null);
            onSaved(field.key, draft);
          })
        }
        canSave={draft !== current}
        busy={busy}
        onRevert={draft !== current ? () => setDraft(current) : undefined}
        error={error}
      >
        {rewritten && (
          <button
            type="button"
            disabled={busy}
            className={BUTTON}
            onClick={() =>
              void run(async () => {
                guiRenderApi.setProjectId(projectId);
                await guiRenderApi.clearText(field.key);
                log(null, current);
                onSaved(field.key, null);
              })
            }
          >
            {t("restore")}
          </button>
        )}
      </PanelSave>
    </div>
  );
}
