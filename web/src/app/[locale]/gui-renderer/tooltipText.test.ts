import { beforeEach, describe, expect, it } from "vitest";
import type { GuiBitmapFont, GuiNode } from "@/lib/guiRenderApi";
import { REFERENCE_TABLES } from "@/lib/__fixtures__/referenceTables";
import { setReferenceTables } from "@/lib/referenceTables";
import {
  TOOLTIP,
  bitmapMeasure,
  dropRuntimeLines,
  fillRuntimeSlots,
  iconNamesOf,
  iconSpriteName,
  layoutTooltip,
  nodeTooltip,
  parseTooltipMarkup,
  placeTooltip,
  shortcutMarkup,
  tooltipAnchorOf,
  tooltipBoxSize,
  tooltipMarkup,
  type TooltipMeasure,
} from "./tooltipText";

const TEXT_COLORS = REFERENCE_TABLES.text_colors.global;
beforeEach(() => setReferenceTables(REFERENCE_TABLES));

const CG_16B: GuiBitmapFont = {
  line: 16,
  base: 13,
  atlas: "cg_16b.png",
  atlas_w: 512,
  atlas_h: 256,
  tint: "luminance",
  chars: {
    "32": [491, 223, 8, 7, -3, 12, 4],
    "43": [99, 218, 13, 13, -3, 2, 8],
    "45": [401, 225, 11, 8, -3, 6, 5],
    "46": [447, 223, 9, 8, -3, 8, 4],
    "58": [328, 214, 9, 13, -3, 3, 4],
    "68": [135, 150, 14, 16, -2, 0, 9],
    "70": [427, 165, 12, 16, -2, 0, 6],
    "72": [210, 149, 14, 16, -3, 0, 9],
    "73": [279, 183, 9, 16, -3, 0, 4],
    "78": [320, 132, 15, 16, -3, 0, 10],
    "80": [429, 148, 13, 16, -3, 0, 7],
    "81": [161, 80, 17, 17, -3, 0, 11],
    "83": [401, 148, 13, 16, -3, 0, 7],
    "84": [492, 165, 12, 16, -3, 0, 5],
    "97": [240, 200, 15, 13, -3, 3, 9],
    "99": [445, 197, 14, 13, -3, 3, 8],
    "100": [112, 133, 15, 16, -3, 0, 9],
    "101": [368, 199, 15, 13, -3, 3, 8],
    "102": [139, 184, 11, 16, -3, 0, 4],
    "104": [373, 149, 13, 16, -3, 0, 8],
    "105": [299, 183, 9, 16, -3, 0, 3],
    "107": [0, 153, 14, 16, -3, 0, 8],
    "108": [289, 183, 9, 16, -3, 0, 3],
    "110": [183, 215, 13, 13, -3, 3, 8],
    "111": [336, 199, 15, 13, -3, 3, 8],
    "112": [439, 114, 15, 16, -3, 3, 9],
    "114": [316, 214, 11, 13, -3, 3, 4],
    "115": [278, 214, 12, 13, -3, 3, 6],
    "116": [199, 183, 11, 16, -3, 0, 4],
    "117": [127, 215, 13, 13, -3, 3, 8],
    "118": [490, 197, 14, 13, -3, 3, 7],
    "119": [118, 201, 17, 13, -3, 3, 11],
    "121": [414, 131, 14, 16, -3, 3, 8],
  },
};

const lineText = (layout: ReturnType<typeof layoutTooltip>, row: number) =>
  layout.lines[row].items
    .map((item) => (item.kind === "text" ? item.text : `[${item.name}]`))
    .join("");

const SYNTH: TooltipMeasure = {
  line: 16,
  advance: (_prev, code) => (code === 32 ? 4 : 1),
};

