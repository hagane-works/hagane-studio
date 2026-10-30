import type { Idea, IdeaType, IdeaLedger } from "@/types";

export interface IdeaFormCore {
  idea_id: string;
  idea_type: IdeaType;
  name: string;
  picture: string;
  removal_cost: number;
  allowed: string;
  allowed_civil_war: string;
  is_designer: boolean;
  name_en: string;
  name_zh: string;
  desc_en: string;
  desc_zh: string;
  display_name_locs: Record<string, string>;
  description_locs: Record<string, string>;
  available: string;
  visible: string;
  allowed_to_remove: string;
  cancel: string;
  do_effect: string;
  on_add: string;
  on_remove: string;
  cancel_if_invalid: boolean;

  auto_attach_on_start: boolean;
  level: number;
  ledger: IdeaLedger;

  cost: number;
}

export interface IdeaFormAux {
  modifierEntries: [string, string][];
  researchEntries: [string, string][];
  designerTraits: string[];
  targetedModifierJson: string;
  equipmentBonusJson: string;
  ruleJson: string;
}

export function seedIdeaFormCore(idea: Idea): IdeaFormCore {
  return {
    idea_id: idea.idea_id,
    idea_type: idea.idea_type,
    name: idea.name,
    picture: idea.picture,
    removal_cost: idea.removal_cost,
    allowed: idea.allowed,
    allowed_civil_war: idea.allowed_civil_war,
    is_designer: idea.is_designer,
    name_en: idea.name_en,
    name_zh: idea.name_zh,
    desc_en: idea.desc_en,
    desc_zh: idea.desc_zh,
    display_name_locs: idea.display_name_locs ?? {},
    description_locs: idea.description_locs ?? {},
    available: idea.available ?? "",
    visible: idea.visible ?? "",
    allowed_to_remove: idea.allowed_to_remove ?? "",
    cancel: idea.cancel ?? "",
    do_effect: idea.do_effect ?? "",
    on_add: idea.on_add ?? "",
    on_remove: idea.on_remove ?? "",
    cancel_if_invalid: idea.cancel_if_invalid ?? true,

    auto_attach_on_start: idea.auto_attach_on_start ?? true,
    level: idea.level ?? 0,
    ledger: (idea.ledger ?? "") as IdeaLedger,

    cost: idea.cost ?? 0,
  };
}

export function seedIdeaAux(idea: Idea): IdeaFormAux {
  return {
    modifierEntries: Object.entries(idea.modifier || {})
      .filter(([k]) => !k.startsWith("__"))
      .map(([k, v]) => [k, String(v)] as [string, string]),
    researchEntries: Object.entries(idea.research_bonus || {}).map(
      ([k, v]) => [k, String(v)] as [string, string],
    ),
    designerTraits: idea.designer_traits || [],
    targetedModifierJson: JSON.stringify(idea.targeted_modifier ?? [], null, 2),
    equipmentBonusJson: JSON.stringify(idea.equipment_bonus ?? {}, null, 2),
    ruleJson: JSON.stringify(idea.rule ?? {}, null, 2),
  };
}

export function ideaDirtyFingerprint(
  core: IdeaFormCore,
  aux: IdeaFormAux,
): string {
  return JSON.stringify({
    ...core,
    modifier: aux.modifierEntries,
    research_bonus: aux.researchEntries,
    designer_traits: aux.designerTraits,
    targeted_modifier: aux.targetedModifierJson,
    equipment_bonus: aux.equipmentBonusJson,
    rule: aux.ruleJson,
  });
}

export function isIdeaFormDirty(
  core: IdeaFormCore,
  aux: IdeaFormAux,
  idea: Idea,
): boolean {
  return (
    ideaDirtyFingerprint(core, aux) !==
    ideaDirtyFingerprint(seedIdeaFormCore(idea), seedIdeaAux(idea))
  );
}

export function ideaContentFingerprint(idea: Idea): string {
  return ideaDirtyFingerprint(seedIdeaFormCore(idea), seedIdeaAux(idea));
}

export function shouldReseedIdeaForm(
  prevSeededId: number,
  prevSeededFingerprint: string,
  idea: Idea,

  currentFormFingerprint?: string,
): {
  reseed: boolean;
  fingerprint: string;

  reason: "switch" | "content" | "none";

  clobberedDirtyDraft: boolean;
} {
  const fingerprint = ideaContentFingerprint(idea);
  if (prevSeededId !== idea.id) {
    return {
      reseed: true,
      fingerprint,
      reason: "switch",
      clobberedDirtyDraft: false,
    };
  }
  if (fingerprint !== prevSeededFingerprint) {
    return {
      reseed: true,
      fingerprint,
      reason: "content",
      clobberedDirtyDraft:
        currentFormFingerprint !== undefined &&
        currentFormFingerprint !== prevSeededFingerprint,
    };
  }
  return {
    reseed: false,
    fingerprint,
    reason: "none",
    clobberedDirtyDraft: false,
  };
}
