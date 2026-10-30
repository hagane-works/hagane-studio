"use client";

import { useState, type CSSProperties } from "react";
import { useTranslations } from "use-intl";
import { toast } from "sonner";
import { guiRenderApi } from "@/lib/guiRenderApi";
import {
  canCropInBrowser,
  pictureFileProblem,
  type SpriteUploadOptions,
} from "@/lib/spriteUploadGeometry";
import {
  SpriteUploadDialog,
  releaseSpriteUpload,
  stageSpriteUpload,
  type StagedSpriteUpload,
} from "@/shared/upload/SpriteUploadDialog";
import { FilePick } from "@/shared/upload/FilePick";
import {
  sourceChanged,
  spriteUploadFailure,
} from "@/shared/upload/spriteUploadFailure";
import type { OwnPicture, SpritePreview } from "./editModel";
import { restoreElementPicture, saveElementPicture } from "./elementPicture";
import { invalidateLayerSpriteMeta } from "./UiEditArt";

const BOX_W = 80;
const BOX_H = 64;

export function framePreviewStyle(preview: SpritePreview): CSSProperties {
  const scale = Math.min(
    BOX_W / preview.frameWidth,
    BOX_H / preview.frameHeight,
    4,
  );
  const w = Math.max(1, Math.round(preview.frameWidth * scale));
  const h = Math.max(1, Math.round(preview.frameHeight * scale));
  return {
    width: w,
    height: h,
    backgroundImage: `url("${preview.url}")`,
    backgroundRepeat: "no-repeat",
    backgroundSize: `${w * preview.frames}px ${h}px`,
    backgroundPosition: `${-(preview.frame - 1) * w}px 0`,
  };
}

