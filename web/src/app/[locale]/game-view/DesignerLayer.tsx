"use client";

import { Fragment, useMemo, useState } from "react";
import useSWR from "swr";
import {
  guiRenderApi,
  type GuiDesigner,
  type GuiNamesGroup,
  type GuiNode,
  type GuiSprite,
  type GuiWindow,
} from "@/lib/guiRenderApi";
import { fontTextColors } from "@/lib/referenceTables";
import { measureWindow, textAlignOf } from "../gui-renderer/GuiCanvas";
import { insetByMargin } from "../gui-renderer/windowMargins";
import { useTemplateWindow } from "./CountrySelectLayer";
import { BitmapText, measureLine } from "../gui-renderer/BitmapText";

import { matchNamePath } from "./gameNavigation";
import { frameStyle, pickable as artTag } from "./spriteFrame";
import { fillRuntimeSlots } from "../gui-renderer/tooltipText";
import { gameTooltipProps, hideGameTooltip } from "../gui-renderer/GameTooltip";
import type { TextSource } from "./editModel";
import { keySource } from "./textSources";
import { DesignerPicker } from "./DesignerPicker";
import { DesignerStatRows } from "./DesignerStatRows";
import { DesignerDropdown, type DropdownItem } from "./DesignerDropdowns";
import { DesignerTerrain } from "./DesignerTerrain";
import { CHECKED, DesignerEquipment } from "./DesignerEquipment";
import { RichText } from "../gui-renderer/RichText";
import { bitmapMeasure, layoutTooltip } from "../gui-renderer/tooltipText";
import {
  DELTA_KEYS,
  SLOT_KEYS,
  changeArrow,
  deltaTooltip,
  deploymentLine,
  hqPickerTooltip,
  lowerIsBetter,
  slotHints,
  slotTooltip,
} from "./designerPreview";
import {
  NEW_TEMPLATE,
  draftOf,
  encodeCells,
  pickerEntries,
  placed,
  regimentSlots,
  regimentalSupportSlots,
  column,
  filledRegiment,
  hqRegimentSlots,
  hqSupportSlots,
  removed,
  removedRegiment,
  swappedAlone,
  sameEntry,
  supportSlots,
  type DesignerDraft,
  type DesignerSlot,
  type PickerEntry,
  type PickerState,
} from "./designerEdit";

export function designerWindow(win: GuiWindow): GuiWindow {
  const node = structuredClone(win.node) as GuiNode;
  for (const picker of matchNamePath(node, "designerregimentswindow"))
    insetByMargin(picker);
  return { ...win, node };
}

const EQUIPMENT_LIST_MARGIN = 15;

const HQ_BONUS_WIDTH = 226;
const HQ_BONUS_PADDING = 6;

const NL = "\n";

export const DIVISION_DESIGNER = "countrydivisiondesignerview";

export function designerTextOverrides(
  win: GuiWindow,
  cost = 0,
  experience = 0,

  bar?: {
    data?: GuiDesigner;
    current?: GuiDesigner;
    texts?: Record<string, string>;
  },
): Record<string, string> {
  const out: Record<string, string> = {};
  if (win.name !== DIVISION_DESIGNER) return out;

  const { value, resources } = designerBarTexts(bar?.data, bar?.texts);

  const hq = bar?.current ?? bar?.data;
  if (hq?.is_army_hq) {
    const days = hq.deploy_days ?? 0;
    const daysText = bar?.texts?.[BAR_TEXTS.days]
      ? fillRuntimeSlots(bar.texts[BAR_TEXTS.days], { DAYS: days })
      : `${days} days`;
    for (const [name, text] of [
      ["hq_deploy_time", daysText],
      ["hq_deploy_cost", String(Math.round(hq.command_power ?? 0))],
    ] as const) {
      for (const node of matchNamePath(win.node, name)) {
        const id = (node as GuiNode & { _id?: string })._id;
        if (id) out[id] = text;
      }
    }
  }
  for (const [name, text] of [
    ["production_value", value],
    ["production_resources", resources],
  ] as const) {
    for (const node of matchNamePath(win.node, name)) {
      const id = (node as GuiNode & { _id?: string })._id;
      if (id) out[id] = text;
    }
  }
  for (const node of matchNamePath(win.node, "save_button")) {
    const markup = node.text ? win.texts[node.text]?.markup : undefined;
    const id = (node as GuiNode & { _id?: string })._id;
    if (markup && id) out[id] = fillRuntimeSlots(markup, { VALUE: cost });
  }

  for (const node of matchNamePath(win.node, "experience")) {
    const id = (node as GuiNode & { _id?: string })._id;
    if (id) out[id] = `£army_experience ${experience}`;
  }
  return out;
}

export function designerTextSources(
  win: GuiWindow,
): Record<string, TextSource> {
  const out: Record<string, TextSource> = {};
  if (win.name !== DIVISION_DESIGNER) return out;
  for (const node of matchNamePath(win.node, "save_button")) {
    const id = (node as GuiNode & { _id?: string })._id;
    const source = keySource(node.text, "everywhere");
    if (id && source) out[id] = source;
  }
  const days = keySource(BAR_TEXTS.days, "everywhere");
  for (const node of matchNamePath(win.node, "hq_deploy_time")) {
    const id = (node as GuiNode & { _id?: string })._id;
    if (id && days) out[id] = days;
  }
  return out;
}

const BAR_TEXTS = {
  range: "DIVISION_PRODUCTION_COST_VAL",
  single: "DIVISION_PRODUCTION_COST_SINGLE_VAL",
  duplicate: "DESIGNER_DUPLICATE_DESC",
  days: "HQ_TEMPLATE_DEPLOY_DAYS_SHORT",
} as const;

export function designerTooltipFor(
  node: { name?: string | null },
  texts: Record<string, string> | undefined,
) {
  if (node.name === "duplicate_button") return texts?.[BAR_TEXTS.duplicate];
  return undefined;
}

export function useDesignerBarTexts(projectId: number, enabled: boolean) {
  return useSWR<Record<string, string>>(
    enabled ? ["game-view-loc", projectId, "designer-bar"] : null,
    () => {
      guiRenderApi.setProjectId(projectId);
      return guiRenderApi.loc(Object.values(BAR_TEXTS), { markup: true });
    },
    { revalidateOnFocus: false },
  );
}

export function designerBarTexts(
  data: GuiDesigner | undefined,
  texts: Record<string, string> | undefined,
): { value: string; resources: string } {
  if (!data?.equipment.length || !data.production_cost)
    return { value: "", resources: "" };
  const low = Math.floor(data.production_cost.min);
  const high = Math.floor(data.production_cost.max);
  const range = texts?.[BAR_TEXTS.range];
  const single = texts?.[BAR_TEXTS.single];
  const value =
    low === high
      ? single
        ? fillRuntimeSlots(single, { VALUE: low })
        : ""
      : range
        ? fillRuntimeSlots(range, { MINVALUE: low, MAXVALUE: high })
        : "";
  const resources = data.resources
    .filter((item) => item.frame)
    .map((item) => `£resources_strip|${item.frame}`)
    .join(" ");
  return { value, resources };
}

