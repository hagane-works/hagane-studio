"use client";

import { useCallback, useMemo, useRef, useState } from "react";
import { useTranslations } from "use-intl";
import useSWR from "swr";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { SegmentedControl } from "@/components/ui/SegmentedControl";
import { FilePick } from "@/shared/upload/FilePick";
import { useConfirm } from "@/components/shared/ConfirmDialogProvider";
import {
  downloadCanvas,
  LegendSwatches,
  sampleIsland,
  useSketchLegend,
} from "./sketchLegend";
import type { LandLayer, MapRect } from "./WorldMapCanvas";
import { withGameLanguage } from "@/lib/guiRenderApi";

interface Finding {
  code: string;
  severity: "error" | "warning";
  message: string;
  count: number;
}

export interface LandSummary {
  land: number;
  sea: number;
  lake: number;
  states: number;
  covered: number[];
  ignored: number;
  first_province: number;
  seconds: number;
}

interface LandPreview {
  summary: LandSummary;
  findings: Finding[];
  overlay: { rect: MapRect; provinces: string; terrain: string };
}

export interface LandAddition {
  id: number;

  kind: "land" | "edit" | "world";
  placement: MapRect;
  seed: number;
  name: string;
  owner: string;
  provinces: number[];
  state_ids: number[];
  region_id: number;
  created_at: string;
}

function loadImage(src: string): Promise<HTMLImageElement> {
  return new Promise((resolve, reject) => {
    const image = new Image();
    image.onload = () => resolve(image);
    image.onerror = () => reject(new Error("image unreadable"));
    image.src = src;
  });
}

type Sketch = { file: File; image: HTMLImageElement };

function previewKey(sketch: Sketch, rect: MapRect, seed: string): string {
  return `${sketch.file.name}:${sketch.file.size}:${sketch.file.lastModified}|${rect.join(",")}|${seed}`;
}

function landForm(
  sketch: Sketch,
  rect: MapRect,
  seed: string,
  extra: Record<string, string> = {},
): FormData {
  const body = new FormData();
  body.append("sketch", sketch.file);
  body.append("x", String(rect[0]));
  body.append("y", String(rect[1]));
  body.append("width", String(rect[2]));
  body.append("height", String(rect[3]));
  body.append("seed", seed || "1");
  for (const [name, value] of Object.entries(extra)) body.append(name, value);
  return body;
}

async function detail(response: Response): Promise<string> {
  const out = (await response.json().catch(() => ({}))) as { detail?: string };
  return out.detail || `HTTP ${response.status}`;
}

