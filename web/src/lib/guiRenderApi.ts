import { trackGuiRead } from "./guiLoadActivity";
import {
  spriteUploadFields,
  type SpriteUploadOptions,
} from "./spriteUploadGeometry";
import { SpriteUploadError } from "./spriteUploadError";

export { SpriteUploadError };

export interface GuiCharacter {
  token: string;
  name: string;
  roles: string[];
  slot?: string;

  idea_token?: string;

  hired?: boolean;
  ideology?: string;
  portrait?: string;
  portrait_sprite?: GuiSprite;

  cost?: number;

  traits?: { token: string; label?: string }[];

  shown?: boolean;

  available?: boolean;

  tooltip?: string;
}

export interface GuiIdeaSlot {
  name: string;
  label: string;
  character: boolean;

  select_text?: string;
}

export interface GuiIdeaCategory {
  name: string;
  label: string;

  index: number;
  slots: GuiIdeaSlot[];
}

export interface GuiCharacters {
  tag: string;
  leader: GuiCharacter | null;
  advisors: GuiCharacter[];
  commanders: GuiCharacter[];
  categories: GuiIdeaCategory[];

  spirits: GuiPreviewIcon[];

  starting_ideas?: string[];

  slot_ideas?: Record<string, GuiPreviewIcon[]>;
}

export interface GuiFlag {
  file: string;
  width: number;
  height: number;
  url: string;
}

export interface GuiPreviewIcon {
  token: string;
  label?: string;
  icon?: string;
  icon_sprite?: GuiSprite;

  tooltip?: string;
}

export interface GuiCountryCard {
  tag: string;
  name: string;

  name_key?: string;
  ideology?: string;
  ideology_label?: string;
  flag?: GuiFlag;
  ruling_party?: string;
  ruling_party_label?: string;
  ruling_party_share?: number;
  elections_allowed?: boolean;
  history?: string;
  history_key?: string;
  ideas?: GuiPreviewIcon[];
  focuses?: GuiPreviewIcon[];
  leader?: GuiCharacter;

  focus_tree?: string;

  continent?: string;
  continent_label?: string;

  placeholder?: boolean;
}

export interface GuiBookmark {
  name: string;
  label?: string;
  desc?: string;

  desc_key?: string;
  date: string;
  default_country: string;
  picture?: string;
  picture_sprite?: GuiSprite;

  filters?: string[];

  continents?: { name: string; label?: string }[];
  countries: GuiCountryCard[];
}

export interface GuiBookmarks {
  bookmarks: GuiBookmark[];

  minors: GuiCountryCard[];
}

export interface GuiRegimentCount {
  type: string;
  label?: string;
  count: number;
}

export interface GuiDivisionTemplate {
  name: string;

  icon?: string;

  priority: number;
  width: number;
  regiments: GuiRegimentCount[];
  support: GuiRegimentCount[];

  deployed?: number;
}

export interface GuiDivisions {
  tag: string;

  oob: string[];
  templates: GuiDivisionTemplate[];
}

export interface GuiDesignerCell {
  type: string;
  label?: string;
  abbreviation?: string;

  icon?: string;

  x: number;
  y: number;
}

export interface GuiDesigner {
  tag: string;
  name: string;

  icon?: string;

  names_group?: string;

  division_names_group?: string;

  priority?: number;
  templates: string[];
  regiments: GuiDesignerCell[];
  support: GuiDesignerCell[];
  regimental_support?: GuiDesignerCell[];
  stats: Record<string, number>;
  equipment: {
    archetype: string;
    amount: number;
    model?: string;
    label?: string;

    short_label?: string;
  }[];

  resources: { resource: string; amount: number; frame?: number }[];

  production_cost?: { min: number; max: number };
  year: number;
  is_special: boolean;

  new?: boolean;

  header_stats?: Record<string, number> | null;

  names_groups?: GuiNamesGroup[];

  template_icons?: Record<string, string>;

  terrain?: GuiDesignerTerrain[];

  equipment_types?: GuiDesignerEquipmentGroup[];