const MAX_DIVISION_BRIGADE_WIDTH = 5;
const MAX_DIVISION_SUPPORT_WIDTH = 1;

const STAT_BOXES: readonly {
  node: string;
  stat: string;
  format?: StatFormat;
  key: string;
}[] = [
  {
    node: "hardness_stat",
    stat: "hardness",
    format: "percent",
    key: "STAT_ARMY_HARDNESS",
  },
  { node: "org_stat", stat: "max_organisation", key: "STAT_COMMON_MAX_ORG" },
  {
    node: "strength_stat",
    stat: "max_strength",
    key: "STAT_COMMON_MAX_STRENGTH",
  },
  {
    node: "speed_stat",
    stat: "speed",
    format: "speed",
    key: "STAT_COMMON_MAXIMUM_SPEED",
  },
];

export function statBoxTooltip(
  key: string,
  texts: Record<string, string>,
): string {
  const title = (texts[key] ?? "").replace(/:\s*$/, "").trim();
  const desc = texts[`${key}_DESC`];
  if (!desc) return "";
  return title ? `§Y${title}§!${NL}${desc}` : desc;
}

type StatRow = { key: string; stat: string; format?: StatFormat };
type StatFormat =
  "speed" | "percent" | "percent2" | "percent0" | "rate" | "whole";

const BASE_STATS: readonly StatRow[] = [
  { key: "STAT_COMMON_MAXIMUM_SPEED", stat: "speed", format: "speed" },
  { key: "STAT_COMMON_MAX_STRENGTH", stat: "max_strength" },
  { key: "STAT_COMMON_MAX_ORG", stat: "max_organisation" },
  { key: "STAT_ARMY_DEFAULT_MORALE", stat: "recovery_rate", format: "rate" },
  { key: "STAT_ARMY_RECON", stat: "recon" },
  { key: "STAT_ARMY_SUPRESSION", stat: "suppression" },
  { key: "STAT_COMMON_WEIGHT", stat: "weight" },
  { key: "STAT_ARMY_SUPPLY_CONSUMPTION", stat: "supply_use", format: "rate" },
  {
    key: "STAT_COMMON_RELIABILITY_AVERAGE",
    stat: "reliability",
    format: "percent",
  },
  {
    key: "STAT_COMMON_RELIABILITY_FACTOR",
    stat: "reliability_bonus",
    format: "percent",
  },
  {
    key: "STAT_ARMY_CASUALTY_TRICKLEBACK",
    stat: "trickleback",
    format: "percent2",
  },
  {
    key: "STAT_ARMY_EXPERIENCE_LOSS_FACTOR",
    stat: "experience_loss",
    format: "percent2",
  },
];

const COMBAT_STATS: readonly StatRow[] = [
  { key: "STAT_ARMY_SOFT_ATTACK", stat: "soft_attack" },
  { key: "STAT_ARMY_HARD_ATTACK", stat: "hard_attack" },
  { key: "STAT_AIR_ATTACK", stat: "air_attack" },
  { key: "STAT_ARMY_DEFENSE", stat: "defense" },
  { key: "STAT_ARMY_BREAKTHROUGH", stat: "breakthrough" },
  { key: "STAT_COMMON_ARMOR", stat: "armor_value" },
  { key: "STAT_COMMON_PIERCING", stat: "ap_attack" },
  { key: "STAT_ARMY_INITIATIVE", stat: "initiative", format: "percent2" },
  { key: "STAT_ARMY_ENTRENCHMENT", stat: "entrenchment" },
  {
    key: "STAT_ARMY_EQUIPMENT_CAPTURE_FACTOR",
    stat: "capture_ratio",
    format: "percent0",
  },

  { key: "DESIGNER_COMBATWIDTH", stat: "combat_width", format: "whole" },
];

const COST_STATS: readonly StatRow[] = [
  { key: "DESIGNER_MANPOWER", stat: "manpower", format: "whole" },
  { key: "DESIGNER_TRAINING_TIME", stat: "training_time", format: "whole" },
  { key: "STAT_ARMY_FUEL_CAPACITY", stat: "fuel_capacity" },

  { key: "STAT_COMMON_FUEL_CONSUMPTION", stat: "fuel_use", format: "rate" },
];

const NO_STAT = "DESIGNER_NO_STAT";

const NONE_KEY = "RULE_OPTION_NONE";

const LABEL_KEYS = [
  ...[...BASE_STATS, ...COMBAT_STATS, ...COST_STATS].map((row) => row.key),
  NO_STAT,
  NONE_KEY,
];

const CELL_BG = "GFX_div_unit_item_bg_wide";
const SUPPORT_BG = "GFX_div_subunit_item_bg";

const REGIMENT_ENTRY = "designer_division_regiment_slot_entry";
const SUPPORT_ENTRY = "designer_division_support_slot_entry";

const CONNECTOR = "rgb(99,106,89)";

const SLOT_ART: Record<Exclude<DesignerSlot["kind"], "unit">, string> = {
  add: "GFX_new_regiment_slot",
  locked: "GFX_locked_icon",
  blocked: "GFX_blocked_icon",
};
const CHROME = [
  CELL_BG,
  SUPPORT_BG,
  "GFX_div_unit_item_bg",
  ...Object.values(SLOT_ART),
];

export function designerKey(
  projectId: number,
  tag: string,
  date: string | null,
  template: string | null,
  cells?: string | null,
) {
  const name = template === NEW_TEMPLATE ? "-" : template;
  const composition = cells ?? (template === NEW_TEMPLATE ? "-" : null);
  return [
    "game-view-designer",
    projectId,
    tag,
    date,
    name,
    composition,
  ] as const;
}

export function useDesigner(
  projectId: number,
  tag: string | null,
  date: string | null,
  template: string | null,
  cells?: string | null,
) {
  const name = template === NEW_TEMPLATE ? "-" : template;
  const composition = cells ?? (template === NEW_TEMPLATE ? "-" : null);
  return useSWR<GuiDesigner>(
    tag ? designerKey(projectId, tag, date, template, cells) : null,
    () => {
      guiRenderApi.setProjectId(projectId);
      return guiRenderApi.designer(
        tag as string,
        date || undefined,
        name || undefined,
        composition || undefined,
      );
    },
    { revalidateOnFocus: false, keepPreviousData: true },
  );
}