export function useNewLand(projectId: number, active: boolean) {
  const [sketch, setSketch] = useState<Sketch | null>(null);
  const [rect, setRect] = useState<MapRect | null>(null);
  const [seed, setSeed] = useState("1");
  const [view, setView] = useState<"provinces" | "terrain">("terrain");
  const [preview, setPreview] = useState<{
    key: string;
    data: LandPreview;
    images: Record<"provinces" | "terrain", HTMLImageElement>;
  } | null>(null);
  const [busy, setBusy] = useState<
    "preview" | "save" | "restore" | "undo" | null
  >(null);
  const [error, setError] = useState<string | null>(null);

  const additions = useSWR(["world-map-land", projectId], async () => {
    const response = await fetch(
      withGameLanguage(
        `/api/go/projects/${projectId}/gui-render/world-map/land`,
      ),
    );
    if (!response.ok) throw new Error(await detail(response));
    return ((await response.json()) as { additions: LandAddition[] }).additions;
  });

  const key = sketch && rect ? previewKey(sketch, rect, seed) : "";
  const fresh = preview && preview.key === key ? preview : null;

  const latestPreview = useRef(0);

  const previewWith = useCallback(
    async (s: Sketch, r: MapRect, seedValue: string) => {
      const request = ++latestPreview.current;
      setBusy("preview");
      setError(null);
      try {
        const response = await fetch(
          `/api/go/projects/${projectId}/gui-render/world-map/land-preview`,
          { method: "POST", body: landForm(s, r, seedValue) },
        );
        if (!response.ok) throw new Error(await detail(response));
        const data = (await response.json()) as LandPreview;
        const [provinces, terrain] = await Promise.all([
          loadImage(data.overlay.provinces),
          loadImage(data.overlay.terrain),
        ]);
        if (request === latestPreview.current)
          setPreview({
            key: previewKey(s, r, seedValue),
            data,
            images: { provinces, terrain },
          });
      } catch (err) {
        if (request === latestPreview.current)
          setError(err instanceof Error ? err.message : String(err));
      } finally {
        if (request === latestPreview.current) setBusy(null);
      }
    },
    [projectId],
  );

  const place = useCallback(
    (r: MapRect) => {
      setRect(r);
      if (sketch) void previewWith(sketch, r, seed);
    },
    [sketch, seed, previewWith],
  );

  const chooseSketch = async (file: File | null) => {
    setError(null);
    if (!file) {
      setSketch(null);
      return;
    }
    const url = URL.createObjectURL(file);
    try {
      const chosen = { file, image: await loadImage(url) };
      setSketch(chosen);
      if (rect) void previewWith(chosen, rect, seed);
    } catch {
      setError("sketchUnreadable");
    } finally {
      URL.revokeObjectURL(url);
    }
  };

  const runPreview = async () => {
    if (sketch && rect) await previewWith(sketch, rect, seed);
  };

  const save = async (
    name: string,
    owner: string,
    languages: string[],
  ): Promise<{ addition: LandAddition; summary: LandSummary } | null> => {
    if (!sketch || !rect) return null;
    setBusy("save");
    setError(null);
    try {
      const response = await fetch(
        `/api/go/projects/${projectId}/gui-render/world-map/land`,
        {
          method: "POST",
          body: landForm(sketch, rect, seed, {
            name,
            owner,
            languages: languages.join(","),
          }),
        },
      );
      if (!response.ok) throw new Error(await detail(response));
      const out = (await response.json()) as {
        addition: LandAddition;
        summary: LandSummary;
      };
      await additions.mutate();
      setSketch(null);
      setRect(null);
      setPreview(null);
      return out;
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
      return null;
    } finally {
      setBusy(null);
    }
  };

  const restore = async (latest = false): Promise<boolean> => {
    setBusy(latest ? "undo" : "restore");
    setError(null);
    try {
      const response = await fetch(
        `/api/go/projects/${projectId}/gui-render/world-map/land${latest ? "?latest=1" : ""}`,
        { method: "DELETE" },
      );
      if (!response.ok) throw new Error(await detail(response));
      await additions.mutate();
      return true;
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
      return false;
    } finally {
      setBusy(null);
    }
  };

  const layer: LandLayer | null = useMemo(
    () =>
      active
        ? {
            sketch: sketch?.image ?? null,
            rect,
            overlay: fresh
              ? { image: fresh.images[view], rect: fresh.data.overlay.rect }
              : null,
            onRect: place,
          }
        : null,
    [active, sketch, rect, fresh, view, place],
  );

  return {
    sketch,
    rect,
    seed,
    setSeed,
    view,
    setView,
    fresh,
    busy,
    error,
    additions,
    chooseSketch,
    runPreview,
    save,
    restore,
    layer,
  };
}

export type NewLand = ReturnType<typeof useNewLand>;

