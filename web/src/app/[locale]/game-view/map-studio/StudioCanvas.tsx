"use client";

import { useCallback, useEffect, useRef, type MutableRefObject } from "react";
import { localPoint } from "@/components/workbench/world-map/WorldMapCanvas";
import {
  CELL,
  cloneSketch,
  fill,
  paintSketch,
  stamp,
  strokeTo,
  unionRect,
  type Brush,
  type Look,
  type Rect,
  type Sketch,
} from "./studioSketch";

interface View {
  scale: number;
  x: number;
  y: number;
}

const mapSize = (s: Sketch): [number, number] => [s.w * CELL, s.h * CELL];

export function StudioCanvas({
  sketch,
  brush,
  look,
  overlay,
  version,
  fitRequest,
  paintable,
  width,
  height,
  onStroke,
}: {
  sketch: MutableRefObject<Sketch>;
  brush: Brush;
  look: Look;

  overlay: HTMLImageElement | null;

  version: number;

  fitRequest: number;

  paintable: boolean;
  width: number;
  height: number;

  onStroke: (before: Sketch, changed: Rect | null) => void;
}) {
  const canvasRef = useRef<HTMLCanvasElement>(null);

  const sketchRef = useRef(sketch);
  const pictureRef = useRef<{
    canvas: HTMLCanvasElement;
    data: ImageData;
  } | null>(null);
  const viewRef = useRef<View | null>(null);
  const lookRef = useRef(look);
  const brushRef = useRef(brush);
  const overlayRef = useRef(overlay);
  const paintableRef = useRef(paintable);
  const hoverRef = useRef<[number, number] | null>(null);
  const pendingRef = useRef<Rect | null>(null);
  const frameRef = useRef(0);
  const spaceRef = useRef(false);
  const stroke = useRef<{
    before: Sketch;
    last: [number, number];
    changed: Rect | null;
  } | null>(null);
  const pan = useRef<{
    x: number;
    y: number;
    viewX: number;
    viewY: number;
  } | null>(null);

  const draw = useCallback(() => {
    const canvas = canvasRef.current;
    const picture = pictureRef.current;
    const view = viewRef.current;
    const ctx = canvas?.getContext("2d");
    if (!canvas || !picture || !view || !ctx) return;
    const dpr = window.devicePixelRatio || 1;
    ctx.setTransform(1, 0, 0, 1, 0, 0);
    ctx.fillStyle = "rgb(16, 22, 30)";
    ctx.fillRect(0, 0, canvas.width, canvas.height);
    ctx.setTransform(dpr, 0, 0, dpr, 0, 0);
    const [mapW, mapH] = mapSize(sketchRef.current.current);
    const w = mapW * view.scale;
    const h = mapH * view.scale;
    const cssW = canvas.width / dpr;

    for (let k = -1; k <= 1; k++) {
      const left = view.x + k * w;
      if (left > cssW || left + w < 0) continue;
      ctx.imageSmoothingEnabled = CELL * view.scale < 2;
      ctx.drawImage(picture.canvas, left, view.y, w, h);
      const shown = overlayRef.current;
      if (shown) {
        ctx.imageSmoothingEnabled = true;
        ctx.drawImage(shown, left, view.y, w, h);
      }
    }
    const hover = hoverRef.current;
    if (
      hover &&
      !pan.current &&
      !spaceRef.current &&
      !overlayRef.current &&
      paintableRef.current
    ) {
      const b = brushRef.current;
      const fills = b.tool === "fill" || b.tool === "countryFill";
      const r = fills ? 6 : Math.max(2, b.radius * CELL * view.scale);
      ctx.lineWidth = 1.5;
      ctx.strokeStyle = "rgba(16, 20, 26, 0.85)";
      ctx.beginPath();
      ctx.arc(hover[0], hover[1], r + 1, 0, Math.PI * 2);
      ctx.stroke();
      ctx.strokeStyle = "rgba(250, 246, 236, 0.95)";
      ctx.beginPath();
      ctx.arc(hover[0], hover[1], r, 0, Math.PI * 2);
      ctx.stroke();
      if (fills) {
        ctx.beginPath();
        ctx.moveTo(hover[0] - r - 4, hover[1]);
        ctx.lineTo(hover[0] + r + 4, hover[1]);
        ctx.moveTo(hover[0], hover[1] - r - 4);
        ctx.lineTo(hover[0], hover[1] + r + 4);
        ctx.stroke();
      }
    }
  }, []);

  const repaint = useCallback(
    (rect: Rect | null) => {
      if (rect)
        pendingRef.current = unionRect(pendingRef.current, {
          x0: rect.x0 - 1,
          y0: rect.y0 - 1,
          x1: rect.x1 + 1,
          y1: rect.y1 + 1,
        });
      if (frameRef.current) return;
      frameRef.current = requestAnimationFrame(() => {
        frameRef.current = 0;
        const picture = pictureRef.current;
        const pending = pendingRef.current;
        pendingRef.current = null;
        const s = sketchRef.current.current;
        if (picture && pending) {
          const r = {
            x0: Math.max(0, pending.x0),
            y0: Math.max(0, pending.y0),
            x1: Math.min(s.w, pending.x1),
            y1: Math.min(s.h, pending.y1),
          };
          if (r.x1 > r.x0 && r.y1 > r.y0) {
            paintSketch(s, picture.data.data, r, lookRef.current);
            picture.canvas
              .getContext("2d")
              ?.putImageData(
                picture.data,
                0,
                0,
                r.x0,
                r.y0,
                r.x1 - r.x0,
                r.y1 - r.y0,
              );
          }
        }
        draw();
      });
    },
    [draw],
  );

  const fit = useCallback(() => {
    const canvas = canvasRef.current;
    if (!canvas) return;
    const [mapW, mapH] = mapSize(sketchRef.current.current);
    const scale = Math.min(
      canvas.clientWidth / mapW,
      canvas.clientHeight / mapH,
    );
    viewRef.current = {
      scale,
      x: (canvas.clientWidth - mapW * scale) / 2,
      y: (canvas.clientHeight - mapH * scale) / 2,
    };
  }, []);

  useEffect(() => {
    sketchRef.current = sketch;
  }, [sketch]);

  useEffect(() => {
    brushRef.current = brush;
    draw();
  }, [brush, draw]);

  useEffect(() => {
    overlayRef.current = overlay;
    paintableRef.current = paintable;
    draw();
  }, [overlay, paintable, draw]);

  useEffect(() => {
    lookRef.current = look;
    const s = sketchRef.current.current;
    if (
      !pictureRef.current ||
      pictureRef.current.data.width !== s.w ||
      pictureRef.current.data.height !== s.h
    ) {
      const canvas = document.createElement("canvas");
      canvas.width = s.w;
      canvas.height = s.h;
      const ctx = canvas.getContext("2d");
      if (!ctx) return;
      pictureRef.current = { canvas, data: ctx.createImageData(s.w, s.h) };
    }
    repaint({ x0: 0, y0: 0, x1: s.w, y1: s.h });
  }, [look, version, repaint]);

  useEffect(() => {
    fit();
    draw();
  }, [fitRequest, fit, draw]);

  useEffect(() => {
    const canvas = canvasRef.current;
    if (!canvas) return;
    const resize = () => {
      const dpr = window.devicePixelRatio || 1;
      canvas.width = Math.max(1, Math.round(canvas.clientWidth * dpr));
      canvas.height = Math.max(1, Math.round(canvas.clientHeight * dpr));
      if (!viewRef.current) fit();
      draw();
    };
    resize();
    const observer = new ResizeObserver(resize);
    observer.observe(canvas);
    const onWheel = (e: WheelEvent) => {
      const view = viewRef.current;
      if (!view) return;
      e.preventDefault();
      const [mx, my] = localPoint(canvas, e.clientX, e.clientY);
      const [mapW, mapH] = mapSize(sketchRef.current.current);
      const least =
        Math.min(canvas.clientWidth / mapW, canvas.clientHeight / mapH) * 0.8;
      const scale = Math.min(
        4,
        Math.max(least, view.scale * (e.deltaY < 0 ? 1.2 : 1 / 1.2)),
      );
      viewRef.current = {
        scale,
        x: mx - ((mx - view.x) * scale) / view.scale,
        y: my - ((my - view.y) * scale) / view.scale,
      };
      draw();
    };
    canvas.addEventListener("wheel", onWheel, { passive: false });
    return () => {
      observer.disconnect();
      canvas.removeEventListener("wheel", onWheel);
    };
  }, [draw, fit]);

  useEffect(() => {
    const down = (e: KeyboardEvent) => {
      if (
        e.code === "Space" &&
        !(
          e.target instanceof HTMLInputElement ||
          e.target instanceof HTMLTextAreaElement
        )
      ) {
        spaceRef.current = true;
        draw();
      }
    };
    const up = (e: KeyboardEvent) => {
      if (e.code === "Space") {
        spaceRef.current = false;
        draw();
      }
    };
    window.addEventListener("keydown", down, true);
    window.addEventListener("keyup", up, true);
    return () => {
      window.removeEventListener("keydown", down, true);
      window.removeEventListener("keyup", up, true);
    };
  }, [draw]);

  useEffect(
    () => () => {
      cancelAnimationFrame(frameRef.current);
      frameRef.current = 0;
    },
    [],
  );

  const cellAt = (
    canvas: HTMLCanvasElement,
    clientX: number,
    clientY: number,
  ): [number, number] | null => {
    const view = viewRef.current;
    if (!view) return null;
    const [x, y] = localPoint(canvas, clientX, clientY);
    const s = sketchRef.current.current;
    const [mapW, mapH] = mapSize(s);
    const my = (y - view.y) / view.scale;
    if (my < 0 || my >= mapH) return null;
    const mx = ((((x - view.x) / view.scale) % mapW) + mapW) % mapW;
    return [mx / CELL, my / CELL];
  };

  return (
    <canvas
      ref={canvasRef}
      data-map-studio-canvas=""
      style={{
        position: "absolute",
        left: 0,
        top: 0,
        width,
        height,
        cursor: paintable && !overlay ? "crosshair" : "grab",
      }}
      className="touch-none"
      onContextMenu={(e) => e.preventDefault()}
      onPointerDown={(e) => {
        const canvas = e.currentTarget;
        canvas.setPointerCapture(e.pointerId);
        const view = viewRef.current;
        if (!view) return;
        if (
          e.button !== 0 ||
          spaceRef.current ||
          !paintableRef.current ||
          overlayRef.current
        ) {
          const [x, y] = localPoint(canvas, e.clientX, e.clientY);
          pan.current = { x, y, viewX: view.x, viewY: view.y };
          draw();
          return;
        }
        const at = cellAt(canvas, e.clientX, e.clientY);
        if (!at) return;
        const s = sketchRef.current.current;
        const b = brushRef.current;
        if (b.tool === "fill" || b.tool === "countryFill") {
          const before = cloneSketch(s);
          const changed = fill(s, Math.floor(at[0]), Math.floor(at[1]), b);
          repaint(changed);
          onStroke(before, changed);
          return;
        }
        const before = cloneSketch(s);
        const changed = stamp(s, at[0], at[1], b);
        stroke.current = { before, last: at, changed };
        repaint(changed);
      }}
      onPointerMove={(e) => {
        const canvas = e.currentTarget;
        const [x, y] = localPoint(canvas, e.clientX, e.clientY);
        hoverRef.current = [x, y];
        const moving = pan.current;
        const view = viewRef.current;
        if (moving && view) {
          viewRef.current = {
            ...view,
            x: moving.viewX + x - moving.x,
            y: moving.viewY + y - moving.y,
          };
          draw();
          return;
        }
        const current = stroke.current;
        if (current) {
          const at = cellAt(canvas, e.clientX, e.clientY);
          if (at) {
            const s = sketchRef.current.current;

            const changed =
              Math.abs(at[0] - current.last[0]) > s.w / 2
                ? stamp(s, at[0], at[1], brushRef.current)
                : strokeTo(s, current.last, at, brushRef.current);
            current.changed = unionRect(current.changed, changed);
            current.last = at;
            repaint(changed);
            return;
          }
        }
        draw();
      }}
      onPointerUp={() => {
        pan.current = null;
        const current = stroke.current;
        stroke.current = null;
        if (current) onStroke(current.before, current.changed);
        draw();
      }}
      onPointerCancel={() => {
        pan.current = null;
        const current = stroke.current;
        stroke.current = null;
        if (current) onStroke(current.before, current.changed);
      }}
      onPointerLeave={() => {
        hoverRef.current = null;
        draw();
      }}
    />
  );
}
