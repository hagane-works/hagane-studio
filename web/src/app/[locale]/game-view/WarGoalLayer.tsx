"use client";

import { useMemo } from "react";
import { useTranslations } from "use-intl";
import useSWR from "swr";
import { guiArray, guiRecord, readGuiResponse } from "./guiResponse";
import type { GuiNode, GuiWindow } from "@/lib/guiRenderApi";
import { withGameLanguage } from "@/lib/guiRenderApi";
import { assignIds, measureWindow } from "../gui-renderer/GuiCanvas";
import { useTemplateWindow } from "./CountrySelectLayer";
import { FieldHit } from "./FieldHit";
import { Filled } from "./FilledWindow";
import type { TextSource } from "./editModel";
import { matchNamePath } from "./gameNavigation";
import { GAME_SCROLLBAR } from "./PageScrollbar";
import { keySource } from "./textSources";

export const WARGOAL_WINDOW = "diplomacy_action_info_window";
const DIALOG = "generate_wargoal";
const TYPE_ENTRY = "diplomacy_create_wargoal_entry";
const STATE_ENTRY = "diplomacy_wargoal_state_entry";

const DIALOG_PARTS = [
  "wargoals/wargoals_grid",
  "additionals/additionals_grid",
  "additionals",
  "cancel_button",
  "send_button",
];

const ENTRY_W = 200;
const ENTRY_H = 30;

const CHOSEN_FRAME = 2;

export interface GuiWarGoalType {
  id: string;
  label: string;
  base_cost: number;
  per_state_cost: number;
  limit: number;
  threat: number;
  states: { id: number; name: string }[];
}

export interface GuiWarGoals {
  tag: string;
  target: string;
  types: GuiWarGoalType[];
  daily_pp: number;
  texts: {
    title?: string;
    no_wargoal?: string;
    select_type?: string;
    cost?: string;
  };

  text_keys?: {
    title?: string;
    no_wargoal?: string;
    select_type?: string;
    cost?: string;
  };
}

export interface WarGoalPick {
  type: string | null;
  states: number[];
}

export const NO_WARGOAL: WarGoalPick = { type: null, states: [] };

export function useWarGoals(
  projectId: number,
  tag: string | null,
  target: string | null,
  date: string | null | undefined,
) {
  return useSWR<GuiWarGoals>(
    tag && target
      ? ["game-view-wargoals", projectId, tag, target, date ?? ""]
      : null,
    async () => {
      const query = new URLSearchParams({
        tag: tag as string,
        target: target as string,
      });
      if (date) query.set("date", date);
      const response = await fetch(
        withGameLanguage(
          `/api/go/projects/${projectId}/gui-render/wargoal?${query}`,
        ),
      );
      return readGuiResponse<GuiWarGoals>(response);
    },
    { revalidateOnFocus: false },
  );
}

export function warGoalCost(
  type: GuiWarGoalType,
  states: number,
  dailyPP: number,
): { days: number; pp: number } {
  const days = type.base_cost + type.per_state_cost * states;
  return { days, pp: days * dailyPP };
}

export function warGoalCostText(
  format: string | undefined,
  days: number,
  pp: number,
): string {
  if (!format) return "";
  return format
    .replace(/\$DAILYCOST[^$]*\$/, `§Y${pp.toFixed(2)}§!`)
    .replace(/\$DAYS[^$]*\$/, `§Y${Math.round(days)}§!`);
}

export function pickType(data: GuiWarGoals, type: string): WarGoalPick {
  const kind = guiArray(data.types).find((entry) => entry.id === type);
  const states = guiArray(kind?.states);
  return { type, states: states.length ? [states[0].id] : [] };
}

export function toggleState(
  pick: WarGoalPick,
  type: GuiWarGoalType,
  state: number,
): WarGoalPick {
  if (pick.states.includes(state)) {
    return pick.states.length > 1
      ? { ...pick, states: pick.states.filter((id) => id !== state) }
      : pick;
  }
  if (type.limit > 0 && pick.states.length >= type.limit) return pick;
  return { ...pick, states: [...pick.states, state] };
}

