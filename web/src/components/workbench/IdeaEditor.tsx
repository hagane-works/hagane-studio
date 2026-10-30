"use client";

import {
  useState,
  useEffect,
  useCallback,
  useId,
  useMemo,
  useRef,
} from "react";
import { useTranslations } from "use-intl";
import { useLocaleStore } from "@/lib/store/localeStore";
import { localizedOptionLabel } from "@/lib/i18n/optionLabel";
import { Input } from "@/components/ui/input";
import { Select } from "@/components/ui/select";
import { Label } from "@/components/ui/label";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Separator } from "@/components/ui/separator";
import { Textarea } from "@/components/ui/textarea";
import { InfoTooltip, TooltipProvider } from "@/components/ui/tooltip";
import {
  CONDITION_TEMPLATES,
  EFFECT_TEMPLATES,
  ScriptBlockEditor,
  SectionHeader,
  extractSaveErrorMessage,
  validateRequiredId,
} from "@/components/shared/editors";
import {
  ImageUpload,
  type ImageUploadResult,
} from "@/shared/upload/ImageUpload";
import { ideasApi } from "@/lib/api";
import { useBaseModDepKeys, gfxUrl } from "@/lib/api/basemod";
import { IDEA_PICTURE_SIZE } from "@/shared/upload/sizes";
import {
  SourceEditorDialog,
  type SourceEditorChoice,
} from "@/shared/dialogs/SourceEditorDialog";
import { DisplayNameLocsTabs } from "@/components/shared/editors/DisplayNameLocsTabs";
import { DescriptionLocsTabs } from "@/components/shared/editors/DescriptionLocsTabs";
import { DEFAULT_EXPORT_LANGUAGES } from "@/components/shared/editors/displayNameLocsHelpers";
import { useProjectStore } from "@/lib/store/projectStore";
import { useBaseModVocabulary } from "@/lib/hooks/useBaseModVocabulary";
import { useRegisterDirty } from "@/lib/hooks/useUnsavedGuard";
import { useCwtoolRules } from "@/shared/editors/cwtoolStore";
import {
  isUnknownModifierKey,
  knownModifierOptions,
  modifierEntriesToRecord,
  modifierValueFeedback,
  updateModifierEntry,
} from "@/components/shared/editors/ideaModifierAssistance";
import {
  seedIdeaFormCore,
  seedIdeaAux,
  isIdeaFormDirty,
  ideaContentFingerprint,
  ideaDirtyFingerprint,
  shouldReseedIdeaForm,
  type IdeaFormCore,
} from "@/components/workbench/ideaFormState";
import type { Idea, IdeaType, IdeaLedger } from "@/types";
import { IDEA_TYPE_OPTIONS, DESIGNER_TYPES } from "@/types";
import {
  CUSTOM_VOCAB_SENTINEL,
  baseModOnlyOptions,
  resolveVocabularySelection,
} from "@/components/shared/editors/vocabularyMode";
import { useApplyReseedNotice } from "@/components/shared/editors/useApplyReseedNotice";
import {
  useVanillaEquipmentArchetypes,
  isEquipmentBonusReservedKey,
  detectInstanceArchetype,
  COMMON_ARCHETYPES,
} from "@/lib/vanillaEquipmentArchetypes";
import {
  useBaseModIdCollision,
  BaseModIdCollisionNotice,
} from "./baseModIdCollision";

interface IdeaEditorProps {
  idea: Idea;
  projectId: number;
  onSave: (data: Partial<Idea>) => Promise<void>;
  onPictureUploaded?: (result: ImageUploadResult) => void;
  onDraftChange?: (draft: Idea) => void;

  allIdeas?: Idea[];

  onForked?: (forked: Idea) => void;

  embedded?: boolean;
}

const LEDGER_OPTIONS: { value: IdeaLedger; label: string }[] = [
  { value: "", label: "(generic / political)" },
  { value: "civilian", label: "civilian" },
  { value: "army", label: "army" },
  { value: "navy", label: "navy" },
  { value: "air", label: "air" },
];

const MODIFIER_KEY_EXAMPLE = "consumer_goods_factor";

