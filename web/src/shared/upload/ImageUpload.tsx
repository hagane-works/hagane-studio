"use client";

import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import ReactCrop, {
  centerCrop,
  makeAspectCrop,
  type Crop,
  type PixelCrop,
} from "react-image-crop";
import "react-image-crop/dist/ReactCrop.css";
import { useTranslations } from "use-intl";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { bustAssetPreview } from "@/lib/store/assetPreviewVersionStore";

export interface ImageUploadResult {
  filename: string;
  picture?: string;
  preview_url?: string;

  warnings?: string[];
  [key: string]: unknown;
}

export interface ImageUploadProps {
  label: string;

  hideLabel?: boolean;

  targetSize: { width: number; height: number };

  previewUrl: string | null;

  onPrepareUpload?: () => Promise<void>;
  onUpload: (
    file: File,
    crop?: { x: number; y: number; width: number; height: number },
  ) => Promise<ImageUploadResult>;

  onUploaded?: (result: ImageUploadResult) => void;

  hint?: string;

  disabled?: boolean;

  gfxName?: string;

  fallbackUrl?: string | null;

  baseModFallbackUrls?: string[];

  previewVersion?: number;
}

export function ImageUpload({
  label,
  hideLabel = false,
  targetSize,
  previewUrl,
  onPrepareUpload,
  onUpload,
  onUploaded,
  hint,
  disabled = false,
  gfxName,
  fallbackUrl,
  baseModFallbackUrls,
  previewVersion,
}: ImageUploadProps) {
  const preparationIntent = useRef(0);
  useEffect(
    () => () => {
      preparationIntent.current += 1;
    },
    [],
  );
  const fileInputRef = useRef<HTMLInputElement>(null);
  const imgRef = useRef<HTMLImageElement>(null);
  const [pendingFile, setPendingFile] = useState<File | null>(null);
  const [pendingPreview, setPendingPreview] = useState<string | null>(null);
  const [crop, setCrop] = useState<Crop>();
  const [completedCrop, setCompletedCrop] = useState<PixelCrop>();
  const [uploading, setUploading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [cacheBust, setCacheBust] = useState(0);

  const [fallbackStep, setFallbackStep] = useState(0);

  const baseModKey = (baseModFallbackUrls ?? []).join("|");
  const fallbackChain = useMemo(() => {
    const chain = [...(baseModFallbackUrls ?? [])];
    if (fallbackUrl) chain.push(fallbackUrl);
    return chain.filter(Boolean) as string[];
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [baseModKey, fallbackUrl]);

  const [previewBroken, setPreviewBroken] = useState(false);

  useEffect(() => {
    setFallbackStep(0);
    setPreviewBroken(false);
  }, [previewUrl, fallbackUrl, baseModKey, cacheBust, previewVersion]);

  const [reviewOpen, setReviewOpen] = useState(false);
  const t = useTranslations("imageUpload");

  const [warnings, setWarnings] = useState<string[]>([]);

  const aspect = targetSize.width / targetSize.height;

  const handleFileSelect = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;
    if (!(await stageFile(file))) e.target.value = "";
  };

  const prepareFile = async (
    file: File,
    accept: (file: File) => void,
  ): Promise<boolean> => {
    const intent = ++preparationIntent.current;

    const okType =
      /^image\//i.test(file.type) ||
      /\.(png|jpe?g|dds|tga|webp)$/i.test(file.name);
    if (!okType) {
      setError(t("unsupportedFormat"));
      return false;
    }

    setError(null);
    try {
      if (onPrepareUpload) await onPrepareUpload();
    } catch (cause) {
      if (intent === preparationIntent.current)
        setError(cause instanceof Error ? cause.message : String(cause));
      return false;
    }
    if (intent !== preparationIntent.current) return false;
    setWarnings([]);
    accept(file);
    return true;
  };

  const stageFile = (file: File): Promise<boolean> =>
    prepareFile(file, (accepted) => {
      if (pendingPreview) URL.revokeObjectURL(pendingPreview);
      setPendingFile(accepted);
      setPendingPreview(URL.createObjectURL(accepted));
      setCrop(undefined);
      setCompletedCrop(undefined);
    });

  const handleImageLoad = useCallback(
    (e: React.SyntheticEvent<HTMLImageElement>) => {
      const { width, height } = e.currentTarget;

      const initial = centerCrop(
        makeAspectCrop({ unit: "%", width: 90 }, aspect, width, height),
        width,
        height,
      );
      setCrop(initial);

      setCompletedCrop({
        unit: "px",
        x: (initial.x / 100) * width,
        y: (initial.y / 100) * height,
        width: (initial.width / 100) * width,
        height: (initial.height / 100) * height,
      });
    },
    [aspect],
  );

  const clearStaging = () => {
    if (pendingPreview) URL.revokeObjectURL(pendingPreview);
    setPendingFile(null);
    setPendingPreview(null);
    setCrop(undefined);
    setCompletedCrop(undefined);
    if (fileInputRef.current) fileInputRef.current.value = "";
  };

  const commitUpload = async (
    file: File,
    crop_payload?: { x: number; y: number; width: number; height: number },
  ) => {
    setUploading(true);
    setError(null);
    try {
      const result = await onUpload(file, crop_payload);
      setCacheBust(Date.now());

      const serverWarnings = Array.isArray(result.warnings)
        ? result.warnings
        : [];
      setWarnings(serverWarnings);
      onUploaded?.(result);

      bustAssetPreview(previewUrl);

      clearStaging();
    } finally {
      setUploading(false);
    }
  };

  const upload = async (
    file: File,
    crop_payload?: { x: number; y: number; width: number; height: number },
  ) => {
    try {
      await commitUpload(file, crop_payload);
    } catch (err) {
      setError((err as Error).message || t("uploadFailed"));
    }
  };

  const submitUpload = async () => {
    if (!pendingFile) return;

    let crop_payload:
      { x: number; y: number; width: number; height: number } | undefined;
    const img = imgRef.current;
    if (completedCrop && img && img.naturalWidth > 0) {
      const sx = img.naturalWidth / img.width;
      const sy = img.naturalHeight / img.height;
      crop_payload = {
        x: completedCrop.x * sx,
        y: completedCrop.y * sy,
        width: completedCrop.width * sx,
        height: completedCrop.height * sy,
      };
    }
    await upload(pendingFile, crop_payload);
  };

  const cancelStaging = () => {
    clearStaging();
    setError(null);
  };

  const finalPreview = previewUrl
    ? `${previewUrl}${previewUrl.includes("?") ? "&" : "?"}t=${cacheBust}${previewVersion ? `&v=${previewVersion}` : ""}`
    : null;

  const currentPreviewSrc =
    fallbackStep === 0
      ? finalPreview
      : (fallbackChain[fallbackStep - 1] ?? finalPreview);

  return (
    <div className="space-y-3">
      {!hideLabel && (
        <h3 className="text-sm font-semibold text-muted-foreground uppercase tracking-wide">
          {label}
        </h3>
      )}

      <div>
        <input
          ref={fileInputRef}
          type="file"
          accept="image/*,.dds,.tga"
          onChange={handleFileSelect}
          disabled={disabled}
          className="hidden"
        />

        <Button
          variant="outline"
          className="w-full border-input bg-card px-3 text-sm hover:bg-muted/40 dark:border-input dark:bg-card dark:hover:bg-muted/40"
          onClick={() => fileInputRef.current?.click()}
          disabled={uploading || disabled}
        >
          {pendingFile
            ? t("selectedFile", { filename: pendingFile.name })
            : finalPreview
              ? t("replaceImage")
              : t("uploadImage")}
        </Button>
        <p className="border-l-2 border-border/60 pl-2 mt-1.5 text-xs leading-snug text-muted-foreground">
          {hint ?? t("defaultHint", targetSize)}
        </p>
      </div>

      {error && (
        <div className="text-xs text-destructive p-2 bg-destructive/10 rounded-md border border-destructive/30">
          {error}
        </div>
      )}

      {warnings.length > 0 && (
        <div className="text-xs text-warning p-2 bg-warning/10 rounded-md border border-warning/30 space-y-1">
          <div className="font-semibold uppercase tracking-wide text-micro text-warning">
            {t("advisoryHeading", { count: warnings.length })}
          </div>
          {warnings.map((w, i) => (
            <p key={i} className="leading-snug">
              {w}
            </p>
          ))}
        </div>
      )}

      {pendingPreview && (
        <div className="space-y-2 p-2 bg-muted/40 rounded-md border border-border">
          <div className="text-xs text-muted-foreground">
            {t("cropInstruction", targetSize)}
          </div>
          <ReactCrop
            crop={crop}
            onChange={(c) => setCrop(c)}
            onComplete={(c) => setCompletedCrop(c)}
            aspect={aspect}
            keepSelection
          >
            <img
              ref={imgRef}
              src={pendingPreview}
              alt={t("cropSourceAlt")}
              onLoad={handleImageLoad}
              style={{ maxWidth: "100%", maxHeight: 360 }}
            />
          </ReactCrop>
          <div className="flex gap-2">
            <Button
              size="sm"
              onClick={submitUpload}
              disabled={uploading || disabled}
              className="flex-1"
            >
              {uploading ? t("uploading") : t("uploadWithCrop")}
            </Button>
            <Button
              size="sm"
              variant="ghost"
              onClick={cancelStaging}
              disabled={uploading || disabled}
            >
              {t("cancel")}
            </Button>
          </div>
        </div>
      )}

      {!pendingPreview && finalPreview && (
        <div className="space-y-2">
          <div className="text-xs text-muted-foreground">
            {t("currentUpload")}
          </div>
          <div className="flex items-start gap-3 p-2 bg-muted/30 rounded-md border border-border">
            {previewBroken ? (
              <div
                className="flex items-center justify-center border border-dashed border-border rounded-md bg-background shrink-0 text-muted-foreground text-micro text-center px-1"
                style={{
                  width: Math.min(targetSize.width, 156),
                  height: Math.min(targetSize.height, 156),
                }}
              >
                {t("referencedSpriteNoPreview")}
              </div>
            ) : (
              <img
                key={currentPreviewSrc ?? "empty"}
                src={currentPreviewSrc ?? undefined}
                alt={label}
                onError={() => {
                  if (fallbackStep < fallbackChain.length) {
                    setFallbackStep((s) => s + 1);
                  } else {
                    setPreviewBroken(true);
                  }
                }}
                onClick={() => setReviewOpen(true)}
                width={Math.min(targetSize.width, 156)}
                height={Math.min(targetSize.height, 156)}
                className="border border-border rounded-md bg-background shrink-0 cursor-zoom-in hover:opacity-90 transition-opacity"
                style={{ objectFit: "contain" }}
                title={t("clickToEnlarge")}
              />
            )}

            <div className="min-w-0 flex-1">
              <div className="text-xs text-muted-foreground">
                {targetSize.width} × {targetSize.height}
              </div>
              {gfxName && (
                <div className="text-xs font-mono text-muted-foreground mt-1 truncate">
                  {gfxName}
                </div>
              )}
            </div>
          </div>
          <Dialog open={reviewOpen} onOpenChange={setReviewOpen}>
            <DialogContent className="sm:max-w-3xl">
              <DialogHeader>
                <DialogTitle>{label}</DialogTitle>
                <DialogDescription className="font-mono text-xs">
                  {targetSize.width} × {targetSize.height}
                  {gfxName ? ` · ${gfxName}` : ""}
                </DialogDescription>
              </DialogHeader>
              <div className="flex items-center justify-center">
                <img
                  src={currentPreviewSrc ?? undefined}
                  alt={label}
                  style={{
                    maxWidth: "100%",
                    maxHeight: "70vh",
                    objectFit: "contain",
                  }}
                  className="rounded-md border border-border bg-background"
                />
              </div>
            </DialogContent>
          </Dialog>
        </div>
      )}

      {!pendingPreview && !finalPreview && (
        <div className="text-center text-muted-foreground text-sm py-6 border border-dashed border-input bg-card rounded-lg">
          <p className="mb-1">{t("emptyTitle")}</p>
          <p className="text-xs">{t("emptyHelp", targetSize)}</p>
        </div>
      )}
    </div>
  );
}
