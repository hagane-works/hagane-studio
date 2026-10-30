import { describe, expect, it } from "vitest";
import type { GuiCountryStats, GuiNode, GuiWindow } from "@/lib/guiRenderApi";
import {
  BASE_RESEARCH_SLOTS,
  UNUSED_SLOT,
  emptySlotRow,
  researchSlotCount,
} from "./ResearchLayer";
import { matchNamePath } from "./gameNavigation";

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

function slotEntry(): GuiWindow {
  const icon = (name: string) => node({ name, sprite: `GFX_${name}` });
  const text = (name: string) =>
    node({ type: "instanttextboxtype", name, font: "hoi_18mbs" });
  return {
    name: "research_slot_entry",
    file: "interface/countrytechnologyview.gui",
    node: node({
      type: "containerwindowtype",
      name: "research_slot_entry",
      children: [
        icon("empty_research_slot_glow"),
        icon("technology"),
        icon("designer"),
        icon("equipment"),
        icon("carrier_fighter"),
        text("title"),
        text("year"),
        text("eta"),
        icon("research_progressbar_frame_icon"),
        icon("research_progressbar"),
      ],
    }),
    sprites: {},
    texts: {},
    fonts: {},
    overlap_groups: [],
  };
}

const idOf = (n: GuiNode) => (n as GuiNode & { _id?: string })._id ?? "";

describe("emptySlotRow", () => {
  it("writes the empty slot's title key into the box the .gui leaves for the engine", () => {
    const row = emptySlotRow(slotEntry());
    expect(matchNamePath(row.node, "title")[0].text).toBe(UNUSED_SLOT);
  });

  it("draws the bar's frame but nothing that describes a technology being researched", () => {
    const row = emptySlotRow(slotEntry());
    const hidden = (name: string) =>
      row.hidden.has(idOf(matchNamePath(row.node, name)[0]));
    for (const name of [
      "technology",
      "designer",
      "equipment",
      "carrier_fighter",
      "year",
      "eta",
      "research_progressbar",
      "empty_research_slot_glow",
    ]) {
      expect(hidden(name), name).toBe(true);
    }
    expect(hidden("research_progressbar_frame_icon")).toBe(false);
    expect(hidden("title")).toBe(false);
  });

  it("leaves the shared template untouched", () => {
    const entry = slotEntry();
    emptySlotRow(entry);
    expect(matchNamePath(entry.node, "title")[0].text).toBeNull();
  });
});

describe("researchSlotCount", () => {
  it("lists the slots the country's history sets", () => {
    expect(researchSlotCount({ research_slots: 4 } as GuiCountryStats)).toBe(4);
  });

  it("falls back to the base slots when history sets none", () => {
    expect(researchSlotCount(undefined)).toBe(BASE_RESEARCH_SLOTS);
    expect(researchSlotCount({ research_slots: 0 } as GuiCountryStats)).toBe(
      BASE_RESEARCH_SLOTS,
    );
  });
});
