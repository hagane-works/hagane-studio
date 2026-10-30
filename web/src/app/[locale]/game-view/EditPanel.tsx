"use client";

import { useState } from "react";
import useSWR from "swr";
import { useTranslations } from "use-intl";
import { charactersApi } from "@/lib/api/characters";
import { gameLocale, guiRenderApi, withGameLanguage } from "@/lib/guiRenderApi";
import type { EditTarget, TextEdit } from "./editModel";
import { SpriteReplace } from "./SpriteReplace";
import { BUTTON, FIELD, GamePanel, PanelSave } from "./GamePanel";
import { recordChange, rowUndo, textUndo } from "./changeLog";
import {
  GuiTextNotExportable,
  restoreGuiText,
  saveGuiText,
} from "./guiTextOverlay";
import { removeElement, showElement, type Removal } from "./elementVisibility";

export interface TextSaveHandlers {
  onTextSaved: (key: string, value: string) => void;

  onTextRestored: (key: string) => void;

  onGuiText: (file: string, target: string, value: string | null) => void;

  onTextReload: () => void;
}

export function EditPanel({
  projectId,
  target,
  spriteVersion,
  onSpriteChanged,
  onOwnPicture,
  onElementVisibility,
  onTextSaved,
  onTextRestored,
  onGuiText,
  onTextReload,
  onPickStacked,
  countryFlag,
  onEditCountryFlag,
  onClose,
}: {
  projectId: number;
  target: EditTarget;

  countryFlag?: {
    tag: string;
    url: string;
    width: number;
    height: number;
  } | null;

  onEditCountryFlag?: () => void;

  spriteVersion?: number;

  onSpriteChanged: (
    sprite: string,
    width: number,
    height: number,
    action: "replace" | "restore",
  ) => void;

  onOwnPicture?: (
    gfx: string | null,
    shape: { width: number; height: number; frames: number },
  ) => void;

  onElementVisibility?: (target: string, state: Removal | "shown") => void;

  onPickStacked?: (id: string) => void;
  onClose: () => void;
} & TextSaveHandlers) {
  const t = useTranslations("gameViewEdit");
  const text = target.text;
  const handlers: TextSaveHandlers = {
    onTextSaved,
    onTextRestored,
    onGuiText,
    onTextReload,
  };

  return (
    <GamePanel
      title={target.title}
      kind={target.windowName}
      wide={false}
      onClose={onClose}
    >
      {target.stack.length > 0 && onPickStacked && (
        <section className="mt-3">
          <div className="text-micro font-medium text-muted-foreground">
            {t("uiEdit.stackHeading")}
          </div>
          <div
            className="mt-1 flex flex-wrap gap-1"
            role="group"
            aria-label={t("uiEdit.stackLabel")}
          >
            {target.stack.map((entry) => (
              <button
                key={entry.id}
                type="button"
                aria-pressed={entry.picked}
                title={entry.name}
                onClick={() => onPickStacked(entry.id)}
                className={`max-w-full truncate rounded-md border px-2 py-0.5 text-xs ui-interactive ${
                  entry.picked
                    ? "border-accent bg-accent text-accent-foreground"
                    : "border-border hover:bg-muted"
                }`}
              >
                {entry.kind ? t(`uiEdit.kind.${entry.kind}`) : entry.type} ·{" "}
                {entry.backdrop ? t("uiEdit.backdrop") : entry.name}
              </button>
            ))}
          </div>
        </section>
      )}

      {target.sprite && target.flag && (
        <section className="mt-4">
          <div className="text-micro font-medium text-muted-foreground">
            {t("uiEdit.flag")}
          </div>

          {countryFlag && target.windowName === "topbar" && (
            <div className="mt-2 flex items-center gap-3">
              <div
                role="img"
                aria-label={t("uiEdit.currentFlag", { tag: countryFlag.tag })}
                className="shrink-0 rounded-md border border-border"
                style={{
                  width: countryFlag.width,
                  height: countryFlag.height,
                  backgroundImage: `url("${countryFlag.url}")`,
                  backgroundSize: "100% 100%",
                }}
              />
              <span className="text-xs text-muted-foreground">
                {t("uiEdit.currentFlag", { tag: countryFlag.tag })}
              </span>
            </div>
          )}
          <p className="mt-2 text-micro text-muted-foreground">
            {t("uiEdit.flagNote")}
          </p>
          {onEditCountryFlag && (
            <button
              type="button"
              className={`mt-2 ${BUTTON}`}
              onClick={onEditCountryFlag}
            >
              {t("uiEdit.editCountryFlag")}
            </button>
          )}
          <details className="mt-3">
            <summary className="cursor-pointer text-micro text-muted-foreground">
              {t("uiEdit.glossSummary")}
            </summary>
            <SpriteReplace
              sprite={target.sprite}
              size={target.spriteSize}
              version={spriteVersion}
              preview={target.preview}
              own={target.ownPicture}
              label={t("uiEdit.gloss")}
              onChanged={onSpriteChanged}
              onOwnPicture={onOwnPicture}
            />
          </details>
        </section>
      )}

      {target.sprite && !target.flag && (
        <SpriteReplace
          sprite={target.sprite}
          size={target.spriteSize}
          version={spriteVersion}
          preview={target.preview}
          own={target.ownPicture}
          onChanged={onSpriteChanged}
          onOwnPicture={onOwnPicture}
        />
      )}

      {text && (
        <section className="mt-4">
          <div className="text-micro font-medium text-muted-foreground">
            {t("uiEdit.text")}
          </div>
          <TextField
            key={`${target.windowName}/${target.title}/${textIdentity(text)}`}
            projectId={projectId}
            title={target.title}
            text={text}
            handlers={handlers}
          />
        </section>
      )}

      {!target.sprite && !text && !target.removal && (
        <p className="mt-4 text-micro text-muted-foreground">
          {t("uiEdit.nothing")}
        </p>
      )}

      {target.removal && onElementVisibility && (
        <RemoveSection
          key={`${target.removal.file}/${target.removal.target}/${target.removal.hidden}`}
          projectId={projectId}
          title={target.title}
          removal={target.removal}
          onChanged={onElementVisibility}
        />
      )}
    </GamePanel>
  );
}

