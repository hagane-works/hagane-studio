"use client";

import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useTranslations } from "use-intl";
import { Plus, Minus, Maximize2, Map as MapIcon, Search } from "lucide-react";
import type { State } from "@/types";
import { useContentLocaleStore } from "@/lib/store/contentLocaleStore";
import { resolveDisplayName } from "@/lib/i18n/localizedName";
import { useVanillaStateIndex } from "@/lib/vanillaStateIndex";
import { useStatesGeometry } from "./useStatesGeometry";
import { Select } from "@/components/ui/select";
import { Input } from "@/components/ui/input";
import {
  buildColorContext,
  buildLegend,
  colorForState,
  ringsToPath,
  COLOR_VIEWS,
  RESOURCE_KEYS,
  NEUTRAL_COLOR,
  EDITED_CREATED_COLOR,
  MEMBERSHIP_CHANGED_COLOR,
  VANILLA_CONTENT_MISSING_COLOR,
  type ColorView,
  type ResourceKey,
} from "./mapColoring";

const MIN_SCALE = 0.5;
const MAX_SCALE = 60;

const DRAG_SLOP_PX = 4;

const BORDER_COLOR = "rgba(60,50,40,0.85)";
const BORDER_WIDTH = 0.75;

const HOVER_STROKE = "rgba(40,32,24,0.95)";

const MAX_HITS = 8;

interface Transform {
  scale: number;
  tx: number;
  ty: number;
}
const IDENTITY: Transform = { scale: 1, tx: 0, ty: 0 };

function clamp(v: number, lo: number, hi: number): number {
  return Math.max(lo, Math.min(hi, v));
}

function clientToViewBox(
  svg: SVGSVGElement,
  clientX: number,
  clientY: number,
): { x: number; y: number } | null {
  const ctm = svg.getScreenCTM();
  if (!ctm) return null;
  const p = new DOMPoint(clientX, clientY).matrixTransform(ctm.inverse());
  return { x: p.x, y: p.y };
}

interface Marquee {
  x0: number;
  y0: number;
  x1: number;
  y1: number;
}

const MULTI_SELECTED_FILL = "hsl(28, 88%, 52%)";
const MULTI_SELECTED_STROKE = "hsl(28, 90%, 28%)";

export interface StatesMapMultiSelect {
  selectedGeoIds: Set<number>;

  onToggleGeoId: (geoId: number) => void;
}

export interface StatesMapProps {
  projectId: number;

  states: State[];

  selectedId: number | null;

  onSelectState: (id: number) => void;

  onAdoptGeoId?: (geoId: number) => void;

  adoptingGeoId?: number | null;

  multiSelect?: StatesMapMultiSelect;
}

