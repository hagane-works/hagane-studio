"use client";

import { useState } from "react";
import { useTranslations } from "use-intl";
import {
  AlertTriangle,
  ChevronDown,
  ChevronRight,
  Info,
  Layers,
} from "lucide-react";

import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { RawOverrideFilesManager } from "@/components/workbench/RawOverrideFilesManager";
import { BatchFlipButton } from "@/components/workbench/BatchFlipButton";
import { UNSUPPORTED_OR_LOSSY_EXPORT_PATHS } from "@/generated/exportCoverage";

export const CONTENT_REPLACE_PATH_OPTIONS = [
  {
    key: "characters",
    path: "common/characters",
    labelKey: "replacePathCharacters",
  },
  { key: "focus", path: "common/national_focus", labelKey: "replacePathFocus" },
  {
    key: "technologies",
    path: "common/technologies",
    labelKey: "replacePathTechnologies",
  },
  { key: "ideas", path: "common/ideas", labelKey: "replacePathIdeas" },
  { key: "events", path: "events", labelKey: "replacePathEvents" },
  {
    key: "decisions",
    path: "common/decisions",
    labelKey: "replacePathDecisions",
  },
] as const;

export const WORLD_REPLACE_PATH_OPTIONS = [
  {
    key: "countries",
    path: "common/countries",
    labelKey: "replacePathCountries",
  },
  {
    key: "historyStates",
    path: "history/states",
    labelKey: "replacePathHistoryStates",
  },
  {
    key: "historyCountries",
    path: "history/countries",
    labelKey: "replacePathHistoryCountries",
  },
  {
    key: "historyUnits",
    path: "history/units",
    labelKey: "replacePathHistoryUnits",
  },
  {
    key: "historyGeneral",
    path: "history/general",
    labelKey: "replacePathHistoryGeneral",
  },
] as const;

export const REPLACE_PATH_OPTIONS = [
  ...CONTENT_REPLACE_PATH_OPTIONS,
  ...WORLD_REPLACE_PATH_OPTIONS,
] as const;

export function getSelectedReplacePaths(
  replacePaths: Record<string, boolean>,
): string[] {
  return REPLACE_PATH_OPTIONS.filter((option) => replacePaths[option.key]).map(
    (option) => option.path,
  );
}

const FILE_FOLDER_ORDER = [
  "common",
  "history",
  "localisation",
  "events",
  "gfx",
  "interface",
  "map",
  "portraits",
  "music",
  "sound",
];
const FILE_ROOT_KEY = "__root__";

interface FileFolderGroup {
  key: string;
  files: string[];

  sub: { key: string; files: string[]; isFolder: boolean }[] | null;
}

function toModRelative(files: string[]): string[] {
  const norm = files.map((f) => f.replace(/\\/g, "/"));
  const firstSegs = new Set<string>();
  for (const f of norm) {
    const i = f.indexOf("/");
    if (i >= 0) firstSegs.add(f.slice(0, i));
  }
  if (firstSegs.size !== 1) return norm;
  const seg = [...firstSegs][0];
  if (FILE_FOLDER_ORDER.includes(seg)) return norm;
  const prefix = seg + "/";
  return norm.map((f) => (f.startsWith(prefix) ? f.slice(prefix.length) : f));
}

