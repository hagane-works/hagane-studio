"use client";

import { useMemo, useState } from "react";
import useSWR from "swr";
import { useTranslations } from "use-intl";
import { guiArray, readGuiResponse } from "./guiResponse";
import type { GuiNode, GuiSprite, GuiWindow } from "@/lib/guiRenderApi";
import { withGameLanguage } from "@/lib/guiRenderApi";
import { GuiCanvas, assignIds, measureWindow } from "../gui-renderer/GuiCanvas";
import { useTemplateWindow } from "./CountrySelectLayer";
import { useEditCanvas } from "./editContext";
import type { TextSource } from "./editModel";
import { keySource, personPrinted } from "./textSources";
import { FieldHit } from "./FieldHit";
import { matchNamePath } from "./gameNavigation";
import { frameStyle } from "./spriteFrame";

export const ARMY_ROSTER = "armyleaderwindow";
export const NAVY_ROSTER = "navyleaderwindow";

export const ROSTERS: Readonly<
  Record<
    string,
    {
      entry: string;
      kind: "army" | "navy";
      portrait: string;
      stats: readonly string[];
    }
  >
> = {
  [ARMY_ROSTER]: {
    entry: "armyleaderentry",
    kind: "army",
    portrait: "leader_button",
    stats: [
      "attack_skill",
      "defense_skill",
      "planning_skill",
      "logistics_skill",
    ],
  },
  [NAVY_ROSTER]: {
    entry: "navyleaderentry",
    kind: "navy",
    portrait: "leader_icon",
    stats: [
      "attack_skill",
      "defense_skill",
      "maneuvering_skill",
      "coordination_skill",
    ],
  },
};

const ROW_H = 98;
const WHEEL_STEP = 98;
const NO_HIDDEN = new Set<string>();

export function opensCommanderDetails(
  name: string | null | undefined,
): boolean {
  return ["leader_button", "traits_btn", "promote_btn"].includes(name ?? "");
}

const HIDDEN_ROW_PARTS = new Set([
  "army_hq_icon",
  "leader_enroute",
  "generate_advisor_button",
  "advisor_role_icon",
  "advisor_role_traits_icon",
  "medal_collection",
  "country_flag",
  "faction_commander",
]);

export interface GuiRosterLeader {
  token: string;
  name: string;
  rank: string;
  rank_label?: string;
  portrait?: string;
  portrait_sprite?: GuiSprite;
  skills: Record<string, number>;
  traits: {
    token: string;
    label?: string;
    icon: string;
    icon_sprite?: GuiSprite;
  }[];
  capacity?: number;
}

export interface GuiCommanderRoster {
  tag: string;
  kind: string;
  items: GuiRosterLeader[];
  new_leader_cost: number;
}

export function useCommanderRoster(
  projectId: number,
  tag: string | null,
  date: string | null | undefined,
  kind: "army" | "navy" = "army",
) {
  return useSWR<GuiCommanderRoster>(
    tag
      ? ["game-view-commander-roster", projectId, tag, date ?? "", kind]
      : null,
    async () => {
      const query = new URLSearchParams({ tag: tag as string, kind });
      if (date) query.set("date", date);
      const response = await fetch(
        withGameLanguage(
          `/api/go/projects/${projectId}/gui-render/commander-roster?${query}`,
        ),
      );
      return readGuiResponse<GuiCommanderRoster>(response);
    },
    { revalidateOnFocus: false },
  );
}

function idOf(node: GuiNode | undefined): string {
  return (node as (GuiNode & { _id?: string }) | undefined)?._id ?? "";
}

export function newLeaderText(template: string, cost: number): string {
  return template.replace(/\$VALUE[^$]*\$/, `§Y${cost}§!`);
}

export function rosterWindow(
  win: GuiWindow,
  roster: GuiCommanderRoster | undefined,
): {
  win: GuiWindow;
  texts: Record<string, string>;
  sources: Record<string, TextSource>;
} {
  const node = structuredClone(win.node) as GuiNode;
  for (const box of matchNamePath(node, "show_assigned")) box.frame = 2;
  const texts: Record<string, string> = {};
  const sources: Record<string, TextSource> = {};
  const button = matchNamePath(node, "new_leader_button")[0];
  const key = button?.text ?? "";
  const template = win.texts[key]?.markup ?? win.texts[key]?.display;
  if (button && roster && template) {
    texts[idOf(button)] = newLeaderText(template, roster.new_leader_cost);

    const source = keySource(key, "everywhere");
    if (source) sources[idOf(button)] = source;
  }
  return { win: { ...win, node }, texts, sources };
}

