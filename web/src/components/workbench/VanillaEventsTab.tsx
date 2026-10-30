"use client";

import { useEffect, useMemo, useState } from "react";
import { useTranslations } from "use-intl";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import {
  vanillaEventsApi,
  type VanillaEventEntry,
  type VanillaEventListResponse,
} from "@/lib/api";

const PAGE_SIZE = 50;
const EVENT_TYPE_FILTERS: { labelKey: string; value: string }[] = [
  { labelKey: "filterAll", value: "" },
  { labelKey: "filterCountry", value: "country_event" },
  { labelKey: "filterNews", value: "news_event" },
  { labelKey: "filterState", value: "state_event" },
];

interface VanillaEventsTabProps {
  onUseAsTemplate?: (entry: VanillaEventEntry) => void;

  canUseAsTemplate?: boolean;

  useAsTemplateDisabledHint?: string;
}

export function VanillaEventsTab({
  onUseAsTemplate,
  canUseAsTemplate = false,
  useAsTemplateDisabledHint,
}: VanillaEventsTabProps = {}) {
  const t = useTranslations("vanillaBrowser");

  const disabledHint = useAsTemplateDisabledHint ?? t("selectNamespaceFirst");
  const [q, setQ] = useState("");
  const [typeFilter, setTypeFilter] = useState("");
  const [offset, setOffset] = useState(0);
  const [data, setData] = useState<VanillaEventListResponse | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const [selected, setSelected] = useState<VanillaEventEntry | null>(null);
  const [detailBody, setDetailBody] = useState<string>("");
  const [detailLoading, setDetailLoading] = useState(false);

  const [copiedId, setCopiedId] = useState<string | null>(null);

  const copyToClipboard = async (id: string) => {
    try {
      if (typeof navigator !== "undefined" && navigator.clipboard) {
        await navigator.clipboard.writeText(id);
      } else {
        const ta = document.createElement("textarea");
        ta.value = id;
        document.body.appendChild(ta);
        ta.select();
        document.execCommand("copy");
        document.body.removeChild(ta);
      }
      setCopiedId(id);
      window.setTimeout(() => {
        setCopiedId((cur) => (cur === id ? null : cur));
      }, 900);
    } catch {}
  };

  const setSearchQuery = (v: string) => {
    setQ(v);
    setOffset(0);
  };
  const setFilterType = (v: string) => {
    setTypeFilter(v);
    setOffset(0);
  };

  useEffect(() => {
    let cancelled = false;

    Promise.resolve().then(() => {
      if (cancelled) return;
      setLoading(true);
      setError(null);
    });
    vanillaEventsApi
      .list({
        q,
        event_type: typeFilter || undefined,
        limit: PAGE_SIZE,
        offset,
      })
      .then((r) => {
        if (cancelled) return;
        setData(r);
      })
      .catch((e) => {
        if (cancelled) return;
        setError(e instanceof Error ? e.message : String(e));
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [q, typeFilter, offset]);

  useEffect(() => {
    let cancelled = false;
    (async () => {
      if (!selected) {
        if (!cancelled) setDetailBody("");
        return;
      }
      const [ns, numStr] = selected.id.split(".");
      const num = Number(numStr);
      if (!ns || Number.isNaN(num)) return;
      if (cancelled) return;
      setDetailLoading(true);
      try {
        const r = await vanillaEventsApi.detail(ns, num);
        if (!cancelled) setDetailBody(r.body || "");
      } catch {
        if (!cancelled) setDetailBody("// failed to load body");
      } finally {
        if (!cancelled) setDetailLoading(false);
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [selected]);

  const total = data?.total_matching ?? 0;
  const items = data?.items ?? [];
  const meta = data?.meta;
  const hasPrev = offset > 0;
  const hasNext = offset + PAGE_SIZE < total;

  const pageInfo = useMemo(() => {
    if (total === 0) return t("pageEmpty");
    const start = offset + 1;
    const end = Math.min(offset + PAGE_SIZE, total);
    return t("pageRange", { start, end, total });
  }, [offset, total, t]);

  return (
    <div className="flex flex-col h-full overflow-hidden">
      <div className="shrink-0 px-3 py-2 border-b border-border space-y-2">
        <div className="flex items-center gap-2">
          <span className="text-xs text-muted-foreground">
            {meta?.total
              ? t("metaEvents", {
                  total: meta.total,
                  namespaces: meta.namespaces,
                })
              : t("indexFallbackEvents")}
          </span>
        </div>
        <Input
          value={q}
          onChange={(e) => setSearchQuery(e.target.value)}
          placeholder={t("searchEventsPlaceholder")}
          className="h-8 text-xs"
        />
        <div className="flex items-center gap-1 flex-wrap">
          {EVENT_TYPE_FILTERS.map((f) => (
            <Button
              key={f.value || "all"}
              size="sm"
              variant={typeFilter === f.value ? "default" : "outline"}
              className="h-6 px-2 text-xs"
              onClick={() => setFilterType(f.value)}
            >
              {t(f.labelKey)}
            </Button>
          ))}
        </div>
      </div>

      <div className="flex-1 min-h-0 overflow-auto">
        {data?._warning && (
          <div className="m-3 rounded-md border border-warning/40 bg-warning/10 p-2 text-xs text-warning leading-snug">
            {data._warning}
          </div>
        )}
        {loading && !data ? (
          <div className="p-6 text-center text-xs text-muted-foreground">
            {t("loading")}
          </div>
        ) : error ? (
          <div className="m-3 rounded-md border border-destructive/40 bg-destructive/5 p-2 text-xs">
            {error}
          </div>
        ) : items.length === 0 ? (
          <div className="p-6 text-center text-xs text-muted-foreground italic">
            {t("noEventsMatch")}
          </div>
        ) : (
          <ul className="divide-y divide-border/40">
            {items.map((ev) => {
              const isActive = selected?.id === ev.id;
              return (
                <li
                  key={ev.id}
                  className={`group px-3 py-1.5 cursor-pointer ui-interactive text-xs ${
                    isActive
                      ? "bg-accent/10 text-accent shadow-[inset_3px_0_0_0] shadow-accent"
                      : "hover:bg-muted"
                  }`}
                  onClick={() => setSelected(ev)}
                  title={`${ev.file}:${ev.line}`}
                >
                  <div className="flex items-center justify-between gap-2">
                    <span className="font-mono truncate">{ev.id}</span>
                    <div className="flex items-center gap-1.5 shrink-0">
                      <button
                        type="button"
                        onClick={(e) => {
                          e.stopPropagation();
                          copyToClipboard(ev.id);
                        }}
                        className="opacity-0 group-hover:opacity-100 text-muted-foreground hover:text-foreground ui-interactive"
                        aria-label={t("copy", { id: ev.id })}
                        title={
                          copiedId === ev.id
                            ? t("copied")
                            : t("copy", { id: ev.id })
                        }
                      >
                        {copiedId === ev.id ? "✓" : "⧉"}
                      </button>
                      <span className="text-muted-foreground">
                        {ev.type.replace("_event", "")}
                      </span>
                    </div>
                  </div>
                  {(ev.title || ev.picture) && (
                    <div className="mt-0.5 text-muted-foreground truncate text-micro">
                      {ev.title || ev.picture}
                    </div>
                  )}
                </li>
              );
            })}
          </ul>
        )}
      </div>

      <div className="shrink-0 px-3 py-2 border-t border-border flex items-center justify-between text-xs">
        <span className="text-muted-foreground">{pageInfo}</span>
        <div className="flex items-center gap-1">
          <Button
            size="sm"
            variant="outline"
            className="h-7 px-2 text-xs"
            disabled={!hasPrev || loading}
            onClick={() => setOffset(Math.max(0, offset - PAGE_SIZE))}
          >
            {t("prev")}
          </Button>
          <Button
            size="sm"
            variant="outline"
            className="h-7 px-2 text-xs"
            disabled={!hasNext || loading}
            onClick={() => setOffset(offset + PAGE_SIZE)}
          >
            {t("next")}
          </Button>
        </div>
      </div>

      {selected && (
        <div className="shrink-0 max-h-[40%] border-t border-border bg-card flex flex-col">
          <div className="shrink-0 px-3 py-1.5 border-b border-border flex items-center justify-between text-xs">
            <span className="font-mono text-foreground truncate">
              {selected.id}
            </span>
            <button
              type="button"
              onClick={() => setSelected(null)}
              className="text-muted-foreground hover:text-foreground ui-interactive"
              aria-label={t("closePreview")}
              title={t("closePreview")}
            >
              ×
            </button>
          </div>
          <div className="flex-1 min-h-0 overflow-auto p-2 text-xs">
            {detailLoading ? (
              <span className="text-muted-foreground">{t("loading")}</span>
            ) : (
              <pre className="font-mono whitespace-pre-wrap text-foreground/90 leading-snug">
                {detailBody || "// (empty)"}
              </pre>
            )}
          </div>
          <div className="shrink-0 px-3 py-1.5 border-t border-border flex items-center justify-between gap-2 text-micro">
            <span className="text-muted-foreground truncate">
              {t("readonlyVanilla", {
                file: selected.file,
                line: selected.line,
              })}
            </span>
            <div className="flex items-center gap-1 shrink-0">
              <Button
                size="sm"
                variant="ghost"
                className="h-6 px-2 text-xs"
                onClick={() => copyToClipboard(selected.id)}
                title={
                  copiedId === selected.id
                    ? t("copied")
                    : t("copy", { id: selected.id })
                }
              >
                {copiedId === selected.id ? t("copiedShort") : t("copyId")}
              </Button>
              {onUseAsTemplate && (
                <Button
                  size="sm"
                  variant="default"
                  className="h-6 px-2 text-xs"
                  disabled={!canUseAsTemplate}
                  onClick={() => onUseAsTemplate(selected)}
                  title={
                    canUseAsTemplate
                      ? t("useAsTemplateEventTooltip", { id: selected.id })
                      : disabledHint
                  }
                >
                  {t("useAsTemplate")}
                </Button>
              )}
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
