import type { GuiNode } from "@/lib/guiRenderApi";
import { flattenTree } from "@/app/[locale]/gui-renderer/GuiCanvas";
import type { UiPanelControl, UiPanelDerived } from "@/types";

export interface ControlEntry {
  path: string;
  ctrl: UiPanelControl;

  derivedName: string | null;

  absX: number;
  absY: number;
}

export function listControlEntries(
  controls: UiPanelControl[],
  derived: UiPanelDerived | undefined,
): ControlEntry[] {
  const out: ControlEntry[] = [];
  const walk = (
    list: UiPanelControl[],
    prefix: string,
    ox: number,
    oy: number,
  ) => {
    list.forEach((ctrl, i) => {
      const path = prefix ? `${prefix}/${i}` : String(i);
      const derivedName = derived?.controls?.[ctrl.name]?.control_name ?? null;
      const ax = ox + (Number.isFinite(ctrl.x) ? ctrl.x : 0);
      const ay = oy + (Number.isFinite(ctrl.y) ? ctrl.y : 0);
      out.push({ path, ctrl, derivedName, absX: ax, absY: ay });
      if (ctrl.children?.length) walk(ctrl.children, path, ax, ay);
    });
  };
  walk(controls, "", 0, 0);
  return out;
}

export interface NodeMaps {
  idByName: Map<string, string>;

  nameById: Map<string, string>;
}

export function buildNodeMaps(root: GuiNode): NodeMaps {
  const idByName = new Map<string, string>();
  const nameById = new Map<string, string>();
  for (const row of flattenTree(root)) {
    const name = row.node.name;
    if (!name) continue;
    if (!idByName.has(name)) idByName.set(name, row.id);
    nameById.set(row.id, name);
  }
  return { idByName, nameById };
}

export function buildSavedByName(
  saved: UiPanelControl[],
): Map<string, UiPanelControl> {
  const m = new Map<string, UiPanelControl>();
  const walk = (list: UiPanelControl[]) => {
    for (const c of list) {
      if (!m.has(c.name)) m.set(c.name, c);
      if (c.children?.length) walk(c.children);
    }
  };
  walk(saved);
  return m;
}

export interface HandleEntry extends ControlEntry {
  id: string;
}

export function ghostSize(ctrl: UiPanelControl): { w: number; h: number } {
  const dflt =
    ctrl.kind === "container"
      ? { w: 200, h: 120 }
      : ctrl.kind === "progressbar"
        ? { w: 200, h: 22 }
        : ctrl.kind === "text"
          ? { w: 120, h: 20 }
          : ctrl.kind === "button"
            ? { w: 261, h: 34 }
            : { w: 48, h: 48 };
  return { w: ctrl.w ?? dflt.w, h: ctrl.h ?? dflt.h };
}
