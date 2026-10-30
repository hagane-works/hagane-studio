"use client";

import { useCallback, useMemo, useState } from "react";
import useSWR from "swr";
import { useParams } from "@/platform/navigation";
import { useTranslations } from "use-intl";
import { SearchX, Swords } from "lucide-react";
import { Button } from "@/components/ui/button";
import { divisionDeploymentsApi, divisionTemplatesApi } from "@/lib/api";
import type { DivisionDeployment, DivisionTemplate } from "@/types";
import { confirmDiscardIfDirty } from "@/lib/hooks/useUnsavedGuard";
import { useActiveCountry, useProjectStore } from "@/lib/store/projectStore";
import { pickDefaultOobFile, tagFromOob } from "@/lib/oob/oobDefaults";
import { useListboxNav } from "@/lib/hooks/useListboxNav";
import { useConfirm } from "@/components/shared/ConfirmDialogProvider";
import { EditorEmptyState } from "@/components/workbench/EditorEmptyState";
import { WorkbenchLoadError } from "@/components/workbench/WorkbenchLoadError";
import { WorkbenchShell } from "@/components/workbench/WorkbenchShell";
import { gameViewHref } from "@/lib/gameViewLinks";
import { WorkbenchListHeader } from "@/components/workbench/WorkbenchListHeader";
import { DeploymentEditor } from "./DeploymentEditor";

export default function DeploymentsPage() {
  const params = useParams();
  const projectId = Number(params.id);
  const t = useTranslations("divisionDeployment");
  const tCommon = useTranslations("common");
  const tShell = useTranslations("workbenchShell");

  const listNav = useListboxNav();

  const confirmDialog = useConfirm();

  const activeCountry = useActiveCountry();
  const projectStartDate = useProjectStore(
    (s) => s.currentProject?.start_date ?? null,
  );

  const [list, setList] = useState<DivisionDeployment[]>([]);

  const [search, setSearch] = useState("");
  const [templates, setTemplates] = useState<DivisionTemplate[]>([]);
  const [selectedId, setSelectedId] = useState<number | null>(null);

  const [error, setError] = useState<string | null>(null);

  const refresh = useCallback(async () => {
    const res = await divisionDeploymentsApi.list(projectId);
    setList(res);
  }, [projectId]);

  const { error: listLoadError, mutate: retryList } = useSWR(
    projectId ? ["division-deployments/list", projectId] : null,
    () => refresh(),
  );
  useSWR(projectId ? ["division-templates/for-deploy", projectId] : null, () =>
    divisionTemplatesApi.list(projectId).then(setTemplates),
  );

  const selected = list.find((x) => x.id === selectedId);

  const defaultOobFile = useMemo(
    () =>
      pickDefaultOobFile({
        countryTag: activeCountry,
        startDate: projectStartDate,
        existing: [
          ...list.map((d) => d.oob_file),
          ...templates.map((tp) => tp.oob_file),
        ],
      }),
    [activeCountry, projectStartDate, list, templates],
  );

  const handleCreate = async () => {
    setError(null);
    try {
      const defaultOob = defaultOobFile;
      const dep = await divisionDeploymentsApi.create(projectId, {
        oob_file: defaultOob,
        country_tag: tagFromOob(defaultOob) || tagFromOob(activeCountry),
        division_template:
          templates.find((tp) => tp.oob_file === defaultOob)?.template_name ||
          "",
        count: 1,
      });
      await refresh();
      setSelectedId(dep.id);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    }
  };

  const handleSave = async (payload: Partial<DivisionDeployment>) => {
    if (!selected) return;
    await divisionDeploymentsApi.update(projectId, selected.id, payload);
    await refresh();
  };

  const handleDelete = async (id: number) => {
    if (!(await confirmDialog({ body: t("deleteConfirm"), danger: true })))
      return;
    await divisionDeploymentsApi.delete(projectId, id);
    await refresh();
    if (selectedId === id) setSelectedId(null);
  };

  const filteredList = list.filter((dep) => {
    const needle = search.trim().toLowerCase();
    if (!needle) return true;
    return (
      (dep.division_name || "").toLowerCase().includes(needle) ||
      (dep.division_template || "").toLowerCase().includes(needle) ||
      (dep.oob_file || "").toLowerCase().includes(needle)
    );
  });

  const zeroHero = (
    <EditorEmptyState
      variant="hero"
      icon={<Swords className="h-10 w-10 text-muted-foreground/60" />}
      title={t("emptyAll")}
      description={t("emptyHint")}
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
        title={t("title")}
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
      {error && (
        <p role="alert" className="px-3 pb-2 text-xs text-destructive">
          {error}
        </p>
      )}
      <div
        className="px-3 pb-3 space-y-0.5 outline-none"
        {...listNav.containerProps}
        aria-label={t("title")}
      >
        {list.length === 0 ? (
          <p className="p-4 text-center text-xs text-muted-foreground">
            {t("emptyAll")}
          </p>
        ) : filteredList.length === 0 ? (
          <div className="p-4 flex flex-col items-center gap-2 text-center border border-dashed rounded-lg bg-muted/20">
            <SearchX className="h-5 w-5 text-muted-foreground" />
            <p className="text-xs text-muted-foreground">
              {tShell("noMatchesGeneric")}
            </p>
          </div>
        ) : null}
        {filteredList.map((dep) => {
          const hasActionable = (dep.warnings || []).some((w) =>
            w.startsWith("actionable:"),
          );
          return (
            <div
              key={dep.id}
              {...listNav.optionProps(dep.id === selectedId)}
              onClick={() => {
                if (dep.id === selectedId || confirmDiscardIfDirty())
                  setSelectedId(dep.id);
              }}
              className={
                "w-full text-left px-2 py-1.5 rounded text-xs group flex items-center justify-between cursor-pointer hover:bg-muted focus:outline-none focus-visible:ring-1 focus-visible:ring-ring " +
                (dep.id === selectedId
                  ? "bg-primary/10 text-primary font-semibold"
                  : "")
              }
            >
              <div className="min-w-0 flex-1">
                <p className="truncate font-medium flex items-center gap-1">
                  {hasActionable && (
                    <span className="text-warning" title={t("hasWarning")}>
                      ⚠
                    </span>
                  )}
                  {dep.division_name || dep.division_template || t("untitled")}
                  {dep.count > 1 && (
                    <span className="text-muted-foreground/70">
                      ×{dep.count}
                    </span>
                  )}
                </p>
                <p className="text-muted-foreground truncate text-micro font-mono">
                  {dep.oob_file} ·{" "}
                  {dep.location != null ? `@${dep.location}` : t("noLocation")}
                </p>
              </div>
              <button
                type="button"
                aria-label={t("deleteAria")}
                title={t("deleteAria")}
                onClick={(e) => {
                  e.stopPropagation();
                  handleDelete(dep.id);
                }}
                className="opacity-0 group-hover:opacity-100 focus-visible:opacity-100 ml-2 text-muted-foreground hover:text-destructive shrink-0 cursor-pointer rounded"
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
        panel: "countrydeploymentview",
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
                icon={<Swords className="h-6 w-6 text-muted-foreground/60" />}
                title={tShell("noSelectionGeneric")}
                description={t("selectHint")}
              />
            )}
          </div>
        ) : (
          <DeploymentEditor
            key={selected.id}
            projectId={projectId}
            deployment={selected}
            templates={templates}
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
  );
}
