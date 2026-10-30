import type { EventOption, GameEvent } from "@/types";
import { draftText } from "./focusEdit";

export interface EventOptionDraft {
  key: string;
  text: string;

  source?: number;

  effects?: string;
  trigger?: string;
}

export interface EventMode {
  triggeredOnly: boolean;
  fireOnce: boolean;
  hidden: boolean;
  major: boolean;

  mtth: string;

  timeoutDays: string;
}

export interface EventDraft {
  title: string;
  desc: string;
  picture: string;
  options: EventOptionDraft[];

  trigger?: string;
  immediate?: string;
  mode?: EventMode;
}

export function isVariantText(
  value: GameEvent["title"] | GameEvent["description"],
): boolean {
  return Array.isArray(value);
}

function optionLetter(index: number): string {
  return index < 26 ? String.fromCharCode(97 + index) : `opt${index}`;
}

export function newOptionKey(
  eventId: string,
  taken: readonly string[],
): string {
  const used = new Set(taken);
  for (let i = 0; ; i++) {
    const key = `${eventId}.${optionLetter(i)}`;
    if (!used.has(key)) return key;
  }
}

export function eventDraft(
  row: GameEvent,
  shown: {
    title?: string;
    desc?: string;
    options: { name: string; label?: string }[];
  },
): EventDraft {
  const title = typeof row.title === "string" ? row.title : "";
  const desc = typeof row.description === "string" ? row.description : "";
  return {
    title: draftText(
      row.display_name_locs,
      row.title_en || undefined,
      shown.title ?? title,
    ),
    desc: draftText(
      row.description_locs,
      row.desc_en || undefined,
      shown.desc ?? desc,
    ),
    picture: row.picture ?? "",
    options: (row.options ?? []).map((option, i) => {
      const key = option.name || `${row.event_id}.${optionLetter(i)}`;
      const screen =
        shown.options.find((o) => o.name === key) ?? shown.options[i];
      return {
        key,
        text: draftText(
          option.display_name_locs,
          option.name_en,
          screen?.label ?? "",
        ),
        source: i,
        effects: option.effects ?? "",
        trigger: option.trigger ?? "",
      };
    }),
    trigger: row.trigger ?? "",
    immediate: row.immediate ?? "",
    mode: {
      triggeredOnly: row.is_triggered_only ?? true,
      fireOnce: !!row.fire_only_once,
      hidden: !!row.hidden,
      major: !!row.major,
      mtth: row.mean_time_to_happen ?? "",
      timeoutDays: row.timeout_days ? String(row.timeout_days) : "",
    },
  };
}

export type EventProblem = "timeoutDays" | "mtth";

export function eventProblem(draft: EventDraft): EventProblem | null {
  const mode = draft.mode;
  if (!mode) return null;
  if (mode.timeoutDays.trim() && !/^\d+$/.test(mode.timeoutDays.trim()))
    return "timeoutDays";

  if (!mode.triggeredOnly && !mode.mtth.trim()) return "mtth";
  return null;
}

const blankOption = (
  key: string,
  text: string,
  effects = "",
  trigger = "",
): EventOption => ({
  name: key,
  trigger,
  effects,
  hidden_effects: "",
  ai_chance: {},
  display_name_locs: { en: text },
});

export function eventPatch(
  row: GameEvent,
  draft: EventDraft,
  start: EventDraft,
): Partial<GameEvent> {
  const patch: Partial<GameEvent> = {};
  if (draft.title !== start.title)
    patch.display_name_locs = {
      ...(row.display_name_locs ?? {}),
      en: draft.title,
    };
  if (draft.desc !== start.desc)
    patch.description_locs = {
      ...(row.description_locs ?? {}),
      en: draft.desc,
    };
  if (draft.picture !== start.picture) patch.picture = draft.picture;
  if (draft.trigger !== undefined && draft.trigger !== start.trigger)
    patch.trigger = draft.trigger;
  if (draft.immediate !== undefined && draft.immediate !== start.immediate)
    patch.immediate = draft.immediate;
  if (draft.mode && start.mode) {
    const mode = draft.mode;
    const was = start.mode;
    if (mode.triggeredOnly !== was.triggeredOnly)
      patch.is_triggered_only = mode.triggeredOnly;
    if (mode.fireOnce !== was.fireOnce) patch.fire_only_once = mode.fireOnce;
    if (mode.hidden !== was.hidden) patch.hidden = mode.hidden;
    if (mode.major !== was.major) patch.major = mode.major;
    if (mode.mtth !== was.mtth) patch.mean_time_to_happen = mode.mtth;
    const days = (text: string) => Number(text.trim() || 0);
    if (days(mode.timeoutDays) !== days(was.timeoutDays))
      patch.timeout_days = days(mode.timeoutDays);
  }
  const startOf = new Map(start.options.map((o) => [o.source, o]));
  const same = (o: EventOptionDraft) => {
    const was = startOf.get(o.source);
    return (
      !!was &&
      o.text === was.text &&
      (o.effects ?? "") === (was.effects ?? "") &&
      (o.trigger ?? "") === (was.trigger ?? "")
    );
  };
  const optionsChanged =
    draft.options.length !== start.options.length ||
    draft.options.some(
      (o, i) => o.source !== start.options[i]?.source || !same(o),
    );
  if (optionsChanged) {
    patch.options = draft.options.map((o) => {
      if (o.source === undefined)
        return blankOption(o.key, o.text, o.effects ?? "", o.trigger ?? "");
      const own = row.options[o.source];
      if (same(o)) return own;
      const was = startOf.get(o.source);
      return {
        ...own,
        ...(o.text !== was?.text
          ? {
              display_name_locs: {
                ...(own.display_name_locs ?? {}),
                en: o.text,
              },
            }
          : {}),
        ...(o.effects !== undefined && o.effects !== was?.effects
          ? { effects: o.effects }
          : {}),
        ...(o.trigger !== undefined && o.trigger !== was?.trigger
          ? { trigger: o.trigger }
          : {}),
      };
    });
  }
  return patch;
}
