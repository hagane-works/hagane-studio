"use client";

import { useCallback, useEffect, useState } from "react";
import { useLinkedRowSelection } from "@/lib/workbench-state/useLinkedRowSelection";
import useSWR from "swr";
import { useParams } from "@/platform/navigation";
import { useTranslations } from "use-intl";
import { Crosshair, Plus, SearchX, Trash2, Upload } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { equipmentApi } from "@/lib/api";
import type { Equipment } from "@/types";
import { useContentLocaleStore } from "@/lib/store/contentLocaleStore";
import { resolveDisplayName } from "@/lib/i18n/localizedName";
import { confirmDiscardIfDirty } from "@/lib/hooks/useUnsavedGuard";
import { useListboxNav } from "@/lib/hooks/useListboxNav";
import { useConfirm } from "@/components/shared/ConfirmDialogProvider";
import { EditorEmptyState } from "@/components/workbench/EditorEmptyState";
import { WorkbenchLoadError } from "@/components/workbench/WorkbenchLoadError";
import { WorkbenchShell } from "@/components/workbench/WorkbenchShell";
import { gameViewHref } from "@/lib/gameViewLinks";
import { WorkbenchListHeader } from "@/components/workbench/WorkbenchListHeader";
import { EquipmentEditor } from "./EquipmentEditor";

