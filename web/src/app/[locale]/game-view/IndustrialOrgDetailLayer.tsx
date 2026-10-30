"use client";

import { useMemo, type CSSProperties } from "react";
import useSWR from "swr";
import { useTranslations } from "use-intl";
import { guiArray, guiRecord, readGuiResponse } from "./guiResponse";
import type { GuiNode, GuiSprite, GuiWindow } from "@/lib/guiRenderApi";
import { withGameLanguage } from "@/lib/guiRenderApi";
import { GuiCanvas, assignIds, measureWindow } from "../gui-renderer/GuiCanvas";
import { useTemplateWindow } from "./CountrySelectLayer";
import { useEditCanvas } from "./editContext";
import type { TextSource } from "./editModel";
import { keySource } from "./textSources";
import { matchNamePath } from "./gameNavigation";
import { FieldHit } from "./FieldHit";
import { frameStyle, spriteUrlOf } from "./spriteFrame";
import type {
  GuiIndustrialOrgEquipment,
  MioCategory,
} from "./IndustrialOrgLayer";

export const MIO_DETAIL_WINDOW = "industrial_organisation_detail_window";
const TREE = "industrial_organisation_tree_window";
const TRAIT = "industrial_organisation_trait_entry";
const HEADER = "industrial_organisation_flavortext_window";
const EXCLUSIVE = "industrial_organisation_mutually_exclusive_item";
const SEPARATOR = "industrial_organisation_detail_separator";
const MODIFIER = "industrial_organisation_detail_modifier_item";
const EQUIPMENT = "industrial_organisation_equipment_type_window";

export const CELL_W = 87;
export const CELL_H = 117;

const LINE_OUT = { x: 42, y: 102 };

const LINE_IN_Y = 20;

const EXCLUSIVE_AT = { x: 74, y: 30 };
const NO_HIDDEN = new Set<string>();

export type MioTab = "traits" | "history";

export interface GuiIndustrialOrgTrait {
  token: string;
  label: string;
  icon?: string;
  x: number;
  y: number;
  special: boolean;
  bonuses: ("design" | "production" | "organization")[];
  parents: string[];
  any_parent: string[];
  exclusive: string[];

  values?: Partial<
    Record<
      "equipment_bonus" | "production_bonus" | "organization_modifier",
      { key: string; value: number }[]
    >
  >;

  relative_to?: string;
}

export interface GuiIndustrialOrgDetail {
  token: string;
  name: string;
  icon?: string;
  icon_sprite?: GuiSprite;
  category: MioCategory;
  size: number;
  points: number;
  background: string;
  headers: { text: string; x: number }[];
  traits: GuiIndustrialOrgTrait[];
  sprites: Record<string, GuiSprite>;
  research_bonus: number;
  assign_cost: number;
  texts: {
    trait_points: string;
    attributes: string;
    equipment_bonuses: string;
    research_bonus: string;
    assign_cost: string;
  };
  initial: { label?: string; equipment: GuiIndustrialOrgEquipment[] };
}

export function useIndustrialOrg(
  projectId: number,
  tag: string | null,
  date: string | null | undefined,
  token: string | null,
) {
  return useSWR<GuiIndustrialOrgDetail>(
    tag && token
      ? ["game-view-industrial-org", projectId, tag, date ?? "", token]
      : null,
    async () => {
      const query = new URLSearchParams({
        tag: tag as string,
        id: token as string,
      });
      if (date) query.set("date", date);
      const response = await fetch(
        withGameLanguage(
          `/api/go/projects/${projectId}/gui-render/industrial-org?${query}`,
        ),
      );
      return readGuiResponse<GuiIndustrialOrgDetail>(response);
    },
    { revalidateOnFocus: false },
  );
}

function idOf(node: GuiNode | undefined): string {
  return (node as (GuiNode & { _id?: string }) | undefined)?._id ?? "";
}

export const MIO_TABS: Readonly<Record<string, MioTab>> = {
  traits_tab: "traits",
  history_tab: "history",
};

