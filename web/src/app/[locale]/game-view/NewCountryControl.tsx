"use client";

import { useDeferredValue, useMemo, useState } from "react";
import { useTranslations } from "use-intl";
import useSWR, { mutate as globalMutate } from "swr";
import { toast } from "sonner";
import { AddCountryDialog } from "@/components/workbench/AddCountryDialog";
import { statesApi } from "@/lib/api/scenario";
import { toRestoreEntries } from "@/lib/states/bulkOwnerCore";
import { recordChange } from "./changeLog";
import { withGameLanguage } from "@/lib/guiRenderApi";

const COUNTRY_KEYS = new Set([
  "game-view-bookmarks",
  "game-view-stats",
  "world-map",
]);

interface StateName {
  id: number;
  name: string;
  owner: string;
}

function useStateNames(projectId: number, date: string, enabled: boolean) {
  return useSWR<StateName[]>(
    enabled ? ["game-view-state-names", projectId, date] : null,
    async () => {
      const response = await fetch(
        withGameLanguage(
          `/api/go/projects/${projectId}/gui-render/world-map?date=${encodeURIComponent(date)}`,
        ),
      );
      if (!response.ok) throw new Error(`HTTP ${response.status}`);
      const payload = (await response.json()) as {
        states: [number, string, string][];
      };
      return payload.states
        .map(([id, name, owner]) => ({ id, name, owner }))
        .sort((a, b) => a.id - b.id);
    },
    { revalidateOnFocus: false },
  );
}