export default function EquipmentPage() {
  const params = useParams();
  const projectId = Number(params.id);
  const contentLocale = useContentLocaleStore((s) => s.locale);
  const t = useTranslations("equipmentEditor");
  const tCommon = useTranslations("common");
  const tShell = useTranslations("workbenchShell");

  const listNav = useListboxNav();

  const confirmDialog = useConfirm();

  const [list, setList] = useState<Equipment[]>([]);

  const [search, setSearch] = useState("");
  const [selectedId, setSelectedId] = useState<number | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [enabledByTechs, setEnabledByTechs] = useState<
    { tech_id: string; name: string }[]
  >([]);
  const [showImport, setShowImport] = useState(false);
  const [importText, setImportText] = useState("");
  const [importSourceFile, setImportSourceFile] = useState("");
  const [importMsg, setImportMsg] = useState<string | null>(null);

  const refresh = useCallback(async () => {
    const res = await equipmentApi.list(projectId);
    setList(res);
  }, [projectId]);

  const { error: listLoadError, mutate: retryList } = useSWR(
    projectId ? ["equipment/list", projectId] : null,
    () => refresh(),
  );

  useLinkedRowSelection("equipment", list, (eq) => setSelectedId(eq.id));

  const selected = list.find((x) => x.id === selectedId);
  useEffect(() => {
    if (!selected) return;
    let alive = true;
    equipmentApi
      .enabledBy(projectId, selected.equipment_id)
      .then((r) => {
        if (alive) setEnabledByTechs(r.techs || []);
      })
      .catch(() => {
        if (alive) setEnabledByTechs([]);
      });
    return () => {
      alive = false;
    };
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [projectId, selected?.equipment_id]);

  const handleCreate = async () => {
    setError(null);
    try {
      const maxSuffix = list.reduce((max, e) => {
        const m = /^new_equipment_(\d+)$/.exec(e.equipment_id || "");
        return m ? Math.max(max, Number(m[1])) : max;
      }, 0);
      const eq = await equipmentApi.create(projectId, {
        equipment_id: list.some((e) => e.equipment_id === "new_equipment")
          ? `new_equipment_${maxSuffix + 1}`
          : "new_equipment",
        equipment_type: "infantry",
      });
      await refresh();
      setSelectedId(eq.id);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    }
  };

  const handleSave = async (payload: Partial<Equipment>) => {
    if (selectedId == null) return;
    setError(null);
    await equipmentApi.update(projectId, selectedId, payload);
    await refresh();
  };

  const handleDelete = async (id: number) => {
    if (!(await confirmDialog({ body: t("deleteConfirm"), danger: true })))
      return;
    await equipmentApi.delete(projectId, id);
    await refresh();
    if (selectedId === id) setSelectedId(null);
  };

  const handleImport = async () => {
    setImportMsg(null);
    try {
      const r = await equipmentApi.importText(
        projectId,
        importText,
        importSourceFile,
      );
      await refresh();
      setImportMsg(
        t("importResult", {
          created: r.created,
          updated: r.updated,
          skipped: r.skipped,
          warnings: r.warnings.length,
        }),
      );
      setImportText("");
    } catch (e) {
      setImportMsg(e instanceof Error ? e.message : String(e));
    }
  };

  const filteredList = list.filter((eq) => {
    const needle = search.trim().toLowerCase();
    if (!needle) return true;
    return (
      (eq.equipment_id || "").toLowerCase().includes(needle) ||
      (eq.equipment_type || "").toLowerCase().includes(needle) ||
      (eq.name || "").toLowerCase().includes(needle) ||
      (eq.name_en || "").toLowerCase().includes(needle) ||
      (eq.name_zh || "").toLowerCase().includes(needle)
    );
  });

  const zeroHero = (
    <EditorEmptyState
      variant="hero"
      icon={<Crosshair className="h-10 w-10 text-muted-foreground/60" />}
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
            <Plus className="h-4 w-4" aria-hidden />
            {t("newButton")}
          </Button>
        }
        filters={
          <button
            type="button"
            onClick={() => setShowImport((v) => !v)}
            className="text-xs underline decoration-dotted text-muted-foreground hover:text-foreground"
          >
            {t("importTitle")}
          </button>
        }
      />
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
        {filteredList.map((eq) => {
          const nm = resolveDisplayName(
            contentLocale,
            eq.display_name_locs,
            eq.name_en,
            eq.name_zh,
            "",
          );
          return (
            <div
              key={eq.id}
              {...listNav.optionProps(eq.id === selectedId)}
              onClick={() => {
                if (eq.id === selectedId || confirmDiscardIfDirty())
                  setSelectedId(eq.id);
              }}
              className={
                "w-full text-left px-2 py-1.5 rounded text-xs group flex items-center justify-between cursor-pointer hover:bg-muted focus:outline-none focus-visible:ring-1 focus-visible:ring-ring " +
                (eq.id === selectedId
                  ? "bg-primary/10 text-primary font-semibold"
                  : "")
              }
            >
              <div className="min-w-0 flex-1">
                <p className="font-mono truncate flex items-center gap-1.5">
                  {eq.is_archetype && (
                    <span className="text-micro px-1 rounded bg-primary/15 text-primary">
                      A
                    </span>
                  )}
                  <span className="truncate">{eq.equipment_id}</span>
                </p>
                {nm && (
                  <p className="text-muted-foreground truncate text-micro">
                    {nm}
                  </p>
                )}
              </div>
              <button
                type="button"
                aria-label={t("deleteAria")}
                title={t("deleteAria")}
                onClick={(e) => {
                  e.stopPropagation();
                  handleDelete(eq.id);
                }}
                className="inline-flex h-7 w-7 items-center justify-center opacity-0 group-hover:opacity-100 focus-visible:opacity-100 ml-2 text-muted-foreground hover:text-destructive shrink-0 cursor-pointer rounded-md ui-interactive ui-focus-ring"
              >
                <Trash2 className="h-4 w-4" aria-hidden />
              </button>
            </div>
          );
        })}
      </div>
    </div>
  );

  const importBlock = showImport && (
    <div className="p-4 border-b border-border bg-muted/30 space-y-2">
      <p className="text-xs text-muted-foreground">{t("importHint")}</p>
      <div className="space-y-1">
        <Label className="text-xs">{t("sourceFileLabel")}</Label>
        <Input
          className="font-mono text-xs"
          placeholder={t("sourceFilePlaceholder")}
          value={importSourceFile}
          onChange={(event) => setImportSourceFile(event.target.value)}
        />
      </div>
      <Textarea
        className="font-mono text-xs min-h-28"
        placeholder={t("importPlaceholder")}
        value={importText}
        onChange={(e) => setImportText(e.target.value)}
      />
      <div className="flex items-center gap-2">
        <Button size="sm" onClick={handleImport} disabled={!importText.trim()}>
          <Upload className="h-4 w-4" aria-hidden />
          {t("importButton")}
        </Button>
        {importMsg && (
          <span className="text-xs text-muted-foreground">{importMsg}</span>
        )}
      </div>
    </div>
  );

  const shownTechs = selected ? enabledByTechs : [];
  const enabledByPanel = (
    <div className="p-3 space-y-2">
      <h3 className="text-xs font-semibold">{t("enabledByTitle")}</h3>
      <p className="text-micro text-muted-foreground">{t("enabledByHint")}</p>
      {shownTechs.length === 0 ? (
        <p className="text-xs text-muted-foreground italic">
          {t("enabledByEmpty")}
        </p>
      ) : (
        <ul className="space-y-1">
          {shownTechs.map((tch) => (
            <li key={tch.tech_id} className="text-xs">
              <span className="font-mono">{tch.tech_id}</span>
              {tch.name && tch.name !== tch.tech_id && (
                <span className="text-muted-foreground block text-micro truncate">
                  {tch.name}
                </span>
              )}
            </li>
          ))}
        </ul>
      )}
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
        panel: "countryproductionlineview",
      })}
      listPanel={listPanel}
      editorPanel={
        <div className="flex min-h-full flex-col">
          {importBlock}
          {!selected ? (
            <div className="flex flex-1 items-center justify-center p-6">
              {list.length === 0 ? (
                zeroHero
              ) : (
                <EditorEmptyState
                  className="max-w-xs"
                  icon={
                    <Crosshair className="h-6 w-6 text-muted-foreground/60" />
                  }
                  title={tShell("noSelectionGeneric")}
                  description={t("selectHint")}
                />
              )}
            </div>
          ) : (
            <EquipmentEditor
              key={selected.id}
              equipment={selected}
              list={list}
              onSave={handleSave}
              error={error}
            />
          )}
        </div>
      }
      previewPanel={enabledByPanel}
    />
  );
}
