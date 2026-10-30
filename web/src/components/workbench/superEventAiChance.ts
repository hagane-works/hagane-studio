export const DEFAULT_SUPER_EVENT_AI_CHANCE = 1;

export function parseSuperEventAiChance(raw: string): number {
  if (raw.trim() === "") return DEFAULT_SUPER_EVENT_AI_CHANCE;
  const parsed = Number(raw);
  return Number.isFinite(parsed) ? parsed : DEFAULT_SUPER_EVENT_AI_CHANCE;
}
