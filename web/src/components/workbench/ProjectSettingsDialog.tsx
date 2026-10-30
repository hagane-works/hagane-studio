"use client";

import { useEffect, useRef, useState } from "react";
import useSWR from "swr";
import { useTranslations } from "use-intl";
import { toast } from "sonner";
import { FolderOpen } from "lucide-react";

import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Separator } from "@/components/ui/separator";
import { projectsApi } from "@/lib/api";
import { ApiError, readableError } from "@/lib/api/_client";
import { FolderPicker } from "@/desktop/FolderPicker";
import { useProjectStore } from "@/lib/store/projectStore";
import {
  type ContentLocale,
  normalizeContentLocales,
} from "@/lib/store/contentLocaleStore";

import { StageScopePicker } from "@/components/start/StageScopePicker";
import {
  HOI4_SUPPORTED_LANGS,
  LANG_DISPLAY,
} from "@/components/shared/editors/displayNameLocsHelpers";
import { ProjectCoverUpload } from "@/components/workbench/ProjectCoverUpload";
import { DlcPicker } from "@/components/workbench/DlcPicker";
import { DEFAULT_SUPPORTED_VERSION } from "@/data/hoi4Versions";

const TAG_PRESETS = [
  "Alternative History",
  "Historical",
  "Gameplay",
  "Map",
  "Sound",
  "Translation",
  "Balance",
  "Fixes",
  "Graphics",
  "Ideologies",
  "National Focuses",
  "Technologies",
] as const;

