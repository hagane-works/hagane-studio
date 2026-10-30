"use client";

import { useEffect, useRef, useState } from "react";
import { useTranslations } from "use-intl";
import { Search } from "lucide-react";
import { Input } from "@/components/ui/input";
import { Select } from "@/components/ui/select";
import { cn } from "@/lib/utils";
import {
  SORT_MODES,
  FILTER_MODES,
  type SortMode,
  type FilterMode,
} from "@/lib/hooks/useDashboardFilters";

interface DashboardToolbarProps {
  q: string;
  sort: SortMode;
  filter: FilterMode;
  onQChange: (next: string) => void;
  onSortChange: (next: SortMode) => void;
  onFilterChange: (next: FilterMode) => void;

  hidden?: boolean;
}

const DEBOUNCE_MS = 200;

export function DashboardToolbar({
  q,
  sort,
  filter,
  onQChange,
  onSortChange,
  onFilterChange,
  hidden,
}: DashboardToolbarProps) {
  const t = useTranslations("landing");

  const [localQ, setLocalQ] = useState(q);
  const debounceRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  const [prevQ, setPrevQ] = useState(q);
  if (q !== prevQ) {
    setPrevQ(q);
    if (q !== localQ) setLocalQ(q);
  }

  useEffect(() => {
    if (localQ === q) return;
    if (debounceRef.current) clearTimeout(debounceRef.current);
    debounceRef.current = setTimeout(() => {
      onQChange(localQ);
    }, DEBOUNCE_MS);
    return () => {
      if (debounceRef.current) clearTimeout(debounceRef.current);
    };
  }, [localQ, q, onQChange]);

  if (hidden) return null;

  return (
    <div className="flex flex-col gap-3 mb-6 md:flex-row md:items-center md:gap-4">
      <div className="relative flex-1 min-w-0">
        <Search
          className="absolute left-3 top-1/2 -translate-y-1/2 size-4 text-muted-foreground pointer-events-none"
          aria-hidden
        />
        <Input
          type="search"
          value={localQ}
          onChange={(e) => setLocalQ(e.target.value)}
          placeholder={t("toolbar.searchPlaceholder")}
          aria-label={t("toolbar.searchPlaceholder")}
          className="pl-9"
        />
      </div>

      <div className="flex items-center gap-2 shrink-0">
        <label
          htmlFor="dashboard-sort"
          className="text-xs text-muted-foreground whitespace-nowrap"
        >
          {t("toolbar.sortLabel")}
        </label>
        <Select
          id="dashboard-sort"
          value={sort}
          onChange={(e) => onSortChange(e.target.value as SortMode)}
          className={cn(
            "h-9 rounded-lg border border-input bg-background px-3 text-sm",
            "outline-none focus-visible:border-primary focus-visible:ring-2 focus-visible:ring-primary/20",
            "cursor-pointer ui-interactive",
          )}
        >
          {SORT_MODES.map((mode) => (
            <option key={mode} value={mode}>
              {t(`toolbar.sortOption.${mode}`)}
            </option>
          ))}
        </Select>
      </div>

      <div
        className="flex items-center gap-2 shrink-0 flex-wrap"
        role="group"
        aria-label={t("toolbar.filterGroupLabel")}
      >
        {FILTER_MODES.map((mode) => {
          const active = filter === mode;
          return (
            <button
              key={mode}
              type="button"
              onClick={() => onFilterChange(mode)}
              aria-pressed={active}
              className={cn(
                "h-7 px-3 rounded-full border text-xs transition-colors ui-interactive",
                active
                  ? "border-primary bg-primary/15 text-primary"
                  : "border-border bg-background text-muted-foreground hover:text-foreground hover:border-foreground/30",
              )}
            >
              {t(`toolbar.filterOption.${mode}`)}
            </button>
          );
        })}
      </div>
    </div>
  );
}