const LIST_TEXTS = {
  create: "DESIGNER_CREATE_NEW_DIVISION",
  createTip: "DESIGNER_CREATE_NEW_DIVISION_TOOLTIP",
  header: "DIVISION_NAMES_GROUP_LIST_HEADER",
  entry: "DIVISION_NAMES_GROUP_LIST_ENTRY",
  etc: "DIVISION_NAMES_GROUP_LIST_ENTRY_ETC",
  fallback: "DIVISION_NAMES_GROUP_LIST_ENTRY_FALLBACK",
  prio0: "TEMPLATE_PRIO_0",
  prio0Desc: "TEMPLATE_PRIO_0_DESC",
  prio1: "TEMPLATE_PRIO_1",
  prio1Desc: "TEMPLATE_PRIO_1_DESC",
  prio2: "TEMPLATE_PRIO_2",
  prio2Desc: "TEMPLATE_PRIO_2_DESC",
  nicheOff: "DESIGNER_SELECT_NICHE_DISABLED",

  hardness: "STAT_ARMY_HARDNESS",
  hardnessDesc: "STAT_ARMY_HARDNESS_DESC",
  org: "STAT_COMMON_MAX_ORG",
  orgDesc: "STAT_COMMON_MAX_ORG_DESC",
  strength: "STAT_COMMON_MAX_STRENGTH",
  strengthDesc: "STAT_COMMON_MAX_STRENGTH_DESC",
  speed: "STAT_COMMON_MAXIMUM_SPEED",
  speedDesc: "STAT_COMMON_MAXIMUM_SPEED_DESC",
  width: "DESIGNER_COMBATWIDTH_TOOLTIP",
} as const;

export function priorityTooltip(
  at: number,
  texts: Record<string, string>,
): string {
  const keys = [
    [LIST_TEXTS.prio0, LIST_TEXTS.prio0Desc],
    [LIST_TEXTS.prio1, LIST_TEXTS.prio1Desc],
    [LIST_TEXTS.prio2, LIST_TEXTS.prio2Desc],
  ][at];
  if (!keys || !texts[keys[0]]) return "";
  return `§Y${texts[keys[0]]}§!\n${texts[keys[1]] ?? ""}`;
}

function useDropdownTexts(projectId: number, enabled: boolean) {
  return useSWR<Record<string, string>>(
    enabled ? ["game-view-loc", projectId, "designer-dropdowns"] : null,
    () => {
      guiRenderApi.setProjectId(projectId);
      return guiRenderApi.loc(Object.values(LIST_TEXTS), { markup: true });
    },
    { revalidateOnFocus: false },
  );
}

export function namesGroupTooltip(
  group: GuiNamesGroup,
  texts: Record<string, string>,
): string {
  let out = texts[LIST_TEXTS.header] ?? "";
  for (const name of group.examples)
    out += fillRuntimeSlots(texts[LIST_TEXTS.entry] ?? "\n  $NAME$", {
      NAME: name,
    });
  if (group.more) out += texts[LIST_TEXTS.etc] ?? "";
  if (group.fallback)
    out += fillRuntimeSlots(texts[LIST_TEXTS.fallback] ?? "", {
      NAME: group.fallback,
    });
  return out;
}

const CREATE_ICON = "GFX_new_regiment_slot";

function useDeltaTexts(projectId: number, enabled: boolean) {
  return useSWR<Record<string, string>>(
    enabled ? ["game-view-loc", projectId, "designer-deltas"] : null,
    () => {
      guiRenderApi.setProjectId(projectId);
      return guiRenderApi.loc([...DELTA_KEYS, ...SLOT_KEYS], { markup: true });
    },
    { revalidateOnFocus: false },
  );
}

function useStatLabels(projectId: number, enabled: boolean) {
  return useSWR<Record<string, string>>(
    enabled ? ["game-view-loc", projectId, "designer-stats"] : null,
    () => {
      guiRenderApi.setProjectId(projectId);
      return guiRenderApi.loc(LABEL_KEYS);
    },
    { revalidateOnFocus: false },
  );
}

function useChrome(projectId: number) {
  return useSWR<Record<string, GuiSprite>>(
    ["game-view-sprites", projectId, "designer-chrome"],
    () => {
      guiRenderApi.setProjectId(projectId);
      return guiRenderApi.sprites(CHROME);
    },
    { revalidateOnFocus: false },
  );
}

function useCellIcons(projectId: number, names: string[]) {
  const key = names.join(",");
  return useSWR<Record<string, GuiSprite>>(
    key ? ["game-view-sprites", projectId, "designer-cells", key] : null,
    () => {
      guiRenderApi.setProjectId(projectId);
      return guiRenderApi.sprites(names);
    },
    { revalidateOnFocus: false },
  );
}

function gridGeometry(
  root: GuiNode | undefined,
  name: string,
  maxColumns: number,
) {
  const node = root ? findNode(root, name) : undefined;
  if (!node || node.type !== "gridboxtype") return undefined;
  const raw = (node.raw ?? {}) as Record<string, unknown>;
  const slot = raw.slotsize as Record<string, number> | undefined;
  const w = Number(slot?.width ?? 0);
  const h = Number(slot?.height ?? 0);
  if (!w || !h) return undefined;
  const across = Number(raw.max_slots_horizontal ?? 0);
  return { slot: { w, h }, columns: across || maxColumns };
}

function slotIcon(entry: GuiWindow | undefined) {
  const button = (entry?.node?.children ?? []).find(
    (child) => child.name === "unit_icon",
  );
  const placeholder = button?.sprite
    ? entry?.sprites?.[button.sprite]
    : undefined;
  const scale = Number(button?.scale ?? 1) || 1;
  const frameW = placeholder?.width
    ? placeholder.width / Math.max(1, placeholder.frames || 1)
    : 76;
  return {
    at: {
      x: Number(button?.position?.x ?? 0),
      y: Number(button?.position?.y ?? 0),
    },
    scale,
    w: Math.round(frameW * scale),
    h: Math.round((placeholder?.height || 42) * scale),
  };
}

function slotChrome(entry: GuiWindow | undefined, fallback: string) {
  const background = entry?.node?.background ?? undefined;
  return {
    sprite: background?.sprite || fallback,
    at: {
      x: Number(background?.position?.x ?? 0),
      y: Number(background?.position?.y ?? 0),
    },
    icon: slotIcon(entry),
  };
}

type SlotChrome = ReturnType<typeof slotChrome>;

function findNode(root: GuiNode, name: string): GuiNode | undefined {
  if (root.name === name) return root;
  for (const child of root.children ?? []) {
    const found = findNode(child, name);
    if (found) return found;
  }
  return undefined;
}

export function cutFixed(value: number, digits: number): string {
  const scale = 10 ** digits;
  const cut = Math.trunc(value * scale + (value >= 0 ? 1e-6 : -1e-6)) / scale;
  return (Object.is(cut, -0) ? 0 : cut).toFixed(digits);
}

function formatStat(value: number | undefined, format?: StatFormat): string {
  const n = value ?? 0;
  switch (format) {
    case "speed":
      return `${cutFixed(n, 1)} km/h`;
    case "percent":
      return `${cutFixed(n * 100, 1)}%`;
    case "percent2":
      return `${cutFixed(n * 100, 2)}%`;
    case "percent0":
      return `${cutFixed(n * 100, 0)}%`;
    case "rate":
      return cutFixed(n, 2);
    case "whole":
      return String(Math.round(n));
    default:
      return cutFixed(n, 1);
  }
}