export function groupExportFiles(files: string[]): FileFolderGroup[] {
  const byTop = new Map<string, string[]>();
  for (const f of toModRelative(files)) {
    const i = f.indexOf("/");
    const key = i >= 0 ? f.slice(0, i) : FILE_ROOT_KEY;
    const arr = byTop.get(key) ?? [];
    arr.push(f);
    byTop.set(key, arr);
  }
  const rank = (k: string) => {
    if (k === FILE_ROOT_KEY) return 1000;
    const idx = FILE_FOLDER_ORDER.indexOf(k);
    return idx >= 0 ? idx : 500;
  };
  return [...byTop.entries()]
    .sort((a, b) => rank(a[0]) - rank(b[0]) || a[0].localeCompare(b[0]))
    .map(([key, fs]) => {
      const sorted = [...fs].sort((a, b) => a.localeCompare(b));
      let sub: FileFolderGroup["sub"] = null;
      if (key === "common") {
        const bySub = new Map<string, string[]>();
        for (const f of sorted) {
          const parts = f.replace(/\\/g, "/").split("/");
          const subKey = parts[1] ?? "";
          const arr = bySub.get(subKey) ?? [];
          arr.push(f);
          bySub.set(subKey, arr);
        }
        sub = [...bySub.entries()]
          .sort((a, b) => a[0].localeCompare(b[0]))
          .map(([k, v]) => ({
            key: k,
            files: v,
            isFolder: v.some(
              (f) => f.replace(/\\/g, "/").split("/").length > 2,
            ),
          }));
      }
      return { key, files: sorted, sub };
    });
}

function FileStructurePreview({
  files,
  modName,
}: {
  files: string[];
  modName: string;
}) {
  const t = useTranslations("export");
  const groups = groupExportFiles(files);
  const folderLabel = (key: string) =>
    key === FILE_ROOT_KEY ? t("fileGroupRoot") : `${key}/`;

  return (
    <div className="rounded-lg border border-border bg-muted/20 p-3">
      <p className="flex items-center gap-1.5 text-xs font-medium text-foreground">
        <Layers
          className="h-3.5 w-3.5 shrink-0 text-muted-foreground"
          aria-hidden
        />
        {t("fileStructureHeadline", {
          mod: modName,
          files: files.length,
          groups: groups.length,
        })}
      </p>

      <div className="mt-2 space-y-2 pl-1">
        {groups.map((g) => (
          <div key={g.key}>
            <p className="text-xs font-medium text-foreground">
              <code className="font-mono">{folderLabel(g.key)}</code>
              <span className="text-muted-foreground"> · {g.files.length}</span>
            </p>
            <pre className="mt-0.5 whitespace-pre-wrap break-all pl-3 font-mono text-xs leading-relaxed text-muted-foreground">
              {g.files.join("\n")}
            </pre>
          </div>
        ))}
      </div>
    </div>
  );
}