function idOf(node: GuiNode | undefined): string {
  return (node as (GuiNode & { _id?: string }) | undefined)?._id ?? "";
}

export function warGoalWindow(
  win: GuiWindow,
  data: GuiWarGoals | undefined,
  pick: WarGoalPick,
): {
  win: GuiWindow;
  hidden: string[];
  texts: Record<string, string>;
  sources: Record<string, TextSource>;
} {
  const node = structuredClone(win.node) as GuiNode;
  assignIds(node);
  const hidden = (node.children ?? [])
    .filter((child) => child.name !== DIALOG)
    .map(idOf)
    .filter(Boolean);
  const texts: Record<string, string> = {};
  const sources: Record<string, TextSource> = {};

  const write = (path: string, text: string, key?: string) => {
    const source = keySource(key, "everywhere");
    for (const hit of matchNamePath(node, `${DIALOG}/${path}`)) {
      if (!idOf(hit)) continue;
      texts[idOf(hit)] = text;
      if (source) sources[idOf(hit)] = source;
    }
  };
  const type = guiArray(data?.types).find((entry) => entry.id === pick.type);
  const strings = guiRecord(data?.texts);
  const keys = guiRecord(data?.text_keys);
  write("title", strings.title ?? "", keys.title);
  if (!type || !data) {
    write("wargoal_warning", strings.no_wargoal ?? "", keys.no_wargoal);
    write("wargoal_name", strings.select_type ?? "", keys.select_type);
    write("description", "");
    write("threat/threat_value", "0%");
  } else {
    const names = guiArray(type.states)
      .filter((state) => pick.states.includes(state.id))
      .map((state) => state.name);
    const { days, pp } = warGoalCost(type, pick.states.length, data.daily_pp);
    write("wargoal_warning", "");

    write("wargoal_name", `${type.label} ${names.join(", ")}`.trim(), type.id);
    write("description", warGoalCostText(strings.cost, days, pp), keys.cost);
    write("threat/threat_value", "");
  }
  return { win: { ...win, node }, hidden, texts, sources };
}

function chosenEntry(entry: GuiWindow): GuiWindow {
  const node = structuredClone(entry.node) as GuiNode & {
    background?: GuiNode;
  };
  if (node.background)
    node.background = { ...node.background, frame: CHOSEN_FRAME };
  return { ...entry, node };
}