  is_army_hq?: boolean;
  commander_bonuses?: string[];
  command_power?: number;
  deploy_days?: number;

  available?: GuiDesignerUnit[];

  group_titles?: Record<string, string>;

  army_experience?: number;

  costs?: GuiDesignerCosts;
}

export interface GuiDesignerUnit {
  type: string;
  label?: string;

  group: string;
  icon?: string;
  special_force?: boolean;

  paradrop?: boolean;

  divisional?: boolean;

  regimental?: boolean;

  priority?: number;

  deployment_cost?: number;

  commander_bonuses?: string[];
  abilities?: string[];
}

export interface GuiDesignerSaveInput {
  tag: string;
  date?: string;
  template: string;
  name: string;
  names_group: string;
  priority?: number;
  regiments: { type: string; x: number; y: number }[];
  support: { type: string; x: number; y: number }[];
  regimental_support?: { type: string; x: number; y: number }[];
}

export interface GuiDesignerSaveResult {
  template: { id: number; template_name: string; oob_file: string };
  oob_file: string;

  imported: boolean;

  started_oob?: boolean;
}

export interface GuiDesignerEquipmentGroup {
  archetype: string;
  label: string;
  types: { type: string; label: string; allowed: number; total: number }[];
}

export interface GuiDesignerTerrain {
  terrain: string;
  attack: number;
  defence: number;
  movement: number;
}

export interface GuiNamesGroup {
  key: string;
  name: string;
  examples: string[];
  more?: boolean;
  fallback?: string;
}

export interface GuiDesignerCosts {
  group: number;
  change: number;
  support: number;
}

export interface GuiCountryStats {
  tag: string;
  name?: string;
  flag?: GuiFlag;
  political_power?: number;
  command_power?: number;
  convoys?: number;
  research_slots?: number;
  stability?: number;
  war_support?: number;

  population?: number;
  manpower?: number;
  conscription?: number;
  civilian_factories?: number;
  military_factories?: number;
  dockyards?: number;
  resources?: Record<string, number>;

  resource_surplus?: Record<string, number>;

  resource_balance?: Record<string, Record<string, number>>;

  factories_total?: number;

  fuel?: GuiFuelStats;

  supply_ratio?: number;

  world_tension?: number;

  decisions_takeable?: { id: string; name: string }[];
  decisions_takeable_count?: number;

  faction?: GuiFactionStats | null;

  politics?: GuiPoliticsStats;

  popularities?: GuiPopularity[];
  army_experience?: number;
  navy_experience?: number;
  air_experience?: number;
  nukes?: number;
}

export interface GuiFuelStats {
  days: number | null;

  ratio: number;
  capacity: number;
  gaining: boolean;
}

export interface GuiFactionStats {
  name: string;

  name_key?: string;

  icon: string;
  initiative: number;
}

export interface GuiPopularity {
  party: string;

  label?: string;

  label_key?: string;
  value: number;
  color: [number, number, number];
}

export interface GuiPoliticsStats {
  ruling_party: string;

  ideology: string;

  elections?: string;

  elections_key?: string;
  next_election?: string;
}

export interface GuiAlert {
  name: string;

  frame: number;

  instant: string;

  delayed: string;
}

export interface GuiAlerts {
  sprite: string;
  alerts: GuiAlert[];
}

export interface GuiEventChoice {
  id: string;
  title: string;
  group?: "country" | "generic";
}

export interface GuiFocusTreeChoice {
  id: string;
  score: number;
  focuses: number;
}

export interface GuiSprite {
  resolved: boolean;
  procedural?: boolean;
  frames: number;
  width: number;
  height: number;
  effect: string | null;
  effect_kind: string | null;
  block_type?: string | null;
  cornered: boolean;
  bordersize: { x: number; y: number } | null;
  missing_file?: boolean;
  overridden?: boolean;
  override_path?: string | null;

  vanilla?: boolean;

  vanilla_url?: string;

  version?: string;
}

export type TextKind = "loc" | "runtime" | "key" | "literal";

export interface GuiText {
  display: string;
  kind: TextKind;

