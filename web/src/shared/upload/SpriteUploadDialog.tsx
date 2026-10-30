"use client";

import { useEffect, useRef, useState } from "react";
import ReactCrop, {
  centerCrop,
  makeAspectCrop,
  type PercentCrop,
} from "react-image-crop";
import "react-image-crop/dist/ReactCrop.css";
import { useTranslations } from "use-intl";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import {
  distortionOf,
  frameSize,
  pictureBytesProblem,
  pictureFileProblem,
  placeInFrame,
  type PictureFileProblem,
  type Rect,
  type SpriteShape,
  type SpriteUploadOptions,
} from "@/lib/spriteUploadGeometry";

const PREVIEW_BOX = 120;

function initialCrop(
  aspect: number,
  width: number,
  height: number,
): PercentCrop {
  const wide = width / height > aspect;
  return centerCrop(
    makeAspectCrop(
      wide ? { unit: "%", height: 100 } : { unit: "%", width: 100 },
      aspect,
      width,
      height,
    ),
    width,
    height,
  );
}

function naturalRect(
  crop: PercentCrop | undefined,
  width: number,
  height: number,
): Rect {
  if (!crop || crop.width <= 0 || crop.height <= 0)
    return { x: 0, y: 0, width, height };
  return {
    x: (crop.x / 100) * width,
    y: (crop.y / 100) * height,
    width: (crop.width / 100) * width,
    height: (crop.height / 100) * height,
  };
}

function canvasOf(
  width: number,
  height: number,
): { canvas: HTMLCanvasElement; ctx: CanvasRenderingContext2D } {
  const canvas = document.createElement("canvas");
  canvas.width = Math.max(1, Math.round(width));
  canvas.height = Math.max(1, Math.round(height));
  const ctx = canvas.getContext("2d");
  if (!ctx) throw new Error("no canvas");
  ctx.imageSmoothingEnabled = true;
  ctx.imageSmoothingQuality = "high";
  return { canvas, ctx };
}

function sharpen(
  ctx: CanvasRenderingContext2D,
  width: number,
  height: number,
  amount: number,
) {
  const image = ctx.getImageData(0, 0, width, height);
  const src = new Uint8ClampedArray(image.data);
  const out = image.data;
  for (let y = 1; y < height - 1; y++) {
    for (let x = 1; x < width - 1; x++) {
      const i = (y * width + x) * 4;
      if (src[i + 3] === 0) continue;
      for (let c = 0; c < 3; c++) {
        let blur = 0;
        for (let dy = -1; dy <= 1; dy++) {
          for (let dx = -1; dx <= 1; dx++)
            blur += src[((y + dy) * width + x + dx) * 4 + c];
        }
        out[i + c] = src[i + c] + amount * (src[i + c] - blur / 9);
      }
    }
  }
  ctx.putImageData(image, 0, 0);
}

function convertedPicture(
  image: HTMLImageElement,
  part: Rect,
  frame: { width: number; height: number },
  name: string,
): Promise<File> {
  const scale = Math.min(frame.width / part.width, frame.height / part.height);
  const width = Math.max(1, Math.round(part.width * scale));
  const height = Math.max(1, Math.round(part.height * scale));

  let step = canvasOf(part.width, part.height);
  step.ctx.drawImage(
    image,
    part.x,
    part.y,
    part.width,
    part.height,
    0,
    0,
    part.width,
    part.height,
  );
  let w = part.width;
  let h = part.height;
  while (w * 2 < width || w / 2 > width) {
    const nw = w < width ? w * 2 : w / 2;
    const nh = w < width ? h * 2 : h / 2;
    const next = canvasOf(nw, nh);
    next.ctx.drawImage(
      step.canvas,
      0,
      0,
      next.canvas.width,
      next.canvas.height,
    );
    step = next;
    w = nw;
    h = nh;
  }
  const out = canvasOf(frame.width, frame.height);
  out.ctx.drawImage(
    step.canvas,
    Math.floor((frame.width - width) / 2),
    Math.floor((frame.height - height) / 2),
    width,
    height,
  );
  if (scale > 1.2) sharpen(out.ctx, out.canvas.width, out.canvas.height, 0.6);
  return new Promise((resolve, reject) =>
    out.canvas.toBlob(
      (blob) =>
        blob
          ? resolve(new File([blob], `${name}.png`, { type: "image/png" }))
          : reject(new Error("no picture")),
      "image/png",
    ),
  );
}

function readHead(file: File): Promise<Uint8Array> {
  return new Promise((resolve, reject) => {
    const reader = new FileReader();
    reader.onload = () => resolve(new Uint8Array(reader.result as ArrayBuffer));
    reader.onerror = () => reject(reader.error);
    reader.readAsArrayBuffer(file.slice(0, 16));
  });
}

