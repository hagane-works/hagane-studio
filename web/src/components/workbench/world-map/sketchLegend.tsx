"use client";

import { useTranslations } from "use-intl";
import useSWR from "swr";
import { withGameLanguage } from "@/lib/guiRenderApi";

export interface LegendEntry {
  key: string;
  name: string;
  color: [number, number, number];
}

export function useSketchLegend(projectId: number, enabled: boolean) {
  return useSWR(enabled ? ["world-map-legend", projectId] : null, async () => {
    const response = await fetch(
      withGameLanguage(
        `/api/go/projects/${projectId}/gui-render/world-map/legend`,
      ),
    );
    return ((await response.json()) as { legend: LegendEntry[] }).legend;
  });
}

export function downloadCanvas(
  sea: [number, number, number],
  width = 1408,
  height = 512,
  filename = "world-sketch.png",
) {
  const canvas = document.createElement("canvas");
  canvas.width = width;
  canvas.height = height;
  const ctx = canvas.getContext("2d");
  if (!ctx) return;
  ctx.fillStyle = `rgb(${sea[0]}, ${sea[1]}, ${sea[2]})`;
  ctx.fillRect(0, 0, canvas.width, canvas.height);
  canvas.toBlob((blob) => {
    if (!blob) return;
    const url = URL.createObjectURL(blob);
    const link = document.createElement("a");
    link.href = url;
    link.download = filename;
    link.click();
    URL.revokeObjectURL(url);
  }, "image/png");
}

export function LegendSwatches({ legend }: { legend: LegendEntry[] }) {
  const t = useTranslations("worldMap");
  return (
    <ul className="mt-2 grid grid-cols-2 gap-1.5">
      {legend.map((entry) => (
        <li key={entry.key} className="flex items-center gap-2 text-xs">
          <span
            className="h-4 w-4 shrink-0 rounded-sm border border-border"
            style={{
              backgroundColor: `rgb(${entry.color[0]}, ${entry.color[1]}, ${entry.color[2]})`,
            }}
            aria-hidden
          />
          <span>{t(`legend_${entry.key}`)}</span>
          <span className="ml-auto font-mono text-micro text-muted-foreground">
            {entry.color.map((v) => v.toString(16).padStart(2, "0")).join("")}
          </span>
        </li>
      ))}
    </ul>
  );
}

export function sampleWorld(legend: LegendEntry[]): Promise<File | null> {
  const colour = (key: string) => {
    const c = legend.find((entry) => entry.key === key)?.color;
    return c ? `rgb(${c[0]}, ${c[1]}, ${c[2]})` : null;
  };
  const sea = colour("ocean");
  const plains = colour("plains");
  const forest = colour("forest");
  const hills = colour("hills");
  const mountain = colour("mountain");
  const canvas = document.createElement("canvas");
  canvas.width = 352;
  canvas.height = 128;
  const ctx = canvas.getContext("2d");
  if (!ctx || !sea || !plains || !forest || !hills || !mountain)
    return Promise.resolve(null);
  ctx.imageSmoothingEnabled = false;
  ctx.fillStyle = sea;
  ctx.fillRect(0, 0, 352, 128);
  const continent = (
    cx: number,
    cy: number,
    rx: number,
    ry: number,
    paint: () => void,
  ) => {
    ctx.save();
    ctx.beginPath();
    ctx.ellipse(cx, cy, rx, ry, 0, 0, Math.PI * 2);
    ctx.clip();
    ctx.fillStyle = plains;
    ctx.fillRect(0, 0, 352, 128);
    paint();
    ctx.restore();
  };
  continent(90, 60, 70, 40, () => {
    ctx.fillStyle = forest;
    ctx.fillRect(20, 20, 60, 35);
    ctx.fillStyle = hills;
    ctx.fillRect(110, 50, 50, 50);
  });
  continent(260, 64, 72, 44, () => {
    ctx.fillStyle = mountain;
    ctx.fillRect(245, 20, 14, 90);
    ctx.fillStyle = forest;
    ctx.fillRect(270, 70, 60, 40);
  });
  continent(176, 100, 12, 9, () => undefined);
  return new Promise((resolve) => {
    canvas.toBlob(
      (blob) =>
        resolve(
          blob
            ? new File([blob], "sample-world.png", { type: "image/png" })
            : null,
        ),
      "image/png",
    );
  });
}

export function sampleIsland(legend: LegendEntry[]): Promise<File | null> {
  const colour = (key: string) => {
    const c = legend.find((entry) => entry.key === key)?.color;
    return c ? `rgb(${c[0]}, ${c[1]}, ${c[2]})` : null;
  };
  const sea = colour("ocean");
  const plains = colour("plains");
  const forest = colour("forest");
  const mountain = colour("mountain");
  const canvas = document.createElement("canvas");
  canvas.width = 120;
  canvas.height = 60;
  const ctx = canvas.getContext("2d");
  if (!ctx || !sea || !plains || !forest || !mountain)
    return Promise.resolve(null);
  ctx.imageSmoothingEnabled = false;
  ctx.fillStyle = sea;
  ctx.fillRect(0, 0, 120, 60);
  ctx.save();
  ctx.beginPath();
  ctx.ellipse(60, 30, 50, 24, 0, 0, Math.PI * 2);
  ctx.clip();
  ctx.fillStyle = plains;
  ctx.fillRect(0, 0, 120, 60);
  ctx.fillStyle = forest;
  ctx.fillRect(0, 0, 45, 32);
  ctx.fillStyle = mountain;
  ctx.fillRect(78, 0, 14, 60);
  ctx.restore();
  return new Promise((resolve) => {
    canvas.toBlob(
      (blob) =>
        resolve(
          blob
            ? new File([blob], "sample-island.png", { type: "image/png" })
            : null,
        ),
      "image/png",
    );
  });
}
