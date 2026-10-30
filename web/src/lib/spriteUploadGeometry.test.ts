import { describe, expect, it } from "vitest";
import {
  canCropInBrowser,
  distortionOf,
  cropAspect,
  frameSources,
  pictureBytesProblem,
  pictureFileProblem,
  placeInFrame,
  spriteUploadFields,
} from "./spriteUploadGeometry";

describe("sprite upload geometry", () => {
  const strip = { width: 120, height: 30, frames: 4 };

  it("locks the crop to the shape that is filled, and frees it when nothing is cut", () => {
    expect(
      cropAspect({ width: 100, height: 50, frames: 1 }, "fill", "strip"),
    ).toBe(2);
    expect(cropAspect(strip, "fill", "repeat")).toBe(1);
    expect(cropAspect(strip, "fill", "strip")).toBe(4);
    expect(cropAspect(strip, "fit", "strip")).toBe(4);
    expect(
      cropAspect({ width: 100, height: 50, frames: 1 }, "fit", "strip"),
    ).toBeUndefined();
    expect(cropAspect(strip, "fill", "grid")).toBeUndefined();
  });

  it("reads a grid left to right, top to bottom", () => {
    const cells = frameSources(
      { x: 10, y: 0, width: 200, height: 100 },
      strip,
      "grid",
      2,
    );
    expect(cells.map((c) => [c.x, c.y, c.width, c.height])).toEqual([
      [10, 0, 100, 50],
      [110, 0, 100, 50],
      [10, 50, 100, 50],
      [110, 50, 100, 50],
    ]);
    expect(
      frameSources({ x: 0, y: 0, width: 8, height: 8 }, strip, "repeat"),
    ).toHaveLength(4);
    expect(
      frameSources({ x: 0, y: 0, width: 120, height: 30 }, strip, "strip")[2],
    ).toEqual({ x: 60, y: 0, width: 30, height: 30 });
  });

  it("covers a frame by cropping the source, or fits it whole with margins", () => {
    const wide = { x: 0, y: 0, width: 200, height: 100 };
    const square = { width: 50, height: 50 };
    expect(placeInFrame(wide, square, "fill")).toEqual({
      src: { x: 50, y: 0, width: 100, height: 100 },
      dst: { x: 0, y: 0, width: 50, height: 50 },
    });
    expect(placeInFrame(wide, square, "fit").dst).toEqual({
      x: 0,
      y: 12,
      width: 50,
      height: 25,
    });
  });

  it("sends nothing for the default and the crop in the picture's pixels", () => {
    expect(spriteUploadFields(undefined)).toEqual({});
    expect(
      spriteUploadFields({
        mode: "fit",
        layout: "grid",
        gridCols: 3,
        crop: { x: 4, y: 5, width: 60, height: 40 },
      }),
    ).toEqual({
      mode: "fit",
      layout: "grid",
      grid_cols: "3",
      crop_x: "4",
      crop_y: "5",
      crop_width: "60",
      crop_height: "40",
    });
  });

  it("only crops pictures the browser can show", () => {
    expect(
      canCropInBrowser(new File([""], "a.png", { type: "image/png" })),
    ).toBe(true);
    expect(canCropInBrowser(new File([""], "a.JPG"))).toBe(true);
    expect(canCropInBrowser(new File([""], "a.dds"))).toBe(false);
    expect(canCropInBrowser(new File([""], "a.tga"))).toBe(false);
  });
});

describe("distortionOf", () => {
  const frame = { width: 64, height: 32 };

  it("finds nothing in a part of the frame's shape with pixels to spare", () => {
    expect(distortionOf({ width: 640, height: 320 }, frame)).toEqual({
      stretched: false,
      upscaled: false,
    });
    expect(distortionOf({ width: 66, height: 32 }, frame)).toEqual({
      stretched: false,
      upscaled: false,
    });
  });

  it("finds a part stretched out of shape, and one blown up past its pixels", () => {
    expect(distortionOf({ width: 300, height: 300 }, frame)).toEqual({
      stretched: true,
      upscaled: false,
    });
    expect(distortionOf({ width: 32, height: 16 }, frame)).toEqual({
      stretched: false,
      upscaled: true,
    });
  });
});

describe("picture file checks (R4)", () => {
  it("knows an empty file, an SVG and a file that is no picture by its name and size", () => {
    expect(
      pictureFileProblem(new File([], "a.png", { type: "image/png" })),
    ).toBe("empty");
    expect(
      pictureFileProblem(
        new File(["<svg/>"], "a.svg", { type: "image/svg+xml" }),
      ),
    ).toBe("svg");
    expect(
      pictureFileProblem(new File(["x"], "a.txt", { type: "text/plain" })),
    ).toBe("notImage");
    expect(pictureFileProblem(new File(["x"], "a.dds"))).toBeNull();
    expect(pictureFileProblem(new File(["x"], "a.tga"))).toBeNull();
  });

  it("knows a renamed text file by its first bytes, and leaves .dds and .tga to the service", () => {
    const png = new Uint8Array([
      0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a,
    ]);
    const text = new TextEncoder().encode("hello, world....");
    const webp = new TextEncoder().encode(
      "RIFF\u0000\u0000\u0000\u0000WEBPVP8 ",
    );
    expect(pictureBytesProblem(new File([], "a.png"), png)).toBeNull();
    expect(pictureBytesProblem(new File([], "a.webp"), webp)).toBeNull();
    expect(pictureBytesProblem(new File([], "a.png"), text)).toBe("notImage");
    expect(pictureBytesProblem(new File([], "a.dds"), text)).toBeNull();
  });
});
