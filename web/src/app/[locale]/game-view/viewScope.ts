export type GameViewScope = Readonly<{
  projectId: number;
  tag: string;
  base: string | null;
  panel: string | null;
}>;

export type GameViewScopeChange = "none" | "project" | "country" | "surface";

export function gameViewScopeChange(
  previous: GameViewScope | null,
  current: GameViewScope,
): GameViewScopeChange {
  if (previous === null) return "none";
  if (previous.projectId !== current.projectId) return "project";
  if (previous.tag !== current.tag) return "country";
  if (previous.base !== current.base || previous.panel !== current.panel)
    return "surface";
  return "none";
}