function LeaderRow({
  entry,
  roster,
  leader,
  top,
  onEdit,
  onDetails,
}: {
  entry: GuiWindow;
  roster: (typeof ROSTERS)[string];
  leader: GuiRosterLeader;
  top: number;
  onEdit?: () => void;
  onDetails?: () => void;
}) {
  const t = useTranslations("gameViewEdit.politics");
  const edit = useEditCanvas();
  const portrait =
    leader.portrait && leader.portrait_sprite ? leader.portrait : null;
  const node = useMemo(() => {
    const clone = structuredClone(entry.node) as GuiNode;
    const walk = (at: GuiNode) => {
      at.children = (at.children ?? []).filter(
        (child) => !HIDDEN_ROW_PARTS.has(child.name ?? ""),
      );
      for (const child of at.children) {
        if (child.name === roster.portrait && portrait) child.sprite = portrait;

        if (child.name === "divisions_count")
          child.text = child.text || "divisions_count";
        walk(child);
      }
    };
    walk(clone);
    assignIds(clone);
    return clone;
  }, [entry.node, portrait, roster.portrait]);
  const sprites = useMemo(
    () =>
      portrait && leader.portrait_sprite
        ? { ...entry.sprites, [portrait]: leader.portrait_sprite }
        : entry.sprites,
    [entry.sprites, portrait, leader.portrait_sprite],
  );
  const texts = useMemo(() => {
    const out: Record<string, string> = {};
    const set = (path: string, text: string) => {
      const id = idOf(matchNamePath(node, path)[0]);
      if (id) out[id] = text;
    };
    const skill = (key: string) => String(leader.skills[key] ?? 0);
    set("title", leader.rank_label ?? "");
    set("name", leader.name);
    set("skill", skill("skill"));

    for (const stat of roster.stats)
      set(`leader_stat_${stat.replace(/_skill$/, "")}`, skill(stat));
    set("divisions_count", leader.capacity ? String(leader.capacity) : "");
    return out;
  }, [node, leader, roster.stats]);

  const printed = useMemo(
    () => personPrinted(node, "name", leader),
    [node, leader],
  );
  const rects = useMemo(
    () => measureWindow(node, sprites, 1920, 1080),
    [node, sprites],
  );
  const grid = rects["name:traits_grid"];
  const nameRect = rects["name:name"];
  const nameNode = matchNamePath(node, "name")[0];
  return (
    <div style={{ position: "absolute", left: 0, top }}>
      <GuiCanvas
        node={node}
        {...edit(
          `${roster.entry}:${leader.token}`,
          { ...entry, sprites },
          node,
          sprites,
          texts,
          printed,
        )}
        texts={entry.texts}
        fonts={entry.fonts}
        hidden={NO_HIDDEN}
        isolateId={null}
        transparent
        baseW={510}
        baseH={ROW_H}
        onActivate={(pressed) => {
          if (!opensCommanderDetails(pressed.name) || !onDetails) return false;
          onDetails();
          return true;
        }}
      />
      {grid &&
        guiArray(leader.traits).map((trait, index) =>
          trait.icon_sprite?.resolved ? (
            <div
              key={trait.token}
              title={trait.label || trait.token}
              style={{
                ...frameStyle(trait.icon_sprite, grid.x + index * 24, grid.y, {
                  name: trait.icon,
                }),
                pointerEvents: "auto",
              }}
            />
          ) : null,
        )}
      {onEdit && nameRect && (
        <FieldHit
          x={nameRect.x}
          y={nameRect.y - 18}
          w={nameNode?.maxwidth || 180}
          h={40}
          label={t("editPerson", { name: leader.name })}
          onClick={onEdit}
        />
      )}
    </div>
  );
}

export function CommanderRosterLayer({
  win,
  projectId,
  tag,
  date,
  onEditPerson,
  onDetails,
}: {
  win: GuiWindow;
  projectId: number;
  tag: string;
  date?: string | null;

  onEditPerson?: (token: string, name: string) => void;

  onDetails?: (token: string) => void;
}) {
  const kind = ROSTERS[win.name];
  const active = Boolean(kind);
  const roster = useCommanderRoster(
    projectId,
    active ? tag : null,
    date,
    kind?.kind,
  );
  const entry = useTemplateWindow(projectId, kind?.entry ?? null);
  const rects = useMemo(
    () => (active ? measureWindow(win.node, win.sprites, 1920, 1080) : {}),
    [active, win],
  );
  const [scroll, setScroll] = useState(0);
  const list = rects["name:leaders_grid"];
  const holder = rects["name:leaders"];
  if (!kind || !list || !holder || !entry.data || !roster.data) return null;

  const box = matchNamePath(win.node, "leaders")[0];
  const margin =
    (box?.raw?.margin as { top?: number; bottom?: number } | undefined) ?? {};
  const viewTop = holder.y + (Number(margin.top) || 0) + (list.y - holder.y);
  const viewH = Math.max(
    0,
    holder.y + holder.h - (Number(margin.bottom) || 0) - viewTop,
  );
  const leaders = guiArray(roster.data.items);
  const maxScroll = Math.max(0, leaders.length * ROW_H - viewH);
  const at = Math.min(scroll, maxScroll);
  return (
    <div
      style={{ position: "absolute", left: 0, top: 0, pointerEvents: "none" }}
    >
      <div
        style={{
          position: "absolute",
          left: list.x,
          top: viewTop,
          width: 510,
          height: viewH,
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
        {leaders.map((leader, index) => (
          <LeaderRow
            key={leader.token}
            entry={entry.data as GuiWindow}
            roster={kind}
            leader={leader}
            top={index * ROW_H - at}
            onEdit={
              onEditPerson
                ? () => onEditPerson(leader.token, leader.name)
                : undefined
            }

            onDetails={
              kind.kind === "army" && onDetails
                ? () => onDetails(leader.token)
                : onEditPerson
                  ? () => onEditPerson(leader.token, leader.name)
                  : undefined
            }
          />
        ))}
      </div>
    </div>
  );
}
