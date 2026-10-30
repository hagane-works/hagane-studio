import type { GuiSprite } from "@/lib/guiRenderApi";

export interface SpriteSwap {
  version: number;
  width: number;
  height: number;
}

export type SpriteSwaps = Record<string, SpriteSwap>;

export function drawSize(
  swaps: SpriteSwaps,
  gfx: string | null | undefined,
  declaredW: number,
  declaredH: number,
): { width: number; height: number } {
  const swap = gfx ? swaps[gfx] : undefined;
  if (!swap || !swap.width || !swap.height)
    return { width: declaredW, height: declaredH };
  return { width: swap.width, height: swap.height };
}

export function applySwaps(
  sprites: Readonly<Record<string, GuiSprite>>,
  swaps: SpriteSwaps,
): { sprites: Record<string, GuiSprite>; versions: Record<string, number> } {
  const versions: Record<string, number> = {};
  let out: Record<string, GuiSprite> | null = null;
  for (const [gfx, swap] of Object.entries(swaps)) {
    versions[gfx] = swap.version;
    const meta = sprites[gfx];
    if (!meta || !swap.width || !swap.height) continue;
    out ??= { ...sprites };
    out[gfx] = { ...meta, width: swap.width, height: swap.height };
  }
  return { sprites: out ?? (sprites as Record<string, GuiSprite>), versions };
}