export function WarGoalLayer({
  win,
  projectId,
  tag,
  target,
  date,
  pick,
  onPick,
  onClose,
  onEditWarGoal,
}: {
  win: GuiWindow;
  projectId: number;
  tag: string;
  target: string | null;
  date?: string | null;
  pick: WarGoalPick;
  onPick: (pick: WarGoalPick) => void;
  onClose: () => void;
  editMode?: boolean;

  onEditWarGoal?: (id: string) => void;
}) {
  const tw = useTranslations("gameViewWorld.warGoal");
  const active = win.name === WARGOAL_WINDOW;
  const data = useWarGoals(
    projectId,
    active ? tag : null,
    active ? target : null,
    date,
  );
  const typeEntry = useTemplateWindow(projectId, active ? TYPE_ENTRY : null);
  const stateEntry = useTemplateWindow(projectId, active ? STATE_ENTRY : null);
  const chosenType = useMemo(
    () => (typeEntry.data ? chosenEntry(typeEntry.data) : undefined),
    [typeEntry.data],
  );
  const chosenState = useMemo(
    () => (stateEntry.data ? chosenEntry(stateEntry.data) : undefined),
    [stateEntry.data],
  );

  const rects = useMemo(() => {
    const out: Record<
      string,
      { x: number; y: number; w: number; h: number } | undefined
    > = {};
    if (!active) return out;
    const node = structuredClone(win.node) as GuiNode;
    const measured = measureWindow(node, win.sprites, 1920, 1080);
    for (const path of DIALOG_PARTS)
      out[path] = measured[idOf(matchNamePath(node, `${DIALOG}/${path}`)[0])];
    return out;
  }, [active, win]);
  if (!active || !data.data) return null;
  const goals = data.data;
  const goalTypes = guiArray(goals.types);
  const typesAt = rects["wargoals/wargoals_grid"];
  const statesAt = rects["additionals/additionals_grid"];
  const statesBox = rects["additionals"];
  const cancel = rects["cancel_button"];
  const send = rects["send_button"];
  const type = goalTypes.find((entry) => entry.id === pick.type);
  const states = guiArray(type?.states);
  const button = (
    rect: { x: number; y: number; w: number; h: number } | undefined,
    label: string,
  ) =>
    rect ? (
      <div
        role="button"
        aria-label={label}
        onClick={onClose}
        style={{
          position: "absolute",
          left: rect.x,
          top: rect.y,
          width: rect.w || 123,
          height: rect.h || 34,
          cursor: "pointer",
          pointerEvents: "auto",
        }}
      />
    ) : null;
  return (
    <div
      style={{ position: "absolute", left: 0, top: 0, pointerEvents: "none" }}
    >
      {typeEntry.data &&
        chosenType &&
        typesAt &&
        goalTypes.map((entry, at) => (
          <div key={entry.id}>
            <Filled
              entry={
                entry.id === pick.type
                  ? chosenType
                  : (typeEntry.data as GuiWindow)
              }
              left={typesAt.x}
              top={typesAt.y + at * ENTRY_H}
              width={ENTRY_W}
              height={ENTRY_H}
              texts={{ name: entry.label }}
              extraSprites={{}}
            />
            <div
              role="button"
              aria-label={`wargoal ${entry.id}`}
              onClick={() => onPick(pickType(goals, entry.id))}
              style={{
                position: "absolute",
                left: typesAt.x,
                top: typesAt.y + at * ENTRY_H,
                width: ENTRY_W,
                height: ENTRY_H,
                cursor: "pointer",
                pointerEvents: "auto",
              }}
            />
            {onEditWarGoal && (
              <FieldHit
                x={typesAt.x}
                y={typesAt.y + at * ENTRY_H}
                w={ENTRY_W}
                h={ENTRY_H}
                label={tw("hit", { name: entry.label })}
                onClick={() => {
                  onPick(pickType(goals, entry.id));
                  onEditWarGoal(entry.id);
                }}
              />
            )}
          </div>
        ))}
      {stateEntry.data && chosenState && type && statesAt && statesBox && (
        <div
          style={{
            position: "absolute",
            left: statesAt.x,
            top: statesAt.y,
            width: ENTRY_W + 4,
            height: Math.max(0, statesBox.y + statesBox.h - statesAt.y - 13),
            overflowY: "auto",
            overflowX: "hidden",
            ...GAME_SCROLLBAR,
            pointerEvents: "auto",
          }}
        >
          <div
            style={{ position: "relative", height: states.length * ENTRY_H }}
          >
            {states.map((state, at) => (
              <div key={state.id}>
                <Filled
                  entry={
                    pick.states.includes(state.id)
                      ? chosenState
                      : (stateEntry.data as GuiWindow)
                  }
                  left={0}
                  top={at * ENTRY_H}
                  width={ENTRY_W}
                  height={ENTRY_H}
                  texts={{ name: state.name }}
                  extraSprites={{}}
                />
                <div
                  role="button"
                  aria-label={`state ${state.name}`}
                  onClick={() => onPick(toggleState(pick, type, state.id))}
                  style={{
                    position: "absolute",
                    left: 0,
                    top: at * ENTRY_H,
                    width: ENTRY_W,
                    height: ENTRY_H,
                    cursor: "pointer",
                  }}
                />
              </div>
            ))}
          </div>
        </div>
      )}
      {button(cancel, "Cancel")}
      {button(send, "Send")}
    </div>
  );
}
