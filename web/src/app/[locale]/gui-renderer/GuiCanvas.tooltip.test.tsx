import { describe, expect, it } from "vitest";
import { renderToStaticMarkup } from "react-dom/server";
import type { GuiNode, GuiSprite, GuiText } from "@/lib/guiRenderApi";
import {
  GuiCanvas,
  buttonLook,
  resolveNodeTooltip,
  tooltipAt,
  tooltipTargets,
} from "./GuiCanvas";

function node(partial: Partial<GuiNode>): GuiNode {
  return {
    type: "icontype",
    name: null,
    position: { x: 0, y: 0 },
    size: null,
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
    ...partial,
  };
}

function sprite(partial: Partial<GuiSprite> = {}): GuiSprite {
  return {
    resolved: true,
    frames: 1,
    width: 40,
    height: 30,
    effect: null,
    effect_kind: null,
    cornered: false,
    bordersize: null,
    vanilla_url: "x.png",
    ...partial,
  };
}

const SPRITES: Record<string, GuiSprite> = {
  GFX_icon: sprite(),
  GFX_toolbar: sprite({ frames: 2, width: 80 }),
  GFX_zoom: sprite({
    effect: "gfx/FX/buttonstate.lua",
    effect_kind: "button_state",
  }),
};

const TEXTS: Record<string, GuiText> = {
  ICON_TIP: { display: "§HPolitical Power§!\nSpend it on laws.", kind: "loc" },
  ZOOM_TIP: { display: "§TDecrease speed§!", kind: "loc" },
  RUNTIME_TIP: { display: "Current value: $VAL|1%H$", kind: "loc" },
};

function topbarLike(): GuiNode {
  return node({
    type: "containerwindowtype",
    name: "w",
    size: { width: 400, height: 200 },
    children: [
      node({
        name: "pol_power_icon",
        sprite: "GFX_icon",
        position: { x: 10, y: 10 },
        raw: { pdx_tooltip: "ICON_TIP" },
      }),
      node({
        name: "runtime_icon",
        sprite: "GFX_icon",
        position: { x: 100, y: 10 },
        raw: { pdx_tooltip: "ICON_TIP" },
      }),
      node({
        name: "runtime_cover",
        sprite: "GFX_icon",
        position: { x: 100, y: 10 },
        raw: { pdx_tooltip: "RUNTIME_TIP" },
      }),
      node({
        type: "buttontype",
        name: "button_speeddown",
        sprite: "GFX_zoom",
        position: { x: 30, y: 20 },
        raw: { pdx_tooltip: "ZOOM_TIP" },
      }),
      node({
        name: "navy_button_like",
        type: "buttontype",
        sprite: "GFX_toolbar",
        position: { x: 200, y: 10 },
      }),
      node({
        name: "glow",
        sprite: "GFX_icon",
        position: { x: 0, y: 0 },
        alwaystransparent: true,
        raw: { pdx_tooltip: "ZOOM_TIP" },
      }),
    ],
  });
}

describe("tooltips on any node", () => {
  const root = topbarLike();
  const tips = tooltipTargets(root, SPRITES, TEXTS, {}, 1920, 1080);
  const resolve =
    (tooltipFor?: (n: GuiNode) => string | null | undefined) => (n: GuiNode) =>
      resolveNodeTooltip(n, TEXTS, tooltipFor);

  it("finds a non-button icon's pdx_tooltip", () => {
    const hit = tooltipAt(tips, resolve(), 15, 15);
    expect(hit?.content.text).toBe("§HPolitical Power§!\nSpend it on laws.");
    expect(hit?.rect).toEqual({ x: 10, y: 10, w: 40, h: 30 });
  });

  it("picks the topmost node, and skips alwaystransparent ones", () => {
    expect(tooltipAt(tips, resolve(), 35, 25)?.content.text).toBe(
      "§TDecrease speed§!",
    );
  });

  it("passes over a node whose tooltip is all run-time to the one below", () => {
    expect(tooltipAt(tips, resolve(), 105, 15)?.content.text).toBe(
      "§HPolitical Power§!\nSpend it on laws.",
    );
  });

  it("lets the screen supply text for nodes the engine fills in code", () => {
    const composed = (n: GuiNode) =>
      n.name === "navy_button_like"
        ? "§HNavy§!\nSee an overview of your fleets."
        : undefined;
    expect(tooltipAt(tips, resolve(composed), 210, 15)?.content.text).toBe(
      "§HNavy§!\nSee an overview of your fleets.",
    );
    expect(tooltipAt(tips, resolve(), 210, 15)).toBeNull();

    const off = (n: GuiNode) =>
      n.name === "pol_power_icon" ? null : undefined;
    expect(tooltipAt(tips, resolve(off), 15, 15)).toBeNull();
    expect(tooltipAt(tips, resolve(off), 35, 25)?.content.text).toBe(
      "§TDecrease speed§!",
    );
  });

  it("leaves hidden nodes out", () => {
    const hiddenRoot = topbarLike();
    const withHidden = tooltipTargets(
      hiddenRoot,
      SPRITES,
      TEXTS,
      {},
      1920,
      1080,
      new Set(["root/0"]),
    );
    expect(tooltipAt(withHidden, resolve(), 15, 15)).toBeNull();
  });
});