export function mioDetailWindow(
  win: GuiWindow,
  detail: GuiIndustrialOrgDetail | undefined,
  tab: MioTab,
): {
  win: GuiWindow;
  hidden: string[];
  shown: string[];
  texts: Record<string, string>;
  sources: Record<string, TextSource>;
} {
  const node = structuredClone(win.node) as GuiNode;
  const icon = detail?.icon && detail.icon_sprite ? detail.icon : null;
  const walk = (at: GuiNode, path: string) => {
    const margin = at.raw?.margin as
      { left?: number; top?: number } | undefined;
    for (const child of at.children ?? []) {
      const here = path ? `${path}/${child.name}` : (child.name ?? "");
      if (at.name === "aggregated_bonuses_window" && margin && child.position) {
        child.position = {
          x: (Number(child.position.x) || 0) + (Number(margin.left) || 0),
          y: (Number(child.position.y) || 0) + (Number(margin.top) || 0),
        };
      }
      if (child.name === "traits_tab") child.frame = tab === "traits" ? 2 : 1;
      if (child.name === "history_tab") child.frame = tab === "history" ? 2 : 1;
      if (child.name === "toggle_auto_designs") child.frame = 2;
      if (here === "top_left_window/icon" && icon) child.sprite = icon;
      walk(child, here);
    }
  };
  walk(node, "");
  const ids = (path: string) =>
    matchNamePath(node, path).map(idOf).filter(Boolean);

  const shown =
    tab === "traits"
      ? ids("tree_scrollbar_window")
      : [
          ...ids("history_window"),
          ...ids("history_window/variants_window"),
          ...ids("history_window/prod_lines_window"),
        ];
  const hidden = [
    ...ids(tab === "traits" ? "history_window" : "tree_scrollbar_window"),
    ...ids("open_policies_window/click_to_open_policies_text"),
    ...ids("open_policies_window/attached_policy_icon"),
    ...ids("open_policies_window/attached_policy_name"),
  ];
  const texts: Record<string, string> = {};
  const sources: Record<string, TextSource> = {};
  if (detail) {
    const set = (path: string, text: string) =>
      ids(path).forEach((id) => (texts[id] = text));
    set("industrial_org_name", detail.name);
    set("top_left_window/points", detail.texts?.trait_points ?? "");
    set("top_left_window/size_icon_window/size", String(detail.size));

    const name = keySource(detail.token, "everywhere");
    if (name) ids("industrial_org_name").forEach((id) => (sources[id] = name));
  }
  const sprites =
    icon && detail?.icon_sprite
      ? { ...win.sprites, [icon]: detail.icon_sprite }
      : win.sprites;
  return { win: { ...win, node, sprites }, hidden, shown, texts, sources };
}

function Item({
  entry,
  top,
  texts,
  swap,
  width,
  height,
  grow = false,
}: {
  entry: GuiWindow;
  top: number;
  texts: Record<string, string>;

  swap?: Record<string, { name: string; sprite?: GuiSprite }>;
  width: number;
  height: number;

  grow?: boolean;
}) {
  const edit = useEditCanvas();
  const node = useMemo(() => {
    const clone = structuredClone(entry.node) as GuiNode;
    const walk = (at: GuiNode) => {
      for (const child of at.children ?? []) {
        const picture = swap?.[child.name ?? ""];
        if (picture?.sprite) child.sprite = picture.name;

        if (grow && child.type === "instanttextboxtype") child.maxheight = null;
        walk(child);
      }
    };
    walk(clone);
    assignIds(clone);
    return clone;
  }, [entry.node, swap, grow]);
  const sprites = useMemo(() => {
    const out = { ...entry.sprites };
    for (const picture of Object.values(swap ?? {}))
      if (picture.sprite) out[picture.name] = picture.sprite;
    return out;
  }, [entry.sprites, swap]);
  const overrides = useMemo(() => {
    const out: Record<string, string> = {};
    for (const [path, text] of Object.entries(texts)) {
      for (const hit of matchNamePath(node, path))
        if (idOf(hit)) out[idOf(hit)] = text;
    }
    return out;
  }, [node, texts]);
  return (
    <div style={{ position: "absolute", left: 0, top }}>
      <GuiCanvas
        node={node}
        {...edit(entry.name, { ...entry, sprites }, node, sprites, overrides)}
        texts={entry.texts}
        fonts={entry.fonts}
        hidden={NO_HIDDEN}
        isolateId={null}
        transparent
        baseW={width}
        baseH={height}
      />
    </div>
  );
}