export function SpriteReplace({
  sprite,
  size,
  version,
  preview,
  own,
  label,
  onChanged,
  onOwnPicture,
}: {
  sprite: string;

  size?: { width: number; height: number; frames?: number } | null;

  version?: number;

  preview?: SpritePreview | null;

  own?: OwnPicture | null;
  label?: string;

  onChanged: (
    sprite: string,
    width: number,
    height: number,
    action: "replace" | "restore",
  ) => void;

  onOwnPicture?: (
    gfx: string | null,
    shape: { width: number; height: number; frames: number },
  ) => void;
}) {
  const t = useTranslations("gameViewEdit.sprite");
  const tu = useTranslations("spriteUpload");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const [errorDetail, setErrorDetail] = useState<string | null>(null);

  const [progress, setProgress] = useState<number | null>(null);

  const [shipped, setShipped] = useState<ShippedNote | null>(null);

  const [staged, setStaged] = useState<StagedSpriteUpload | null>(null);

  const shownPreview = preview
    ? {
        ...preview,
        url:
          version !== undefined
            ? guiRenderApi.spriteUrl(sprite, version)
            : preview.url,
      }
    : null;
  const run = async (work: () => Promise<void>): Promise<boolean> => {
    setBusy(true);
    setError(null);
    setErrorDetail(null);
    try {
      await work();
      return true;
    } catch (err) {
      const failure = spriteUploadFailure(err);
      setError(tu(failure.key));
      setErrorDetail(failure.detail);
      return false;
    } finally {
      setBusy(false);
      setProgress(null);
    }
  };
  const replace = (file: File, options?: SpriteUploadOptions) =>
    run(async () => {
      if (own) return replaceOwn(own, file, options);
      const projectId = guiRenderApi.currentProjectId();
      setProgress(0);
      const send = () =>
        guiRenderApi.override(sprite, file, options, setProgress);

      const result = await send().catch((err: unknown) => {
        if (!sourceChanged(err)) throw err;
        invalidateLayerSpriteMeta(projectId, sprite);
        setProgress(0);
        return send();
      });
      invalidateLayerSpriteMeta(projectId, sprite);
      onChanged(sprite, result.width ?? 0, result.height ?? 0, "replace");
      setShipped(exportNote(result));
      toast.success(tu("replaced", { sprite }));
    });

  const replaceOwn = async (
    at: OwnPicture,
    file: File,
    options?: SpriteUploadOptions,
  ) => {
    const projectId = guiRenderApi.currentProjectId();
    const shape = { width: at.width, height: at.height, frames: at.frames };
    setProgress(0);
    const result = await guiRenderApi.override(
      at.gfx,
      file,
      options ?? {
        mode: "stretch",
        layout: at.frames > 1 ? "repeat" : "strip",
      },
      setProgress,
      shape,
    );
    await saveElementPicture(at, at.gfx);
    invalidateLayerSpriteMeta(projectId, at.gfx);
    onChanged(
      at.gfx,
      result.width ?? at.width,
      result.height ?? at.height,
      "replace",
    );
    onOwnPicture?.(at.gfx, shape);
    setShipped(
      result.export
        ? {
            key: "exportOwn",
            values: { texture: result.export.texture },
            ok: true,
          }
        : exportNote(result),
    );
    toast.success(t("ownDone"));
  };
  const closeDialog = () => {
    releaseSpriteUpload(staged);
    setStaged(null);
  };
  const frames = own
    ? own.frames
    : Math.max(1, size?.frames ?? preview?.frames ?? 1);

  const shownSize = own
    ? { width: Math.round(own.width / own.frames), height: own.height }
    : size;
  const shape = own
    ? { width: own.width, height: own.height, frames }
    : { width: (size?.width ?? 0) * frames, height: size?.height ?? 0, frames };

  return (
    <section className="mt-4">
      <div className="text-micro font-medium text-muted-foreground">
        {label ?? t("picture")}
      </div>
      <div className="mt-2 flex items-center gap-3">
        {shownPreview ? (
          <div className="flex h-16 w-20 shrink-0 items-center justify-center rounded border border-border bg-muted">
            <div style={framePreviewStyle(shownPreview)} />
          </div>
        ) : (
          <div
            className="h-16 w-20 shrink-0 rounded border border-border bg-muted bg-contain bg-center bg-no-repeat"
            style={{
              backgroundImage: `url("${guiRenderApi.spriteUrl(sprite, version)}")`,
            }}
          />
        )}
        <div className="min-w-0 break-all font-mono text-micro text-muted-foreground">
          {sprite}
          {shownSize && (
            <div className="mt-1">
              {shownSize.width}×{shownSize.height}
            </div>
          )}
          {!own && preview && preview.frames > 1 && (
            <div className="mt-1">
              {t("frames", { frames: preview.frames, frame: preview.frame })}
            </div>
          )}
        </div>
      </div>
      <p className="mt-2 text-micro text-muted-foreground">
        {t("hint")}
        {own && ` ${t("ownHint")}`}
      </p>
      <FilePick
        className="mt-2"
        accept="image/png,image/jpeg,image/webp,.dds,.tga"
        disabled={busy}
        onPick={(file) => {
          if (!file) return;
          setError(null);
          setErrorDetail(null);

          if (
            shownSize &&
            (canCropInBrowser(file) || pictureFileProblem(file))
          ) {
            setStaged(stageSpriteUpload(file));
            return;
          }

          const problem = pictureFileProblem(file);
          if (problem) {
            setError(
              tu(
                problem === "empty"
                  ? "problemEmpty"
                  : problem === "svg"
                    ? "problemSvg"
                    : "problemNotImage",
              ),
            );
            return;
          }
          void replace(file);
        }}
      />
      <SpriteUploadDialog
        staged={staged}
        sprite={sprite}
        shape={shape}
        busy={busy}
        progress={progress}
        error={error}
        errorDetail={errorDetail}
        onCancel={closeDialog}
        onConfirm={async (file, options) => {
          if (await replace(file, options)) closeDialog();
        }}
      />
      <button
        type="button"
        disabled={busy}
        className="mt-2 rounded border border-border px-2 py-1 text-micro hover:bg-muted disabled:cursor-not-allowed disabled:opacity-50"
        onClick={() =>
          void run(async () => {
            const projectId = guiRenderApi.currentProjectId();
            if (own?.replaced) {
              await restoreElementPicture(own);
              invalidateLayerSpriteMeta(projectId, own.gfx);
              onChanged(own.gfx, own.width, own.height, "restore");
              onOwnPicture?.(null, {
                width: own.width,
                height: own.height,
                frames: own.frames,
              });
              setShipped({ key: "restored", ok: true });
              return;
            }
            const restored = await guiRenderApi.revert(sprite);
            invalidateLayerSpriteMeta(projectId, sprite);
            onChanged(sprite, restored.width, restored.height, "restore");
            setShipped({ key: "restored", ok: true });
          })
        }
      >
        {t("restore")}
      </button>
      {shipped && (
        <p
          className={`mt-2 break-all text-micro ${shipped.ok ? "text-muted-foreground" : "text-warning"}`}
        >
          {t(shipped.key, shipped.values ?? {})}
        </p>
      )}
      {error && !staged && (
        <div className="mt-2 text-micro text-destructive">
          <p>{error}</p>
          {errorDetail && (
            <details>
              <summary className="cursor-pointer">{tu("errorDetails")}</summary>
              <code className="break-all">{errorDetail}</code>
            </details>
          )}
        </div>
      )}
    </section>
  );
}

export interface ShippedNote {
  key:
    | "restored"
    | "exportGfx"
    | "exportOwn"
    | "exportPath"
    | "exportNone"
    | "exportNoneWhy";
  values?: Record<string, string>;
  ok: boolean;
}

export function exportNote(result: {
  export?: { gfx?: string; texture: string };
  export_error?: string;
}): ShippedNote {
  if (result.export?.gfx) {
    return {
      key: "exportGfx",
      values: { texture: result.export.texture, gfx: result.export.gfx },
      ok: true,
    };
  }
  if (result.export) {
    return {
      key: "exportPath",
      values: { texture: result.export.texture },
      ok: true,
    };
  }
  return result.export_error
    ? {
        key: "exportNoneWhy",
        values: { error: result.export_error },
        ok: false,
      }
    : { key: "exportNone", ok: false };
}
