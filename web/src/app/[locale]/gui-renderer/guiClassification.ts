import type { GuiNode, GuiText } from "@/lib/guiRenderApi";

export type EditAxis = "art" | "text" | "dynamic" | "skeleton";

export const EDIT_AXIS_ORDER: EditAxis[] = [
  "art",
  "text",
  "dynamic",
  "skeleton",
];

export interface AxisStyle {
  badge: string;

  dot: string;

  outline: string;

  fill: string;
}

export const AXIS_STYLE: Record<EditAxis, AxisStyle> = {
  art: {
    badge: "border-emerald-800 text-emerald-300",
    dot: "bg-emerald-400",
    outline: "rgba(52,211,153,0.85)",
    fill: "rgba(52,211,153,0.16)",
  },
  text: {
    badge: "border-sky-800 text-sky-300",
    dot: "bg-sky-400",
    outline: "rgba(56,189,248,0.85)",
    fill: "rgba(56,189,248,0.16)",
  },
  dynamic: {
    badge: "border-amber-800 text-amber-300",
    dot: "bg-amber-400",
    outline: "rgba(251,191,36,0.85)",
    fill: "rgba(251,191,36,0.16)",
  },
  skeleton: {
    badge: "border-neutral-700 text-neutral-400",
    dot: "bg-neutral-500",
    outline: "rgba(120,120,128,0.7)",
    fill: "rgba(120,120,128,0.12)",
  },
};

export function classifyNode(
  node: GuiNode,
  texts?: Record<string, GuiText>,
): EditAxis {
  if (node.sprite) return "art";

  if (node.text) {
    const info = texts?.[node.text];
    if (info?.kind === "runtime") return "dynamic";

    return "text";
  }

  return "skeleton";
}
