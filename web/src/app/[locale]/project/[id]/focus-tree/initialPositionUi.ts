import { isAmbiguousLegacyInitialShowPosition } from "@/lib/api/focus-tech";
import type { InitialShowPosition } from "@/types";

export type InitialPositionMode =
  "unresolved" | "none" | "focus" | "coordinates";

export interface UnresolvedInitialPosition {
  kind: "legacy_ambiguous" | "invalid";
  raw: string;
}

export interface InitialPositionEditorState {
  mode: InitialPositionMode;
  focusId: string;
  x: number;
  y: number;
  offsets: { x: number; y: number; trigger: string }[];
  unresolved: UnresolvedInitialPosition | null;
}

interface InitialPositionSource {
  initial_show_position: InitialShowPosition | null;
  initial_show_focus: string;
}

function emptyState(mode: InitialPositionMode): InitialPositionEditorState {
  return {
    mode,
    focusId: "",
    x: 0,
    y: 0,
    offsets: [],
    unresolved: null,
  };
}

function formatInvalidRaw(value: unknown, rawText: string): string {
  const preservedText = rawText.trim();
  if (preservedText) return preservedText;
  if (typeof value === "string") return value;
  try {
    return JSON.stringify(value) ?? String(value);
  } catch {
    return String(value);
  }
}

export function getInitialPositionEditorState(
  tree: InitialPositionSource | null | undefined,
): InitialPositionEditorState {
  if (!tree) return emptyState("none");

  const position = tree.initial_show_position;
  const legacy = (tree.initial_show_focus || "").trim();

  if (position?.kind === "focus") {
    return {
      ...emptyState("focus"),
      focusId: position.focus_id,
    };
  }

  if (position?.kind === "coordinates") {
    return {
      ...emptyState("coordinates"),
      x: position.x,
      y: position.y,
      offsets: position.offsets.map((offset) => ({ ...offset })),
    };
  }

  if (position?.kind === "legacy_ambiguous") {
    return {
      ...emptyState("unresolved"),
      unresolved: { kind: "legacy_ambiguous", raw: position.raw },
    };
  }

  if (position?.kind === "invalid") {
    return {
      ...emptyState("unresolved"),
      unresolved: {
        kind: "invalid",
        raw: formatInvalidRaw(position.raw, position.raw_text),
      },
    };
  }

  if (legacy && isAmbiguousLegacyInitialShowPosition(legacy)) {
    return {
      ...emptyState("unresolved"),
      unresolved: { kind: "legacy_ambiguous", raw: legacy },
    };
  }

  if (legacy) {
    return {
      ...emptyState("focus"),
      focusId: legacy,
    };
  }

  return emptyState("none");
}
