import {
  gameLanguageName,
  type GuiNode,
  type GuiSprite,
  type GuiWindow,
} from "@/lib/guiRenderApi";
import { assignIds } from "../gui-renderer/GuiCanvas";
import { matchNamePath } from "./gameNavigation";

export const MAIN_MENU = "mainmenu";
export const MENU_BACKGROUND = "frontend_background";
export const MENU_UPPER_LEFT = "mainmenu_panel_upperleft";
export const MENU_UPPER_RIGHT = "mainmenu_panel_upperright";
export const MENU_MAIN_PANEL = "mainmenu_panel_bottom";
export const MENU_SINGLE_PLAYER = "mainmenu_single_player";
export const MENU_SOCIAL = "mainmenu_social_buttons";
export const MENU_CHANGE_BACKGROUND = "change_background";
export const LOAD_SCREEN = "load_screen";

export type MenuAction =
  | { kind: "single-player" }
  | { kind: "back" }
  | { kind: "new-game" }
  | { kind: "continue" }
  | { kind: "exit" }
  | { kind: "not-built"; button: string; named: boolean };

const ACTIONS: Readonly<Record<string, MenuAction>> = {
  single_player_button: { kind: "single-player" },
  sp_back_button: { kind: "back" },
  sp_new_button: { kind: "new-game" },
  sp_continue_button: { kind: "continue" },
  exit_button: { kind: "exit" },
};

const NAMED = new Set([
  "multi_player_button",
  "settings_button",
  "career_profile_button",
  "friends_button",
  "credits_button",
  "sp_load_button",
  "sp_tutorial_button",
  "nudge_button",
  "change_background_button",
  "show_subscription",
  "dlc_button",
]);

export function menuActionOf(node: GuiNode): MenuAction {
  const name = node.name ?? "";
  return (
    ACTIONS[name] ?? { kind: "not-built", button: name, named: NAMED.has(name) }
  );
}

const MENU_HIDDEN = new Set([
  "single_player_button_warn",
  "career_profile_button_star",
  "sp_continue_flag_left",
  "sp_continue_flag_left_overlay",
  "sp_continue_flag_right",
  "sp_continue_flag_right_overlay",
  "achievements_button",
  "social_view_interface_window",
  "background_selection",
]);

function idOf(node: GuiNode): string {
  return (node as GuiNode & { _id?: string })._id ?? "";
}

function pruned(node: GuiNode): GuiNode {
  return {
    ...node,
    children: (node.children ?? [])
      .filter((child) => !(child.name && MENU_HIDDEN.has(child.name)))
      .map(pruned),
  };
}

function positionOf(
  root: GuiNode,
  name: string,
): { x: number; y: number } | null {
  const node = (root.children ?? []).find((child) => child.name === name);
  if (!node) return null;
  return { x: Number(node.position?.x) || 0, y: Number(node.position?.y) || 0 };
}

export function mainPanelSize(win: GuiWindow): {
  width: number;
  height: number;
} {
  const grown = positionOf(win.node, "subscription_size");
  if (grown) return { width: grown.x, height: grown.y };
  return {
    width: Number(win.node.size?.width) || 320,
    height: Number(win.node.size?.height) || 450,
  };
}

export function mainMenuWindow(win: GuiWindow, version = ""): GuiWindow {
  const node = pruned(win.node);
  const texts = { ...win.texts };
  if (win.name === MENU_MAIN_PANEL) {
    node.size = mainPanelSize(win);
    const at = positionOf(win.node, "version_label_subscription_position");
    for (const child of node.children ?? []) {
      if (child.name !== "version_label") continue;
      if (at) child.position = at;
      if (version) {
        child.text = "__game_version";
        texts.__game_version = { kind: "runtime", display: `v${version}` };
      }
    }
  }
  return { ...win, node, texts };
}

export const SOCIAL_IN_PANEL = { x: 36, y: 477 };

export function socialWindow(
  win: GuiWindow,
  panel: { width: number; height: number },
  screen: { width: number; height: number },
): GuiWindow {
  const node = pruned(win.node);

  node.orientation = "UPPER_LEFT";
  node.origo = null;
  node.position = {
    x: Math.round((screen.width - panel.width) / 2) + SOCIAL_IN_PANEL.x,
    y: Math.round((screen.height - panel.height) / 2) + SOCIAL_IN_PANEL.y,
  };
  return { ...win, node };
}

export const MENU_DLC_COLUMN = "mainmenu_dlc_ownership";
export const MENU_DLC_ITEM = "owned_dlc_item";

export const MENU_DLCS = [
  "028",
  "031",
  "034",
  "036",
  "038",
  "040",
  "043",
  "046",
  "049",
  "051",
  "052",
] as const;

export function dlcColumnSprites(): string[] {
  return MENU_DLCS.flatMap((dlc) => [
    `GFX_frontend_banner_dlc${dlc}`,
    `GFX_frontend_medal_dlc${dlc}`,
  ]);
}

const DLC_ROW_PITCH = 51;

