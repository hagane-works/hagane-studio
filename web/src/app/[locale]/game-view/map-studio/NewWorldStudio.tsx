"use client";

import { mapToolFetch, mapResponseProblem } from "@/lib/mapToolRequest";

import {
  useEffect,
  useMemo,
  useRef,
  useState,
  useSyncExternalStore,
} from "react";
import { createPortal } from "react-dom";
import { useLocale, useTranslations } from "use-intl";
import useSWR from "swr";
import {
  Brush,
  Eraser,
  Mountain,
  PaintBucket,
  Redo2,
  Undo2,
  Maximize,
} from "lucide-react";
import { Link } from "@/i18n/navigation";
import { workbenchTargetOf } from "@/lib/gameViewLinks";
import {
  mapFindingFix,
  mapFindingMessage,
  mapProblemMessage,
} from "./mapProblems";
import { projectsApi } from "@/lib/api/project";
import { SegmentedControl } from "@/components/ui/SegmentedControl";
import { useConfirm } from "@/components/shared/ConfirmDialogProvider";
import { AddCountryDialog } from "@/components/workbench/AddCountryDialog";
import { nameLanguages } from "@/components/workbench/world-map/mapNames";
import { loadWorldMap } from "@/components/workbench/world-map/worldMapApi";
import { useSketchLegend } from "@/components/workbench/world-map/sketchLegend";
import {
  unloadedLabel,
  WORLD_MAX_COUNTRIES,
  WORLD_SIZES,
  type WorldPreview,
  type WorldSize,
} from "@/components/workbench/world-map/newWorldModel";
import { BUTTON, GamePanel, PRIMARY } from "../GamePanel";
import { StudioCanvas } from "./StudioCanvas";
import {
  AddCountrySelect,
  colourFor,
  longTagCount,
  swatch,
  TAG,
  useStudioCountries,
} from "./studioCountries";
import { withGameLanguage } from "@/lib/guiRenderApi";
import {
  applyPatch,
  blankSketch,
  cloneSketch,
  countryShares,
  decodeDraft,
  draftKey,
  encodeDraft,
  legendPixels,
  ownerPixels,
  patchOf,
  RANDOM_DEFAULTS,
  randomWorld,
  removeCountry,
  sampleWorld,
  TERRAINS,
  type Brush as BrushState,
  type Draft,
  type Patch,
  type RandomOptions,
  type Sketch,
  type Tool,
} from "./studioSketch";

type Step = "terrain" | "countries" | "make";
type Shown = "sketch" | "political" | "terrain";

const UNDO_LIMIT = 80;

function readDraft(projectId: number): Draft {
  try {
    const saved = decodeDraft(window.localStorage.getItem(draftKey(projectId)));
    if (saved) return saved;
  } catch {}
  return {
    sketch: randomWorld({ seed: 1, ...RANDOM_DEFAULTS }),
    countries: [],
    size: "medium",
    seed: 1,
  };
}

function pngOf(
  pixels: Uint8ClampedArray<ArrayBuffer>,
  w: number,
  h: number,
): Promise<Blob> {
  const canvas = document.createElement("canvas");
  canvas.width = w;
  canvas.height = h;
  canvas.getContext("2d")?.putImageData(new ImageData(pixels, w, h), 0, 0);
  return new Promise((resolve, reject) =>
    canvas.toBlob(
      (blob) => (blob ? resolve(blob) : reject(new Error("png"))),
      "image/png",
    ),
  );
}

function loadImage(src: string): Promise<HTMLImageElement> {
  return new Promise((resolve, reject) => {
    const image = new Image();
    image.onload = () => resolve(image);
    image.onerror = () => reject(new Error("preview unreadable"));
    image.src = src;
  });
}

const subscribeNothing = () => () => {};

