"use client";

import { useState, useEffect, useMemo, useRef } from "react";
import { useTranslations } from "use-intl";
import useSWR from "swr";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import type { FocusTree, BaseModRef } from "@/types";
import { focusTreesApi, basemodIndexApi } from "@/lib/api";
import { useContentLocaleStore } from "@/lib/store/contentLocaleStore";
import { localizedName } from "@/lib/i18n/localizedName";

type EffectType = "has_completed_focus" | "focus_progress";

interface FocusReferenceDropdownProps {
  projectId: number;
  effectType: EffectType;
  onSelect: (code: string) => void;
  buttonLabel: string;
}

interface DisplayItem {
  focus_id: string;
  source: "project" | "base";

  tree_id: string;
  name_en: string;
  name_zh: string;

  dependencyName?: string;
}

const BASE_SEARCH_LIMIT = 25;

export function FocusReferenceDropdown({
  projectId,
  effectType,
  onSelect,
  buttonLabel,
}: FocusReferenceDropdownProps) {
  const [open, setOpen] = useState(false);
  const [search, setSearch] = useState("");

  const [debouncedSearch, setDebouncedSearch] = useState("");
  const [activeTree, setActiveTree] = useState<number | null>(null);
  const dropdownRef = useRef<HTMLDivElement>(null);
  const t = useTranslations("referenceDropdown");
  const entity = t("entityFocus");
  const locale = useContentLocaleStore((s) => s.locale);

  const { data: trees = [] } = useSWR<FocusTree[]>(
    open && projectId ? ["focus-trees/dropdown", projectId] : null,
    () => focusTreesApi.list(projectId),
    { keepPreviousData: true },
  );

  useEffect(() => {
    const timer = setTimeout(() => setDebouncedSearch(search.trim()), 250);
    return () => clearTimeout(timer);
  }, [search]);

  const { data: baseResult, isLoading: baseLoading } = useSWR(
    open && projectId
      ? ["basemod-refs/focus", projectId, debouncedSearch]
      : null,
    () =>
      basemodIndexApi.searchRefs(
        projectId,
        "focus",
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
    const baseRefs: BaseModRef[] = baseResult?.results ?? [];
    const allNodes = trees.flatMap((t) =>
      (t.nodes || []).map((n) => ({
        focus_id: n.focus_id,
        tree_id: t.tree_id,
        tree_db_id: t.id,
        name_en: n.name_en || "",
        name_zh: n.name_zh || "",
      })),
    );

    let projectNodes: typeof allNodes;
    if (q) {
      projectNodes = allNodes.filter(
        (n) =>
          n.focus_id.toLowerCase().includes(q) ||
          n.name_en.toLowerCase().includes(q),
      );
    } else if (activeTree !== null) {
      projectNodes = allNodes.filter((n) => n.tree_db_id === activeTree);
    } else {
      projectNodes = allNodes;
    }

    const results: DisplayItem[] = projectNodes.map((n) => ({
      focus_id: n.focus_id,
      source: "project" as const,
      tree_id: n.tree_id,
      name_en: n.name_en,
      name_zh: n.name_zh,
    }));

    if (q || activeTree === null) {
      for (const ref of baseRefs) {
        if (results.some((r) => r.focus_id === ref.id)) continue;
        results.push({
          focus_id: ref.id,
          source: "base",
          tree_id: "",
          name_en: ref.display_name,
          name_zh: ref.display_name,
          dependencyName: ref.dependency_name,
        });
      }
    }

    return results;
  }, [search, trees, activeTree, baseResult]);

  const handleSelect = (focusId: string) => {
    let code: string;
    if (effectType === "focus_progress") {
      code = `focus_progress = { focus = ${focusId} progress > 0 }`;
    } else {
      code = `has_completed_focus = ${focusId}`;
    }
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

          {!search && trees.length > 0 && (
            <div className="flex flex-wrap gap-0.5 p-1.5 border-b">
              <button
                className={`px-2 py-0.5 rounded text-xs transition-colors ${
                  activeTree === null
                    ? "bg-primary text-primary-foreground"
                    : "bg-muted hover:bg-muted/80"
                }`}
                onClick={() => setActiveTree(null)}
              >
                {t("all")}
              </button>
              {trees.map((t) => (
                <button
                  key={t.id}
                  className={`px-2 py-0.5 rounded text-xs transition-colors ${
                    activeTree === t.id
                      ? "bg-primary text-primary-foreground"
                      : "bg-muted hover:bg-muted/80"
                  }`}
                  onClick={() => setActiveTree(t.id)}
                >
                  {t.tree_id} ({(t.nodes || []).length})
                </button>
              ))}
            </div>
          )}

          <div className="max-h-60 overflow-y-auto">
            {items.length === 0 ? (
              <p className="p-3 text-xs text-muted-foreground">
                {basePending
                  ? t("searchingBase")
                  : trees.length === 0
                    ? t("emptyProject", { entity })
                    : t("noResults")}
              </p>
            ) : (
              <>
                {items.map((item) => (
                  <button
                    key={`${item.source}-${item.focus_id}`}
                    className="w-full text-left px-3 py-1.5 text-xs hover:bg-muted transition-colors flex items-center justify-between gap-2"
                    onClick={() => handleSelect(item.focus_id)}
                  >
                    <span className="flex flex-col min-w-0">
                      <span className="font-mono truncate">
                        {item.focus_id}
                      </span>
                      <span className="text-muted-foreground truncate">
                        {localizedName(
                          locale,
                          item.name_en,
                          item.name_zh,
                          item.tree_id,
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
