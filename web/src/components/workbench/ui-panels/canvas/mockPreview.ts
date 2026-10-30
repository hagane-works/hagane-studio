import type { GuiNode } from "@/lib/guiRenderApi";
import { assignIds } from "@/app/[locale]/gui-renderer/GuiCanvas";
import { evalRuntimeTokens } from "@/components/workbench/ui-panels/mockVars";
import type { ControlEntry, NodeMaps } from "./nodeControlMap";

const EMPTY_OVERRIDES: Record<string, string> = {};

export function buildTextOverrides(
  entries: ControlEntry[],
  maps: NodeMaps | null,
  vars: Record<string, number>,
  enabled: boolean,
): Record<string, string> {
  if (!enabled || !maps) return EMPTY_OVERRIDES;
  const out: Record<string, string> = {};
  for (const e of entries) {
    if (
      (e.ctrl.kind !== "text" && e.ctrl.kind !== "button") ||
      !e.derivedName
    ) {
      continue;
    }
    const id = maps.idByName.get(e.derivedName);
    if (!id) continue;

    out[id] = evalRuntimeTokens(
      e.ctrl.text_locs?.en || e.ctrl.text || "",
      vars,
    );
  }
  return out;
}

function cloneNode(n: GuiNode): GuiNode {
  return {
    ...n,
    background: n.background ? cloneNode(n.background) : null,
    children: (n.children || []).map(cloneNode),
  };
}

export function applyMockFrames(
  root: GuiNode,
  entries: ControlEntry[],
  vars: Record<string, number>,
  enabled: boolean,
): GuiNode {
  if (!enabled) return root;
  const frameByName = new Map<string, number>();
  for (const e of entries) {
    const frameVar = e.ctrl.frame_var?.trim();
    if (!e.derivedName || !frameVar) continue;
    frameByName.set(
      e.derivedName,
      Math.max(1, Math.round(vars[frameVar] ?? 0)),
    );
  }
  if (frameByName.size === 0) return root;
  const clone = cloneNode(root);
  const walk = (n: GuiNode) => {
    if (n.name && frameByName.has(n.name)) n.frame = frameByName.get(n.name)!;
    (n.children || []).forEach(walk);
  };
  walk(clone);

  assignIds(clone);
  return clone;
}