const LINE_H = 18;
const SEPARATOR_H = 29;

export interface BonusRow {
  kind: "separator" | "modifier" | "equipment";
  text?: string;
  equipment?: GuiIndustrialOrgEquipment;
  height: number;

  top: number;
}

export function bonusRows(detail: GuiIndustrialOrgDetail): BonusRow[] {
  const texts = detail.texts ?? ({} as GuiIndustrialOrgDetail["texts"]);
  const initial =
    detail.initial ?? ({ equipment: [] } as GuiIndustrialOrgDetail["initial"]);
  const percent = Math.round((Number(detail.research_bonus) || 0) * 100);
  const rows: Omit<BonusRow, "top">[] = [
    { kind: "separator", text: texts.attributes ?? "", height: SEPARATOR_H },
    {
      kind: "modifier",
      text: `${texts.research_bonus ?? ""}: §G${percent}%§!`,
      height: LINE_H,
    },

    {
      kind: "modifier",
      text: `${texts.assign_cost ?? ""}:\n£pol_power §Y${(Number(detail.assign_cost) || 0).toFixed(2)}§!/day`,
      height: 2 * LINE_H,
    },
    {
      kind: "separator",
      text: texts.equipment_bonuses ?? "",
      height: SEPARATOR_H,
    },
  ];
  if (initial.label)
    rows.push({
      kind: "modifier",
      text: `§Y${initial.label}§!`,
      height: LINE_H,
    });
  for (const equipment of guiArray(initial.equipment))
    rows.push({ kind: "equipment", equipment, height: 60 });
  let top = 0;
  return rows.map((row) => {
    const placed = { ...row, top };
    top += row.height;
    return placed;
  });
}

export interface TreeLine {
  x: number;
  y: number;

  length: number;
  vertical: boolean;
  dotted: boolean;
}

export function treeLines(
  traits: readonly GuiIndustrialOrgTrait[],
): TreeLine[] {
  const safeTraits = guiArray(traits);
  const at = new Map(safeTraits.map((trait) => [trait.token, trait]));
  const out: TreeLine[] = [];
  for (const child of safeTraits) {
    const links = [
      ...guiArray(child.parents).map((token) => ({ token, dotted: false })),
      ...guiArray(child.any_parent).map((token) => ({ token, dotted: true })),
    ];
    for (const { token, dotted } of links) {
      const parent = at.get(token);
      if (!parent) continue;
      const px = parent.x * CELL_W + LINE_OUT.x;
      const bus = parent.y * CELL_H + LINE_OUT.y;
      const cx = child.x * CELL_W + LINE_OUT.x;
      if (child.y === parent.y) {
        const left = Math.min(parent.x, child.x) * CELL_W + CELL_W - 8;
        const right = Math.max(parent.x, child.x) * CELL_W + 8;
        out.push({
          x: left,
          y: parent.y * CELL_H + 40,
          length: right - left,
          vertical: false,
          dotted,
        });
        continue;
      }

      out.push({
        x: px,
        y: parent.y * CELL_H + LINE_OUT.y - 12,
        length: 12,
        vertical: true,
        dotted,
      });
      if (cx !== px)
        out.push({
          x: Math.min(px, cx),
          y: bus,
          length: Math.abs(cx - px),
          vertical: false,
          dotted,
        });
      const enter = child.y * CELL_H + LINE_IN_Y;
      out.push({
        x: cx,
        y: Math.min(bus, enter),
        length: Math.abs(enter - bus),
        vertical: true,
        dotted,
      });
    }
  }
  return out;
}

export function exclusivePairs(
  traits: readonly GuiIndustrialOrgTrait[],
): [GuiIndustrialOrgTrait, GuiIndustrialOrgTrait][] {
  const safeTraits = guiArray(traits);
  const at = new Map(safeTraits.map((trait) => [trait.token, trait]));
  const out: [GuiIndustrialOrgTrait, GuiIndustrialOrgTrait][] = [];
  for (const trait of safeTraits) {
    for (const token of guiArray(trait.exclusive)) {
      const other = at.get(token);
      if (other && other.y === trait.y && trait.x < other.x)
        out.push([trait, other]);
    }
  }
  return out;
}