export function NewLandPanel({
  projectId,
  land,
  owners,
  languages,
  onSaved,
  onRestored,
}: {
  projectId: number;
  land: NewLand;

  languages: string[];

  owners: string[];
  onSaved: (name: string, owner: string) => Promise<void> | void;
  onRestored: (latest: boolean) => Promise<void> | void;
}) {
  const t = useTranslations("worldMap");
  const confirm = useConfirm();
  const legend = useSketchLegend(projectId, true);
  const sea = legend.data?.find((entry) => entry.key === "ocean")?.color;
  const [name, setName] = useState("");
  const [owner, setOwner] = useState("");
  const summary = land.fresh?.data.summary;
  const findings = land.fresh?.data.findings ?? [];
  const blocked = findings.some((finding) => finding.severity === "error");
  const records = land.additions.data ?? [];
  const additions = records.filter((a) => a.kind === "land");
  const edits = records.filter((a) => a.kind === "edit").length;
  const errorText =
    land.error === "sketchUnreadable" ? t("landSketchUnreadable") : land.error;

  const waitingFor = !land.sketch
    ? t("landNeedSketch")
    : !land.rect
      ? t("landNeedFrame")
      : land.busy === "preview"
        ? t("landPreviewing")
        : !land.fresh
          ? t("landNeedPreview")
          : blocked
            ? t("landBlocked")
            : owner !== "" && owner.length !== 3
              ? t("landOwnerShort")
              : null;
  const step = (n: number, title: string) => (
    <div className="flex items-center gap-2 text-xs">
      <span className="flex h-5 w-5 shrink-0 items-center justify-center rounded-full border border-border tabular-nums">
        {n}
      </span>
      <span>{title}</span>
    </div>
  );

  return (
    <div className="space-y-5">
      <section>
        <h2 className="font-medium">{t("landTitle")}</h2>
        <p className="mt-1 text-xs text-muted-foreground">{t("landIntro")}</p>
      </section>

      <section className="space-y-2">
        {step(1, t("landStepSketch"))}
        <LegendSwatches legend={legend.data ?? []} />
        <div className="flex flex-wrap gap-2">
          <Button
            size="sm"
            variant="outline"
            disabled={!legend.data}
            onClick={() =>
              void sampleIsland(legend.data ?? []).then(
                (file) => file && land.chooseSketch(file),
              )
            }
          >
            {t("landSample")}
          </Button>
          <Button
            size="sm"
            variant="outline"
            disabled={!sea}
            onClick={() =>
              sea && downloadCanvas(sea, 600, 300, "new-land-sketch.png")
            }
          >
            {t("downloadCanvas")}
          </Button>
        </div>
        <label className="block text-xs">
          <span className="text-muted-foreground">{t("uploadSketch")}</span>
          <FilePick
            className="mt-1"
            accept="image/png,image/jpeg"
            onPick={(file) => void land.chooseSketch(file)}
          />
        </label>
        {land.sketch && (
          <p className="text-xs text-muted-foreground">
            {t("landSketchChosen", { name: land.sketch.file.name })}
          </p>
        )}
      </section>

      <section className="space-y-2">
        {step(2, t("landStepFrame"))}
        <p className="text-xs text-muted-foreground">
          {land.rect
            ? t("landFrame", {
                x: land.rect[0],
                y: land.rect[1],
                w: land.rect[2],
                h: land.rect[3],
              })
            : t("landFrameHint")}
        </p>
      </section>

      <section className="space-y-2">
        {step(3, t("landStepPreview"))}
        {summary ? (
          <>
            <div className="flex items-center justify-between gap-2">
              <div className="text-xs text-muted-foreground">
                {t("landResult")}
              </div>
              <SegmentedControl
                label={t("layers")}
                size="sm"
                className="w-40"
                value={land.view}
                onChange={land.setView}
                options={[
                  { value: "terrain", label: t("viewTerrain") },
                  { value: "provinces", label: t("viewProvinces") },
                ]}
              />
            </div>
            <p className="text-xs">
              {t("landSummary", {
                land: summary.land,
                water: summary.sea + summary.lake,
                states: summary.states,
                first: summary.first_province,
              })}
            </p>
            {summary.covered.length > 0 && (
              <p className="text-xs text-muted-foreground">
                {t("landCovered", {
                  list: summary.covered.map((id) => `#${id}`).join("、"),
                })}
              </p>
            )}
            {summary.ignored > 0 && (
              <p className="text-xs text-warning">
                {t("landIgnored", { pixels: summary.ignored })}
              </p>
            )}
            {findings.length === 0 ? (
              <p className="text-xs text-success">{t("landNoFindings")}</p>
            ) : (
              <ul className="space-y-1 text-xs">
                {findings.map((finding) => (
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
          </>
        ) : (
          <p className="text-xs text-muted-foreground">
            {land.busy === "preview"
              ? t("landPreviewing")
              : t("landAutoPreview")}
          </p>
        )}
        <div className="flex items-end gap-2">
          <label className="block flex-1 text-xs">
            <span className="text-muted-foreground">{t("seed")}</span>
            <Input
              className="mt-1"
              inputMode="numeric"
              value={land.seed}
              onChange={(e) =>
                land.setSeed(e.target.value.replace(/[^\d-]/g, ""))
              }
            />
          </label>
          <Button
            size="sm"
            variant="outline"
            disabled={!land.sketch || !land.rect || land.busy !== null}
            onClick={() => void land.runPreview()}
          >
            {land.fresh ? t("landRepreview") : t("landPreview")}
          </Button>
        </div>
        {errorText && <p className="text-xs text-destructive">{errorText}</p>}
      </section>

      <section className="space-y-2">
        {step(4, t("landStepSave"))}
        <label className="block text-xs">
          <span className="text-muted-foreground">{t("landName")}</span>
          <Input
            className="mt-1"
            value={name}
            maxLength={60}
            placeholder="New Land"
            onChange={(e) => setName(e.target.value)}
          />
        </label>
        <label className="block text-xs">
          <span className="text-muted-foreground">{t("landOwner")}</span>
          <Input
            className="mt-1 font-mono uppercase"
            value={owner}
            maxLength={3}
            list="world-map-owners"
            placeholder={t("landOwnerNone")}
            onChange={(e) =>
              setOwner(e.target.value.toUpperCase().replace(/[^A-Z0-9]/g, ""))
            }
          />
          <datalist id="world-map-owners">
            {owners.map((tag) => (
              <option key={tag} value={tag} />
            ))}
          </datalist>
        </label>
        {owner === "" && (
          <p className="text-xs text-warning">{t("landOwnerHint")}</p>
        )}
      </section>

      <section className="space-y-2 border-t border-border pt-4">
        <div className="text-xs text-muted-foreground">{t("landAdded")}</div>
        {records.length === 0 ? (
          <p className="text-xs text-muted-foreground">{t("landNoneYet")}</p>
        ) : (
          <>
            <ul className="space-y-1 text-xs">
              {additions.map((a) => (
                <li key={a.id}>
                  <span className="font-medium">{a.name}</span>
                  <span className="text-muted-foreground">
                    {" · "}
                    {t("landAddedLine", {
                      states: a.state_ids.length,
                      first: a.provinces[0] ?? 0,
                      last: a.provinces[a.provinces.length - 1] ?? 0,
                    })}
                    {a.owner ? ` · ${a.owner}` : ""}
                  </span>
                </li>
              ))}
            </ul>
            {edits > 0 && (
              <p className="text-xs text-muted-foreground">
                {t("landEdits", { count: edits })}
              </p>
            )}
            <Button
              size="sm"
              variant="outline"
              className="mr-2"
              disabled={land.busy !== null}
              onClick={async () => {
                const latest = records[records.length - 1];
                const body =
                  latest.kind === "edit"
                    ? t("landUndoEdit", { id: latest.provinces[0] ?? 0 })
                    : latest.kind === "world"
                      ? t("worldUndoWorld")
                      : t("landUndoLand", { name: latest.name });
                if (
                  !(await confirm({
                    title: t("landUndoTitle"),
                    body,
                    danger: true,
                    confirmLabel: t("landUndo"),
                  }))
                )
                  return;
                if (await land.restore(true)) await onRestored(true);
              }}
            >
              {land.busy === "undo" ? t("landRestoring") : t("landUndo")}
            </Button>
            <Button
              size="sm"
              variant="outline"
              disabled={land.busy !== null}
              onClick={async () => {
                if (
                  !(await confirm({
                    title: t("landRestoreTitle"),
                    body: [
                      records.some((a) => a.kind === "world")
                        ? t("worldUndoWorld")
                        : "",
                      additions.length > 0
                        ? t("landRestoreBodyLand", { count: additions.length })
                        : "",
                      edits > 0 ? t("landRestoreBodyEdits", { edits }) : "",
                      t("landRestoreBodyKeep"),
                    ]
                      .filter(Boolean)
                      .join(t("sentenceGap")),
                    danger: true,
                    confirmLabel: t("landRestore"),
                  }))
                )
                  return;
                if (await land.restore()) await onRestored(false);
              }}
            >
              {land.busy === "restore" ? t("landRestoring") : t("landRestore")}
            </Button>
          </>
        )}
      </section>

      <div className="sticky bottom-0 -mx-4 -mb-4 space-y-1 border-t border-border bg-card px-4 py-3">
        <Button
          size="sm"
          className="w-full"
          disabled={waitingFor !== null || land.busy !== null}
          onClick={async () => {
            if (await land.save(name.trim(), owner, languages)) {
              await onSaved(name.trim() || "New Land", owner);
              setName("");
            }
          }}
        >
          {land.busy === "save" ? t("landSaving") : t("landSave")}
        </Button>
        <p className="text-xs text-muted-foreground">
          {waitingFor ?? t("landSaveNote")}
        </p>
      </div>
    </div>
  );
}
