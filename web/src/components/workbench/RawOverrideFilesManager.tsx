"use client";

import { useCallback, useEffect, useState } from "react";
import { useLocale, useTranslations } from "use-intl";
import { toast } from "sonner";
import {
  Plus,
  Trash2,
  FileWarning,
  FileX2,
  Upload,
  Sparkles,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import {
  ApiError,
  rawFilesApi,
  type RawFileRow,
  type RawTemplateInfo,
} from "@/lib/api";
import { useConfirm } from "@/components/shared/ConfirmDialogProvider";

interface RawOverrideFilesManagerProps {
  projectId: number;
  allowReplacePath?: boolean;
}

const EMPTY_DRAFT = {
  dest_path: "",
  content: "",
  generate_replace_path: false,
  note: "",
};
const EMPTY_SHADOW_DRAFT = {
  dest_path: "",
  target_layer: "",
  target_sha256: "",
  note: "",
};
const SHA256_RE = /^[0-9a-f]{64}$/;

type EmptyShadowFailure = { message: string; code?: string };

function emptyShadowFailure(error: unknown): EmptyShadowFailure {
  if (
    error instanceof ApiError &&
    error.payload &&
    typeof error.payload === "object"
  ) {
    const payload = error.payload as { code?: unknown; message?: unknown };
    return {
      message:
        typeof payload.message === "string" ? payload.message : error.message,
      code: typeof payload.code === "string" ? payload.code : undefined,
    };
  }
  return { message: error instanceof Error ? error.message : String(error) };
}

const TEMPLATE_KEY_RE = /^[a-z0-9_]{2,40}$/;
const LOCALIZED_TEMPLATE_KEYS = new Set([
  "scripted_gui",
  "on_actions",
  "game_rules",
  "ai_strategy",
  "gfx_sprite",
  "frontendgamesetup_slotsize",
  "defines",
  "bop",
  "music",
  "peace_conference",
]);

export function RawOverrideFilesManager({
  projectId,
  allowReplacePath = false,
}: RawOverrideFilesManagerProps) {
  const t = useTranslations("export");
  const tTemplates = useTranslations("rawTemplateCatalog");
  const locale = useLocale() as
    "en" | "zh" | "ru" | "de" | "es" | "fr" | "ja" | "ko";
  const confirmDialog = useConfirm();
  const [files, setFiles] = useState<RawFileRow[]>([]);
  const [loading, setLoading] = useState(true);
  const [adding, setAdding] = useState(false);
  const [draft, setDraft] = useState(EMPTY_DRAFT);
  const [saving, setSaving] = useState(false);

  const [addingEmptyShadow, setAddingEmptyShadow] = useState(false);
  const [emptyShadowDraft, setEmptyShadowDraft] = useState(EMPTY_SHADOW_DRAFT);
  const [emptyShadowError, setEmptyShadowError] =
    useState<EmptyShadowFailure | null>(null);

  const [addingBinary, setAddingBinary] = useState(false);
  const [binDest, setBinDest] = useState("");
  const [binFile, setBinFile] = useState<File | null>(null);
  const [binReplace, setBinReplace] = useState(false);
  const [binNote, setBinNote] = useState("");

  const [addingTemplate, setAddingTemplate] = useState(false);
  const [tmplKey, setTmplKey] = useState("");
  const [tmplTitle, setTmplTitle] = useState("");
  const [tmplVars, setTmplVars] = useState("");

  const [templates, setTemplates] = useState<RawTemplateInfo[] | null>(null);
  const [templatesLoading, setTemplatesLoading] = useState(false);
  const [templatesFailed, setTemplatesFailed] = useState(false);
  const [selectedTemplateKey, setSelectedTemplateKey] = useState<string | null>(
    null,
  );
  const [genericKey, setGenericKey] = useState("");

  const refresh = useCallback(async () => {
    try {
      setFiles(await rawFilesApi.list(projectId));
    } catch (err) {
      console.error("raw-files list failed", err);
    } finally {
      setLoading(false);
    }
  }, [projectId]);

  useEffect(() => {
    void refresh();
  }, [refresh]);

  const handleCreate = async () => {
    const destPath = draft.dest_path.trim();
    if (!destPath) {
      toast.error(t("rawFileDestRequired"));
      return;
    }
    setSaving(true);
    try {
      await rawFilesApi.create(projectId, {
        dest_path: destPath,
        content: draft.content,
        generate_replace_path: allowReplacePath && draft.generate_replace_path,
        note: draft.note.trim(),
      });
      toast.success(t("rawFileCreated"));
      setDraft(EMPTY_DRAFT);
      setAdding(false);
      await refresh();
    } catch (err) {
      toast.error(t("rawFileCreateFailed", { err: String(err) }));
    } finally {
      setSaving(false);
    }
  };

  const resetEmptyShadowDraft = () => {
    setEmptyShadowDraft(EMPTY_SHADOW_DRAFT);
    setEmptyShadowError(null);
  };

  const handleCreateEmptyShadow = async () => {
    const destPath = emptyShadowDraft.dest_path.trim();
    const targetLayer = emptyShadowDraft.target_layer.trim();
    const targetSha256 = emptyShadowDraft.target_sha256.trim().toLowerCase();
    if (!destPath || !targetLayer || !SHA256_RE.test(targetSha256)) {
      setEmptyShadowError({ message: t("rawFileEmptyShadowEvidenceRequired") });
      return;
    }
    setSaving(true);
    setEmptyShadowError(null);
    try {
      await rawFilesApi.createEmptyShadow(projectId, {
        dest_path: destPath,
        target_layer: targetLayer,
        target_sha256: targetSha256,
        note: emptyShadowDraft.note.trim(),
      });
      toast.success(t("rawFileCreated"));
      resetEmptyShadowDraft();
      setAddingEmptyShadow(false);
      await refresh();
    } catch (err) {
      setEmptyShadowError(emptyShadowFailure(err));
    } finally {
      setSaving(false);
    }
  };

  const resetBinaryDraft = () => {
    setBinDest("");
    setBinFile(null);
    setBinReplace(false);
    setBinNote("");
  };

  const handleCreateBinary = async () => {
    const destPath = binDest.trim();
    if (!destPath) {
      toast.error(t("rawFileDestRequired"));
      return;
    }
    if (!binFile) {
      toast.error(t("rawFileBinaryNoFile"));
      return;
    }
    setSaving(true);
    try {
      await rawFilesApi.createBinary(projectId, {
        dest_path: destPath,
        file: binFile,
        generate_replace_path: allowReplacePath && binReplace,
        note: binNote.trim(),
      });
      toast.success(t("rawFileCreated"));
      resetBinaryDraft();
      setAddingBinary(false);
      await refresh();
    } catch (err) {
      toast.error(t("rawFileCreateFailed", { err: String(err) }));
    } finally {
      setSaving(false);
    }
  };

  const handleGenerateTemplate = async () => {
    const panelKey = tmplKey.trim();
    if (!panelKey) {
      toast.error(t("rawFileTemplateKeyRequired"));
      return;
    }

    const variables = tmplVars
      .split("\n")
      .map((l) => l.trim())
      .filter(Boolean)
      .map((l) => {
        const sp = l.search(/\s/);
        return sp === -1
          ? { key: l, label: l }
          : { key: l.slice(0, sp), label: l.slice(sp + 1).trim() };
      });
    setSaving(true);
    try {
      await rawFilesApi.fromTemplate(projectId, {
        panel_key: panelKey,
        title: tmplTitle.trim(),
        variables,
      });
      toast.success(t("rawFileTemplateGenerated"));
      setTmplKey("");
      setTmplTitle("");
      setTmplVars("");
      setAddingTemplate(false);
      await refresh();
    } catch (err) {
      toast.error(t("rawFileCreateFailed", { err: String(err) }));
    } finally {
      setSaving(false);
    }
  };

  const openTemplatePicker = () => {
    setAddingTemplate(true);
    setSelectedTemplateKey(null);
    setGenericKey("");
    if (templates !== null || templatesLoading) return;
    setTemplatesLoading(true);
    setTemplatesFailed(false);
    void rawFilesApi
      .listTemplates(projectId)
      .then((res) => {
        setTemplates(res.templates);
      })
      .catch((err) => {
        console.error("raw-file template registry fetch failed", err);
        setTemplatesFailed(true);
      })
      .finally(() => {
        setTemplatesLoading(false);
      });
  };

  const closeTemplatePicker = () => {
    setAddingTemplate(false);
    setSelectedTemplateKey(null);
    setGenericKey("");
    setTmplKey("");
    setTmplTitle("");
    setTmplVars("");
  };

  const selectedTemplate =
    templates?.find((tpl) => tpl.key === selectedTemplateKey) ?? null;

  const tplLabel = (tpl: RawTemplateInfo) =>
    LOCALIZED_TEMPLATE_KEYS.has(tpl.key)
      ? tTemplates(`${tpl.key}.label`)
      : tpl.labels?.[locale] ||
        tpl.labels?.en ||
        tpl.label_en ||
        tpl.label_zh ||
        tpl.key;
  const tplDesc = (tpl: RawTemplateInfo) =>
    LOCALIZED_TEMPLATE_KEYS.has(tpl.key)
      ? tTemplates(`${tpl.key}.description`)
      : tpl.descriptions?.[locale] ||
        tpl.descriptions?.en ||
        tpl.desc_en ||
        tpl.desc_zh ||
        "";

  const handleGenerateGeneric = async () => {
    const tpl = selectedTemplate;
    if (!tpl) return;
    const seed = genericKey.trim();
    if (tpl.main_param && !TEMPLATE_KEY_RE.test(seed)) {
      toast.error(t("rawFileTemplateKeyInvalid"));
      return;
    }
    setSaving(true);
    try {
      const rows = await rawFilesApi.fromTemplateGeneric(
        projectId,
        tpl.key,
        seed,
        tpl,
      );
      toast.success(
        t("rawFileTemplateGeneratedPath", {
          path: rows.map((r) => r.dest_path).join(", "),
        }),
      );
      closeTemplatePicker();
      await refresh();
    } catch (err) {
      toast.error(t("rawFileCreateFailed", { err: String(err) }));
    } finally {
      setSaving(false);
    }
  };

  const handleDelete = async (rf: RawFileRow) => {
    if (
      !(await confirmDialog({
        body: t("rawFileDeleteConfirm", { path: rf.dest_path }),
        danger: true,
      }))
    )
      return;
    try {
      await rawFilesApi.remove(projectId, rf.id);
      toast.success(t("rawFileDeleted"));
      await refresh();
    } catch (err) {
      toast.error(t("rawFileDeleteFailed", { err: String(err) }));
    }
  };

  return (
    <div className="space-y-3">
      <div className="flex items-start gap-2">
        <FileWarning className="h-4 w-4 text-muted-foreground shrink-0 mt-0.5" />
        <div className="space-y-1">
          <p className="text-sm font-semibold text-foreground">
            {t("rawFilesHeading")}
          </p>
          <p className="text-xs text-muted-foreground">{t("rawFilesDesc")}</p>
        </div>
      </div>

      <p className="text-xs text-muted-foreground italic">
        {t("rawFileTlarHint")}
      </p>

      {loading ? (
        <p className="text-xs text-muted-foreground">{t("rawFileLoading")}</p>
      ) : files.length === 0 ? (
        <p className="text-xs text-muted-foreground">{t("rawFileEmpty")}</p>
      ) : (
        <div className="space-y-2">
          {files.map((rf) => (
            <RawFileCard
              key={rf.id}
              file={rf}
              projectId={projectId}
              allowReplacePath={allowReplacePath}
              onSaved={refresh}
              onDelete={() => handleDelete(rf)}
            />
          ))}
        </div>
      )}

      {adding ? (
        <div className="rounded-md border border-destructive/40 bg-destructive/5 p-3 space-y-2">
          <RawFileFields
            destPath={draft.dest_path}
            content={draft.content}
            generateReplacePath={draft.generate_replace_path}
            allowReplacePath={allowReplacePath}
            note={draft.note}
            onChange={(patch) => setDraft((d) => ({ ...d, ...patch }))}
          />
          <div className="flex items-center gap-2">
            <Button size="sm" onClick={handleCreate} disabled={saving}>
              {saving ? t("rawFileSaving") : t("rawFileCreate")}
            </Button>
            <Button
              size="sm"
              variant="ghost"
              onClick={() => {
                setAdding(false);
                setDraft(EMPTY_DRAFT);
              }}
            >
              {t("rawFileCancel")}
            </Button>
          </div>
        </div>
      ) : addingEmptyShadow ? (
        <div className="rounded-md border border-warning/40 bg-warning/5 p-3 space-y-3">
          <div className="space-y-1">
            <p className="text-sm font-semibold text-foreground">
              {t("rawFileEmptyShadowTitle")}
            </p>
            <p className="text-xs text-muted-foreground">
              {t("rawFileEmptyShadowDesc")}
            </p>
          </div>
          <div className="rounded-md border border-warning/30 bg-warning/10 p-2 text-xs text-foreground">
            {t("rawFileEmptyShadowEvidenceWarning")}
          </div>
          <div className="space-y-1">
            <label className="text-xs font-medium text-foreground">
              {t("rawFileDestLabel")}
            </label>
            <Input
              value={emptyShadowDraft.dest_path}
              onChange={(e) =>
                setEmptyShadowDraft((draftValue) => ({
                  ...draftValue,
                  dest_path: e.target.value,
                }))
              }
              placeholder="common/on_actions/exact_source_file.txt"
              className="font-mono text-xs"
            />
          </div>
          <div className="space-y-1">
            <label className="text-xs font-medium text-foreground">
              {t("rawFileEmptyShadowLayerLabel")}
            </label>
            <Input
              value={emptyShadowDraft.target_layer}
              onChange={(e) =>
                setEmptyShadowDraft((draftValue) => ({
                  ...draftValue,
                  target_layer: e.target.value,
                }))
              }
              placeholder={t("rawFileEmptyShadowLayerPlaceholder")}
            />
          </div>
          <div className="space-y-1">
            <label className="text-xs font-medium text-foreground">
              {t("rawFileEmptyShadowHashLabel")}
            </label>
            <Input
              value={emptyShadowDraft.target_sha256}
              onChange={(e) =>
                setEmptyShadowDraft((draftValue) => ({
                  ...draftValue,
                  target_sha256: e.target.value,
                }))
              }
              placeholder={t("rawFileEmptyShadowHashPlaceholder")}
              className="font-mono text-xs"
              spellCheck={false}
            />
          </div>
          <div className="space-y-1">
            <label className="text-xs font-medium text-foreground">
              {t("rawFileNoteLabel")}
            </label>
            <Input
              value={emptyShadowDraft.note}
              onChange={(e) =>
                setEmptyShadowDraft((draftValue) => ({
                  ...draftValue,
                  note: e.target.value,
                }))
              }
              placeholder={t("rawFileNotePlaceholder")}
            />
          </div>
          {emptyShadowError ? (
            <div
              role="alert"
              className="rounded-md border border-destructive/40 bg-destructive/10 p-2 text-xs text-destructive"
            >
              <p>
                {t("rawFileEmptyShadowRejected", {
                  message: emptyShadowError.message,
                })}
              </p>
              {emptyShadowError.code ? (
                <p className="mt-1 font-mono">
                  {t("rawFileEmptyShadowPolicyCode", {
                    code: emptyShadowError.code,
                  })}
                </p>
              ) : null}
            </div>
          ) : null}
          <div className="flex items-center gap-2">
            <Button
              size="sm"
              onClick={handleCreateEmptyShadow}
              disabled={saving}
            >
              {saving ? t("rawFileSaving") : t("rawFileEmptyShadowSubmit")}
            </Button>
            <Button
              size="sm"
              variant="ghost"
              onClick={() => {
                setAddingEmptyShadow(false);
                resetEmptyShadowDraft();
              }}
            >
              {t("rawFileCancel")}
            </Button>
          </div>
        </div>
      ) : addingBinary ? (
        <div className="rounded-md border border-destructive/40 bg-destructive/5 p-3 space-y-2">
          <p className="text-xs text-muted-foreground">
            {t("rawFileBinaryDesc")}
          </p>
          <div className="space-y-1">
            <label className="text-xs font-medium text-foreground">
              {t("rawFileDestLabel")}
            </label>
            <Input
              value={binDest}
              onChange={(e) => setBinDest(e.target.value)}
              placeholder="gfx/interface/your_override.dds"
              className="font-mono text-xs"
            />
          </div>
          <div className="space-y-1">
            <label className="text-xs font-medium text-foreground">
              {t("rawFileBinaryFileLabel")}
            </label>
            <input
              type="file"
              onChange={(e) => setBinFile(e.target.files?.[0] ?? null)}
              className="block w-full text-xs text-muted-foreground file:mr-2 file:rounded file:border-0 file:bg-muted file:px-2 file:py-1 file:text-xs"
            />
          </div>
          {allowReplacePath && (
            <label className="flex items-start gap-2 cursor-pointer">
              <input
                type="checkbox"
                checked={binReplace}
                onChange={(e) => setBinReplace(e.target.checked)}
                className="accent-destructive h-4 w-4 mt-0.5"
              />
              <span className="text-xs text-muted-foreground">
                {t("rawFileGenReplacePath")}
              </span>
            </label>
          )}
          <div className="space-y-1">
            <label className="text-xs font-medium text-foreground">
              {t("rawFileNoteLabel")}
            </label>
            <Input
              value={binNote}
              onChange={(e) => setBinNote(e.target.value)}
              placeholder={t("rawFileNotePlaceholder")}
            />
          </div>
          <div className="flex items-center gap-2">
            <Button size="sm" onClick={handleCreateBinary} disabled={saving}>
              {saving ? t("rawFileSaving") : t("rawFileCreate")}
            </Button>
            <Button
              size="sm"
              variant="ghost"
              onClick={() => {
                setAddingBinary(false);
                resetBinaryDraft();
              }}
            >
              {t("rawFileCancel")}
            </Button>
          </div>
        </div>
      ) : addingTemplate ? (
        (() => {
          const scriptedGuiForm = (
            <div className="space-y-2">
              <p className="text-xs text-muted-foreground">
                {t("rawFileTemplateDesc")}
              </p>
              <div className="space-y-1">
                <label className="text-xs font-medium text-foreground">
                  {t("rawFileTemplateKeyLabel")}
                </label>
                <Input
                  value={tmplKey}
                  onChange={(e) => setTmplKey(e.target.value)}
                  placeholder="fng_legit_overview"
                  className="font-mono text-xs"
                />
              </div>
              <div className="space-y-1">
                <label className="text-xs font-medium text-foreground">
                  {t("rawFileTemplateTitleLabel")}
                </label>
                <Input
                  value={tmplTitle}
                  onChange={(e) => setTmplTitle(e.target.value)}
                />
              </div>
              <div className="space-y-1">
                <label className="text-xs font-medium text-foreground">
                  {t("rawFileTemplateVarsLabel")}
                </label>
                <Textarea
                  value={tmplVars}
                  onChange={(e) => setTmplVars(e.target.value)}
                  placeholder={t("rawFileTemplateVarsPlaceholder")}
                  className="font-mono text-xs min-h-24"
                  spellCheck={false}
                />
              </div>
              <div className="flex items-center gap-2">
                <Button
                  size="sm"
                  onClick={handleGenerateTemplate}
                  disabled={saving}
                >
                  {saving ? t("rawFileSaving") : t("rawFileTemplateGenerate")}
                </Button>
                <Button size="sm" variant="ghost" onClick={closeTemplatePicker}>
                  {t("rawFileCancel")}
                </Button>
              </div>
            </div>
          );
          return (
            <div className="rounded-md border border-destructive/40 bg-destructive/5 p-3 space-y-3">
              {templatesLoading ? (
                <>
                  <p className="text-xs text-muted-foreground">
                    {t("rawFileTemplateLoading")}
                  </p>
                  <Button
                    size="sm"
                    variant="ghost"
                    onClick={closeTemplatePicker}
                  >
                    {t("rawFileCancel")}
                  </Button>
                </>
              ) : templates === null ? (
                <>
                  {templatesFailed ? (
                    <p className="text-xs text-muted-foreground">
                      {t("rawFileTemplateListFailed")}
                    </p>
                  ) : null}
                  {scriptedGuiForm}
                </>
              ) : (
                <>
                  <div className="space-y-1">
                    <p className="text-sm font-semibold text-foreground">
                      {t("rawFileTemplatePickerTitle")}
                    </p>
                    <p className="text-xs text-muted-foreground">
                      {t("rawFileTemplatePickerSubtitle")}
                    </p>
                  </div>
                  <div className="grid gap-2 sm:grid-cols-2">
                    {templates.map((tpl) => (
                      <button
                        key={tpl.key}
                        type="button"
                        onClick={() => {
                          setSelectedTemplateKey(tpl.key);
                          setGenericKey("");
                        }}
                        className={`rounded-md border p-2 text-left transition-colors ${
                          selectedTemplateKey === tpl.key
                            ? "border-destructive bg-destructive/10"
                            : "border-destructive/20 bg-background/60 hover:border-destructive/50"
                        }`}
                      >
                        <span className="block text-xs font-semibold text-foreground">
                          {tplLabel(tpl)}
                        </span>
                        {tplDesc(tpl) ? (
                          <span className="mt-0.5 block text-micro text-muted-foreground">
                            {tplDesc(tpl)}
                          </span>
                        ) : null}
                      </button>
                    ))}
                  </div>
                  {selectedTemplate &&
                  selectedTemplate.key === "scripted_gui" ? (
                    <div className="border-t border-destructive/20 pt-2">
                      {scriptedGuiForm}
                    </div>
                  ) : selectedTemplate ? (
                    <div className="space-y-2 border-t border-destructive/20 pt-2">
                      {selectedTemplate.main_param ? (
                        <div className="space-y-1">
                          <label className="text-xs font-medium text-foreground">
                            {t("rawFileTemplateKeyGenericLabel")}
                          </label>
                          <Input
                            value={genericKey}
                            onChange={(e) => setGenericKey(e.target.value)}
                            placeholder="my_tweaks"
                            className="font-mono text-xs"
                          />
                          <p className="text-micro text-muted-foreground">
                            {t("rawFileTemplateKeyGenericHelp")}
                          </p>
                        </div>
                      ) : null}
                      <div className="flex items-center gap-2">
                        <Button
                          size="sm"
                          onClick={handleGenerateGeneric}
                          disabled={saving}
                        >
                          {saving
                            ? t("rawFileSaving")
                            : t("rawFileTemplateGenerate")}
                        </Button>
                        <Button
                          size="sm"
                          variant="ghost"
                          onClick={closeTemplatePicker}
                        >
                          {t("rawFileCancel")}
                        </Button>
                      </div>
                    </div>
                  ) : (
                    <Button
                      size="sm"
                      variant="ghost"
                      onClick={closeTemplatePicker}
                    >
                      {t("rawFileCancel")}
                    </Button>
                  )}
                </>
              )}
            </div>
          );
        })()
      ) : (
        <div className="flex flex-wrap items-center gap-2">
          <Button size="sm" variant="outline" onClick={() => setAdding(true)}>
            <Plus className="h-3.5 w-3.5 mr-1" />
            {t("rawFileAdd")}
          </Button>
          <Button
            size="sm"
            variant="outline"
            onClick={() => setAddingEmptyShadow(true)}
          >
            <FileX2 className="h-3.5 w-3.5 mr-1" />
            {t("rawFileEmptyShadowAdd")}
          </Button>
          <Button
            size="sm"
            variant="outline"
            onClick={() => setAddingBinary(true)}
          >
            <Upload className="h-3.5 w-3.5 mr-1" />
            {t("rawFileBinaryAdd")}
          </Button>
          <Button size="sm" variant="outline" onClick={openTemplatePicker}>
            <Sparkles className="h-3.5 w-3.5 mr-1" />
            {t("rawFileTemplateAdd")}
          </Button>
        </div>
      )}
    </div>
  );
}

function RawFileCard({
  file,
  projectId,
  allowReplacePath,
  onSaved,
  onDelete,
}: {
  file: RawFileRow;
  projectId: number;
  allowReplacePath: boolean;
  onSaved: () => Promise<void> | void;
  onDelete: () => void;
}) {
  const t = useTranslations("export");
  const [destPath, setDestPath] = useState(file.dest_path);
  const [content, setContent] = useState(file.content);
  const [genReplacePath, setGenReplacePath] = useState(
    file.generate_replace_path,
  );
  const [targetLayer, setTargetLayer] = useState(file.target_layer ?? "");
  const [targetSha256, setTargetSha256] = useState(file.target_sha256 ?? "");
  const [note, setNote] = useState(file.note);
  const [saving, setSaving] = useState(false);
  const [emptyShadowError, setEmptyShadowError] =
    useState<EmptyShadowFailure | null>(null);

  const dirty =
    destPath !== file.dest_path ||
    content !== file.content ||
    genReplacePath !== file.generate_replace_path ||
    targetLayer !== (file.target_layer ?? "") ||
    targetSha256 !== (file.target_sha256 ?? "") ||
    note !== file.note;

  const handleSave = async () => {
    setSaving(true);
    setEmptyShadowError(null);
    try {
      if (file.raw_kind === "empty_shadow") {
        const normalisedSha = targetSha256.trim().toLowerCase();
        if (
          !destPath.trim() ||
          !targetLayer.trim() ||
          !SHA256_RE.test(normalisedSha)
        ) {
          setEmptyShadowError({
            message: t("rawFileEmptyShadowEvidenceRequired"),
          });
          return;
        }
        await rawFilesApi.update(projectId, file.id, {
          dest_path: destPath.trim(),
          content: "",
          generate_replace_path: false,
          raw_kind: "empty_shadow",
          target_layer: targetLayer.trim(),
          target_sha256: normalisedSha,
          note: note.trim(),
        });
      } else {
        await rawFilesApi.update(projectId, file.id, {
          dest_path: destPath.trim(),

          content,
          generate_replace_path: allowReplacePath && genReplacePath,
          note: note.trim(),
        });
      }
      toast.success(t("rawFileSaved"));
      await onSaved();
    } catch (err) {
      if (file.raw_kind === "empty_shadow") {
        setEmptyShadowError(emptyShadowFailure(err));
      } else {
        toast.error(t("rawFileSaveFailed", { err: String(err) }));
      }
    } finally {
      setSaving(false);
    }
  };

  if (file.raw_kind === "empty_shadow") {
    return (
      <div className="rounded-md border border-warning/40 bg-warning/5 p-3 space-y-2">
        <div className="flex items-center gap-2">
          <span className="rounded bg-warning/15 px-1.5 py-0.5 text-micro font-semibold uppercase tracking-wide text-warning">
            {t("rawFileEmptyShadowBadge")}
          </span>
          <span className="font-mono text-xs text-foreground break-all">
            {file.dest_path}
          </span>
        </div>
        <p className="text-xs text-muted-foreground">
          {t("rawFileEmptyShadowDesc")}
        </p>
        <div className="space-y-1">
          <label className="text-xs font-medium text-foreground">
            {t("rawFileDestLabel")}
          </label>
          <Input
            value={destPath}
            onChange={(e) => setDestPath(e.target.value)}
            className="font-mono text-xs"
          />
        </div>
        <div className="space-y-1">
          <label className="text-xs font-medium text-foreground">
            {t("rawFileEmptyShadowLayerLabel")}
          </label>
          <Input
            value={targetLayer}
            onChange={(e) => setTargetLayer(e.target.value)}
          />
        </div>
        <div className="space-y-1">
          <label className="text-xs font-medium text-foreground">
            {t("rawFileEmptyShadowHashLabel")}
          </label>
          <Input
            value={targetSha256}
            onChange={(e) => setTargetSha256(e.target.value)}
            className="font-mono text-xs"
            spellCheck={false}
          />
        </div>
        <div className="space-y-1">
          <label className="text-xs font-medium text-foreground">
            {t("rawFileNoteLabel")}
          </label>
          <Input value={note} onChange={(e) => setNote(e.target.value)} />
        </div>
        {emptyShadowError ? (
          <div
            role="alert"
            className="rounded-md border border-destructive/40 bg-destructive/10 p-2 text-xs text-destructive"
          >
            <p>
              {t("rawFileEmptyShadowRejected", {
                message: emptyShadowError.message,
              })}
            </p>
            {emptyShadowError.code ? (
              <p className="mt-1 font-mono">
                {t("rawFileEmptyShadowPolicyCode", {
                  code: emptyShadowError.code,
                })}
              </p>
            ) : null}
          </div>
        ) : null}
        <div className="flex items-center gap-2">
          <Button size="sm" onClick={handleSave} disabled={!dirty || saving}>
            {saving ? t("rawFileSaving") : t("rawFileSave")}
          </Button>
          <Button size="sm" variant="ghost" onClick={onDelete}>
            <Trash2 className="h-3.5 w-3.5 mr-1" />
            {t("rawFileDelete")}
          </Button>
        </div>
      </div>
    );
  }

  if (file.is_binary) {
    return (
      <div className="rounded-md border border-destructive/30 bg-destructive/5 p-3 space-y-2">
        <div className="flex items-center gap-2">
          <span className="rounded bg-destructive/15 px-1.5 py-0.5 text-micro font-semibold uppercase tracking-wide text-destructive">
            {t("rawFileBinaryBadge")}
          </span>
          <span className="font-mono text-xs text-foreground break-all">
            {file.dest_path}
          </span>
        </div>
        <div className="space-y-1">
          <label className="text-xs font-medium text-foreground">
            {t("rawFileDestLabel")}
          </label>
          <Input
            value={destPath}
            onChange={(e) => setDestPath(e.target.value)}
            className="font-mono text-xs"
          />
        </div>
        {allowReplacePath && (
          <label className="flex items-start gap-2 cursor-pointer">
            <input
              type="checkbox"
              checked={genReplacePath}
              onChange={(e) => setGenReplacePath(e.target.checked)}
              className="accent-destructive h-4 w-4 mt-0.5"
            />
            <span className="text-xs text-muted-foreground">
              {t("rawFileGenReplacePath")}
            </span>
          </label>
        )}
        <div className="space-y-1">
          <label className="text-xs font-medium text-foreground">
            {t("rawFileNoteLabel")}
          </label>
          <Input value={note} onChange={(e) => setNote(e.target.value)} />
        </div>
        <div className="flex items-center gap-2">
          <Button size="sm" onClick={handleSave} disabled={!dirty || saving}>
            {saving ? t("rawFileSaving") : t("rawFileSave")}
          </Button>
          <Button size="sm" variant="ghost" onClick={onDelete}>
            <Trash2 className="h-3.5 w-3.5 mr-1" />
            {t("rawFileDelete")}
          </Button>
        </div>
      </div>
    );
  }

  return (
    <div className="rounded-md border border-destructive/30 bg-destructive/5 p-3 space-y-2">
      <RawFileFields
        destPath={destPath}
        content={content}
        generateReplacePath={genReplacePath}
        allowReplacePath={allowReplacePath}
        note={note}
        onChange={(patch) => {
          if (patch.dest_path !== undefined) setDestPath(patch.dest_path);
          if (patch.content !== undefined) setContent(patch.content);
          if (patch.generate_replace_path !== undefined)
            setGenReplacePath(patch.generate_replace_path);
          if (patch.note !== undefined) setNote(patch.note);
        }}
      />
      <div className="flex items-center gap-2">
        <Button size="sm" onClick={handleSave} disabled={!dirty || saving}>
          {saving ? t("rawFileSaving") : t("rawFileSave")}
        </Button>
        <Button size="sm" variant="ghost" onClick={onDelete}>
          <Trash2 className="h-3.5 w-3.5 mr-1" />
          {t("rawFileDelete")}
        </Button>
      </div>
    </div>
  );
}

function RawFileFields({
  destPath,
  content,
  generateReplacePath,
  allowReplacePath,
  note,
  onChange,
}: {
  destPath: string;
  content: string;
  generateReplacePath: boolean;
  allowReplacePath: boolean;
  note: string;
  onChange: (
    patch: Partial<{
      dest_path: string;
      content: string;
      generate_replace_path: boolean;
      note: string;
    }>,
  ) => void;
}) {
  const t = useTranslations("export");
  return (
    <div className="space-y-2">
      <div className="space-y-1">
        <label className="text-xs font-medium text-foreground">
          {t("rawFileDestLabel")}
        </label>
        <Input
          value={destPath}
          onChange={(e) => onChange({ dest_path: e.target.value })}
          placeholder="common/on_actions/TNO_Reich_on_actions.txt"
          className="font-mono text-xs"
        />
      </div>
      <div className="space-y-1">
        <label className="text-xs font-medium text-foreground">
          {t("rawFileContentLabel")}
        </label>

        <Textarea
          value={content}
          onChange={(e) => onChange({ content: e.target.value })}
          placeholder={t("rawFileContentPlaceholder")}
          className="font-mono text-xs min-h-32 max-h-72"
          spellCheck={false}
        />
      </div>
      {allowReplacePath && (
        <label className="flex items-start gap-2 cursor-pointer">
          <input
            type="checkbox"
            checked={generateReplacePath}
            onChange={(e) =>
              onChange({ generate_replace_path: e.target.checked })
            }
            className="accent-destructive h-4 w-4 mt-0.5"
          />
          <span className="text-xs text-muted-foreground">
            {t("rawFileGenReplacePath")}
          </span>
        </label>
      )}
      <div className="space-y-1">
        <label className="text-xs font-medium text-foreground">
          {t("rawFileNoteLabel")}
        </label>
        <Input
          value={note}
          onChange={(e) => onChange({ note: e.target.value })}
          placeholder={t("rawFileNotePlaceholder")}
        />
      </div>
    </div>
  );
}