function RemoveSection({
  projectId,
  title,
  removal,
  onChanged,
}: {
  projectId: number;
  title: string;
  removal: { file: string; target: string; hidden: boolean };
  onChanged: (target: string, state: Removal | "shown") => void;
}) {
  const t = useTranslations("gameViewEdit");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const kind = t("page.elementKind");
  const act = async (work: () => Promise<void>) => {
    setBusy(true);
    setError(null);
    try {
      await work();
    } catch (err) {
      setError(
        t("uiEdit.removeFailed", {
          error:
            err instanceof GuiTextNotExportable
              ? t("uiEdit.removeNotExportable")
              : err instanceof Error
                ? err.message
                : String(err),
        }),
      );
    } finally {
      setBusy(false);
    }
  };
  const show = () =>
    act(async () => {
      await showElement(removal);
      recordChange({
        projectId,
        title: t("page.elementShown", { name: title }),
        kind,
      });
      onChanged(removal.target, "shown");
    });
  const remove = () =>
    act(async () => {
      const done = await removeElement(removal);
      recordChange({
        projectId,
        title: t(
          done.kind === "removed"
            ? "page.elementRemoved"
            : "page.elementHidden",
          { name: title },
        ),
        kind,
        undo: async () => {
          await done.undo();
          onChanged(removal.target, "shown");
        },
      });
      onChanged(removal.target, done.kind);
    });
  return (
    <section className="mt-4">
      <div className="text-micro font-medium text-muted-foreground">
        {t("uiEdit.removeHeading")}
      </div>
      <p className="mt-1 text-micro text-muted-foreground">
        {removal.hidden ? t("uiEdit.hiddenNote") : t("uiEdit.removeHint")}
      </p>
      <button
        type="button"
        disabled={busy}
        className={`mt-2 ${BUTTON}`}
        onClick={() => void (removal.hidden ? show() : remove())}
      >
        {busy
          ? t("uiEdit.removing")
          : removal.hidden
            ? t("uiEdit.showAgain")
            : t("uiEdit.remove")}
      </button>
      {error && (
        <p className="mt-2 break-all text-micro text-destructive">{error}</p>
      )}
    </section>
  );
}