  markup?: string;
}

export interface GuiNode {
  type: string;
  name: string | null;
  position: { x: number | string; y: number | string };
  size: {
    width?: number | string;
    height?: number | string;
    x?: number | string;
    y?: number | string;
  } | null;
  orientation: string | null;
  origo: string | null;
  scale: number | null;
  sprite: string | null;
  frame: number | null;
  text: string | null;
  text_source: "text" | "buttontext" | null;
  font: string | null;
  format: string | null;

  format_token?: string | null;
  maxwidth: number | null;
  maxheight: number | null;
  text_color: string | null;
  centerposition: boolean;
  alwaystransparent: boolean;
  background: GuiNode | null;
  children: GuiNode[];
  raw?: Record<string, unknown>;
}

export interface GuiBitmapFont {
  line: number;
  base: number;
  atlas: string;
  atlas_w: number;
  atlas_h: number;

  tint: "alpha" | "luminance";

  color?: string;

  opacity?: number;

  chars: Record<string, number[]>;

  kerning?: Record<string, number>;
}

export interface GuiFontMeta {
  family: string | null;
  size: number;
  outline: boolean;
  line_ratio: number;

  bitmap?: GuiBitmapFont;
}

export interface OverlapMember {
  id: string;
  name: string;
  kind: "tab" | "overlay" | "state";
}
export interface OverlapGroup {
  members: OverlapMember[];
  anchor_rect: { x: number; y: number; w: number; h: number };
  default: string;
}

export interface GuiWindow {
  name: string;
  file: string;
  source?: string;

  scaffold_hash?: string;
  current_scaffold_hash?: string;
  revision?: string;

  ui_panel?: GuiUIPanelEnvelope;
  node: GuiNode;
  sprites: Record<string, GuiSprite>;
  texts: Record<string, GuiText>;
  fonts: Record<string, GuiFontMeta>;
  overlap_groups: OverlapGroup[];
}

export interface GuiUIPanelEnvelope {
  panel_id: number;
  token: string;
  revision: string;
  window_name: string;
  controls: Record<string, { name: string; path: string }>;
}

export type WindowCategory = "screen" | "entry" | "grid" | "fragment";
export interface WindowRow {
  name: string;
  file: string;
  source?: string;
  children: number;
  category: WindowCategory;
}

export interface GuiOverlayEdit {
  id: string;
  target: string;

  op:
    | "set_text"
    | "swap_sprite"
    | "set_layout"
    | "set_orientation"
    | "set_origo"
    | "hide_node"
    | "add_node"
    | "delete_node";
  payload: Record<string, unknown>;
}

export interface GuiOverlayResponse {
  ok: boolean;
  id?: number;
  project_id: number | null;
  file: string;
  source: string;
  scaffold_hash: string;
  schema_version: number;
  edits: GuiOverlayEdit[];
  exists?: boolean;

  revision: string;

  materializable?: boolean;

  scaffold_hash_pinned?: boolean;
  current_scaffold_hash?: string;
  scaffold_drift?: boolean;
}

export interface InjectionPoint {
  window_name: string;
  extensible: boolean;
  entry_container: string | null;
  array_name: string | null;
  gate: string[];
  member_add_effects: string[];

  member_add_ideas?: string[];
  member_recipe?: string | null;
  dynamic_lists?: Array<{
    entry_container: string | null;
    array_name: string | null;
  }>;
}

export interface InjectionPointsResponse {
  injection_points: InjectionPoint[];
}

let projectId: number | null = null;

let gameLanguage = "";

export function setGameLanguage(locale: string | null | undefined): void {
  const code = (locale ?? "").trim();
  gameLanguage = code && code !== "en" ? code : "";
}

export function gameLocale(): string {
  return gameLanguage || "en";
}

const GAME_LANGUAGE_NAMES: Record<string, string> = {
  en: "english",
  zh: "simp_chinese",
  ru: "russian",
  ja: "japanese",
  ko: "korean",
  es: "spanish",
  fr: "french",
  de: "german",
  pl: "polish",
  pt: "braz_por",
};