const BONUS_ICONS: Readonly<Record<string, string>> = {
  design: "GFX_design_team_icon",
  production: "GFX_industrial_manufacturer_icon",
  organization: "GFX_organization_modifier_icon",
};

function TraitEntry({
  entry,
  trait,
  sprites,
  left,
  top,
}: {
  entry: GuiWindow;
  trait: GuiIndustrialOrgTrait;
  sprites: Record<string, GuiSprite>;
  left: number;
  top: number;
}) {
  const edit = useEditCanvas();
  const button = trait.special
    ? "GFX_country_spefific_org_trait_button"
    : "GFX_industrial_org_trait_button";
  const icon = trait.icon && sprites[trait.icon] ? trait.icon : null;
  const node = useMemo(() => {
    const clone = structuredClone(entry.node) as GuiNode;
    const walk = (at: GuiNode) => {
      at.children = (at.children ?? []).filter(
        (child) =>
          child.name !== "queue_number_window" && child.name !== "ai_value",
      );
      for (const child of at.children) {
        if (child.name === "icon_frame") {
          child.sprite = button;
          child.frame = 3;
        }
        if (child.name === "trait_icon") child.sprite = icon;
        walk(child);
      }
    };
    walk(clone);
    assignIds(clone);
    return clone;
  }, [entry.node, button, icon]);
  const merged = useMemo(() => {
    const out = { ...entry.sprites };
    if (sprites[button]) out[button] = sprites[button];
    if (icon) out[icon] = sprites[icon];
    return out;
  }, [entry.sprites, sprites, button, icon]);
  const rects = useMemo(
    () => measureWindow(node, merged, 1920, 1080),
    [node, merged],
  );
  const box = rects["name:bonus_icon_set"];
  const icons = guiArray(trait.bonuses)
    .map((bonus) => BONUS_ICONS[bonus])
    .filter((name) => sprites[name] ?? entry.sprites[name]);

  const start = box ? box.x + (box.w - icons.length * 23) / 2 : 0;
  return (
    <div style={{ position: "absolute", left, top }}>
      <GuiCanvas
        node={node}
        {...edit(
          `${TRAIT}:${trait.token}`,
          { ...entry, sprites: merged },
          node,
          merged,
        )}
        texts={entry.texts}
        fonts={entry.fonts}
        hidden={NO_HIDDEN}
        isolateId={null}
        transparent
        baseW={CELL_W}
        baseH={CELL_H}
      />
      {box &&
        icons.map((name, index) => {
          const sprite = (sprites[name] ?? entry.sprites[name]) as GuiSprite;
          return (
            <div
              key={name}
              style={frameStyle(
                sprite,
                start + index * 23 + (23 - sprite.width) / 2,
                box.y + (box.h - sprite.height) / 2,
                { name },
              )}
            />
          );
        })}
    </div>
  );
}

const LINE_TINT = "rgb(161 177 219)";

export function lineStyle(
  line: TreeLine,
  sprites: Record<string, GuiSprite>,
): CSSProperties {
  const name = line.dotted
    ? "GFX_trait_line_dotted_tile_8x8"
    : line.vertical
      ? "GFX_trait_line_vertical_tile_8x8"
      : "GFX_trait_line_horizontal_tile_8x8";
  const sprite = sprites[name];
  const url = sprite ? spriteUrlOf(sprite, name) : null;
  const image = url ? `url("${url}")` : undefined;
  const repeat = line.vertical ? "repeat-y" : "repeat-x";

  return {
    position: "absolute",
    left: line.x,
    top: line.y,
    width: line.vertical ? 8 : line.length + 8,
    height: line.vertical ? line.length + 8 : 8,
    backgroundImage: image,
    backgroundRepeat: repeat,
    backgroundSize: "8px 8px",
    backgroundColor: LINE_TINT,
    backgroundBlendMode: "multiply",
    maskImage: image,
    maskRepeat: repeat,
    maskSize: "8px 8px",
    WebkitMaskImage: image,
    WebkitMaskRepeat: repeat,
    WebkitMaskSize: "8px 8px",
    pointerEvents: "none",
  };
}

