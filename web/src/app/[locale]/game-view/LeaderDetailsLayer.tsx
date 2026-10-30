"use client";

import { useMemo, useState } from "react";
import useSWR from "swr";
import { useTranslations } from "use-intl";
import { readGuiResponse } from "./guiResponse";
import type { GuiNode, GuiSprite, GuiWindow } from "@/lib/guiRenderApi";
import { withGameLanguage } from "@/lib/guiRenderApi";
import { GuiCanvas, assignIds, measureWindow } from "../gui-renderer/GuiCanvas";
import { measureLine, wrapLines } from "../gui-renderer/BitmapText";
import { useTemplateWindow } from "./CountrySelectLayer";
import { useEditCanvas } from "./editContext";
import type { PrintedText, TextSource } from "./editModel";
import { namedPrinted } from "./textSources";
import { FieldHit } from "./FieldHit";
import { matchNamePath } from "./gameNavigation";
import { lineStyle, type TreeLine } from "./IndustrialOrgDetailLayer";
import { frameStyle } from "./spriteFrame";

export const LEADER_DETAILS = "unitleader_trait_tree_window";
const TRAIT_NODE = "unit_leader_trait";
const ACTIVE_TRAIT = "active_trait";
const ACTIVE_ADVISOR = "active_advisor_trait";

const CELL_W = 210;
const CELL_H = 55;

const TREE_PAD = { x: 0, y: 0 };

const LINE_OUT = { x: 185, y: 15 };

const BAR_END = 160;

const EXCLUSIVE_AT = { x: 81, y: 43 };

const WHEEL_STEP = 40;

const ENTRY_H = 27;
const EFFECT_LINE_H = 16;
const SLOT_H = 42;
const EFFECT_WIDTH = 210;
const TITLE_GAP: Readonly<Record<string, number>> = {
  officer_corps_title: 21,
  personal_traits_title: 26,
  traits_title: 18,
};
const PICK_SLOTS = ["add_trait_0", "add_trait_1", "add_trait_2", "add_trait_3"];

const NAME_WIDTH = 170;
const NO_HIDDEN = new Set<string>();

export interface GuiLeaderTrait {
  token: string;
  label: string;
  icon: string;
  icon_sprite?: GuiSprite;
  effects?: string[];
}

export interface GuiLeaderTreeTrait extends GuiLeaderTrait {
  row: number;
  column: number;
  parents: string[];
  exclusive: string[];
  held: boolean;
}

export interface GuiLeaderDetails {
  tag: string;
  token: string;
  name?: string;
  rank?: string;
  rank_label?: string;
  portrait?: string;
  portrait_sprite?: GuiSprite;
  skills?: Record<string, number>;
  advisor?: {
    token: string;
    label: string;
    slot?: string;
    sprite?: number;
    effects: string[];
  };
  background?: GuiLeaderTrait[];
  traits?: GuiLeaderTrait[];
  pick_slots?: number;
  regular?: GuiLeaderTreeTrait[];
  terrain?: GuiLeaderTreeTrait[];
  sprites?: Record<string, GuiSprite>;
}

export function useLeaderDetails(
  projectId: number,
  tag: string | null,
  date: string | null | undefined,
  token: string | null,
) {
  return useSWR<GuiLeaderDetails>(
    tag && token
      ? ["game-view-leader-details", projectId, tag, date ?? "", token]
      : null,
    async () => {
      const query = new URLSearchParams({
        tag: tag as string,
        id: token as string,
      });
      if (date) query.set("date", date);
      const response = await fetch(
        withGameLanguage(
          `/api/go/projects/${projectId}/gui-render/leader-details?${query}`,
        ),
      );
      return readGuiResponse<GuiLeaderDetails>(response);
    },
    { revalidateOnFocus: false },
  );
}

function idOf(node: GuiNode | undefined): string {
  return (node as (GuiNode & { _id?: string }) | undefined)?._id ?? "";
}

export interface ColumnItem {
  kind: "title" | "advisor" | "trait" | "slot";