export function StatesMap({
  projectId,
  states,
  selectedId,
  onSelectState,
  onAdoptGeoId,
  adoptingGeoId,
  multiSelect,
}: StatesMapProps) {
  const t = useTranslations("states");
  const locale = useContentLocaleStore((s) => s.locale);
  const { geometry, source, available, loading, error } =
    useStatesGeometry(projectId);

  const [view, setView] = useState<ColorView>("political");
  const [resourceKey, setResourceKey] = useState<ResourceKey>("oil");
  const [transform, setTransform] = useState<Transform>(IDENTITY);

  const multi = multiSelect != null;
  const onToggleGeoId = multiSelect?.onToggleGeoId;
  const multiSelectedGeoIds = multiSelect?.selectedGeoIds;

  const pickerOnly = multi && states.length === 0;

  const [shiftHeld, setShiftHeld] = useState(false);
  const boxMode = multi && (pickerOnly || shiftHeld);
  const [marquee, setMarquee] = useState<Marquee | null>(null);
  const marqueeRef = useRef<Marquee | null>(null);

  const [hoveredGeoId, setHoveredGeoId] = useState<number | null>(null);

  const svgRef = useRef<SVGSVGElement | null>(null);
  const panRef = useRef<{ lastX: number; lastY: number } | null>(null);

  const draggedRef = useRef(false);

  const downRef = useRef<{ x: number; y: number } | null>(null);

  const capturedRef = useRef(false);

  const stateByGeoId = useMemo(() => {
    const m = new Map<number, State>();
    for (const s of states) m.set(s.state_id, s);
    return m;
  }, [states]);

  const pathData = useMemo(() => {
    if (!geometry) return [];
    return Object.entries(geometry.states).map(([sid, shape]) => ({
      geoId: Number(sid),
      d: ringsToPath(shape.rings),
    }));
  }, [geometry]);

  const vanillaStates = useVanillaStateIndex();
  const ctx = useMemo(
    () => buildColorContext(states, view, resourceKey, vanillaStates.content),
    [states, view, resourceKey, vanillaStates],
  );

  const fillByGeoId = useMemo(() => {
    const m = new Map<number, string>();
    for (const p of pathData) {
      if (multiSelectedGeoIds?.has(p.geoId)) {
        m.set(p.geoId, MULTI_SELECTED_FILL);
        continue;
      }
      const st = stateByGeoId.get(p.geoId);
      m.set(p.geoId, st ? colorForState(view, st, ctx) : NEUTRAL_COLOR);
    }
    return m;
  }, [pathData, stateByGeoId, view, ctx, multiSelectedGeoIds]);

  const centerByGeoId = useMemo(() => {
    const m = new Map<number, [number, number]>();
    if (!multi || !geometry) return m;
    for (const [sid, shape] of Object.entries(geometry.states)) {
      m.set(Number(sid), [shape.centroid[0] + 0.5, shape.centroid[1] + 0.5]);
    }
    return m;
  }, [multi, geometry]);

  const legend = useMemo(() => buildLegend(view, ctx), [view, ctx]);

  const [query, setQuery] = useState("");
  const [activeHit, setActiveHit] = useState(0);
  const searchRef = useRef<HTMLDivElement | null>(null);

  const hits = useMemo(() => {
    const needle = query.trim().toLowerCase();
    if (!needle || !geometry) return [];
    const out: {
      geoId: number;
      label: string;
      owner: string | null;
      inProject: boolean;
    }[] = [];
    for (const st of states) {
      if (out.length >= MAX_HITS) break;
      const dn = st.display_name ?? {};
      const dl = st.display_name_locs ?? {};
      const hay = [
        String(st.state_id),
        st.name ?? "",
        st.owner ?? "",
        dn["en"] ?? "",
        dn["zh"] ?? "",
        dl["en"] ?? "",
        dl["zh"] ?? "",
      ];
      if (!hay.some((h) => h.toLowerCase().includes(needle))) continue;
      if (!geometry.states[String(st.state_id)]) continue;
      out.push({
        geoId: st.state_id,
        label: resolveDisplayName(
          locale,
          st.display_name ?? st.display_name_locs,
          null,
          null,
          st.name ?? `#${st.state_id}`,
        ),
        owner: st.owner ?? null,
        inProject: true,
      });
    }

    if (/^\d+$/.test(needle)) {
      const gid = Number(needle);
      if (
        geometry.states[String(gid)] &&
        !out.some((h) => h.geoId === gid) &&
        out.length < MAX_HITS
      ) {
        out.push({
          geoId: gid,
          label: `#${gid}`,
          owner: null,
          inProject: false,
        });
      }
    }
    return out;
  }, [query, states, geometry, locale]);

  useEffect(() => {
    if (!query) return;
    const onDocDown = (e: PointerEvent) => {
      if (!searchRef.current?.contains(e.target as Node)) {
        setQuery("");
        setActiveHit(0);
      }
    };
    document.addEventListener("pointerdown", onDocDown);
    return () => document.removeEventListener("pointerdown", onDocDown);
  }, [query]);

  const selectedGeoId = useMemo(() => {
    if (selectedId == null) return null;
    const s = states.find((x) => x.id === selectedId);
    return s ? s.state_id : null;
  }, [states, selectedId]);

  const handlePathClick = useCallback(
    (geoId: number, additive: boolean) => {
      if (draggedRef.current) return;

      if (onToggleGeoId && (pickerOnly || additive)) {
        onToggleGeoId(geoId);
        return;
      }
      const st = stateByGeoId.get(geoId);
      if (st) {
        onSelectState(st.id);
        return;
      }

      onAdoptGeoId?.(geoId);
    },
    [onToggleGeoId, pickerOnly, stateByGeoId, onSelectState, onAdoptGeoId],
  );

  const baseLayer = useMemo(
    () => (
      <g>
        {pathData.map((p) => {
          const st = stateByGeoId.get(p.geoId);

          const membershipChanged =
            view === "membership" &&
            st?.membership_provenance?.status === "known" &&
            st.membership_provenance.membership_changed === true;
          const created = view !== "membership" && st?.source === "created";

          const stageSelected = multiSelectedGeoIds?.has(p.geoId) ?? false;

          const clickable = multi || st != null || onAdoptGeoId != null;
          return (
            <path
              key={p.geoId}
              d={p.d}
              fill={fillByGeoId.get(p.geoId) ?? NEUTRAL_COLOR}

              fillRule="evenodd"
              vectorEffect="non-scaling-stroke"
              strokeWidth={
                stageSelected || membershipChanged || created
                  ? 1.25
                  : BORDER_WIDTH
              }
              style={{
                stroke: stageSelected
                  ? MULTI_SELECTED_STROKE
                  : membershipChanged
                    ? "hsl(28, 90%, 28%)"
                    : created
                      ? "var(--accent)"
                      : BORDER_COLOR,
                cursor: clickable ? "pointer" : "default",
              }}

              data-geo-id={p.geoId}
              onClick={(e) => handlePathClick(p.geoId, e.ctrlKey || e.metaKey)}
              onPointerEnter={() => setHoveredGeoId(p.geoId)}
            />
          );
        })}
      </g>
    ),
    [
      pathData,
      fillByGeoId,
      stateByGeoId,
      view,
      handlePathClick,
      multiSelectedGeoIds,
      multi,
      onAdoptGeoId,
    ],
  );

  const selectedPath = useMemo(() => {
    if (selectedGeoId == null) return null;
    return pathData.find((p) => p.geoId === selectedGeoId) ?? null;
  }, [pathData, selectedGeoId]);

  const adoptingPath = useMemo(() => {
    if (adoptingGeoId == null) return null;
    return pathData.find((p) => p.geoId === adoptingGeoId) ?? null;
  }, [pathData, adoptingGeoId]);

  const hoveredPath = useMemo(() => {
    if (hoveredGeoId == null || hoveredGeoId === selectedGeoId) return null;
    return pathData.find((p) => p.geoId === hoveredGeoId) ?? null;
  }, [pathData, hoveredGeoId, selectedGeoId]);

  const hoverLabel = useMemo(() => {
    if (hoveredGeoId == null) return null;
    const st = stateByGeoId.get(hoveredGeoId);
    if (!st) {
      return { id: hoveredGeoId, name: null, owner: null, adopt: !multi };
    }
    return {
      id: hoveredGeoId,
      name: resolveDisplayName(
        locale,
        st.display_name ?? st.display_name_locs,
        null,
        null,
        st.name ?? `#${hoveredGeoId}`,
      ),
      owner: st.owner ?? null,
      adopt: false,
    };
  }, [hoveredGeoId, stateByGeoId, locale, multi]);

  useEffect(() => {
    const el = svgRef.current;
    if (!el) return;
    const onWheel = (e: WheelEvent) => {
      e.preventDefault();
      const svg = svgRef.current;
      if (!svg) return;
      const u = clientToViewBox(svg, e.clientX, e.clientY);
      if (!u) return;
      const factor = Math.exp(-e.deltaY * 0.0015);
      setTransform((prev) => {
        const next = clamp(prev.scale * factor, MIN_SCALE, MAX_SCALE);
        const k = next / prev.scale;
        return {
          scale: next,
          tx: u.x - k * (u.x - prev.tx),
          ty: u.y - k * (u.y - prev.ty),
        };
      });
    };
    el.addEventListener("wheel", onWheel, { passive: false });
    return () => el.removeEventListener("wheel", onWheel);
  }, [geometry]);

  const applyMarquee = useCallback(
    (mq: Marquee) => {
      if (!onToggleGeoId) return;
      const { scale, tx, ty } = transform;
      const gx0 = (Math.min(mq.x0, mq.x1) - tx) / scale;
      const gx1 = (Math.max(mq.x0, mq.x1) - tx) / scale;
      const gy0 = (Math.min(mq.y0, mq.y1) - ty) / scale;
      const gy1 = (Math.max(mq.y0, mq.y1) - ty) / scale;
      for (const [geoId, [cx, cy]] of centerByGeoId) {
        if (cx >= gx0 && cx <= gx1 && cy >= gy0 && cy <= gy1) {
          onToggleGeoId(geoId);
        }
      }
    },
    [onToggleGeoId, centerByGeoId, transform],
  );

  const onPointerDown = useCallback(
    (e: React.PointerEvent<SVGSVGElement>) => {
      if (e.button !== 0) return;
      const svg = svgRef.current;
      if (!svg) return;
      const u = clientToViewBox(svg, e.clientX, e.clientY);
      downRef.current = { x: e.clientX, y: e.clientY };
      draggedRef.current = false;
      capturedRef.current = false;

      if (multi && (pickerOnly || e.shiftKey)) {
        e.preventDefault();
        marqueeRef.current = u ? { x0: u.x, y0: u.y, x1: u.x, y1: u.y } : null;
        setMarquee(marqueeRef.current);
        return;
      }
      panRef.current = u ? { lastX: u.x, lastY: u.y } : null;
    },
    [multi, pickerOnly],
  );

  useEffect(() => {
    if (!multi || pickerOnly) return;
    const down = (e: KeyboardEvent) => {
      if (e.key === "Shift") setShiftHeld(true);
    };
    const up = (e: KeyboardEvent) => {
      if (e.key === "Shift") setShiftHeld(false);
    };

    const blur = () => setShiftHeld(false);
    window.addEventListener("keydown", down);
    window.addEventListener("keyup", up);
    window.addEventListener("blur", blur);
    return () => {
      window.removeEventListener("keydown", down);
      window.removeEventListener("keyup", up);
      window.removeEventListener("blur", blur);
    };
  }, [multi, pickerOnly]);

  const onPointerMove = useCallback((e: React.PointerEvent<SVGSVGElement>) => {
    const down = downRef.current;
    if (
      down &&
      (Math.abs(e.clientX - down.x) > DRAG_SLOP_PX ||
        Math.abs(e.clientY - down.y) > DRAG_SLOP_PX)
    ) {
      draggedRef.current = true;

      if (!capturedRef.current) {
        capturedRef.current = true;
        svgRef.current?.setPointerCapture?.(e.pointerId);
      }
    }

    const mq = marqueeRef.current;
    if (mq) {
      const svg = svgRef.current;
      if (!svg) return;
      const u = clientToViewBox(svg, e.clientX, e.clientY);
      if (!u) return;
      marqueeRef.current = { ...mq, x1: u.x, y1: u.y };
      setMarquee(marqueeRef.current);
      return;
    }
    if (!panRef.current) return;
    const svg = svgRef.current;
    if (!svg) return;
    const u = clientToViewBox(svg, e.clientX, e.clientY);
    if (!u) return;

    const dx = u.x - panRef.current.lastX;
    const dy = u.y - panRef.current.lastY;
    panRef.current = { lastX: u.x, lastY: u.y };
    setTransform((prev) => ({ ...prev, tx: prev.tx + dx, ty: prev.ty + dy }));
  }, []);

  const endPan = useCallback(
    (e: React.PointerEvent<SVGSVGElement>) => {
      const mq = marqueeRef.current;
      if (mq) {
        marqueeRef.current = null;
        setMarquee(null);

        if (e.type === "pointerup") applyMarquee(mq);
      }
      panRef.current = null;
      downRef.current = null;
      if (capturedRef.current) {
        capturedRef.current = false;
        svgRef.current?.releasePointerCapture?.(e.pointerId);
      }
    },
    [applyMarquee],
  );

  const zoomBy = useCallback(
    (mult: number) => {
      setTransform((prev) => {
        const next = clamp(prev.scale * mult, MIN_SCALE, MAX_SCALE);
        const k = next / prev.scale;
        const w = geometry?.width ?? 0;
        const h = geometry?.height ?? 0;

        const cx = w / 2;
        const cy = h / 2;
        return {
          scale: next,
          tx: cx - k * (cx - prev.tx),
          ty: cy - k * (cy - prev.ty),
        };
      });
    },
    [geometry],
  );

  const resetView = useCallback(() => setTransform(IDENTITY), []);

  const flyToGeoId = useCallback(
    (geoId: number) => {
      const shape = geometry?.states[String(geoId)];
      if (!shape || !geometry) return;
      const [bx0, by0, bx1, by1] = shape.bbox;
      const bw = Math.max(1, bx1 - bx0);
      const bh = Math.max(1, by1 - by0);

      const next = clamp(
        Math.min(geometry.width / (bw * 1.45), geometry.height / (bh * 1.45)),
        MIN_SCALE,
        MAX_SCALE,
      );
      setTransform({
        scale: next,
        tx: geometry.width / 2 - next * (bx0 + bw / 2),
        ty: geometry.height / 2 - next * (by0 + bh / 2),
      });
    },
    [geometry],
  );

  const commitHit = useCallback(
    (geoId: number) => {
      flyToGeoId(geoId);
      setQuery("");
      setActiveHit(0);

      draggedRef.current = false;

      handlePathClick(geoId, false);
    },
    [flyToGeoId, handlePathClick],
  );

  if (loading) {
    return (
      <div className="flex-1 grid place-items-center p-8">
        <p className="text-xs text-muted-foreground">{t("mapLoading")}</p>
      </div>
    );
  }
  if (error) {
    return (
      <div className="flex-1 grid place-items-center p-8">
        <div className="ui-panel p-4 max-w-sm space-y-1 text-center">
          <p className="text-xs font-medium">{t("mapError")}</p>
          <p className="text-xs text-muted-foreground font-mono break-all">
            {String(error.message ?? error)}
          </p>
        </div>
      </div>
    );
  }
  if (!available || !geometry) {
    return (
      <div className="flex-1 grid place-items-center p-8">
        <div className="ui-panel p-5 max-w-sm space-y-2 text-center">
          <MapIcon
            className="h-7 w-7 mx-auto text-muted-foreground/60"
            aria-hidden
          />
          <p className="text-sm font-medium text-foreground">
            {t("mapUnavailableTitle")}
          </p>
          <p className="text-xs text-muted-foreground">
            {source ? t("mapUnavailableFor", { source }) : t("mapUnavailable")}
          </p>
        </div>
      </div>
    );
  }

  const { width, height } = geometry;

  return (
    <div className="relative flex-1 min-h-0 overflow-hidden bg-muted/20">
      {pickerOnly ? null : (
        <div className="absolute z-10 top-2 left-2 flex flex-col gap-1.5 rounded-lg border border-border/60 bg-popover/95 backdrop-blur px-2.5 py-2 shadow-sm max-w-[15rem]">
          <div ref={searchRef} className="relative">
            <Search
              className="pointer-events-none absolute left-1.5 top-1/2 -translate-y-1/2 h-3 w-3 text-muted-foreground/60"
              aria-hidden
            />
            <Input
              value={query}
              onChange={(e) => {
                setQuery(e.target.value);
                setActiveHit(0);
              }}
              onKeyDown={(e) => {
                if (e.key === "Escape") {
                  setQuery("");
                  setActiveHit(0);
                  return;
                }
                if (!hits.length) return;
                if (e.key === "ArrowDown") {
                  e.preventDefault();
                  setActiveHit((i) => (i + 1) % hits.length);
                } else if (e.key === "ArrowUp") {
                  e.preventDefault();
                  setActiveHit((i) => (i - 1 + hits.length) % hits.length);
                } else if (e.key === "Enter") {
                  e.preventDefault();
                  commitHit(hits[Math.min(activeHit, hits.length - 1)].geoId);
                }
              }}
              placeholder={t("mapSearchPlaceholder")}
              aria-label={t("mapSearchPlaceholder")}
              className="h-6 pl-6 pr-1.5 text-xs focus:ring-1 focus:ring-ring ui-interactive"
            />
            {query.trim() !== "" && (
              <div className="absolute left-0 right-0 top-7 z-20 overflow-hidden rounded-md border border-border/60 bg-popover shadow-md">
                {hits.length === 0 ? (
                  <p className="px-2 py-1.5 text-micro text-muted-foreground">
                    {t("mapSearchNoHits")}
                  </p>
                ) : (
                  hits.map((h, i) => (
                    <button
                      key={h.geoId}
                      type="button"
                      onClick={() => commitHit(h.geoId)}
                      onPointerEnter={() => setActiveHit(i)}
                      className={`flex w-full items-center gap-1.5 px-2 py-1 text-left ui-interactive ${
                        i === activeHit ? "bg-accent/15" : "hover:bg-muted"
                      }`}
                    >
                      <span className="text-micro font-mono text-muted-foreground shrink-0">
                        {h.geoId}
                      </span>
                      <span className="truncate text-xs">
                        {h.inProject ? h.label : t("mapHoverNotInProject")}
                      </span>
                      {h.owner && (
                        <span className="ml-auto shrink-0 rounded bg-muted px-1 text-micro font-mono text-muted-foreground">
                          {h.owner}
                        </span>
                      )}
                    </button>
                  ))
                )}
              </div>
            )}
          </div>

          <div className="flex items-center gap-1.5">
            <label className="shrink-0 whitespace-nowrap text-micro uppercase tracking-wide font-semibold text-muted-foreground">
              {t("mapColoring")}
            </label>
            <Select
              value={view}
              onChange={(e) => setView(e.target.value as ColorView)}
              className="h-6 text-xs px-1 focus:outline-none focus:ring-1 focus:ring-ring ui-interactive"
              aria-label={t("mapColoring")}
            >
              {COLOR_VIEWS.map((v) => (
                <option key={v} value={v}>
                  {t(`mapView_${v}`)}
                </option>
              ))}
            </Select>
          </div>

          {view === "resources" && (
            <div className="flex items-center gap-1.5">
              <label className="shrink-0 whitespace-nowrap text-micro uppercase tracking-wide font-semibold text-muted-foreground">
                {t("mapResource")}
              </label>
              <Select
                value={resourceKey}
                onChange={(e) => setResourceKey(e.target.value as ResourceKey)}
                className="h-6 text-xs px-1 focus:outline-none focus:ring-1 focus:ring-ring ui-interactive font-mono"
                aria-label={t("mapResource")}
              >
                {RESOURCE_KEYS.map((r) => (
                  <option key={r} value={r}>
                    {r}
                  </option>
                ))}
              </Select>
            </div>
          )}

          <MapLegend legend={legend} />
          {view === "membership" && (
            <p className="max-w-[15rem] text-micro leading-snug text-muted-foreground">
              {t("mapMembershipStaticHint")}
            </p>
          )}

          {multi && !pickerOnly && (
            <p className="max-w-[15rem] border-t border-border/40 pt-1.5 text-micro font-medium leading-snug text-warning">
              {t("mapMultiSelectHint")}
            </p>
          )}
        </div>
      )}

      <div className="absolute z-10 bottom-2 left-2 flex flex-col rounded-md border border-border/60 bg-popover/95 backdrop-blur shadow-sm overflow-hidden">
        <button
          type="button"
          onClick={() => zoomBy(1.4)}
          title={t("mapZoomIn")}
          className="p-1.5 text-muted-foreground hover:text-foreground hover:bg-muted ui-interactive"
        >
          <Plus className="h-3.5 w-3.5" />
        </button>
        <button
          type="button"
          onClick={() => zoomBy(1 / 1.4)}
          title={t("mapZoomOut")}
          className="p-1.5 text-muted-foreground hover:text-foreground hover:bg-muted border-t border-border/60 ui-interactive"
        >
          <Minus className="h-3.5 w-3.5" />
        </button>
        <button
          type="button"
          onClick={resetView}
          title={t("mapResetView")}
          className="p-1.5 text-muted-foreground hover:text-foreground hover:bg-muted border-t border-border/60 ui-interactive"
        >
          <Maximize2 className="h-3.5 w-3.5" />
        </button>
      </div>

      <svg
        ref={svgRef}
        viewBox={`0 0 ${width} ${height}`}
        preserveAspectRatio="xMidYMid meet"
        className={
          boxMode
            ? "w-full h-full touch-none select-none cursor-crosshair"
            : "w-full h-full touch-none select-none cursor-grab active:cursor-grabbing"
        }
        onPointerDown={onPointerDown}
        onPointerMove={onPointerMove}
        onPointerUp={endPan}
        onPointerLeave={(e) => {
          endPan(e);
          setHoveredGeoId(null);
        }}
        onPointerCancel={endPan}
        role="img"
        aria-label={t("mapAriaLabel", { count: geometry.state_count })}
      >
        <g
          transform={`translate(${transform.tx},${transform.ty}) scale(${transform.scale})`}
        >
          {baseLayer}
          {adoptingPath && (
            <path
              d={adoptingPath.d}
              fill="var(--accent)"
              fillRule="evenodd"
              className="animate-pulse"
              opacity={0.45}
              pointerEvents="none"
            />
          )}
          {hoveredPath && (
            <path
              d={hoveredPath.d}
              fill="none"
              vectorEffect="non-scaling-stroke"
              strokeWidth={1.75}
              style={{ stroke: HOVER_STROKE }}
              pointerEvents="none"
            />
          )}
          {selectedPath && (
            <g pointerEvents="none">
              <path
                d={selectedPath.d}
                fill="var(--accent)"
                fillRule="evenodd"
                opacity={0.4}
              />
              <path
                d={selectedPath.d}
                fill="none"
                vectorEffect="non-scaling-stroke"
                strokeWidth={4}
                style={{ stroke: "var(--foreground)" }}
              />
              <path
                d={selectedPath.d}
                fill="none"
                vectorEffect="non-scaling-stroke"
                strokeWidth={1.75}
                style={{ stroke: "var(--accent)" }}
              />
            </g>
          )}
        </g>

        {marquee && (
          <rect
            x={Math.min(marquee.x0, marquee.x1)}
            y={Math.min(marquee.y0, marquee.y1)}
            width={Math.abs(marquee.x1 - marquee.x0)}
            height={Math.abs(marquee.y1 - marquee.y0)}
            fill="hsla(28, 90%, 50%, 0.12)"
            vectorEffect="non-scaling-stroke"
            strokeWidth={1.5}
            strokeDasharray="4 3"
            style={{ stroke: "var(--accent)" }}
            pointerEvents="none"
          />
        )}
      </svg>

      {hoverLabel && (
        <div className="pointer-events-none absolute z-10 bottom-2 left-1/2 -translate-x-1/2 flex items-center gap-2 rounded-md border border-border/60 bg-popover/95 backdrop-blur px-2.5 py-1 shadow-sm max-w-[22rem]">
          <span className="text-micro font-mono text-muted-foreground shrink-0">
            #{hoverLabel.id}
          </span>
          {hoverLabel.name ? (
            <span className="text-xs font-medium truncate">
              {hoverLabel.name}
            </span>
          ) : (
            <span className="text-xs text-muted-foreground italic truncate">
              {t("mapHoverNotInProject")}
            </span>
          )}
          {hoverLabel.owner && (
            <span className="text-micro font-mono px-1 rounded bg-muted text-muted-foreground shrink-0">
              {hoverLabel.owner}
            </span>
          )}
          {hoverLabel.adopt && (
            <span className="text-micro text-accent shrink-0">
              {t("mapHoverClickToAdd")}
            </span>
          )}
        </div>
      )}
    </div>
  );
}