export function DesignerLayer({
  win,
  projectId,
  tag,
  date,
  template,
  cells = null,
  draft = null,
  original = null,
  onDraft,
  picker = null,
  onPicker,
  onTemplate,
  preview = null,
  onPreview,
  equipmentView = false,
  onSave,
  onReset,
  onEditSubUnit,
}: {
  win: GuiWindow;
  projectId: number;
  tag: string;
  date?: string | null;

  cells?: string | null;

  draft?: DesignerDraft | null;

  original?: DesignerDraft | null;

  onDraft?: (draft: DesignerDraft) => void;

  picker?: PickerState | null;
  onPicker?: (state: PickerState | null) => void;

  onTemplate?: (name: string) => void;

  preview?: DesignerDraft | null;
  onPreview?: (draft: DesignerDraft | null) => void;

  equipmentView?: boolean;

  onSave?: () => void;
  onReset?: () => void;

  onEditSubUnit?: (type: string) => void;

  template?: string | null;
}) {
  const active = win.name === DIVISION_DESIGNER;
  const designer = useDesigner(
    projectId,
    active ? tag : null,
    date ?? null,
    template ?? null,
    cells,
  );

  const hq = !!designer.data?.is_army_hq;
  const labels = useStatLabels(projectId, active);
  const listTexts = useDropdownTexts(projectId, active);
  const deltaTexts = useDeltaTexts(projectId, active);
  const previewDesigner = useDesigner(
    projectId,
    active && preview ? tag : null,
    date ?? null,
    template ?? null,
    preview ? encodeCells(preview) : null,
  );

  const previewData = preview ? previewDesigner.data : undefined;

  const [hovered, setHovered] = useState<PickerEntry | null>(null);

  const [slotHover, setSlotHover] = useState<{
    grid: "regiment" | "support" | "regimental";
    type: string;
  } | null>(null);
  const alone = useDesigner(
    projectId,
    active && slotHover ? tag : null,
    date ?? null,
    NEW_TEMPLATE,
    slotHover
      ? `${{ support: "s", regiment: "r", regimental: "rs" }[slotHover.grid]}:0:0:${slotHover.type}`
      : null,
  );
  const aloneData = slotHover ? alone.data : undefined;

  const tipBase = draft ?? (designer.data ? draftOf(designer.data) : null);
  const tipDraft =
    hovered?.kind === "unit" &&
    hovered.line === "replace" &&
    picker?.grid === "regiment" &&
    !picker.fill &&
    tipBase
      ? swappedAlone(tipBase, picker, hovered.unit.type)
      : null;
  const tipDesigner = useDesigner(
    projectId,
    active && tipDraft ? tag : null,
    date ?? null,
    template ?? null,
    tipDraft ? encodeCells(tipDraft) : null,
  );
  const tipData = tipDraft
    ? tipDesigner.isLoading
      ? undefined
      : tipDesigner.data
    : preview && !previewDesigner.isLoading
      ? previewDesigner.data
      : undefined;

  const [openList, setOpenList] = useState<"templates" | "names" | null>(null);

  const [renaming, setRenaming] = useState<string | null>(null);
  const iconNames = useMemo(() => {
    const data = designer.data;
    if (!data) return [] as string[];
    const names = new Set<string>();
    for (const cell of [
      ...data.regiments,
      ...data.support,
      ...(data.regimental_support ?? []),
    ]) {
      if (cell.icon) names.add(cell.icon);
    }

    for (const unit of data.available ?? [])
      if (unit.icon) names.add(unit.icon);
    if (data.icon) names.add(data.icon);
    return [...names].sort();
  }, [designer.data]);
  const icons = useCellIcons(projectId, iconNames);
  const chrome = useChrome(projectId);

  const regimentEntry = useTemplateWindow(
    projectId,
    active ? REGIMENT_ENTRY : null,
  );
  const supportEntry = useTemplateWindow(
    projectId,
    active ? SUPPORT_ENTRY : null,
  );
  const rects = useMemo(
    () => (active ? measureWindow(win.node, win.sprites, 1920, 1080) : {}),
    [active, win],
  );
  const font = useMemo(() => {
    const name =
      Object.keys(win.fonts).find((key) => key.includes("18mbs")) ??
      Object.keys(win.fonts)[0];
    return name ? win.fonts[name]?.bitmap : undefined;
  }, [win.fonts]);

  const filled = useMemo(() => {
    const at = (path: string) =>
      (win.node ? matchNamePath(win.node, path)[0] : undefined) as
        (GuiNode & { _id?: string }) | undefined;
    const view = hq ? "hq_view" : "non_hq_view";
    return {
      name: at(`${view}/divisions/name`),
      symbol: at(`${view}/div_templ_symbol_button`),
      priorityIcon: at(`${view}/division_priority_icon`),
      priority: [1, 2, 3].map((k) => at(`${view}/priority_button_${k}`)),

      niche: {
        bg: hq ? undefined : at("non_hq_view/niche_icon_bg"),
        button: hq ? undefined : at("non_hq_view/niche_button"),
        cross: hq ? undefined : at("non_hq_view/niche_icon_disabled"),
      },
      group: hq ? undefined : at("division_names/name"),
      templates: at(`${view}/divisions`),
      names: hq ? undefined : at("division_names"),
      stats: Object.fromEntries(
        STAT_BOXES.map((box) => [box.node, at(`${box.node}/stat_text`)]),
      ),
      hqLead: at("subunits_hq/regiments_grid_lead"),
      hqRow: at("subunits_hq/regiments_grid"),
      hqSupport: at("subunits_hq/support_grid"),
      hqBonuses: at("hq_modifiers_list/content"),

      equipAllowed: at("division_equipment_view/is_allowed_button"),
      equipForeign: at(
        "division_equipment_view/foreign_equipment_allowed_button",
      ),
      equipGrid: at("division_equipment_view/equipment_variants/grid"),
    };
  }, [win.node, hq]);

  const regimentGrid = useMemo(
    () => gridGeometry(win.node, "regiments_grid", MAX_DIVISION_BRIGADE_WIDTH),
    [win.node],
  );
  const supportGrid = useMemo(
    () => gridGeometry(win.node, "support_grid", MAX_DIVISION_SUPPORT_WIDTH),
    [win.node],
  );

  const regimentSupportGrid = useMemo(
    () =>
      gridGeometry(
        win.node,
        "regiment_support_grid",
        MAX_DIVISION_BRIGADE_WIDTH,
      ),
    [win.node],
  );

  if (!active || !designer.data || !font) return null;
  const data = designer.data;
  const text = labels.data ?? {};
  const rectOf = (node: (GuiNode & { _id?: string }) | undefined) =>
    node?._id ? rects[node._id] : undefined;
  const rectOfNode = rectOf;

  const gridOf = (node: GuiNode | undefined, across: number) => {
    const slot = (
      (node?.raw ?? {}) as { slotsize?: { width?: number; height?: number } }
    ).slotsize;
    const w = Number(slot?.width) || 0;
    const h = Number(slot?.height) || 0;
    return w && h ? { slot: { w, h }, columns: across } : undefined;
  };

  const bonusFont = win.fonts["hoi_16mbs"]?.bitmap ?? font;
  const bonusRect = rectOf(filled.hqBonuses);
  let bonusTop = bonusRect?.y ?? 0;
  const bonusRows =
    hq && bonusRect && bonusFont
      ? (data.commander_bonuses ?? []).map((line, at) => {
          const lines =
            layoutTooltip(
              line,
              bitmapMeasure(bonusFont),
              () => null,
              HQ_BONUS_WIDTH,
            ).lines.length || 1;
          const top = bonusTop;
          bonusTop += lines * bonusFont.line + HQ_BONUS_PADDING;
          return (
            <RichText
              key={`bonus-${at}`}
              font={bonusFont}
              markup={line}
              left={bonusRect.x}
              top={top}
              width={HQ_BONUS_WIDTH}
              align="left"
              vcenter={false}
              maxLines={3}
            />
          );
        })
      : null;

  const regimentRect = rects["name:regiments_grid"];
  const supportRect = rects["name:support_grid"];

  const regimentSupportRect = rects["name:regiment_support_grid"];
  const regimentChrome = slotChrome(regimentEntry.data, CELL_BG);
  const supportChrome = slotChrome(supportEntry.data, SUPPORT_BG);

  const widthRect =
    rects[hq ? "name:combat_width_view_hq" : "name:combat_width_view"];

  const shownDraft = draft ?? draftOf(data);

  const covered = !!picker || equipmentView;
  const units = data.available ?? [];
  const iconOf = (type: string) =>
    [
      ...data.regiments,
      ...data.support,
      ...(data.regimental_support ?? []),
    ].find((cell) => cell.type === type)?.icon ??
    units.find((unit) => unit.type === type)?.icon;
  const labelOf = (type: string) =>
    [
      ...data.regiments,
      ...data.support,
      ...(data.regimental_support ?? []),
    ].find((cell) => cell.type === type)?.label ||
    units.find((unit) => unit.type === type)?.label ||
    type;

  const hqRow = (
    slot: DesignerSlot,
    grid: "regiment" | "support" | "regimental",
  ) => hq && grid === "regiment" && slot.y > 0;
  const aboveOf = (slot: DesignerSlot) =>
    column(shownDraft, slot.x)[slot.y - 1]?.type;
  const tipOf = (
    slot: DesignerSlot,
    grid: "regiment" | "support" | "regimental" = "regimental",
  ) => {
    const texts = deltaTexts.data ?? {};
    if (hqRow(slot, grid)) {
      const above = aboveOf(slot);
      if (slot.kind === "unit")
        return [texts.DESIGNER_HQ_REMOVE ?? ""].filter(Boolean);
      if (slot.kind === "add" && above) {
        return [
          fillRuntimeSlots(texts.DESIGNER_HQ_ADD_COPY_ABOVE ?? "", {
            UNIT: labelOf(above),
          }),
        ].filter(Boolean);
      }
      return [];
    }
    return slotHints(
      slot,
      grid,
      slot.x > 0 && column(shownDraft, slot.x - 1).length > 0,
      texts,
    );
  };

  const hoverSlot = (
    slot: DesignerSlot,
    grid: "regiment" | "support" | "regimental",
    label: string,
  ) => {
    const kind = grid;

    const copy =
      hqRow(slot, grid) && slot.kind === "add" ? aboveOf(slot) : undefined;
    const type = slot.cell?.type ?? copy;
    const alone =
      slotHover?.grid === kind && slotHover.type === type
        ? aloneData
        : undefined;
    const content =
      slot.kind === "unit" && type
        ? slotTooltip(
            label,
            tipOf(slot, grid),
            alone,
            deltaTexts.data ?? {},
            hq
              ? [
                  deploymentLine(
                    units.find((unit) => unit.type === type),
                    deltaTexts.data ?? {},
                  ),
                ].filter((line): line is string => !!line)
              : [],
          )
        : copy
          ? slotTooltip(null, tipOf(slot, grid), alone, deltaTexts.data ?? {})
          : tipOf(slot, grid).join(NL);
    const tip = gameTooltipProps(content || null, {
      key: `designer-slot:${grid}:${slot.x}:${slot.y}`,
    });
    return {
      onPointerEnter: (event: React.PointerEvent<Element>) => {
        if (type) setSlotHover({ grid: kind, type });
        tip.onPointerEnter(event);
      },
      onPointerMove: tip.onPointerMove,
      onPointerLeave: () => {
        setSlotHover(null);
        tip.onPointerLeave();
      },
    };
  };

  const clickSlot = (
    slot: DesignerSlot,
    grid: "regiment" | "support" | "regimental",
    event: React.MouseEvent,
  ) => {
    hideGameTooltip(`designer-slot:${grid}:${slot.x}:${slot.y}`);
    setSlotHover(null);
    const target = { grid, x: slot.x, y: slot.y };
    const regiment = target.grid === "regiment";
    if (hqRow(slot, grid)) {
      const above = aboveOf(slot);
      if (slot.kind === "unit") return onDraft?.(removed(shownDraft, target));
      if (slot.kind === "add" && above)
        return onDraft?.(placed(shownDraft, target, above, units));
      return;
    }
    if (slot.kind === "unit") {
      if (regiment && event.ctrlKey)
        return onDraft?.(removedRegiment(shownDraft, slot.x));
      if (event.shiftKey) return onDraft?.(removed(shownDraft, target));
    } else if (regiment && slot.y > 0) {
      const above = column(shownDraft, slot.x)[slot.y - 1]?.type;
      if (above && event.ctrlKey)
        return onDraft?.(
          filledRegiment(shownDraft, slot.x, above, units, slot.y),
        );
      if (above && event.shiftKey)
        return onDraft?.(placed(shownDraft, target, above, units));
    } else if (regiment) {
      const left =
        slot.x > 0 ? column(shownDraft, slot.x - 1)[0]?.type : undefined;
      if (left && event.ctrlKey && event.shiftKey)
        return onDraft?.(filledRegiment(shownDraft, slot.x, left, units));
      if (left && event.shiftKey)
        return onDraft?.(placed(shownDraft, target, left, units));
      if (event.ctrlKey)
        return onPicker?.({ ...target, group: null, fill: true });
    }
    onPicker?.({ ...target, group: null });
  };
  const slotView = (
    slots: DesignerSlot[],
    rect: { x: number; y: number } | undefined,
    geom: { slot: { w: number; h: number }; columns: number } | undefined,
    chromeOf: SlotChrome,
    grid: "regiment" | "support" | "regimental",

    spineTo?: number,
  ) => {
    if (!rect || !geom) return null;
    const slotW = geom.slot.w;
    const slotH = geom.slot.h;
    const box = chrome.data?.[chromeOf.sprite];
    const icon = chromeOf.icon;

    const spines =
      spineTo !== undefined
        ? [...new Set(slots.map((slot) => slot.x))].map((column) => ({
            column,
            height: spineTo - rect.y,
          }))
        : [];

    const lineTop = icon.at.y + Math.round((icon.h - font.line) / 2);
    return (
      <>
        {spines.map((run) => (
          <div
            key={`spine-${grid}-${run.column}`}
            aria-hidden
            style={{
              position: "absolute",
              left: rect.x + run.column * slotW + chromeOf.at.x,
              top: rect.y,
              width: 1,
              height: Math.max(0, run.height),
              background: CONNECTOR,
            }}
          />
        ))}
        {slots.map((slot) => {
          const type = slot.cell?.type;
          const name = type
            ? iconOf(type)
            : SLOT_ART[slot.kind as keyof typeof SLOT_ART];
          const sprite = name
            ? type
              ? icons.data?.[name]
              : chrome.data?.[name]
            : undefined;
          const left = rect.x + (slot.at?.x ?? slot.x) * slotW;
          const top = rect.y + (slot.at?.y ?? slot.y) * slotH;
          const label = type ? labelOf(type) : "";
          const hasArt = !!sprite && !sprite.missing_file;

          const unnamed = slot.kind === "unit" && !!icons.data && !hasArt;
          const pickable =
            !!onPicker && (slot.kind === "unit" || slot.kind === "add");
          return (
            <div
              key={`${grid}-${slot.x}-${slot.y}-${slot.kind}`}
              style={{
                position: "absolute",
                left,
                top,
                width: slotW,
                height: slotH,
              }}
            >
              {box && (
                <div
                  style={frameStyle(box, chromeOf.at.x, chromeOf.at.y)}
                  {...artTag(chromeOf.sprite)}
                />
              )}
              {sprite && hasArt && (
                <div
                  style={frameStyle(sprite, icon.at.x, icon.at.y, {
                    scale: icon.scale,
                    name,
                  })}
                  {...artTag(name, 1, label || name)}
                />
              )}
              {unnamed && (
                <BitmapText
                  font={font}
                  text={label}
                  left={icon.at.x}
                  top={lineTop}
                  width={icon.w}
                  height={font.line + 2}
                  align="center"
                  vcenter={false}
                  maxLines={1}
                />
              )}
              {!pickable && tipOf(slot, grid).length > 0 && (
                <div
                  {...gameTooltipProps(tipOf(slot, grid).join(NL), {
                    key: `designer-slot:${grid}:${slot.x}:${slot.y}`,
                  })}
                  style={{
                    position: "absolute",
                    left: icon.at.x,
                    top: icon.at.y,
                    width: icon.w,
                    height: icon.h,
                    pointerEvents: "auto",
                  }}
                />
              )}
              {pickable && (
                <button
                  type="button"
                  aria-label={label || `${grid} ${slot.x} ${slot.y}`}
                  onClick={(event) => clickSlot(slot, grid, event)}
                  {...hoverSlot(slot, grid, label)}
                  style={{
                    position: "absolute",
                    left: icon.at.x,
                    top: icon.at.y,
                    width: icon.w,
                    height: icon.h,
                    background: "transparent",
                    border: "none",
                    padding: 0,
                    cursor: "inherit",
                    pointerEvents: "auto",
                  }}
                />
              )}
            </div>
          );
        })}
      </>
    );
  };

  const pickerRect = rects["name:designer_regiments_grid"];
  const pickerGrid = gridGeometry(win.node, "designer_regiments_grid", 2);
  const pickerTarget = picker;

  const previewOf = (entry: PickerEntry): DesignerDraft | null => {
    if (!pickerTarget) return null;
    if (entry.kind === "remove") return removed(shownDraft, pickerTarget);
    if (entry.kind === "unit" && entry.line !== "selected")
      return placed(shownDraft, pickerTarget, entry.unit.type, units);
    return null;
  };
  const choose = (entry: PickerEntry) => {
    if (!pickerTarget) return;
    if (entry.kind === "remove") {
      onDraft?.(removed(shownDraft, pickerTarget));
      onPicker?.(null);
    } else if (entry.kind === "group") {
      onPicker?.({ ...pickerTarget, group: entry.group });
    } else {
      if (pickerTarget.fill && pickerTarget.grid === "regiment") {
        onDraft?.(
          filledRegiment(shownDraft, pickerTarget.x, entry.unit.type, units),
        );
      } else if (entry.line !== "selected") {
        onDraft?.(placed(shownDraft, pickerTarget, entry.unit.type, units));
      }
      onPicker?.(null);
    }
  };

  const paint = (node: (GuiNode & { _id?: string }) | undefined, frame = 1) => {
    const rect = rectOf(node);
    const sprite = node?.sprite ? win.sprites[node.sprite] : undefined;
    if (!rect || !sprite || !sprite.resolved) return null;
    return (
      <div
        key={node?._id}
        style={frameStyle(sprite, rect.x, rect.y, {
          frame,
          scale: node?.scale || 1,
          name: node?.sprite,
        })}
        {...artTag(node?.sprite, frame, node?.name ?? undefined)}
      />
    );
  };

  const priority = Math.min(
    2,
    Math.max(0, Math.round(shownDraft.priority ?? data.priority ?? 1)),
  );

  const boxOf = (node: (GuiNode & { _id?: string }) | undefined) => {
    const rect = rectOf(node);
    const sprite = node?.sprite ? win.sprites[node.sprite] : undefined;
    if (!rect || !sprite) return undefined;
    const scale = node?.scale || 1;
    return {
      x: rect.x,
      y: rect.y,
      w: Math.round((sprite.width / Math.max(1, sprite.frames || 1)) * scale),
      h: Math.round(sprite.height * scale),
    };
  };
  const control = (box: {
    x: number;
    y: number;
    w: number;
    h: number;
  }): React.CSSProperties => ({
    position: "absolute",
    left: box.x,
    top: box.y,
    width: box.w,
    height: box.h,
    background: "transparent",
    border: "none",
    padding: 0,
    cursor: "inherit",
    pointerEvents: "auto",
  });

  const namesKey =
    shownDraft.namesGroup === undefined
      ? data.division_names_group || null
      : shownDraft.namesGroup;
  const nameRect = rectOf(filled.name);

  const listText = listTexts.data ?? {};
  const templateItems: DropdownItem[] = [
    {
      key: NEW_TEMPLATE,
      label: listText[LIST_TEXTS.create] ?? "Create Empty",
      icon: CREATE_ICON,
      tooltip: listText[LIST_TEXTS.createTip],
    },
    ...data.templates.map((name) => ({
      key: name,
      label: name,
      icon: data.template_icons?.[name],
      selected: !data.new && name === data.name,
    })),
  ];

  const namesItems: DropdownItem[] = [
    { key: "", label: "None", selected: !namesKey },
    ...(data.names_groups ?? []).map((group) => ({
      key: group.key,
      label: group.name,
      selected: group.key === namesKey,
      tooltip: namesGroupTooltip(group, listText),
    })),
  ];
  const symbolRect = rectOf(filled.symbol);
  const symbol = data.icon ? icons.data?.[data.icon] : undefined;
  const groupRect = rectOf(filled.group);
  const groupFont = filled.group?.font
    ? win.fonts[filled.group.font]?.bitmap
    : undefined;

  const noneText = (labels.data ?? {})[NONE_KEY] || "None";
  const groupText =
    shownDraft.namesGroup === undefined
      ? data.names_group || noneText
      : ((data.names_groups ?? []).find((group) => group.key === namesKey)
          ?.name ?? noneText);

  const empty = shownDraft.regiments.length + shownDraft.support.length === 0;
  const header = empty && data.header_stats ? data.header_stats : data.stats;

  const paper = previewData ?? data;
  const paperEmpty = previewData
    ? (preview?.regiments.length ?? 0) + (preview?.support.length ?? 0) === 0
    : empty;
  const arrowOf = (
    stat: string,
    before: number | undefined,
    after: number | undefined,
    lowerBetter = lowerIsBetter(stat),
  ) => (previewData ? changeArrow(before ?? 0, after ?? 0, lowerBetter) : "");
  const statRows = (rows: readonly StatRow[]) =>
    rows.map((row) => ({
      key: row.key,
      label: text[row.key] ?? row.key,
      value: paperEmpty
        ? (text[NO_STAT] ?? "--")
        : formatStat(paper.stats[row.stat], row.format) +
          arrowOf(
            row.stat,
            empty ? 0 : data.stats[row.stat],
            paper.stats[row.stat],
          ),
    }));

  const equipmentRows = paperEmpty
    ? []
    : paper.equipment.map((item) => ({
        key: item.archetype,
        label: item.short_label || item.label || item.archetype,
        value:
          String(Math.round(item.amount)) +
          arrowOf(
            item.archetype,
            empty
              ? 0
              : data.equipment.find((was) => was.archetype === item.archetype)
                  ?.amount,
            item.amount,
            true,
          ),
      }));
  const statList = (
    rows: { key: string; label: string; value: string }[],
    rect: { x: number; y: number } | undefined,
  ) =>
    rect ? (
      <DesignerStatRows
        projectId={projectId}
        rows={rows}
        left={rect.x + 5}
        top={rect.y}
      />
    ) : null;

  const baseRect = rects["name:base_stats"];
  const combatRect = rects["name:combat_stats"];
  const equipRect = rects["name:equipment_stats"];

  return (
    <div
      style={{ position: "absolute", left: 0, top: 0, pointerEvents: "none" }}
    >
      {!covered && nameRect && renaming === null && (
        <BitmapText
          font={font}
          text={shownDraft.name ?? data.name}
          left={nameRect.x}

          top={
            nameRect.y + Math.round(((nameRect.h || font.line) - font.line) / 2)
          }
          width={nameRect.w || 185}
          height={font.line + 2}
          align="left"
          vcenter={false}
          maxLines={1}
        />
      )}

      {!covered && symbolRect && symbol && !symbol.missing_file && (
        <div
          style={frameStyle(symbol, symbolRect.x, symbolRect.y, {
            name: data.icon,
          })}
          {...artTag(data.icon, 1, data.name)}
        />
      )}

      {!covered && paint(filled.priorityIcon, priority + 1)}

      {!covered &&
        filled.priority.map((node, at) =>
          paint(node, at + 1 + (priority === at ? 3 : 0)),
        )}

      {!covered && paint(filled.niche.bg, 2)}
      {!covered && paint(filled.niche.button, 1)}

      {!covered &&
        filled.priority.map((node, at) => {
          const box = boxOf(node);
          if (!box) return null;
          return (
            <button
              key={`priority-${at}`}
              type="button"
              aria-label={`priority_button_${at + 1}`}
              onClick={() => onDraft?.({ ...shownDraft, priority: at })}
              {...gameTooltipProps(priorityTooltip(at, listText), {
                key: `designer-priority:${at}`,
              })}
              style={control(box)}
            />
          );
        })}
      {!covered &&
        boxOf(filled.niche.button) &&
        listText[LIST_TEXTS.nicheOff] && (
          <div
            {...gameTooltipProps(listText[LIST_TEXTS.nicheOff], {
              key: "designer-niche",
            })}
            style={control(
              boxOf(filled.niche.button) as {
                x: number;
                y: number;
                w: number;
                h: number;
              },
            )}
          />
        )}
      {!covered && paint(filled.niche.cross, 1)}

      {!covered && groupRect && groupText && (
        <BitmapText
          font={groupFont ?? font}
          text={groupText}
          left={groupRect.x}
          top={groupRect.y}
          width={filled.group?.maxwidth || 185}
          height={(groupFont ?? font).line + 4}
          align={textAlignOf(filled.group?.format)}
          vcenter={false}
          maxLines={1}
        />
      )}
      {(
        [
          ["save_button", onSave],
          ["reset_button", onReset],
        ] as const
      ).map(([name, run]) => {
        const rect = rects[`name:${name}`];
        return run && rect ? (
          <button
            key={name}
            type="button"
            aria-label={`${name}_edit`}
            onClick={run}
            style={{
              position: "absolute",
              left: rect.x,
              top: rect.y,
              width: rect.w,
              height: rect.h,
              background: "transparent",
              border: "none",
              padding: 0,
              cursor: "pointer",
              pointerEvents: "auto",
              zIndex: 7,
            }}
          />
        ) : null;
      })}

      {!covered &&
        nameRect &&
        onDraft &&
        (renaming === null ? (
          <button
            type="button"
            aria-label="rename_template"
            onClick={() => {
              setOpenList(null);
              setRenaming(shownDraft.name ?? data.name);
            }}
            style={{
              position: "absolute",
              left: nameRect.x,
              top: nameRect.y,
              width: nameRect.w || 185,
              height: nameRect.h || 20,
              background: "transparent",
              border: "none",
              padding: 0,
              cursor: "text",
              pointerEvents: "auto",
              zIndex: 7,
            }}
          />
        ) : (
          <input
            aria-label="template_name"
            autoFocus
            value={renaming}
            maxLength={60}
            onFocus={(event) => event.currentTarget.select()}
            onChange={(event) => setRenaming(event.target.value)}
            onKeyDown={(event) => {
              event.stopPropagation();
              if (event.key === "Enter") event.currentTarget.blur();
              if (event.key === "Escape") setRenaming(null);
            }}
            onBlur={() => {
              const next = renaming.trim();
              setRenaming(null);
              if (next)
                onDraft({
                  ...shownDraft,
                  name: next === data.name ? undefined : next,
                });
            }}
            className="rounded-sm border border-accent bg-popover px-1 text-sm text-popover-foreground outline-none"
            style={{
              position: "absolute",
              left: nameRect.x - 2,
              top: nameRect.y - 2,
              width: (nameRect.w || 185) + 4,
              height: (nameRect.h || 20) + 4,
              pointerEvents: "auto",
              zIndex: 9,
            }}
          />
        ))}
      {!covered && (
        <>
          <DesignerDropdown
            projectId={projectId}
            name="divisions"
            box={filled.templates}
            at={rectOf(filled.templates)}
            windowSprites={win.sprites}
            entryName="designer_division_entry"
            items={templateItems}
            open={openList === "templates"}
            onToggle={() =>
              setOpenList((open) => (open === "templates" ? null : "templates"))
            }
            onPick={(name) => {
              setOpenList(null);
              onTemplate?.(name);
            }}
          />
          <DesignerDropdown
            projectId={projectId}
            name="division_names"
            box={filled.names}
            at={rectOf(filled.names)}
            windowSprites={win.sprites}
            entryName="designer_div_name_group_entry"
            items={namesItems}
            open={openList === "names"}
            onToggle={() =>
              setOpenList((open) => (open === "names" ? null : "names"))
            }
            onPick={(key) => {
              setOpenList(null);
              onDraft?.({ ...shownDraft, namesGroup: key || null });
            }}
          />
        </>
      )}
      {!covered && widthRect && listText[LIST_TEXTS.width] && (
        <div
          {...gameTooltipProps(
            fillRuntimeSlots(listText[LIST_TEXTS.width], {
              VALUE: Math.round(header.combat_width ?? 0),
            }),
            {
              key: "designer-width",
            },
          )}
          style={control({
            x: widthRect.x,
            y: widthRect.y,
            w: widthRect.w || 40,
            h: widthRect.h || 40,
          })}
        />
      )}
      {!covered && widthRect && (
        <BitmapText
          font={font}
          text={formatStat(header.combat_width, "whole")}
          left={widthRect.x}
          top={widthRect.y + 10}
          width={40}
          height={font.line + 4}
          align="center"
          vcenter={false}
          maxLines={1}
        />
      )}
      {STAT_BOXES.map((box) => {
        const rect = rects[`name:${box.node}`];
        if (!rect || covered) return null;

        const at = filled.stats[box.node];
        const left = rect.x + Number(at?.position?.x ?? 45);
        const top = rect.y + Number(at?.position?.y ?? 2);

        const [value, ...unit] = formatStat(header[box.stat], box.format).split(
          " ",
        );
        const tip = statBoxTooltip(box.key, listText);
        return (
          <Fragment key={box.node}>
            {tip && (
              <div
                {...gameTooltipProps(tip, { key: `designer-box:${box.node}` })}
                style={control(rect)}
              />
            )}
            <BitmapText
              font={font}
              text={value}
              left={left}
              top={top}
              width={0}
              height={font.line + 4}
              align="left"
              vcenter={false}
              maxLines={1}
              color={fontTextColors("hoi_18mbs").Y}
            />
            {unit.length > 0 && (
              <BitmapText
                font={font}
                text={unit.join(" ")}
                left={left + measureLine(font, `${value} `)}
                top={top}
                width={0}
                height={font.line + 4}
                align="left"
                vcenter={false}
                maxLines={1}
              />
            )}
          </Fragment>
        );
      })}

      {!covered &&
        !hq &&
        slotView(
          regimentSlots(shownDraft),
          regimentRect,
          regimentGrid,
          regimentChrome,
          "regiment",
          regimentSupportRect
            ? regimentSupportRect.y + supportChrome.at.y + 16
            : undefined,
        )}
      {!covered &&
        !hq &&
        slotView(
          supportSlots(shownDraft),
          supportRect,
          supportGrid,
          supportChrome,
          "support",
        )}
      {!covered &&
        !hq &&
        slotView(
          regimentalSupportSlots(shownDraft),
          regimentSupportRect,
          regimentSupportGrid,
          supportChrome,
          "regimental",
        )}

      {!covered && hq && (
        <>
          {slotView(
            hqRegimentSlots(shownDraft).lead,
            rectOfNode(filled.hqLead),
            gridOf(filled.hqLead, 1),
            regimentChrome,
            "regiment",
          )}
          {slotView(
            hqRegimentSlots(shownDraft).row,
            rectOfNode(filled.hqRow),
            gridOf(filled.hqRow, 3),
            regimentChrome,
            "regiment",
          )}
          {slotView(
            hqSupportSlots(shownDraft),
            rectOfNode(filled.hqSupport),
            gridOf(filled.hqSupport, 4),
            supportChrome,
            "support",
          )}
        </>
      )}

      {!covered && hq && bonusRows}

      {equipmentView && paint(filled.equipAllowed, CHECKED)}
      {equipmentView && paint(filled.equipForeign, CHECKED)}
      {equipmentView && (
        <DesignerEquipment
          projectId={projectId}
          rect={(() => {
            const grid = rectOf(filled.equipGrid);
            return grid
              ? {
                  x: grid.x,
                  y: grid.y + EQUIPMENT_LIST_MARGIN,
                  w: grid.w || 572,
                  h: (grid.h || 466) - 2 * EQUIPMENT_LIST_MARGIN,
                }
              : undefined;
          })()}
          groups={data.equipment_types ?? []}
        />
      )}
      {picker && pickerRect && pickerGrid && (
        <DesignerPicker
          projectId={projectId}
          grid={{
            x: pickerRect.x,
            y: pickerRect.y,
            slot: pickerGrid.slot,
            columns: pickerGrid.columns,
          }}
          back={rects["name:cancel_button"]}
          entries={pickerEntries(
            shownDraft,
            picker,
            units,
            data.costs ?? { group: 20, change: 5, support: 10 },
            original ?? shownDraft,
          )}
          units={units}
          titles={data.group_titles ?? {}}
          experience={data.army_experience ?? 0}
          onEntry={(entry) => {
            setHovered(null);
            onPreview?.(null);
            choose(entry);
          }}
          onHover={(entry) => {
            setHovered(entry);
            onPreview?.(entry ? previewOf(entry) : null);
          }}
          onEditSubUnit={onEditSubUnit}
          tooltipOf={(entry) => {
            if (entry.kind === "unit" && entry.line === "selected")
              return `§Y${entry.unit.label || entry.unit.type}§!`;
            if (
              !hovered ||
              !sameEntry(entry, hovered) ||
              !tipData ||
              entry.kind === "group"
            )
              return undefined;
            const tip = deltaTooltip(
              entry.kind === "remove"
                ? (deltaTexts.data?.DESIGNER_REMOVE ?? "Remove")
                : entry.unit.label || entry.unit.type,
              empty ? { ...data, stats: {}, equipment: [] } : data,
              tipData,
              deltaTexts.data ?? {},
            );

            return hq && entry.kind === "unit"
              ? hqPickerTooltip(tip, entry.unit, deltaTexts.data ?? {})
              : tip;
          }}

          onBack={() =>
            onPicker?.(picker.group ? { ...picker, group: null } : null)
          }
        />
      )}
      {statList(statRows(BASE_STATS), baseRect)}
      {statList(statRows(COMBAT_STATS), combatRect)}
      {statList([...statRows(COST_STATS), ...equipmentRows], equipRect)}

      <DesignerTerrain
        projectId={projectId}
        rect={rects["name:adjuster_stats"]}
        terrain={paperEmpty ? [] : (paper.terrain ?? [])}
      />
    </div>
  );
}
