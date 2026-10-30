import type { EventOption, EventType, GameEvent } from "@/types";

const EVENT_TYPES = new Set<EventType>([
  "country_event",
  "news_event",
  "state_event",
  "unit_leader_event",
  "operative_leader_event",
]);

function stringValue(value: unknown): string {
  return typeof value === "string" ? value : "";
}

function numberValue(value: unknown, fallback = 0): number {
  return typeof value === "number" && Number.isFinite(value) ? value : fallback;
}

function recordValue(value: unknown): Record<string, unknown> {
  return value !== null && typeof value === "object" && !Array.isArray(value)
    ? (value as Record<string, unknown>)
    : {};
}

function normalizeEventOption(value: unknown): EventOption | null {
  if (value === null || typeof value !== "object" || Array.isArray(value)) {
    return null;
  }

  const raw = value as Record<string, unknown>;
  return {
    ...raw,
    name: stringValue(raw.name),
    trigger: stringValue(raw.trigger),
    effects: stringValue(raw.effects),
    hidden_effects: stringValue(raw.hidden_effects),
    ai_chance: recordValue(raw.ai_chance),
  } as EventOption;
}

export function normalizeGameEvent(value: GameEvent): GameEvent {
  const raw = value as GameEvent & Record<string, unknown>;
  const eventType = stringValue(raw.event_type) as EventType;

  return {
    ...value,
    event_id: stringValue(raw.event_id),
    event_number: numberValue(raw.event_number),
    event_type: EVENT_TYPES.has(eventType) ? eventType : "country_event",
    title:
      typeof raw.title === "string" || Array.isArray(raw.title)
        ? raw.title
        : "",
    description:
      typeof raw.description === "string" || Array.isArray(raw.description)
        ? raw.description
        : "",
    picture: stringValue(raw.picture),
    trigger: stringValue(raw.trigger),
    mean_time_to_happen: stringValue(raw.mean_time_to_happen),
    immediate: stringValue(raw.immediate),
    after: stringValue(raw.after),
    options: Array.isArray(raw.options)
      ? raw.options
          .map(normalizeEventOption)
          .filter((option): option is EventOption => option !== null)
      : [],
    timeout_days: numberValue(raw.timeout_days),
    raw_extras: recordValue(raw.raw_extras),
    country_tag: stringValue(raw.country_tag),
    scope_kind: stringValue(raw.scope_kind) || "own",
    shared_country_tags: Array.isArray(raw.shared_country_tags)
      ? raw.shared_country_tags.filter(
          (item): item is string => typeof item === "string",
        )
      : null,
    title_en: stringValue(raw.title_en),
    title_zh: stringValue(raw.title_zh),
    desc_en: stringValue(raw.desc_en),
    desc_zh: stringValue(raw.desc_zh),
    display_name_locs: recordValue(raw.display_name_locs) as Record<
      string,
      string
    >,
    description_locs: recordValue(raw.description_locs) as Record<
      string,
      string
    >,
    auto_fire_on_startup: raw.auto_fire_on_startup === true,
    auto_fire_after_days: numberValue(raw.auto_fire_after_days),
    on_action_attach: stringValue(raw.on_action_attach),
    source: stringValue(raw.source) || "created",
    created_at: stringValue(raw.created_at),
    updated_at: stringValue(raw.updated_at),
  };
}
