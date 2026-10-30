"use client";

import { useEffect, useRef, useState } from "react";
import { useTranslations } from "use-intl";
import { ImageIcon, Trash2, Upload } from "lucide-react";

import { Button } from "@/components/ui/button";

export interface ProjectCoverUploadResult {
  filename: "thumbnail.png";
  previewUrl?: string | null;
  size?: number;
}

export interface ProjectCoverUploadProps {
  projectId: number;
  previewUrl?: string | null;
  onUpload: (file: File) => Promise<ProjectCoverUploadResult>;
  onDelete: () => Promise<void>;
  onChanged?: (result: ProjectCoverUploadResult | null) => void;
  disabled?: boolean;
}

const ACCEPTED_IMAGE = /\.(png|jpe?g|webp|gif|dds|tga)$/i;
const ACCEPTED_MIME = new Set([
  "image/png",
  "image/jpeg",
  "image/webp",
  "image/gif",
  "image/vnd-ms.dds",
  "image/x-dds",
  "image/x-tga",
  "image/tga",
]);

export function ProjectCoverUpload({
  projectId,
  previewUrl,
  onUpload,
  onDelete,
  onChanged,
  disabled = false,
}: ProjectCoverUploadProps) {
  const t = useTranslations("projectSettingsDialog.projectCover");
  const inputRef = useRef<HTMLInputElement>(null);
  const mountedRef = useRef(false);
  const projectGenerationRef = useRef(0);
  const [currentUrl, setCurrentUrl] = useState<string | null>(
    previewUrl ?? null,
  );
  const [stagedFile, setStagedFile] = useState<File | null>(null);
  const [stagedUrl, setStagedUrl] = useState<string | null>(null);
  const [stagedPreviewFailed, setStagedPreviewFailed] = useState(false);
  const [previewFailed, setPreviewFailed] = useState(false);
  const [busy, setBusy] = useState<"upload" | "delete" | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    mountedRef.current = true;
    projectGenerationRef.current += 1;
    setCurrentUrl(previewUrl ?? null);
    setStagedFile(null);
    setStagedUrl(null);
    setStagedPreviewFailed(false);
    setPreviewFailed(false);
    setBusy(null);
    setError(null);
    if (inputRef.current) inputRef.current.value = "";
    return () => {
      mountedRef.current = false;
      projectGenerationRef.current += 1;
    };
  }, [previewUrl, projectId]);

  useEffect(() => {
    return () => {
      if (stagedUrl) URL.revokeObjectURL(stagedUrl);
    };
  }, [stagedUrl]);

  const clearStaged = () => {
    setStagedFile(null);
    setStagedUrl(null);
    setStagedPreviewFailed(false);
    if (inputRef.current) inputRef.current.value = "";
  };

  const handleFile = (event: React.ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0];
    if (!file) return;
    const accepted =
      ACCEPTED_MIME.has(file.type.toLowerCase()) ||
      ACCEPTED_IMAGE.test(file.name);
    if (!accepted) {
      clearStaged();
      setError(t("invalidType"));
      event.target.value = "";
      return;
    }
    setError(null);
    setStagedFile(file);
    setStagedPreviewFailed(false);
    setStagedUrl(URL.createObjectURL(file));
  };

  const uploadSelected = async () => {
    if (!stagedFile || busy) return;
    const operationGeneration = projectGenerationRef.current;
    const operationProjectId = projectId;
    const isCurrentOperation = () =>
      mountedRef.current &&
      projectGenerationRef.current === operationGeneration &&
      projectId === operationProjectId;
    setBusy("upload");
    setError(null);
    try {
      const result = await onUpload(stagedFile);
      if (!isCurrentOperation()) return;
      const nextUrl = result.previewUrl ?? previewUrl ?? currentUrl;
      setCurrentUrl(nextUrl ?? null);
      setPreviewFailed(false);
      clearStaged();
      onChanged?.(result);
    } catch (cause) {
      if (!isCurrentOperation()) return;
      const detail = cause instanceof Error ? cause.message : t("unknownError");
      setError(t("uploadFailed", { detail }));
    } finally {
      if (isCurrentOperation()) setBusy(null);
    }
  };

  const deleteCurrent = async () => {
    if (busy) return;
    const operationGeneration = projectGenerationRef.current;
    const operationProjectId = projectId;
    const isCurrentOperation = () =>
      mountedRef.current &&
      projectGenerationRef.current === operationGeneration &&
      projectId === operationProjectId;
    setBusy("delete");
    setError(null);
    try {
      await onDelete();
      if (!isCurrentOperation()) return;
      setCurrentUrl(null);
      setPreviewFailed(false);
      clearStaged();
      onChanged?.(null);
    } catch (cause) {
      if (!isCurrentOperation()) return;
      const detail = cause instanceof Error ? cause.message : t("unknownError");
      setError(t("deleteFailed", { detail }));
    } finally {
      if (isCurrentOperation()) setBusy(null);
    }
  };

  const hasPreview = Boolean(currentUrl) && !previewFailed;
  const locked = disabled || busy !== null;

  return (
    <section
      className="space-y-3"
      aria-labelledby={`project-cover-${projectId}`}
      aria-busy={busy !== null}
    >
      <div className="space-y-1">
        <h3
          id={`project-cover-${projectId}`}
          className="text-sm font-semibold tracking-tight"
        >
          {t("title")}
        </h3>
        <p className="text-xs text-muted-foreground">{t("hint")}</p>
      </div>

      <input
        ref={inputRef}
        type="file"
        accept="image/png,image/jpeg,image/webp,image/gif,.dds,.tga"
        className="hidden"
        onChange={handleFile}
        disabled={locked}
        aria-label={t("choose")}
      />

      {stagedUrl && stagedFile ? (
        <div className="space-y-3 rounded-md border border-border bg-muted/30 p-3">
          {stagedPreviewFailed ? (
            <div
              role="img"
              aria-label={t("selectedAlt")}
              className="flex aspect-square w-40 flex-col items-center justify-center gap-2 rounded-md border border-border bg-background p-3 text-center"
            >
              <ImageIcon
                className="size-5 text-muted-foreground"
                aria-hidden="true"
              />
              <span className="text-xs text-muted-foreground">
                {t("selectedPreviewUnavailable")}
              </span>
            </div>
          ) : (
            <img
              src={stagedUrl}
              alt={t("selectedAlt")}
              onError={() => setStagedPreviewFailed(true)}
              className="aspect-square w-40 rounded-md border border-border bg-background object-cover"
            />
          )}
          <p className="truncate text-xs text-muted-foreground">
            {stagedFile.name}
          </p>
          <div className="flex items-center gap-2">
            <Button
              type="button"
              size="sm"
              onClick={uploadSelected}
              disabled={locked}
            >
              <Upload data-icon="inline-start" />
              {busy === "upload" ? t("uploading") : t("confirm")}
            </Button>
            <Button
              type="button"
              size="sm"
              variant="ghost"
              onClick={clearStaged}
              disabled={locked}
            >
              {t("cancel")}
            </Button>
          </div>
        </div>
      ) : hasPreview ? (
        <div className="flex items-start gap-3 rounded-md border border-border bg-muted/30 p-3">
          <img
            src={currentUrl ?? undefined}
            alt={t("previewAlt")}
            onError={() => setPreviewFailed(true)}
            className="aspect-square w-32 shrink-0 rounded-md border border-border bg-background object-cover"
          />
          <div className="min-w-0 flex-1 space-y-2">
            <div>
              <p className="text-sm font-medium">thumbnail.png</p>
              <p className="text-xs text-muted-foreground">{t("ready")}</p>
            </div>
            <div className="flex flex-wrap items-center gap-2">
              <Button
                type="button"
                size="sm"
                variant="outline"
                onClick={() => inputRef.current?.click()}
                disabled={locked}
              >
                <Upload data-icon="inline-start" />
                {t("replace")}
              </Button>
              <Button
                type="button"
                size="sm"
                variant="destructive"
                onClick={deleteCurrent}
                disabled={locked}
              >
                <Trash2 data-icon="inline-start" />
                {busy === "delete" ? t("deleting") : t("delete")}
              </Button>
            </div>
          </div>
        </div>
      ) : (
        <div className="flex items-center gap-3 rounded-md border border-dashed border-border bg-muted/20 p-4">
          <ImageIcon
            className="size-5 shrink-0 text-muted-foreground"
            aria-hidden="true"
          />
          <div className="min-w-0 flex-1">
            <p className="text-sm font-medium">
              {previewFailed ? t("unavailable") : t("empty")}
            </p>
            <p className="text-xs text-muted-foreground">{t("emptyHint")}</p>
          </div>
          <Button
            type="button"
            size="sm"
            variant="outline"
            onClick={() => inputRef.current?.click()}
            disabled={locked}
          >
            <Upload data-icon="inline-start" />
            {t("choose")}
          </Button>
        </div>
      )}

      {error && (
        <p
          role="alert"
          className="rounded-md border border-destructive/40 bg-destructive/10 px-3 py-2 text-xs text-destructive"
        >
          {error}
        </p>
      )}
    </section>
  );
}