export function matchStates(
  states: readonly StateName[],
  query: string,
  limit = 40,
): StateName[] {
  const q = query.trim().toLowerCase();
  if (!q) return [];
  const number = Number(q.replace(/^#/, ""));
  return states
    .filter(
      (state) =>
        state.name.toLowerCase().includes(q) ||
        (Number.isInteger(number) && state.id === number) ||
        state.owner.toLowerCase() === q,
    )
    .slice(0, limit);
}

export function NewCountryControl({
  projectId,
  date,
}: {
  projectId: number;
  date: string;
}) {
  const t = useTranslations("gameViewWorld.newCountry");
  const [open, setOpen] = useState(false);
  const [dialogOpen, setDialogOpen] = useState(false);
  const [tag, setTag] = useState<string | null>(null);
  const [query, setQuery] = useState("");
  const q = useDeferredValue(query);
  const [chosen, setChosen] = useState<number[]>([]);
  const [busy, setBusy] = useState(false);
  const names = useStateNames(projectId, date, open && tag !== null);
  const hits = useMemo(() => matchStates(names.data ?? [], q), [names.data, q]);
  const nameOf = useMemo(
    () => new Map((names.data ?? []).map((state) => [state.id, state])),
    [names.data],
  );

  const refresh = async () => {
    await names.mutate();
    await globalMutate(
      (key) =>
        Array.isArray(key) &&
        typeof key[0] === "string" &&
        COUNTRY_KEYS.has(key[0]),
    );
  };

  const give = async () => {
    if (!tag || chosen.length === 0) return;
    setBusy(true);
    try {
      let rows = await statesApi.list(projectId);
      const missing = chosen.filter(
        (id) => !rows.some((row) => row.state_id === id),
      );
      if (missing.length) {
        await statesApi.seedFromVanilla(projectId, { state_ids: missing });
        rows = await statesApi.list(projectId);
      }
      const ids = chosen
        .map((id) => rows.find((row) => row.state_id === id)?.id)
        .filter((id): id is number => typeof id === "number");
      const result = await statesApi.bulkOwnerCore(projectId, {
        state_db_ids: ids,
        owner: tag,
        cores: { mode: "add", tags: [tag] },
      });
      const entries = toRestoreEntries(result);
      recordChange({
        projectId,
        title: t("gaveLog", { tag, count: chosen.length }),
        kind: t("statesKind", { ids: chosen.map((id) => `#${id}`).join(" ") }),
        workbenchHref: `/project/${projectId}/states`,
        undo: async () => {
          await statesApi.bulkRestore(projectId, entries);
          await refresh();
        },
      });
      await refresh();
      toast.success(t("gave", { count: chosen.length, tag }), {
        description: t("gaveHint"),
      });
      setChosen([]);
      setQuery("");
    } catch (err) {
      toast.error(
        t("giveFailed", {
          error: err instanceof Error ? err.message : String(err),
        }),
      );
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="w-full rounded-md border border-border bg-card p-3 text-card-foreground shadow-lg">
      <button
        type="button"
        onClick={() => setOpen((v) => !v)}
        aria-expanded={open}
        className="flex w-full items-center justify-between gap-2 text-left"
      >
        <span className="min-w-0">
          <span className="block text-micro text-muted-foreground">
            {t("heading")}
          </span>
          <span className="block truncate text-sm">{t("summary")}</span>
        </span>
        <span className="text-muted-foreground">{open ? "▴" : "▾"}</span>
      </button>
      {open && (
        <div className="mt-3 space-y-3">
          <button
            type="button"
            onClick={() => setDialogOpen(true)}
            className="rounded border border-border px-2 py-0.5 text-micro hover:bg-muted"
          >
            {t("create")}
          </button>
          {tag ? (
            <section className="space-y-2 border-t border-border pt-3">
              <div className="text-micro text-muted-foreground">
                {t("giveTitle", { tag })}
              </div>
              <input
                value={query}
                onChange={(e) => setQuery(e.target.value)}
                placeholder={t("searchPlaceholder")}
                aria-label={t("searchLabel")}
                className="w-full rounded border border-input bg-background px-2 py-1 text-sm"
              />
              <ul className="max-h-60 space-y-0.5 overflow-y-auto">
                {hits.map((state) => {
                  const on = chosen.includes(state.id);
                  return (
                    <li key={state.id}>
                      <button
                        type="button"
                        aria-pressed={on}
                        onClick={() =>
                          setChosen((list) =>
                            on
                              ? list.filter((id) => id !== state.id)
                              : [...list, state.id],
                          )
                        }
                        className={`flex w-full items-center justify-between gap-2 rounded px-2 py-1 text-left text-micro ${on ? "bg-primary text-primary-foreground" : "hover:bg-muted"}`}
                      >
                        <span className="truncate">
                          #{state.id} {state.name}
                        </span>
                        <span className="shrink-0 font-mono">
                          {state.owner || "—"}
                        </span>
                      </button>
                    </li>
                  );
                })}
                {q.trim() && names.data && hits.length === 0 && (
                  <li className="text-micro text-muted-foreground">
                    {t("noMatch")}
                  </li>
                )}
                {!names.data && (
                  <li className="text-micro text-muted-foreground">
                    {t("loading")}
                  </li>
                )}
              </ul>
              {chosen.length > 0 && (
                <p className="text-micro text-muted-foreground">
                  {t("chosen", {
                    list: chosen
                      .map((id) =>
                        `#${id} ${nameOf.get(id)?.name ?? ""}`.trim(),
                      )
                      .join(t("listSeparator")),
                  })}
                </p>
              )}
              <button
                type="button"
                disabled={busy || chosen.length === 0}
                onClick={() => void give()}
                className="rounded bg-primary px-2 py-0.5 text-micro text-primary-foreground disabled:opacity-50"
              >
                {busy ? t("giving") : t("give", { count: chosen.length, tag })}
              </button>
            </section>
          ) : (
            <p className="text-micro text-muted-foreground">{t("hint")}</p>
          )}
        </div>
      )}
      <AddCountryDialog
        projectId={projectId}
        open={dialogOpen}
        onOpenChange={setDialogOpen}
        onAdded={(added) => {
          setTag(added);
          setOpen(true);
          recordChange({
            projectId,
            title: t("createdLog", { tag: added }),
            kind: t("countryKind", { tag: added }),
            workbenchHref: `/project/${projectId}/country-profile`,
          });
          void globalMutate(
            (key) =>
              Array.isArray(key) &&
              typeof key[0] === "string" &&
              COUNTRY_KEYS.has(key[0]),
          );
        }}
      />
    </div>
  );
}
