"use client";

import { useState } from "react";
import useSWR from "swr";
import { useTranslations } from "use-intl";
import { miosApi, type MIORow, type MIOTraitEntry } from "@/lib/api";
import { MIOTraitEditor } from "@/components/workbench/MIOTraitEditor";
import { GamePanel, PRIMARY } from "./GamePanel";
import { useIndustrialOrg } from "./IndustrialOrgDetailLayer";
import {
  storedTrait,
  traitEntryIndex,
  traitStart,
  traitsWithEntry,
} from "./mioTraitEdit";
import { pick, recordChange, rowUndo } from "./changeLog";
import { changedKeys, useWorkbenchProject } from "./workbenchProject";

async function loadMio(
  projectId: number,
  id: number,
): Promise<MIORow | undefined> {
  return (await miosApi.list(projectId)).mios.find((row) => row.id === id);
}

export function MioTraitEditPanel({
  projectId,
  tag,
  date,
  mioId,
  token: picked,
  onSaved,
  onClose,
}: {
  projectId: number;
  tag: string;
  date: string | null;

  mioId: string;

  token: string;
  onSaved: (message: string) => void;
  onClose: () => void;
}) {
  const t = useTranslations("gameViewResearch.mio");
  const tPanel = useTranslations("gameViewEdit.panel");
  const detail = useIndustrialOrg(projectId, tag, date, mioId);

  const rows = useSWR(
    ["game-view-mios-db", projectId],
    () => miosApi.list(projectId),
    { revalidateOnFocus: false },
  );

  const [token, setToken] = useState(picked);
  const shown = detail.data?.traits.find((t) => t.token === token);
  const row = rows.data?.mios.find((m) => m.mio_id === mioId) ?? null;
  const place = row ? traitEntryIndex(row, token) : null;
  const anchorTrait = shown?.relative_to
    ? detail.data?.traits.find((t) => t.token === shown.relative_to)
    : undefined;
  const anchor = anchorTrait ? { x: anchorTrait.x, y: anchorTrait.y } : null;
  const start =
    row && place
      ? traitStart(
          place.index >= 0 ? (row.traits ?? [])[place.index] : undefined,
          token,
          shown,
          anchor,
        )
      : null;

  const projectReady = useWorkbenchProject(projectId);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const title = shown?.label ?? token;
  const kind = t("kind", { name: detail.data?.name ?? mioId, token });
  const workbenchHref = row ? `/project/${projectId}/mio?mio=${row.id}` : null;

  const takeIn = async () => {
    setBusy(true);
    setError(null);
    try {
      const response = await fetch(
        `/api/go/projects/${projectId}/gui-render/mio-take`,
        {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ id: mioId }),
        },
      );
      const body = (await response.json()) as {
        file?: string;
        imported?: number;
        detail?: string;
      };
      if (!response.ok)
        throw new Error(body.detail ?? `HTTP ${response.status}`);
      const fresh = await rows.mutate();
      const taken = fresh?.mios.find((m) => m.mio_id === mioId);
      recordChange({
        projectId,
        title: t("takeInLog", { name: detail.data?.name ?? mioId }),
        kind,
        workbenchHref: taken
          ? `/project/${projectId}/mio?mio=${taken.id}`
          : null,
      });
      onSaved(
        t("takenIn", {
          file: body.file?.split("/").pop() ?? t("fileFallback"),
          count: body.imported ?? 0,
        }),
      );
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setBusy(false);
    }
  };

  const saveTrait = async (edited: MIOTraitEntry) => {
    if (!row || !start) return;
    const id = row.id;
    const now = await loadMio(projectId, id);
    if (!now) throw new Error(t("gone"));
    const patch = {
      traits: traitsWithEntry(now, token, storedTrait(edited, start)),
    };
    const keys = changedKeys(now, patch);
    if (!keys.length) return;
    const before = pick(now, keys);
    const saved = await miosApi.update(projectId, id, patch);
    recordChange({
      projectId,
      title,
      kind,
      workbenchHref: `/project/${projectId}/mio?mio=${id}`,
      undo: rowUndo<MIORow>({
        before,
        after: pick(saved, keys),
        load: () => loadMio(projectId, id),
        save: (back) => miosApi.update(projectId, id, back),
      }),
    });
    const renamed = (edited.token ?? "").trim();
    if (renamed) setToken(renamed);
    await rows.mutate();
    onSaved(t("saved", { name: title }));
  };

  const loading = rows.isLoading || detail.isLoading;

  return (
    <GamePanel
      title={title}
      kind={kind}
      source={
        rows.isLoading
          ? undefined
          : row
            ? "项目"
            : rows.data
              ? "原版"
              : undefined
      }
      workbenchHref={workbenchHref}

      wide={row ? "workbench" : false}
      onClose={onClose}
    >
      {loading && (
        <p className="mt-4 text-micro text-muted-foreground">{t("loading")}</p>
      )}
      {!rows.isLoading && rows.data && !row && (
        <section className="mt-4 space-y-2">
          <p className="text-micro text-muted-foreground">{t("fromGame")}</p>
          <button
            type="button"
            disabled={busy}
            onClick={() => void takeIn()}
            className={PRIMARY}
          >
            {t("takeIn")}
          </button>
          {error && <p className="text-xs text-destructive">{error}</p>}
        </section>
      )}
      {!loading &&
        row &&
        place &&
        start &&
        (projectReady ? (
          <>
            <p className="mt-3 text-micro text-muted-foreground">
              {place.own
                ? t("own")
                : place.index >= 0
                  ? t("overridden")
                  : t("inherited")}
              {shown &&
                (shown.relative_to
                  ? t("placeRelative", {
                      x: shown.x,
                      y: shown.y,
                      anchor: shown.relative_to,
                    })
                  : t("place", { x: shown.x, y: shown.y }))}
            </p>

            <div className="-mx-4 mt-2">
              <MIOTraitEditor
                key={row.id}
                trait={start.trait}
                index={
                  place.index >= 0 ? place.index : (row.traits ?? []).length
                }
                projectId={projectId}
                mioDbId={row.id}
                onSave={saveTrait}
              />
            </div>
          </>
        ) : (
          <p className="mt-4 text-micro text-muted-foreground">
            {tPanel("loadingProject")}
          </p>
        ))}
    </GamePanel>
  );
}
