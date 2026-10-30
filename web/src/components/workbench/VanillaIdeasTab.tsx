"use client";

import { useEffect, useMemo, useState } from "react";
import { useTranslations } from "use-intl";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import {
  API_BASE,
  vanillaIdeasApi,
  type VanillaIdeaEntry,
  type VanillaIdeaListResponse,
} from "@/lib/api";

const PAGE_SIZE = 50;
const CATEGORY_FILTERS: { labelKey: string; value: string }[] = [
  { labelKey: "filterAll", value: "" },
  { labelKey: "filterCountry", value: "country" },
  { labelKey: "filterHidden", value: "hidden_ideas" },
  { labelKey: "filterAdvisor", value: "political_advisor" },
  { labelKey: "filterHighCmd", value: "high_command" },
];

interface VanillaIdeasTabProps {
  onUseAsTemplate?: (entry: VanillaIdeaEntry) => void;

  canUseAsTemplate?: boolean;

  useAsTemplateDisabledHint?: string;
}

export function VanillaIdeasTab({
  onUseAsTemplate,
  canUseAsTemplate = false,
  useAsTemplateDisabledHint,
}: VanillaIdeasTabProps = {}) {
  const t = useTranslations("vanillaBrowser");

  const disabledHint = useAsTemplateDisabledHint ?? t("pickCategoryFirst");
  const [q, setQ] = useState("");
  const [categoryFilter, setCategoryFilter] = useState("");
  const [picturedOnly, setPicturedOnly] = useState(false);
  const [offset, setOffset] = useState(0);
  const [data, setData] = useState<VanillaIdeaListResponse | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const [selected, setSelected] = useState<VanillaIdeaEntry | null>(null);

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
  const setFilterCategory = (v: string) => {
    setCategoryFilter(v);
    setOffset(0);
  };
  const togglePicturedOnly = () => {
    setPicturedOnly((prev) => !prev);
    setOffset(0);
  };

  useEffect(() => {
    let cancelled = false;

    Promise.resolve().then(() => {
      if (cancelled) return;
      setLoading(true);
      setError(null);
    });
    vanillaIdeasApi
      .list({
        q,
        category: categoryFilter || undefined,
        has_picture: picturedOnly ? true : undefined,
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
  }, [q, categoryFilter, picturedOnly, offset]);

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

  const pictureUrl = (picture: string | null) =>
    picture
      ? `${API_BASE}/go/vanilla/focus-icon/${encodeURIComponent(picture.startsWith("GFX_") ? picture : `GFX_idea_${picture}`)}`
      : null;

  return (
    <div className="flex flex-col h-full overflow-hidden">
      <div className="shrink-0 px-3 py-2 border-b border-border space-y-2">
        <div className="flex items-center gap-2">
          <span className="text-xs text-muted-foreground">
            {meta?.total
              ? t("metaIdeas", {
                  total: meta.total,
                  categories: meta.categories,
                })
              : t("indexFallbackIdeas")}
          </span>
        </div>
        <Input
          value={q}
          onChange={(e) => setSearchQuery(e.target.value)}
          placeholder={t("searchIdeasPlaceholder")}
          className="h-8 text-xs"
        />
        <div className="flex items-center gap-1 flex-wrap">
          {CATEGORY_FILTERS.map((f) => (
            <Button
              key={f.value || "all"}
              size="sm"
              variant={categoryFilter === f.value ? "default" : "outline"}
              className="h-6 px-2 text-xs"
              onClick={() => setFilterCategory(f.value)}
            >
              {t(f.labelKey)}
            </Button>
          ))}
          <Button
            size="sm"
            variant={picturedOnly ? "default" : "outline"}
            className="h-6 px-2 text-xs"
            onClick={togglePicturedOnly}
            title={t("withPictureTooltip")}
          >
            {t("withPicture")}
          </Button>
        </div>
      </div>

      <div className="flex-1 min-h-0 overflow-auto">
        {data?._warning && (
          <div className="m-3 rounded-md border border-warning/40 bg-warning/10 p-2 text-xs leading-snug text-warning">
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
            {t("noIdeasMatch")}
          </div>
        ) : (
          <ul className="divide-y divide-border/40">
            {items.map((ide) => {
              const key = `${ide.category}/${ide.idea_id}`;
              const isActive =
                selected &&
                selected.category === ide.category &&
                selected.idea_id === ide.idea_id;
              const thumb = pictureUrl(ide.picture);
              return (
                <li
                  key={key}
                  className={`group px-3 py-1.5 cursor-pointer ui-interactive text-xs ${
                    isActive
                      ? "bg-accent/10 text-accent shadow-[inset_3px_0_0_0] shadow-accent"
                      : "hover:bg-muted"
                  }`}
                  onClick={() => setSelected(ide)}
                  title={`${ide.file}:${ide.line}`}
                >
                  <div className="flex items-center gap-2">
                    {thumb ? (
                      <img
                        src={thumb}
                        alt={ide.picture || ""}
                        className="h-8 w-8 rounded border border-border/50 object-cover shrink-0"
                        onError={(e) => {
                          (e.target as HTMLImageElement).style.visibility =
                            "hidden";
                        }}
                      />
                    ) : (
                      <span className="h-8 w-8 rounded border border-dashed border-border/50 shrink-0 grid place-items-center text-muted-foreground text-micro">
                        {t("noPic")}
                      </span>
                    )}
                    <div className="min-w-0 flex-1">
                      <div className="flex items-center justify-between gap-2">
                        <span className="font-mono truncate">
                          {ide.idea_id}
                        </span>
                        <div className="flex items-center gap-1.5 shrink-0">
                          <button
                            type="button"
                            onClick={(e) => {
                              e.stopPropagation();
                              copyToClipboard(ide.idea_id);
                            }}
                            className="opacity-0 group-hover:opacity-100 text-muted-foreground hover:text-foreground ui-interactive"
                            aria-label={t("copy", { id: ide.idea_id })}
                            title={
                              copiedId === ide.idea_id
                                ? t("copied")
                                : t("copy", { id: ide.idea_id })
                            }
                          >
                            {copiedId === ide.idea_id ? "✓" : "⧉"}
                          </button>
                          <span className="text-muted-foreground">
                            {ide.category}
                          </span>
                        </div>
                      </div>
                      {ide.picture && (
                        <div className="mt-0.5 text-muted-foreground truncate text-micro font-mono">
                          {ide.picture}
                        </div>
                      )}
                    </div>
                  </div>
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
              {selected.category} / {selected.idea_id}
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
          <div className="flex-1 min-h-0 overflow-auto p-3 text-xs space-y-2">
            {selected.picture ? (
              <div className="flex items-start gap-3">
                <img
                  src={pictureUrl(selected.picture) || ""}
                  alt={selected.picture}
                  className="h-16 w-16 rounded border border-border object-cover shrink-0"
                />
                <div className="space-y-1 min-w-0">
                  <p className="font-mono text-muted-foreground truncate">
                    picture ={" "}
                    <span className="text-foreground">{selected.picture}</span>
                  </p>
                  <p className="text-muted-foreground">
                    {t.raw("pictureServedVia")}
                  </p>
                </div>
              </div>
            ) : (
              <p className="text-muted-foreground italic">
                {t("noPictureField")}
              </p>
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
                onClick={() => copyToClipboard(selected.idea_id)}
                title={
                  copiedId === selected.idea_id
                    ? t("copied")
                    : t("copy", { id: selected.idea_id })
                }
              >
                {copiedId === selected.idea_id ? t("copiedShort") : t("copyId")}
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
                      ? t("useAsTemplateIdeaTooltip", { id: selected.idea_id })
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