function textIdentity(text: TextEdit): string {
  if (!text.editable) return `locked/${text.reason}/${text.current}`;
  if (text.via === "key") return `key/${text.key}/${text.current}`;
  if (text.via === "gui")
    return `gui/${text.file}/${text.target}/${text.current}`;
  return `person/${text.token}/${text.current}`;
}

function saveErrorText(
  err: unknown,
  t: (key: string, values?: Record<string, string>) => string,
): string {
  if (err instanceof GuiTextNotExportable) return t("notExportable");
  const refused = err as { code?: unknown; file?: unknown } | null;
  if (refused?.code === "owned_by_project_data") {
    return t("ownedByProject", {
      file: typeof refused.file === "string" ? refused.file : "",
    });
  }
  return err instanceof Error ? err.message : String(err);
}

function useSaving() {
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const t = useTranslations("gameViewEdit.uiEdit");
  const run = async (work: () => Promise<void>) => {
    setBusy(true);
    setError(null);
    try {
      await work();
    } catch (err) {
      setError(saveErrorText(err, t));
    } finally {
      setBusy(false);
    }
  };
  return { busy, error, run };
}

function TextField({
  projectId,
  title,
  text,
  handlers,
}: {
  projectId: number;
  title: string;
  text: TextEdit;
  handlers: TextSaveHandlers;
}) {
  const t = useTranslations("gameViewEdit.uiEdit");
  if (!text.editable) {
    return (
      <>
        <div className="mt-1 whitespace-pre-wrap break-words text-sm">
          {text.current}
        </div>
        <p className="mt-2 text-micro text-muted-foreground">
          {t(`reason.${text.reason}`)}
        </p>
      </>
    );
  }
  if (text.via === "gui")
    return (
      <GuiTextField
        text={text}
        title={title}
        projectId={projectId}
        handlers={handlers}
      />
    );
  if (text.via === "person")
    return (
      <PersonNameField
        text={text}
        title={title}
        projectId={projectId}
        handlers={handlers}
      />
    );
  const note =
    text.scope === "everywhere" ? t("textNote") : t(`scope.${text.scope}`);
  return text.runtime ? (
    <StoredKeyField
      projectId={projectId}
      title={title}
      textKey={text.key}
      shown={text.current}
      rewritten={text.rewritten}
      note={note}
      handlers={handlers}
    />
  ) : (
    <KeyField
      projectId={projectId}
      title={title}
      textKey={text.key}
      current={text.current}
      rewritten={text.rewritten}
      note={note}
      handlers={handlers}
    />
  );
}

function StoredKeyField({
  projectId,
  title,
  textKey,
  shown,
  rewritten,
  note,
  handlers,
}: {
  projectId: number;
  title: string;
  textKey: string;
  shown: string;
  rewritten: boolean;
  note: string;
  handlers: TextSaveHandlers;
}) {
  const t = useTranslations("gameViewEdit.uiEdit");
  const stored = useSWR(
    ["game-view-ui-text", projectId, gameLocale(), textKey],
    async () => {
      guiRenderApi.setProjectId(projectId);
      return (await guiRenderApi.loc([textKey], { markup: true }))[textKey];
    },
    { revalidateOnFocus: false },
  );
  if (stored.isLoading)
    return (
      <p className="mt-1 text-micro text-muted-foreground">
        {t("loadingText")}
      </p>
    );

  const current = stored.data ?? shown;
  return (
    <KeyField
      key={current}
      projectId={projectId}
      title={title}
      textKey={textKey}
      current={current}
      rewritten={rewritten}
      note={note}
      runtime
      handlers={{
        ...handlers,
        onTextSaved: (key, value) => {
          void stored.mutate(value, { revalidate: false });
          handlers.onTextSaved(key, value);
          handlers.onTextReload();
        },
        onTextRestored: (key) => {
          void stored.mutate();
          handlers.onTextRestored(key);
          handlers.onTextReload();
        },
      }}
    />
  );
}

