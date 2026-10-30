export const SELECTABLE_DLCS = [
  "No Compromise, No Surrender",
  "Arms Against Tyranny",
  "La Resistance",
  "By Blood Alone",
  "No Step Back",
  "Gotterdammerung",
  "Battle for the Bosporus",
  "Graveyard of Empires",
  "Trial of Allegiance",
  "Poland: United and Ready",
] as const;

export const INTEGRATED_DLCS = [
  "Together for Victory",
  "Death or Dishonor",
  "Waking the Tiger",
  "Man the Guns",
] as const;

export function normalizeDlcChoice(
  choice: readonly string[] | null | undefined,
): string[] | null {
  if (choice == null) return null;
  const chosen = new Set(choice);
  return SELECTABLE_DLCS.filter((name) => chosen.has(name));
}
