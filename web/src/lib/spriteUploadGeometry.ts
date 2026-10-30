export type SpriteUploadMode = "fill" | "fit" | "stretch";
export type SpriteUploadLayout = "strip" | "repeat" | "grid";

export interface SpriteUploadOptions {
  crop?: { x: number; y: number; width: number; height: number };

  mode: SpriteUploadMode;

  layout: SpriteUploadLayout;
  gridCols?: number;
}

export interface SpriteShape {
  width: number;
  height: number;
  frames: number;
}

export interface Rect {
  x: number;
  y: number;
  width: number;
  height: number;
}

export function frameSize(shape: SpriteShape): {
  width: number;
  height: number;
} {
  const frames = Math.max(1, shape.frames);
  return { width: Math.floor(shape.width / frames), height: shape.height };
}

export function gridRows(frames: number, cols: number): number {
  const c = Math.max(1, Math.min(cols, frames));
  return Math.ceil(frames / c);
}

export function cropAspect(
  shape: SpriteShape,
  mode: SpriteUploadMode,
  layout: SpriteUploadLayout,
): number | undefined {
  const frames = Math.max(1, shape.frames);
  if (frames > 1 && layout === "strip") return shape.width / shape.height;
  if (mode === "fit" || (frames > 1 && layout === "grid")) return undefined;
  const frame = frameSize(shape);
  return frame.width / frame.height;
}

export function frameSources(
  part: Rect,
  shape: SpriteShape,
  layout: SpriteUploadLayout,
  gridCols = 1,
): Rect[] {
  const frames = Math.max(1, shape.frames);
  if (frames === 1 || layout === "repeat")
    return Array.from({ length: frames }, () => part);
  if (layout === "grid") {
    const cols = Math.max(1, Math.min(gridCols, frames));
    const rows = gridRows(frames, cols);
    const cw = part.width / cols;
    const ch = part.height / rows;
    return Array.from({ length: frames }, (_, f) => {
      const x0 = Math.round((f % cols) * cw);
      const y0 = Math.round(Math.floor(f / cols) * ch);
      const x1 = Math.round(((f % cols) + 1) * cw);
      const y1 = Math.round((Math.floor(f / cols) + 1) * ch);
      return {
        x: part.x + x0,
        y: part.y + y0,
        width: x1 - x0,
        height: y1 - y0,
      };
    });
  }
  const w = part.width / frames;
  return Array.from({ length: frames }, (_, f) => ({
    x: part.x + f * w,
    y: part.y,
    width: w,
    height: part.height,
  }));
}

export function placeInFrame(
  src: Rect,
  frame: { width: number; height: number },
  mode: SpriteUploadMode,
): { src: Rect; dst: Rect } {
  if (mode === "stretch")
    return {
      src,
      dst: { x: 0, y: 0, width: frame.width, height: frame.height },
    };
  if (mode === "fit") {
    const scale = Math.min(frame.width / src.width, frame.height / src.height);
    const w = Math.max(1, Math.round(src.width * scale));
    const h = Math.max(1, Math.round(src.height * scale));
    return {
      src,
      dst: {
        x: Math.floor((frame.width - w) / 2),
        y: Math.floor((frame.height - h) / 2),
        width: w,
        height: h,
      },
    };
  }
  const dst = { x: 0, y: 0, width: frame.width, height: frame.height };
  if (src.width * frame.height > src.height * frame.width) {
    const w = Math.max(1, (src.height * frame.width) / frame.height);
    return {
      src: {
        x: src.x + (src.width - w) / 2,
        y: src.y,
        width: w,
        height: src.height,
      },
      dst,
    };
  }
  const h = Math.max(1, (src.width * frame.height) / frame.width);
  return {
    src: {
      x: src.x,
      y: src.y + (src.height - h) / 2,
      width: src.width,
      height: h,
    },
    dst,
  };
}

export function spriteUploadFields(
  options: SpriteUploadOptions | undefined,
): Record<string, string> {
  if (!options) return {};
  const fields: Record<string, string> = {
    mode: options.mode,
    layout: options.layout,
  };
  if (options.layout === "grid")
    fields.grid_cols = String(Math.max(1, Math.round(options.gridCols ?? 1)));
  if (options.crop && options.crop.width > 0 && options.crop.height > 0) {
    fields.crop_x = String(Math.max(0, options.crop.x));
    fields.crop_y = String(Math.max(0, options.crop.y));
    fields.crop_width = String(options.crop.width);
    fields.crop_height = String(options.crop.height);
  }
  return fields;
}

export function canCropInBrowser(file: File): boolean {
  return (
    /^image\/(png|jpe?g|webp|gif)$/i.test(file.type) ||
    /\.(png|jpe?g|webp|gif)$/i.test(file.name)
  );
}

export const STRETCH_TOLERANCE = 0.1;
export const UPSCALE_LIMIT = 1.5;

export function distortionOf(
  part: { width: number; height: number },
  frame: { width: number; height: number },
): { stretched: boolean; upscaled: boolean } {
  if (
    part.width <= 0 ||
    part.height <= 0 ||
    frame.width <= 0 ||
    frame.height <= 0
  )
    return { stretched: false, upscaled: false };
  const shape = part.width / part.height / (frame.width / frame.height);
  return {
    stretched: Math.abs(Math.log(shape)) > Math.log(1 + STRETCH_TOLERANCE),
    upscaled:
      Math.max(frame.width / part.width, frame.height / part.height) >
      UPSCALE_LIMIT,
  };
}

export type PictureFileProblem = "empty" | "svg" | "notImage" | "broken";

export function pictureFileProblem(file: File): PictureFileProblem | null {
  if (file.size === 0) return "empty";
  if (/svg/i.test(file.type) || /\.svgz?$/i.test(file.name)) return "svg";
  const known =
    /^image\/(png|jpe?g|webp|gif|vnd-ms\.dds|x-dds|x-tga|x-targa|tga)$/i.test(
      file.type,
    ) || /\.(png|jpe?g|webp|gif|dds|tga)$/i.test(file.name);
  return known ? null : "notImage";
}

export function pictureBytesProblem(
  file: File,
  head: Uint8Array,
): PictureFileProblem | null {
  if (/\.(dds|tga)$/i.test(file.name)) return null;
  const starts = (...bytes: number[]) => bytes.every((b, i) => head[i] === b);
  const ascii = (at: number, text: string) =>
    [...text].every((c, i) => head[at + i] === c.charCodeAt(0));
  const image =
    starts(0x89, 0x50, 0x4e, 0x47) ||
    starts(0xff, 0xd8, 0xff) ||
    ascii(0, "GIF8") ||
    (ascii(0, "RIFF") && ascii(8, "WEBP"));
  return image ? null : "notImage";
}
