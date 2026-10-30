"use client";

import { useState } from "react";
import { useLocale, useTranslations } from "use-intl";
import useSWR from "swr";
import { toast } from "sonner";
import { useConfirm } from "@/components/shared/ConfirmDialogProvider";
import { mapResponseProblem, mapToolFetch } from "@/lib/mapToolRequest";
import { BUTTON } from "../GamePanel";
import { mapProblemMessage } from "./mapProblems";

export interface MapRecord {
  id: number;
  kind: string;
  name: string;
  created_at?: string;
}

export interface MapRecords {
  count: number;
  records: MapRecord[];
  confirmation: string;
}

const SHOWN = 8;

export function mapRecordsOf(out: {
  map_changes?: {
    count?: number;
    records?: MapRecord[];
    confirmation?: string;
  } | null;
}): MapRecords {
  const changes = out.map_changes;
  const records = [...(changes?.records ?? [])].sort((a, b) => b.id - a.id);
  return {
    count: changes?.count ?? records.length,
    records,
    confirmation: changes?.confirmation ?? "",
  };
}

export function MapHistory({
  projectId,
  disabled,
  blockedReason,
  onChanged,
}: {
  projectId: number;

  disabled: boolean;

  blockedReason?: string;

  onChanged: () => Promise<void> | void;
}) {
  const t = useTranslations("mapStudio");
  const locale = useLocale();
  const confirm = useConfirm();
  const [open, setOpen] = useState(false);
  const [busy, setBusy] = useState<"latest" | "all" | null>(null);
  const [error, setError] = useState<string | null>(null);
  const records = useSWR(["world-map-records", projectId], async () => {
    const response = await mapToolFetch(
      `/api/go/projects/${projectId}/gui-render/world-map/land`,
    );
    if (!response.ok)
      throw new Error(
        mapProblemMessage(await mapResponseProblem(response), t, locale),
      );
    return mapRecordsOf(await response.json());
  });
  const data = records.data;
  const latest = data?.records[0];

  const kindLabel = (kind: string) =>
    kind === "land"
      ? t("history.kind.land")
      : kind === "world"
        ? t("history.kind.world")
        : t("history.kind.edit");

  const nameShown = locale.toLowerCase().startsWith("zh");
  const when = (at?: string) => {
    const date = at ? new Date(at) : null;
    return date && !Number.isNaN(date.getTime())
      ? new Intl.DateTimeFormat(locale, {
          month: "2-digit",
          day: "2-digit",
          hour: "2-digit",
          minute: "2-digit",
        }).format(date)
      : "";
  };
  const label = (record: MapRecord) =>
    nameShown && record.name ? record.name : kindLabel(record.kind);

  const take = async (which: "latest" | "all") => {
    if (!data || !latest) return;
    const asked =
      which === "latest"
        ? await confirm({
            title: t("history.undoLatestTitle"),
            body: t("history.undoLatestBody", { name: label(latest) }),
            danger: true,
            confirmLabel: t("history.undoLatest"),
          })
        : await confirm({
            title: t("history.restoreAllTitle"),
            body: t("history.restoreAllBody", { count: data.count }),
            danger: true,
            confirmLabel: t("history.restoreAll"),
          });
    if (!asked) return;
    setBusy(which);
    setError(null);
    try {
      const query =
        which === "latest"
          ? "latest=1"
          : `confirm=${encodeURIComponent(data.confirmation)}`;
      const response = await mapToolFetch(
        `/api/go/projects/${projectId}/gui-render/world-map/land?${query}`,
        { method: "DELETE" },
      );
      if (!response.ok) {
        const problem = await mapResponseProblem(response);

        if (problem.code === "map_changes_moved") {
          await records.mutate();
          setError(t("history.moved"));
          return;
        }
        throw new Error(mapProblemMessage(problem, t, locale));
      }
      await records.mutate();
      await onChanged();
      toast.success(
        which === "latest"
          ? t("history.undone", { name: label(latest) })
          : t("history.restored"),
      );
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setBusy(null);
    }
  };

  const count = data?.count ?? 0;
  return (
    <section className="space-y-2 border-t border-border pt-3" data-map-history>
      <button
        type="button"
        className="flex w-full items-center justify-between text-xs font-medium"
        aria-expanded={open}
        onClick={() => setOpen((now) => !now)}
      >
        <span>{t("history.title", { count })}</span>
        <span className="text-muted-foreground">
          {open ? t("history.hide") : t("history.show")}
        </span>
      </button>
      {open && (
        <div className="space-y-2">
          {records.error ? (
            <p role="alert" className="text-xs text-destructive">
              {records.error instanceof Error
                ? records.error.message
                : String(records.error)}
            </p>
          ) : !data ? (
            <p className="text-xs text-muted-foreground">
              {t("history.loading")}
            </p>
          ) : count === 0 ? (
            <p className="text-xs text-muted-foreground">
              {t("history.empty")}
            </p>
          ) : (
            <>
              <ul className="space-y-1 text-xs">
                {data.records.slice(0, SHOWN).map((record) => (
                  <li
                    key={record.id}
                    className="flex min-w-0 items-center gap-2"
                    title={record.name || undefined}
                  >
                    <span className="shrink-0 rounded border border-border px-1.5 text-muted-foreground">
                      {kindLabel(record.kind)}
                    </span>
                    <span className="min-w-0 flex-1 truncate">
                      {label(record)}
                    </span>
                    <span className="shrink-0 tabular-nums text-muted-foreground">
                      {when(record.created_at)}
                    </span>
                  </li>
                ))}
                {count > SHOWN && (
                  <li className="text-muted-foreground">
                    {t("history.more", { count: count - SHOWN })}
                  </li>
                )}
              </ul>
              {blockedReason && (
                <p className="text-xs text-muted-foreground">{blockedReason}</p>
              )}
              <div className="flex flex-wrap gap-1">
                <button
                  type="button"
                  className={BUTTON}
                  disabled={disabled || busy !== null}
                  onClick={() => void take("latest")}
                >
                  {busy === "latest"
                    ? t("history.working")
                    : t("history.undoLatest")}
                </button>
                <button
                  type="button"
                  className={BUTTON}
                  disabled={disabled || busy !== null}
                  onClick={() => void take("all")}
                >
                  {busy === "all"
                    ? t("history.working")
                    : t("history.restoreAll")}
                </button>
              </div>
            </>
          )}
          {error && (
            <p role="alert" className="text-xs text-destructive">
              {error}
            </p>
          )}
        </div>
      )}
    </section>
  );
}
