"use client";

import { mapToolFetch, mapResponseProblem } from "@/lib/mapToolRequest";

import {
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
  useSyncExternalStore,
} from "react";
import { createPortal } from "react-dom";
import { useLocale, useTranslations } from "use-intl";
import useSWR, { mutate as globalMutate } from "swr";
import { toast } from "sonner";
import {
  Brush,
  Eraser,
  Flag,
  FolderInput,
  Mountain,
  PaintBucket,
  Pipette,
  Plus,
  Redo2,
  TreePalm,
  Undo2,
  Waves,
} from "lucide-react";
import { projectsApi } from "@/lib/api/project";
import { loadVanillaCountryTags } from "@/lib/vanillaCountryTags";
import { useConfirm } from "@/components/shared/ConfirmDialogProvider";
import { nameLanguages } from "@/components/workbench/world-map/mapNames";
import {
  WorldMapCanvas,
  type MapBrush,
  type MapFocus,
  type MapMarks,
} from "@/components/workbench/world-map/WorldMapCanvas";
import {
  pack,
  provinceNeighbours,
  TERRAIN_COLOURS,
  type WorldMapPayload,
} from "@/components/workbench/world-map/worldMapModel";
import { BUTTON, GamePanel, PRIMARY } from "../GamePanel";
import { useGameWorldMap } from "../GameWorldMap";
import {
  AddCountrySelect,
  swatch,
  TAG,
  useStudioCountries,
} from "./studioCountries";
import { MapHistory } from "./MapHistory";
import {
  balance,
  connectedArea,
  countryProvinces,
  freeTag,
  landOf,
  ownerOf,
  paintedMap,
  pendingChanges,
  restateProvinces,
  statesTouched,
  type Neighbours,
  type Overrides,
} from "./ownerPaint";
import { TERRAINS, type RGB } from "./studioSketch";
import {
  LAND_MAX_PIXELS,
  LAND_PROVINCE_PIXELS,
  landArea,
  landSketch,
  strokePixels,
  unreachedSeas,
} from "./landPaint";
import {
  MapProblemError,
  mapProblemMessage,
  problemProvinceIndexes,
} from "./mapProblems";
import {
  seaRegionSplits,
  statesAcrossRegions,
  statesWithoutLand,
} from "./seaRegions";
import { withGameLanguage } from "@/lib/guiRenderApi";
import { describeHover } from "./hoverLine";
import {
  TERRAIN_BRUSHES,
  terrainArea,
  terrainChanges,
  terrainCounts,
  terrainLand,
  terrainOf,
  terrainRequest,
  type TerrainBrush,
  type TerrainOverrides,
} from "./terrainPaint";
import {
  extendRiver,
  isRiver,
  parseRivers,
  RIVER_NONE,
  RIVER_SIZES,
  riverAt,
  riverChanges,
  riverColour,
  riverCounts,
  riversNear,
  riverTree,
  riverValues,
  startRiver,
  withTributaries,
  type RiverDraft,
  type RiverPayload,
  type RiverSize,
  type RiverTree,
} from "./riverPaint";

type Tool = "brush" | "eraser" | "bucket" | "picker";

type Mode = "countries" | "terrain" | "rivers" | "land";

type Layer = Mode | "lakes";

interface Step {
  layer: Layer;
  prev: Map<number, string | number | undefined>;
  next: Map<number, string | number>;
}

interface Saved {
  count: number;
  records: number;
  owners: Map<number, string>;
  terrains: Map<number, string>;
  rivers: boolean;
  land: boolean;

  lakes: number[];

  seas: number[];

  regions: boolean;
}

interface Fresh {
  name: string;
  color: RGB;
}

const UNDO_LIMIT = 100;

const LAND_MIN_RADIUS = 2;
const subscribeNothing = () => () => {};
const sleep = (ms: number) => new Promise((resolve) => setTimeout(resolve, ms));
const hex = (c: RGB) =>
  `#${c.map((v) => v.toString(16).padStart(2, "0")).join("")}`;
const fromHex = (h: string): RGB => [
  parseInt(h.slice(1, 3), 16),
  parseInt(h.slice(3, 5), 16),
  parseInt(h.slice(5, 7), 16),
];
const isBrush = (terrain: string): terrain is TerrainBrush =>
  (TERRAIN_BRUSHES as readonly string[]).includes(terrain);
const riverPack = (v: number) => {
  const [r, g, b] = riverColour(v);
  return pack(r, g, b);
};
const RIVER_SIZE_KEYS: RiverSize[] = ["stream", "river", "large"];

const LAND_CLASSES = TERRAINS.map((terrain, index) => ({
  ...terrain,
  index,
})).filter((terrain) => terrain.index > 0);
const NO_MARKS: ReadonlyMap<number, number> = new Map();

function freshColour(n: number): RGB {
  const hue = (n * 137.508 + 40) % 360;
  const c = 0.55;
  const x = c * (1 - Math.abs(((hue / 60) % 2) - 1));
  const [r, g, b] =
    hue < 60
      ? [c, x, 0]
      : hue < 120
        ? [x, c, 0]
        : hue < 180
          ? [0, c, x]
          : hue < 240
            ? [0, x, c]
            : hue < 300
              ? [x, 0, c]
              : [c, 0, x];
  const m = 0.25;
  return [
    Math.round((r + m) * 255),
    Math.round((g + m) * 255),
    Math.round((b + m) * 255),
  ];
}

