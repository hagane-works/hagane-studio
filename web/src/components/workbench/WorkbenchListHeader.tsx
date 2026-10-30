"use client";

import type { ReactNode } from "react";
import { useTranslations } from "use-intl";
import { Search, X } from "lucide-react";
import { Input } from "@/components/ui/input";
import { cn } from "@/lib/utils";

interface WorkbenchListHeaderProps {
  title: string;

  count?: number;

  totalCount?: number;
  searchValue: string;
  onSearchChange: (value: string) => void;
  searchPlaceholder?: string;

  renderSearch?: boolean;

  action?: ReactNode;

  countTooltip?: string;

  filters?: ReactNode;

  searchAriaLabel?: string;
  className?: string;
}

export function WorkbenchListHeader({
  title,
  count,
  totalCount,
  searchValue,
  onSearchChange,
  searchPlaceholder,
  renderSearch = true,
  action,
  filters,
  searchAriaLabel,
  countTooltip,
  className,
}: WorkbenchListHeaderProps) {
  const t = useTranslations("common");

  const isFiltered =
    count !== undefined && totalCount !== undefined && totalCount !== count;
  const countLabel =
    count === undefined
      ? null
      : isFiltered
        ? `${count} / ${totalCount}`
        : String(count);

  const badgeTitle =
    countTooltip ??
    (isFiltered
      ? t("countBadgeTooltip", { count: count!, total: totalCount! })
      : undefined);

  return (
    <div className={cn("p-3", className)}>
      <div className="flex flex-wrap items-center justify-between gap-2">
        <div className="flex items-baseline gap-2">
          <h3 className="text-base font-semibold tracking-tight">{title}</h3>
          {countLabel !== null && (
            <span
              title={badgeTitle}
              className="shrink-0 bg-muted text-muted-foreground rounded-full px-2 text-xs"
            >
              {countLabel}
            </span>
          )}
        </div>
        {action && (
          <div className="flex items-center gap-1 shrink-0">{action}</div>
        )}
      </div>

      {filters && <div className="mt-3">{filters}</div>}

      {renderSearch && (
        <div className="relative mt-3">
          <Search className="pointer-events-none absolute left-2.5 top-1/2 -translate-y-1/2 h-3.5 w-3.5 text-muted-foreground/60" />
          <Input
            value={searchValue}
            onChange={(e) => onSearchChange(e.target.value)}
            placeholder={searchPlaceholder}
            aria-label={searchAriaLabel ?? searchPlaceholder ?? t("search")}
            className="h-8 pl-8 pr-7 text-xs ui-focus-ring"
          />

          {searchValue && (
            <button
              type="button"
              onClick={() => onSearchChange("")}
              title={t("clear")}
              aria-label={t("clear")}
              className="ui-interactive absolute right-1.5 top-1/2 -translate-y-1/2 rounded p-0.5 text-muted-foreground hover:text-foreground"
            >
              <X className="h-3.5 w-3.5" aria-hidden />
            </button>
          )}
        </div>
      )}
    </div>
  );
}
