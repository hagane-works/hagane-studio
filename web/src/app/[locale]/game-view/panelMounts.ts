import type { UiPanel, UiPanelControl } from "@/types";
import {
  appendControl,
  controlNameSlug,
  getControlAtPath,
  removeControlAtPath,
  uniqueControlName,
} from "@/components/workbench/ui-panels/selectionStore";

export const PANEL_MOUNTS: Readonly<Record<string, string>> = {
  countrypoliticsview: "politics_tab",
  countrydecisionview: "decision_tab",
  countrytechnologyview: "technology_tab",
  countrydiplomacyview: "diplomacy_tab",
  countrytradeview: "trade_tab",
  countryconstructionsview: "construction_tab",
  countryproductionlineview: "production_tab",
  countrydeploymentview: "deployment_tab",
  countrylogisticsview: "logistics_tab",
  nationalfocusview: "national_focus",

  countrystateview: "selected_state_view",
};

export const TOPBAR_MOUNT = "top_bar";

export function contextOfMount(mount: string | null | undefined): string {
  return mount === "selected_state_view"
    ? "selected_state_context"
    : "player_context";
}

export function mountOf(screen: string | null | undefined): string | null {
  return (screen && PANEL_MOUNTS[screen]) || null;
}

export function panelsOnScreen(
  panels: readonly UiPanel[],
  screen: string | null | undefined,
): UiPanel[] {
  const mount = mountOf(screen);
  return mount
    ? panels.filter((panel) => panel.parent_window_token === mount)
    : [];
}

export function panelWindowName(
  panel: Pick<UiPanel, "token" | "derived">,
): string {
  return panel.derived?.window_name || `${panel.token}_panel`;
}

export type ComponentPreset =
  | "box"
  | "group"
  | "scroll_box"
  | "list"
  | "grid"
  | "text"
  | "title"
  | "paragraph"
  | "button"
  | "small_button"
  | "wide_button"
  | "image_button"
  | "switch"
  | "icon"
  | "divider"
  | "progressbar";

export interface Preset {
  kind: UiPanelControl["kind"];

  look: Partial<UiPanelControl> & { w: number; h: number };

  words?: "button" | "text" | "title" | "paragraph" | "listItem";
}

const CLICK = { click_effect: "", click_enabled_trigger: "" };

export const PRESETS: Record<ComponentPreset, Preset> = {
  box: {
    kind: "container",
    look: {
      w: 220,
      h: 120,
      sprite: "GFX_tiled_window_1b_border",
      children: [],
    },
  },

  group: {
    kind: "container",
    look: { w: 200, h: 100, sprite: "", children: [] },
  },

  scroll_box: {
    kind: "container",
    look: {
      w: 220,
      h: 160,
      sprite: "GFX_tiled_window_1b_border",
      scroll: true,
      children: [],
    },
  },

  list: {
    kind: "container",
    words: "listItem",
    look: {
      w: 240,
      h: 150,
      sprite: "GFX_tiled_window_1b_border",
      scroll: true,
      columns: 1,
      slot_w: 220,
      slot_h: 36,
      children: [],
    },
  },

  grid: {
    kind: "container",
    look: {
      w: 200,
      h: 104,
      sprite: "GFX_tiled_window_1b_border",
      columns: 4,
      slot_w: 48,
      slot_h: 48,
      children: [],
    },
  },
  text: { kind: "text", words: "text", look: { w: 200, h: 24 } },

  title: {
    kind: "text",
    words: "title",
    look: { w: 260, h: 32, font: "hoi_24header" },
  },

  paragraph: { kind: "text", words: "paragraph", look: { w: 260, h: 80 } },
  button: {
    kind: "button",
    words: "button",
    look: { w: 261, h: 34, sprite: "GFX_button_261x34", ...CLICK },
  },
  small_button: {
    kind: "button",
    words: "button",
    look: { w: 148, h: 36, sprite: "GFX_button_148x34", ...CLICK },
  },

  wide_button: {
    kind: "button",
    words: "button",
    look: { w: 180, h: 30, sprite: "GFX_tiling_sort_button", ...CLICK },
  },
  image_button: {
    kind: "button",
    look: {
      w: 32,
      h: 31,
      sprite: "GFX_decision_generic_political_discourse",
      ...CLICK,
    },
  },

  switch: {
    kind: "button",
    look: { w: 34, h: 30, sprite: "GFX_generic_checkbox", ...CLICK },
  },
  icon: {
    kind: "icon",
    look: { w: 94, h: 76, sprite: "GFX_goal_unknown", frame: 1 },
  },
  divider: {
    kind: "icon",
    look: { w: 514, h: 9, sprite: "GFX_divider", frame: 1 },
  },
  progressbar: {
    kind: "progressbar",
    look: { w: 220, h: 20, sprite: "GFX_building_level_bar", frame: 1 },
  },
};

