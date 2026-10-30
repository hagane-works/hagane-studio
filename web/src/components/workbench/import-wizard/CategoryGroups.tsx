import { useTranslations } from "use-intl";
import type { EntityTriCount, ScanStatsResult } from "@/lib/api";
import type {
  ScanResults,
  ScanCategoryKey,
  Phase,
  CategoryGroup,
} from "./types";

export interface CategoryGroupsProps {
  CATEGORY_GROUPS: CategoryGroup[];
  scan: ScanResults;
  collapsedGroups: Record<string, boolean>;
  toggleGroupCollapsed: (id: string) => void;
  scanStats: ScanStatsResult | null;
  scanStatsLoading: boolean;
  scanStatsError: string | null;
  toggleCategory: (key: ScanCategoryKey) => void;
  phase: Phase;
  superEventCount: number;
  t: ReturnType<typeof useTranslations>;
}

export function CategoryGroups({
  CATEGORY_GROUPS,
  scan,
  collapsedGroups,
  toggleGroupCollapsed,
  scanStats,
  scanStatsLoading,
  scanStatsError,
  toggleCategory,
  phase,
  superEventCount,
  t,
}: CategoryGroupsProps) {
  return (
    <>
      {CATEGORY_GROUPS.map((group) => {
        const totalGroupFiles = group.items.reduce(
          (n, it) => n + scan[it.key].fileCount,
          0,
        );
        if (totalGroupFiles === 0) {
          return null;
        }
        const selectedCount = group.items.filter(
          (it) => scan[it.key].enabled && scan[it.key].fileCount > 0,
        ).length;
        const eligibleCount = group.items.filter(
          (it) => scan[it.key].fileCount > 0,
        ).length;
        const collapsed = !!collapsedGroups[group.id];
        return (
          <div key={group.id} className="rounded-md border border-border/60">
            <button
              type="button"
              onClick={() => toggleGroupCollapsed(group.id)}
              aria-expanded={!collapsed}
              className="w-full flex items-center justify-between px-2.5 py-1.5 text-xs font-medium hover:bg-muted/40 transition-colors"
            >
              <span className="flex items-center gap-2">
                <span className="text-muted-foreground">
                  {collapsed ? "▶" : "▼"}
                </span>
                <span>{group.label}</span>
              </span>
              <span className="text-muted-foreground">
                {t("groupSelectedCount", {
                  selected: selectedCount,
                  eligible: eligibleCount,
                })}
              </span>
            </button>
            {!collapsed && (
              <div className="grid gap-1.5 p-1.5 pt-0 sm:grid-cols-2">
                {group.items.map(({ key, label, icon: Icon, desc }) => {
                  const cat = scan[key];
                  const tri: EntityTriCount | undefined =
                    scanStats?.entity_stats[key];
                  const descText = desc(scan);
                  return (
                    <div key={key} className="space-y-1">
                      <label
                        className={`flex items-start gap-2 rounded-md border p-2 cursor-pointer transition-colors ${
                          cat.fileCount === 0
                            ? "opacity-40 cursor-not-allowed"
                            : cat.enabled
                              ? "border-primary/50 bg-primary/5"
                              : "border-border"
                        }`}
                      >
                        <input
                          type="checkbox"
                          checked={cat.enabled && cat.fileCount > 0}
                          disabled={
                            cat.fileCount === 0 ||
                            phase === "parsing" ||
                            phase === "uploading"
                          }
                          onChange={() => toggleCategory(key)}
                          className="accent-primary h-4 w-4 mt-0.5 shrink-0"
                        />
                        <Icon
                          className="w-3.5 h-3.5 mt-1 shrink-0 text-muted-foreground"
                          strokeWidth={1.8}
                          aria-hidden
                        />
                        <div className="flex-1 min-w-0">
                          <div className="flex items-baseline gap-2">
                            <p className="truncate text-sm font-medium">
                              {label}
                            </p>

                            {cat.fileCount > 0 && (
                              <span className="ml-auto shrink-0 text-xs">
                                {scanStatsLoading && !tri && (
                                  <span className="text-muted-foreground">
                                    {t("statsLoading")}
                                  </span>
                                )}
                                {scanStatsError && !tri && (
                                  <span className="text-destructive">
                                    {t("statsUnavailable")}
                                  </span>
                                )}
                                {tri && (
                                  <>
                                    <span className="text-emerald-500">
                                      {t("triNew", { n: tri.new })}
                                    </span>
                                    <span className="text-muted-foreground">
                                      /
                                    </span>
                                    <span className="text-amber-500">
                                      {t("triModified", { n: tri.modified })}
                                    </span>
                                    <span className="text-muted-foreground">
                                      /
                                    </span>
                                    <span className="text-muted-foreground">
                                      {t("triUnchanged", { n: tri.unchanged })}
                                    </span>
                                  </>
                                )}
                              </span>
                            )}
                          </div>
                          <p
                            className="truncate text-xs text-muted-foreground"
                            title={descText}
                          >
                            {descText}
                          </p>
                        </div>
                      </label>

                      {key === "events" && superEventCount > 0 && (
                        <p className="text-xs text-amber-500 pl-8">
                          {t("superEventsNote", { count: superEventCount })}
                        </p>
                      )}
                    </div>
                  );
                })}
              </div>
            )}
          </div>
        );
      })}
    </>
  );
}