interface ProjectSettingsDialogProps {
  projectId: number;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export function ProjectSettingsDialog({
  projectId,
  open,
  onOpenChange,
}: ProjectSettingsDialogProps) {
  const t = useTranslations("projectSettingsDialog");
  const tLayout = useTranslations("layout");
  const { currentProject, fetchProject, setContentLanguageEnabled } =
    useProjectStore();
  const [version, setVersion] = useState("");
  const [tagsText, setTagsText] = useState("");
  const [dependenciesText, setDependenciesText] = useState("");
  const [supportedVersion, setSupportedVersion] = useState("");

  const [updatingLanguage, setUpdatingLanguage] =
    useState<ContentLocale | null>(null);
  const [languageError, setLanguageError] = useState<string | null>(null);
  const [removedLanguage, setRemovedLanguage] = useState<ContentLocale | null>(
    null,
  );
  const languageEpochRef = useRef(0);
  const seededProjectRef = useRef<number | null>(null);

  const [ownedDlcs, setOwnedDlcs] = useState<string[] | null>(null);
  const [saving, setSaving] = useState(false);
  const [status, setStatus] = useState<"" | "saved" | "error">("");
  const [error, setError] = useState<string | null>(null);

  const [stageOpen, setStageOpen] = useState(false);

  const baseMods = useSWR(open ? ["project-base-mods", projectId] : null, () =>
    projectsApi.baseMods(projectId),
  );
  const [baseModPick, setBaseModPick] = useState<{
    dependency?: string;
    initialPath: string;
  } | null>(null);
  const [baseModBusy, setBaseModBusy] = useState(false);
  const [baseModError, setBaseModError] = useState<string | null>(null);

  useEffect(() => {
    if (!open) {
      seededProjectRef.current = null;
      return;
    }
    if (
      !currentProject ||
      currentProject.id !== projectId ||
      seededProjectRef.current === projectId
    ) {
      return;
    }
    seededProjectRef.current = projectId;
    setVersion(currentProject.version || "1.0");
    setTagsText((currentProject.tags || ["Alternative History"]).join("\n"));
    setDependenciesText((currentProject.dependencies || []).join("\n"));
    setSupportedVersion(
      currentProject.supported_version || DEFAULT_SUPPORTED_VERSION,
    );
    setOwnedDlcs(currentProject.owned_dlcs ?? null);
    setStatus("");
    setError(null);
  }, [open, currentProject, projectId]);

  useEffect(() => {
    if (!open) return;
    languageEpochRef.current += 1;
    setUpdatingLanguage(null);
    setLanguageError(null);
    setRemovedLanguage(null);
  }, [open, projectId]);

  const parseList = (raw: string): string[] =>
    raw
      .split(/[\n,]+/)
      .map((s) => s.trim())
      .filter(Boolean);

  const toggleTag = (tag: string) => {
    const current = new Set(parseList(tagsText));
    if (current.has(tag)) {
      current.delete(tag);
    } else {
      current.add(tag);
    }
    setTagsText(Array.from(current).join("\n"));
  };

  const baseModFolders = new Map(
    (baseMods.data?.base_mods ?? []).map((mod) => [mod.name, mod]),
  );

  const changeBaseMod = async (folder: string, dependency?: string) => {
    setBaseModBusy(true);
    setBaseModError(null);
    try {
      const result = await projectsApi.setBaseMod(
        projectId,
        folder,
        dependency,
      );
      if (!dependency) {
        setDependenciesText((text) => {
          const names = parseList(text);
          return names.includes(result.dependency)
            ? text
            : [...names, result.dependency].join("\n");
        });
      }
      await baseMods.mutate(
        { base_mods: result.base_mods },
        { revalidate: false },
      );
      await fetchProject(projectId);
    } catch (cause) {
      setBaseModError(
        cause instanceof ApiError &&
          (cause.payload as { code?: unknown } | null)?.code === "not_a_mod"
          ? t("baseMod.notAMod")
          : readableError(cause),
      );
    } finally {
      setBaseModBusy(false);
    }
  };

  const exportLanguages = normalizeContentLocales(
    currentProject?.export_languages,
  );

  const toggleLanguage = async (lang: ContentLocale) => {
    if (lang === "en") return;
    const enable = !exportLanguages.includes(lang);
    const requestProjectId = projectId;
    const requestEpoch = languageEpochRef.current;
    setUpdatingLanguage(lang);
    setLanguageError(null);
    setRemovedLanguage(null);
    try {
      await setContentLanguageEnabled(lang, enable);
      if (
        languageEpochRef.current !== requestEpoch ||
        useProjectStore.getState().currentProject?.id !== requestProjectId
      ) {
        return;
      }
      if (!enable) setRemovedLanguage(lang);
    } catch (cause) {
      if (
        languageEpochRef.current !== requestEpoch ||
        useProjectStore.getState().currentProject?.id !== requestProjectId
      ) {
        return;
      }
      setLanguageError(
        (cause as Error).message || tLayout("contentLanguageUnknownError"),
      );
    } finally {
      if (
        languageEpochRef.current === requestEpoch &&
        useProjectStore.getState().currentProject?.id === requestProjectId
      ) {
        setUpdatingLanguage(null);
      }
    }
  };

  const handleSave = async () => {
    setSaving(true);
    setError(null);
    setStatus("");
    try {
      await projectsApi.update(projectId, {
        version: version.trim() || "1.0",
        tags: parseList(tagsText),
        dependencies: parseList(dependenciesText),
        supported_version: supportedVersion.trim() || DEFAULT_SUPPORTED_VERSION,
        owned_dlcs: ownedDlcs,
      });
      await fetchProject(projectId);

      void baseMods.mutate();
      setStatus("saved");

      toast.success(t("savedToast"));
    } catch (e) {
      const msg = (e as Error).message || t("projectCover.unknownError");
      setError(msg);
      setStatus("error");
      toast.error(t("saveFailed", { msg }));
    } finally {
      setSaving(false);
    }
  };

  const activeTags = new Set(parseList(tagsText));
  return (
    <Dialog open={open} onOpenChange={onOpenChange} disablePointerDismissal>
      <DialogContent className="sm:max-w-2xl max-h-[88vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>{t("title")}</DialogTitle>
        </DialogHeader>

        <p className="text-xs text-muted-foreground mb-3">
          {t.rich("intro", {
            mono: (chunks) => <span className="font-mono">{chunks}</span>,
            b: (chunks) => <span className="font-semibold">{chunks}</span>,
          })}
        </p>

        <Separator className="mb-4" />

        <div className="grid grid-cols-2 gap-3">
          <div className="space-y-1">
            <Label>version</Label>
            <Input
              value={version}
              onChange={(e) => setVersion(e.target.value)}
              placeholder="1.0"
              className="font-mono text-xs"
            />
            <p className="text-xs text-muted-foreground">{t("versionHint")}</p>
          </div>
          <div className="space-y-1">
            <Label>supported_version</Label>
            <Input
              value={supportedVersion}
              onChange={(e) => setSupportedVersion(e.target.value)}
              placeholder={DEFAULT_SUPPORTED_VERSION}
              className="font-mono text-xs"
            />
            <p className="text-xs text-muted-foreground">
              {t.rich("supportedVersionHint", {
                mono: (chunks) => <span className="font-mono">{chunks}</span>,
              })}
            </p>
          </div>

          <div className="space-y-1">
            <Label>{t("stageScopeLabel")}</Label>
            <div className="flex items-center gap-2">
              <Button
                type="button"
                variant="outline"
                size="sm"
                onClick={() => setStageOpen(true)}
              >
                {t("stageScopeCta")}
              </Button>
              <span className="text-xs text-muted-foreground">
                {(currentProject?.stage_state_ids?.length ?? 0) > 0
                  ? t("stageScopeCount", {
                      count: currentProject?.stage_state_ids?.length ?? 0,
                    })
                  : t("stageScopeEmpty")}
              </span>
            </div>
          </div>
        </div>

        <div className="mt-3">
          <ProjectCoverUpload
            key={projectId}
            projectId={projectId}
            previewUrl={
              currentProject?.picture === "thumbnail.png"
                ? projectsApi.coverUrl(projectId)
                : null
            }
            onUpload={(file) => projectsApi.uploadCover(projectId, file)}
            onDelete={() => projectsApi.deleteCover(projectId)}
            onChanged={() => fetchProject(projectId)}
          />
        </div>

        <Separator className="my-4" />

        <div className="space-y-1">
          <Label>{t("tagsLabel")}</Label>
          <div className="flex flex-wrap gap-1.5 mb-2">
            {TAG_PRESETS.map((tag) => {
              const active = activeTags.has(tag);
              return (
                <button
                  key={tag}
                  type="button"
                  onClick={() => toggleTag(tag)}
                  className={
                    "rounded-full px-2 py-0.5 text-xs border transition-colors " +
                    (active
                      ? "border-primary bg-primary text-primary-foreground"
                      : "border-input bg-background hover:bg-muted")
                  }
                >
                  {tag}
                </button>
              );
            })}
          </div>
          <textarea
            value={tagsText}
            onChange={(e) => setTagsText(e.target.value)}
            placeholder={"Alternative History\nHistorical"}
            className="w-full min-h-16 px-2 py-1 text-xs font-mono border rounded-md bg-background"
          />
          <p className="text-xs text-muted-foreground">
            {t("tagsHintBefore")}{" "}
            <span className="font-mono">
              tags={"{"} &quot;X&quot; &quot;Y&quot; {"}"}
            </span>{" "}
            {t("tagsHintAfter")}
          </p>
        </div>

        <Separator className="my-4" />

        <div className="space-y-1">
          <Label>{t("depsLabel")}</Label>
          <textarea
            value={dependenciesText}
            onChange={(e) => setDependenciesText(e.target.value)}
            placeholder={"Old World Blues\nKaiserreich"}
            className="w-full min-h-16 px-2 py-1 text-xs font-mono border rounded-md bg-background"
          />
          <p className="text-xs text-muted-foreground">
            {t("depsHintBefore")}{" "}
            <span className="font-mono">dependencies={"{}"}</span>{" "}
            {t("depsHintAfter")}
          </p>
          <div className="flex items-center gap-2 pt-1">
            <Button
              type="button"
              variant="outline"
              size="sm"
              disabled={baseModBusy}
              onClick={() => setBaseModPick({ initialPath: "" })}
            >
              <FolderOpen className="h-3.5 w-3.5" />
              {t("baseMod.add")}
            </Button>
            <span className="text-xs text-muted-foreground">
              {t("baseMod.addHint")}
            </span>
          </div>
          {baseModError && (
            <p className="text-xs text-destructive" role="alert">
              {baseModError}
            </p>
          )}

          {(currentProject?.dependencies ?? [])
            .filter(
              (d): d is string => typeof d === "string" && d.trim().length > 0,
            )
            .map((dep) => {
              const folder = baseModFolders.get(dep)?.folder ?? "";
              const found = baseModFolders.get(dep)?.found ?? false;
              return (
                <div
                  key={dep}
                  className="mt-2 space-y-2 rounded-md border border-border bg-muted/20 px-2.5 py-2"
                >
                  <div className="flex items-start gap-2">
                    <div className="min-w-0 flex-1 text-xs">
                      <p className="truncate font-mono text-foreground">
                        {dep}
                      </p>
                      {!folder ? (
                        <p className="text-muted-foreground">
                          {t("baseMod.noFolder")}
                        </p>
                      ) : found ? (
                        <p
                          className="truncate font-mono text-muted-foreground"
                          title={folder}
                        >
                          {folder}
                        </p>
                      ) : (
                        <p className="break-all text-warning">
                          {t("baseMod.folderMissing", { folder })}
                        </p>
                      )}
                    </div>
                    <Button
                      type="button"
                      variant="outline"
                      size="sm"
                      disabled={baseModBusy}
                      onClick={() =>
                        setBaseModPick({
                          dependency: dep,
                          initialPath: found ? folder : "",
                        })
                      }
                    >
                      {folder ? t("baseMod.change") : t("baseMod.choose")}
                    </Button>
                    {folder && (
                      <Button
                        type="button"
                        variant="ghost"
                        size="sm"
                        disabled={baseModBusy}
                        onClick={() => void changeBaseMod("", dep)}
                      >
                        {t("baseMod.clear")}
                      </Button>
                    )}
                  </div>
                </div>
              );
            })}
        </div>

        <Separator className="my-4" />

        <DlcPicker
          value={ownedDlcs}
          onChange={setOwnedDlcs}
          disabled={saving}
        />

        <Separator className="my-4" />

        <div className="space-y-2">
          <Label>{t("langsLabel")}</Label>
          <p className="text-xs text-muted-foreground">{t("langsHint")}</p>
          <p className="text-xs text-warning">
            {tLayout("contentLanguageRemovalWarning")}
          </p>
          <div
            className="flex flex-wrap gap-2"
            aria-busy={updatingLanguage !== null}
          >
            {HOI4_SUPPORTED_LANGS.map((lang) => {
              const checked = exportLanguages.includes(lang);
              const disabled = lang === "en" || updatingLanguage !== null;
              return (
                <label
                  key={lang}
                  className={
                    "flex items-center gap-1.5 rounded-md border px-2 py-1 text-xs cursor-pointer transition-colors " +
                    (checked
                      ? "border-primary bg-primary/10 text-foreground"
                      : "border-input bg-background hover:bg-muted text-muted-foreground") +
                    (disabled ? " opacity-80 cursor-default" : "")
                  }
                >
                  <input
                    type="checkbox"
                    checked={checked}
                    disabled={disabled}
                    onChange={() => void toggleLanguage(lang)}
                    className="accent-accent h-3.5 w-3.5"
                  />
                  <span className="font-mono w-5 text-center">
                    {LANG_DISPLAY[lang] ?? lang.toUpperCase()}
                  </span>
                  <span>{t(`langName.${lang}`)}</span>
                  {lang === "en" && (
                    <span className="text-micro text-amber-600 dark:text-amber-400">
                      ({t("langForced")})
                    </span>
                  )}
                </label>
              );
            })}
          </div>
          {updatingLanguage ? (
            <p className="text-xs text-muted-foreground" role="status">
              {tLayout("contentLanguageUpdating", {
                language: t(`langName.${updatingLanguage}`),
              })}
            </p>
          ) : null}
          {removedLanguage ? (
            <p className="text-xs text-warning" role="status">
              {tLayout("contentLanguageRemovedNotice", {
                language: t(`langName.${removedLanguage}`),
              })}
            </p>
          ) : null}
          {languageError ? (
            <p className="text-xs text-destructive" role="alert">
              {tLayout("contentLanguageUpdateFailed", {
                message: languageError,
              })}
            </p>
          ) : null}
        </div>

        <Separator className="my-4" />

        <div className="flex items-center gap-3">
          <Button onClick={handleSave} disabled={saving} size="sm">
            {saving ? t("saving") : t("save")}
          </Button>
          {status === "saved" && (
            <span className="text-xs text-emerald-600 dark:text-emerald-400">
              {t("savedIndicator")}
            </span>
          )}
          {error && (
            <span className="text-xs text-red-600 dark:text-red-400">
              ✗ {error}
            </span>
          )}
        </div>

        <StageScopePicker
          projectId={projectId}
          open={stageOpen}
          onOpenChange={setStageOpen}
          value={currentProject?.stage_state_ids ?? null}
          onSaved={() => void fetchProject(projectId)}
        />
        <FolderPicker
          open={baseModPick !== null}
          onOpenChange={(next) => !next && setBaseModPick(null)}
          title={
            baseModPick?.dependency
              ? t("baseMod.pickTitleFor", { name: baseModPick.dependency })
              : t("baseMod.pickTitle")
          }
          description={t("baseMod.pickHint")}
          initialPath={baseModPick?.initialPath ?? ""}
          onPick={(folder) => {
            const dependency = baseModPick?.dependency;
            setBaseModPick(null);
            void changeBaseMod(folder, dependency);
          }}
        />
      </DialogContent>
    </Dialog>
  );
}