function KeyField({
  projectId,
  title,
  textKey,
  current,
  rewritten,
  note,
  runtime = false,
  handlers,
}: {
  projectId: number;
  title: string;
  textKey: string;
  current: string;
  rewritten: boolean;
  note: string;

  runtime?: boolean;
  handlers: TextSaveHandlers;
}) {
  const tBase = useTranslations("gameViewEdit.baseText");
  const [draft, setDraft] = useState(current);
  const { busy, error, run } = useSaving();

  const logText = (wrote: string | null, before: string | null) =>
    recordChange({
      projectId,
      title,
      kind: tBase("logKind", { key: textKey }),
      undo: textUndo(textKey, wrote, before, {
        set: (key, value) => {
          handlers.onTextSaved(key, value);
          if (runtime) handlers.onTextReload();
        },
        cleared: (key) => {
          handlers.onTextRestored(key);
          if (runtime) handlers.onTextReload();
        },
      }),
    });
  return (
    <>
      <div className="mt-1 font-mono text-micro text-muted-foreground">
        {textKey}
      </div>
      <textarea
        value={draft}
        disabled={busy}
        rows={Math.min(8, Math.max(2, draft.split("\n").length))}
        onChange={(e) => setDraft(e.target.value)}
        className={FIELD}
      />
      <PanelSave
        onSave={() =>
          void run(async () => {
            guiRenderApi.setProjectId(projectId);
            await guiRenderApi.setText(textKey, draft);
            handlers.onTextSaved(textKey, draft);
            logText(draft, rewritten ? current : null);
          })
        }
        canSave={draft !== current}
        busy={busy}
        onRevert={draft !== current ? () => setDraft(current) : undefined}
        note={note}
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
                await guiRenderApi.clearText(textKey);
                handlers.onTextRestored(textKey);
                logText(null, current);
              })
            }
          >
            {tBase("restore")}
          </button>
        )}
      </PanelSave>
    </>
  );
}

function GuiTextField({
  projectId,
  title,
  text,
  handlers,
}: {
  projectId: number;
  title: string;
  text: Extract<TextEdit, { via: "gui" }>;
  handlers: TextSaveHandlers;
}) {
  const t = useTranslations("gameViewEdit.uiEdit");
  const tBase = useTranslations("gameViewEdit.baseText");
  const [draft, setDraft] = useState(text.current);
  const { busy, error, run } = useSaving();
  const at = {
    file: text.file,
    target: text.target,
    originalText: text.originalText,
    originalKind: text.originalKind,
  };
  const log = () =>
    recordChange({
      projectId,
      title,
      kind: t("guiLogKind", { target: text.target }),
    });
  return (
    <>
      <div className="mt-1 break-all font-mono text-micro text-muted-foreground">
        {text.target}
      </div>
      <textarea
        value={draft}
        disabled={busy}
        rows={Math.min(8, Math.max(2, draft.split("\n").length))}
        onChange={(e) => setDraft(e.target.value)}
        className={FIELD}
      />
      <PanelSave
        onSave={() =>
          void run(async () => {
            guiRenderApi.setProjectId(projectId);
            await saveGuiText(at, gameLocale(), draft);
            log();
            handlers.onGuiText(text.file, text.target, draft);
            handlers.onTextReload();
          })
        }
        canSave={draft !== text.current && draft.trim() !== ""}
        busy={busy}
        onRevert={
          draft !== text.current ? () => setDraft(text.current) : undefined
        }
        note={t("guiNote")}
        error={error}
      >
        {text.rewritten && (
          <button
            type="button"
            disabled={busy}
            className={BUTTON}
            onClick={() =>
              void run(async () => {
                guiRenderApi.setProjectId(projectId);
                await restoreGuiText(at);
                log();
                handlers.onGuiText(text.file, text.target, null);
                handlers.onTextReload();
              })
            }
          >
            {tBase("restore")}
          </button>
        )}
      </PanelSave>
    </>
  );
}

