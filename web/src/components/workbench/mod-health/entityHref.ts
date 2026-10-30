import type { ModHealthEntityRef } from "@/lib/modHealthApi";

const KIND_TO_ROUTE: Record<string, string> = {
  focus: "focus-tree",
  event: "events",
  decision: "decisions",
  character: "characters",
  idea: "ideas",
  state: "states",
  history_file: "history-files",
  technology: "technology",
};

export function kindLabelKey(kind: string): string {
  switch (kind) {
    case "focus":
      return "kindFocus";
    case "event":
      return "kindEvent";
    case "decision":
      return "kindDecision";
    case "character":
      return "kindCharacter";
    case "idea":
      return "kindIdea";
    case "state":
      return "kindState";
    case "history_file":
      return "kindHistoryFile";
    case "technology":
      return "kindTechnology";
    default:
      return "kindEvent";
  }
}

export function entityHref(
  projectId: number,
  ref: ModHealthEntityRef | null,
): string | null {
  if (!ref) return null;
  const route = KIND_TO_ROUTE[ref.kind];
  if (!route) return null;
  return `/project/${projectId}/${route}`;
}