export const PRESET_EN_WORDS: Record<NonNullable<Preset["words"]>, string> = {
  button: "Button",
  text: "Text",
  title: "Title",
  paragraph: "Several lines of text: the game wraps them at the box's width.",
  listItem: "Item [?v]",
};

export const PRESET_GROUPS: {
  group: "container" | "text" | "button" | "image";
  presets: ComponentPreset[];
}[] = [
  {
    group: "container",
    presets: ["box", "scroll_box", "list", "grid", "group"],
  },
  { group: "text", presets: ["text", "title", "paragraph"] },
  {
    group: "button",
    presets: [
      "button",
      "small_button",
      "wide_button",
      "image_button",
      "switch",
    ],
  },
  { group: "image", presets: ["icon", "divider", "progressbar"] },
];

const KIND_SIZE: Record<UiPanelControl["kind"], { w: number; h: number }> = {
  container: { w: 220, h: 120 },
  icon: { w: 94, h: 76 },
  text: { w: 200, h: 24 },
  progressbar: { w: 220, h: 20 },
  button: { w: 261, h: 34 },
};

export interface ComponentWords {
  en: string;
  locale: string;
  own: string;
}

export function switchEffect(variable: string): string {
  return [
    "if = {",
    `\tlimit = { check_variable = { ${variable} = 2 } }`,
    `\tset_variable = { ${variable} = 1 }`,
    "}",
    "else = {",
    `\tset_variable = { ${variable} = 2 }`,
    "}",
  ].join("\n");
}

export function newComponent(
  preset: ComponentPreset,
  x: number,
  y: number,
  controls: UiPanelControl[],
  text: ComponentWords,
  token = "",
): UiPanelControl {
  const { kind, look, words } = PRESETS[preset];
  const name = uniqueControlName(controls, preset);
  const control: UiPanelControl = {
    kind,
    name,
    x: Math.round(x),
    y: Math.round(y),
    ...look,
  };
  if (look.children) control.children = [];
  if (preset === "list" || preset === "grid") {
    control.array = `${token ? `${token}_` : ""}${name}`.replace(
      /[^A-Za-z0-9_]/g,
      "_",
    );
    const taken = [...controls, control];
    const row: UiPanelControl =
      preset === "list"
        ? {
            kind: "text",
            name: uniqueControlName(taken, `${name}_text`),
            x: 8,
            y: 6,
            w: 140,
            h: 24,
            text: text.en,
          }
        : {
            kind: "icon",
            name: uniqueControlName(taken, `${name}_icon`),
            x: 8,
            y: 8,
            w: 32,
            h: 31,
            sprite: "GFX_decision_generic_political_discourse",
            frame: 1,
          };
    if (preset === "list" && text.locale !== "en" && text.own)
      row.text_locs = { [text.locale]: text.own };
    control.children = [row];
    return control;
  }
  if (words) {
    control.text = text.en;
    if (text.locale !== "en" && text.own)
      control.text_locs = { [text.locale]: text.own };
  } else if (kind === "button") {
    control.text = "";
  }
  if (preset === "switch") {
    const variable = `${token ? `${token}_` : ""}${name}`.replace(
      /[^A-Za-z0-9_]/g,
      "_",
    );
    control.frame_var = variable;
    control.click_effect = switchEffect(variable);
  }
  return control;
}