export function NewWorldStudio({
  projectId,
  date,
  width,
  height,
  onClose,
  onBack,
  onSaved,
}: {
  projectId: number;

  date: string;

  width: number;
  height: number;
  onClose: () => void;

  onBack: () => void;

  onSaved: (states: number) => Promise<void> | void;
}) {
  const t = useTranslations("mapStudio");
  const tw = useTranslations("worldMap");
  const locale = useLocale();
  const confirm = useConfirm();

  const mounted = useSyncExternalStore(
    subscribeNothing,
    () => true,
    () => false,
  );

  const [initial] = useState(() =>
    typeof window === "undefined" ? null : readDraft(projectId),
  );
  const sketch = useRef<Sketch>(initial?.sketch ?? blankSketch());
  const undoStack = useRef<Patch[]>([]);
  const redoStack = useRef<Patch[]>([]);
  const [history, setHistory] = useState({ undo: 0, redo: 0 });
  const [version, setVersion] = useState(0);
  const [fitRequest, setFitRequest] = useState(0);
  const [countries, setCountries] = useState<string[]>(
    initial?.countries ?? [],
  );
  const [shares, setShares] = useState(() =>
    countryShares(sketch.current, countries.length),
  );
  const [step, setStep] = useState<Step>("terrain");
  const [terrainTool, setTerrainTool] = useState<Tool>("paint");
  const [countryTool, setCountryTool] = useState<Tool>("country");
  const [terrain, setTerrain] = useState(1);
  const [country, setCountry] = useState(1);
  const [radius, setRadius] = useState(10);
  const [natural, setNatural] = useState(true);
  const [random, setRandom] = useState<RandomOptions>({
    seed: 1,
    ...RANDOM_DEFAULTS,
  });
  const [size, setSize] = useState<WorldSize>(initial?.size ?? "medium");
  const [seed, setSeed] = useState(initial?.seed ?? 1);
  const [preview, setPreview] = useState<{
    key: string;
    data: WorldPreview;
    images: Record<"political" | "terrain", HTMLImageElement | null>;
  } | null>(null);
  const [shown, setShown] = useState<Shown>("sketch");
  const [busy, setBusy] = useState<"preview" | "save" | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [confirmed, setConfirmed] = useState(false);
  const [addOpen, setAddOpen] = useState(false);

  const project = useSWR(
    ["game-view-project", projectId],
    () => projectsApi.get(projectId),
    { revalidateOnFocus: false },
  );
  const additions = useSWR(["world-map-land", projectId], async () => {
    const response = await mapToolFetch(
      withGameLanguage(
        `/api/go/projects/${projectId}/gui-render/world-map/land`,
      ),
    );
    if (!response.ok)
      throw new Error(
        mapProblemMessage(await mapResponseProblem(response), t, locale),
      );
    return ((await response.json()) as { additions: unknown[] }).additions;
  });
  const legend = useSketchLegend(projectId, true);

  const worldMap = useSWR(
    ["world-map", projectId, date],
    () => loadWorldMap(projectId, date),
    { revalidateOnFocus: false },
  );
  const mapColours = worldMap.data?.colors;
  const countryInfo = useStudioCountries(projectId, mapColours);
  const { rows, rowOf, nameOf } = countryInfo;
  const mapOwners = useMemo(
    () => [
      ...new Set(
        (worldMap.data?.states ?? [])
          .map(([, , owner]) => owner)
          .filter(Boolean),
      ),
    ],
    [worldMap.data],
  );
  const exportLanguages = project.data?.export_languages;
  const languages = useMemo(
    () => nameLanguages(exportLanguages),
    [exportLanguages],
  );

  const colours = useMemo(
    () => countries.map((tag) => colourFor(rowOf, mapColours, tag)),
    [countries, rowOf, mapColours],
  );
  const look = useMemo(
    () => ({ countries: colours, showCountries: step !== "terrain" }),
    [colours, step],
  );
  const brush = useMemo<BrushState>(
    () => ({
      tool: step === "countries" ? countryTool : terrainTool,
      terrain,
      country,
      radius,
      natural,
    }),
    [step, countryTool, terrainTool, terrain, country, radius, natural],
  );

  const key = `${version}|${countries.join(",")}|${size}|${seed}`;
  const fresh = preview && preview.key === key ? preview : null;
  const overlay = fresh && shown !== "sketch" ? fresh.images[shown] : null;
  const findings = fresh?.data.findings ?? [];
  const blocked = findings.some((finding) => finding.severity === "error");
  const notBase = (additions.data ?? []).length > 0;

  useEffect(() => {
    const timer = window.setTimeout(() => {
      try {
        window.localStorage.setItem(
          draftKey(projectId),
          encodeDraft({ sketch: sketch.current, countries, size, seed }),
        );
      } catch {}
    }, 600);
    return () => window.clearTimeout(timer);
  }, [projectId, version, countries, size, seed]);

  const changed = (count = countries.length) => {
    setVersion((v) => v + 1);
    setShares(countryShares(sketch.current, count));
    setHistory({
      undo: undoStack.current.length,
      redo: redoStack.current.length,
    });
  };
  const record = (patch: Patch | null) => {
    if (!patch) return;
    undoStack.current.push(patch);
    if (undoStack.current.length > UNDO_LIMIT) undoStack.current.shift();
    redoStack.current = [];
  };
  const undo = () => {
    const patch = undoStack.current.pop();
    if (!patch) return;
    applyPatch(sketch.current, patch, "before");
    redoStack.current.push(patch);
    changed();
  };
  const redo = () => {
    const patch = redoStack.current.pop();
    if (!patch) return;
    applyPatch(sketch.current, patch, "after");
    undoStack.current.push(patch);
    changed();
  };
  const start = (next: Sketch) => {
    const before = cloneSketch(sketch.current);
    record(patchOf(before, next, { x0: 0, y0: 0, x1: next.w, y1: next.h }));
    sketch.current.terrain.set(next.terrain);
    sketch.current.owner.set(next.owner);
    changed();
    setFitRequest((n) => n + 1);
  };
  const addCountry = (tag: string) => {
    if (
      !TAG.test(tag) ||
      countries.includes(tag) ||
      countries.length >= WORLD_MAX_COUNTRIES
    )
      return;
    const next = [...countries, tag];
    setCountries(next);
    setCountry(next.length);
    setShares(countryShares(sketch.current, next.length));
  };
  const dropCountry = async (slot: number) => {
    const tag = countries[slot - 1];
    if (!tag) return;
    if (
      !(await confirm({
        title: t("removeTitle", { name: nameOf(tag) }),
        body: t("removeBody"),
        danger: true,
        confirmLabel: t("removeConfirm"),
      }))
    )
      return;
    removeCountry(sketch.current, slot);

    undoStack.current = [];
    redoStack.current = [];
    const next = countries.filter((_, i) => i !== slot - 1);
    setCountries(next);
    setCountry((c) => Math.max(1, Math.min(next.length, c > slot ? c - 1 : c)));
    changed(next.length);
  };

  const keys = useRef({ undo, redo });
  useEffect(() => {
    keys.current = { undo, redo };
  });
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      const target = e.target as HTMLElement | null;
      if (
        target &&
        (target.tagName === "INPUT" ||
          target.tagName === "TEXTAREA" ||
          target.tagName === "SELECT" ||
          target.isContentEditable)
      )
        return;
      if (e.key === "Escape") return;
      e.stopPropagation();
      const mod = e.ctrlKey || e.metaKey;
      if (mod && e.key.toLowerCase() === "z") {
        e.preventDefault();
        if (e.shiftKey) keys.current.redo();
        else keys.current.undo();
      } else if (mod && e.key.toLowerCase() === "y") {
        e.preventDefault();
        keys.current.redo();
      } else if (e.key === "[") {
        setRadius((r) => Math.max(1, Math.round(r / 1.25)));
      } else if (e.key === "]") {
        setRadius((r) => Math.min(80, Math.round(r * 1.25) + 1));
      } else if (e.code === "Space") {
        e.preventDefault();
      }
    };
    window.addEventListener("keydown", onKey, true);
    return () => window.removeEventListener("keydown", onKey, true);
  }, []);

  const generate = async (save: boolean) => {
    setBusy(save ? "save" : "preview");
    setError(null);
    try {
      const s = sketch.current;
      const legendColours = Object.fromEntries(
        (legend.data ?? []).map((entry) => [entry.key, entry.color] as const),
      );
      const [picture, owners] = await Promise.all([
        pngOf(legendPixels(s, legendColours), s.w, s.h),
        pngOf(ownerPixels(s), s.w, s.h),
      ]);
      const body = new FormData();
      body.append("source", "sketch");
      body.append("sketch", picture, "sketch.png");
      body.append("owners", owners, "owners.png");
      body.append("land", String(WORLD_SIZES[size]));
      body.append("seed", String(seed));
      body.append("tags", countries.join(","));
      body.append("languages", languages.join(","));
      const madeFor = key;
      const response = await mapToolFetch(
        `/api/go/projects/${projectId}/gui-render/world-map/${save ? "world" : "world-preview"}`,
        { method: "POST", body },
      );
      if (!response.ok)
        throw new Error(
          mapProblemMessage(await mapResponseProblem(response), t, locale),
        );
      if (save) {
        const out = (await response.json()) as {
          addition: { state_ids: number[] };
        };
        await additions.mutate();

        await onSaved(out.addition.state_ids.length);
        setPreview(null);
        setConfirmed(false);
        setShown("sketch");
        return;
      }
      const data = (await response.json()) as WorldPreview;
      const [political, terrainImage] = await Promise.all([
        loadImage(data.previews.political).catch(() => null),
        data.previews.terrain
          ? loadImage(data.previews.terrain).catch(() => null)
          : Promise.resolve(null),
      ]);
      setPreview({
        key: madeFor,
        data,
        images: { political, terrain: terrainImage },
      });
      setShown("political");
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setBusy(null);
    }
  };

  const longTags = longTagCount(countryInfo);
  const percent = (n: number) =>
    shares.land ? Math.round((n / shares.land) * 1000) / 10 : 0;
  const unpaintedCountries = countries.filter((_, i) => !shares.owned[i]);
  const waitingFor = notBase
    ? t("waitNotBase")
    : countries.length === 0
      ? t("needCountries")
      : busy === "preview"
        ? t("generating")
        : !fresh
          ? preview
            ? t("stale")
            : t("waitPreview")
          : blocked
            ? t("waitBlocked")
            : !confirmed
              ? t("waitConfirm")
              : null;

  const terrainTools: {
    tool: Tool;
    icon: typeof Brush;
    label: string;
    hint: string;
  }[] = [
    {
      tool: "paint",
      icon: Brush,
      label: t("toolPaint"),
      hint: t("toolPaintHint"),
    },
    {
      tool: "terrain",
      icon: Mountain,
      label: t("toolTerrain"),
      hint: t("toolTerrainHint"),
    },
    {
      tool: "fill",
      icon: PaintBucket,
      label: t("toolFill"),
      hint: t("toolFillHint"),
    },
  ];
  const countryTools: {
    tool: Tool;
    icon: typeof Brush;
    label: string;
    hint: string;
  }[] = [
    {
      tool: "country",
      icon: Brush,
      label: t("toolCountry"),
      hint: t("toolCountryHint"),
    },
    {
      tool: "countryFill",
      icon: PaintBucket,
      label: t("toolCountryFill"),
      hint: t("toolCountryFillHint"),
    },
    {
      tool: "erase",
      icon: Eraser,
      label: t("toolErase"),
      hint: t("toolEraseHint"),
    },
  ];
  const toolRow = (
    tools: typeof terrainTools,
    current: Tool,
    choose: (tool: Tool) => void,
  ) => (
    <div
      role="radiogroup"
      aria-label={t("toolsTitle")}
      className="grid grid-cols-3 gap-1"
    >
      {tools.map(({ tool, icon: Icon, label, hint }) => (
        <button
          key={tool}
          type="button"
          role="radio"
          aria-checked={current === tool}
          title={hint}
          onClick={() => choose(tool)}
          className={`ui-interactive ui-focus-ring flex flex-col items-center gap-1 rounded-md border px-1 py-2 text-xs ${current === tool ? "border-accent bg-accent/15 text-foreground" : "border-border text-muted-foreground hover:bg-muted"}`}
        >
          <Icon className="h-4 w-4" aria-hidden />
          {label}
        </button>
      ))}
    </div>
  );
  const usesRadius =
    brush.tool === "paint" ||
    brush.tool === "terrain" ||
    brush.tool === "country" ||
    brush.tool === "erase";

  const tools = (
    <div
      data-map-studio-tools=""
      className="pointer-events-auto absolute left-3 top-16 max-h-[calc(100vh-8rem)] w-60 space-y-3 overflow-y-auto rounded-md border border-border bg-card p-3 text-sm text-card-foreground shadow-lg"
    >
      {step === "terrain" && (
        <>
          <div className="text-xs text-muted-foreground">{t("toolsTitle")}</div>
          {toolRow(terrainTools, terrainTool, setTerrainTool)}
          <div className="text-xs text-muted-foreground">
            {t("terrainsTitle")}
          </div>
          <div
            role="radiogroup"
            aria-label={t("terrainsTitle")}
            className="grid grid-cols-2 gap-1"
          >
            {TERRAINS.map((kind, cls) => (
              <button
                key={kind.key}
                type="button"
                role="radio"
                aria-checked={terrain === cls}
                disabled={terrainTool === "terrain" && cls === 0}
                onClick={() => setTerrain(cls)}
                className={`ui-interactive ui-focus-ring flex items-center gap-2 rounded-md border px-2 py-1 text-left text-xs disabled:cursor-not-allowed disabled:opacity-50 ${terrain === cls ? "border-accent bg-accent/15" : "border-border hover:bg-muted"}`}
              >
                <span
                  className="h-3.5 w-3.5 shrink-0 rounded-sm border border-border"
                  style={swatch(kind.look)}
                  aria-hidden
                />
                {t(`terrain_${kind.key}`)}
              </button>
            ))}
          </div>
        </>
      )}
      {step === "countries" && (
        <>
          <div className="text-xs text-muted-foreground">{t("toolsTitle")}</div>
          {toolRow(countryTools, countryTool, setCountryTool)}
          <div className="text-xs text-muted-foreground">
            {t("countriesTitle")}
          </div>
          {countries.length === 0 ? (
            <p className="text-xs text-muted-foreground">
              {t("countriesEmpty")}
            </p>
          ) : (
            <div
              role="radiogroup"
              aria-label={t("countriesTitle")}
              className="space-y-1"
            >
              {countries.map((tag, i) => (
                <button
                  key={tag}
                  type="button"
                  role="radio"
                  aria-checked={country === i + 1}
                  onClick={() => setCountry(i + 1)}
                  className={`ui-interactive ui-focus-ring flex w-full items-center gap-2 rounded-md border px-2 py-1 text-left text-xs ${country === i + 1 ? "border-accent bg-accent/15" : "border-border hover:bg-muted"}`}
                >
                  <span
                    className="h-3.5 w-3.5 shrink-0 rounded-sm border border-border"
                    style={swatch(colours[i])}
                    aria-hidden
                  />
                  <span className="min-w-0 flex-1 truncate">{nameOf(tag)}</span>
                  <span className="shrink-0 tabular-nums text-muted-foreground">
                    {percent(shares.owned[i] ?? 0)}%
                  </span>
                </button>
              ))}
            </div>
          )}
        </>
      )}
      {step !== "make" && (
        <>
          {usesRadius && (
            <label className="block text-xs">
              <span className="flex justify-between text-muted-foreground">
                {t("brushSize")}
                <span className="tabular-nums">{radius}</span>
              </span>
              <input
                type="range"
                min={1}
                max={80}
                value={radius}
                onChange={(e) => setRadius(Number(e.target.value))}
                className="mt-1 w-full accent-accent"
              />
            </label>
          )}
          {usesRadius && step === "terrain" && (
            <label className="flex items-center gap-2 text-xs">
              <input
                type="checkbox"
                className="ui-focus-ring"
                checked={natural}
                onChange={(e) => setNatural(e.target.checked)}
              />
              {t("natural")}
            </label>
          )}
          <div className="flex gap-1">
            <button
              type="button"
              className={`${BUTTON} flex items-center gap-1`}
              disabled={history.undo === 0}
              onClick={undo}
              title="Ctrl+Z"
            >
              <Undo2 className="h-3.5 w-3.5" aria-hidden />
              {t("undo")}
            </button>
            <button
              type="button"
              className={`${BUTTON} flex items-center gap-1`}
              disabled={history.redo === 0}
              onClick={redo}
              title="Ctrl+Y"
            >
              <Redo2 className="h-3.5 w-3.5" aria-hidden />
              {t("redo")}
            </button>
            <button
              type="button"
              className={`${BUTTON} ml-auto flex items-center gap-1`}
              onClick={() => setFitRequest((n) => n + 1)}
              title={t("fit")}
            >
              <Maximize className="h-3.5 w-3.5" aria-hidden />
              <span className="sr-only">{t("fit")}</span>
            </button>
          </div>
        </>
      )}
      <p className="text-xs text-muted-foreground">{t("hints")}</p>
    </div>
  );

  const slider = (
    label: string,
    value: number,
    min: number,
    max: number,
    stepBy: number,
    onChange: (v: number) => void,
    shownValue: string,
  ) => (
    <label className="block text-xs">
      <span className="flex justify-between text-muted-foreground">
        {label}
        <span className="tabular-nums">{shownValue}</span>
      </span>
      <input
        type="range"
        min={min}
        max={max}
        step={stepBy}
        value={value}
        onChange={(e) => onChange(Number(e.target.value))}
        className="mt-1 w-full accent-accent"
      />
    </label>
  );

  const panel = (
    <GamePanel
      title={t("newWorldTitle")}
      kind={t("newWorldKind")}
      workbenchHref={`/project/${projectId}/world-map`}
      onClose={onClose}
    >
      <button
        type="button"
        className="mt-2 text-xs text-accent hover:underline"
        onClick={onBack}
      >
        {t("backToMap")}
      </button>
      <SegmentedControl
        label={t("stepsLabel")}
        size="sm"
        className="mt-3"
        value={step}
        onChange={setStep}
        options={[
          { value: "terrain", label: t("stepTerrain") },
          { value: "countries", label: t("stepCountries") },
          { value: "make", label: t("stepMake") },
        ]}
      />
      <div className="mt-3 space-y-4 text-sm">
        {step === "terrain" && (
          <>
            <section className="space-y-2">
              <h3 className="text-xs font-medium">{t("startTitle")}</h3>
              <p className="text-xs text-muted-foreground">{t("startIntro")}</p>
              <div className="space-y-2 rounded-md border border-border p-2">
                {slider(
                  t("landShare"),
                  random.land,
                  0.1,
                  0.6,
                  0.05,
                  (land) => setRandom((r) => ({ ...r, land })),
                  `${Math.round(random.land * 100)}%`,
                )}
                {slider(
                  t("continents"),
                  random.continents,
                  1,
                  6,
                  1,
                  (continents) => setRandom((r) => ({ ...r, continents })),
                  String(random.continents),
                )}
                {slider(
                  t("mountains"),
                  random.mountains,
                  0,
                  1,
                  0.1,
                  (mountains) => setRandom((r) => ({ ...r, mountains })),
                  `${Math.round(random.mountains * 100)}%`,
                )}
                <div className="flex gap-1">
                  <button
                    type="button"
                    className={PRIMARY}
                    onClick={() => start(randomWorld(random))}
                  >
                    {t("startRandom")}
                  </button>
                  <button
                    type="button"
                    className={BUTTON}
                    onClick={() => {
                      const next = { ...random, seed: random.seed + 1 };
                      setRandom(next);
                      start(randomWorld(next));
                    }}
                  >
                    {t("startRandomAgain")}
                  </button>
                </div>
              </div>
              <div className="flex gap-1">
                <button
                  type="button"
                  className={BUTTON}
                  onClick={() => start(sampleWorld())}
                >
                  {t("startSample")}
                </button>
                <button
                  type="button"
                  className={BUTTON}
                  onClick={() => start(blankSketch())}
                >
                  {t("startBlank")}
                </button>
              </div>
            </section>
            <button
              type="button"
              className={PRIMARY}
              onClick={() => setStep("countries")}
            >
              {t("terrainNext")}
            </button>
          </>
        )}

        {step === "countries" && (
          <>
            <p className="text-xs text-muted-foreground">
              {t("countriesIntro", { max: WORLD_MAX_COUNTRIES })}
            </p>
            <div className="flex flex-wrap items-center gap-1">
              <AddCountrySelect
                countries={countryInfo}
                chosen={countries}
                onMap={mapOwners}
                disabled={countries.length >= WORLD_MAX_COUNTRIES}
                onAdd={addCountry}
              />
              <button
                type="button"
                className={BUTTON}
                disabled={countries.length >= WORLD_MAX_COUNTRIES}
                onClick={() => setAddOpen(true)}
              >
                {t("newCountry")}
              </button>
            </div>
            {longTags > 0 && (
              <p className="text-xs text-muted-foreground">
                {t("longTags", { count: longTags })}
              </p>
            )}
            {countries.length > 0 && (
              <ul className="space-y-1">
                {countries.map((tag, i) => (
                  <li key={tag} className="flex items-center gap-2 text-xs">
                    <span
                      className="h-3.5 w-3.5 shrink-0 rounded-sm border border-border"
                      style={swatch(colours[i])}
                      aria-hidden
                    />
                    <span className="min-w-0 flex-1 truncate">
                      {nameOf(tag)}
                    </span>
                    <span className="tabular-nums text-muted-foreground">
                      {shares.owned[i]
                        ? t("share", { share: percent(shares.owned[i]) })
                        : t("notPaintedShort")}
                    </span>
                    <button
                      type="button"
                      className="rounded px-1 text-muted-foreground hover:text-foreground"
                      aria-label={t("removeTitle", { name: nameOf(tag) })}
                      onClick={() => void dropCountry(i + 1)}
                    >
                      ×
                    </button>
                  </li>
                ))}
              </ul>
            )}
            {shares.land > 0 && (
              <p
                className={`text-xs ${shares.unowned ? "text-muted-foreground" : "text-success"}`}
              >
                {shares.unowned
                  ? t("unpainted", { share: percent(shares.unowned) })
                  : t("allPainted")}
              </p>
            )}
            {unpaintedCountries.length > 0 && (
              <p className="text-xs text-warning">
                {t("notPainted", {
                  names: new Intl.ListFormat(locale, {
                    type: "conjunction",
                  }).format(unpaintedCountries.map(nameOf)),
                })}
              </p>
            )}
            <button
              type="button"
              className={PRIMARY}
              onClick={() => setStep("make")}
            >
              {t("countriesNext")}
            </button>
          </>
        )}

        {step === "make" && (
          <>
            {notBase && (
              <div className="space-y-1 rounded-md border border-warning/50 bg-warning/10 p-2 text-xs">
                <p>{t("notBase")}</p>
                <Link
                  href={`/project/${projectId}/world-map`}
                  target={workbenchTargetOf(`/project/${projectId}/world-map`)}
                  className="text-accent hover:underline"
                >
                  {t("openWorkbench")}
                </Link>
              </div>
            )}
            <section className="space-y-2">
              <h3 className="text-xs font-medium">{t("sizeTitle")}</h3>
              <SegmentedControl
                label={t("sizeTitle")}
                size="sm"
                value={size}
                onChange={setSize}
                options={[
                  { value: "fine", label: t("sizeFine") },
                  { value: "medium", label: t("sizeMedium") },
                  { value: "coarse", label: t("sizeCoarse") },
                ]}
              />
              <p className="text-xs text-muted-foreground">{t("sizeHint")}</p>
              <div className="flex items-end gap-2">
                <label className="block flex-1 text-xs">
                  <span className="text-muted-foreground">{t("seed")}</span>
                  <input
                    className="mt-1 w-full rounded border border-input bg-background px-2 py-1 text-sm tabular-nums"
                    inputMode="numeric"
                    value={seed}
                    onChange={(e) =>
                      setSeed(Number(e.target.value.replace(/[^\d]/g, "")) || 0)
                    }
                  />
                </label>
                <button
                  type="button"
                  className={BUTTON}
                  disabled={countries.length === 0 || busy !== null}
                  onClick={() => void generate(false)}
                >
                  {preview ? t("regenerate") : t("generate")}
                </button>
              </div>
            </section>
            {busy === "preview" ? (
              <div className="animate-pulse-ring rounded-md border border-accent px-3 py-2 text-xs text-muted-foreground">
                {t("generating")}
              </div>
            ) : fresh ? (
              <section className="space-y-2">
                <SegmentedControl
                  label={t("showLabel")}
                  size="sm"
                  value={shown}
                  onChange={setShown}
                  options={[
                    { value: "sketch", label: t("showSketch") },
                    { value: "political", label: t("showPolitical") },
                    ...(fresh.images.terrain
                      ? [{ value: "terrain" as const, label: t("showTerrain") }]
                      : []),
                  ]}
                />
                <p className="text-xs">
                  {t("summary", {
                    land: fresh.data.summary.provinces.land,
                    sea: fresh.data.summary.provinces.sea,
                    states: fresh.data.summary.states,
                    regions: fresh.data.summary.regions,
                  })}
                </p>
                <ul className="space-y-0.5 text-xs text-muted-foreground">
                  {fresh.data.summary.countries.map((c) => (
                    <li key={c.tag}>
                      {t("countryStates", {
                        name: nameOf(c.tag),
                        states: c.states,
                      })}
                    </li>
                  ))}
                </ul>
                {findings.length > 0 && (
                  <ul className="space-y-1 text-xs">
                    {findings.map((finding) => {
                      const howTo =
                        finding.severity === "error"
                          ? mapFindingFix(finding.code, t)
                          : null;
                      return (
                        <li
                          key={finding.code}
                          className={
                            finding.severity === "error"
                              ? "text-destructive"
                              : "text-warning"
                          }
                        >
                          {mapFindingMessage(finding, t, locale)} ×
                          {finding.count}
                          {howTo && (
                            <p className="text-muted-foreground">{howTo}</p>
                          )}
                        </li>
                      );
                    })}
                  </ul>
                )}
                <div className="space-y-1">
                  <h3 className="text-xs font-medium">{t("unloadTitle")}</h3>
                  <ul className="grid grid-cols-2 gap-x-3 gap-y-0.5 text-xs text-muted-foreground">
                    {fresh.data.summary.unloaded.map((folder) => (
                      <li key={folder} title={folder}>
                        {tw(unloadedLabel(folder))}
                      </li>
                    ))}
                  </ul>
                </div>
                <label className="flex items-start gap-2 text-xs">
                  <input
                    type="checkbox"
                    className="ui-focus-ring mt-0.5"
                    checked={confirmed}
                    onChange={(e) => setConfirmed(e.target.checked)}
                  />
                  <span>{t("confirm")}</span>
                </label>
              </section>
            ) : (
              <p className="text-xs text-muted-foreground">
                {preview ? t("stale") : t("previewHint")}
              </p>
            )}
            {error && (
              <p role="alert" className="text-xs text-destructive">
                {error}
              </p>
            )}
            <div className="space-y-1 border-t border-border pt-3">
              <button
                type="button"
                className={`${PRIMARY} w-full py-1.5`}
                disabled={waitingFor !== null || busy !== null}
                onClick={() => void generate(true)}
              >
                {busy === "save" ? t("saving") : t("save")}
              </button>
              <p className="text-xs text-muted-foreground">
                {busy === "save" ? t("saving") : (waitingFor ?? t("saveNote"))}
              </p>
            </div>
          </>
        )}
      </div>
    </GamePanel>
  );

  return (
    <>
      <StudioCanvas
        sketch={sketch}
        brush={brush}
        look={look}
        overlay={overlay}
        version={version}
        fitRequest={fitRequest}
        paintable={step !== "make"}
        width={width}
        height={height}
        onStroke={(before, rect) => {
          record(patchOf(before, sketch.current, rect));
          changed();
        }}
      />
      {mounted &&
        createPortal(
          <div
            data-map-studio=""
            className="pointer-events-none fixed inset-0 z-[65]"
          >
            {tools}
            {panel}
            <AddCountryDialog
              projectId={projectId}
              open={addOpen}
              onOpenChange={setAddOpen}
              onAdded={(tag) => {
                void rows.mutate();
                addCountry(tag);
              }}
            />
          </div>,
          document.body,
        )}
    </>
  );
}