export interface StagedSpriteUpload {
  file: File;
  url: string;
}

export function stageSpriteUpload(file: File): StagedSpriteUpload {
  return { file, url: URL.createObjectURL(file) };
}

export function releaseSpriteUpload(staged: StagedSpriteUpload | null): void {
  if (staged) URL.revokeObjectURL(staged.url);
}

export interface SpriteUploadDialogProps {
  staged: StagedSpriteUpload | null;
  sprite: string;
  shape: SpriteShape;
  busy?: boolean;

  progress?: number | null;

  error?: string | null;

  errorDetail?: string | null;
  onCancel: () => void;
  onConfirm: (file: File, options: SpriteUploadOptions) => void | Promise<void>;
}

export function SpriteUploadDialog({
  staged,
  busy = false,
  onCancel,
  ...rest
}: SpriteUploadDialogProps) {
  return (
    <Dialog
      open={staged !== null}
      onOpenChange={(open) => !open && !busy && onCancel()}
      disablePointerDismissal
    >
      <DialogContent className="sm:max-w-3xl">
        {staged && (
          <SpriteUploadForm
            key={staged.url}
            staged={staged}
            busy={busy}
            onCancel={onCancel}
            {...rest}
          />
        )}
      </DialogContent>
    </Dialog>
  );
}

function SpriteUploadForm({
  staged,
  sprite,
  shape,
  busy,
  progress,
  error,
  errorDetail,
  onCancel,
  onConfirm,
}: Omit<SpriteUploadDialogProps, "staged" | "busy"> & {
  staged: StagedSpriteUpload;
  busy: boolean;
}) {
  const t = useTranslations("spriteUpload");
  const frames = Math.max(1, shape.frames);
  const frame = frameSize(shape);

  const aspect = frame.width / frame.height;
  const [crop, setCrop] = useState<PercentCrop>();

  const [problem, setProblem] = useState<PictureFileProblem | null>(() =>
    pictureFileProblem(staged.file),
  );

  const [image, setImage] = useState<HTMLImageElement | null>(null);

  const submitting = useRef(false);
  const [retouching, setRetouching] = useState(false);
  const [retouchFailed, setRetouchFailed] = useState(false);

  const [converted, setConverted] = useState<StagedSpriteUpload | null>(null);
  useEffect(() => () => releaseSpriteUpload(converted), [converted]);
  const picture = converted ?? staged;

  useEffect(() => {
    if (pictureFileProblem(staged.file)) return;
    let live = true;
    readHead(staged.file)
      .then((head) => {
        const found = pictureBytesProblem(staged.file, head);
        if (live && found) setProblem(found);
      })
      .catch(() => {});
    return () => {
      live = false;
    };
  }, [staged.file]);

  const natural = image
    ? { width: image.naturalWidth, height: image.naturalHeight }
    : null;
  const part = natural
    ? naturalRect(crop, natural.width, natural.height)
    : null;

  const options: SpriteUploadOptions = {
    mode: "stretch",
    layout: frames > 1 ? "repeat" : "strip",
    crop: part ?? undefined,
  };
  const ready = Boolean(natural && !problem && !retouching);

  const distortion = part && !problem ? distortionOf(part, frame) : null;
  const distorted = Boolean(
    distortion && (distortion.stretched || distortion.upscaled),
  );
  const retouch = async () => {
    if (!image || !part || retouching || busy) return;
    setRetouching(true);
    setRetouchFailed(false);
    try {
      const file = await convertedPicture(
        image,
        part,
        frame,
        sprite || "sprite",
      );
      setImage(null);
      setConverted(stageSpriteUpload(file));
    } catch {
      setRetouchFailed(true);
    } finally {
      setRetouching(false);
    }
  };

  const confirm = () => {
    if (!ready || busy || submitting.current) return;
    submitting.current = true;
    void Promise.resolve(onConfirm(picture.file, options)).finally(() => {
      submitting.current = false;
    });
  };

  const showPicture =
    problem !== "empty" && problem !== "svg" && problem !== "notImage";

  return (
    <>
      <DialogHeader>
        <DialogTitle>{t("title")}</DialogTitle>
        <DialogDescription>
          <span className="break-all font-mono">{sprite}</span>
          {" · "}
          {frames > 1
            ? t("shapeFrames", {
                width: shape.width,
                height: shape.height,
                frames,
                frameWidth: frame.width,
              })
            : t("shape", { width: shape.width, height: shape.height })}
        </DialogDescription>
      </DialogHeader>

      <div className="grid gap-4 md:grid-cols-[minmax(0,1fr)_14rem]">
        <div className="min-w-0 space-y-2">
          {showPicture && (
            <ReactCrop
              crop={crop}
              onChange={(_, percent) => setCrop(percent)}
              keepSelection
              disabled={busy || retouching}
            >
              <img
                key={picture.url}
                src={picture.url}
                alt={t("sourceAlt")}
                style={{ maxWidth: "100%", maxHeight: 360 }}
                onError={() => setProblem("broken")}
                onLoad={(e) => {
                  const img = e.currentTarget;
                  setImage(img);
                  setCrop(
                    initialCrop(aspect, img.naturalWidth, img.naturalHeight),
                  );
                }}
              />
            </ReactCrop>
          )}
          {problem ? (
            <p
              role="alert"
              className="rounded-md border border-destructive/40 bg-destructive/10 px-3 py-2 text-xs text-destructive"
            >
              {problem === "empty"
                ? t("problemEmpty")
                : problem === "svg"
                  ? t("problemSvg")
                  : problem === "broken"
                    ? t("problemBroken")
                    : t("problemNotImage")}
            </p>
          ) : (
            <p className="text-xs text-muted-foreground">
              {t("lockedHint", { width: frame.width, height: frame.height })}
              {frames > 1 && ` ${t("framesHint", { frames })}`}
            </p>
          )}
          {distorted && distortion && (
            <div
              role="status"
              className="space-y-2 rounded-md border border-warning/50 bg-warning/10 px-3 py-2 text-xs"
            >
              <p>
                {t("distorted", {
                  reason: [
                    distortion.stretched && t("distortStretched"),
                    distortion.upscaled && t("distortUpscaled"),
                  ]
                    .filter(Boolean)
                    .join(t("reasonGap")),
                })}
              </p>
              <div className="flex flex-wrap items-center gap-2">
                <button
                  type="button"
                  className="ui-interactive ui-focus-ring rounded-md border border-accent/70 bg-card px-2.5 py-1 hover:bg-accent/10 disabled:opacity-50"
                  disabled={busy || retouching}
                  onClick={() => void retouch()}
                >
                  {retouching ? t("retouching") : t("retouch")}
                </button>
                <span className="text-muted-foreground">
                  {t("retouchCost")}
                </span>
              </div>
              {retouchFailed && (
                <p className="text-destructive">{t("retouchFailed")}</p>
              )}
            </div>
          )}
        </div>

        <div className="space-y-4">
          <div className="space-y-2">
            <div className="text-xs font-medium">{t("preview")}</div>
            {image && part ? (
              <ResultPreview image={image} part={part} shape={shape} />
            ) : (
              <div className="h-16 rounded-md bg-muted/60" />
            )}
          </div>
        </div>
      </div>

      {error && (
        <div
          role="alert"
          className="space-y-1 rounded-md border border-destructive/40 bg-destructive/10 px-3 py-2 text-xs text-destructive"
        >
          <p>{error}</p>
          {errorDetail && (
            <details>
              <summary className="cursor-pointer">{t("errorDetails")}</summary>
              <code className="break-all">{errorDetail}</code>
            </details>
          )}
        </div>
      )}
      <DialogFooter>
        <Button variant="outline" onClick={onCancel} disabled={busy}>
          {t("cancel")}
        </Button>
        <Button onClick={confirm} disabled={busy || !ready}>
          {busy
            ? progress != null && progress < 100
              ? t("uploadingPercent", { percent: progress })
              : t("processing")
            : error
              ? t("retry")
              : t("confirm")}
        </Button>
      </DialogFooter>
    </>
  );
}