export function listSlot(control: UiPanelControl): {
  w: number;
  h: number;
  columns: number;
} {
  return {
    w: Number(control.slot_w) || Number(control.w) || 100,
    h: Number(control.slot_h) || 30,
    columns: Math.max(1, Number(control.columns) || 1),
  };
}

function inSlot(
  x: number,
  y: number,
  w: number,
  h: number,
  slot?: { w: number; h: number },
): { x: number; y: number } {
  if (!slot) return { x, y };
  return {
    x: Math.min(Math.max(0, x), Math.max(0, slot.w - w)),
    y: Math.min(Math.max(0, y), Math.max(0, slot.h - h)),
  };
}

export function addTarget(
  controls: UiPanelControl[],
  selectedPath: string | null,
): string | null {
  if (!selectedPath) return null;
  if (getControlAtPath(controls, selectedPath)?.kind === "container")
    return selectedPath;
  const parent = selectedPath.split("/").slice(0, -1).join("/");
  return parent && getControlAtPath(controls, parent)?.kind === "container"
    ? parent
    : null;
}

export function addPreset(
  controls: UiPanelControl[],
  selectedPath: string | null,
  preset: ComponentPreset,
  text: ComponentWords,
  token = "",
): [UiPanelControl[], string] {
  const into = addTarget(controls, selectedPath);
  const holder = into ? getControlAtPath(controls, into) : null;
  const { w, h } = PRESETS[preset].look;
  if (!into || !holder)
    return appendControl(
      controls,
      null,
      newComponent(preset, 20, 20, controls, text, token),
    );
  const room = holder.array ? listSlot(holder) : null;
  const kids = holder.children ?? [];
  let x = 8;
  let y = 8;
  if (kids.length) {
    x = Math.max(...kids.map((kid) => kid.x + extent(kid).w)) + 4;
    y = Math.min(...kids.map((kid) => kid.y));
    if (x + w > (room ? room.w : extent(holder).w)) {
      x = 8;
      y = Math.max(...kids.map((kid) => kid.y + extent(kid).h)) + 4;
    }
  }
  const at = inSlot(x, y, w, h, room ?? undefined);
  return appendControl(
    controls,
    into,
    newComponent(preset, at.x, at.y, controls, text, token),
  );
}

export function extent(control: UiPanelControl): { w: number; h: number } {
  const size = KIND_SIZE[control.kind];
  return {
    w: Number(control.w) || size?.w || 100,
    h: Number(control.h) || size?.h || 24,
  };
}

export interface PanelPlace {
  window_props: Record<string, unknown>;
  controls: UiPanelControl[];
}

export function fitPanel(
  windowProps: Record<string, unknown>,
  controls: UiPanelControl[],
): PanelPlace {
  if (controls.length === 0) return { window_props: windowProps, controls };
  const left = Number(windowProps.x) || 0;
  const top = Number(windowProps.y) || 0;
  let minX = Infinity;
  let minY = Infinity;
  let maxX = -Infinity;
  let maxY = -Infinity;
  for (const control of controls) {
    const { w, h } = extent(control);
    minX = Math.min(minX, control.x);
    minY = Math.min(minY, control.y);
    maxX = Math.max(maxX, control.x + w);
    maxY = Math.max(maxY, control.y + h);
  }
  return {
    window_props: {
      ...windowProps,
      x: left + minX,
      y: top + minY,
      width: Math.max(1, maxX - minX),
      height: Math.max(1, maxY - minY),
    },
    controls: controls.map((control) => ({
      ...control,
      x: control.x - minX,
      y: control.y - minY,
    })),
  };
}

export type ScrollOf = (container: UiPanelControl) => number;

const NO_SCROLL: ScrollOf = () => 0;

export function isWindowPanel(panel: {
  parent_window_token?: string;
}): boolean {
  return panel.parent_window_token === "";
}

