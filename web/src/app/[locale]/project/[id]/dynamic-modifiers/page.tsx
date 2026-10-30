"use client";

import { useCallback, useMemo, useState } from "react";
import useSWR from "swr";
import { useParams } from "@/platform/navigation";
import { useTranslations } from "use-intl";
import { Layers, SearchX } from "lucide-react";
import { Button } from "@/components/ui/button";
import { dynamicModifiersApi } from "@/lib/api";
import type { DynamicModifier } from "@/types";
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
import { DynamicModifierEditor } from "@/components/workbench/DynamicModifierEditor";
import { useContentLocaleStore } from "@/lib/store/contentLocaleStore";
import { resolveDisplayName } from "@/lib/i18n/localizedName";
import { confirmDiscardIfDirty } from "@/lib/hooks/useUnsavedGuard";
import { useListboxNav } from "@/lib/hooks/useListboxNav";
import { useConfirm } from "@/components/shared/ConfirmDialogProvider";

export default function DynamicModifiersPage() {
  const params = useParams();
  const projectId = Number(params.id);
  const activeCountry = useActiveCountry();

  const filterCountry = useEffectiveCountryFilter();
  const viewMode = useProjectStore((s) => s.viewMode);
  const locale = useContentLocaleStore((s) => s.locale);
  const t = useTranslations("dynamicModifiers");
  const tCommon = useTranslations("common");
  const tShell = useTranslations("workbenchShell");

  const listNav = useListboxNav();

  const confirmDialog = useConfirm();

  const [list, setList] = useState<DynamicModifier[]>([]);

  const [search, setSearch] = useState("");
  const [selectedId, setSelectedId] = useState<number | null>(null);

  const refresh = useCallback(async () => {
    const res = await dynamicModifiersApi.list(projectId);
    setList(res.dynamic_modifiers);
  }, [projectId]);

  const { error: listLoadError, mutate: retryList } = useSWR(
    projectId ? ["dyn-mod/list", projectId] : null,
    () => refresh(),
  );

  const handleCreate = async () => {
    try {
      const maxSuffix = list.reduce((max, dm) => {
        const m = /^dynamic_modifier_(\d+)$/.exec(dm.dm_id || "");
        return m ? Math.max(max, Number(m[1])) : max;
      }, 0);
      const dm = await dynamicModifiersApi.create(projectId, {
        dm_id: `dynamic_modifier_${maxSuffix + 1}`,
        country_tag: activeCountry || "",
      });
      await refresh();
      setSelectedId(dm.id);
    } catch {}
  };

  const handleSave = async (data: Partial<DynamicModifier>) => {
    if (selectedId == null) return;
    await dynamicModifiersApi.update(projectId, selectedId, data);
    await refresh();
  };

  const handleDelete = async (id: number) => {
    if (!(await confirmDialog({ body: t("deleteConfirm"), danger: true })))
      return;
    await dynamicModifiersApi.delete(projectId, id);
    await refresh();
    if (selectedId === id) setSelectedId(null);
  };

  const selected = list.find((x) => x.id === selectedId);

  const filteredList = useMemo(() => {
    const scoped = list.filter((dm) => matchesCountryScope(dm, filterCountry));
    const needle = search.trim().toLowerCase();
    if (!needle) return scoped;
    return scoped.filter(
      (dm) =>
        (dm.dm_id || "").toLowerCase().includes(needle) ||
        (dm.name_en || "").toLowerCase().includes(needle) ||
        (dm.name_zh || "").toLowerCase().includes(needle),
    );
  }, [list, filterCountry, search]);

  const zeroHero = (
    <EditorEmptyState
      variant="hero"
      icon={<Layers className="h-10 w-10 text-muted-foreground/60" />}
      title={t("emptyAll")}
      actions={
        <>
          <Button
            size="sm"
            onClick={() => {
              void handleCreate();
            }}
          >
            {tCommon("emptyCtaCreate")}
          </Button>
        </>
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
              <Layers className="h-6 w-6 text-muted-foreground/60" />
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
        {filteredList.map((dm) => {
          const dmName = resolveDisplayName(
            locale,
            dm.display_name_locs,
            dm.name_en,
            dm.name_zh,
            "",
          );
          return (
            <div
              key={dm.id}
              {...listNav.optionProps(dm.id === selectedId)}
              onClick={() => {
                if (dm.id === selectedId || confirmDiscardIfDirty())
                  setSelectedId(dm.id);
              }}
              className={
                "w-full text-left px-2 py-1.5 rounded text-xs group flex items-center justify-between cursor-pointer hover:bg-muted focus:outline-none focus-visible:ring-1 focus-visible:ring-ring " +
                (dm.id === selectedId
                  ? "bg-primary/10 text-primary font-semibold"
                  : "")
              }
            >
              <div className="min-w-0 flex-1">
                <p className="font-mono truncate flex items-center gap-1.5">
                  {viewMode === "all" && (
                    <CountryTagChip tag={dm.country_tag} />
                  )}
                  <span className="truncate">{dm.dm_id}</span>
                </p>
                {dmName && (
                  <p className="text-muted-foreground truncate text-micro">
                    {dmName}
                  </p>
                )}
              </div>
              <button
                type="button"
                aria-label={t("deleteAria")}
                title={t("deleteAria")}
                onClick={(e) => {
                  e.stopPropagation();
                  handleDelete(dm.id);
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
    <div className="flex h-full min-w-0 overflow-hidden">
      <div className="flex-1 min-w-0 h-full overflow-hidden">
        <WorkbenchShell
          gameViewHref={gameViewHref(projectId, {
            panel: "countrypoliticsview",
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
                    icon={
                      <Layers className="h-6 w-6 text-muted-foreground/60" />
                    }
                    title={tShell("noSelectionGeneric")}
                    description={t("selectHint")}
                  />
                )}
              </div>
            ) : (
              <DynamicModifierEditor
                key={selected.id}
                dm={selected}
                onSave={handleSave}
              />
            )
          }
          previewPanel={
            <div className="flex h-full items-center justify-center p-6 text-center text-xs text-muted-foreground">
              {tShell("previewUnavailableGeneric")}
            </div>
          }
        />
      </div>
    </div>
  );
}