export function MapEditor({
  projectId,
  date,
  width,
  height,
  mapVersion,
  onClose,
  onNewWorld,
  onImportMod,
  onMapChanged,
}: {
  projectId: number;

  date: string;

  width: number;
  height: number;

  mapVersion: number;
  onClose: () => void;

  onNewWorld: () => void;

  onImportMod: () => void;

  onMapChanged?: () => void;
}) {
  const t = useTranslations("mapStudio");
  const locale = useLocale();

  const compactLabels = /^(zh|ja|ko)\b/i.test(locale);
  const confirm = useConfirm();
  const mounted = useSyncExternalStore(
    subscribeNothing,
    () => true,
    () => false,
  );
  const map = useGameWorldMap(projectId, date);
  const project = useSWR(
    ["game-view-project", projectId],
    () => projectsApi.get(projectId),
    { revalidateOnFocus: false },
  );
  const countries = useStudioCountries(projectId, map?.colors);
  const { nameOf, colourOf, rows } = countries;
  const exportLanguages = project.data?.export_languages;
  const languages = useMemo(
    () => nameLanguages(exportLanguages),
    [exportLanguages],
  );

  const [mode, setMode] = useState<Mode>("countries");
  const [tool, setTool] = useState<Tool>("brush");
  const [terrainKey, setTerrainKey] = useState<TerrainBrush>("forest");
  const [radius, setRadius] = useState(25);

  const [chosen, setChosen] = useState<string[] | null>(null);
  const [selected, setSelected] = useState<string | null>(null);
  const [fresh, setFresh] = useState<Map<string, Fresh>>(() => new Map());
  const [renaming, setRenaming] = useState<string | null>(null);
  const [overrides, setOverrides] = useState<Overrides>(() => new Map());
  const [terrains, setTerrains] = useState<TerrainOverrides>(() => new Map());
  const [look, setLook] = useState(true);
  const [riversOpened, setRiversOpened] = useState(false);
  const [riverSize, setRiverSize] = useState<RiverSize>("river");
  const [riverEdits, setRiverEdits] = useState<ReadonlyMap<number, number>>(
    () => new Map(),
  );

  const [riverDraft, setRiverDraft] = useState<Map<number, number> | null>(
    null,
  );
  const [riverNote, setRiverNote] = useState<
    "blocked" | "start" | "short" | null
  >(null);
  const [landClass, setLandClass] = useState(1);
  const [landOwner, setLandOwner] = useState("");
  const [landEdits, setLandEdits] = useState<ReadonlyMap<number, number>>(
    () => new Map(),
  );

  const [landWay, setLandWay] = useState<"land" | "lake" | "sea">("land");
  const [lakeEdits, setLakeEdits] = useState<ReadonlyMap<number, string>>(
    () => new Map(),
  );
  const [history, setHistory] = useState({ undo: 0, redo: 0 });
  const [hovered, setHovered] = useState(0);
  const [touching, setTouching] = useState<Neighbours | null>(null);
  const [addCores, setAddCores] = useState(true);
  const [busy, setBusy] = useState<
    | "countries"
    | "save"
    | "terrain"
    | "rivers"
    | "land"
    | "lakes"
    | "seas"
    | "repair"
    | "refresh"
    | "undo"
    | null
  >(null);
  const [error, setError] = useState<string | null>(null);

  const [flagged, setFlagged] = useState<number[]>([]);
  const [focus, setFocus] = useState<MapFocus | null>(null);
  const [lastSave, setLastSave] = useState<Saved | null>(null);

  const projectTags = project.data?.country_tags;
  const palette = useMemo(
    () => chosen ?? (projectTags ?? []).filter((tag) => TAG.test(tag)),
    [chosen, projectTags],
  );
  const brushKey =
    selected && palette.includes(selected) ? selected : (palette[0] ?? null);

  const riversKey = ["world-map-rivers", projectId];
  const riversData = useSWR<RiverPayload>(
    riversOpened ? riversKey : null,
    async () => {
      const response = await mapToolFetch(
        withGameLanguage(
          `/api/go/projects/${projectId}/gui-render/world-map/rivers`,
        ),
      );
      if (!response.ok)
        throw new Error(
          mapProblemMessage(await mapResponseProblem(response), t, locale),
        );
      return response.json();
    },
    { revalidateOnFocus: false },
  );
  const riverMap = useMemo(
    () => (riversData.data ? parseRivers(riversData.data) : null),
    [riversData.data],
  );

  const overridesRef = useRef(new Map<number, string>());
  const terrainRef = useRef(new Map<number, string>());
  const riverRef = useRef(new Map<number, number>());
  const landRef = useRef(new Map<number, number>());
  const lakeRef = useRef(new Map<number, string>());
  const landWayRef = useRef(landWay);
  const landClassRef = useRef(landClass);
  const draftRef = useRef<RiverDraft | null>(null);
  const treeRef = useRef<RiverTree | null>(null);
  const indexRef = useRef<{ index: Uint16Array; width: number } | null>(null);

  const mapRefreshRef = useRef<{
    minimumVersion: number;
    finish: () => void;
  } | null>(null);
  const riverMapRef = useRef(riverMap);
  const riverSizeRef = useRef(riverSize);
  const radiusRef = useRef(radius);
  const undoStack = useRef<Step[]>([]);
  const redoStack = useRef<Step[]>([]);
  const stroke = useRef<Step | null>(null);
  const mapRef = useRef(map);
  const toolRef = useRef(tool);
  const modeRef = useRef(mode);
  const terrainKeyRef = useRef(terrainKey);
  const brushKeyRef = useRef(brushKey);
  const touchingRef = useRef(touching);
  const paletteRef = useRef(palette);
  const mods = useRef({ shift: false, alt: false });
  useEffect(() => {
    mapRef.current = map;
    toolRef.current = tool;
    modeRef.current = mode;
    terrainKeyRef.current = terrainKey;
    brushKeyRef.current = brushKey;
    touchingRef.current = touching;
    paletteRef.current = palette;
    riverMapRef.current = riverMap;
    riverSizeRef.current = riverSize;
    radiusRef.current =
      mode === "land" ? Math.max(LAND_MIN_RADIUS, radius) : radius;
    landClassRef.current = landClass;
    landWayRef.current = landWay;
  });
  useEffect(
    () => () => {
      mapRefreshRef.current?.finish();
    },
    [],
  );

  const labelOf = useCallback(
    (key: string) => fresh.get(key)?.name ?? nameOf(key),
    [fresh, nameOf],
  );
  const colourOfKey = useCallback(
    (key: string): RGB => fresh.get(key)?.color ?? colourOf(key),
    [fresh, colourOf],
  );
  const colours = useMemo(
    () =>
      Object.fromEntries(
        palette.map((key) => [
          key,
          [...colourOfKey(key)] as [number, number, number],
        ]),
      ),
    [palette, colourOfKey],
  );
  const shown = useMemo(() => {
    if (!map) return null;
    const painted = paintedMap(map, overrides, colours);
    const water = [...lakeEdits].filter(([, v]) => v);
    if (!terrains.size && !water.length) return painted;

    return {
      ...painted,
      provinceTerrains: new Map<number, string>([
        ...terrains,
        ...water.map(([i, v]): [number, string] => [
          i,
          v === "sea" ? "ocean" : "lakes",
        ]),
      ]),
    };
  }, [map, overrides, colours, terrains, lakeEdits]);
  const changes = useMemo(
    () => (map ? pendingChanges(map, overrides, touching) : []),
    [map, overrides, touching],
  );
  const erased = useMemo(
    () => [...overrides.values()].filter((tag) => tag === "").length,
    [overrides],
  );
  const onMap = useMemo(
    () => [
      ...new Set(
        [...(map?.states.values() ?? [])].map((s) => s.owner).filter(Boolean),
      ),
    ],
    [map],
  );
  const summary = useMemo(() => balance(changes), [changes]);
  const terrainList = useMemo(
    () => (map ? terrainChanges(map, terrains) : []),
    [map, terrains],
  );
  const terrainSummary = useMemo(
    () => terrainCounts(terrainList),
    [terrainList],
  );

  const total = useMemo(
    () =>
      new Set([...changes.map((c) => c.id), ...terrainList.map((c) => c.id)])
        .size,
    [changes, terrainList],
  );
  const riverList = useMemo(
    () => (riverMap ? riverChanges(riverMap, riverEdits) : []),
    [riverMap, riverEdits],
  );
  const riverSummary = useMemo(
    () =>
      riverMap ? riverCounts(riverMap, riverList) : { drawn: 0, erased: 0 },
    [riverMap, riverList],
  );
  const riverPending = riverList.length > 0;
  const landPixels = useMemo(() => landArea(landEdits), [landEdits]);
  const landPending = landPixels > 0;
  const lakeList = useMemo(
    () =>
      map
        ? [...lakeEdits]
            .filter(
              ([i, v]) => v === "lake" && map.provinces[i]?.kind === "land",
            )
            .map(([i]) => i)
        : [],
    [map, lakeEdits],
  );
  const lakePending = lakeList.length > 0;
  const seaList = useMemo(
    () =>
      map
        ? [...lakeEdits]
            .filter(
              ([i, v]) => v === "sea" && map.provinces[i]?.kind === "land",
            )
            .map(([i]) => i)
        : [],
    [map, lakeEdits],
  );
  const seaPending = seaList.length > 0;

  const drowned = useMemo(() => {
    if (!map || (!lakePending && !seaPending)) return [];
    const marked = new Map<number, number>();
    for (const i of [...lakeList, ...seaList]) {
      const state = map.provinces[i].state;
      if (state) marked.set(state, (marked.get(state) ?? 0) + 1);
    }
    const size = new Map<number, number>();
    for (const p of map.provinces)
      if (p.kind === "land" && marked.has(p.state))
        size.set(p.state, (size.get(p.state) ?? 0) + 1);
    return [...marked]
      .filter(([state, n]) => n >= (size.get(state) ?? 0))
      .map(([state]) => map.states.get(state)?.name ?? `#${state}`);
  }, [map, lakeList, lakePending, seaList, seaPending]);

  const landlocked = useMemo(
    () =>
      map && touching && seaPending
        ? unreachedSeas(map, touching, seaList)
        : [],
    [map, touching, seaList, seaPending],
  );

  const seaSplits = useMemo(
    () => (map && touching ? seaRegionSplits(map, touching) : []),
    [map, touching],
  );
  const crossStates = useMemo(
    () => (map ? statesAcrossRegions(map) : []),
    [map],
  );

  const landless = useMemo(() => (map ? statesWithoutLand(map) : []), [map]);

  const flaggedIndexes = useMemo(
    () =>
      map
        ? problemProvinceIndexes(map.provinces, [
            ...flagged,
            ...landlocked,
            ...seaSplits.flatMap((split) => split.provinces),
            ...landless.flatMap((state) => state.provinces),
          ])
        : [],
    [map, flagged, landlocked, seaSplits, landless],
  );
  const drawn = useMemo(
    () =>
      shown && flaggedIndexes.length
        ? { ...shown, provinceFlags: new Set(flaggedIndexes) }
        : shown,
    [shown, flaggedIndexes],
  );
  const unsaved =
    overrides.size +
    terrainList.length +
    (riverPending ? Math.max(1, riverSummary.drawn + riverSummary.erased) : 0) +
    (landPending ? 1 : 0) +
    lakeList.length +
    seaList.length;

  const riverBase = useMemo(
    () =>
      riverMap
        ? new Map([...riverMap.base].map(([p, v]) => [p, riverPack(v)]))
        : null,
    [riverMap],
  );
  const marks = useMemo<MapMarks | null>(() => {
    if (mode === "land") {
      const over = new Map<number, number>();
      for (const [p, v] of landEdits) {
        const look = TERRAINS[v]?.look;
        if (v > 0 && look) over.set(p, pack(look[0], look[1], look[2]));
      }
      return { base: NO_MARKS, over };
    }
    if (mode !== "rivers" || !riverBase) return null;
    const over = new Map<number, number>();
    for (const [p, v] of riverEdits) over.set(p, isRiver(v) ? riverPack(v) : 0);
    for (const [p, v] of riverDraft ?? []) over.set(p, riverPack(v));
    return { base: riverBase, over };
  }, [mode, riverBase, riverEdits, riverDraft, landEdits]);
  const terrainName = (terrain: string) =>
    isBrush(terrain) ? t(`terrain_${terrain}`) : terrain;

  const storeOf = (layer: Layer): Map<number, string | number> =>
    layer === "terrain"
      ? terrainRef.current
      : layer === "rivers"
        ? riverRef.current
        : layer === "land"
          ? landRef.current
          : layer === "lakes"
            ? lakeRef.current
            : overridesRef.current;
  const setIn = (
    layer: Layer,
    i: number,
    value: string | number | undefined,
  ) => {
    if (value === undefined) storeOf(layer).delete(i);
    else storeOf(layer).set(i, value);
  };
  const show = (layer: Layer) =>
    layer === "terrain"
      ? setTerrains(new Map(terrainRef.current))
      : layer === "rivers"
        ? setRiverEdits(new Map(riverRef.current))
        : layer === "land"
          ? setLandEdits(new Map(landRef.current))
          : layer === "lakes"
            ? setLakeEdits(new Map(lakeRef.current))
            : setOverrides(new Map(overridesRef.current));
  const commit = () => {
    setOverrides(new Map(overridesRef.current));
    setTerrains(new Map(terrainRef.current));
    setRiverEdits(new Map(riverRef.current));
    setLandEdits(new Map(landRef.current));
    setLakeEdits(new Map(lakeRef.current));
    setHistory({
      undo: undoStack.current.length,
      redo: redoStack.current.length,
    });
  };

  const give = (
    provinces: readonly number[],
    key: string | null,
    step: Step,
  ): boolean => {
    const base = mapRef.current;
    if (!base) return false;
    const terrain = step.layer === "terrain";
    const store = (terrain ? terrainRef.current : overridesRef.current) as Map<
      number,
      string
    >;
    let any = false;
    for (const i of terrain
      ? terrainLand(base, provinces)
      : landOf(base, provinces)) {
      const value = key ?? base.provinces[i].terrain;
      if (
        (terrain ? terrainOf(base, store, i) : ownerOf(base, store, i)) ===
        value
      )
        continue;
      if (!step.prev.has(i)) step.prev.set(i, store.get(i));
      step.next.set(i, value);
      store.set(i, value);
      any = true;
    }
    return any;
  };
  const record = (step: Step) => {
    if (step.next.size === 0) return;
    undoStack.current.push(step);
    if (undoStack.current.length > UNDO_LIMIT) undoStack.current.shift();
    redoStack.current = [];
  };
  const undo = () => {
    const step = undoStack.current.pop();
    if (!step) return;
    for (const [i, prev] of step.prev) setIn(step.layer, i, prev);
    redoStack.current.push(step);
    commit();
  };
  const redo = () => {
    const step = redoStack.current.pop();
    if (!step) return;
    for (const [i, next] of step.next) setIn(step.layer, i, next);
    undoStack.current.push(step);
    commit();
  };

  const addToPalette = (key: string) => {
    setChosen((list) => {
      const now = list ?? paletteRef.current;
      return now.includes(key) ? now : [...now, key];
    });
    setSelected(key);
  };
  const removeFromPalette = (key: string) => {
    setChosen(palette.filter((other) => other !== key));
    if (selected === key) setSelected(null);
  };

  const makeCountry = () => {
    let n = 1;
    while (fresh.has(`NEW${n}`)) n++;
    const key = `NEW${n}`;
    setFresh((all) =>
      new Map(all).set(key, {
        name: t("newCountryName", { n }),
        color: freshColour(n),
      }),
    );
    addToPalette(key);
    setMode("countries");
  };

  const pick = (index: number) => {
    const base = mapRef.current;
    if (!base || !index) return;
    if (modeRef.current === "terrain") {
      const terrain = terrainLand(base, [index]).length
        ? terrainOf(base, terrainRef.current, index)
        : "";
      if (isBrush(terrain)) setTerrainKey(terrain);
      return;
    }
    if (!landOf(base, [index]).length) return;
    const owner = ownerOf(base, overridesRef.current, index);
    if (owner) addToPalette(owner);
  };

  const ground = (p: number) => {
    const read = indexRef.current;
    const base = mapRef.current;
    if (!read || !base || p < 0 || p >= read.index.length) return false;
    const i = read.index[p];
    return i > 0 && base.provinces[i]?.kind === "land";
  };
  const showDraft = () => {
    const draft = draftRef.current;
    setRiverDraft(
      draft
        ? riverValues(draft, RIVER_SIZES[riverSizeRef.current], true)
        : null,
    );
  };

  const paintRiver = (
    phase: "start" | "move" | "end",
    line?: { from: [number, number]; to: [number, number] },
  ) => {
    const rivers = riverMapRef.current;
    const read = indexRef.current;
    if (!rivers || !read || read.width !== rivers.width) return;
    const edits = riverRef.current;
    const pixelOf = (point: [number, number]) =>
      Math.floor(point[1]) * rivers.width + Math.floor(point[0]);
    if (toolRef.current === "eraser") {
      if (phase === "start") {
        treeRef.current = riverTree(rivers, edits);
        stroke.current = { layer: "rivers", prev: new Map(), next: new Map() };
      }
      const step = stroke.current;
      const tree = treeRef.current;
      if (step && tree && line) {
        let any = false;
        const near = riversNear(
          tree,
          rivers.width,
          line.from[0],
          line.from[1],
          line.to[0],
          line.to[1],
          radiusRef.current,
        );
        for (const id of withTributaries(tree, near)) {
          for (const p of tree.pixels[id]) {
            if (riverAt(rivers, edits, p) === RIVER_NONE) continue;
            if (!step.prev.has(p)) step.prev.set(p, edits.get(p));
            step.next.set(p, RIVER_NONE);
            edits.set(p, RIVER_NONE);
            any = true;
          }
        }
        if (any) show("rivers");
      }
      if (phase === "end" && step) {
        stroke.current = null;
        treeRef.current = null;
        record(step);
        setHistory({
          undo: undoStack.current.length,
          redo: redoStack.current.length,
        });
      }
      return;
    }
    if (phase === "end") {
      const draft = draftRef.current;
      draftRef.current = null;
      setRiverDraft(null);
      if (!draft) return;
      const values = riverValues(draft, RIVER_SIZES[riverSizeRef.current]);
      if (values.size === 0) {
        setRiverNote("short");
        return;
      }
      const step: Step = { layer: "rivers", prev: new Map(), next: new Map() };
      for (const [p, v] of values) {
        step.prev.set(p, edits.get(p));
        step.next.set(p, v);
        edits.set(p, v);
      }
      record(step);
      commit();
      return;
    }
    if (!line) return;
    if (!draftRef.current) {
      draftRef.current = startRiver(rivers, edits, ground, pixelOf(line.to));
      setRiverNote(draftRef.current ? null : "start");
    } else {
      extendRiver(
        draftRef.current,
        rivers,
        edits,
        ground,
        line.to[0],
        line.to[1],
      );
      setRiverNote(draftRef.current.end === "blocked" ? "blocked" : null);
    }
    showDraft();
  };

  const paintLand = (
    provinces: number[],
    phase: "start" | "move" | "end",
    line?: { from: [number, number]; to: [number, number] },
  ) => {
    const read = indexRef.current;
    const base = mapRef.current;
    if (!read || !base) return;
    if (landWayRef.current !== "land") {
      if (phase === "start")
        stroke.current = { layer: "lakes", prev: new Map(), next: new Map() };
      const step = stroke.current;
      if (!step) return;
      const value = toolRef.current === "eraser" ? "" : landWayRef.current;
      const store = lakeRef.current;
      let any = false;
      for (const i of provinces) {
        if (
          base.provinces[i]?.kind !== "land" ||
          (store.get(i) ?? "") === value
        )
          continue;
        if (!step.prev.has(i)) step.prev.set(i, store.get(i));
        step.next.set(i, value);
        store.set(i, value);
        any = true;
      }
      if (any) show("lakes");
      if (phase === "end") {
        stroke.current = null;
        record(step);
        setHistory({
          undo: undoStack.current.length,
          redo: redoStack.current.length,
        });
      }
      return;
    }
    if (phase === "start")
      stroke.current = { layer: "land", prev: new Map(), next: new Map() };
    const step = stroke.current;
    if (!step) return;
    if (line) {
      const value = toolRef.current === "eraser" ? 0 : landClassRef.current;
      const store = landRef.current;
      let any = false;
      for (const p of strokePixels(
        read.width,
        read.index.length / read.width,
        line.from,
        line.to,
        radiusRef.current,
      )) {
        const i = read.index[p];
        if (
          !(i > 0 && base.provinces[i]?.kind === "sea") ||
          (store.get(p) ?? 0) === value
        )
          continue;
        if (!step.prev.has(p)) step.prev.set(p, store.get(p));
        step.next.set(p, value);
        store.set(p, value);
        any = true;
      }
      if (any) show("land");
    }
    if (phase === "end") {
      stroke.current = null;
      record(step);
      setHistory({
        undo: undoStack.current.length,
        redo: redoStack.current.length,
      });
    }
  };

  const onPaint = useCallback(
    (
      provinces: number[],
      phase: "start" | "move" | "end",
      at: number,
      line?: { from: [number, number]; to: [number, number] },
    ) => {
      const base = mapRef.current;
      if (!base) return;
      const current = toolRef.current;
      const layer = modeRef.current;
      if (layer === "rivers") {
        paintRiver(phase, line);
        return;
      }
      if (layer === "land") {
        paintLand(provinces, phase, line);
        return;
      }
      const terrainMode = layer === "terrain";
      const key = terrainMode ? terrainKeyRef.current : brushKeyRef.current;
      if (phase === "start") {
        stroke.current = null;
        if (mods.current.alt || current === "picker") {
          pick(at);
          if (current === "picker") setTool("brush");
          return;
        }
        const step: Step = { layer, prev: new Map(), next: new Map() };
        if (current === "bucket") {
          if (key && at && terrainMode) {
            give(
              touchingRef.current
                ? terrainArea(base, terrainRef.current, touchingRef.current, at)
                : [at],
              key,
              step,
            );
          } else if (key && at) {
            const owner = ownerOf(base, overridesRef.current, at);
            const area =
              mods.current.shift && owner
                ? countryProvinces(base, overridesRef.current, owner)
                : touchingRef.current
                  ? connectedArea(
                      base,
                      overridesRef.current,
                      touchingRef.current,
                      at,
                    )
                  : [at];
            give(area, key, step);
          }
          record(step);
          commit();
          return;
        }
        if (current === "eraser" && mods.current.shift && !terrainMode) {
          const owner = at ? ownerOf(base, overridesRef.current, at) : "";
          if (owner)
            give(countryProvinces(base, overridesRef.current, owner), "", step);
          record(step);
          commit();
          return;
        }
        if (current === "brush" && !key) return;
        stroke.current = step;
      }
      const step = stroke.current;
      if (!step) return;

      const paintWith = current === "eraser" ? (terrainMode ? null : "") : key;
      if (
        (paintWith !== null || current === "eraser") &&
        provinces.length &&
        give(provinces, paintWith, step)
      )
        show(step.layer);
      if (phase === "end") {
        stroke.current = null;
        record(step);
        setHistory({
          undo: undoStack.current.length,
          redo: redoStack.current.length,
        });
      }
    },
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [],
  );
  const strokes = tool === "brush" || tool === "eraser";

  const sized = strokes && !(mode === "rivers" && tool === "brush");
  const brush = useMemo<MapBrush>(
    () => ({
      radius: sized
        ? mode === "land"
          ? Math.max(LAND_MIN_RADIUS, radius)
          : radius
        : 1,
      onPaint,
    }),
    [sized, mode, radius, onPaint],
  );

  const worldMapKey = ["world-map", projectId, date];
  const refreshScreens = () =>
    globalMutate(
      (key) =>
        Array.isArray(key) &&
        typeof key[0] === "string" &&
        (key[0].startsWith("game-view") || key[0] === "world-map-state-rows"),
    );

  const refreshMapPicture = async () => {
    if (!onMapChanged) return;
    const minimumVersion = mapVersion + 1;
    indexRef.current = null;
    await new Promise<void>((resolve) => {
      const timeout: { id?: ReturnType<typeof setTimeout> } = {};
      const waiter: { minimumVersion: number; finish: () => void } = {
        minimumVersion,
        finish: () => {
          if (timeout.id !== undefined) clearTimeout(timeout.id);
          if (mapRefreshRef.current === waiter) mapRefreshRef.current = null;
          resolve();
        },
      };

      mapRefreshRef.current?.finish();
      mapRefreshRef.current = waiter;
      timeout.id = setTimeout(waiter.finish, 60_000);
      onMapChanged();
    });
  };

  const waitForOwners = async (expected: ReadonlyMap<number, string>) => {
    for (let tries = 0; tries < 6; tries++) {
      const payload = (await globalMutate(worldMapKey)) as
        WorldMapPayload | undefined;
      if (payload) {
        const stateOf = new Map(
          payload.provinces.map((row) => [row[0], row[6]]),
        );
        const owner = new Map(payload.states.map(([id, , tag]) => [id, tag]));
        if (
          [...expected].every(
            ([id, tag]) => owner.get(stateOf.get(id) ?? 0) === tag,
          )
        )
          return;
      }
      await sleep(1500);
    }
  };

  const waitForTerrains = async (expected: ReadonlyMap<number, string>) => {
    for (let tries = 0; tries < 6; tries++) {
      const payload = (await globalMutate(worldMapKey)) as
        WorldMapPayload | undefined;
      if (payload) {
        const terrainOfId = new Map(
          payload.provinces.map((row) => [row[0], row[4]]),
        );
        if (
          [...expected].every(
            ([id, terrain]) => terrainOfId.get(id) === terrain,
          )
        )
          return;
      }
      await sleep(1500);
    }
  };

  const waitForKinds = async (ids: readonly number[], kind: number) => {
    for (let tries = 0; tries < 6; tries++) {
      const payload = (await globalMutate(worldMapKey)) as
        WorldMapPayload | undefined;
      if (payload) {
        const kindOf = new Map(
          payload.provinces.map((row) => [row[0], row[2]]),
        );
        if (ids.every((id) => kindOf.get(id) === kind)) return;
      }
      await sleep(1500);
    }
  };

  const settle = (layer: Layer) => {
    storeOf(layer).clear();
    undoStack.current = undoStack.current.filter(
      (step) => step.layer !== layer,
    );
    redoStack.current = redoStack.current.filter(
      (step) => step.layer !== layer,
    );
    commit();
  };

  const showProblem = (err: unknown) => {
    if (!(err instanceof MapProblemError)) {
      setError(err instanceof Error ? err.message : String(err));
      return;
    }
    setError(mapProblemMessage(err.problem, t, locale));
    const ids = err.problem.provinces ?? [];
    setFlagged(ids);
    const first = map
      ? problemProvinceIndexes(map.provinces, ids)[0]
      : undefined;
    if (first) setFocus({ province: first, scale: 2, at: Date.now() });
  };

  const save = async () => {
    if (
      !map ||
      (total === 0 &&
        !riverPending &&
        !landPending &&
        !lakePending &&
        !seaPending) ||
      drowned.length ||
      landlocked.length
    )
      return;
    setError(null);
    setFlagged([]);
    const done: Saved = {
      count: total,
      records: 0,
      owners: new Map(),
      terrains: new Map(),
      rivers: false,
      land: false,
      lakes: [],
      seas: [],
      regions: false,
    };
    try {
      if (changes.length) await saveOwners();
      done.owners = new Map(changes.map((c) => [c.id, c.from]));
      if (changes.length) done.records++;
      if (terrainList.length) {
        setBusy("terrain");
        const response = await mapToolFetch(
          `/api/go/projects/${projectId}/gui-render/world-map/terrain`,
          {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({
              provinces: terrainRequest(terrainList),
              look,
            }),
          },
        );
        if (!response.ok)
          throw new MapProblemError(await mapResponseProblem(response));
        done.records++;
        done.terrains = new Map(terrainList.map((c) => [c.id, c.from]));
        setBusy("refresh");
        await waitForTerrains(new Map(terrainList.map((c) => [c.id, c.to])));
        settle("terrain");
      }
      if (riverPending) {
        setBusy("rivers");
        const response = await mapToolFetch(
          `/api/go/projects/${projectId}/gui-render/world-map/rivers`,
          {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({ pixels: riverList }),
          },
        );
        if (!response.ok)
          throw new MapProblemError(await mapResponseProblem(response));
        done.records++;
        done.rivers = true;
        setBusy("refresh");
        await riversData.mutate();
        settle("rivers");
      }
      if (landPending) await saveLand(done);

      const drowns = [
        {
          pending: lakePending,
          list: lakeList,
          busy: "lakes",
          route: "flood",
          kind: 2,
          saved: done.lakes,
        },
        {
          pending: seaPending,
          list: seaList,
          busy: "seas",
          route: "sink",
          kind: 1,
          saved: done.seas,
        },
      ] as const;
      for (const drown of drowns) {
        if (!drown.pending) continue;
        setBusy(drown.busy);
        const ids = drown.list.map((i) => map.provinces[i].id);
        const response = await mapToolFetch(
          `/api/go/projects/${projectId}/gui-render/world-map/${drown.route}`,
          {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({ date, provinces: ids }),
          },
        );
        if (!response.ok)
          throw new MapProblemError(await mapResponseProblem(response));
        done.records++;
        drown.saved.push(...ids);
        setBusy("refresh");
        await waitForKinds(ids, drown.kind);
      }
      if (lakePending || seaPending) {
        settle("lakes");
        await riversData.mutate();
      }
      await refreshScreens();
      toast.success(
        total
          ? t("savedChanges", { count: total })
          : landPending
            ? t("savedLand")
            : lakePending
              ? t("savedLakes")
              : seaPending
                ? t("savedSeas")
                : t("savedRivers"),
        { description: t("savedChangesBody") },
      );
    } catch (err) {
      showProblem(err);
    } finally {
      if (done.records) setLastSave(done);
      setBusy(null);
    }
  };

  const saveLand = async (done: Saved) => {
    const read = indexRef.current;
    if (!read) return;
    const sketch = landSketch(
      landRef.current,
      read.width,
      read.index.length / read.width,
    );
    if (!sketch) return;
    const [x, y, w, h] = sketch.rect;
    if (w * h > LAND_MAX_PIXELS) throw new Error(t("landTooBig"));
    setBusy("land");
    const canvas = document.createElement("canvas");
    canvas.width = w;
    canvas.height = h;
    canvas
      .getContext("2d")
      ?.putImageData(new ImageData(sketch.rgba, w, h), 0, 0);
    const png = await new Promise<Blob | null>((resolve) =>
      canvas.toBlob(resolve, "image/png"),
    );
    if (!png) throw new Error(t("landTooBig"));
    const body = new FormData();
    body.append("sketch", png, "land.png");
    for (const [name, value] of Object.entries({
      x,
      y,
      width: w,
      height: h,
      seed: Date.now() % 100000,
      land: LAND_PROVINCE_PIXELS,
    }))
      body.append(name, String(value));
    body.append("name", t("newLandName"));
    body.append("owner", landOwner);
    body.append("languages", languages.join(","));
    const response = await mapToolFetch(
      `/api/go/projects/${projectId}/gui-render/world-map/land`,
      { method: "POST", body },
    );
    if (!response.ok)
      throw new MapProblemError(await mapResponseProblem(response));
    done.records++;
    done.land = true;
    setBusy("refresh");

    await globalMutate(worldMapKey);
    await refreshMapPicture();
    await riversData.mutate();

    settle("land");
  };

  const saveOwners = async () => {
    if (!map) return;

    const used = [...new Set(changes.map((c) => c.to))].filter((key) =>
      fresh.has(key),
    );
    const tagOf = new Map<string, string>();
    if (used.length) {
      setBusy("countries");
      const taken = new Set<string>([
        ...(rows.data?.countries ?? []).map((row) => row.tag),
        ...(await loadVanillaCountryTags()).tags.map((c) => c.tag),
        ...onMap,
        ...Object.keys(map.colors),
      ]);
      for (const key of used) {
        const tag = freeTag(taken);
        taken.add(tag);
        const country = fresh.get(key)!;
        const response = await mapToolFetch(
          `/api/go/projects/${projectId}/countries`,
          {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({
              tag,
              name: country.name,
              ideology: "neutrality",
              color: country.color,
            }),
          },
        );
        if (!response.ok)
          throw new Error(
            t("countryFailed", {
              name: country.name,
              detail: mapProblemMessage(
                await mapResponseProblem(response),
                t,
                locale,
              ),
            }),
          );
        tagOf.set(key, tag);
      }
      await rows.mutate();
    }
    setBusy("save");
    const tagFor = (key: string) => tagOf.get(key) ?? key;
    const response = await mapToolFetch(
      `/api/go/projects/${projectId}/gui-render/world-map/restate`,
      {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          date,
          provinces: restateProvinces(changes, tagFor),
          add_cores: addCores,
          languages,
        }),
      },
    );
    if (!response.ok)
      throw new MapProblemError(await mapResponseProblem(response));

    if (tagOf.size) {
      setChosen(palette.map((key) => tagOf.get(key) ?? key));
      setSelected((key) => (key ? (tagOf.get(key) ?? key) : key));
      setFresh((all) => {
        const next = new Map(all);
        for (const key of tagOf.keys()) next.delete(key);
        return next;
      });
    }

    setBusy("refresh");
    await waitForOwners(new Map(changes.map((c) => [c.id, tagFor(c.to)])));
    settle("countries");
  };

  const undoSave = async () => {
    if (!lastSave) return;
    setBusy("undo");
    setError(null);
    setFlagged([]);
    try {
      for (let n = 0; n < lastSave.records; n++) {
        const response = await mapToolFetch(
          `/api/go/projects/${projectId}/gui-render/world-map/land?latest=1`,
          { method: "DELETE" },
        );
        if (!response.ok)
          throw new MapProblemError(await mapResponseProblem(response));
      }
      if (lastSave.lakes.length) await waitForKinds(lastSave.lakes, 0);
      if (lastSave.seas.length) await waitForKinds(lastSave.seas, 0);
      if (lastSave.land) {
        await globalMutate(worldMapKey);

        void refreshMapPicture();
      } else if (lastSave.regions) {
        await globalMutate(worldMapKey);
      }
      if (
        lastSave.rivers ||
        lastSave.land ||
        lastSave.lakes.length ||
        lastSave.seas.length
      )
        await riversData.mutate();
      if (lastSave.terrains.size) await waitForTerrains(lastSave.terrains);
      if (lastSave.owners.size) await waitForOwners(lastSave.owners);
      await refreshScreens();
      toast.success(
        lastSave.count
          ? t("undoneSave", { count: lastSave.count })
          : t("undoneSaveAll"),
      );
      setLastSave(null);
    } catch (err) {
      showProblem(err);
    } finally {
      setBusy(null);
    }
  };

  const repairMap = async () => {
    setBusy("repair");
    setError(null);
    setFlagged([]);
    try {
      const response = await mapToolFetch(
        `/api/go/projects/${projectId}/gui-render/world-map/repair`,
        {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ date, languages }),
        },
      );
      if (!response.ok)
        throw new MapProblemError(await mapResponseProblem(response));
      const { records } = (await response.json()) as { records?: number };
      setLastSave({
        count: 0,
        records: records ?? 1,
        owners: new Map(),
        terrains: new Map(),
        rivers: false,
        land: false,
        lakes: [],
        seas: [],
        regions: true,
      });
      await globalMutate(worldMapKey);
      await refreshScreens();
      toast.success(t("repairFixed"));
    } catch (err) {
      showProblem(err);
    } finally {
      setBusy(null);
    }
  };

  const recordsTakenBack = async () => {
    setLastSave(null);
    await globalMutate(
      (key) =>
        Array.isArray(key) &&
        typeof key[0] === "string" &&
        key[0].startsWith("world-map"),
    );
    await riversData.mutate();
    await refreshMapPicture();
    await refreshScreens();
  };

  const discard = async () => {
    if (
      !(await confirm({
        title: t("discardTitle", { count: unsaved }),
        body: t("discardBody"),
        danger: true,
        confirmLabel: t("discardConfirm"),
      }))
    )
      return;
    overridesRef.current = new Map();
    terrainRef.current = new Map();
    riverRef.current = new Map();
    landRef.current = new Map();
    lakeRef.current = new Map();
    undoStack.current = [];
    redoStack.current = [];
    setFlagged([]);
    setError(null);
    commit();
  };
  const close = async () => {
    const count = unsaved;
    if (
      count &&
      !(await confirm({
        title: t("closeTitle", { count }),
        body: t("closeBody"),
        danger: true,
        confirmLabel: t("closeConfirm"),
      }))
    )
      return;
    onClose();
  };

  const keys = useRef({ undo, redo, makeCountry });
  useEffect(() => {
    keys.current = { undo, redo, makeCountry };
  });
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      mods.current = { shift: e.shiftKey, alt: e.altKey };
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
      const key = e.key.toLowerCase();
      if (mod && key === "z") {
        e.preventDefault();
        if (e.shiftKey) keys.current.redo();
        else keys.current.undo();
      } else if (mod && key === "y") {
        e.preventDefault();
        keys.current.redo();
      } else if (!mod && !e.altKey) {
        if (key === "b") setTool("brush");
        else if (key === "e") setTool("eraser");
        else if (
          key === "g" &&
          modeRef.current !== "rivers" &&
          modeRef.current !== "land"
        )
          setTool("bucket");
        else if (
          key === "i" &&
          modeRef.current !== "rivers" &&
          modeRef.current !== "land"
        )
          setTool("picker");
        else if (key === "n") keys.current.makeCountry();
        else if (key === "[")
          setRadius((r) =>
            Math.max(
              modeRef.current === "land" ? LAND_MIN_RADIUS : 1,
              r <= 3 ? r - 1 : Math.round(r / 1.25),
            ),
          );
        else if (key === "]")
          setRadius((r) => Math.min(200, Math.round(r * 1.25) + 1));
      }

      if (e.key === "Alt") e.preventDefault();
    };
    const onUp = (e: KeyboardEvent) => {
      mods.current = { shift: e.shiftKey, alt: e.altKey };
    };
    const onBlur = () => {
      mods.current = { shift: false, alt: false };
    };
    window.addEventListener("keydown", onKey, true);
    window.addEventListener("keyup", onUp, true);
    window.addEventListener("blur", onBlur);
    return () => {
      window.removeEventListener("keydown", onKey, true);
      window.removeEventListener("keyup", onUp, true);
      window.removeEventListener("blur", onBlur);
    };
  }, []);

  const hoverLine = describeHover(map, hovered, mode, overrides, terrains, {
    say: (key, values) => t(key, values),
    ownerLabel: (key) => (key ? labelOf(key) : t("erasedLabel")),
    terrainName,
  });

  const terrainMode = mode === "terrain";
  const riverMode = mode === "rivers";
  const landMode = mode === "land";
  const allTools: {
    tool: Tool;
    icon: typeof Brush;
    label: string;
    key: string;
    hint: string;
  }[] = [
    {
      tool: "brush",
      icon: Brush,
      label: t("toolBrush"),
      key: "B",
      hint: landMode
        ? landWay === "lake"
          ? t("lakeBrushHint")
          : landWay === "sea"
            ? t("seaBrushHint")
            : t("landBrushHint")
        : riverMode
          ? t("riverBrushHint")
          : terrainMode
            ? t("terrainBrushHint")
            : t("toolBrushHint"),
    },
    {
      tool: "eraser",
      icon: Eraser,
      label: t("toolEraser"),
      key: "E",
      hint: landMode
        ? landWay === "lake"
          ? t("lakeEraserHint")
          : landWay === "sea"
            ? t("seaEraserHint")
            : t("landEraserHint")
        : riverMode
          ? t("riverEraserHint")
          : terrainMode
            ? t("terrainEraserHint")
            : t("toolEraserHint"),
    },
    {
      tool: "bucket",
      icon: PaintBucket,
      label: t("toolBucket"),
      key: "G",
      hint: terrainMode ? t("terrainBucketHint") : t("toolBucketHint"),
    },
    {
      tool: "picker",
      icon: Pipette,
      label: t("toolPicker"),
      key: "I",
      hint: terrainMode ? t("terrainPickerHint") : t("toolPickerHint"),
    },
  ];

  const lineTools = riverMode || landMode;
  const tools = lineTools
    ? allTools.filter((x) => x.tool === "brush" || x.tool === "eraser")
    : allTools;
  const modes: { mode: Mode; icon: typeof Brush; label: string }[] = [
    { mode: "countries", icon: Flag, label: t("modeCountries") },
    { mode: "terrain", icon: Mountain, label: t("modeTerrain") },
    { mode: "rivers", icon: Waves, label: t("modeRivers") },
    { mode: "land", icon: TreePalm, label: t("modeLand") },
  ];
  const chooseMode = (value: Mode) => {
    setMode(value);
    if (value === "rivers") setRiversOpened(true);
    if (
      (value === "rivers" || value === "land") &&
      (tool === "bucket" || tool === "picker")
    )
      setTool("brush");
  };

  const dock = (
    <div
      data-map-studio-tools=""
      className={`pointer-events-auto absolute left-3 top-16 max-h-[calc(100vh-8rem)] ${compactLabels ? "w-64" : "w-80"} space-y-3 overflow-y-auto rounded-md border border-border bg-card p-3 text-sm text-card-foreground shadow-lg`}
    >
      <div
        role="radiogroup"
        aria-label={t("modeTitle")}
        className="grid grid-cols-4 gap-1"
      >
        {modes.map(({ mode: value, icon: Icon, label }) => (
          <button
            key={value}
            type="button"
            role="radio"
            aria-checked={mode === value}
            onClick={() => chooseMode(value)}
            className={`ui-interactive ui-focus-ring flex flex-col items-center gap-1 rounded-md border px-1 py-1.5 text-center text-xs leading-tight ${mode === value ? "border-accent bg-accent/15 text-foreground" : "border-border text-muted-foreground hover:bg-muted"}`}
          >
            <Icon className="h-3.5 w-3.5" aria-hidden />
            {label}
          </button>
        ))}
      </div>
      <div
        role="radiogroup"
        aria-label={t("toolsTitle")}
        className={`grid gap-1 ${lineTools ? "grid-cols-2" : "grid-cols-4"}`}
      >
        {tools.map(({ tool: value, icon: Icon, label, key, hint }) => (
          <button
            key={value}
            type="button"
            role="radio"
            aria-checked={tool === value}
            aria-keyshortcuts={key}
            title={`${hint} (${key})`}
            onClick={() => setTool(value)}
            className={`ui-interactive ui-focus-ring flex flex-col items-center gap-1 rounded-md border px-1 py-2 text-center text-xs leading-tight ${tool === value ? "border-accent bg-accent/15 text-foreground" : "border-border text-muted-foreground hover:bg-muted"}`}
          >
            <Icon className="h-4 w-4" aria-hidden />
            {label}
          </button>
        ))}
      </div>
      <p className="text-xs text-muted-foreground">
        {tools.find((x) => x.tool === tool)?.hint}
      </p>
      {sized && (
        <label className="block text-xs">
          <span className="flex justify-between text-muted-foreground">
            {t("brushSize")}
            <span className="tabular-nums">
              {landMode ? Math.max(LAND_MIN_RADIUS, radius) : radius}
            </span>
          </span>
          <input
            type="range"
            min={landMode ? LAND_MIN_RADIUS : 1}
            max={200}
            value={landMode ? Math.max(LAND_MIN_RADIUS, radius) : radius}
            onChange={(e) => setRadius(Number(e.target.value))}
            className="mt-1 w-full accent-accent"
          />
          {landMode && radius <= LAND_MIN_RADIUS && (
            <span className="mt-1 block leading-5 text-warning">
              {t("landBrushSmall")}
            </span>
          )}
        </label>
      )}
      {landMode ? (
        <div className="space-y-2">
          <div
            role="radiogroup"
            aria-label={t("landWay")}
            className={`grid gap-1 ${compactLabels ? "grid-cols-3" : "grid-cols-1"}`}
          >
            {(["land", "lake", "sea"] as const).map((way) => (
              <button
                key={way}
                type="button"
                role="radio"
                aria-checked={landWay === way}
                onClick={() => setLandWay(way)}
                className={`ui-interactive ui-focus-ring min-h-7 rounded-md border px-1 py-1 text-xs leading-tight ${landWay === way ? "border-accent bg-accent/15" : "border-border hover:bg-muted"}`}
              >
                {way === "land"
                  ? t("landWayLand")
                  : way === "lake"
                    ? t("landWayLake")
                    : t("landWaySea")}
              </button>
            ))}
          </div>
          {landWay !== "land" ? (
            <p className="text-xs text-muted-foreground">
              {landWay === "sea" ? t("seaNote") : t("lakeNote")}
            </p>
          ) : (
            <>
              <div
                role="radiogroup"
                aria-label={t("landTerrain")}
                className="grid grid-cols-2 gap-1"
              >
                {LAND_CLASSES.map((terrain) => (
                  <button
                    key={terrain.key}
                    type="button"
                    role="radio"
                    aria-checked={landClass === terrain.index}
                    onClick={() => setLandClass(terrain.index)}
                    className={`ui-interactive ui-focus-ring flex h-7 items-center gap-1.5 rounded-md border px-2 text-left text-xs ${landClass === terrain.index ? "border-accent bg-accent/15" : "border-border hover:bg-muted"}`}
                  >
                    <span
                      className="h-3.5 w-3.5 shrink-0 rounded-sm border border-border"
                      style={swatch(terrain.look)}
                      aria-hidden
                    />
                    <span className="min-w-0 flex-1 truncate">
                      {t(`terrain_${terrain.key}`)}
                    </span>
                  </button>
                ))}
              </div>
              <label className="block text-xs">
                <span className="text-muted-foreground">{t("landOwner")}</span>
                <select
                  value={landOwner}
                  onChange={(e) => setLandOwner(e.target.value)}
                  className="ui-focus-ring mt-1 h-7 w-full rounded-md border border-input bg-card px-1 text-xs"
                >
                  <option value="">{t("landOwnerNone")}</option>
                  {palette
                    .filter((key) => !fresh.has(key))
                    .map((key) => (
                      <option key={key} value={key}>
                        {labelOf(key)}
                      </option>
                    ))}
                </select>
              </label>
            </>
          )}
        </div>
      ) : riverMode ? (
        <div className="space-y-2">
          {tool === "brush" && (
            <div
              role="radiogroup"
              aria-label={t("riverSize")}
              className="grid grid-cols-3 gap-1"
            >
              {RIVER_SIZE_KEYS.map((size) => (
                <button
                  key={size}
                  type="button"
                  role="radio"
                  aria-checked={riverSize === size}
                  onClick={() => setRiverSize(size)}
                  className={`ui-interactive ui-focus-ring min-h-7 rounded-md border px-1 py-1 text-xs leading-tight ${riverSize === size ? "border-accent bg-accent/15" : "border-border hover:bg-muted"}`}
                >
                  {t(`riverSize_${size}`)}
                </button>
              ))}
            </div>
          )}
          {riversData.error ? (
            <p role="alert" className="text-xs text-destructive">
              {t("riverUnavailable", {
                detail:
                  riversData.error instanceof Error
                    ? riversData.error.message
                    : String(riversData.error),
              })}
            </p>
          ) : !riverMap ? (
            <p className="text-xs text-muted-foreground">{t("riverLoading")}</p>
          ) : (
            riverNote && (
              <p className="text-xs text-warning">
                {t(`riverNote_${riverNote}`)}
              </p>
            )
          )}
        </div>
      ) : terrainMode ? (
        <div
          role="radiogroup"
          aria-label={t("terrainsTitle")}
          className="grid grid-cols-2 gap-1"
        >
          {TERRAIN_BRUSHES.map((terrain) => (
            <button
              key={terrain}
              type="button"
              role="radio"
              aria-checked={terrainKey === terrain}
              onClick={() => setTerrainKey(terrain)}
              className={`ui-interactive ui-focus-ring flex h-7 items-center gap-1.5 rounded-md border px-2 text-left text-xs ${terrainKey === terrain ? "border-accent bg-accent/15" : "border-border hover:bg-muted"}`}
            >
              <span
                className="h-3.5 w-3.5 shrink-0 rounded-sm border border-border"
                style={swatch(TERRAIN_COLOURS[terrain])}
                aria-hidden
              />
              <span className="min-w-0 flex-1 truncate">
                {t(`terrain_${terrain}`)}
              </span>
            </button>
          ))}
        </div>
      ) : (
        <>
          <div className="flex items-center justify-between">
            <span className="text-xs text-muted-foreground">
              {t("countriesTitle")}
            </span>
            <button
              type="button"
              className={`${BUTTON} flex items-center gap-1`}
              onClick={makeCountry}
              title={t("newCountryQuickHint")}
              aria-keyshortcuts="N"
            >
              <Plus className="h-3.5 w-3.5" aria-hidden />
              {t("newCountryQuick")}
            </button>
          </div>
          {palette.length === 0 ? (
            <p className="text-xs text-muted-foreground">{t("paletteEmpty")}</p>
          ) : (
            <div
              role="radiogroup"
              aria-label={t("countriesTitle")}
              className="max-h-56 space-y-1 overflow-y-auto"
            >
              {palette.map((key) => {
                const isFresh = fresh.has(key);
                return (
                  <div key={key} className="flex items-center gap-1">
                    {isFresh ? (
                      <label
                        className="relative h-6 w-6 shrink-0 cursor-pointer rounded-sm border border-border"
                        style={swatch(colourOfKey(key))}
                        title={t("colorHint")}
                      >
                        <input
                          type="color"
                          aria-label={t("colorHint")}
                          className="absolute inset-0 h-full w-full cursor-pointer opacity-0"
                          value={hex(colourOfKey(key))}
                          onChange={(e) => {
                            const color = fromHex(e.target.value);
                            setFresh((all) =>
                              new Map(all).set(key, {
                                ...all.get(key)!,
                                color,
                              }),
                            );
                          }}
                        />
                      </label>
                    ) : (
                      <span
                        className="h-6 w-6 shrink-0 rounded-sm border border-border"
                        style={swatch(colourOfKey(key))}
                        aria-hidden
                      />
                    )}
                    {renaming === key && isFresh ? (
                      <input
                        autoFocus
                        aria-label={t("renameHint")}
                        className="h-7 min-w-0 flex-1 rounded border border-input bg-background px-1 text-xs"
                        defaultValue={fresh.get(key)!.name}
                        onBlur={(e) => {
                          const name = e.target.value.trim();
                          if (name)
                            setFresh((all) =>
                              new Map(all).set(key, { ...all.get(key)!, name }),
                            );
                          setRenaming(null);
                        }}
                        onKeyDown={(e) => {
                          if (e.key === "Enter" || e.key === "Escape")
                            (e.target as HTMLInputElement).blur();
                        }}
                      />
                    ) : (
                      <button
                        type="button"
                        role="radio"
                        aria-checked={brushKey === key}
                        title={isFresh ? t("renameHint") : undefined}
                        onClick={() => setSelected(key)}
                        onDoubleClick={() => isFresh && setRenaming(key)}
                        className={`ui-interactive ui-focus-ring flex h-7 min-w-0 flex-1 items-center rounded-md border px-2 text-left text-xs ${brushKey === key ? "border-accent bg-accent/15" : "border-border hover:bg-muted"}`}
                      >
                        <span className="min-w-0 flex-1 truncate">
                          {labelOf(key)}
                        </span>
                        {isFresh && (
                          <span className="ml-1 shrink-0 text-muted-foreground">
                            {t("freshBadge")}
                          </span>
                        )}
                      </button>
                    )}
                    <button
                      type="button"
                      className="rounded px-1 text-muted-foreground hover:text-foreground"
                      aria-label={t("removeFromPalette", {
                        name: labelOf(key),
                      })}
                      onClick={() => removeFromPalette(key)}
                    >
                      ×
                    </button>
                  </div>
                );
              })}
            </div>
          )}
          <AddCountrySelect
            countries={countries}
            chosen={palette}
            onMap={onMap}
            onAdd={addToPalette}
          />
        </>
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
      </div>
      {hoverLine && <p className="text-xs">{hoverLine}</p>}
      <p className="text-xs text-muted-foreground">
        {landMode
          ? landWay !== "land"
            ? t("lakeHints")
            : t("landHints")
          : riverMode
            ? t("riverHints")
            : terrainMode
              ? t("terrainHints")
              : t("editHints")}
      </p>
    </div>
  );

  const panel = (
    <GamePanel
      title={t("title")}
      kind={t("editKind")}
      workbenchHref={`/project/${projectId}/world-map`}
      onClose={() => void close()}
    >
      <div className="mt-3 space-y-4 text-sm">
        <p className="text-xs text-muted-foreground">{t("editIntro")}</p>
        {(seaSplits.length > 0 ||
          crossStates.length > 0 ||
          landless.length > 0) &&
          map && (
            <section
              role="status"
              className="space-y-1.5 rounded-md border border-warning/40 bg-warning/10 p-2"
            >
              {seaSplits.length > 0 && (
                <p className="text-xs font-medium text-warning">
                  {t("seaSplitTitle", {
                    regions: new Intl.ListFormat(locale, {
                      type: "conjunction",
                    }).format(
                      seaSplits.map(
                        (split) =>
                          map.regions.get(split.region) || `#${split.region}`,
                      ),
                    ),
                  })}
                </p>
              )}
              {crossStates.length > 0 && (
                <p className="text-xs font-medium text-warning">
                  {t("stateSplitTitle", {
                    states: new Intl.ListFormat(locale, {
                      type: "conjunction",
                    }).format(
                      crossStates.map(
                        (state) => map.states.get(state)?.name || `#${state}`,
                      ),
                    ),
                  })}
                </p>
              )}
              {landless.length > 0 && (
                <div className="space-y-1">
                  <p className="text-xs font-medium text-warning">
                    {t("landlessTitle", { count: landless.length })}
                  </p>
                  <div className="flex flex-wrap gap-1">
                    {landless.map(({ state, provinces }) => (
                      <button
                        key={state}
                        type="button"
                        className="rounded border border-warning/40 px-1.5 py-0.5 text-xs text-warning hover:bg-warning/20"
                        title={t("landlessShow")}
                        onClick={() => {
                          const first = problemProvinceIndexes(
                            map.provinces,
                            provinces,
                          )[0];
                          if (first)
                            setFocus({
                              province: first,
                              scale: 2,
                              at: Date.now(),
                            });
                        }}
                      >
                        {map.states.get(state)?.name || `#${state}`}
                      </button>
                    ))}
                  </div>
                </div>
              )}
              <p className="text-xs text-muted-foreground">{t("repairBody")}</p>
              <button
                type="button"
                className={BUTTON}
                disabled={busy !== null}
                onClick={() => void repairMap()}
              >
                {busy === "repair" ? t("repairFixing") : t("repairFix")}
              </button>
            </section>
          )}
        <section className="space-y-2">
          <h3 className="text-xs font-medium">{t("changesTitle")}</h3>
          {unsaved === 0 && (
            <p className="text-xs text-muted-foreground">{t("changesNone")}</p>
          )}
          {overrides.size > 0 && (
            <>
              <p className="text-xs">
                {t("changesCount", {
                  count: changes.length,
                  states: map ? statesTouched(map, changes) : 0,
                })}
              </p>
              {erased > 0 && (
                <p className="text-xs text-muted-foreground">
                  {t("erasedNote", { count: erased })}
                </p>
              )}
              <ul className="max-h-40 space-y-0.5 overflow-y-auto text-xs text-muted-foreground">
                {summary.map(([key, { gain, loss }]) => (
                  <li key={key} className="flex items-center gap-2">
                    <span
                      className="h-3 w-3 shrink-0 rounded-sm border border-border"
                      style={swatch(colourOfKey(key))}
                      aria-hidden
                    />
                    <span className="min-w-0 flex-1 truncate">
                      {labelOf(key)}
                    </span>
                    <span className="tabular-nums">
                      {t("changeLine", { gain, loss })}
                    </span>
                  </li>
                ))}
              </ul>
              <label
                className="flex items-start gap-2 text-xs"
                title={t("addCoresHint")}
              >
                <input
                  type="checkbox"
                  className="ui-focus-ring mt-0.5"
                  checked={addCores}
                  onChange={(e) => setAddCores(e.target.checked)}
                />
                <span>{t("addCores")}</span>
              </label>
            </>
          )}
          {riverPending && (
            <p className="text-xs">{t("riverChangesCount", riverSummary)}</p>
          )}
          {landPending && (
            <p className="text-xs">
              {t("landChangesCount", {
                provinces: Math.max(
                  1,
                  Math.round(landPixels / LAND_PROVINCE_PIXELS),
                ),
              })}
            </p>
          )}
          {lakePending && (
            <p className="text-xs">
              {t("lakeChangesCount", { count: lakeList.length })}
            </p>
          )}
          {seaPending && (
            <p className="text-xs">
              {t("seaChangesCount", { count: seaList.length })}
            </p>
          )}
          {landlocked.slice(0, 3).map((id) => (
            <p key={id} className="text-xs text-destructive">
              {t("seaLandlocked", { id })}
            </p>
          ))}
          {landlocked.length > 3 && (
            <p className="text-xs text-destructive">
              {t("seaLandlockedMore", { count: landlocked.length - 3 })}
            </p>
          )}
          {drowned.slice(0, 3).map((state) => (
            <p key={state} className="text-xs text-destructive">
              {t("lakeWholeState", { state })}
            </p>
          ))}
          {drowned.length > 3 && (
            <p className="text-xs text-destructive">
              {t("lakeWholeMore", { count: drowned.length - 3 })}
            </p>
          )}
          {terrainList.length > 0 && (
            <>
              <p className="text-xs">
                {t("terrainChangesCount", { count: terrainList.length })}
              </p>
              <ul className="space-y-0.5 text-xs text-muted-foreground">
                {terrainSummary.map(([terrain, count]) => (
                  <li key={terrain} className="flex items-center gap-2">
                    <span
                      className="h-3 w-3 shrink-0 rounded-sm border border-border"
                      style={swatch(
                        TERRAIN_COLOURS[terrain] ?? [160, 156, 140],
                      )}
                      aria-hidden
                    />
                    <span className="min-w-0 flex-1 truncate">
                      {terrainName(terrain)}
                    </span>
                    <span className="tabular-nums">{count}</span>
                  </li>
                ))}
              </ul>
              <label
                className="flex items-start gap-2 text-xs"
                title={t("terrainLookHint")}
              >
                <input
                  type="checkbox"
                  className="ui-focus-ring mt-0.5"
                  checked={look}
                  onChange={(e) => setLook(e.target.checked)}
                />
                <span>{t("terrainLook")}</span>
              </label>
            </>
          )}
          {error && (
            <p role="alert" className="text-xs text-destructive">
              {error}
            </p>
          )}
          {flaggedIndexes.length > 0 && (
            <p className="text-xs text-muted-foreground">
              {t("problem.marked")}
            </p>
          )}
          {lastSave && unsaved === 0 && (
            <button
              type="button"
              className={BUTTON}
              disabled={busy !== null}
              onClick={() => void undoSave()}
            >
              {busy === "undo"
                ? t("undoingSave")
                : lastSave.count
                  ? t("undoSave", { count: lastSave.count })
                  : t("undoSaveAll")}
            </button>
          )}
          <div className="flex gap-1">
            <button
              type="button"
              className={`${PRIMARY} flex-1 py-1.5`}
              disabled={
                (total === 0 &&
                  !riverPending &&
                  !landPending &&
                  !lakePending &&
                  !seaPending) ||
                drowned.length > 0 ||
                landlocked.length > 0 ||
                busy !== null ||
                (changes.length > 0 && !touching)
              }
              onClick={() => void save()}
            >
              {busy === "countries"
                ? t("creatingCountries")
                : busy === "save"
                  ? t("savingChanges")
                  : busy === "terrain"
                    ? t("savingTerrain")
                    : busy === "rivers"
                      ? t("savingRivers")
                      : busy === "land"
                        ? t("savingLand")
                        : busy === "lakes"
                          ? t("savingLakes")
                          : busy === "seas"
                            ? t("savingSeas")
                            : busy === "refresh"
                              ? t("refreshing")
                              : riverPending ||
                                  landPending ||
                                  lakePending ||
                                  seaPending
                                ? t("saveAll")
                                : total
                                  ? t("saveChanges", { count: total })
                                  : t("saveNone")}
            </button>
            <button
              type="button"
              className={BUTTON}
              disabled={unsaved === 0 || busy !== null}
              onClick={() => void discard()}
            >
              {t("discard")}
            </button>
          </div>
        </section>
        <button
          type="button"
          className={`${BUTTON} ui-focus-ring flex w-full items-center justify-center gap-2 py-2 font-medium`}
          onClick={onImportMod}
          title={t("modLink")}
        >
          <FolderInput className="h-4 w-4 shrink-0" aria-hidden />
          {t("modTitle")}
        </button>
        <MapHistory
          projectId={projectId}
          disabled={busy !== null || unsaved > 0}
          blockedReason={unsaved > 0 ? t("history.saveFirst") : undefined}
          onChanged={recordsTakenBack}
        />
        <div className="flex flex-wrap gap-x-4 gap-y-1">
          <button
            type="button"
            className="text-xs text-accent hover:underline"
            onClick={onNewWorld}
          >
            {t("newWorldLink")}
          </button>
        </div>
      </div>
    </GamePanel>
  );

  return (
    <>
      <div
        data-map-studio-canvas=""
        style={{ position: "absolute", left: 0, top: 0, width, height }}
      >
        {drawn && (
          <WorldMapCanvas
            map={drawn}
            focus={focus}
            imageUrl={`/api/go/projects/${projectId}/gui-render/world-map/provinces.png?v=${mapVersion}`}
            layer={
              terrainMode || riverMode || landMode ? "terrain" : "political"
            }
            selected={0}
            onPick={() => {}}
            onHover={setHovered}
            loadingText={t("loadingMap")}
            brush={brush}
            onIndex={(index, w) => {
              const waiting = mapRefreshRef.current;

              const stale = Boolean(
                waiting && mapVersion < waiting.minimumVersion,
              );
              if (stale) return;
              indexRef.current = { index, width: w };
              setTouching(provinceNeighbours(index, w));
              if (waiting && mapVersion >= waiting.minimumVersion)
                waiting.finish();
            }}
            marks={marks}
          />
        )}
      </div>
      {mounted &&
        createPortal(
          <div
            data-map-studio=""
            className="pointer-events-none fixed inset-0 z-[65]"
          >
            {dock}
            {panel}
          </div>,
          document.body,
        )}
    </>
  );
}