describe("tooltip box, against the recordings", () => {
  it("sizes the Navy tooltip as game_0136 shows it: widest line 193 → 229 wide", () => {
    const { text, icons } = shortcutMarkup("p");
    const markup = `§HNavy§!\nSee an overview of your fleets.\n--------------\nShortcut: ${text}`;
    const layout = layoutTooltip(
      markup,
      bitmapMeasure(CG_16B),
      (name) => icons[name] ?? null,
    );
    expect(layout.lines.map((line) => line.width)).toEqual([34, 193, 70, 66]);
    expect(layout.width).toBe(229);
    expect(layout.height).toBe(4 * 16 + 36);

    expect(layout.lines[0].items).toEqual([
      { kind: "text", x: 0, text: "Navy", color: TEXT_COLORS.H, prev: -1 },
    ]);
    expect(lineText(layout, 3)).toBe("Shortcut: P");
  });

  it("wraps the Decisions description where the game does, keeping the trailing space", () => {
    const markup =
      "§HDecisions§!\nTake decisions and react to events that happen to your nation.\n\n--------------";
    const layout = layoutTooltip(markup, bitmapMeasure(CG_16B));
    expect(lineText(layout, 1)).toBe(
      "Take decisions and react to events that happen to ",
    );
    expect(layout.lines[1].width).toBe(325);
    expect(lineText(layout, 2)).toBe("your nation.");
    expect(lineText(layout, 3)).toBe("");
    expect(layout.width).toBe(361);
    expect(layout.height).toBe(5 * 16 + 36);
  });

  it("measures the three recorded boxes: 229, 361x548, 356", () => {
    expect(tooltipBoxSize([34, 193, 70, 66]).width).toBe(229);
    const decisions = tooltipBoxSize([325, ...Array(31).fill(100)]);
    expect([decisions.width, decisions.height]).toEqual([361, 548]);
    expect(tooltipBoxSize([320, 200]).width).toBe(356);

    expect(decisions.height).toBeGreaterThan(256);
  });

  it("keeps a 321 px line (325 with its space) and wraps a 328 px one (332)", () => {
    const stays = layoutTooltip(
      `${"a".repeat(150)} ${"a".repeat(167)} ${"a".repeat(10)}`,
      SYNTH,
    );
    expect(stays.lines.map((line) => line.width)).toEqual([325, 10]);
    const wraps = layoutTooltip(
      `${"a".repeat(150)} ${"a".repeat(174)} ${"a".repeat(10)}`,
      SYNTH,
    );
    expect(wraps.lines.map((line) => line.width)).toEqual([154, 188]);
    expect(TOOLTIP.wrapWidth).toBe(328);
  });

  it("splits a word only when its letters, not its trailing space, overflow", () => {
    const edge = layoutTooltip(`${"a".repeat(328)} b`, SYNTH);
    expect(edge.lines.map((_, row) => lineText(edge, row).length)).toEqual([
      329, 1,
    ]);
    const cjk = layoutTooltip("字".repeat(400), SYNTH);
    expect(cjk.lines.map((line) => line.width)).toEqual([328, 72]);
  });
});

