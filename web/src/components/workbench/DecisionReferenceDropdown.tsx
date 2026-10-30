"use client";

import { useState, useEffect, useMemo, useRef } from "react";
import { useTranslations } from "use-intl";
import useSWR from "swr";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import type { Decision, DecisionCategory, BaseModRef } from "@/types";
import { decisionsApi, basemodIndexApi } from "@/lib/api";
import { useContentLocaleStore } from "@/lib/store/contentLocaleStore";
import { localizedName } from "@/lib/i18n/localizedName";

type EffectType = "activate_decision" | "remove_decision" | "has_decision";

interface DecisionReferenceDropdownProps {
  projectId: number;
  effectType: EffectType;
  onSelect: (code: string) => void;
  buttonLabel: string;
}

interface DisplayItem {
  decision_id: string;
  source: "project" | "base";

  category: string;
  name_en: string;
  name_zh: string;

  dependencyName?: string;
}

const BASE_SEARCH_LIMIT = 25;

export function DecisionReferenceDropdown({
  projectId,
  effectType,
  onSelect,
  buttonLabel,
}: DecisionReferenceDropdownProps) {
  const [open, setOpen] = useState(false);
  const [search, setSearch] = useState("");

  const [debouncedSearch, setDebouncedSearch] = useState("");
  const [activeCat, setActiveCat] = useState<number | null>(null);
  const dropdownRef = useRef<HTMLDivElement>(null);
  const t = useTranslations("referenceDropdown");
  const entity = t("entityDecision");
  const locale = useContentLocaleStore((s) => s.locale);

  const { data: categories = [] } = useSWR<DecisionCategory[]>(
    open && projectId ? ["decisions/dropdown-cats", projectId] : null,
    () => decisionsApi.listCategories(projectId),
    { keepPreviousData: true },
  );
  const { data: decisions = [] } = useSWR<Decision[]>(
    open && projectId ? ["decisions/dropdown-all", projectId] : null,
    () => decisionsApi.list(projectId),
    { keepPreviousData: true },
  );

  useEffect(() => {
    const timer = setTimeout(() => setDebouncedSearch(search.trim()), 250);
    return () => clearTimeout(timer);
  }, [search]);

  const { data: baseResult, isLoading: baseLoading } = useSWR(
    open && projectId
      ? ["basemod-refs/decision", projectId, debouncedSearch]
      : null,
    () =>
      basemodIndexApi.searchRefs(
        projectId,
        "decision",
        debouncedSearch,
        BASE_SEARCH_LIMIT,
      ),
    { keepPreviousData: true },
  );
  useEffect(() => {
    if (!open) return;
    const handler = (e: MouseEvent) => {
      if (
        dropdownRef.current &&
        !dropdownRef.current.contains(e.target as Node)
      ) {
        setOpen(false);
      }
    };
    document.addEventListener("mousedown", handler);
    return () => document.removeEventListener("mousedown", handler);
  }, [open]);

  const items = useMemo((): DisplayItem[] => {
    const q = search.toLowerCase().trim();
    let filtered = decisions;
    const baseRefs: BaseModRef[] = baseResult?.results ?? [];

    if (q) {
      filtered = decisions.filter(
        (d) =>
          d.decision_id.toLowerCase().includes(q) ||
          (d.name_en || "").toLowerCase().includes(q),
      );
    } else if (activeCat !== null) {
      filtered = decisions.filter((d) => d.category_id === activeCat);
    }

    const results: DisplayItem[] = filtered.map((d) => ({
      decision_id: d.decision_id,
      source: "project" as const,
      category:
        categories.find((c) => c.id === d.category_id)?.category_id || "",
      name_en: d.name_en || "",
      name_zh: d.name_zh || "",
    }));

    if (q || activeCat === null) {
      for (const ref of baseRefs) {
        if (results.some((r) => r.decision_id === ref.id)) continue;
        results.push({
          decision_id: ref.id,
          source: "base",
          category: "",
          name_en: ref.display_name,
          name_zh: ref.display_name,
          dependencyName: ref.dependency_name,
        });
      }
    }

    return results;
  }, [search, decisions, categories, activeCat, baseResult]);

  const handleSelect = (decisionId: string) => {
    const code = `${effectType} = ${decisionId}`;
    onSelect(code);
    setOpen(false);
    setSearch("");
  };

  const basePending = open && baseLoading && !baseResult;

  return (
    <div className="relative inline-block" ref={dropdownRef}>
      <Button
        size="sm"
        variant="outline"
        className="text-xs h-6 px-2"
        onClick={() => setOpen(!open)}
      >
        {buttonLabel} ▾
      </Button>

      {open && (
        <div className="absolute z-50 top-7 left-0 w-80 bg-popover border rounded-md shadow-lg">
          <div className="p-2 border-b">
            <Input
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              placeholder={t("searchPlaceholder", { entity })}
              className="text-xs h-7"
              autoFocus
            />
          </div>

          {!search && categories.length > 0 && (
            <div className="flex flex-wrap gap-0.5 p-1.5 border-b">
              <button
                className={`px-2 py-0.5 rounded text-xs transition-colors ${
                  activeCat === null
                    ? "bg-accent text-accent-foreground"
                    : "bg-muted hover:bg-muted/80"
                }`}
                onClick={() => setActiveCat(null)}
              >
                {t("all")}
              </button>
              {categories.map((cat) => (
                <button
                  key={cat.id}
                  className={`px-2 py-0.5 rounded text-xs transition-colors ${
                    activeCat === cat.id
                      ? "bg-accent text-accent-foreground"
                      : "bg-muted hover:bg-muted/80"
                  }`}
                  onClick={() => setActiveCat(cat.id)}
                >
                  {cat.category_id} ({cat.decision_count})
                </button>
              ))}
            </div>
          )}

          <div className="max-h-60 overflow-y-auto">
            {items.length === 0 ? (
              <p className="p-3 text-xs text-muted-foreground">
                {basePending
                  ? t("searchingBase")
                  : decisions.length === 0
                    ? t("emptyProject", { entity })
                    : t("noResults")}
              </p>
            ) : (
              <>
                {items.map((item) => (
                  <button
                    key={`${item.source}-${item.decision_id}`}
                    className="w-full text-left px-3 py-1.5 text-xs hover:bg-muted transition-colors flex items-center justify-between gap-2"
                    onClick={() => handleSelect(item.decision_id)}
                  >
                    <span className="flex flex-col min-w-0">
                      <span className="font-mono truncate">
                        {item.decision_id}
                      </span>
                      <span className="text-muted-foreground truncate">
                        {localizedName(
                          locale,
                          item.name_en,
                          item.name_zh,
                          item.category,
                        )}
                      </span>
                    </span>
                    <span className="ml-2 shrink-0 text-right">
                      {item.source === "project" ? (
                        <span className="text-success">
                          {t("projectBadge")}
                        </span>
                      ) : (
                        <span className="text-muted-foreground">
                          {item.dependencyName || t("baseBadge")}
                        </span>
                      )}
                    </span>
                  </button>
                ))}
                {basePending && (
                  <p className="p-2 text-xs text-muted-foreground">
                    {t("searchingBase")}
                  </p>
                )}
              </>
            )}
          </div>

          <div className="border-t p-2">
            <button
              className="w-full text-left px-2 py-1 text-xs text-muted-foreground hover:text-foreground"
              onClick={() => {
                const id = prompt(t("promptManual", { entity }));
                if (id) handleSelect(id.trim());
              }}
            >
              {t("enterManually")}
            </button>
          </div>
        </div>
      )}
    </div>
  );
}
