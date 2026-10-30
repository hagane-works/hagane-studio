"use client";

import { Panel, useViewport } from "@xyflow/react";

import {
  GRID_CELL_HEIGHT,
  GRID_CELL_WIDTH,
  GRID_OFFSET_X,
  GRID_OFFSET_Y,
  NODE_HEIGHT_NOMINAL,
  NODE_WIDTH,
} from "@/lib/focusTreeGraph";

const LABEL_OFFSET_X = GRID_OFFSET_X + NODE_WIDTH / 2;
const LABEL_OFFSET_Y = GRID_OFFSET_Y + NODE_HEIGHT_NOMINAL / 2;

const RULER_THICKNESS = 24;
const MIN_TICK_SPACING_PX = 22;
const VISIBLE_TICK_SPAN_X = 4200;
const VISIBLE_TICK_SPAN_Y = 2400;

function pickStep(cellPx: number) {
  if (cellPx >= MIN_TICK_SPACING_PX) return 1;
  return Math.ceil(MIN_TICK_SPACING_PX / cellPx);
}

function buildTickRange(
  cellPx: number,
  viewportPx: number,
  offsetPxScaled: number,
  spanPx: number,
) {
  const step = pickStep(cellPx);
  const startHoi4 = Math.floor((-viewportPx - offsetPxScaled) / cellPx) - 1;
  const endHoi4 =
    Math.ceil((spanPx - viewportPx - offsetPxScaled) / cellPx) + 1;
  const ticks: number[] = [];
  const first = Math.ceil(startHoi4 / step) * step;
  for (let h = first; h <= endHoi4; h += step) ticks.push(h);
  return ticks;
}

export function FocusGraphRuler() {
  const { x: vx, y: vy, zoom } = useViewport();

  const cellPxX = GRID_CELL_WIDTH * zoom;
  const cellPxY = GRID_CELL_HEIGHT * zoom;
  const offsetPxX = LABEL_OFFSET_X * zoom;
  const offsetPxY = LABEL_OFFSET_Y * zoom;

  const xTicks = buildTickRange(cellPxX, vx, offsetPxX, VISIBLE_TICK_SPAN_X);
  const yTicks = buildTickRange(cellPxY, vy, offsetPxY, VISIBLE_TICK_SPAN_Y);

  return (
    <>
      <Panel
        position="top-left"
        className="m-0! p-0! pointer-events-none left-0! right-0! w-full! translate-x-0!"
      >
        <div
          className="relative bg-card/85 border-b border-border backdrop-blur-sm"
          style={{ height: RULER_THICKNESS, paddingLeft: RULER_THICKNESS }}
        >
          {xTicks.map((hx) => {
            const screenX = hx * cellPxX + offsetPxX + vx;
            if (screenX < RULER_THICKNESS - 10 || screenX > VISIBLE_TICK_SPAN_X)
              return null;
            return (
              <span
                key={hx}
                className="absolute top-1/2 -translate-x-1/2 -translate-y-1/2 text-micro text-muted-foreground tech-tabular select-none"
                style={{ left: screenX }}
              >
                {hx}
              </span>
            );
          })}
        </div>
      </Panel>
      <Panel
        position="top-left"
        className="m-0! p-0! pointer-events-none top-0! bottom-0! h-full! translate-y-0!"
      >
        <div
          className="relative bg-card/85 border-r border-border backdrop-blur-sm"
          style={{
            width: RULER_THICKNESS,
            paddingTop: RULER_THICKNESS,
            height: "100%",
          }}
        >
          {yTicks.map((hy) => {
            const screenY = hy * cellPxY + offsetPxY + vy;
            if (screenY < RULER_THICKNESS - 10 || screenY > VISIBLE_TICK_SPAN_Y)
              return null;
            return (
              <span
                key={hy}
                className="absolute left-1/2 -translate-x-1/2 -translate-y-1/2 text-micro text-muted-foreground tech-tabular select-none"
                style={{ top: screenY }}
              >
                {hy}
              </span>
            );
          })}
        </div>
      </Panel>
    </>
  );
}
