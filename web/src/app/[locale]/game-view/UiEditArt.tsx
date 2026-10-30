"use client";

import { useContext } from "react";
import type { GuiNode, GuiSprite, GuiWindow } from "@/lib/guiRenderApi";
import { guiRenderApi } from "@/lib/guiRenderApi";
import { AXIS_STYLE } from "../gui-renderer/guiClassification";
import { EditContext } from "./editContext";
import type { EditSelection } from "./editModel";
import { spriteUrlOf } from "./spriteFrame";
import type { SpriteSwaps } from "./spriteSwaps";

export function artSelection(
  owner: string,
  id: string,
  gfx: string,
  sprite: GuiSprite,
  title: string,
  frame = 1,
): EditSelection {
  const node = { type: "icontype", name: title, sprite: gfx, frame } as GuiNode;
  const win = {
    name: owner,
    file: "",
    node,
    sprites: { [gfx]: sprite },
    texts: {},
    fonts: {},
    overlap_groups: [],
  } as GuiWindow;
  return { owner, win, id, node };
}

const spriteMeta = new Map<string, Promise<GuiSprite | undefined>>();

export function invalidateLayerSpriteMeta(
  projectId: number | null,
  name: string,
): void {
  if (!projectId || !name) return;
  spriteMeta.delete(`${projectId}:${name}`);
}

export function layerSpriteMeta(
  projectId: number,
  name: string,
): Promise<GuiSprite | undefined> {
  const key = `${projectId}:${name}`;
  let found = spriteMeta.get(key);
  if (!found) {
    guiRenderApi.setProjectId(projectId);
    found = guiRenderApi
      .sprites([name])
      .then((sprites) => sprites[name])
      .catch(() => {
        spriteMeta.delete(key);
        return undefined;
      });
    spriteMeta.set(key, found);
  }
  return found;
}

export function artUrl(
  gfx: string | undefined,
  sprite: GuiSprite | undefined,
  swaps: SpriteSwaps,
): string | undefined {
  if (gfx && swaps[gfx]) return guiRenderApi.spriteUrl(gfx, swaps[gfx].version);
  return sprite ? spriteUrlOf(sprite, gfx) : undefined;
}

export function UiEditArt({
  owner,
  gfx,
  sprite,
  title,
  frame,
  x,
  y,
  w,
  h,
}: {
  owner: string;
  gfx: string | undefined;
  sprite: GuiSprite | undefined;

  title: string;
  frame?: number;
  x: number;
  y: number;
  w: number;
  h: number;
}) {
  const edit = useContext(EditContext);
  if (!edit.editMode || !gfx || !sprite) return null;
  const id = `${owner}:${gfx}`;
  const picked = edit.selection?.owner === owner && edit.selection.id === id;
  const axis = AXIS_STYLE.art;
  return (
    <div
      title={`${title} · ${gfx}`}
      data-ui-edit-art={gfx}
      onPointerDown={(e) => {
        e.stopPropagation();
        edit.select(artSelection(owner, id, gfx, sprite, title, frame));
      }}

      onClick={(e) => e.stopPropagation()}
      className="transition-opacity hover:opacity-100"
      style={{
        position: "absolute",
        left: x,
        top: y,
        width: w,
        height: h,
        cursor: "pointer",
        pointerEvents: "auto",
        outline: picked
          ? `2px solid ${axis.outline}`
          : `1px dashed ${axis.outline}`,
        outlineOffset: picked ? 1 : undefined,
        opacity: picked ? 1 : 0.55,
        background: picked ? axis.fill : "transparent",
      }}
    />
  );
}