describe("markup", () => {
  it("colours § spans and pops back with §!", () => {
    const [line] = parseTooltipMarkup("§YA§GB§!C§!D");
    expect(line).toEqual([
      { kind: "text", text: "A", color: TEXT_COLORS.Y },
      { kind: "text", text: "B", color: TEXT_COLORS.G },
      { kind: "text", text: "C", color: TEXT_COLORS.Y },
      { kind: "text", text: "D", color: null },
    ]);

    expect(parseTooltipMarkup("§zX")[0]).toEqual([
      { kind: "text", text: "X", color: null },
    ]);
  });

  it("draws every § code in the default colour when the colour table is empty", () => {
    setReferenceTables({});
    expect(parseTooltipMarkup("§YA§GB§!C§!D")[0]).toEqual([
      { kind: "text", text: "ABCD", color: null },
    ]);
    expect(fillRuntimeSlots("Save $VALUE|Y0$", { VALUE: 5 })).toBe("Save 5");
  });

  it("reads £icons in all three spellings", () => {
    expect(parseTooltipMarkup("£civ_factory Civilian")[0]).toEqual([
      { kind: "icon", name: "civ_factory", frame: 1, color: null },
      { kind: "text", text: "Civilian", color: null },
    ]);
    expect(parseTooltipMarkup("£command_power  5")[0][1]).toEqual({
      kind: "text",
      text: " 5",
      color: null,
    });
    expect(parseTooltipMarkup("£production_cost£ 43")[0][1]).toEqual({
      kind: "text",
      text: " 43",
      color: null,
    });
    expect(parseTooltipMarkup("£pol_power|2£x")[0][0]).toEqual({
      kind: "icon",
      name: "pol_power",
      frame: 2,
      color: null,
    });
    expect(parseTooltipMarkup("£ 5")[0]).toEqual([
      { kind: "text", text: "£ 5", color: null },
    ]);
    expect(iconNamesOf("Shortcut: £shift£+£q£ and £shift£")).toEqual([
      "shift",
      "q",
    ]);
  });

  it("spells shortcuts as keycap icons named the way the .gui names keys", () => {
    expect(shortcutMarkup("shift+q")).toEqual({
      text: "£shift£+£q£",
      icons: { shift: "SHIFT", q: "Q" },
    });
    expect(iconSpriteName("KP_MINUS")).toBe("GFX_KP_MINUS");
    expect(iconSpriteName("GFX_p")).toBe("GFX_p");
  });

  it("draws an icon as a picture, as its stand-in text, or not at all", () => {
    const measure = SYNTH;
    const picture = layoutTooltip("x£p£y", measure, () => ({ w: 17, h: 16 }));
    expect(picture.lines[0].width).toBe(19);
    expect(picture.lines[0].items[1]).toMatchObject({
      kind: "icon",
      name: "p",
      x: 1,
      w: 17,
    });
    expect(
      lineText(
        layoutTooltip("x£p£y", measure, () => "P"),
        0,
      ),
    ).toBe("xPy");
    expect(
      lineText(
        layoutTooltip("x£p£y", measure, () => null),
        0,
      ),
    ).toBe("xy");
  });
});

describe("run-time slots", () => {
  it("drops a line that still needs a running game", () => {
    expect(
      dropRuntimeLines("§HWorld Tension\n§! Current value: $VAL|1%H$"),
    ).toBe("§HWorld Tension");
    expect(dropRuntimeLines("[Root.GetName] leads\nStatic")).toBe("Static");

    expect(dropRuntimeLines("§Y$X$\nkept§!")).toBe("§Ykept§!");
  });

  it("shows nothing when nothing printable is left", () => {
    expect(tooltipMarkup({ text: "$COUNT|Y$ slots" }, true)).toBeNull();
    expect(tooltipMarkup({ text: "§H§!\n" }, true)).toBeNull();
    expect(tooltipMarkup({ text: "Shown\n" }, true)).toBe("Shown");
  });

  it("adds the delayed text under the instant text once it is due", () => {
    const content = {
      text: "§YResearch Slot Available§!",
      delayed: "Research is idle.",
    };
    expect(tooltipMarkup(content, false)).toBe("§YResearch Slot Available§!");
    expect(tooltipMarkup(content, true)).toBe(
      "§YResearch Slot Available§!\nResearch is idle.",
    );
    expect(tooltipMarkup({ delayed: "Only later" }, false)).toBeNull();
    expect(tooltipMarkup({ delayed: "Only later" }, true)).toBe("Only later");

    const joined = tooltipMarkup(
      { text: "§YTitle\n§!", delayed: "Body" },
      true,
    ) as string;
    expect(joined).toBe("§YTitle§!\nBody");
    expect(parseTooltipMarkup(joined)[1]).toEqual([
      { kind: "text", text: "Body", color: null },
    ]);
  });
});