describe("button under the pointer", () => {
  const toolbar = node({ type: "buttontype", sprite: "GFX_toolbar" });
  const zoom = node({ type: "buttontype", sprite: "GFX_zoom" });

  it("lights a multi-frame button with its frame 2 (game_0537's Decisions button)", () => {
    expect(buttonLook(toolbar, SPRITES.GFX_toolbar, "over")).toEqual({
      frame: 2,
    });
    expect(buttonLook(toolbar, SPRITES.GFX_toolbar, "down")).toEqual({
      frame: 2,
    });
    expect(
      buttonLook(toolbar, sprite({ frames: 4, width: 160 }), "down"),
    ).toEqual({ frame: 3 });
    expect(buttonLook(toolbar, SPRITES.GFX_toolbar, null)).toEqual({
      frame: null,
    });
  });

  it("recolours a buttonstate sprite the way its shader does", () => {
    const over = buttonLook(zoom, SPRITES.GFX_zoom, "over");
    expect(over.frame).toBeNull();

    expect(over.style).toMatchObject({
      filter: "contrast(0.9032) brightness(1.24)",
    });
    expect(String(over.style?.transition)).toContain("250ms");
    expect(buttonLook(zoom, SPRITES.GFX_zoom, "down").style).toMatchObject({
      filter: "brightness(0.7857) contrast(1.12)",
    });
    const noDown = sprite({
      effect: "gfx/FX/buttonstate_nodowneffect.lua",
      effect_kind: "button_state",
    });
    expect(buttonLook(zoom, noDown, "down").style).toEqual({});
    const onlyDisable = sprite({
      effect: "gfx/FX/buttonstate_onlydisable.lua",
      effect_kind: "button_state",
    });
    expect(buttonLook(zoom, onlyDisable, "over").style).toEqual({});
  });

  it("leaves a strip frame the .gui chose, one-frame art and non-buttons alone", () => {
    expect(
      buttonLook(
        { ...toolbar, frame: 2 },
        sprite({ frames: 3, width: 120 }),
        "over",
      ),
    ).toEqual({ frame: 2 });
    expect(buttonLook(toolbar, SPRITES.GFX_icon, "over")).toEqual({
      frame: null,
    });
    expect(
      buttonLook(node({ sprite: "GFX_toolbar" }), SPRITES.GFX_toolbar, "over"),
    ).toEqual({ frame: null });
  });
});

describe("play layer", () => {
  const render = (editMode: boolean) =>
    renderToStaticMarkup(
      <GuiCanvas
        node={topbarLike()}
        sprites={SPRITES}
        texts={TEXTS}
        fonts={{}}
        versions={{}}
        hidden={new Set()}
        isolateId={null}
        selectedId={null}
        onSelect={() => {}}
        editMode={editMode}
        onActivate={() => {}}
      />,
    );

  it("drops the flat white hover box and the browser's hand cursor", () => {
    const html = render(false);
    expect(html).toContain('role="button"');
    expect(html).not.toContain("hover:bg-white");
    expect(html).not.toContain("cursor:pointer");
    expect(html).not.toContain("title=");
  });

  it("carries a tooltip host only in play mode", () => {
    expect(render(false)).toContain("position:fixed");
    expect(render(true)).not.toContain("position:fixed");
  });
});