export function ExportSettingsDialog({
  open,
  onOpenChange,
  projectId,
  projectName,
  files,
  buildIdentity,
  onBuildIdentityChange,
  hasTechnologySurface,
  replacePaths,
  onToggleReplacePath,
  showCoverageNote,
  showSourceModeChoice,
  includeImported,
  onIncludeImportedChange,
  onOpenAppend,
  onOpenDiffMerge,
  onBatchFlipSuccess,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  projectId: number;
  projectName: string;

  files: string[];
  buildIdentity: boolean;
  onBuildIdentityChange: (value: boolean) => void;
  hasTechnologySurface: boolean;
  replacePaths: Record<string, boolean>;
  onToggleReplacePath: (key: string, checked: boolean) => void;

  showCoverageNote: boolean;

  showSourceModeChoice: boolean;
  includeImported: boolean;
  onIncludeImportedChange: (value: boolean) => void;
  onOpenAppend: () => void;
  onOpenDiffMerge: () => void;
  onBatchFlipSuccess: () => void;
}) {
  const t = useTranslations("export");
  const tBuild = useTranslations("exportBuildIdentity");

  const [rawFilesOpen, setRawFilesOpen] = useState(false);
  const [replaceOpen, setReplaceOpen] = useState(false);
  const [advancedOpen, setAdvancedOpen] = useState(false);

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-2xl">
        <DialogHeader>
          <DialogTitle>{t("ui.dialogSettings.title")}</DialogTitle>
          <DialogDescription>{t("ui.dialogSettings.sub")}</DialogDescription>
        </DialogHeader>

        <div className="max-h-[62vh] space-y-4 overflow-y-auto pr-1">
          {hasTechnologySurface ? (
            <section
              data-testid="technology-additive-export-guidance"
              className="flex items-start gap-2.5 rounded-lg border border-border bg-muted/20 p-3"
            >
              <Info
                className="mt-0.5 h-4 w-4 shrink-0 text-muted-foreground"
                aria-hidden
              />
              <div className="min-w-0 space-y-1.5">
                <p className="text-sm font-medium text-foreground">
                  {t("technologyAdditiveTitle")}
                </p>
                <p className="text-xs text-muted-foreground">
                  {t("technologyAdditiveBody")}
                </p>
                <p className="text-xs text-muted-foreground">
                  {t("technologyAdditiveFiles")}
                </p>
                <p className="text-xs text-muted-foreground">
                  {t("technologyReplacePathNote")}
                </p>
              </div>
            </section>
          ) : null}

          {showSourceModeChoice ? (
            <section className="space-y-2 rounded-lg border border-border bg-muted/20 p-3">
              <p className="text-sm font-medium text-foreground">
                {t("identityAmbiguousTitle")}
              </p>
              <p className="text-xs text-muted-foreground">
                {t("identityAmbiguousDesc")}
              </p>
              <div className="grid gap-2 sm:grid-cols-2">
                {([true, false] as const).map((full) => (
                  <button
                    key={full ? "full" : "delta"}
                    type="button"
                    onClick={() => onIncludeImportedChange(full)}
                    aria-pressed={includeImported === full}
                    className={`rounded-lg border p-3 text-left ui-interactive ${
                      includeImported === full
                        ? "border-accent bg-accent/10"
                        : "border-border bg-card hover:bg-muted/20"
                    }`}
                  >
                    <span className="block text-sm font-semibold text-foreground">
                      {t(
                        full
                          ? "identityAmbiguousFullBtn"
                          : "identityAmbiguousDeltaBtn",
                      )}
                    </span>
                    <span className="mt-1 block text-xs text-muted-foreground">
                      {t(
                        full
                          ? "identityAmbiguousFullDesc"
                          : "identityAmbiguousDeltaDesc",
                      )}
                    </span>
                  </button>
                ))}
              </div>
            </section>
          ) : null}

          <label className="flex items-start gap-3 rounded-lg border border-border bg-muted/20 p-3 ui-interactive">
            <input
              type="checkbox"
              checked={buildIdentity}
              onChange={(event) => onBuildIdentityChange(event.target.checked)}
              className="mt-0.5 h-4 w-4 rounded border-border accent-accent ui-focus-ring"
            />
            <span className="min-w-0 space-y-1">
              <span className="flex items-center gap-2 text-sm font-medium text-foreground">
                {tBuild("label")}
                <span className="rounded-full bg-muted px-2 py-0.5 text-xs text-muted-foreground">
                  {tBuild("badge")}
                </span>
              </span>
              <span className="block text-xs text-muted-foreground">
                {tBuild("hint")}
              </span>
            </span>
          </label>

          <section className="space-y-1.5">
            <h4 className="text-sm font-semibold text-foreground">
              {t("ui.dialogSettings.filePreviewTitle")}
            </h4>
            {files.length > 0 ? (
              <FileStructurePreview files={files} modName={projectName} />
            ) : (
              <p className="text-xs text-muted-foreground">
                {t("ui.dialogSettings.filePreviewPending")}
              </p>
            )}
          </section>

          <section className="space-y-2 border-t border-border pt-3">
            <button
              type="button"
              onClick={() => setRawFilesOpen((v) => !v)}
              aria-expanded={rawFilesOpen}
              className="flex w-full items-center gap-2 text-left ui-interactive ui-focus-ring"
            >
              {rawFilesOpen ? (
                <ChevronDown
                  className="h-4 w-4 text-muted-foreground"
                  aria-hidden
                />
              ) : (
                <ChevronRight
                  className="h-4 w-4 text-muted-foreground"
                  aria-hidden
                />
              )}
              <span>
                <span className="block text-sm font-semibold text-foreground">
                  {t("rawFilesHeading")}
                </span>
                <span className="block text-xs text-muted-foreground">
                  {t("rawFilesDesc")}
                </span>
              </span>
            </button>
            {rawFilesOpen ? (
              <RawOverrideFilesManager projectId={projectId} allowReplacePath />
            ) : null}
          </section>

          <section className="space-y-2 border-t border-border pt-3">
            <button
              type="button"
              onClick={() => setReplaceOpen((v) => !v)}
              aria-expanded={replaceOpen}
              className="flex w-full items-center gap-2 text-left ui-interactive ui-focus-ring"
            >
              {replaceOpen ? (
                <ChevronDown className="h-4 w-4 text-destructive" aria-hidden />
              ) : (
                <ChevronRight
                  className="h-4 w-4 text-destructive"
                  aria-hidden
                />
              )}
              <AlertTriangle className="h-4 w-4 text-destructive" aria-hidden />
              <span className="text-sm font-semibold text-destructive">
                {t("overrideSectionTitle")}
              </span>
            </button>
            {replaceOpen ? (
              <div className="space-y-3 rounded-lg border border-destructive/40 bg-destructive/5 p-3">
                <p className="text-xs text-destructive">
                  {t("overrideSectionDanger")}
                </p>
                <p className="text-xs text-muted-foreground">
                  {t("overrideCrossLink")}
                </p>
                <div className="space-y-1.5">
                  <p className="text-sm font-semibold text-foreground">
                    {t("overrideReplacePathHeading")}
                  </p>
                  <p className="text-xs text-muted-foreground">
                    {t("overrideReplacePathDesc")}
                  </p>
                  <p className="text-xs font-medium text-foreground">
                    {t("overrideContentGroup")}
                  </p>
                  <div className="grid grid-cols-1 gap-1.5 sm:grid-cols-2">
                    {CONTENT_REPLACE_PATH_OPTIONS.map((opt) => (
                      <label
                        key={opt.key}
                        className={`flex cursor-pointer items-center gap-2 rounded-md border px-2.5 py-1.5 ui-interactive ${
                          replacePaths[opt.key]
                            ? "border-destructive/50 bg-destructive/10"
                            : "border-border bg-card"
                        }`}
                      >
                        <input
                          type="checkbox"
                          checked={!!replacePaths[opt.key]}
                          onChange={(e) =>
                            onToggleReplacePath(opt.key, e.target.checked)
                          }
                          className="h-3.5 w-3.5 shrink-0 accent-destructive"
                        />
                        <span className="min-w-0">
                          <span className="block text-xs">
                            {t(opt.labelKey)}
                          </span>
                          <span className="block truncate font-mono text-xs text-muted-foreground">
                            {opt.path}
                          </span>
                        </span>
                      </label>
                    ))}
                  </div>
                  <div className="space-y-1.5 pt-2">
                    <p className="text-xs font-medium text-destructive">
                      {t("overrideWorldGroup")}
                    </p>
                    <p className="text-xs text-destructive">
                      {t("overrideWorldDanger")}
                    </p>
                    <div className="grid grid-cols-1 gap-1.5 sm:grid-cols-2">
                      {WORLD_REPLACE_PATH_OPTIONS.map((opt) => (
                        <label
                          key={opt.key}
                          className={`flex cursor-pointer items-center gap-2 rounded-md border px-2.5 py-1.5 ui-interactive ${
                            replacePaths[opt.key]
                              ? "border-destructive/50 bg-destructive/10"
                              : "border-border bg-card"
                          }`}
                        >
                          <input
                            type="checkbox"
                            checked={!!replacePaths[opt.key]}
                            onChange={(e) =>
                              onToggleReplacePath(opt.key, e.target.checked)
                            }
                            className="h-3.5 w-3.5 shrink-0 accent-destructive"
                          />
                          <span className="min-w-0">
                            <span className="block text-xs">
                              {t(opt.labelKey)}
                            </span>
                            <span className="block truncate font-mono text-xs text-muted-foreground">
                              {opt.path}
                            </span>
                          </span>
                        </label>
                      ))}
                    </div>
                  </div>
                </div>
              </div>
            ) : null}
          </section>

          {showCoverageNote ? (
            <section className="space-y-2 border-t border-border pt-3">
              <div className="flex items-start gap-2.5 rounded-lg border border-border bg-muted/20 px-3 py-2.5">
                <AlertTriangle
                  className="mt-0.5 h-4 w-4 shrink-0 text-muted-foreground"
                  aria-hidden
                />
                <div className="min-w-0 flex-1 space-y-2">
                  <p className="text-xs text-muted-foreground">
                    {t("identityLoadedUnmodeledNote", {
                      count: UNSUPPORTED_OR_LOSSY_EXPORT_PATHS.length,
                    })}
                  </p>
                  <p className="text-xs font-medium text-foreground">
                    {t("identityLoadedUnmodeledPaths", {
                      count: UNSUPPORTED_OR_LOSSY_EXPORT_PATHS.length,
                    })}
                  </p>
                  <div className="grid grid-cols-1 gap-1.5 sm:grid-cols-2">
                    {UNSUPPORTED_OR_LOSSY_EXPORT_PATHS.map((path) => (
                      <code
                        key={path}
                        className="break-all rounded-md border border-border bg-card px-2 py-1 font-mono text-xs text-muted-foreground"
                      >
                        {path}
                      </code>
                    ))}
                  </div>
                  <p className="text-xs text-muted-foreground">
                    {t("identityLoadedUnmodeledAction")}
                  </p>
                </div>
              </div>
            </section>
          ) : null}

          <section className="space-y-2 border-t border-border pt-3">
            <button
              type="button"
              onClick={() => setAdvancedOpen((v) => !v)}
              aria-expanded={advancedOpen}
              className="flex w-full items-center gap-2 text-left ui-interactive ui-focus-ring"
            >
              {advancedOpen ? (
                <ChevronDown
                  className="h-4 w-4 text-muted-foreground"
                  aria-hidden
                />
              ) : (
                <ChevronRight
                  className="h-4 w-4 text-muted-foreground"
                  aria-hidden
                />
              )}
              <span className="text-sm font-semibold text-foreground">
                {t("step3AdvancedExportToggle")}
              </span>
            </button>
            {advancedOpen ? (
              <div className="space-y-2">
                <p className="text-xs text-muted-foreground">
                  {t("advancedExportIntro")}
                </p>
                <div className="flex items-start gap-2 rounded-md border border-warning/30 bg-warning/5 px-2.5 py-2">
                  <AlertTriangle
                    className="mt-0.5 h-3.5 w-3.5 shrink-0 text-warning"
                    aria-hidden
                  />
                  <p className="text-xs text-muted-foreground">
                    {t("advancedAppendWarning")}
                  </p>
                </div>
                <Button
                  onClick={onOpenAppend}
                  variant="outline"
                  className="w-full"
                >
                  {t("appendToExisting")}
                </Button>
                <Button
                  onClick={onOpenDiffMerge}
                  variant="outline"
                  className="w-full ui-interactive"
                >
                  {t("diffMerge")}
                </Button>
                <div className="rounded-lg border border-border bg-muted/20 p-3 space-y-2">
                  <div>
                    <p className="mb-1 text-sm font-semibold">
                      {t("fullPackModeTitle")}
                    </p>
                    <p className="text-xs text-muted-foreground">
                      {t("fullPackModeDesc")}
                    </p>
                    <p className="mt-1 text-xs text-warning">
                      {t("fullPackModeWarning")}
                    </p>
                  </div>
                  <BatchFlipButton
                    projectId={projectId}
                    onSuccess={onBatchFlipSuccess}
                    variant="outline"
                  />
                </div>
              </div>
            ) : null}
          </section>
        </div>

        <DialogFooter>
          <Button
            variant="outline"
            size="sm"
            onClick={() => onOpenChange(false)}
          >
            {t("ui.dialogCheck.close")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