function MapLegend({ legend }: { legend: ReturnType<typeof buildLegend> }) {
  const t = useTranslations("states");

  if (legend.kind === "political") {
    return (
      <div className="flex items-center gap-1.5 text-micro text-muted-foreground">
        <span
          className="inline-block h-2.5 w-2.5 rounded-sm"
          style={{ background: NEUTRAL_COLOR }}
        />
        <span>{t("mapLegendPolitical")}</span>
      </div>
    );
  }

  if (legend.kind === "edited") {
    return (
      <div className="flex flex-col gap-0.5 text-micro text-muted-foreground">
        <div className="flex items-center gap-1.5">
          <span
            className="inline-block h-2.5 w-2.5 rounded-sm"
            style={{ background: EDITED_CREATED_COLOR }}
          />
          <span>{t("mapLegendCreated")}</span>
        </div>
        <div className="flex items-center gap-1.5">
          <span
            className="inline-block h-2.5 w-2.5 rounded-sm"
            style={{ background: NEUTRAL_COLOR }}
          />
          <span>{t("mapLegendImported")}</span>
        </div>
      </div>
    );
  }

  if (legend.kind === "membership") {
    return (
      <div className="flex flex-col gap-0.5 text-micro text-muted-foreground">
        <div className="flex items-center gap-1.5">
          <span
            className="inline-block h-2.5 w-2.5 rounded-sm"
            style={{ background: MEMBERSHIP_CHANGED_COLOR }}
          />
          <span>
            {t("mapLegendMembershipChanged", { count: legend.changedCount })}
          </span>
        </div>
        <div className="flex items-center gap-1.5">
          <span
            className="inline-block h-2.5 w-2.5 rounded-sm"
            style={{ background: NEUTRAL_COLOR }}
          />
          <span>
            {t("mapLegendMembershipUnchanged", {
              count: legend.unchangedCount,
            })}
          </span>
        </div>
        <span>
          {t("mapLegendMembershipUnknown", { count: legend.unknownCount })}
        </span>
      </div>
    );
  }

  if (legend.kind === "vanillaContent") {
    return (
      <div className="flex flex-col gap-0.5 text-micro text-muted-foreground">
        <div className="flex items-center gap-1.5">
          <span
            className="inline-block h-2.5 w-2.5 rounded-sm"
            style={{ background: VANILLA_CONTENT_MISSING_COLOR }}
          />
          <span>
            {t("mapLegendVanillaContentMissing", {
              count: legend.missingCount,
            })}
          </span>
        </div>
        <div className="flex items-center gap-1.5">
          <span
            className="inline-block h-2.5 w-2.5 rounded-sm"
            style={{ background: NEUTRAL_COLOR }}
          />
          <span>{t("mapLegendVanillaContentOk")}</span>
        </div>
        <p className="max-w-[15rem] leading-snug">
          {t("mapVanillaContentHint")}
        </p>
      </div>
    );
  }

  if (legend.kind === "gradient") {
    return (
      <div className="flex flex-col gap-1">
        <div
          className="h-2.5 w-full rounded-sm"
          style={{
            background: `linear-gradient(to right, ${legend.from}, ${legend.to})`,
          }}
        />
        <div className="flex justify-between text-micro text-muted-foreground font-mono">
          <span>{legend.min}</span>
          <span>{legend.max}</span>
        </div>
      </div>
    );
  }

  return (
    <div className="flex flex-col gap-0.5 max-h-40 overflow-y-auto text-micro text-muted-foreground">
      {legend.entries.map((e) => (
        <div key={e.label} className="flex items-center gap-1.5">
          <span
            className="inline-block h-2.5 w-2.5 rounded-sm shrink-0"
            style={{ background: e.color }}
          />
          <span className="truncate font-mono" title={e.label}>
            {e.label}
          </span>
        </div>
      ))}
    </div>
  );
}