export default function IdeaEditor({
  idea,
  projectId,
  onSave,
  onPictureUploaded,
  onDraftChange,
  allIdeas,
  onForked,
  embedded = false,
}: IdeaEditorProps) {
  const t = useTranslations("ideaEditor");
  const tIdeas = useTranslations("ideas");

  const tEnums = useTranslations("enums");

  const tIdiom = useTranslations("idiomWarning");
  const baseModVocab = useBaseModVocabulary(projectId);

  const tCommon = useTranslations("common");
  const locale = useLocaleStore((s) => s.locale);
  const cwtoolRules = useCwtoolRules();
  const knownModifiers = useMemo(
    () => knownModifierOptions(cwtoolRules?.known_modifiers),
    [cwtoolRules?.known_modifiers],
  );
  const knownModifierSet = useMemo(
    () => (cwtoolRules ? new Set(knownModifiers) : null),
    [cwtoolRules, knownModifiers],
  );
  const modifierDatalistId = useId();

  const exportLanguages = useProjectStore(
    (s) => s.currentProject?.export_languages ?? DEFAULT_EXPORT_LANGUAGES,
  );

  const [form, setForm] = useState<IdeaFormCore>(() => seedIdeaFormCore(idea));

  const baseModId = useBaseModIdCollision("idea", form.idea_id || "");

  const verifyBaseModId = baseModId.verify;

  const [ideaTypeCustomMode, setIdeaTypeCustomMode] = useState(false);
  const vanillaIdeaTypes = useMemo(
    () => IDEA_TYPE_OPTIONS.map((o) => o.value as string),
    [],
  );
  const baseModIdeaTypes = useMemo(
    () => baseModOnlyOptions(vanillaIdeaTypes, baseModVocab.idea_types),
    [vanillaIdeaTypes, baseModVocab.idea_types],
  );
  const ideaTypeSel = resolveVocabularySelection(
    form.idea_type,
    vanillaIdeaTypes,
    baseModVocab.idea_types,
    ideaTypeCustomMode,
  );

  const baseModDepKeys = useBaseModDepKeys();
  const baseModPictureFallbacks = useMemo(
    () =>
      form.picture && baseModDepKeys.length
        ? baseModDepKeys.map((dk) =>
            gfxUrl(
              projectId,
              dk,
              form.picture.startsWith("GFX_")
                ? form.picture
                : `GFX_idea_${form.picture}`,
            ),
          )
        : [],
    [projectId, baseModDepKeys, form.picture],
  );

  const [modifierEntries, setModifierEntries] = useState<[string, string][]>(
    () => seedIdeaAux(idea).modifierEntries,
  );

  const [researchEntries, setResearchEntries] = useState<[string, string][]>(
    () => seedIdeaAux(idea).researchEntries,
  );

  const [designerTraits, setDesignerTraits] = useState<string[]>(
    () => seedIdeaAux(idea).designerTraits,
  );

  const [targetedModifierJson, setTargetedModifierJson] = useState(
    () => seedIdeaAux(idea).targetedModifierJson,
  );
  const [equipmentBonusJson, setEquipmentBonusJson] = useState(
    () => seedIdeaAux(idea).equipmentBonusJson,
  );
  const [ruleJson, setRuleJson] = useState(() => seedIdeaAux(idea).ruleJson);
  const [jsonError, setJsonError] = useState<string>("");

  const archetypes = useVanillaEquipmentArchetypes();
  const equipmentBonusWarning = useMemo<{
    type: "instance" | "unknown";
    key: string;
    archetype?: string;
  } | null>(() => {
    const text = equipmentBonusJson.trim();
    if (!text || text === "{}" || archetypes.size === 0) return null;
    let parsed: unknown;
    try {
      parsed = JSON.parse(text);
    } catch {
      return null;
    }
    if (!parsed || typeof parsed !== "object" || Array.isArray(parsed))
      return null;
    for (const key of Object.keys(parsed as Record<string, unknown>)) {
      if (isEquipmentBonusReservedKey(key)) continue;
      if (archetypes.has(key)) continue;
      const detected = detectInstanceArchetype(key, archetypes);
      if (detected) {
        return { type: "instance", key, archetype: detected };
      }
      return { type: "unknown", key };
    }
    return null;
  }, [equipmentBonusJson, archetypes]);

  const insertArchetypeChip = useCallback((archetype: string) => {
    setEquipmentBonusJson((prev) => {
      const text = prev.trim();
      let obj: Record<string, unknown> = {};
      if (text && text !== "{}") {
        try {
          const parsed = JSON.parse(text);
          if (parsed && typeof parsed === "object" && !Array.isArray(parsed)) {
            obj = parsed as Record<string, unknown>;
          }
        } catch {
          return prev;
        }
      }
      if (archetype in obj) return prev;
      obj[archetype] = { soft_attack: 0.1 };
      return JSON.stringify(obj, null, 2);
    });
  }, []);

  const [saving, setSaving] = useState(false);
  const [saveStatus, setSaveStatus] = useState<"idle" | "success" | "error">(
    "idle",
  );

  const [saveError, setSaveError] = useState<string | null>(null);
  const [idError, setIdError] = useState<string | null>(null);

  const [showSourceDialog, setShowSourceDialog] = useState(false);
  const [pendingPayload, setPendingPayload] = useState<Partial<Idea> | null>(
    null,
  );

  const resetFormFromIdea = useCallback(() => {
    if (!idea) return;

    const aux = seedIdeaAux(idea);
    setForm(seedIdeaFormCore(idea));
    setModifierEntries(aux.modifierEntries);
    setResearchEntries(aux.researchEntries);
    setDesignerTraits(aux.designerTraits);
    setTargetedModifierJson(aux.targetedModifierJson);
    setEquipmentBonusJson(aux.equipmentBonusJson);
    setRuleJson(aux.ruleJson);
    setJsonError("");
  }, [idea]);

  const seededRef = useRef({
    id: idea.id,
    fingerprint: ideaContentFingerprint(idea),
  });

  const notifyReseed = useApplyReseedNotice();
  useEffect(() => {
    const { reseed, fingerprint, reason, clobberedDirtyDraft } =
      shouldReseedIdeaForm(
        seededRef.current.id,
        seededRef.current.fingerprint,
        idea,

        ideaDirtyFingerprint(form, {
          modifierEntries,
          researchEntries,
          designerTraits,
          equipmentBonusJson,
          targetedModifierJson,
          ruleJson,
        }),
      );
    seededRef.current = { id: idea.id, fingerprint };
    if (reseed) resetFormFromIdea();
    notifyReseed({ reason, clobberedDirtyDraft });
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [resetFormFromIdea, idea, notifyReseed]);

  useEffect(() => {
    const modifier = modifierEntriesToRecord(modifierEntries);

    const research_bonus: Record<string, number> = {};
    for (const [key, value] of researchEntries) {
      if (!key.trim()) continue;
      research_bonus[key.trim()] = parseFloat(value) || 0;
    }

    const parseOrFallback = <T,>(text: string, fallback: T): T => {
      const trimmed = text.trim();
      if (!trimmed) return fallback;
      try {
        return JSON.parse(trimmed) as T;
      } catch {
        return fallback;
      }
    };

    onDraftChange?.({
      ...idea,
      ...form,
      idea_id: (form.idea_id || "").trim(),
      modifier,
      research_bonus,
      designer_traits: designerTraits,
      is_designer: DESIGNER_TYPES.includes(form.idea_type),
      targeted_modifier: parseOrFallback<Idea["targeted_modifier"]>(
        targetedModifierJson,
        idea.targeted_modifier ?? [],
      ),
      equipment_bonus: parseOrFallback<Idea["equipment_bonus"]>(
        equipmentBonusJson,
        idea.equipment_bonus ?? {},
      ),
      rule: parseOrFallback<Idea["rule"]>(ruleJson, idea.rule ?? {}),
    });
  }, [
    idea,
    form,
    modifierEntries,
    researchEntries,
    designerTraits,
    targetedModifierJson,
    equipmentBonusJson,
    ruleJson,
    onDraftChange,
  ]);

  const isDesigner = DESIGNER_TYPES.includes(form.idea_type);

  const handleSave = useCallback(async () => {
    const idValidation = validateRequiredId(form.idea_id, "Idea ID");
    if (idValidation) {
      setIdError(idValidation);
      setSaveError(idValidation);
      setSaveStatus("error");
      return;
    }
    setIdError(null);

    if (await verifyBaseModId((form.idea_id || "").trim())) return;
    setSaving(true);
    setSaveStatus("idle");
    setSaveError(null);
    setJsonError("");

    const modifier = modifierEntriesToRecord(modifierEntries);

    if (idea.modifier) {
      for (const [k, v] of Object.entries(idea.modifier)) {
        if (k.startsWith("__")) modifier[k] = v;
      }
    }

    const research_bonus: Record<string, number> = {};
    for (const [k, v] of researchEntries) {
      if (k.trim()) research_bonus[k.trim()] = parseFloat(v) || 0;
    }

    let targeted_modifier: Idea["targeted_modifier"];
    let equipment_bonus: Idea["equipment_bonus"];
    let rule: Idea["rule"];
    try {
      targeted_modifier = targetedModifierJson.trim()
        ? JSON.parse(targetedModifierJson)
        : [];
      equipment_bonus = equipmentBonusJson.trim()
        ? JSON.parse(equipmentBonusJson)
        : {};
      rule = ruleJson.trim() ? JSON.parse(ruleJson) : {};
    } catch (err) {
      setJsonError(
        `JSON parse error in advanced fields: ${
          err instanceof Error ? err.message : String(err)
        }`,
      );
      setSaving(false);
      setSaveStatus("error");
      return;
    }

    const payload: Partial<Idea> = {
      ...form,
      idea_id: (form.idea_id || "").trim(),
      modifier,
      research_bonus,
      designer_traits: designerTraits,
      is_designer: isDesigner,
      targeted_modifier,
      equipment_bonus,
      rule,
    };

    if (idea.source === "imported") {
      setPendingPayload(payload);
      setShowSourceDialog(true);
      setSaving(false);
      return;
    }

    try {
      await onSave(payload);
      setSaveStatus("success");
      setTimeout(() => setSaveStatus("idle"), 2000);
    } catch (err) {
      setSaveError(extractSaveErrorMessage(err));
      setSaveStatus("error");
    } finally {
      setSaving(false);
    }
  }, [
    form,
    modifierEntries,
    researchEntries,
    designerTraits,
    isDesigner,
    idea,
    onSave,
    targetedModifierJson,
    equipmentBonusJson,
    ruleJson,
    verifyBaseModId,
  ]);

  const handleSourceDialogChoice = useCallback(
    async (choice: SourceEditorChoice, newId?: string) => {
      setShowSourceDialog(false);
      if (choice === "keep_imported") {
        resetFormFromIdea();
        setPendingPayload(null);
        return;
      }
      if (choice === "override_as_creator") {
        if (!pendingPayload) return;
        setSaving(true);
        setSaveError(null);
        try {
          await onSave(pendingPayload);
          setSaveStatus("success");
          setTimeout(() => setSaveStatus("idle"), 2000);
        } catch (err) {
          setSaveError(extractSaveErrorMessage(err));
          setSaveStatus("error");
        } finally {
          setSaving(false);
          setPendingPayload(null);
        }
        return;
      }
      if (choice === "create_new_copy" && newId) {
        setSaving(true);
        setSaveError(null);
        try {
          const forked = await ideasApi.fork(projectId, idea.id, newId);
          setSaveStatus("success");
          setTimeout(() => setSaveStatus("idle"), 2000);
          onForked?.(forked);
        } catch (err) {
          setSaveError(extractSaveErrorMessage(err, "Fork failed"));
          setSaveStatus("error");
        } finally {
          setSaving(false);
          setPendingPayload(null);
        }
      }
    },
    [pendingPayload, resetFormFromIdea, onSave, onForked, projectId, idea.id],
  );

  const handleSourceDialogCancel = useCallback(() => {
    setShowSourceDialog(false);
    setPendingPayload(null);
  }, []);

  const addModifier = () => setModifierEntries((e) => [...e, ["", ""]]);
  const removeModifier = (idx: number) =>
    setModifierEntries((e) => e.filter((_, i) => i !== idx));
  const updateModifier = (idx: number, pos: 0 | 1, val: string) =>
    setModifierEntries((entries) =>
      updateModifierEntry(entries, idx, pos, val),
    );

  const addResearch = () => setResearchEntries((e) => [...e, ["", ""]]);
  const removeResearch = (idx: number) =>
    setResearchEntries((e) => e.filter((_, i) => i !== idx));
  const updateResearch = (idx: number, pos: 0 | 1, val: string) =>
    setResearchEntries((e) =>
      e.map((entry, i) =>
        i === idx ? (pos === 0 ? [val, entry[1]] : [entry[0], val]) : entry,
      ),
    );

  const [newTrait, setNewTrait] = useState("");
  const addDesignerTrait = () => {
    if (newTrait.trim() && !designerTraits.includes(newTrait.trim())) {
      setDesignerTraits((t) => [...t, newTrait.trim()]);
      setNewTrait("");
    }
  };

  const isUnsaved = isIdeaFormDirty(
    form,
    {
      modifierEntries,
      researchEntries,
      designerTraits,
      targetedModifierJson,
      equipmentBonusJson,
      ruleJson,
    },
    idea,
  );

  useRegisterDirty("ideas", isUnsaved);

  return (
    <TooltipProvider>
      <div
        className={`space-y-4 p-4 max-w-2xl mx-auto ${embedded ? "" : "overflow-y-auto max-h-[calc(100vh-120px)]"}`}
      >
        <div
          className={`
          sticky top-0 z-10 -mx-4 -mt-4 px-4 py-1.5 text-xs font-medium flex items-center justify-between shrink-0
          transition-[opacity,background-color,border-color] duration-300
          ${
            saving
              ? "bg-muted/80 text-muted-foreground border-b border-border"
              : saveStatus === "success"
                ? "bg-success/10 text-success border-b border-success/20"
                : saveStatus === "error"
                  ? "bg-destructive/10 text-destructive border-b border-destructive/20"
                  : isUnsaved
                    ? "bg-warning/10 text-warning border-b border-warning/20"
                    : "bg-muted/40 text-muted-foreground border-b border-border"
          }
        `}
        >
          <span className="font-mono truncate">
            {saving
              ? t("saving")
              : saveStatus === "success"
                ? t("saveSucceeded")
                : saveStatus === "error"
                  ? (saveError ?? t("saveFailed"))
                  : isUnsaved
                    ? t("unsavedChanges")
                    : idea.idea_id || t("newIdeaFallback")}
          </span>
          <Button
            onClick={handleSave}
            disabled={saving || !!baseModId.collision}
            size="sm"
            className="h-6 px-2 text-xs shrink-0"
          >
            {t("saveButton")}
          </Button>
        </div>

        <Separator />

        {form.idea_type === "country" && (
          <div className="rounded-md border border-border bg-muted/30 p-3 text-xs space-y-1.5">
            <p className="font-semibold text-foreground">{t("introTitle")}</p>
            <ol className="list-decimal list-inside space-y-0.5 text-muted-foreground">
              <li>
                {t.rich("introLi1", {
                  b: (c) => <strong className="text-foreground">{c}</strong>,
                  gfx: () => <>GFX_idea_{`{TAG}_{idea_id}`}</>,
                })}
              </li>
              <li>
                {t.rich("introLi2", {
                  b: (c) => <strong className="text-foreground">{c}</strong>,
                })}
              </li>
              <li>
                {t.rich("introLi3", {
                  b: (c) => <strong className="text-foreground">{c}</strong>,
                  loc: () => (
                    <>
                      localisation/{`{lang}`}/ideas_l_{`{lang}`}.yml
                    </>
                  ),
                })}
              </li>
            </ol>
            <p className="text-muted-foreground italic">{t("introOutcome")}</p>
          </div>
        )}

        <div className="grid grid-cols-2 gap-3">
          <div className="space-y-1">
            <div className="flex items-center gap-1">
              <Label>{t("ideaIdLabel")}</Label>
              <InfoTooltip
                title={t("ideaIdTitle")}
                content={t.raw("ideaIdContent")}
              />
            </div>
            <Input
              value={form.idea_id}
              onChange={(e) => {
                if (idError) setIdError(null);
                setForm((f) => ({ ...f, idea_id: e.target.value }));
              }}
              aria-invalid={idError || baseModId.collision ? true : undefined}
              className={
                idError || baseModId.collision
                  ? "border-destructive"
                  : undefined
              }
              placeholder="e.g. great_depression"
            />
            {idError && (
              <p className="mt-1 text-xs text-destructive">{idError}</p>
            )}
            <BaseModIdCollisionNotice
              entityType="idea"
              collision={baseModId.collision}
            />

            {(() => {
              const trimmedId = (form.idea_id || "").trim();
              if (!trimmedId || !allIdeas) return null;
              const colliding = allIdeas.find(
                (i) => i.id !== idea.id && i.idea_id === trimmedId,
              );
              if (!colliding) return null;
              return (
                <div
                  className="mt-1 inline-flex items-center gap-1 rounded-md border border-accent bg-accent/15 px-2 py-1 text-xs text-foreground"
                  title={t("idCollisionTitle", {
                    name: colliding.name_en || colliding.idea_id,
                  })}
                >
                  <span className="font-medium">{t("idCollisionLabel")}</span>
                  <span className="font-mono">
                    {colliding.name_en || colliding.idea_id}
                  </span>
                </div>
              );
            })()}
          </div>
          <div className="space-y-1">
            <div className="flex items-center gap-1">
              <Label>{t("typeLabel")}</Label>
              <InfoTooltip
                title={t("typeTitle")}
                content={t.raw("typeContent")}
              />
            </div>

            <Select
              className="rounded px-2"
              value={form.idea_type}
              trailingChip={
                ideaTypeSel.isCustom
                  ? tCommon("vocabulary.customChip")
                  : undefined
              }
              onChange={(e) => {
                const v = e.target.value;
                if (v === CUSTOM_VOCAB_SENTINEL) {
                  setIdeaTypeCustomMode(true);
                  return;
                }
                setIdeaTypeCustomMode(false);
                setForm((f) => ({ ...f, idea_type: v as IdeaType }));
              }}
            >
              {baseModIdeaTypes.length > 0 ? (
                <optgroup label={tCommon("vocabulary.vanillaGroup")}>
                  {IDEA_TYPE_OPTIONS.map((opt) => (
                    <option key={opt.value} value={opt.value}>
                      {tEnums(opt.label_i18n_key)}
                    </option>
                  ))}
                </optgroup>
              ) : (
                IDEA_TYPE_OPTIONS.map((opt) => (
                  <option key={opt.value} value={opt.value}>
                    {tEnums(opt.label_i18n_key)}
                  </option>
                ))
              )}
              {baseModIdeaTypes.length > 0 && (
                <optgroup label={tCommon("vocabulary.baseModGroup")}>
                  {baseModIdeaTypes.map((tp) => (
                    <option key={tp} value={tp}>
                      {tp}
                    </option>
                  ))}
                </optgroup>
              )}
              <option value={CUSTOM_VOCAB_SENTINEL}>
                {tCommon("vocabulary.customOption")}
              </option>
            </Select>
            {ideaTypeSel.freeText && (
              <Input
                value={form.idea_type || ""}
                onChange={(e) =>
                  setForm((f) => ({
                    ...f,
                    idea_type: e.target.value as IdeaType,
                  }))
                }
                placeholder="kr_political_faction"
                className="mt-1 font-mono text-xs"
              />
            )}

            {ideaTypeSel.showUnknownNote && (
              <p className="mt-1 text-xs text-muted-foreground">
                {tIdiom("ideaTypeNote", {
                  value: form.idea_type,
                  list: baseModVocab.idea_types.join(", "),
                })}
              </p>
            )}
          </div>
        </div>

        <div className="space-y-1">
          <Label>{t("displayNameLabel")}</Label>
          <DisplayNameLocsTabs
            entityKey={`idea:${idea.id}:display-name`}
            value={form.display_name_locs}
            exportLanguages={exportLanguages}
            legacyFallback={{ en: form.name_en, zh: form.name_zh }}
            onChange={(next) =>
              setForm((f) => ({ ...f, display_name_locs: next }))
            }
            ariaLabel="Idea display name"
          />
        </div>

        <div className="space-y-1">
          <div className="flex items-center gap-1">
            <Label>{t("pictureLabel")}</Label>
            <InfoTooltip
              title={t("pictureTitle")}
              content={t.raw("pictureContent")}
            />
          </div>
          <Input
            value={form.picture}
            onChange={(e) =>
              setForm((f) => ({ ...f, picture: e.target.value }))
            }
            placeholder="e.g. usa_great_depression_3 (or upload below)"
          />
          <p className="text-xs text-muted-foreground mt-1">
            {t("pictureShipNote")}
          </p>
        </div>

        <div className="grid grid-cols-2 gap-3">
          <div className="space-y-1">
            <div className="flex items-center gap-1">
              <Label>{t("removalCostLabel")}</Label>
              <InfoTooltip
                title={t("removalCostTitle")}
                content={t.raw("removalCostContent")}
              />
            </div>
            <Input
              type="number"
              value={form.removal_cost}
              onChange={(e) => {
                const n = parseInt(e.target.value, 10);
                setForm((f) => ({
                  ...f,
                  removal_cost: Number.isNaN(n) ? -1 : n,
                }));
              }}
            />
            <p className="text-xs text-muted-foreground mt-1">
              -1 = cannot remove
            </p>
          </div>

          <div className="space-y-1">
            <div className="flex items-center gap-1">
              <Label>{t("costLabel")}</Label>
              <InfoTooltip
                title={t("costTitle")}
                content={t.raw("costContent")}
              />
            </div>
            <Input
              type="number"
              min={0}
              value={form.cost}
              onChange={(e) => {
                const n = parseInt(e.target.value, 10);
                setForm((f) => ({ ...f, cost: Number.isNaN(n) ? 0 : n }));
              }}
            />
            <p className="text-xs text-muted-foreground mt-1">
              0 = unset (omitted)
            </p>
          </div>
        </div>

        <ImageUpload
          label={t("nationalSpiritPictureLabel")}
          targetSize={IDEA_PICTURE_SIZE}
          previewUrl={
            form.picture ? ideasApi.pictureUrl(projectId, idea.id) : null
          }
          baseModFallbackUrls={baseModPictureFallbacks}
          gfxName={form.picture || undefined}
          hint="PNG / JPG / DDS · 95 × 65 · auto-converted to DDS at export · GFX name auto-generated"
          onUpload={async (file, crop) =>
            ideasApi.uploadPicture(projectId, idea.id, file, crop)
          }
          onUploaded={(result) => {
            if (result.picture) {
              const nextPicture = result.picture as string;
              setForm((f) => ({ ...f, picture: nextPicture }));

              seededRef.current = {
                ...seededRef.current,
                fingerprint: ideaContentFingerprint({
                  ...idea,
                  picture: nextPicture,
                }),
              };
            }
            onPictureUploaded?.(result);
          }}
        />

        <Separator />

        <div>
          <SectionHeader
            title={t("modifierTitle")}
            tooltip={t("modifierContent")}
            className="mb-2"
          >
            <Button size="sm" variant="outline" onClick={addModifier}>
              + {t("addEntryButton")}
            </Button>
          </SectionHeader>
          {modifierEntries.length === 0 && (
            <p className="text-sm text-muted-foreground">{t("noModifiers")}</p>
          )}
          <p className="mb-2 text-xs text-muted-foreground">
            {tIdeas("modifierSuggestionHint")}
          </p>
          {modifierEntries.map(([key, val], idx) => {
            const unknownKey = isUnknownModifierKey(key, knownModifierSet);
            const valueFeedback = modifierValueFeedback(val);
            return (
              <div key={idx} className="mb-2 flex items-start gap-2">
                <div className="min-w-0 flex-1 space-y-1">
                  <Input
                    list={modifierDatalistId}
                    aria-label={tIdeas("modifierKeyAria", { index: idx + 1 })}
                    value={key}
                    onChange={(event) =>
                      updateModifier(idx, 0, event.target.value)
                    }
                    placeholder={`${tIdeas("modifierKeyPlaceholder")} (e.g. ${MODIFIER_KEY_EXAMPLE})`}
                  />
                  {unknownKey && (
                    <p className="text-xs text-warning">
                      {tIdeas("modifierUnknownKeyHint")}
                    </p>
                  )}
                </div>
                <div className="w-32 space-y-1">
                  <Input
                    inputMode="decimal"
                    aria-label={tIdeas("modifierValueAria", { index: idx + 1 })}
                    value={val}
                    onChange={(event) =>
                      updateModifier(idx, 1, event.target.value)
                    }
                    placeholder={tIdeas("modifierValuePlaceholder")}
                  />
                  {valueFeedback.kind === "text" && (
                    <p className="text-xs text-muted-foreground">
                      {tIdeas("modifierTextValueHint")}
                    </p>
                  )}
                  {valueFeedback.kind === "truncated" && (
                    <p className="text-xs text-warning">
                      {tIdeas("modifierTruncatedValueHint", {
                        value: valueFeedback.savedValue,
                      })}
                    </p>
                  )}
                </div>
                <Button
                  size="sm"
                  variant="ghost"
                  onClick={() => removeModifier(idx)}
                  className="text-destructive px-2"
                  aria-label={tIdeas("modifierRemoveAria", { index: idx + 1 })}
                >
                  X
                </Button>
              </div>
            );
          })}
          {knownModifiers.length > 0 && (
            <datalist id={modifierDatalistId}>
              {knownModifiers.map((modifier) => (
                <option key={modifier} value={modifier} />
              ))}
            </datalist>
          )}
        </div>

        <Separator />

        {!isDesigner && (
          <>
            <ScriptBlockEditor
              label={t("allowedLabel")}
              labelTooltip={t.raw("allowedContent")}
              value={form.allowed}
              onChange={(allowed) => setForm((f) => ({ ...f, allowed }))}
              placeholder="always = no"
              templates={CONDITION_TEMPLATES}
              minHeight="min-h-20"
            />
            <ScriptBlockEditor
              label={t("allowedCivilWarLabel")}
              labelTooltip={t.raw("allowedCivilWarContent")}
              value={form.allowed_civil_war}
              onChange={(allowed_civil_war) =>
                setForm((f) => ({ ...f, allowed_civil_war }))
              }
              placeholder="always = yes"
              templates={CONDITION_TEMPLATES}
            />
          </>
        )}

        {isDesigner && (
          <>
            <Separator />
            <h4 className="font-medium">{t("designCompanyFields")}</h4>

            <ScriptBlockEditor
              label={t("allowedLabel")}
              labelTooltip={t.raw("allowedContent")}
              value={form.allowed}
              onChange={(allowed) => setForm((f) => ({ ...f, allowed }))}
              placeholder="original_tag = USA"
              templates={CONDITION_TEMPLATES}
            />

            <div className="space-y-1">
              <div className="flex items-center justify-between mb-2">
                <Label>{t("researchBonusLabel")}</Label>
                <Button size="sm" variant="outline" onClick={addResearch}>
                  + {t("addEntryButton")}
                </Button>
              </div>
              {researchEntries.map(([cat, bonus], idx) => (
                <div key={idx} className="flex gap-2 mb-1.5">
                  <Input
                    className="flex-1"
                    value={cat}
                    onChange={(e) => updateResearch(idx, 0, e.target.value)}
                    placeholder="category (e.g. armor)"
                  />
                  <Input
                    className="w-28"
                    value={bonus}
                    onChange={(e) => updateResearch(idx, 1, e.target.value)}
                    placeholder="bonus (e.g. 0.15)"
                  />
                  <Button
                    size="sm"
                    variant="ghost"
                    onClick={() => removeResearch(idx)}
                    className="text-destructive px-2"
                  >
                    X
                  </Button>
                </div>
              ))}
            </div>

            <div className="space-y-1">
              <Label>{t("designerTraitsLabel")}</Label>
              <div className="flex gap-2 mb-2">
                <Input
                  value={newTrait}
                  onChange={(e) => setNewTrait(e.target.value)}
                  placeholder="e.g. fast_tank_manufacturer"
                  onKeyDown={(e) => e.key === "Enter" && addDesignerTrait()}
                />
                <Button size="sm" onClick={addDesignerTrait}>
                  {t("addTraitButton")}
                </Button>
              </div>
              <div className="flex flex-wrap gap-1">
                {designerTraits.map((t) => (
                  <Badge
                    key={t}
                    variant="secondary"
                    className="cursor-pointer"
                    onClick={() =>
                      setDesignerTraits((traits) =>
                        traits.filter((x) => x !== t),
                      )
                    }
                  >
                    {t} x
                  </Badge>
                ))}
              </div>
            </div>
          </>
        )}

        <Separator />

        <div className="space-y-1">
          <Label>{t("descriptionLabel")}</Label>
          <DescriptionLocsTabs
            entityKey={`idea:${idea.id}:description`}
            value={form.description_locs || {}}
            exportLanguages={exportLanguages}
            legacyFallback={{ en: form.desc_en, zh: form.desc_zh }}
            onChange={(next) =>
              setForm((f) => ({ ...f, description_locs: next }))
            }
            ariaLabel="Idea description"
            rows={2}
          />
        </div>

        <details className="group rounded-md border border-border bg-card/50">
          <summary className="cursor-pointer select-none px-3 py-2 text-sm font-semibold text-muted-foreground uppercase tracking-wide hover:bg-card/80 rounded-md">
            {t("advancedSummary")}
            <span className="ml-2 text-xs font-normal normal-case text-muted-foreground">
              {t("advancedHint")}
            </span>
          </summary>
          <div className="space-y-3 p-3 border-t border-border">
            <ScriptBlockEditor
              label="available (trigger block)"
              labelTooltip={t.raw("availableContent")}
              value={form.available}
              onChange={(available) => setForm((f) => ({ ...f, available }))}
              placeholder="has_war = no"
              templates={CONDITION_TEMPLATES}
            />
            <ScriptBlockEditor
              label="visible (trigger block)"
              labelTooltip={t.raw("visibleContent")}
              value={form.visible}
              onChange={(visible) => setForm((f) => ({ ...f, visible }))}
              placeholder="has_government = democratic"
              templates={CONDITION_TEMPLATES}
            />
            <ScriptBlockEditor
              label="cancel (trigger block)"
              labelTooltip={t.raw("cancelContent")}
              value={form.cancel}
              onChange={(cancel) => setForm((f) => ({ ...f, cancel }))}
              placeholder="has_completed_focus = recovery_act"
              templates={CONDITION_TEMPLATES}
            />
            <ScriptBlockEditor
              label="allowed_to_remove (trigger block)"
              labelTooltip={t.raw("allowedToRemoveContent")}
              value={form.allowed_to_remove}
              onChange={(allowed_to_remove) =>
                setForm((f) => ({ ...f, allowed_to_remove }))
              }
              placeholder="has_political_power > 50"
              templates={CONDITION_TEMPLATES}
            />
            <ScriptBlockEditor
              label="do_effect (effect block)"
              labelTooltip={t.raw("doEffectContent")}
              value={form.do_effect}
              onChange={(do_effect) => setForm((f) => ({ ...f, do_effect }))}
              placeholder="add_political_power = 1"
              templates={EFFECT_TEMPLATES}
            />
            <ScriptBlockEditor
              label="on_add (one-shot effect)"
              labelTooltip={t.raw("onAddContent")}
              value={form.on_add}
              onChange={(on_add) => setForm((f) => ({ ...f, on_add }))}
              placeholder="set_country_flag = idea_X_added"
              templates={EFFECT_TEMPLATES}
            />
            <ScriptBlockEditor
              label="on_remove (one-shot effect)"
              labelTooltip={t.raw("onRemoveContent")}
              value={form.on_remove}
              onChange={(on_remove) => setForm((f) => ({ ...f, on_remove }))}
              placeholder="clr_country_flag = idea_X_added"
              templates={EFFECT_TEMPLATES}
            />

            {form.idea_type === "country" && (
              <div className="flex items-center gap-2">
                <input
                  id="auto_attach_on_start"
                  type="checkbox"
                  checked={form.auto_attach_on_start}
                  onChange={(e) =>
                    setForm((f) => ({
                      ...f,
                      auto_attach_on_start: e.target.checked,
                    }))
                  }
                  className="h-4 w-4"
                />
                <label htmlFor="auto_attach_on_start" className="text-sm">
                  {t("autoAttachOnStartLabel")}
                </label>
                <InfoTooltip
                  title={t("autoAttachOnStartTitle")}
                  content={t.raw("autoAttachOnStartContent")}
                />
              </div>
            )}

            <div className="grid grid-cols-3 gap-3">
              <div className="flex items-center gap-2 pt-5">
                <input
                  id="cancel_if_invalid"
                  type="checkbox"
                  checked={form.cancel_if_invalid}
                  onChange={(e) =>
                    setForm((f) => ({
                      ...f,
                      cancel_if_invalid: e.target.checked,
                    }))
                  }
                  className="h-4 w-4"
                />
                <label htmlFor="cancel_if_invalid" className="text-sm">
                  cancel_if_invalid
                </label>
                <InfoTooltip
                  title={t("cancelIfInvalidTitle")}
                  content={t.raw("cancelIfInvalidContent")}
                />
              </div>
              <div className="space-y-1">
                <div className="flex items-center gap-1">
                  <Label>level</Label>
                  <InfoTooltip
                    title={t("levelTitle")}
                    content={t.raw("levelContent")}
                  />
                </div>
                <Input
                  type="number"
                  min={0}
                  max={10}
                  value={form.level}
                  onChange={(e) =>
                    setForm((f) => ({
                      ...f,
                      level: parseInt(e.target.value) || 0,
                    }))
                  }
                />
              </div>
              <div className="space-y-1">
                <div className="flex items-center gap-1">
                  <Label>ledger</Label>
                  <InfoTooltip
                    title={t("ledgerTitle")}
                    content={t.raw("ledgerContent")}
                  />
                </div>
                <Select
                  className="rounded px-2"
                  value={form.ledger}
                  onChange={(e) =>
                    setForm((f) => ({
                      ...f,
                      ledger: e.target.value as IdeaLedger,
                    }))
                  }
                >
                  {LEDGER_OPTIONS.map((opt) => (
                    <option key={opt.value} value={opt.value}>
                      {localizedOptionLabel(opt.label, locale)}
                    </option>
                  ))}
                </Select>
              </div>
            </div>

            {jsonError && (
              <div className="rounded-md border border-destructive/40 bg-destructive/10 px-3 py-2 text-xs text-destructive">
                ⚠ {jsonError}
              </div>
            )}
            <div className="space-y-1">
              <div className="flex items-center gap-1">
                <Label>targeted_modifier (JSON list)</Label>
                <InfoTooltip
                  title={t("targetedModifierTitle")}
                  content={t.raw("targetedModifierContent")}
                />
              </div>
              <Textarea
                rows={3}
                value={targetedModifierJson}
                onChange={(e) => setTargetedModifierJson(e.target.value)}
                placeholder='[{"tag": "USA", "modifier": {"trade_opinion_factor": 0.2}}]'
                className="font-mono text-xs"
              />
            </div>
            <div className="space-y-1">
              <div className="flex items-center gap-1">
                <Label>equipment_bonus (JSON dict)</Label>
                <InfoTooltip
                  title={t("equipmentBonusTitle")}
                  content={t.raw("equipmentBonusContent")}
                />
              </div>

              <div className="mt-1 flex flex-wrap items-center gap-1 text-xs">
                <span className="text-muted-foreground">
                  {t("equipmentBonusQuickInsert")}
                </span>
                {COMMON_ARCHETYPES.filter(
                  (arch) => archetypes.size === 0 || archetypes.has(arch),
                ).map((arch) => (
                  <button
                    key={arch}
                    type="button"
                    onClick={() => insertArchetypeChip(arch)}
                    className="rounded border border-input bg-background px-1.5 py-0.5 font-mono text-micro hover:bg-accent hover:text-accent-foreground"
                    title={t("equipmentBonusChipTooltip", { archetype: arch })}
                  >
                    {arch}
                  </button>
                ))}
              </div>
              <Textarea
                rows={3}
                value={equipmentBonusJson}
                onChange={(e) => setEquipmentBonusJson(e.target.value)}
                placeholder='{"infantry_equipment": {"soft_attack": 0.1}, "instant": true}'
                className="mt-1 font-mono text-xs"
              />
              {equipmentBonusWarning?.type === "instance" && (
                <div className="mt-1 text-xs text-warning">
                  {t("equipmentBonusInstanceWarning", {
                    key: equipmentBonusWarning.key,
                    archetype: equipmentBonusWarning.archetype ?? "",
                  })}
                </div>
              )}
              {equipmentBonusWarning?.type === "unknown" && (
                <div className="mt-1 text-xs text-warning">
                  {t("equipmentBonusUnknownWarning", {
                    key: equipmentBonusWarning.key,
                  })}
                </div>
              )}
            </div>
            <div className="space-y-1">
              <div className="flex items-center gap-1">
                <Label>rule (JSON dict)</Label>
                <InfoTooltip
                  title={t("ruleTitle")}
                  content={t.raw("ruleContent")}
                />
              </div>
              <Textarea
                rows={2}
                value={ruleJson}
                onChange={(e) => setRuleJson(e.target.value)}
                placeholder='{"can_create_factions": true}'
                className="font-mono text-xs"
              />
            </div>
          </div>
        </details>

        {idea.raw_extras && Object.keys(idea.raw_extras).length > 0 && (
          <>
            <Separator />
            <div className="space-y-1">
              <Label className="text-warning">
                {t("preservedFieldsHeader", {
                  count: Object.keys(idea.raw_extras).length,
                })}
              </Label>
              <div className="flex flex-wrap gap-1 mt-1">
                {Object.keys(idea.raw_extras).map((k) => (
                  <Badge
                    key={k}
                    variant="outline"
                    className="text-warning border-warning/40"
                  >
                    {k}
                  </Badge>
                ))}
              </div>
              <p className="text-xs text-muted-foreground mt-1">
                {t("preservedFieldsNote")}
              </p>
            </div>
          </>
        )}
      </div>

      <SourceEditorDialog
        open={showSourceDialog}
        entityType="idea"
        entityId={idea.idea_id}
        entityLabel={idea.name_en || idea.name || idea.idea_id}
        defaultNewId={`${idea.idea_id}_copy`}
        onChoice={handleSourceDialogChoice}
        onCancel={handleSourceDialogCancel}
      />
    </TooltipProvider>
  );
}