export function gameLanguageName(locale: string = gameLocale()): string {
  return (
    GAME_LANGUAGE_NAMES[locale.toLowerCase().split(/[-_]/)[0]] ?? "english"
  );
}

export function withGameLanguage(url: string): string {
  if (!gameLanguage) return url;
  return `${url}${url.includes("?") ? "&" : "?"}lang=${encodeURIComponent(gameLanguage)}`;
}

function readBase() {
  return `/api/go/projects/${projectId}/gui-render`;
}

export interface GuiRenderStatus {
  mod_root: string | null;
  project_id?: number | null;
  scope?: string;
  extra_roots?: string[];
  vanilla_root?: string | null;
  resolved: boolean;
  windows: number;
  sprites: number;
  extra_sprites?: number;
  vanilla_sprites?: number;
  loc_language?: string;
}

async function jget<T>(path: string): Promise<T> {
  return trackGuiRead(
    (async () => {
      const res = await fetch(withGameLanguage(`${readBase()}${path}`));
      if (!res.ok) throw new Error(`${res.status} ${res.statusText}`);
      return res.json() as Promise<T>;
    })(),
  );
}

export const guiRenderApi = {
  setProjectId: (id: number | null) => {
    projectId = id && Number.isFinite(id) && id > 0 ? id : null;
  },

  currentProjectId: (): number | null => projectId,

  status: () => jget<GuiRenderStatus>("/status"),

  dataVersion: () =>
    jget<{ version: string }>("/data-version").then((r) => r.version),

  refresh: async () => {
    const res = await fetch(`${readBase()}/refresh`, { method: "POST" });
    if (!res.ok) throw new Error(`refresh failed: ${res.status}`);
    return res.json() as Promise<GuiRenderStatus>;
  },

  listWindows: () =>
    jget<{ windows: WindowRow[] }>("/windows").then((r) => r.windows),

  getWindow: (name: string, file?: string) =>
    jget<GuiWindow>(
      `/window?name=${encodeURIComponent(name)}${file ? `&file=${encodeURIComponent(file)}` : ""}`,
    ),

  spriteUrl: (name: string, v?: number | string) =>
    `${readBase()}/sprite?name=${encodeURIComponent(name)}${v ? `&v=${v}` : ""}`,

  fontUrl: (family: string) => `${readBase()}/font?family=${family}`,

  focusTree: (id: string | null, tag?: string, date?: string | null) => {
    const query = new URLSearchParams();
    if (id) query.set("id", id);
    query.set("tag", tag ?? "");
    if (date) query.set("date", date);
    return jget<unknown>(`/focus-tree?${query}`);
  },

  eventChoices: (q: string, tag?: string | null) =>
    jget<{ events: GuiEventChoice[] }>(
      `/event-choices?q=${encodeURIComponent(q)}${tag ? `&tag=${encodeURIComponent(tag)}` : ""}`,
    ).then((r) => r.events),

  eventChoicePage: (
    q: string,
    offset: number,
    limit: number,
    tag?: string | null,
    scope: "country" | "all" = "country",
  ) =>
    jget<{
      events: GuiEventChoice[];
      total: number;
      scope?: "country" | "all";
    }>(
      `/event-choices?q=${encodeURIComponent(q)}&offset=${offset}&limit=${limit}` +
        (tag ? `&tag=${encodeURIComponent(tag)}&scope=${scope}` : ""),
    ),

  focusTreeChoices: (tag: string) =>
    jget<{ trees: GuiFocusTreeChoice[] }>(
      `/focus-tree-choices?tag=${encodeURIComponent(tag)}`,
    ).then((r) => r.trees),

  characters: (tag: string, date?: string) =>
    jget<GuiCharacters>(
      `/characters?tag=${encodeURIComponent(tag)}${date ? `&date=${encodeURIComponent(date)}` : ""}`,
    ),

  bookmarks: () => jget<GuiBookmarks>("/bookmarks"),

  frontend: () =>
    jget<{
      backgrounds: {
        name: string;
        url: string;
        width: number;
        height: number;
        locale?: string;
        overridden?: boolean;
      }[];
      loading: {
        name: string;
        url: string;
        width: number;
        height: number;
        overridden?: boolean;
      }[];

      version?: string;
    }>("/frontend"),

  sprites: (names: string[]) =>
    jget<{ sprites: Record<string, GuiSprite> }>(
      `/sprites?names=${encodeURIComponent(names.join(","))}`,
    ).then((r) => r.sprites),

  countryStats: (tag: string, date?: string) =>
    jget<GuiCountryStats>(
      `/country-stats?tag=${encodeURIComponent(tag)}${date ? `&date=${encodeURIComponent(date)}` : ""}`,
    ),

  alerts: () => jget<GuiAlerts>("/alerts"),

  divisions: (tag: string, date?: string) =>
    jget<GuiDivisions>(
      `/divisions?tag=${encodeURIComponent(tag)}${date ? `&date=${encodeURIComponent(date)}` : ""}`,
    ),

  designer: (tag: string, date?: string, template?: string, cells?: string) =>
    jget<GuiDesigner>(
      `/designer?tag=${encodeURIComponent(tag)}` +
        (date ? `&date=${encodeURIComponent(date)}` : "") +
        (template ? `&template=${encodeURIComponent(template)}` : "") +
        (cells ? `&cells=${encodeURIComponent(cells)}` : ""),
    ),

  loc: (keys: string[], opts: { markup?: boolean } = {}) =>
    jget<{ texts: Record<string, string> }>(
      `/loc?keys=${encodeURIComponent(keys.join(","))}${opts.markup ? "&markup=1" : ""}`,
    ).then((r) => r.texts),

  textOverrides: () =>
    jget<{ overrides: Record<string, string> }>("/text").then(
      (r) => r.overrides,
    ),

  setText: async (key: string, value: string) => {
    const res = await fetch(withGameLanguage(`${readBase()}/text`), {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ key, value }),
    });
    if (!res.ok) {
      const body = (await res.json().catch(() => ({}))) as {
        detail?: string;
        code?: string;
        file?: string;
      };
      throw Object.assign(
        new Error(body.detail || `save text failed: ${res.status}`),
        {
          status: res.status,
          code: body.code,
          file: body.file,
        },
      );
    }
    return res.json() as Promise<{ key: string; value: string }>;
  },

  saveDesigner: async (input: GuiDesignerSaveInput) => {
    const res = await fetch(`${readBase()}/designer-save`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(input),
    });
    const body = (await res.json().catch(() => ({}))) as { detail?: string };
    if (!res.ok) throw new Error(body.detail || `save failed: ${res.status}`);
    return body as unknown as GuiDesignerSaveResult;
  },

  takeOOB: async (tag: string, date?: string | null) => {
    const res = await fetch(`${readBase()}/oob-take`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ tag, ...(date ? { date } : {}) }),
    });
    const body = (await res.json().catch(() => ({}))) as {
      detail?: string;
      oob_file?: string;
      imported?: boolean;
    };
    if (!res.ok || !body.oob_file)
      throw new Error(body.detail || `take failed: ${res.status}`);
    return { oob_file: body.oob_file, imported: !!body.imported };
  },

  clearText: async (key: string) => {
    const res = await fetch(
      withGameLanguage(`${readBase()}/text?key=${encodeURIComponent(key)}`),
      { method: "DELETE" },
    );
    if (!res.ok) throw new Error(`restore text failed: ${res.status}`);
    return res.json() as Promise<{ key: string; removed: boolean }>;
  },

  override: async (
    name: string,
    file: File,
    options?: SpriteUploadOptions,

    onProgress?: (percent: number) => void,

    create?: { width: number; height: number; frames: number },
  ) => {
    const fd = new FormData();
    fd.append("name", name);
    for (const [key, value] of Object.entries(spriteUploadFields(options)))
      fd.append(key, value);
    if (create) {
      fd.append("create", "1");
      fd.append("width", String(create.width));
      fd.append("height", String(create.height));
      fd.append("frames", String(create.frames));
    }
    fd.append("file", file);
    const url = `${readBase()}/sprite/override`;
    const { status, body } = onProgress
      ? await postWithProgress(url, fd, onProgress)
      : await fetch(url, { method: "POST", body: fd }).then(async (res) => ({
          status: res.status,
          body: (await res.json().catch(() => ({}))) as SpriteOverrideBody,
        }));
    if (status < 200 || status >= 300) {
      const detail =
        typeof body.detail === "string" && body.detail.trim()
          ? body.detail
          : `override failed: ${status}`;
      throw new SpriteUploadError(status, detail);
    }
    return body;
  },

  revert: async (name: string) => {
    const res = await fetch(
      `${readBase()}/sprite/override?name=${encodeURIComponent(name)}`,
      {
        method: "DELETE",
      },
    );
    const body = (await res.json().catch(() => ({}))) as {
      ok: boolean;
      name: string;
      resolved: boolean;
      width: number;
      height: number;
      detail?: unknown;
    };
    if (!res.ok) {
      throw new Error(
        typeof body.detail === "string" && body.detail.trim()
          ? body.detail
          : `revert failed: ${res.status}`,
      );
    }
    return body;
  },

  loadOverlay: async (file: string): Promise<GuiOverlayResponse> => {
    const res = await fetch(
      `${readBase()}/overlay?file=${encodeURIComponent(file)}`,
    );
    if (!res.ok) throw new Error(`load overlay failed: ${res.status}`);
    return res.json() as Promise<GuiOverlayResponse>;
  },

  saveOverlay: async (payload: {
    file: string;
    edits: GuiOverlayEdit[];
    scaffold_hash: string;
    expected_revision: string;
    expected_scaffold_hash: string;
    source?: string;
    schema_version?: number;
  }): Promise<GuiOverlayResponse> => {
    const res = await fetch(`${readBase()}/overlay`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(payload),
    });
    if (!res.ok) {
      const body = (await res.json().catch(() => null)) as {
        detail?: string;
      } | null;

      throw Object.assign(
        new Error(body?.detail || `save overlay failed: ${res.status}`),
        { status: res.status },
      );
    }
    return res.json() as Promise<GuiOverlayResponse>;
  },

  getInjectionPoints: (): Promise<InjectionPointsResponse> =>
    fetch(`${readBase()}/scripted-gui-injection-points`).then(async (res) => {
      if (!res.ok) throw new Error(`${res.status} ${res.statusText}`);
      return res.json() as Promise<InjectionPointsResponse>;
    }),

  exportBundle: async (file: string): Promise<Blob> => {
    const res = await fetch(
      `${readBase()}/export?file=${encodeURIComponent(file)}`,
      {
        method: "POST",
      },
    );
    if (!res.ok) throw new Error(`export failed: ${res.status}`);
    return res.blob();
  },
};

interface SpriteOverrideBody {
  ok: boolean;
  name: string;
  width: number;
  height: number;
  override_path?: string | null;

  export?: { gfx?: string; texture: string };
  export_error?: string;
  detail?: unknown;
}

function postWithProgress(
  url: string,
  body: FormData,
  onProgress: (percent: number) => void,
): Promise<{ status: number; body: SpriteOverrideBody }> {
  return new Promise((resolve, reject) => {
    const xhr = new XMLHttpRequest();
    xhr.open("POST", url);
    xhr.upload.onprogress = (event) => {
      if (event.lengthComputable && event.total > 0)
        onProgress(
          Math.min(99, Math.floor((event.loaded / event.total) * 100)),
        );
    };
    xhr.upload.onload = () => onProgress(100);
    xhr.onload = () => {
      let parsed = {} as SpriteOverrideBody;
      try {
        parsed = JSON.parse(xhr.responseText || "{}") as SpriteOverrideBody;
      } catch {}
      resolve({ status: xhr.status, body: parsed });
    };
    xhr.onerror = () => reject(new SpriteUploadError(0, "network error"));
    xhr.send(body);
  });
}
