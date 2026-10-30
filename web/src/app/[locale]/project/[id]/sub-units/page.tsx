"use client";

import { useCallback, useEffect, useState } from "react";
import { useParams } from "@/platform/navigation";
import { useTranslations } from "use-intl";
import useSWR from "swr";
import { FileCode2, Plus, SearchX, Shield, Trash2, Upload } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { useConfirm } from "@/components/shared/ConfirmDialogProvider";
import { EditorEmptyState } from "@/components/workbench/EditorEmptyState";
import { WorkbenchLoadError } from "@/components/workbench/WorkbenchLoadError";
import { WorkbenchShell } from "@/components/workbench/WorkbenchShell";
import { gameViewHref } from "@/lib/gameViewLinks";
import { useActiveCountry } from "@/lib/store/projectStore";
import { WorkbenchListHeader } from "@/components/workbench/WorkbenchListHeader";
import { confirmDiscardIfDirty } from "@/lib/hooks/useUnsavedGuard";
import { resolveDisplayName } from "@/lib/i18n/localizedName";
import { subUnitsApi } from "@/lib/api";
import { useContentLocaleStore } from "@/lib/store/contentLocaleStore";
import type { SubUnit } from "@/types";
import { emptySubUnitDraft, subUnitDraftToPayload } from "./subUnitForm";
import { SubUnitEditor } from "./SubUnitEditor";

