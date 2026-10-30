"use client";

import { useMemo, useRef, useState } from "react";
import useSWR from "swr";
import {
  guiRenderApi,
  type GuiDesignerTerrain,
  type GuiNode,
  type GuiSprite,
  type GuiWindow,
} from "@/lib/guiRenderApi";
import { GuiCanvas, assignIds } from "../gui-renderer/GuiCanvas";
import { useTemplateWindow } from "./CountrySelectLayer";
import { useEditCanvas } from "./editContext";
import { SLIDER, SLIDER_INSET, SLIDER_W, Slider } from "./DesignerDropdowns";
import { GAME_SCROLLBAR } from "./PageScrollbar";

const ENTRY = "designer_adjuster_entry";

const SLOT = { w: 77, h: 72 };
const COLUMNS = 3;
const NO_HIDDEN: Set<string> = new Set();

export function terrainFrame(value: number): number {
  if (value > 1e-6) return 1;
  if (value < -1e-6) return 2;
  return 3;
}

export function terrainSprite(terrain: string): string {
  return `GFX_adjuster_${terrain}_bg`;
}

function Entry({
  entry,
  item,
  sprites,
  left,
  top,
}: {
  entry: GuiWindow;
  item: GuiDesignerTerrain;
  sprites: Record<string, GuiSprite>;
  left: number;
  top: number;
}) {
  const built = useMemo(() => {
    const node = structuredClone(entry.node) as GuiNode;
    for (const child of node.children ?? []) {
      if (child.name === "background")
        child.sprite = terrainSprite(item.terrain);
      if (child.name === "movement") child.frame = terrainFrame(item.movement);
      if (child.name === "attack") child.frame = terrainFrame(item.attack);
      if (child.name === "defense") child.frame = terrainFrame(item.defence);
    }
    assignIds(node);
    return node;
  }, [entry.node, item]);
  const all = useMemo(
    () => ({ ...entry.sprites, ...sprites }),
    [entry.sprites, sprites],
  );
  const edit = useEditCanvas();
  return (
    <div
      style={{ position: "absolute", left, top, width: SLOT.w, height: SLOT.h }}
    >
      <GuiCanvas
        node={built}
        {...edit(`${ENTRY}:${item.terrain}`, entry, built, all)}
        texts={entry.texts}
        fonts={entry.fonts}
        hidden={NO_HIDDEN}
        isolateId={null}
        transparent
        baseW={SLOT.w}
        baseH={SLOT.h}
      />
    </div>
  );
}

export function DesignerTerrain({
  projectId,
  rect,
  terrain,
}: {
  projectId: number;

  rect: { x: number; y: number; w: number; h: number } | undefined;
  terrain: readonly GuiDesignerTerrain[];
}) {
  const entry = useTemplateWindow(projectId, terrain.length ? ENTRY : null);
  const names = useMemo(
    () =>
      [
        ...new Set([
          ...terrain.map((item) => terrainSprite(item.terrain)),
          ...Object.values(SLIDER),
        ]),
      ].sort(),
    [terrain],
  );
  const sprites = useSWR<Record<string, GuiSprite>>(
    terrain.length
      ? ["game-view-sprites", projectId, "designer-terrain", names.join(",")]
      : null,
    () => {
      guiRenderApi.setProjectId(projectId);
      return guiRenderApi.sprites(names);
    },
    { revalidateOnFocus: false },
  );
  const scroller = useRef<HTMLDivElement>(null);
  const [scroll, setScroll] = useState({ top: 0, max: 0 });
  if (!rect || !terrain.length || !entry.data) return null;
  const all = sprites.data ?? {};
  const height = rect.h || 236;
  const rowsH = Math.ceil(terrain.length / COLUMNS) * SLOT.h;
  const scrolls = rowsH > height;
  const measure = () => {
    const el = scroller.current;
    if (el)
      setScroll({
        top: el.scrollTop,
        max: Math.max(0, el.scrollHeight - el.clientHeight),
      });
  };
  return (
    <div
      style={{
        position: "absolute",
        left: rect.x,
        top: rect.y,
        width: rect.w || 262,
        height,
      }}
    >
      <div
        ref={scroller}
        onScroll={measure}
        onPointerEnter={measure}
        style={{
          position: "absolute",
          inset: 0,
          overflowY: "auto",
          overflowX: "hidden",
          ...GAME_SCROLLBAR,
          pointerEvents: "auto",
        }}
      >
        <div style={{ position: "relative", height: rowsH }}>
          {terrain.map((item, at) => (
            <Entry
              key={item.terrain}
              entry={entry.data as GuiWindow}
              item={item}
              sprites={all}
              left={(at % COLUMNS) * SLOT.w}
              top={Math.floor(at / COLUMNS) * SLOT.h}
            />
          ))}
        </div>
      </div>
      {scrolls && (
        <div
          style={{
            position: "absolute",
            left: (rect.w || 262) - SLIDER_INSET - SLIDER_W,
            top: 0,
            width: SLIDER_W,
            height,
          }}
        >
          <Slider
            height={height}
            scroll={scroll}
            sprites={all}
            onStep={(dir) => {
              const el = scroller.current;
              if (!el) return;
              el.scrollTop += dir * SLOT.h;
              measure();
            }}
          />
        </div>
      )}
    </div>
  );
}