  name?: string;
  trait?: GuiLeaderTrait;
  top: number;
}

export function leftColumn(
  details: GuiLeaderDetails,
  linesOf: (effects: readonly string[]) => number,
): ColumnItem[] {
  const out: ColumnItem[] = [];
  let y = 6;
  const title = (name: string) => {
    out.push({ kind: "title", name, top: y });
    y += TITLE_GAP[name] ?? 20;
  };
  const entry = (kind: "advisor" | "trait", trait: GuiLeaderTrait) => {
    out.push({ kind, trait, top: y });
    y += ENTRY_H + EFFECT_LINE_H * linesOf(trait.effects ?? []);
  };
  title("officer_corps_title");
  if (details.advisor) entry("advisor", { ...details.advisor, icon: "" });
  else {
    out.push({ kind: "slot", name: "add_advisor_role", top: y });
    y += SLOT_H;
  }
  if (details.background?.length) {
    title("personal_traits_title");
    for (const trait of details.background) entry("trait", trait);
  }
  title("traits_title");
  for (const trait of details.traits ?? []) entry("trait", trait);
  for (const name of PICK_SLOTS.slice(0, details.pick_slots ?? 0)) {
    out.push({ kind: "slot", name, top: y });
    y += SLOT_H;
  }
  return out;
}

export function leaderDetailsWindow(
  win: GuiWindow,
  details: GuiLeaderDetails | undefined,
): {
  win: GuiWindow;
  hidden: string[];
  shown: string[];
  texts: Record<string, string>;
  sources: Record<string, TextSource>;
} {
  const node = structuredClone(win.node) as GuiNode;
  const portrait =
    details?.portrait && details.portrait_sprite ? details.portrait : null;
  const ids = (path: string) =>
    matchNamePath(node, path).map(idOf).filter(Boolean);
  const hidden: string[] = [];
  for (const child of node.children ?? []) {
    if (child.name === "unit_leader" && portrait) child.sprite = portrait;
    if (child.name === "traits_tabs") {
      for (const tab of child.children ?? [])
        if (tab.name === "leader_traits_tab") tab.frame = 2;
    }
    if (child.name === "active_traits") {
      for (const item of child.children ?? []) {
        if (COLUMN_PARTS.has(item.name ?? "")) {
          hidden.push(...ids(`active_traits/${item.name}`));
        }
      }
    }
  }

  const shown = ids("regular_traits_window");
  const texts: Record<string, string> = {};
  const sources: Record<string, TextSource> = {};
  if (details?.name) {
    const set = (path: string, text: string) =>
      ids(path).forEach((id) => (texts[id] = text));
    const skill = (key: string) => String(details.skills?.[key] ?? 0);
    set("title", details.rank_label ?? "");
    set("name", details.name);

    if (details.token)
      ids("name").forEach(
        (id) => (sources[id] = { kind: "person", token: details.token }),
      );
    set("skill", skill("skill"));
    set("leader_stat_attack", skill("attack_skill"));
    set("leader_stat_defense", skill("defense_skill"));
    set("leader_stat_planning", skill("planning_skill"));
    set("leader_stat_logistics", skill("logistics_skill"));
  }
  const sprites =
    portrait && details?.portrait_sprite
      ? { ...win.sprites, [portrait]: details.portrait_sprite }
      : win.sprites;
  return { win: { ...win, node, sprites }, hidden, shown, texts, sources };
}

const COLUMN_PARTS = new Set([
  "officer_corps_title",
  "personal_traits_title",
  "traits_title",
  "add_advisor_role",
  ...PICK_SLOTS,
]);

export function fitLine(
  font: Parameters<typeof measureLine>[0] | undefined,
  text: string,
  width: number,
): string {
  if (!font || measureLine(font, text) <= width) return text;
  let cut = text;
  while (cut && measureLine(font, `${cut}...`) > width) cut = cut.slice(0, -1);
  return `${cut}...`;
}

