"use client";

import { useDeferredValue, useState } from "react";
import useSWR from "swr";
import { useTranslations } from "use-intl";
import { toast } from "sonner";
import { eventsApi } from "@/lib/api/content";
import { superEventsApi, type SuperEventRow } from "@/lib/api/scenario";
import { guiRenderApi } from "@/lib/guiRenderApi";
import { newOptionKey } from "./eventEdit";
import { superEventTexts, useSuperEvents } from "./superEventModel";

const EVENT_LIST_MAX = 10000;

export function superEventLabel(row: SuperEventRow): string {
  return `#${String(row.number).padStart(4, "0")} ${superEventTexts(row).title}`;
}

export function EventSwitcher({
  projectId,
  current,
  currentTitle,
  onChoose,
  currentSuperEvent = null,
  onChooseSuperEvent,
  browseOnly = false,
  tag = null,
}: {
  projectId: number;

  current: string | null;
  currentTitle?: string;
  onChoose: (eventId: string) => void;

  currentSuperEvent?: SuperEventRow | null;
  onChooseSuperEvent?: (row: SuperEventRow) => void;

  browseOnly?: boolean;

  tag?: string | null;
}) {
  const t = useTranslations("gameViewContent.eventSwitcher");
  const [open, setOpen] = useState(false);
  const [query, setQuery] = useState("");
  const q = useDeferredValue(query.trim());
  const [scope, setScope] = useState<"country" | "all">("country");
  const listScope = tag ? scope : "all";

  const choices = useSWR(
    open
      ? ["game-view-event-choices", projectId, q, tag ?? "", listScope]
      : null,
    () => {
      guiRenderApi.setProjectId(projectId);
      return guiRenderApi.eventChoicePage(q, 0, EVENT_LIST_MAX, tag, listScope);
    },
    { revalidateOnFocus: false, keepPreviousData: true },
  );
  const choiceList = choices.data?.events ?? [];
  const choiceTotal = choices.data?.total ?? 0;
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const superEvents = useSuperEvents(
    projectId,
    open && Boolean(onChooseSuperEvent),
  );
  const superMatches = (superEvents.data?.super_events ?? [])
    .filter(
      (row) =>
        !q || superEventLabel(row).toLowerCase().includes(q.toLowerCase()),
    )
    .sort((a, b) => a.number - b.number);

  const choose = (id: string) => {
    setOpen(false);
    onChoose(id);
  };
  const chooseSuper = (row: SuperEventRow) => {
    setOpen(false);
    onChooseSuperEvent?.(row);
  };

  const createSuper = async () => {
    if (!onChooseSuperEvent) return;
    setBusy(true);
    setError(null);
    try {
      const rows = (await superEventsApi.list(projectId)).super_events;
      const number = Math.max(0, ...rows.map((row) => row.number || 0)) + 1;
      const created = await superEventsApi.create(projectId, {
        number,
        title_en: "New Super Event",
        display_name_locs: { en: "New Super Event" },
      });
      await superEvents.mutate();
      chooseSuper(created);
      toast.success(
        t("superCreated", { number: String(number).padStart(4, "0") }),
      );
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setBusy(false);
    }
  };

  const create = async () => {
    if (!current) return;
    setBusy(true);
    setError(null);
    try {
      const row = await eventsApi.findByScriptId(projectId, current);
      if (!row) throw new Error(t("notInProject"));
      const namespace = row.event_id.split(".")[0];
      const siblings = await eventsApi.list(projectId, row.namespace_id);
      const number =
        Math.max(0, ...siblings.map((e) => e.event_number || 0)) + 1;
      const eventId = `${namespace}.${number}`;

      const created = await eventsApi.create(projectId, {
        namespace_id: row.namespace_id,
        event_id: eventId,
        event_number: number,
        event_type: row.event_type,
        title: "",
        description: "",
        display_name_locs: { en: "New Event" },
        description_locs: { en: "" },
        picture: "GFX_report_event_001",
        is_triggered_only: true,
        options: [
          {
            name: newOptionKey(eventId, []),
            trigger: "",
            effects: "",
            hidden_effects: "",
            ai_chance: {},
            display_name_locs: { en: "OK" },
          },
        ],
      });
      choose(created.event_id);
      toast.success(t("created", { id: created.event_id }));
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
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
        {currentSuperEvent ? (
          <span className="min-w-0">
            <span className="block text-micro text-muted-foreground">
              {t("superEvent")}
            </span>
            <span className="block truncate text-sm">
              {superEventLabel(currentSuperEvent)}
            </span>
          </span>
        ) : (
          <span className="min-w-0">
            <span className="block text-micro text-muted-foreground">
              {t("event")}
            </span>

            <span className="block truncate text-sm">
              <span className="font-mono">{current ?? "—"}</span>
              {currentTitle && (
                <span className="text-muted-foreground"> · {currentTitle}</span>
              )}
            </span>
          </span>
        )}
        <span className="text-muted-foreground">{open ? "▴" : "▾"}</span>
      </button>
      {open && (
        <div className="mt-3 space-y-3">
          <input
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder={t("search")}
            className="w-full rounded border border-input bg-background px-2 py-1 text-sm"
          />
          {tag && (
            <div
              role="group"
              aria-label={t("scopeLabel")}
              className="inline-flex rounded border border-border p-0.5"
            >
              {(
                [
                  ["country", t("countryEvents", { tag })],
                  ["all", t("allEvents")],
                ] as const
              ).map(([value, label]) => (
                <button
                  key={value}
                  type="button"
                  aria-pressed={scope === value}
                  onClick={() => setScope(value)}
                  className={`ui-interactive rounded px-2 py-0.5 text-micro ${
                    scope === value
                      ? "bg-primary text-primary-foreground"
                      : "text-muted-foreground hover:bg-muted"
                  }`}
                >
                  {label}
                </button>
              ))}
            </div>
          )}
          {choiceTotal > 0 && (
            <p className="text-micro text-muted-foreground">
              {choiceList.length < choiceTotal
                ? t("totalPartial", {
                    total: choiceTotal,
                    shown: choiceList.length,
                  })
                : t("totalAll", { total: choiceTotal })}
            </p>
          )}
          <ul
            className="space-y-1 overflow-y-auto"

            style={{
              maxHeight: "60vh",
              scrollbarWidth: "auto",
              scrollbarColor: "var(--muted-foreground) transparent",
            }}
          >
            {choiceList.map((choice, i) => (
              <li
                key={choice.id}
                style={{
                  contentVisibility: "auto",
                  containIntrinsicSize: "auto 40px",
                }}
              >
                {choice.group && choice.group !== choiceList[i - 1]?.group && (
                  <div className="px-2 pb-1 pt-2 text-micro font-medium text-muted-foreground">
                    {choice.group === "country"
                      ? t("countryEvents", { tag: tag ?? "" })
                      : t("genericEvents")}
                  </div>
                )}
                <button
                  type="button"
                  onClick={() => choose(choice.id)}
                  className={`w-full rounded px-2 py-1 text-left text-micro ${
                    choice.id === current && !currentSuperEvent
                      ? "bg-primary text-primary-foreground"
                      : "hover:bg-muted"
                  }`}
                >
                  <span className="block truncate font-mono">{choice.id}</span>
                  {choice.title && (
                    <span className="block truncate opacity-80">
                      {choice.title}
                    </span>
                  )}
                </button>
              </li>
            ))}
            {choices.data && !choiceList.length && (
              <li className="text-micro text-muted-foreground">
                {listScope === "country" && !q
                  ? t("noCountryEvents")
                  : t("noMatch")}
              </li>
            )}
          </ul>
          {!browseOnly && (
            <>
              <button
                type="button"
                disabled={busy || !current}
                onClick={() => void create()}
                className="rounded bg-primary px-3 py-1 text-micro text-primary-foreground hover:opacity-90 disabled:cursor-not-allowed disabled:opacity-50"
              >
                {t("newEvent")}
              </button>
              <p className="text-micro text-muted-foreground">
                {t("newEventNote")}
              </p>
            </>
          )}
          {onChooseSuperEvent && (
            <div className="space-y-2 border-t border-border pt-3">
              <div className="text-micro text-muted-foreground">
                {t("superEvent")}
              </div>
              <ul className="max-h-40 space-y-1 overflow-y-auto">
                {superMatches.map((row) => (
                  <li key={row.id}>
                    <button
                      type="button"
                      aria-label={t("superAria", {
                        label: superEventLabel(row),
                      })}
                      onClick={() => chooseSuper(row)}
                      className={`w-full truncate rounded px-2 py-1 text-left text-micro ${
                        row.id === currentSuperEvent?.id
                          ? "bg-primary text-primary-foreground"
                          : "hover:bg-muted"
                      }`}
                    >
                      {superEventLabel(row)}
                    </button>
                  </li>
                ))}
                {superEvents.data && !superMatches.length && (
                  <li className="text-micro text-muted-foreground">
                    {t("noSuperEvents")}
                  </li>
                )}
              </ul>
              {!browseOnly && (
                <button
                  type="button"
                  disabled={busy}
                  onClick={() => void createSuper()}
                  className="rounded bg-primary px-3 py-1 text-micro text-primary-foreground hover:opacity-90 disabled:cursor-not-allowed disabled:opacity-50"
                >
                  {t("newSuperEvent")}
                </button>
              )}
            </div>
          )}
          {error && <p className="text-micro text-destructive">{error}</p>}
        </div>
      )}
    </div>
  );
}