describe("placement", () => {
  const screen = { w: 1920, h: 1080 };

  it("sits at cursor + (20, 23)", () => {
    expect(placeTooltip({ w: 229, h: 100 }, { x: 164, y: 52 }, screen)).toEqual(
      { x: 184, y: 75 },
    );
  });

  it("flips left of the cursor at the right edge (the Navy capture: box at 1466, 66)", () => {
    expect(
      placeTooltip({ w: 229, h: 100 }, { x: 1715, y: 43 }, screen),
    ).toEqual({ x: 1466, y: 66 });
  });

  it("goes above the cursor at the bottom edge, and stays on screen", () => {
    expect(
      placeTooltip({ w: 200, h: 300 }, { x: 100, y: 1000 }, screen),
    ).toEqual({ x: 120, y: 700 });
    expect(placeTooltip({ w: 2000, h: 50 }, { x: 10, y: 10 }, screen).x).toBe(
      0,
    );
  });

  it("pins to the element with pdx_tooltip_anchor", () => {
    const element = { x: 564, y: 11, w: 40, h: 40 };
    expect(
      placeTooltip(
        { w: 229, h: 100 },
        { x: 0, y: 0 },
        screen,
        { x: -310, y: 0, relative: true },
        element,
      ),
    ).toEqual({
      x: 254,
      y: 11,
    });
    expect(
      placeTooltip(
        { w: 229, h: 100 },
        { x: 0, y: 0 },
        screen,
        { x: 50, y: 60, relative: false },
        element,
      ),
    ).toEqual({
      x: 50,
      y: 60,
    });
  });
});

function node(raw: Record<string, unknown>): GuiNode {
  return {
    type: "icontype",
    name: "n",
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
    raw,
  };
}

describe("reading a node", () => {
  it("reads pdx_tooltip_anchor in both shapes the parser stores", () => {
    expect(tooltipAnchorOf({ x: -310, y: 0 })).toEqual({
      x: -310,
      y: 0,
      relative: true,
    });
    expect(
      tooltipAnchorOf({
        __block__: [
          { k: "x", v: -158 },
          { k: "y", v: 0 },
          { k: "relative", v: "no" },
        ],
      }),
    ).toEqual({ x: -158, y: 0, relative: false });
    expect(tooltipAnchorOf("nope")).toBeNull();
  });

  it("takes the node's tooltip keys from the window's texts", () => {
    const texts = {
      SPEED_DECREASE: { display: "§TDecrease speed§!", kind: "loc" as const },
      LATER: { display: "Then this.", kind: "loc" as const },
      MISSING: { display: "MISSING", kind: "key" as const },
    };
    expect(
      nodeTooltip(
        node({ pdx_tooltip: "SPEED_DECREASE", pdx_tooltip_delayed: "LATER" }),
        texts,
      ),
    ).toEqual({
      text: "§TDecrease speed§!",
      delayed: "Then this.",
      anchor: null,
    });

    expect(nodeTooltip(node({ pdx_tooltip: "MISSING" }), texts)).toBeNull();
    expect(nodeTooltip(node({ pdx_tooltip: "NOT_SENT" }), texts)).toBeNull();
    expect(nodeTooltip(node({}), texts)).toBeNull();
  });
});

describe("fillRuntimeSlots", () => {
  it("fills a slot in the game's format: colour and decimals", () => {
    expect(
      fillRuntimeSlots("Save £army_experience $VALUE|Y0$", { VALUE: 0 }),
    ).toBe("Save £army_experience §Y0§!");
    expect(fillRuntimeSlots("$X|1$", { X: 2.345 })).toBe("2.3");
  });

  it("prints a share as a percentage and signs it on request", () => {
    expect(fillRuntimeSlots("$BONUS|%+G1$", { BONUS: 0.155 })).toBe(
      "§G+15.5%§!",
    );
  });

  it("drops a slot nobody filled instead of printing its name", () => {
    expect(fillRuntimeSlots("Save £army_experience $VALUE|Y0$", {})).toBe(
      "Save £army_experience ",
    );

    expect(fillRuntimeSlots("Buy Equipment ($NUM|Y$)", {})).toBe(
      "Buy Equipment",
    );
    expect(fillRuntimeSlots("Buy Equipment ($NUM|Y$)", { NUM: 3 })).toBe(
      "Buy Equipment (§Y3§!)",
    );
  });

  it("writes a text value as it is, keeping the colour", () => {
    expect(fillRuntimeSlots("$NAME|H$ joins", { NAME: "Axis" })).toBe(
      "§HAxis§! joins",
    );
  });
});