export function leaderTreeLines(
  traits: readonly GuiLeaderTreeTrait[],
): TreeLine[] {
  const at = new Map(traits.map((trait) => [trait.token, trait]));
  const cell = (trait: GuiLeaderTreeTrait) => ({
    x: TREE_PAD.x + trait.column * CELL_W,
    y: TREE_PAD.y + trait.row * CELL_H,
  });
  const out: TreeLine[] = [];
  for (const child of traits) {
    const parents = child.parents ?? [];
    const dotted = parents.length > 1;
    const to = cell(child);
    const bus = to.x - CELL_W + LINE_OUT.x;
    let linked = false;
    for (const token of parents) {
      const parent = at.get(token);
      if (!parent || parent.column >= child.column) continue;
      const from = cell(parent);
      out.push({
        x: from.x + BAR_END,
        y: from.y + LINE_OUT.y,
        length: bus - from.x - BAR_END,
        vertical: false,
        dotted,
      });
      if (from.y !== to.y) {
        out.push({
          x: bus,
          y: Math.min(from.y, to.y) + LINE_OUT.y,
          length: Math.abs(to.y - from.y),
          vertical: true,
          dotted,
        });
      }
      linked = true;
    }
    if (linked)
      out.push({
        x: bus,
        y: to.y + LINE_OUT.y,
        length: to.x - bus,
        vertical: false,
        dotted,
      });
  }
  return out;
}

export function leaderExclusivePairs(
  traits: readonly GuiLeaderTreeTrait[],
): [GuiLeaderTreeTrait, GuiLeaderTreeTrait][] {
  const out: [GuiLeaderTreeTrait, GuiLeaderTreeTrait][] = [];
  for (const trait of traits) {
    for (const other of traits) {
      const rules =
        (trait.exclusive ?? []).includes(other.token) ||
        (other.exclusive ?? []).includes(trait.token);
      if (rules && other.column === trait.column && other.row === trait.row + 1)
        out.push([trait, other]);
    }
  }
  return out;
}

