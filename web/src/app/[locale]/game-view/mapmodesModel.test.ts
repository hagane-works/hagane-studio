import { describe, expect, it } from "vitest";
import type { GuiNode, GuiWindow } from "@/lib/guiRenderApi";
import {
  MAP_MODE_LAYER,
  MAP_MODE_SLOTS,
  mapModeSlotOf,
  mapmodesWindow,
} from "./mapmodesModel";

let next = 0;
const node = (name: string, children: GuiNode[] = []): GuiNode =>
  ({
    name,
    _id: `n${next++}`,
    type: "buttontype",
    children,
    frame: null,
  }) as unknown as GuiNode;

function panel(): GuiWindow {
  const four = node("4_button_mapmode", [
    node("mapmode_button_1_bg"),
    node("mapmode_button_1"),
    node("mapmode_button_1_s"),
    node("mapmode_button_2"),
    node("mapmode_button_2_s"),
    node("tutorial_highlight_default_mapmode"),
    node("mapmode_button_5_s_bg"),
    node("mapmode_button_5"),
    node("mapmode_button_5_s"),
    node("day_night_toggle"),
  ]);
  const three = node("3_button_mapmode", [
    node("mapmode_button_1"),
    node("mapmode_button_1_s"),
  ]);
  const root = node("MapmodesInterface_Ingame", [
    node("social_view_interface_window"),
    node("open_find_view_button"),
    three,
    four,
  ]);
  return {
    name: "MapmodesInterface_Ingame",
    node: root,
  } as unknown as GuiWindow;
}

const byName = (
  root: GuiNode,
  name: string,
  under = "",
): GuiNode | undefined => {
  let found: GuiNode | undefined;
  const walk = (at: GuiNode, path: string) => {
    if (at.name === name && path.includes(under)) found ??= at;
    for (const child of at.children ?? []) walk(child, `${path}/${at.name}`);
  };
  walk(root, "");
  return found;
};
const id = (n: GuiNode | undefined) => (n as unknown as { _id: string })._id;

describe("the map mode panel", () => {
  it("draws each slot's own icon, the selected mode lit and the others unlit", () => {
    const { win, hidden } = mapmodesWindow(panel(), "naval");
    const four = byName(win.node, "4_button_mapmode")!;
    expect(byName(four, "mapmode_button_1")!.frame).toBe(
      MAP_MODE_SLOTS.mapmode_button_1.frame,
    );
    expect(byName(four, "mapmode_button_2")!.frame).toBe(3);
    expect(byName(four, "mapmode_button_5")!.frame).toBe(6);

    expect(hidden).toContain(id(byName(four, "mapmode_button_2")));
    expect(hidden).not.toContain(id(byName(four, "mapmode_button_2_s")));
    expect(hidden).toContain(id(byName(four, "mapmode_button_1_s")));
    expect(hidden).not.toContain(id(byName(four, "mapmode_button_1")));

    expect(hidden).toContain(id(byName(four, "mapmode_button_5_s_bg")));

    expect(hidden).not.toContain(id(byName(four, "mapmode_button_1_bg")));
    expect(hidden).not.toContain(id(byName(four, "day_night_toggle")));
  });

  it("shows the four-button layout without the social view or the tutorial's highlights", () => {
    const { win, hidden } = mapmodesWindow(panel(), "land");
    expect(hidden).toContain(id(byName(win.node, "3_button_mapmode")));
    expect(hidden).toContain(
      id(byName(win.node, "social_view_interface_window")),
    );
    expect(hidden).toContain(
      id(byName(win.node, "tutorial_highlight_default_mapmode")),
    );
    expect(hidden).not.toContain(id(byName(win.node, "4_button_mapmode")));
    expect(hidden).not.toContain(id(byName(win.node, "open_find_view_button")));
  });

  it("names the four-button layout as shown, over the overlap rule that defaults to the three", () => {
    const { win, shown } = mapmodesWindow(panel(), "land");
    expect(shown).toEqual([id(byName(win.node, "4_button_mapmode"))]);
  });

  it("knows a button's slot and which modes the map can draw", () => {
    expect(mapModeSlotOf("mapmode_button_3_s")).toBe("mapmode_button_3");
    expect(mapModeSlotOf("mapmode_button_10")).toBe("mapmode_button_10");
    expect(mapModeSlotOf("mapmode_button_4_s_bg")).toBeNull();
    expect(mapModeSlotOf("open_find_view_button")).toBeNull();
    expect(MAP_MODE_LAYER.land).toBe("political");
    expect(MAP_MODE_LAYER.naval).toBe("region");
    expect(MAP_MODE_LAYER.supply).toBeUndefined();
  });
});