export function openFlagOf(token: string): string {
  return `${token.replace(/[^A-Za-z0-9_]/g, "_")}_open`;
}

export function openTrigger(flag: string): string {
  return `has_country_flag = ${flag}`;
}

export function opensByButton(
  panel: Pick<UiPanel, "token" | "visible_trigger">,
): boolean {
  return (
    (panel.visible_trigger ?? "").trim() ===
    openTrigger(openFlagOf(panel.token))
  );
}

export function toggleFlagEffect(flag: string): string {
  return [
    "if = {",
    `\tlimit = { has_country_flag = ${flag} }`,
    `\tclr_country_flag = ${flag}`,
    "}",
    "else = {",
    `\tset_country_flag = ${flag}`,
    "}",
  ].join("\n");
}

export function closeFlagEffect(flag: string): string {
  return `clr_country_flag = ${flag}`;
}

export function flagOpsOf(
  effect: string | undefined,
): { flag: string; op: "toggle" | "set" | "clear" }[] {
  const text = effect ?? "";
  const ops: { flag: string; op: "toggle" | "set" | "clear" }[] = [];
  const toggle =
    /limit\s*=\s*\{\s*has_country_flag\s*=\s*(\w+)\s*\}\s*clr_country_flag\s*=\s*\1[\s\S]*?else\s*=\s*\{\s*set_country_flag\s*=\s*\1/;
  const toggled = text.match(toggle);
  if (toggled) return [{ flag: toggled[1], op: "toggle" }];
  for (const m of text.matchAll(/(set|clr)_country_flag\s*=\s*(\w+)/g))
    ops.push({ flag: m[2], op: m[1] === "set" ? "set" : "clear" });
  return ops;
}

export const WINDOW_LOOK = {
  w: 420,
  h: 280,
  sprite: "GFX_tiled_window_2b_border",
} as const;

export function newWindowPanel(
  token: string,
  x: number,
  y: number,
  title: ComponentWords,
): Partial<UiPanel> {
  const flag = openFlagOf(token);
  const titleControl: UiPanelControl = {
    kind: "text",
    name: "title",
    x: 24,
    y: 18,
    w: 300,
    h: 32,
    font: "hoi_24header",
    text: title.en,
  };
  if (title.locale !== "en" && title.own)
    titleControl.text_locs = { [title.locale]: title.own };
  return {
    token,
    context_type: "player_context",
    parent_window_token: "",
    visible_trigger: openTrigger(flag),
    window_props: {
      x: Math.round(x),
      y: Math.round(y),
      width: WINDOW_LOOK.w,
      height: WINDOW_LOOK.h,
      background_sprite: WINDOW_LOOK.sprite,
      moveable: true,
    },
    controls: [
      titleControl,
      {
        kind: "button",
        name: "close",
        x: -42,
        y: 9,
        w: 32,
        h: 33,
        sprite: "GFX_closebutton",
        text: "",
        click_effect: closeFlagEffect(flag),
        click_enabled_trigger: "",
        raw_extras: { orientation: "UPPER_RIGHT" },
      },
    ],
  };
}

export function controlByGuiName(
  panels: readonly UiPanel[],
  guiName: string,
): { panel: UiPanel; control: UiPanelControl } | null {
  for (const panel of panels) {
    const prefix = `${controlNameSlug(panel.token) || "panel"}_`;
    if (!guiName.startsWith(prefix)) continue;
    const walk = (
      controls: readonly UiPanelControl[],
    ): UiPanelControl | null => {
      for (const control of controls) {
        if (
          `${prefix}${controlNameSlug(control.name) || "control"}` === guiName
        )
          return control;
        const inner = control.children ? walk(control.children) : null;
        if (inner) return inner;
      }
      return null;
    };
    const control = walk(panel.controls ?? []);
    if (control) return { panel, control };
  }
  return null;
}

export function fitFor(
  panel: { parent_window_token?: string },
  windowProps: Record<string, unknown>,
  controls: UiPanelControl[],
): PanelPlace {
  return isWindowPanel(panel)
    ? { window_props: windowProps, controls }
    : fitPanel(windowProps, controls);
}

export function placeComponent(
  panel: Pick<UiPanel, "window_props" | "controls"> & {
    token?: string;
    parent_window_token?: string;
  },
  preset: ComponentPreset,
  x: number,
  y: number,
  text: ComponentWords,
  scrollOf: ScrollOf = NO_SCROLL,

  size?: { w: number; h: number },
): PanelPlace & { path: string } {
  const left = Number(panel.window_props?.x) || 0;
  const top = Number(panel.window_props?.y) || 0;
  const controls = panel.controls ?? [];
  const { w, h } = size ?? PRESETS[preset].look;
  const sized = (control: UiPanelControl): UiPanelControl =>
    size ? { ...control, w: Math.round(w), h: Math.round(h) } : control;

  const inside = containerAt(
    controls,
    x - left + w / 2,
    y - top + h / 2,
    { w, h },
    scrollOf,
  );
  if (inside) {
    const at = inSlot(
      x - left - inside.x,
      y - top - inside.y,
      w,
      h,
      inside.slot,
    );
    const added = sized(
      newComponent(preset, at.x, at.y, controls, text, panel.token),
    );
    const [next, path] = appendControl(controls, inside.path, added);
    return { ...fitFor(panel, panel.window_props ?? {}, next), path };
  }
  const added = sized(
    newComponent(preset, x - left, y - top, controls, text, panel.token),
  );
  const fitted = fitFor(panel, panel.window_props ?? {}, [...controls, added]);
  return { ...fitted, path: String(controls.length) };
}

export function containerAt(
  controls: UiPanelControl[],
  x: number,
  y: number,
  fits?: { w: number; h: number },
  scrollOf: ScrollOf = NO_SCROLL,
): {
  path: string;
  x: number;
  y: number;
  slot?: { w: number; h: number };
} | null {
  let found: {
    path: string;
    x: number;
    y: number;
    slot?: { w: number; h: number };
  } | null = null;
  const walk = (
    list: UiPanelControl[],
    prefix: string,
    ox: number,
    oy: number,
  ) => {
    list.forEach((control, i) => {
      if (control.kind !== "container") return;
      const cx = ox + control.x;
      const cy = oy + control.y;
      const { w, h } = extent(control);
      if (x < cx || y < cy || x >= cx + w || y >= cy + h) return;
      const path = prefix ? `${prefix}/${i}` : String(i);
      if (control.array) {
        const slot = listSlot(control);
        const col = Math.floor((x - cx) / slot.w);
        const row = Math.floor((y - cy) / slot.h);
        if (
          col >= slot.columns ||
          (fits && (fits.w > slot.w || fits.h > slot.h))
        )
          return;
        found = {
          path,
          x: cx + col * slot.w,
          y: cy + row * slot.h,
          slot: { w: slot.w, h: slot.h },
        };
        walk(
          control.children ?? [],
          path,
          cx + col * slot.w,
          cy + row * slot.h,
        );
        return;
      }
      if (fits && (fits.w > w || (!control.scroll && fits.h > h))) return;
      const from = cy - scrollOf(control);
      found = { path, x: cx, y: from };
      walk(control.children ?? [], path, cx, from);
    });
  };
  walk(controls, "", 0, 0);
  return found;
}

export function moveControl(
  panel: Pick<UiPanel, "window_props" | "controls"> & {
    parent_window_token?: string;
  },
  path: string,
  to: { x?: number; y?: number; width?: number; height?: number },
): PanelPlace {
  const indices = path.split("/").map(Number);
  const walk = (list: UiPanelControl[], depth: number): UiPanelControl[] =>
    list.map((control, i) => {
      if (i !== indices[depth]) return control;
      if (depth < indices.length - 1)
        return {
          ...control,
          children: walk(control.children ?? [], depth + 1),
        };
      const moved = { ...control };
      if (to.x !== undefined) moved.x = Math.round(to.x);
      if (to.y !== undefined) moved.y = Math.round(to.y);
      if (to.width !== undefined) moved.w = Math.max(1, Math.round(to.width));
      if (to.height !== undefined) moved.h = Math.max(1, Math.round(to.height));
      return moved;
    });
  return fitFor(panel, panel.window_props ?? {}, walk(panel.controls ?? [], 0));
}

function parentCorner(
  controls: UiPanelControl[],
  path: string,
  scrollOf: ScrollOf,
): { x: number; y: number } {
  const indices = path.split("/").map(Number).slice(0, -1);
  let x = 0;
  let y = 0;
  let level: UiPanelControl[] | undefined = controls;
  for (const i of indices) {
    const control: UiPanelControl | undefined = level?.[i];
    if (!control) break;
    x += control.x;
    y += control.y - scrollOf(control);
    level = control.children;
  }
  return { x, y };
}

export function relocateControl(
  panel: Pick<UiPanel, "window_props" | "controls"> & {
    parent_window_token?: string;
  },
  path: string,
  to: { x?: number; y?: number; width?: number; height?: number },
  scrollOf: ScrollOf = NO_SCROLL,
): PanelPlace & { path: string } {
  const controls = panel.controls ?? [];
  const control = getControlAtPath(controls, path);
  if (!control || (to.x === undefined && to.y === undefined))
    return { ...moveControl(panel, path, to), path };
  const corner = parentCorner(controls, path, scrollOf);
  const size = extent({
    ...control,
    w: to.width ?? control.w,
    h: to.height ?? control.h,
  });
  const at = {
    x: corner.x + Math.round(to.x ?? control.x),
    y: corner.y + Math.round(to.y ?? control.y),
  };
  const parentPath = path.split("/").slice(0, -1).join("/") || null;
  const rest = removeControlAtPath(controls, path);
  const target = containerAt(
    rest,
    at.x + size.w / 2,
    at.y + size.h / 2,
    size,
    scrollOf,
  );

  const local = inSlot(
    at.x - (target?.x ?? 0),
    at.y - (target?.y ?? 0),
    size.w,
    size.h,
    target?.slot,
  );
  if ((target?.path ?? null) === parentPath)
    return { ...moveControl(panel, path, { ...to, ...local }), path };
  const moved: UiPanelControl = {
    ...control,
    ...local,
    ...(to.width !== undefined ? { w: Math.max(1, Math.round(to.width)) } : {}),
    ...(to.height !== undefined
      ? { h: Math.max(1, Math.round(to.height)) }
      : {}),
  };
  const [next, nextPath] = appendControl(rest, target?.path ?? null, moved);
  return { ...fitFor(panel, panel.window_props ?? {}, next), path: nextPath };
}

export interface Box {
  x: number;
  y: number;
  w: number;
  h: number;
}

export interface Guide {
  axis: "x" | "y";
  at: number;
  from: number;
  to: number;
}

export const SCREEN_EN_NAMES: Record<string, string> = {
  politics_tab: "Politics",
  decision_tab: "Decisions",
  technology_tab: "Research",
  diplomacy_tab: "Diplomacy",
  trade_tab: "Trade",
  construction_tab: "Construction",
  production_tab: "Production",
  deployment_tab: "Deployment",
  logistics_tab: "Logistics",
  national_focus: "National focus",
  selected_state_view: "State",
  top_bar: "Top bar",
};

export function panelNames(
  english: string,
  locale: string,
  own: string,
): Pick<UiPanel, "name_en" | "name_zh" | "display_name_locs"> {
  return {
    name_en: english,
    name_zh: locale === "zh" ? own : "",
    display_name_locs:
      locale === "en" ? { en: english } : { en: english, [locale]: own },
  };
}

export function renamedPanel(
  panel: Pick<UiPanel, "display_name_locs">,
  locale: string,
  name: string,
): Pick<UiPanel, "display_name_locs"> &
  Partial<Pick<UiPanel, "name_en" | "name_zh">> {
  return {
    display_name_locs: { ...(panel.display_name_locs ?? {}), [locale]: name },
    ...(locale === "en"
      ? { name_en: name }
      : locale === "zh"
        ? { name_zh: name }
        : {}),
  };
}

export function controlPathByName(
  controls: UiPanelControl[],
  name: string,
): string | null {
  const walk = (list: UiPanelControl[], prefix: string): string | null => {
    for (let i = 0; i < list.length; i += 1) {
      const path = prefix ? `${prefix}/${i}` : String(i);
      if (list[i].name === name) return path;
      const inner = list[i].children ? walk(list[i].children!, path) : null;
      if (inner) return inner;
    }
    return null;
  };
  return walk(controls, "");
}

export function boxesOf(
  controls: UiPanelControl[],
  scrollOf: ScrollOf = NO_SCROLL,
): (Box & { path: string; kind: UiPanelControl["kind"] })[] {
  const out: (Box & { path: string; kind: UiPanelControl["kind"] })[] = [];
  const walk = (
    list: UiPanelControl[],
    prefix: string,
    ox: number,
    oy: number,
  ) => {
    list.forEach((control, i) => {
      const path = prefix ? `${prefix}/${i}` : String(i);
      const { w, h } = extent(control);
      out.push({
        path,
        kind: control.kind,
        x: ox + control.x,
        y: oy + control.y,
        w,
        h,
      });
      if (control.children)
        walk(
          control.children,
          path,
          ox + control.x,
          oy + control.y - scrollOf(control),
        );
    });
  };
  walk(controls, "", 0, 0);
  return out;
}

export function snapBox(
  box: Box,
  others: readonly Box[],
  mode: "move" | "resize",
  threshold = 6,
): Box & { guides: Guide[] } {
  const out = { ...box };
  const guides: Guide[] = [];
  for (const axis of ["x", "y"] as const) {
    const len = axis === "x" ? "w" : "h";

    const own = mode === "move" ? [0, 0.5, 1] : [1];
    const theirs = mode === "move" ? [0, 0.5, 1] : [0, 1];
    let best: { diff: number; at: number; other: Box } | null = null;
    for (const other of others) {
      for (const f of own) {
        const line = box[axis] + box[len] * f;
        for (const g of theirs) {
          const at = other[axis] + other[len] * g;
          const diff = at - line;
          if (
            Math.abs(diff) <= threshold &&
            (!best || Math.abs(diff) < Math.abs(best.diff))
          )
            best = { diff, at, other };
        }
      }
    }
    if (!best) continue;
    if (mode === "move") out[axis] = Math.round(box[axis] + best.diff);
    else out[len] = Math.max(1, Math.round(box[len] + best.diff));
    const cross = axis === "x" ? "y" : "x";
    const crossLen = axis === "x" ? "h" : "w";
    guides.push({
      axis,
      at: Math.round(best.at),
      from: Math.min(out[cross], best.other[cross]),
      to: Math.max(
        out[cross] + out[crossLen],
        best.other[cross] + best.other[crossLen],
      ),
    });
  }
  return { ...out, guides };
}

export function controlPathOf(
  controls: UiPanelControl[],
  namePath: string,
): string | null {
  const indices: number[] = [];
  let level: UiPanelControl[] | undefined = controls;
  for (const name of namePath.split("/").filter(Boolean)) {
    const at: number =
      level?.findIndex((control) => control.name === name) ?? -1;
    if (!level || at < 0) return null;
    indices.push(at);
    level = level[at].children;
  }
  return indices.length ? indices.join("/") : null;
}

export function pickedControlNames(
  envelope: { controls: Record<string, { path: string }> },
  nodeName: string | null | undefined,
): string | null {
  if (!nodeName) return null;
  for (const [namePath, control] of Object.entries(envelope.controls)) {
    if (namePath === nodeName || namePath.endsWith(`/${nodeName}`))
      return control.path;
  }
  return null;
}

export function screenPanelToken(
  tag: string | null | undefined,
  mount: string,
): string {
  const prefix = (tag ?? "").toUpperCase().replace(/[^A-Z0-9]/g, "") || "MOD";
  return `${prefix}_${mount}_ui`;
}