export function dlcColumnWindow(
  column: GuiWindow,
  item: GuiWindow,
  sprites: Readonly<Record<string, GuiSprite>>,
): GuiWindow {
  const node = structuredClone(column.node) as GuiNode;
  const grid = (node.children ?? []).find(
    (child) => child.type === "gridboxtype",
  );
  const left = Number(grid?.position?.x) || 0;
  const height = Number(node.size?.height) || 684;
  const shown = MENU_DLCS.filter(
    (dlc) => sprites[`GFX_frontend_medal_dlc${dlc}`]?.resolved !== false,
  );
  const rows = shown.map((dlc, at) => {
    const row = structuredClone(item.node) as GuiNode;
    row.orientation = "UPPER_LEFT";
    row.origo = null;
    row.position = { x: left, y: height - (shown.length - at) * DLC_ROW_PITCH };
    for (const part of row.children ?? []) {
      if (String(part.orientation ?? "").toUpperCase() === "CENTER_LEFT")
        part.orientation = "UPPER_LEFT";
      if (part.name === "dlc_button")
        part.sprite = `GFX_frontend_banner_dlc${dlc}`;
      if (part.name === "owned_tick") {
        part.sprite = `GFX_frontend_medal_dlc${dlc}`;
        part.frame = null;
      }
    }
    return row;
  });
  node.children = [
    ...(node.children ?? []).filter((child) => child !== grid),
    ...rows,
  ];
  assignIds(node);
  const merged: Record<string, GuiSprite> = {
    ...column.sprites,
    ...item.sprites,
  };
  for (const name of dlcColumnSprites())
    if (sprites[name]) merged[name] = sprites[name];
  return { ...column, node, sprites: merged };
}

export interface FrontendPicture {
  name: string;
  url: string;
  width: number;
  height: number;

  locale?: string;

  overridden?: boolean;
}

export function defaultBackground(
  backgrounds: readonly FrontendPicture[],
  language: string = gameLanguageName(),
): FrontendPicture | undefined {
  return (
    backgrounds.find((picture) => picture.locale === language) ??
    backgrounds.find((picture) => !picture.locale) ??
    backgrounds[0]
  );
}

export function screenPictureSprite(name: string): string {
  return `GFX_${name}`;
}

export function menuBackgroundWindow(
  win: GuiWindow,
  backgrounds: readonly FrontendPicture[],
): GuiWindow {
  const name = win.node.background?.sprite ?? "";
  const sprite = name ? win.sprites[name] : undefined;
  const chosen = defaultBackground(backgrounds);
  if (!sprite || sprite.overridden || !chosen || !win.node.background)
    return win;
  const picture = screenPictureSprite(chosen.name);
  const swapped: GuiSprite = {
    ...sprite,
    vanilla_url: chosen.url,
    width: chosen.width,
    height: chosen.height,
    resolved: true,
    missing_file: false,
    overridden: Boolean(chosen.overridden),
  };
  const node = {
    ...win.node,
    background: { ...win.node.background, sprite: picture },
  };
  return { ...win, node, sprites: { ...win.sprites, [picture]: swapped } };
}

export function loadScreenWindow(
  win: GuiWindow,
  status: { key: string; text: string },
  tip: { key: string; text: string },
  screen: { width: number; height: number },
): { win: GuiWindow; barId: string } {
  const node = structuredClone(win.node) as GuiNode;
  node.size = { width: screen.width, height: screen.height };
  const statusText = matchNamePath(node, "status/text")[0];
  const tipText = matchNamePath(node, "tip/text")[0];
  if (statusText) statusText.text = status.key;
  if (tipText) tipText.text = tip.key;
  const bar = matchNamePath(node, "status/progressbar")[0];

  if (bar) bar.sprite = null;
  assignIds(node);
  const texts = { ...win.texts };
  if (status.key) texts[status.key] = { kind: "loc", display: status.text };
  if (tip.key) texts[tip.key] = { kind: "loc", display: tip.text };
  return { win: { ...win, node, texts }, barId: bar ? idOf(bar) : "" };
}

export const LOADING_STEPS = [
  "LOADING_DATABASES",
  "LOADING_GUI",
  "LOADING_MAP_SPRITES",
  "LOADING_HISTORY",
] as const;

export const LOADING_TIP_KEYS = Array.from(
  { length: 60 },
  (_, n) => `LOADING_TIP_${n}`,
);

export const LOADING_MS = 1800;

export function pickTip(
  loc: Readonly<Record<string, string>>,
  seed: number,
): { key: string; text: string } {
  const have = LOADING_TIP_KEYS.filter((key) => loc[key] && loc[key] !== key);
  if (!have.length) return { key: "", text: "" };
  const key = have[Math.min(have.length - 1, Math.floor(seed * have.length))];
  return { key, text: loc[key] };
}

export function loadingStepAt(
  progress: number,
): (typeof LOADING_STEPS)[number] {
  const at = Math.min(
    LOADING_STEPS.length - 1,
    Math.max(0, Math.floor(progress * LOADING_STEPS.length)),
  );
  return LOADING_STEPS[at];
}