export default function SubUnitsPage() {
  const params = useParams<{ id: string }>();
  const projectId = Number(params.id);
  const t = useTranslations("subUnitEditor");
  const tCommon = useTranslations("common");
  const tShell = useTranslations("workbenchShell");
  const locale = useContentLocaleStore((state) => state.locale);
  const confirmDialog = useConfirm();

  const [rows, setRows] = useState<SubUnit[]>([]);
  const activeCountry = useActiveCountry();
  const [selectedId, setSelectedId] = useState<number | null>(null);

  const [search, setSearch] = useState("");
  const [preview, setPreview] = useState("");

  const [error, setError] = useState<string | null>(null);
  const [showImport, setShowImport] = useState(false);
  const [importText, setImportText] = useState("");
  const [importSourceFile, setImportSourceFile] = useState(
    "common/units/hagane_units.txt",
  );
  const [importPolicy, setImportPolicy] = useState<"overwrite" | "skip">(
    "overwrite",
  );
  const [importMessage, setImportMessage] = useState<string | null>(null);

  const refresh = useCallback(async () => {
    setRows(await subUnitsApi.list(projectId));
  }, [projectId]);

  const { error: listLoadError, mutate: retryList } = useSWR(
    projectId ? ["sub-units/list", projectId] : null,
    refresh,
  );

  const selected = rows.find((row) => row.id === selectedId) ?? null;

  const refreshPreview = useCallback(
    (id: number) =>
      subUnitsApi.preview(projectId, id).then(
        (result) => setPreview(result.script ?? ""),
        (caught: unknown) => {
          setPreview("");
          setError(caught instanceof Error ? caught.message : String(caught));
        },
      ),
    [projectId],
  );

  useEffect(() => {
    if (selectedId !== null) void refreshPreview(selectedId);
  }, [refreshPreview, selectedId]);

  const createSubUnit = async () => {
    setError(null);
    try {
      const row = await subUnitsApi.create(
        projectId,
        subUnitDraftToPayload(emptySubUnitDraft()),
      );
      await refresh();
      setSelectedId(row.id);
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : String(caught));
    }
  };

  const saveSubUnit = async (update: Partial<SubUnit>) => {
    if (!selected) return;
    setError(null);
    if (Object.keys(update).length > 0) {
      await subUnitsApi.update(projectId, selected.id, update);
      await refresh();
    }
    await refreshPreview(selected.id);
  };

  const deleteSubUnit = async (row: SubUnit) => {
    if (
      !(await confirmDialog({
        body: t("deleteConfirm", { id: row.sub_unit_id }),
        danger: true,
      }))
    ) {
      return;
    }
    try {
      await subUnitsApi.delete(projectId, row.id);

      if (selectedId === row.id) {
        setSelectedId(null);
        setPreview("");
      }
      await refresh();
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : String(caught));
    }
  };

  const importSubUnits = async () => {
    setImportMessage(null);
    setError(null);
    try {
      const result = await subUnitsApi.importText(
        projectId,
        importText,
        importSourceFile,
        importPolicy,
      );
      await refresh();
      setImportMessage(
        t("importResult", {
          created: result.created,
          updated: result.updated,
          skipped: result.skipped,
          warnings: result.warnings.length,
        }),
      );
      setImportText("");
    } catch (caught) {
      setImportMessage(
        caught instanceof Error ? caught.message : String(caught),
      );
    }
  };

  const filteredRows = rows.filter((row) => {
    const needle = search.trim().toLowerCase();
    if (!needle) return true;
    return (
      (row.sub_unit_id || "").toLowerCase().includes(needle) ||
      (row.name_en || "").toLowerCase().includes(needle) ||
      (row.name_zh || "").toLowerCase().includes(needle) ||
      (row.group || "").toLowerCase().includes(needle)
    );
  });

  const zeroHero = (
    <EditorEmptyState
      variant="hero"
      icon={<Shield className="h-10 w-10 text-muted-foreground/60" />}
      title={t("emptyTitle")}
      description={t("emptyDescription")}
      actions={
        <Button size="sm" onClick={createSubUnit}>
          {tCommon("emptyCtaCreate")}
        </Button>
      }
    />
  );

  const listPanel = (
    <div>
      <WorkbenchListHeader
        title={t("title")}
        count={filteredRows.length}
        totalCount={rows.length}
        searchValue={search}
        onSearchChange={setSearch}
        searchPlaceholder={tShell("searchPlaceholderGeneric")}
        renderSearch={rows.length > 0}
        action={
          <Button size="sm" onClick={createSubUnit}>
            <Plus className="h-4 w-4" aria-hidden />
            {t("newButton")}
          </Button>
        }
        filters={
          <button
            type="button"
            onClick={() => setShowImport((value) => !value)}
            className="text-xs text-muted-foreground hover:text-foreground ui-interactive ui-focus-ring rounded-md underline decoration-dotted"
          >
            {t("importTitle")}
          </button>
        }
      />
      <div
        className="px-3 pb-3 space-y-1"
        role="listbox"
        aria-label={t("title")}
      >
        {rows.length === 0 ? (
          <p className="p-4 text-center text-xs text-muted-foreground">
            {t("emptyTitle")}
          </p>
        ) : filteredRows.length === 0 ? (
          <div className="p-4 flex flex-col items-center gap-2 text-center border border-dashed rounded-lg bg-muted/20">
            <SearchX className="h-5 w-5 text-muted-foreground" />
            <p className="text-xs text-muted-foreground">
              {tShell("noMatchesGeneric")}
            </p>
          </div>
        ) : null}
        {filteredRows.map((row) => {
          const displayName = resolveDisplayName(
            locale,
            row.display_name_locs,
            row.name_en,
            row.name_zh,
            "",
          );
          const isSelected = row.id === selectedId;
          return (
            <div
              key={row.id}
              role="option"
              aria-selected={isSelected}
              tabIndex={0}
              onClick={() => {
                if (row.id === selectedId || confirmDiscardIfDirty())
                  setSelectedId(row.id);
              }}
              onKeyDown={(event) => {
                if (event.key === "Enter" || event.key === " ") {
                  event.preventDefault();
                  setSelectedId(row.id);
                }
              }}
              className={`ui-card group cursor-pointer px-2 py-2 flex items-center gap-2 ui-focus-ring ${
                isSelected ? "border-accent/50 bg-accent/10 text-accent" : ""
              }`}
            >
              <div className="min-w-0 flex-1">
                <p className="font-mono text-xs truncate">{row.sub_unit_id}</p>
                <p className="text-xs text-muted-foreground truncate">
                  {displayName || row.group || row.source_file}
                </p>
              </div>
              <button
                type="button"
                title={t("deleteAria")}
                aria-label={t("deleteAria")}
                onClick={(event) => {
                  event.stopPropagation();
                  void deleteSubUnit(row);
                }}
                className="inline-flex h-7 w-7 shrink-0 items-center justify-center opacity-0 group-hover:opacity-100 focus-visible:opacity-100 text-muted-foreground hover:text-destructive rounded-md ui-interactive ui-focus-ring"
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
    <section className="p-4 border-b border-border bg-muted/30 space-y-3">
      <div className="grid grid-cols-[minmax(0,1fr)_auto] gap-3 items-end">
        <div className="space-y-1">
          <Label className="text-xs">{t("sourceFile")}</Label>
          <Input
            className="font-mono text-xs"
            value={importSourceFile}
            onChange={(event) => setImportSourceFile(event.target.value)}
          />
        </div>
        <div className="flex rounded-md border border-border p-0.5">
          {(["overwrite", "skip"] as const).map((policy) => (
            <button
              key={policy}
              type="button"
              onClick={() => setImportPolicy(policy)}
              className={`h-7 px-3 rounded-md text-xs ui-interactive ui-focus-ring ${
                importPolicy === policy
                  ? "bg-accent text-accent-foreground"
                  : "text-muted-foreground"
              }`}
            >
              {t(policy === "overwrite" ? "overwrite" : "skip")}
            </button>
          ))}
        </div>
      </div>
      <Textarea
        className="min-h-32 font-mono text-xs"
        placeholder={t("importPlaceholder")}
        value={importText}
        onChange={(event) => setImportText(event.target.value)}
      />
      <div className="flex items-center gap-3">
        <Button
          size="sm"
          onClick={importSubUnits}
          disabled={!importText.trim()}
        >
          <Upload className="h-4 w-4" aria-hidden />
          {t("importButton")}
        </Button>
        {importMessage && (
          <p className="text-xs text-muted-foreground">{importMessage}</p>
        )}
      </div>
    </section>
  );

  if (listLoadError && rows.length === 0) {
    return (
      <WorkbenchLoadError
        onRetry={() => retryList(undefined, { throwOnError: false })}
      />
    );
  }

  return (
    <WorkbenchShell
      gameViewHref={gameViewHref(projectId, {
        panel: "countrydivisiondesignerview",
        tag: activeCountry,
      })}
      listPanel={listPanel}
      editorPanel={
        <div className="flex min-h-full flex-col">
          {importBlock}
          {!selected ? (
            <div className="flex flex-1 items-center justify-center p-6">
              {rows.length === 0 ? (
                zeroHero
              ) : (
                <EditorEmptyState
                  className="max-w-xs"
                  icon={<Shield className="h-6 w-6 text-muted-foreground/60" />}
                  title={tShell("noSelectionGeneric")}
                  description={t("selectHint")}
                />
              )}
            </div>
          ) : (
            <SubUnitEditor
              key={selected.id}
              subUnit={selected}
              onSave={saveSubUnit}
              error={error}
            />
          )}
        </div>
      }
      previewPanel={
        <div className="p-4">
          <div className="flex items-center gap-2 mb-3">
            <FileCode2 className="h-4 w-4 text-muted-foreground" aria-hidden />
            <h3 className="text-xs font-semibold">{t("preview")}</h3>
          </div>
          {preview ? (
            <pre className="text-xs font-mono whitespace-pre-wrap break-words rounded-md border border-border bg-background p-3 overflow-auto">
              {preview}
            </pre>
          ) : (
            <p className="text-xs text-muted-foreground">{t("previewEmpty")}</p>
          )}
        </div>
      }
    />
  );
}
