"use client";

import { useMemo } from "react";
import type { GuiNode, GuiSprite, GuiWindow } from "@/lib/guiRenderApi";
import { GuiCanvas, assignIds } from "../gui-renderer/GuiCanvas";
import { useEditCanvas } from "./editContext";
import { matchNamePath } from "./gameNavigation";

const NO_HIDDEN = new Set<string>();

function idOf(node: GuiNode | undefined): string {
  return (node as (GuiNode & { _id?: string }) | undefined)?._id ?? "";
}

export function Filled({
  entry,
  left,
  top,
  width,
  height,
  texts,
  pictures,
  frames,
  hide,
  disabled,
  extraSprites,
}: {
  entry: GuiWindow;
  left: number;
  top: number;
  width: number;
  height: number;
  texts: Record<string, string>;
  pictures?: Record<string, string | undefined>;
  frames?: Record<string, number>;
  hide?: readonly string[];
  disabled?: readonly string[];
  extraSprites: Record<string, GuiSprite>;
}) {
  const edit = useEditCanvas();
  const node = useMemo(() => {
    const clone = structuredClone(entry.node) as GuiNode;
    const walk = (at: GuiNode) => {
      for (const child of at.children ?? []) {
        const picture = pictures?.[child.name ?? ""];
        if (picture && extraSprites[picture]) child.sprite = picture;
        const frame = frames?.[child.name ?? ""];
        if (frame !== undefined) child.frame = frame;
        walk(child);
      }
    };
    walk(clone);
    assignIds(clone);
    return clone;
  }, [entry.node, pictures, frames, extraSprites]);
  const sprites = useMemo(
    () => ({ ...entry.sprites, ...extraSprites }),
    [entry.sprites, extraSprites],
  );
  const overrides = useMemo(() => {
    const out: Record<string, string> = {};
    for (const [path, text] of Object.entries(texts))
      for (const hit of matchNamePath(node, path))
        if (idOf(hit)) out[idOf(hit)] = text;
    return out;
  }, [node, texts]);
  const hidden = useMemo(() => {
    if (!hide?.length) return NO_HIDDEN;
    return new Set(
      hide
        .flatMap((path) => matchNamePath(node, path).map(idOf))
        .filter(Boolean),
    );
  }, [node, hide]);
  const switchedOff = useMemo(() => {
    if (!disabled?.length) return undefined;
    return new Set(
      disabled
        .flatMap((path) => matchNamePath(node, path).map(idOf))
        .filter(Boolean),
    );
  }, [node, disabled]);
  return (
    <div style={{ position: "absolute", left, top }}>
      <GuiCanvas
        node={node}
        {...edit(entry.name, { ...entry, sprites }, node, sprites, overrides)}
        texts={entry.texts}
        fonts={entry.fonts}
        hidden={hidden}
        disabled={switchedOff}
        isolateId={null}
        transparent
        baseW={width}
        baseH={height}
      />
    </div>
  );
}
