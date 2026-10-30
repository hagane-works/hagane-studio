import { create } from "zustand";

import type { UiPanelControl } from "@/types";

interface UiPanelSelectionState {
  selectedPath: string | null;
  setSelectedPath: (path: string | null) => void;
}

export const useUiPanelSelection = create<UiPanelSelectionState>((set) => ({
  selectedPath: null,
  setSelectedPath: (path) => set({ selectedPath: path }),
}));

function parsePath(path: string): number[] {
  return path
    .split("/")
    .map((seg) => Number.parseInt(seg, 10))
    .filter((n) => Number.isInteger(n) && n >= 0);
}

export function getControlAtPath(
  controls: UiPanelControl[],
  path: string | null,
): UiPanelControl | null {
  if (!path) return null;
  let list: UiPanelControl[] = controls;
  let node: UiPanelControl | null = null;
  for (const idx of parsePath(path)) {
    node = list[idx] ?? null;
    if (!node) return null;
    list = node.children ?? [];
  }
  return node;
}

export function updateControlAtPath(
  controls: UiPanelControl[],
  path: string,
  partial: Partial<UiPanelControl>,
): UiPanelControl[] {
  const indices = parsePath(path);
  if (indices.length === 0) return controls;

  const walk = (list: UiPanelControl[], depth: number): UiPanelControl[] => {
    const idx = indices[depth];
    if (idx == null || idx >= list.length) return list;
    return list.map((ctrl, i) => {
      if (i !== idx) return ctrl;
      if (depth === indices.length - 1) return { ...ctrl, ...partial };
      return { ...ctrl, children: walk(ctrl.children ?? [], depth + 1) };
    });
  };
  return walk(controls, 0);
}

export function removeControlAtPath(
  controls: UiPanelControl[],
  path: string,
): UiPanelControl[] {
  const indices = parsePath(path);
  if (indices.length === 0) return controls;

  const walk = (list: UiPanelControl[], depth: number): UiPanelControl[] => {
    const idx = indices[depth];
    if (idx == null || idx >= list.length) return list;
    if (depth === indices.length - 1) return list.filter((_, i) => i !== idx);
    return list.map((ctrl, i) =>
      i === idx
        ? { ...ctrl, children: walk(ctrl.children ?? [], depth + 1) }
        : ctrl,
    );
  };
  return walk(controls, 0);
}

export function appendControl(
  controls: UiPanelControl[],
  parentPath: string | null,
  control: UiPanelControl,
): [UiPanelControl[], string] {
  if (!parentPath) {
    return [[...controls, control], String(controls.length)];
  }
  const parent = getControlAtPath(controls, parentPath);
  if (!parent || parent.kind !== "container") {
    return [[...controls, control], String(controls.length)];
  }
  const childIdx = (parent.children ?? []).length;
  const next = updateControlAtPath(controls, parentPath, {
    children: [...(parent.children ?? []), control],
  });
  return [next, `${parentPath}/${childIdx}`];
}

export function controlNameSlug(raw: string): string {
  return (raw || "")
    .trim()
    .replace(/[^A-Za-z0-9_]+/g, "_")
    .replace(/^_+|_+$/g, "");
}

export function uniqueControlName(
  controls: UiPanelControl[],
  base: string,
  excludePath?: string | null,
): string {
  const used = new Set<string>();
  const walk = (list: UiPanelControl[], prefix: string) => {
    list.forEach((c, i) => {
      const p = prefix ? `${prefix}/${i}` : String(i);
      if (p !== excludePath) used.add(controlNameSlug(c.name) || "control");
      if (c.children) walk(c.children, p);
    });
  };
  walk(controls, "");
  const trimmed = base.trim();
  if (!used.has(controlNameSlug(trimmed) || "control")) return trimmed;
  let n = 2;
  while (used.has(controlNameSlug(`${trimmed}_${n}`) || "control")) n += 1;
  return `${trimmed}_${n}`;
}