function ResultPreview({
  image,
  part,
  shape,
}: {
  image: HTMLImageElement;
  part: Rect;
  shape: SpriteShape;
}) {
  const frame = frameSize(shape);
  const scale = Math.min(
    PREVIEW_BOX / frame.width,
    PREVIEW_BOX / frame.height,
    4,
  );
  const w = Math.max(1, Math.round(frame.width * scale));
  const h = Math.max(1, Math.round(frame.height * scale));
  const canvas = useRef<HTMLCanvasElement | null>(null);
  useEffect(() => {
    const ctx = canvas.current?.getContext("2d");
    if (!canvas.current || !ctx) return;
    ctx.clearRect(0, 0, canvas.current.width, canvas.current.height);
    const placed = placeInFrame(part, frame, "stretch");
    ctx.imageSmoothingQuality = "high";
    ctx.drawImage(
      image,
      placed.src.x,
      placed.src.y,
      placed.src.width,
      placed.src.height,
      placed.dst.x * scale,
      placed.dst.y * scale,
      placed.dst.width * scale,
      placed.dst.height * scale,
    );
  });
  return (
    <canvas
      ref={canvas}
      width={w}
      height={h}
      className="rounded-sm border border-border"
      style={{
        width: w,
        height: h,
        backgroundImage:
          "repeating-conic-gradient(var(--muted) 0% 25%, var(--card) 0% 50%)",
        backgroundSize: "8px 8px",
      }}
    />
  );
}
