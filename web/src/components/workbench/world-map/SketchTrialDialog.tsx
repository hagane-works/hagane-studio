"use client";

import { useState } from "react";
import { useTranslations } from "use-intl";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { SegmentedControl } from "@/components/ui/SegmentedControl";
import { FilePick } from "@/shared/upload/FilePick";
import {
  downloadCanvas,
  LegendSwatches,
  useSketchLegend,
} from "./sketchLegend";

interface Finding {
  code: string;
  severity: "error" | "warning";
  message: string;
  count: number;
}

interface Preview {
  width: number;
  height: number;
  seconds: number;
  provinces: { land: number; sea: number; lake: number };
  findings: Finding[];
  previews: { provinces: string; terrain: string; heightmap: string };
}

const SIZES = {
  quick: [1408, 512],
  preview: [2816, 1024],
} as const;

export function SketchTrialDialog({
  projectId,
  open,
  onOpenChange,
}: {
  projectId: number;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const t = useTranslations("worldMap");
  const legend = useSketchLegend(projectId, open);
  const [file, setFile] = useState<File | null>(null);
  const [size, setSize] = useState<keyof typeof SIZES>("quick");
  const [seed, setSeed] = useState("1");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [result, setResult] = useState<Preview | null>(null);
  const [view, setView] = useState<"provinces" | "terrain" | "heightmap">(
    "terrain",
  );

  const generate = async () => {
    if (!file) return;
    setBusy(true);
    setError(null);
    try {
      const form = new FormData();
      form.append("sketch", file);
      form.append("width", String(SIZES[size][0]));
      form.append("height", String(SIZES[size][1]));
      form.append("seed", seed || "1");
      const response = await fetch(
        `/api/go/projects/${projectId}/gui-render/world-map/sketch-preview`,
        { method: "POST", body: form },
      );
      const out = (await response.json().catch(() => ({}))) as Preview & {
        detail?: string;
      };
      if (!response.ok)
        throw new Error(out.detail || `HTTP ${response.status}`);
      setResult(out);
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setBusy(false);
    }
  };

  const sea = legend.data?.find((entry) => entry.key === "ocean")?.color;

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-5xl">
        <DialogHeader>
          <DialogTitle>{t("sketchTitle")}</DialogTitle>
          <DialogDescription>{t("sketchIntro")}</DialogDescription>
        </DialogHeader>
        <div className="grid gap-5 md:grid-cols-[16rem_1fr]">
          <div className="space-y-4">
            <section>
              <div className="text-xs text-muted-foreground">{t("legend")}</div>
              <LegendSwatches legend={legend.data ?? []} />
              <Button
                className="mt-2"
                size="sm"
                variant="outline"
                disabled={!sea}
                onClick={() => sea && downloadCanvas(sea)}
              >
                {t("downloadCanvas")}
              </Button>
            </section>
            <section className="space-y-2">
              <label className="block text-xs">
                <span className="text-muted-foreground">
                  {t("uploadSketch")}
                </span>
                <FilePick
                  className="mt-1"
                  accept="image/png,image/jpeg"
                  fileName={file?.name ?? null}
                  onPick={setFile}
                />
              </label>
              <div className="text-xs text-muted-foreground">{t("size")}</div>
              <SegmentedControl
                label={t("size")}
                size="sm"
                value={size}
                onChange={setSize}
                options={[
                  { value: "quick", label: t("sizeQuick") },
                  { value: "preview", label: t("sizePreview") },
                ]}
              />
              <label className="block text-xs">
                <span className="text-muted-foreground">{t("seed")}</span>
                <Input
                  className="mt-1"
                  inputMode="numeric"
                  value={seed}
                  onChange={(e) =>
                    setSeed(e.target.value.replace(/[^\d-]/g, ""))
                  }
                />
              </label>
              <Button
                size="sm"
                disabled={!file || busy}
                onClick={() => void generate()}
              >
                {busy ? t("generating") : t("generate")}
              </Button>
              {error && <p className="text-xs text-destructive">{error}</p>}
            </section>
          </div>
          <div className="min-w-0 space-y-3">
            {result ? (
              <>
                <div className="flex flex-wrap items-center justify-between gap-2">
                  <p className="text-xs text-muted-foreground">
                    {t("sketchSummary", {
                      ...result.provinces,
                      seconds: result.seconds.toFixed(1),
                      width: result.width,
                      height: result.height,
                    })}
                  </p>
                  <SegmentedControl
                    label={t("layers")}
                    size="sm"
                    className="w-64"
                    value={view}
                    onChange={setView}
                    options={[
                      { value: "terrain", label: t("viewTerrain") },
                      { value: "provinces", label: t("viewProvinces") },
                      { value: "heightmap", label: t("viewHeight") },
                    ]}
                  />
                </div>
                <img
                  src={result.previews[view]}
                  alt={t("sketchTitle")}
                  className="w-full rounded-md border border-border [image-rendering:pixelated]"
                />
                <section>
                  <div className="text-xs text-muted-foreground">
                    {t("findingsTitle")}
                  </div>
                  {result.findings.length === 0 ? (
                    <p className="mt-1 text-xs text-success">
                      {t("noFindings")}
                    </p>
                  ) : (
                    <ul className="mt-1 space-y-1 text-xs">
                      {result.findings.map((finding) => (
                        <li
                          key={finding.code}
                          className={
                            finding.severity === "error"
                              ? "text-destructive"
                              : "text-warning"
                          }
                        >
                          {finding.message} ×{finding.count}
                        </li>
                      ))}
                    </ul>
                  )}
                </section>
                <p className="text-xs text-muted-foreground">{t("notSaved")}</p>
              </>
            ) : (
              <div className="flex h-full min-h-48 items-center justify-center rounded-md border border-dashed border-border p-6 text-center text-xs text-muted-foreground">
                {busy ? t("generating") : t("sketchEmpty")}
              </div>
            )}
          </div>
        </div>
      </DialogContent>
    </Dialog>
  );
}
