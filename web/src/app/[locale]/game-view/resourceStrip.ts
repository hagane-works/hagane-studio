import type { GuiCountryStats, GuiNode } from "@/lib/guiRenderApi";
import { textColors } from "@/lib/referenceTables";

export function stripEntries(
  frames: Record<string, number>,
  stats: GuiCountryStats | undefined,
): [string, number][] {
  const surplus = stats?.resource_surplus;
  const amounts = surplus ?? stats?.resources ?? {};
  const names = surplus
    ? Object.keys(frames)
    : Object.keys(amounts).filter((name) => (amounts[name] ?? 0) > 0);
  return names
    .map((name): [string, number] => [name, amounts[name] ?? 0])
    .sort(
      (a, b) =>
        (frames[a[0]] ?? 99) - (frames[b[0]] ?? 99) || a[0].localeCompare(b[0]),
    );
}

export function amountColor(amount: number): string | undefined {
  if (amount > 0) return textColors().G;
  if (amount < 0) return textColors().R;
  return undefined;
}

export function stripSlot(
  grid: GuiNode | undefined,
  at: number,
  fallbackWidth: number,
): { x: number; y: number } {
  const raw = (grid?.raw ?? {}) as Record<string, unknown>;
  const slot = (raw.slotsize ?? {}) as Record<string, unknown>;
  const max = (raw.max_slots ?? {}) as Record<string, unknown>;
  const w = Number(slot.width) || fallbackWidth;
  const h = Number(slot.height) || 0;
  const across = Number(max.x ?? raw.max_slots_horizontal) || Infinity;
  const down = Number(max.y ?? raw.max_slots_vertical) || 1;
  if (String(raw.add_horizontal ?? "").toLowerCase() === "yes") {
    const columns = Number.isFinite(across) ? across : at + 1;
    return { x: (at % columns) * w, y: Math.floor(at / columns) * h };
  }
  return { x: Math.floor(at / down) * w, y: (at % down) * h };
}

export function nodeNamed(
  root: GuiNode | undefined,
  name: string,
): GuiNode | undefined {
  if (!root) return undefined;
  if (root.name === name) return root;
  for (const child of root.children ?? []) {
    const found = nodeNamed(child, name);
    if (found) return found;
  }
  return undefined;
}
