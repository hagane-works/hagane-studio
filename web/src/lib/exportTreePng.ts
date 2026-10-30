"use client";

import { toPng } from "html-to-image";
import {
  getNodesBounds,
  getViewportForBounds,
  type ReactFlowInstance,
} from "@xyflow/react";

const MIN_DIM = 800;

const MAX_DIM = 12000;
const MAX_AREA = 40_000_000;
const PADDING_PX = 120;
const PADDING_RATIO = 0.1;

export interface ExportGeometry {
  width: number;
  height: number;

  scale: number;

  pixelRatio: number;
}

export function computeExportGeometry(bounds: {
  width: number;
  height: number;
}): ExportGeometry {
  const naturalW = Math.max(bounds.width + PADDING_PX * 2, MIN_DIM);
  const naturalH = Math.max(bounds.height + PADDING_PX * 2, MIN_DIM);

  const scale = Math.min(
    1,
    MAX_DIM / naturalW,
    MAX_DIM / naturalH,
    Math.sqrt(MAX_AREA / (naturalW * naturalH)),
  );

  const width = Math.max(1, Math.floor(naturalW * scale));
  const height = Math.max(1, Math.floor(naturalH * scale));

  const pixelRatio = width * height * 4 <= MAX_AREA ? 2 : 1;

  return { width, height, scale, pixelRatio };
}

export interface ExportTreePngOptions {
  filenameBase?: string;

  backgroundColor?: string;
}

export async function exportTreePng(
  rfInstance: ReactFlowInstance,
  opts: ExportTreePngOptions = {},
): Promise<void> {
  const nodes = rfInstance.getNodes();
  if (nodes.length === 0) {
    throw new Error("Empty tree — nothing to export.");
  }

  const viewportEl = document.querySelector<HTMLElement>(
    ".react-flow__viewport",
  );
  if (!viewportEl) {
    throw new Error("React Flow viewport element not found in DOM.");
  }

  const bounds = getNodesBounds(nodes);
  const geom = computeExportGeometry(bounds);
  const targetWidth = geom.width;
  const targetHeight = geom.height;

  const viewport = getViewportForBounds(
    bounds,
    targetWidth,
    targetHeight,
    geom.scale,
    2,
    PADDING_RATIO,
  );

  const bg =
    opts.backgroundColor ||
    getComputedStyle(document.body).backgroundColor ||
    "#0a0a0a";

  const dataUrl = await toPng(viewportEl, {
    backgroundColor: bg,
    width: targetWidth,
    height: targetHeight,
    style: {
      width: `${targetWidth}px`,
      height: `${targetHeight}px`,
      transform: `translate(${viewport.x}px, ${viewport.y}px) scale(${viewport.zoom})`,
    },
    cacheBust: true,
    pixelRatio: geom.pixelRatio,
    filter: (node) => {
      if (!(node instanceof HTMLElement)) return true;
      const cls = node.classList;
      if (cls.contains("react-flow__minimap")) return false;
      if (cls.contains("react-flow__controls")) return false;
      if (cls.contains("react-flow__panel")) return false;
      if (cls.contains("react-flow__attribution")) return false;
      return true;
    },
  });

  triggerDownload(dataUrl, buildFilename(opts.filenameBase));
}

function sanitizeBase(base: string): string {
  const trimmed = base.trim().replace(/[^A-Za-z0-9._-]+/g, "_");
  return trimmed || "focus_tree";
}

function buildFilename(base?: string): string {
  const d = new Date();
  const yyyy = d.getFullYear();
  const mm = String(d.getMonth() + 1).padStart(2, "0");
  const dd = String(d.getDate()).padStart(2, "0");
  const stem = sanitizeBase(base ?? "focus_tree");
  return `${stem}_${yyyy}-${mm}-${dd}.png`;
}

function triggerDownload(dataUrl: string, filename: string): void {
  const a = document.createElement("a");
  a.href = dataUrl;
  a.download = filename;
  document.body.appendChild(a);
  a.click();
  document.body.removeChild(a);
}
