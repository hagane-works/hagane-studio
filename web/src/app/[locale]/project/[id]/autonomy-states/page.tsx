"use client";

import { useCallback, useMemo, useState } from "react";
import useSWR from "swr";
import { useParams } from "@/platform/navigation";
import { useTranslations } from "use-intl";
import { Crown, SearchX } from "lucide-react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { autonomyStatesApi } from "@/lib/api";
import type { AutonomyState } from "@/types";
import {
  useActiveCountry,
  useEffectiveCountryFilter,
  useProjectStore,
} from "@/lib/store/projectStore";
import { matchesCountryScope } from "@/lib/utils/scope";
import { CountryTagChip } from "@/components/workbench/CountryTagChip";
import { EditorEmptyState } from "@/components/workbench/EditorEmptyState";
import { WorkbenchLoadError } from "@/components/workbench/WorkbenchLoadError";
import { WorkbenchShell } from "@/components/workbench/WorkbenchShell";
import { gameViewHref } from "@/lib/gameViewLinks";
import { WorkbenchListHeader } from "@/components/workbench/WorkbenchListHeader";
import { ScopeSectionHeader } from "@/components/workbench/ScopeBadge";
import { AutonomyStateEditor } from "@/components/workbench/AutonomyStateEditor";
import { useContentLocaleStore } from "@/lib/store/contentLocaleStore";
import { resolveDisplayName } from "@/lib/i18n/localizedName";
import { confirmDiscardIfDirty } from "@/lib/hooks/useUnsavedGuard";
import { useListboxNav } from "@/lib/hooks/useListboxNav";
import { useConfirm } from "@/components/shared/ConfirmDialogProvider";

