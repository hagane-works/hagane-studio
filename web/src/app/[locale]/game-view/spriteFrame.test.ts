import { describe, expect, it } from "vitest";
import type { GuiNode, GuiSprite } from "@/lib/guiRenderApi";
import {
  expandButtonOf,
  frameStyle,
  gameCursor,
  spriteUrlOf,
} from "./spriteFrame";

function sprite(partial: Partial<GuiSprite>): GuiSprite {
  return {
    resolved: true,
    frames: 1,
    width: 0,
    height: 0,
    effect: null,
    effect_kind: null,
    cornered: false,
    bordersize: null,
    ...partial,
  };
}

function box(
  raw: Record<string, unknown>,
  size = { width: 180, height: 30 },
): GuiNode {
  return {
    type: "dropdownboxtype",
    name: "equipment_categories",
    position: { x: 150, y: 283 },
    size,
    orientation: null,
    origo: null,
    scale: null,
    sprite: null,
    frame: null,
    text: null,
    text_source: null,
    font: null,
    format: null,
    maxwidth: null,
    maxheight: null,
    text_color: null,
    centerposition: false,
    alwaystransparent: false,
    background: null,
    children: [],
    raw,
  };
}

describe("a sprite frame drawn by a layer", () => {
  it("takes its picture the way GuiCanvas does", () => {
    const vanilla = sprite({
      vanilla_url: "https://store/unit_infantry_icon.png",
    });
    expect(spriteUrlOf(vanilla, "GFX_unit_infantry_icon_medium")).toBe(
      "https://store/unit_infantry_icon.png",
    );

    const modded = sprite({});
    expect(spriteUrlOf(modded, "GFX_unit_mod_icon_medium")).toContain(
      "sprite?name=GFX_unit_mod_icon_medium",
    );

    const swapped = sprite({
      vanilla_url: "/api/go/gui-render/sprite?name=GFX_x&v=content-hash",
      overridden: true,
    });
    expect(spriteUrlOf(swapped, "GFX_x")).toBe(
      "/api/go/gui-render/sprite?name=GFX_x&v=content-hash",
    );

    expect(
      spriteUrlOf(
        sprite({ version: "a1b2c3d4e5f6" }),
        "gfx/leaders/GER/portrait_GER_new_leader.png",
      ),
    ).toContain("&v=a1b2c3d4e5f6");

    expect(
      spriteUrlOf(sprite({ missing_file: true, resolved: false }), "GFX_gone"),
    ).toBeUndefined();
  });

  it("draws one frame at the element's scale", () => {
    const counter = sprite({
      width: 152,
      height: 42,
      frames: 2,
      vanilla_url: "u.png",
    });
    const style = frameStyle(counter, 2, 20, { scale: 0.8 });
    expect([style.width, style.height, style.backgroundSize]).toEqual([
      61,
      34,
      "122px 34px",
    ]);
    expect(style.backgroundPosition).toBe("0px 0");

    const strip = sprite({
      width: 216,
      height: 37,
      frames: 6,
      vanilla_url: "p.png",
    });
    expect(frameStyle(strip, 0, 0, { frame: 5 }).backgroundPosition).toBe(
      "-144px 0",
    );

    expect(frameStyle(strip, 0, 0, { frame: 9 }).backgroundPosition).toBe(
      "-180px 0",
    );
  });
});

describe("a dropdown's expand button", () => {
  it("is read from the raw block the parser keeps", () => {
    const raw = {
      expandbutton: {
        __block__: [
          { k: "name", q: true, v: "expand_button" },
          {
            k: "position",
            v: {
              __block__: [
                { k: "x", v: -9 },
                { k: "y", v: 6 },
              ],
            },
          },
          { k: "quadTextureSprite", q: true, v: "GFX_expand_button" },
          { k: "Orientation", q: true, v: "UPPER_RIGHT" },
        ],
      },
    };
    expect(expandButtonOf(box(raw))).toEqual({
      sprite: "GFX_expand_button",
      x: 171,
      y: 6,
      scale: 1,
    });
  });

  it("carries its own scale, as the foreign-templates button does", () => {
    const raw = {
      expandbutton: {
        __block__: [
          {
            k: "position",
            v: {
              __block__: [
                { k: "x", v: -88 },
                { k: "y", v: 17 },
              ],
            },
          },
          {
            k: "quadTextureSprite",
            q: true,
            v: "GFX_foreign_templates_dropdown_button",
          },
          { k: "Orientation", q: true, v: "UPPER_RIGHT" },
          { k: "scale", v: 0.83 },
        ],
      },
    };
    expect(expandButtonOf(box(raw, { width: 60, height: 40 }))).toEqual({
      sprite: "GFX_foreign_templates_dropdown_button",
      x: -28,
      y: 17,
      scale: 0.83,
    });
  });

  it("is left to the canvas if the parser ever models it", () => {
    const parsed = box({
      expandbutton: { quadTextureSprite: "GFX_expand_button" },
    });
    expect(expandButtonOf(parsed)?.sprite).toBe("GFX_expand_button");
    parsed.children = [{ ...parsed, type: "expandbutton", children: [] }];
    expect(expandButtonOf(parsed)).toBeUndefined();
    expect(expandButtonOf(box({}))).toBeUndefined();
  });
});

describe("gameCursor", () => {
  it("points at the game's own pointer in the store the window's art comes from", () => {
    const sprites = {
      GFX_generated: sprite({ vanilla_url: undefined }),
      GFX_menu_button: sprite({
        vanilla_url:
          "http://127.0.0.1:8099/vanilla/gfx/interface/topbar/menu_button.png",
      }),
    };
    expect(gameCursor(sprites)).toBe(
      'url("http://127.0.0.1:8099/vanilla/gfx/cursors/normal.png") 0 0, auto',
    );
  });

  it("takes the pointer from the base-game route when the art comes as signed links", () => {
    const sprites = {
      GFX_menu_button: sprite({
        vanilla_url:
          "/art/vanilla/gfx/interface/topbar/menu_button.png?KeyId=AK&Expires=1&Signature=s",
      }),
    };
    expect(gameCursor(sprites)).toBe(
      'url("/api/go/gui-vanilla/gfx/cursors/normal.png") 0 0, auto',
    );
  });

  it("leaves the browser's pointer when no art comes from the store", () => {
    expect(gameCursor({ GFX_mod: sprite({}) })).toBeUndefined();
  });
});