function PersonNameField({
  projectId,
  title,
  text,
  handlers,
}: {
  projectId: number;
  title: string;
  text: Extract<TextEdit, { via: "person" }>;
  handlers: TextSaveHandlers;
}) {
  const t = useTranslations("gameViewEdit.uiEdit");

  const rows = useSWR(
    ["game-view-characters-db", projectId],
    () => charactersApi.list(projectId),
    {
      revalidateOnFocus: false,
    },
  );
  const row =
    rows.data?.find((character) => character.tag === text.token) ?? null;
  const base = useSWR(
    rows.data && !row
      ? ["game-view-character-key", projectId, text.token]
      : null,
    async () => {
      const response = await fetch(
        withGameLanguage(
          `/api/go/projects/${projectId}/gui-render/character?token=${encodeURIComponent(text.token)}`,
        ),
      );
      return response.ok
        ? ((await response.json()) as { name_key?: string })
        : null;
    },
    { revalidateOnFocus: false },
  );
  if (rows.isLoading || (!row && base.isLoading)) {
    return (
      <p className="mt-1 text-micro text-muted-foreground">
        {t("loadingText")}
      </p>
    );
  }
  if (!row) {
    return (
      <StoredKeyField
        projectId={projectId}
        title={title}
        textKey={base.data?.name_key || text.token}
        shown={text.current}
        rewritten={false}
        note={t("personNote")}
        handlers={handlers}
      />
    );
  }
  const lang = gameLocale();
  const current = row.display_name_locs?.[lang] || text.current;
  return (
    <ProjectPersonName
      key={`${row.id}/${current}`}
      projectId={projectId}
      title={title}
      token={text.token}
      rowId={row.id}
      names={row.display_name_locs ?? {}}
      lang={lang}
      current={current}
      onSaved={async () => {
        await rows.mutate();
        handlers.onTextReload();
      }}
    />
  );
}

function ProjectPersonName({
  projectId,
  title,
  token,
  rowId,
  names,
  lang,
  current,
  onSaved,
}: {
  projectId: number;
  title: string;
  token: string;
  rowId: number;
  names: Record<string, string>;
  lang: string;
  current: string;
  onSaved: () => Promise<void>;
}) {
  const t = useTranslations("gameViewEdit.uiEdit");
  const tPerson = useTranslations("gameViewEdit.person");
  const [draft, setDraft] = useState(current);
  const { busy, error, run } = useSaving();
  return (
    <>
      <textarea
        value={draft}
        disabled={busy}
        rows={1}
        onChange={(e) => setDraft(e.target.value)}
        className={FIELD}
      />
      <PanelSave
        onSave={() =>
          void run(async () => {
            const before = { display_name_locs: names };
            const saved = await charactersApi.update(projectId, rowId, {
              display_name_locs: { ...names, [lang]: draft },
            });

            const after = { display_name_locs: saved.display_name_locs };
            recordChange({
              projectId,
              title,
              kind: tPerson("kind", { token }),
              workbenchHref: `/project/${projectId}/characters?c=${rowId}`,
              undo: rowUndo({
                before,
                after,
                load: () => charactersApi.get(projectId, rowId),
                save: (back) => charactersApi.update(projectId, rowId, back),
              }),
            });
            await onSaved();
          })
        }
        canSave={draft !== current && draft.trim() !== ""}
        busy={busy}
        onRevert={draft !== current ? () => setDraft(current) : undefined}
        note={t("personNote")}
        error={error}
      />
    </>
  );
}