export default function AutonomyStatesPage() {
  const params = useParams();
  const projectId = Number(params.id);
  const activeCountry = useActiveCountry();
  const filterCountry = useEffectiveCountryFilter();
  const viewMode = useProjectStore((s) => s.viewMode);
  const locale = useContentLocaleStore((s) => s.locale);
  const t = useTranslations("autonomyStates");
  const tCommon = useTranslations("common");
  const tShell = useTranslations("workbenchShell");

  const listNav = useListboxNav();

  const confirmDialog = useConfirm();

  const [list, setList] = useState<AutonomyState[]>([]);

  const [search, setSearch] = useState("");
  const [selectedId, setSelectedId] = useState<number | null>(null);

  const refresh = useCallback(async () => {
    const res = await autonomyStatesApi.list(projectId);
    setList(res.autonomy_states);
  }, [projectId]);

  const { error: listLoadError, mutate: retryList } = useSWR(
    projectId ? ["autonomy-states/list", projectId] : null,
    () => refresh(),
  );

  const handleCreate = async () => {
    try {
      const maxSuffix = list.reduce((max, s) => {
        const m = /^autonomy_state_(\d+)$/.exec(s.autonomy_id || "");
        return m ? Math.max(max, Number(m[1])) : max;
      }, 0);
      const s = await autonomyStatesApi.create(projectId, {
        autonomy_id: `autonomy_state_${maxSuffix + 1}`,
        country_tag: activeCountry || "",
      });
      await refresh();
      setSelectedId(s.id);
    } catch (e) {
      toast.error(e instanceof Error ? e.message : String(e));
    }
  };

  const handleDelete = async (id: number) => {
    if (!(await confirmDialog({ body: t("deleteConfirm"), danger: true })))
      return;
    await autonomyStatesApi.delete(projectId, id);
    await refresh();
    if (selectedId === id) setSelectedId(null);
  };

  const selected = list.find((x) => x.id === selectedId);

  const filteredList = useMemo(() => {
    const scoped = list.filter((s) => matchesCountryScope(s, filterCountry));
    const needle = search.trim().toLowerCase();
    if (!needle) return scoped;
    return scoped.filter(
      (s) =>
        (s.autonomy_id || "").toLowerCase().includes(needle) ||
        (s.name_en || "").toLowerCase().includes(needle) ||
        (s.name_zh || "").toLowerCase().includes(needle),
    );
  }, [list, filterCountry, search]);

  const zeroHero = (
    <EditorEmptyState
      variant="hero"
      icon={<Crown className="h-10 w-10 text-muted-foreground/60" />}
      title={t("emptyAll")}
      actions={
        <Button
          size="sm"
          onClick={() => {
            void handleCreate();
          }}
        >
          {tCommon("emptyCtaCreate")}
        </Button>
      }
    />
  );

  const listPanel = (
    <div>
      <WorkbenchListHeader
        title={t("editorHeading")}
        count={filteredList.length}
        totalCount={list.length}
        searchValue={search}
        onSearchChange={setSearch}
        searchPlaceholder={tShell("searchPlaceholderGeneric")}
        renderSearch={list.length > 0}
        action={
          <Button size="sm" onClick={handleCreate}>
            {t("newButton")}
          </Button>
        }
      />
      <div
        className="px-3 pb-3 space-y-0.5 outline-none"
        {...listNav.containerProps}
        aria-label={t("editorHeading")}
      >
        {list.length === 0 ? (
          <p className="p-4 text-center text-xs text-muted-foreground">
            {t("emptyAll")}
          </p>
        ) : filteredList.length === 0 ? (
          search.trim() ? (
            <div className="p-4 flex flex-col items-center gap-2 text-center border border-dashed rounded-lg bg-muted/20">
              <SearchX className="h-5 w-5 text-muted-foreground" />
              <p className="text-xs text-muted-foreground">
                {tShell("noMatchesGeneric")}
              </p>
            </div>
          ) : (
            <div className="p-4 rounded-md border border-border bg-card flex flex-col items-center gap-2 text-center">
              <Crown className="h-6 w-6 text-muted-foreground/60" />
              <p className="text-xs font-medium text-foreground">
                {t("emptyScoped", {
                  country: activeCountry || t("currentCountry"),
                })}
              </p>
              <p className="text-xs text-muted-foreground">
                {t("newButton")} →
              </p>
            </div>
          )
        ) : null}
        {!filterCountry && filteredList.length > 0 && (
          <ScopeSectionHeader variant="all" count={filteredList.length} />
        )}
        {filteredList.map((s) => {
          const sName = resolveDisplayName(
            locale,
            s.display_name_locs,
            s.name_en,
            s.name_zh,
            "",
          );
          return (
            <div
              key={s.id}
              {...listNav.optionProps(s.id === selectedId)}
              onClick={() => {
                if (s.id === selectedId || confirmDiscardIfDirty())
                  setSelectedId(s.id);
              }}
              className={
                "w-full text-left px-2 py-1.5 rounded text-xs group flex items-center justify-between cursor-pointer hover:bg-muted focus:outline-none focus-visible:ring-1 focus-visible:ring-ring " +
                (s.id === selectedId
                  ? "bg-primary/10 text-primary font-semibold"
                  : "")
              }
            >
              <div className="min-w-0 flex-1">
                <p className="font-mono truncate flex items-center gap-1.5">
                  {viewMode === "all" && <CountryTagChip tag={s.country_tag} />}
                  <span className="truncate">{s.autonomy_id}</span>
                </p>
                {sName && (
                  <p className="text-muted-foreground truncate text-micro">
                    {sName}
                  </p>
                )}
              </div>
              <button
                type="button"
                aria-label={t("deleteAria")}
                title={t("deleteAria")}
                onClick={(e) => {
                  e.stopPropagation();
                  handleDelete(s.id);
                }}
                className="opacity-0 group-hover:opacity-100 focus-visible:opacity-100 ml-2 text-muted-foreground hover:text-destructive shrink-0 cursor-pointer focus:outline-none focus-visible:ring-1 focus-visible:ring-ring rounded"
              >
                ×
              </button>
            </div>
          );
        })}
      </div>
    </div>
  );

  if (listLoadError && list.length === 0) {
    return (
      <WorkbenchLoadError
        onRetry={() => retryList(undefined, { throwOnError: false })}
      />
    );
  }

  return (
    <WorkbenchShell
      gameViewHref={gameViewHref(projectId, {
        panel: "manage_subjects_window",
        tag: activeCountry,
      })}
      listPanel={listPanel}
      editorPanel={
        !selected ? (
          <div className="flex h-full items-center justify-center p-6">
            {list.length === 0 ? (
              zeroHero
            ) : (
              <EditorEmptyState
                className="max-w-xs"
                icon={<Crown className="h-6 w-6 text-muted-foreground/60" />}
                title={tShell("noSelectionGeneric")}
                description={t("selectHint")}
              />
            )}
          </div>
        ) : (
          <AutonomyStateEditor
            key={selected.id}
            state={selected}
            onSave={async (data) => {
              await autonomyStatesApi.update(projectId, selected.id, data);
              await refresh();
            }}
          />
        )
      }
      previewPanel={
        <div className="flex h-full items-center justify-center p-6 text-center text-xs text-muted-foreground">
          {tShell("previewUnavailableGeneric")}
        </div>
      }
    />
  );
}