function Piece({
  entry,
  left,
  top,
  texts,
  keys,
  swap,
  hide,
  frames,
  alphas,
  width,
  height,
}: {
  entry: GuiWindow;
  left: number;
  top: number;

  texts: Record<string, string>;

  keys?: Record<string, string>;

  swap?: Record<string, { name: string; sprite?: GuiSprite }>;

  hide?: readonly string[];

  frames?: Record<string, number>;

  alphas?: Record<string, number>;
  width: number;
  height: number;
}) {
  const edit = useEditCanvas();
  const node = useMemo(() => {
    const clone = structuredClone(entry.node) as GuiNode;

    clone.position = { x: 0, y: 0 };
    const walk = (at: GuiNode) => {
      at.children = (at.children ?? []).filter(
        (child) => !hide?.includes(child.name ?? ""),
      );
      for (const child of at.children) {
        const picture = swap?.[child.name ?? ""];
        if (picture?.sprite) child.sprite = picture.name;
        const frame = frames?.[child.name ?? ""];
        if (frame) child.frame = frame;
        const alpha = alphas?.[child.name ?? ""];
        if (alpha !== undefined) child.raw = { ...child.raw, alpha };
        walk(child);
      }
    };
    walk(clone);
    assignIds(clone);
    return clone;
  }, [entry.node, swap, hide, frames, alphas]);
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
  const printed = useMemo(() => {
    const out: Record<string, PrintedText> = {};
    for (const [path, key] of Object.entries(keys ?? {}))
      Object.assign(out, namedPrinted(node, path, texts[path], key));
    return out;
  }, [node, texts, keys]);
  return (
    <div style={{ position: "absolute", left, top }}>
      <GuiCanvas
        node={node}
        {...edit(
          entry.name,
          { ...entry, sprites },
          node,
          sprites,
          overrides,
          printed,
        )}
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

const NEW_ICON = ["new"];
const NO_TEXTS: Record<string, string> = {};

const HELD_PARTS = ["disabled_bg", "pickable_bg"];
const FREE_PARTS = ["pickable_bg", "picked_bg"];

const DISABLED_ALPHA: Record<string, number> = { icon: 0.42, name: 0.42 };

const SLOT_ALPHA: Record<string, number> = { title: 0.3 };

function TraitTree({
  traits,
  box,
  node,
  sprites,
}: {
  traits: readonly GuiLeaderTreeTrait[];
  box: { x: number; y: number; w: number; h: number };
  node: GuiWindow;
  sprites: Record<string, GuiSprite>;
}) {
  const [scroll, setScroll] = useState(0);
  const lines = useMemo(() => leaderTreeLines(traits), [traits]);
  const cells = useMemo(
    () =>
      traits.map((trait) => ({
        trait,
        texts: { name: trait.label },
        swap: { icon: { name: trait.icon, sprite: trait.icon_sprite } },
      })),
    [traits],
  );
  const pairs = useMemo(() => leaderExclusivePairs(traits), [traits]);
  const rows = traits.reduce((most, trait) => Math.max(most, trait.row + 1), 0);
  const maxScroll = Math.max(0, TREE_PAD.y + rows * CELL_H - box.h);
  const at = Math.min(scroll, maxScroll);

  const mark = sprites["GFX_trait_tree_exclusive_vertical"];
  return (
    <div
      style={{
        position: "absolute",
        left: box.x,
        top: box.y,
        width: box.w,
        height: box.h,
        overflow: "hidden",
        pointerEvents: "auto",
      }}
      onWheel={(e) => {
        const step = e.deltaY > 0 ? WHEEL_STEP : -WHEEL_STEP;
        setScroll((now) =>
          Math.min(maxScroll, Math.max(0, Math.min(now, maxScroll) + step)),
        );
      }}
    >
      <div style={{ position: "absolute", left: 0, top: -at }}>
        {lines.map((line, index) => (
          <div key={index} style={lineStyle(line, sprites)} />
        ))}
        {cells.map(({ trait, texts, swap }) => (
          <Piece
            key={trait.token}
            entry={node}
            left={TREE_PAD.x + trait.column * CELL_W}
            top={TREE_PAD.y + trait.row * CELL_H}
            texts={texts}
            keys={{ name: trait.token }}
            swap={swap}
            hide={trait.held ? HELD_PARTS : FREE_PARTS}
            alphas={trait.held ? undefined : DISABLED_ALPHA}
            width={CELL_W}
            height={CELL_H}
          />
        ))}
        {mark &&
          pairs.map(([upper]) => {
            const w = mark.width * 0.6;
            const h = mark.height * 0.6;
            return (
              <div
                key={`x:${upper.token}`}
                style={frameStyle(
                  mark,
                  TREE_PAD.x + upper.column * CELL_W + EXCLUSIVE_AT.x - w / 2,
                  TREE_PAD.y + upper.row * CELL_H + EXCLUSIVE_AT.y - h / 2,
                  { scale: 0.6, name: "GFX_trait_tree_exclusive_vertical" },
                )}
                aria-hidden
              />
            );
          })}
      </div>
    </div>
  );
}

export function LeaderDetailsLayer({
  win,
  projectId,
  tag,
  date,
  token,
  onEditPerson,
}: {
  win: GuiWindow;
  projectId: number;
  tag: string;
  date?: string | null;
  token: string | null;

  onEditPerson?: (token: string, name: string) => void;
}) {
  const t = useTranslations("gameViewEdit.politics");
  const active = win.name === LEADER_DETAILS;
  const details = useLeaderDetails(projectId, active ? tag : null, date, token);
  const want = (name: string) => (active ? name : null);
  const traitNode = useTemplateWindow(projectId, want(TRAIT_NODE));
  const activeTrait = useTemplateWindow(projectId, want(ACTIVE_TRAIT));
  const activeAdvisor = useTemplateWindow(projectId, want(ACTIVE_ADVISOR));
  const rects = useMemo(
    () => (active ? measureWindow(win.node, win.sprites, 1920, 1080) : {}),
    [active, win],
  );
  const effectFont =
    activeTrait.data?.fonts["hoi4_typewriter16_inverted"]?.bitmap;
  const column = useMemo(() => {
    if (!details.data?.name) return [];
    return leftColumn(details.data, (effects) =>
      effects.reduce(
        (sum, line) =>
          sum +
          (effectFont ? wrapLines(effectFont, line, EFFECT_WIDTH).length : 1),
        0,
      ),
    );
  }, [details.data, effectFont]);
  const treeSprites = useMemo(
    () => ({
      ...(traitNode.data?.sprites ?? {}),
      ...(details.data?.sprites ?? {}),
    }),
    [traitNode.data, details.data],
  );
  if (!active || !details.data?.name) return null;
  const data = details.data;
  const nameFont = activeTrait.data?.fonts["hoi_20b"]?.bitmap;
  const left = rects["name:active_traits"];
  const regular = rects["name:regular_traits"];
  const terrain = rects["name:terrain_traits"];
  const nameRect = rects["name:name"];

  const xp = rects["name:leader_xp_progressbar_position"];
  const xpTrack = data.sprites?.["GFX_leader_exp_bg"];
  return (
    <div
      style={{ position: "absolute", left: 0, top: 0, pointerEvents: "none" }}
    >
      {left &&
        column.map((item) => {
          const part = item.name
            ? matchNamePath(win.node, `active_traits/${item.name}`)[0]
            : undefined;
          if (part && item.name) {
            return (
              <Piece
                key={item.name}
                entry={{ ...win, node: part }}
                left={left.x + (Number(part.position?.x) || 0)}
                top={left.y + item.top}
                texts={NO_TEXTS}
                alphas={item.kind === "slot" ? SLOT_ALPHA : undefined}
                width={265}
                height={SLOT_H}
              />
            );
          }
          if (item.kind === "advisor" && activeAdvisor.data && data.advisor) {
            return (
              <Piece
                key="advisor"
                entry={activeAdvisor.data}
                left={left.x}
                top={left.y + item.top}
                texts={{
                  name: fitLine(nameFont, data.advisor.label, NAME_WIDTH),
                  buff: data.advisor.effects.join("\n"),
                }}
                frames={
                  data.advisor.sprite
                    ? { idea_traits: data.advisor.sprite }
                    : undefined
                }
                width={260}
                height={ENTRY_H + EFFECT_LINE_H * 4}
              />
            );
          }
          if (item.kind === "trait" && item.trait && activeTrait.data) {
            const trait = item.trait;
            return (
              <Piece
                key={trait.token}
                entry={activeTrait.data}
                left={left.x}
                top={left.y + item.top}
                texts={{
                  name: fitLine(nameFont, trait.label, NAME_WIDTH),
                  buff: (trait.effects ?? []).join("\n"),
                }}
                keys={{ name: trait.token }}
                swap={{ icon: { name: trait.icon, sprite: trait.icon_sprite } }}
                hide={NEW_ICON}
                width={260}
                height={ENTRY_H + EFFECT_LINE_H * 8}
              />
            );
          }
          return null;
        })}
      {xp && xpTrack && (
        <div
          style={frameStyle(xpTrack, xp.x, xp.y, { name: "GFX_leader_exp_bg" })}
        />
      )}
      {regular && traitNode.data && (
        <TraitTree
          traits={data.regular ?? []}
          box={regular}
          node={traitNode.data}
          sprites={treeSprites}
        />
      )}
      {terrain && traitNode.data && (
        <TraitTree
          traits={data.terrain ?? []}
          box={terrain}
          node={traitNode.data}
          sprites={treeSprites}
        />
      )}
      {onEditPerson && nameRect && (
        <FieldHit
          x={nameRect.x}
          y={nameRect.y - 18}
          w={150}
          h={40}
          label={t("editPerson", { name: data.name ?? data.token })}
          onClick={() => onEditPerson(data.token, data.name ?? data.token)}
        />
      )}
    </div>
  );
}
