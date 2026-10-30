import { describe, expect, it } from "vitest";
import type { GuiSprite } from "@/lib/guiRenderApi";
import { applySwaps, drawSize, type SpriteSwaps } from "./spriteSwaps";

describe("drawSize", () => {
  it("keeps the game's size while the art is untouched", () => {
    expect(drawSize({}, "GFX_portrait_CHI_chiang_kai_shek", 156, 210)).toEqual({
      width: 156,
      height: 210,
    });
  });

  it("uses the replacement's own size, the way the engine will draw it", () => {
    const swaps: SpriteSwaps = {
      GFX_focus: { version: 1, width: 94, height: 76 },
    };
    expect(drawSize(swaps, "GFX_focus", 156, 210)).toEqual({
      width: 94,
      height: 76,
    });
  });

  it("falls back to the declared size when a swap reports none (a revert)", () => {
    const swaps: SpriteSwaps = {
      GFX_focus: { version: 2, width: 0, height: 0 },
    };
    expect(drawSize(swaps, "GFX_focus", 156, 210)).toEqual({
      width: 156,
      height: 210,
    });
  });

  it("ignores a missing sprite name", () => {
    expect(
      drawSize(
        { GFX_focus: { version: 1, width: 10, height: 10 } },
        null,
        64,
        64,
      ),
    ).toEqual({
      width: 64,
      height: 64,
    });
  });
});

describe("applySwaps", () => {
  const sprite = (width: number, height: number): GuiSprite =>
    ({
      resolved: true,
      frames: 1,
      width,
      height,
      vanilla_url: "http://store/x.png",
    }) as GuiSprite;

  it("busts the cache for a swapped sprite and draws it at the replacement's size", () => {
    const sprites = { GFX_a: sprite(156, 210), GFX_b: sprite(10, 10) };
    const out = applySwaps(sprites, {
      GFX_a: { version: 7, width: 94, height: 76 },
    });
    expect(out.versions).toEqual({ GFX_a: 7 });
    expect(out.sprites.GFX_a).toMatchObject({ width: 94, height: 76 });
    expect(out.sprites.GFX_b).toBe(sprites.GFX_b);

    expect(sprites.GFX_a.width).toBe(156);
  });

  it("keeps the original size on a revert (no size) but still reloads the art", () => {
    const sprites = { GFX_a: sprite(156, 210) };
    const out = applySwaps(sprites, {
      GFX_a: { version: 8, width: 0, height: 0 },
    });
    expect(out.versions).toEqual({ GFX_a: 8 });
    expect(out.sprites).toBe(sprites);
  });
});
