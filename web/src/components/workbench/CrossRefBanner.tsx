"use client";

import { useState, useMemo } from "react";
import { useTranslations } from "use-intl";
import { ChevronDown, ChevronRight, Info, Loader2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { InfoTooltip, TooltipProvider } from "@/components/ui/tooltip";
import type { CrossRefReport } from "@/lib/api";

const UNMODELED_CATEGORY_ORDER = [
  "scripted_effect_trigger",
  "ideology",
  "cosmetic_tag",
  "game_rule",
  "equipment",
  "special_project",
] as const;

interface CrossRefBannerProps {
  report: CrossRefReport;

  existingCountryTags: string[];

  onAddCountryAndReImport: (tag: string) => Promise<void>;

  disabled?: boolean;
}

const TOP_SUGGESTED = 5;
const REFS_FIRST_PAGE = 10;

export function CrossRefBanner({
  report,
  existingCountryTags,
  onAddCountryAndReImport,
  disabled,
}: CrossRefBannerProps) {
  const t = useTranslations("crossRefBanner");
  const [showDetails, setShowDetails] = useState(false);
  const [showScope, setShowScope] = useState(false);
  const [showUnmodeled, setShowUnmodeled] = useState(false);

  const [addingTag, setAddingTag] = useState<string | null>(null);
  const handleAdd = async (addTag: string) => {
    setAddingTag(addTag);
    try {
      await onAddCountryAndReImport(addTag);
    } finally {
      setAddingTag(null);
    }
  };

  const normalizedExisting = useMemo(
    () => new Set(existingCountryTags.map((ct) => ct.toUpperCase())),
    [existingCountryTags],
  );

  const topSuggestions = useMemo(
    () => report.suggested_countries.slice(0, TOP_SUGGESTED),
    [report.suggested_countries],
  );

  const brokenRefsSorted = useMemo(
    () => Object.entries(report.broken_refs).sort((a, b) => b[1] - a[1]),
    [report.broken_refs],
  );

  const unmodeledCategories = useMemo(() => {
    const refs = report.unmodeled_refs ?? {};
    return UNMODELED_CATEGORY_ORDER.filter((c) => (refs[c] ?? 0) > 0).map(
      (c) => ({ cat: c, count: refs[c] as number }),
    );
  }, [report.unmodeled_refs]);

  const totalUnmodeled = report.total_unmodeled_calls ?? 0;
  const hasBroken = report.total_broken_calls > 0;
  const hasUnmodeled = totalUnmodeled > 0;

  if (!hasBroken && !hasUnmodeled) return null;

  return (
    <TooltipProvider>
      <div className="space-y-2">
        {hasBroken && (
          <div className="rounded-md border border-border bg-muted/40 px-3 py-2 text-sm space-y-2">
            <div className="flex items-start gap-2">
              <Info className="h-4 w-4 text-muted-foreground shrink-0 mt-0.5" />
              <div className="flex-1 min-w-0">
                <p className="font-medium flex items-center gap-1">
                  {t.rich("headline", {
                    count: report.total_broken_calls,
                    mono: (chunks) => (
                      <span className="font-mono">{chunks}</span>
                    ),
                  })}
                  <InfoTooltip
                    title={t("tooltipTitle")}
                    content={t.raw("tooltipContent")}
                  />
                </p>
                <p className="text-xs text-muted-foreground">
                  {t.rich("reason", {
                    b: (chunks) => <strong>{chunks}</strong>,
                  })}
                </p>
              </div>
            </div>

            {topSuggestions.length > 0 && (
              <div className="space-y-1.5 pt-1">
                <p className="text-xs font-medium">{t("suggestLabel")}</p>
                <div className="flex flex-wrap gap-2">
                  {topSuggestions.map(({ tag, hits }) => {
                    const already = normalizedExisting.has(tag.toUpperCase());
                    const isAdding = addingTag === tag;
                    return (
                      <Button
                        key={tag}
                        size="sm"
                        variant={already ? "ghost" : "outline"}

                        disabled={disabled || already || addingTag !== null}
                        onClick={() => handleAdd(tag)}
                        className="h-7 text-xs"
                        title={
                          already
                            ? t("alreadyInProject", { tag })
                            : t("expandTitle", { tag })
                        }
                      >
                        {isAdding ? (
                          <>
                            <Loader2 className="h-3 w-3 animate-spin mr-1" />
                            <span>{t("addingCountry", { tag })}</span>
                          </>
                        ) : already ? (
                          <>
                            <span className="font-mono">{tag}</span>
                            <span className="text-muted-foreground ml-1">
                              {t("added")}
                            </span>
                          </>
                        ) : (
                          <>
                            <span>{t("expandAddCountry")}</span>
                            <span className="font-mono mx-1">{tag}</span>
                            <span className="text-muted-foreground">
                              ({hits})
                            </span>
                          </>
                        )}
                      </Button>
                    );
                  })}
                </div>
              </div>
            )}

            <div className="flex items-center gap-3 pt-1">
              <button
                type="button"
                onClick={() => setShowDetails((v) => !v)}
                className="flex items-center gap-1 text-xs text-muted-foreground hover:underline"
              >
                {showDetails ? (
                  <ChevronDown className="h-3 w-3" />
                ) : (
                  <ChevronRight className="h-3 w-3" />
                )}
                {showDetails
                  ? t("collapseDetails")
                  : t("viewDetails", { count: brokenRefsSorted.length })}
              </button>
              <button
                type="button"
                onClick={() => setShowScope((v) => !v)}
                className="text-xs text-muted-foreground hover:underline"
              >
                {showScope ? t("hideScope") : t("showScope")}
              </button>
            </div>

            {showScope && (
              <div className="rounded border border-border bg-muted/40 px-2 py-1.5 text-xs text-muted-foreground">
                <p className="font-medium mb-0.5">{t("currentScope")}</p>
                <p className="break-all">
                  {report.active_batch.length > 0
                    ? report.active_batch.join(" / ")
                    : t("none")}
                </p>
                <p className="mt-1">{t("scopeNote")}</p>
              </div>
            )}

            {showDetails && (
              <div className="rounded border border-border bg-background px-2 py-1.5 space-y-1 max-h-56 overflow-y-auto">
                {brokenRefsSorted
                  .slice(0, REFS_FIRST_PAGE)
                  .map(([key, count]) => {
                    const samples = report.sample_ids[key] ?? [];
                    return (
                      <div key={key} className="text-xs">
                        <p className="font-mono">
                          <span className="font-medium">{key}</span>
                          <span className="text-muted-foreground">
                            : {count}
                          </span>
                        </p>
                        {samples.length > 0 && (
                          <p className="pl-3 text-muted-foreground">
                            {t("egPrefix")}
                            {samples.slice(0, 3).join(", ")}
                            {samples.length > 3
                              ? ` (+${samples.length - 3})`
                              : ""}
                          </p>
                        )}
                      </div>
                    );
                  })}
                {brokenRefsSorted.length > REFS_FIRST_PAGE && (
                  <p className="text-xs text-muted-foreground pt-1 border-t border-border">
                    {t("moreHidden", {
                      hidden: brokenRefsSorted.length - REFS_FIRST_PAGE,
                      total: brokenRefsSorted.length,
                    })}
                  </p>
                )}
              </div>
            )}
          </div>
        )}

        {hasUnmodeled && (
          <div className="rounded-md border border-border bg-muted/40 px-3 py-2 text-sm space-y-2">
            <div className="flex items-start gap-2">
              <Info className="h-4 w-4 text-muted-foreground shrink-0 mt-0.5" />
              <div className="flex-1 min-w-0">
                <p className="font-medium flex items-center gap-1">
                  {t.rich("unmodeled.headline", {
                    count: totalUnmodeled,
                    mono: (chunks) => (
                      <span className="font-mono">{chunks}</span>
                    ),
                  })}
                  <InfoTooltip
                    title={t("unmodeled.tooltipTitle")}
                    content={t.raw("unmodeled.tooltipContent")}
                  />
                </p>
                <p className="text-xs text-muted-foreground">
                  {t.rich("unmodeled.note", {
                    b: (chunks) => <strong>{chunks}</strong>,
                  })}
                </p>
              </div>
            </div>

            <div className="flex items-center gap-3 pt-1">
              <button
                type="button"
                onClick={() => setShowUnmodeled((v) => !v)}
                className="flex items-center gap-1 text-xs text-muted-foreground hover:underline"
              >
                {showUnmodeled ? (
                  <ChevronDown className="h-3 w-3" />
                ) : (
                  <ChevronRight className="h-3 w-3" />
                )}
                {showUnmodeled
                  ? t("unmodeled.collapseDetails")
                  : t("unmodeled.viewDetails", {
                      count: unmodeledCategories.length,
                    })}
              </button>
            </div>

            {showUnmodeled && (
              <div className="rounded border border-border bg-background px-2 py-1.5 space-y-1 max-h-56 overflow-y-auto">
                {unmodeledCategories.map(({ cat, count }) => {
                  const samples = report.unmodeled_sample_ids?.[cat] ?? [];
                  return (
                    <div key={cat} className="text-xs">
                      <p>
                        <span className="font-medium">
                          {t(`unmodeled.cat.${cat}`)}
                        </span>
                        <span className="text-muted-foreground">: {count}</span>
                      </p>
                      {samples.length > 0 && (
                        <p className="pl-3 text-muted-foreground font-mono">
                          {t("egPrefix")}
                          {samples.slice(0, 3).join(", ")}
                          {samples.length > 3
                            ? ` (+${samples.length - 3})`
                            : ""}
                        </p>
                      )}
                    </div>
                  );
                })}
              </div>
            )}
          </div>
        )}
      </div>
    </TooltipProvider>
  );
}
