import type { GuiEquipmentDesign } from "./EquipmentDesignerLayer";

export interface DesignDraft {
  name: string;

  modules: Record<string, string>;
  upgrades: Record<string, number>;
}

export function designDraftOf(design: GuiEquipmentDesign): DesignDraft {
  const modules: Record<string, string> = {};
  for (const slot of design.slots) modules[slot.slot] = slot.module ?? "";
  const upgrades: Record<string, number> = {};
  for (const upgrade of design.upgrades) upgrades[upgrade.name] = upgrade.level;
  return { name: design.name, modules, upgrades };
}

export function draftQuery(draft: DesignDraft | null | undefined): {
  modules: string;
  upgrades: string;
} {
  if (!draft) return { modules: "", upgrades: "" };
  const modules = Object.entries(draft.modules)
    .map(([slot, module]) => `${slot}:${module || "empty"}`)
    .join(";");
  const upgrades = Object.entries(draft.upgrades)
    .map(([name, level]) => `${name}:${level}`)
    .join(";");
  return { modules, upgrades };
}

export function draftChanged(
  draft: DesignDraft | null | undefined,
  design: GuiEquipmentDesign | undefined,
): boolean {
  if (!draft || !design) return false;
  return JSON.stringify(draft) !== JSON.stringify(designDraftOf(design));
}

export type DesignProblem =
  | { key: "nameRequired" | "nameInvalid" | "nameTooLong" }
  | { key: "slotsMissing"; slots: string[] };

export function designProblem(
  draft: DesignDraft,
  design: GuiEquipmentDesign,
): DesignProblem | null {
  const name = draft.name.trim();
  if (!name) return { key: "nameRequired" };
  if (/["\\\r\n]/.test(name)) return { key: "nameInvalid" };
  if (name.length > 100) return { key: "nameTooLong" };
  const missing = design.slots.filter(
    (slot) => slot.required && !draft.modules[slot.slot],
  );
  if (missing.length)
    return {
      key: "slotsMissing",
      slots: missing.map((slot) => slot.slot_label || slot.slot),
    };
  return null;
}

export function nameTaken(
  draft: DesignDraft,
  design: GuiEquipmentDesign,
  mode: "new" | "update",
): boolean {
  const name = draft.name.trim();
  if (mode === "update" && name === design.name) return false;
  return (design.designs ?? []).some((other) => other.name === name);
}

export function designSaveBody(
  tag: string,
  design: GuiEquipmentDesign,
  draft: DesignDraft,
  mode: "new" | "update" | "delete",
): Record<string, unknown> {
  if (mode === "delete") return { tag, original: design.name, delete: true };
  const upgrades: Record<string, number> = {};
  for (const [name, level] of Object.entries(draft.upgrades))
    if (level > 0) upgrades[name] = level;
  return {
    tag,
    original: mode === "update" ? design.name : "",
    name: draft.name.trim(),
    type: design.type,

    modules: design.slots.map((slot) => ({
      slot: slot.slot,
      module: draft.modules[slot.slot] ?? "",
    })),
    upgrades,
  };
}