function ExclusiveMark({
  entry,
  left,
  top,
  width,
}: {
  entry: GuiWindow;
  left: number;
  top: number;
  width: number;
}) {
  const link = entry.sprites["GFX_focus_link_exclusive"];
  const line = entry.sprites["GFX_focus_exclusive_line1"];
  if (!link) return null;
  const frames = Math.max(1, link.frames || 1);
  const w = link.width / frames;
  return (
    <div style={{ position: "absolute", left, top, pointerEvents: "none" }}>
      {line && (
        <div
          style={{
            ...frameStyle(line, 16, 10, { name: "GFX_focus_exclusive_line1" }),
            width: Math.max(0, width - 32),
            backgroundSize: `${Math.max(0, width - 32)}px ${line.height}px`,
          }}
        />
      )}
      <div
        style={frameStyle(link, 0, 0, {
          frame: 2,
          name: "GFX_focus_link_exclusive",
        })}
      />
      <div
        style={frameStyle(link, width - w, 0, {
          frame: 3,
          name: "GFX_focus_link_exclusive",
        })}
      />
      <div
        style={frameStyle(link, (width - w) / 2, 0, {
          frame: 1,
          name: "GFX_focus_link_exclusive",
        })}
      />
    </div>
  );
}

export function IndustrialOrgDetailLayer({
  win,
  projectId,
  tag,
  date,
  token,
  tab,
  onEditTrait,
}: {
  win: GuiWindow;
  projectId: number;
  tag: string;
  date?: string | null;
  token: string | null;
  tab: MioTab;

  onEditTrait?: (token: string) => void;
}) {
  const tLayer = useTranslations("gameViewResearch.layer");
  const active = win.name === MIO_DETAIL_WINDOW;
  const detail = useIndustrialOrg(projectId, active ? tag : null, date, token);
  const want = (name: string, when = true) => (active && when ? name : null);
  const tree = useTemplateWindow(projectId, want(TREE, tab === "traits"));
  const trait = useTemplateWindow(projectId, want(TRAIT, tab === "traits"));
  const header = useTemplateWindow(projectId, want(HEADER, tab === "traits"));
  const exclusive = useTemplateWindow(
    projectId,
    want(EXCLUSIVE, tab === "traits"),
  );
  const separator = useTemplateWindow(projectId, want(SEPARATOR));
  const modifier = useTemplateWindow(projectId, want(MODIFIER));
  const equipment = useTemplateWindow(projectId, want(EQUIPMENT));
  const rects = useMemo(
    () => (active ? measureWindow(win.node, win.sprites, 1920, 1080) : {}),
    [active, win],
  );
  const treeNode = useMemo(() => {
    if (!tree.data || !detail.data) return null;
    const clone = structuredClone(tree.data.node) as GuiNode;

    if (clone.background)
      clone.background = {
        ...clone.background,
        sprite: detail.data.background,
      };
    assignIds(clone);
    return clone;
  }, [tree.data, detail.data]);
  const treeSprites = useMemo(
    () =>
      tree.data && detail.data
        ? { ...guiRecord(tree.data.sprites), ...guiRecord(detail.data.sprites) }
        : {},
    [tree.data, detail.data],
  );
  const treeRects = useMemo(
    () => (treeNode ? measureWindow(treeNode, treeSprites, 1920, 1080) : {}),
    [treeNode, treeSprites],
  );
  const lines = useMemo(
    () => treeLines(guiArray(detail.data?.traits)),
    [detail.data],
  );
  const pairs = useMemo(
    () => exclusivePairs(guiArray(detail.data?.traits)),
    [detail.data],
  );
  const rows = useMemo(
    () => (detail.data ? bonusRows(detail.data) : []),
    [detail.data],
  );
  const edit = useEditCanvas();
  if (!active || !detail.data) return null;
  const data = detail.data;
  const headers = guiArray(data.headers);
  const traits = guiArray(data.traits);
  const sprites = guiRecord(data.sprites);

  const list = rects["name:aggregated_bonus_list"];
  const slot = rects["name:tree_list"];
  const scroller = treeRects["name:scrollbar_window"];
  const traitsAt = treeRects["name:traits_window"];
  const headerAt = treeRects["name:tree_header"];
  return (
    <div
      style={{ position: "absolute", left: 0, top: 0, pointerEvents: "none" }}
    >
      {list && separator.data && modifier.data && equipment.data && (
        <div style={{ position: "absolute", left: list.x, top: list.y }}>
          {rows.map((row, index) => {
            const top = row.top;
            if (row.kind === "separator") {
              return (
                <Item
                  key={index}
                  entry={separator.data as GuiWindow}
                  top={top + 5}
                  texts={{ separator_name: row.text ?? "" }}
                  width={250}
                  height={30}
                />
              );
            }
            if (row.kind === "modifier") {
              return (
                <Item
                  key={index}
                  entry={modifier.data as GuiWindow}
                  top={top}
                  texts={{ modifier_text: row.text ?? "" }}
                  width={250}
                  height={row.height}
                  grow
                />
              );
            }
            const group = row.equipment as GuiIndustrialOrgEquipment;
            return (
              <Item
                key={index}
                entry={equipment.data as GuiWindow}
                top={top}
                texts={{
                  equipment_type_text: group.label ?? group.group,
                  "design_team_bonuses_window/design_team_bonuses_count":
                    String(group.design),
                  "manufacturer_bonuses_window/manufacturer_bonuses_count":
                    String(group.production),
                }}
                swap={{
                  equipment_type_icon: {
                    name: group.icon,
                    sprite: group.icon_sprite,
                  },
                }}
                width={260}
                height={60}
              />
            );
          })}
        </div>
      )}
      {tab === "traits" && slot && treeNode && tree.data && (
        <div style={{ position: "absolute", left: slot.x, top: slot.y }}>
          <GuiCanvas
            node={treeNode}
            {...edit(
              TREE,
              { ...tree.data, sprites: treeSprites },
              treeNode,
              treeSprites,
            )}
            texts={tree.data.texts}
            fonts={tree.data.fonts}
            hidden={NO_HIDDEN}
            isolateId={null}
            transparent
            baseW={945}
            baseH={665}
          />
          {header.data &&
            headerAt &&
            headers.map((title) => (
              <div
                key={`${title.text}@${title.x}`}
                style={{
                  position: "absolute",
                  left: headerAt.x + title.x * CELL_W,
                  top: headerAt.y,
                }}
              >
                <Item
                  entry={header.data as GuiWindow}
                  top={0}
                  texts={{ flavortext: title.text }}
                  width={250}
                  height={80}
                />
              </div>
            ))}
          {scroller && traitsAt && (
            <div
              style={{
                position: "absolute",
                left: scroller.x,
                top: scroller.y,
                width: scroller.w,
                height: scroller.h,
                overflow: "hidden",
              }}
            >
              <div
                style={{
                  position: "absolute",
                  left: traitsAt.x - scroller.x,
                  top: traitsAt.y - scroller.y,
                }}
              >
                {lines.map((line, index) => (
                  <div key={index} style={lineStyle(line, sprites)} />
                ))}
                {exclusive.data &&
                  pairs.map(([left, right]) => (
                    <ExclusiveMark
                      key={`${left.token}-${right.token}`}
                      entry={exclusive.data as GuiWindow}
                      left={left.x * CELL_W + EXCLUSIVE_AT.x}
                      top={left.y * CELL_H + EXCLUSIVE_AT.y}
                      width={(right.x - left.x) * CELL_W - 61}
                    />
                  ))}
                {trait.data &&
                  traits.map((item) => (
                    <TraitEntry
                      key={item.token}
                      entry={trait.data as GuiWindow}
                      trait={item}
                      sprites={sprites}
                      left={item.x * CELL_W}
                      top={item.y * CELL_H}
                    />
                  ))}
                {onEditTrait &&
                  traits.map((item) => (
                    <FieldHit
                      key={`edit-${item.token}`}
                      x={item.x * CELL_W}
                      y={item.y * CELL_H}
                      w={CELL_W}
                      h={CELL_H}
                      label={tLayer("editTrait", { name: item.label })}
                      onClick={() => onEditTrait(item.token)}
                    />
                  ))}
              </div>
            </div>
          )}
        </div>
      )}
    </div>
  );
}
