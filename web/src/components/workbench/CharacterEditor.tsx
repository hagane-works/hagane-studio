"use client";

import { useState, useEffect, useCallback, useMemo, useRef } from "react";
import { useTranslations } from "use-intl";
import {
  nextDraftOnEntityChange,
  stableFingerprint,
} from "@/components/shared/editors/draftReseed";
import { useApplyReseedNotice } from "@/components/shared/editors/useApplyReseedNotice";
import {
  CUSTOM_VOCAB_SENTINEL,
  baseModOnlyOptions,
  resolveVocabularySelection,
} from "@/components/shared/editors/vocabularyMode";
import {
  confirmDiscardIfDirty,
  useRegisterDirty,
} from "@/lib/hooks/useUnsavedGuard";
import { useContentLocaleStore } from "@/lib/store/contentLocaleStore";
import { Link } from "@/i18n/navigation";
import { localizedName, resolveDisplayName } from "@/lib/i18n/localizedName";
import { ChevronDown, ChevronRight } from "lucide-react";
import { Input } from "@/components/ui/input";
import { Select } from "@/components/ui/select";
import { Label } from "@/components/ui/label";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Separator } from "@/components/ui/separator";
import type {
  Character,
  CustomTrait,
  RoleType,
  TraitApplicabilityMetadata,
} from "@/types";
import {
  ROLE_TYPE_OPTIONS,
  ADVISOR_SLOT_OPTIONS,
  SCIENTIST_SPECIALIZATION_OPTIONS,
  filterUnitLeaderTraitsForRole,
  getVisibleVanillaTraitsForPicker,
  isUnitLeaderTraitEligibleForRole,
  SCIENTIST_GENERIC_TRAITS,
  SCIENTIST_SPEC_TRAITS,
} from "@/types";
import {
  useIdeologyOptions,
  CUSTOM_IDEOLOGY_SENTINEL,
} from "@/lib/ideology/useIdeologyOptions";
import type { ScientistSpecialization } from "@/types";
import { PortraitUpload } from "./PortraitUpload";
import { SaveToModFolderButton } from "./SaveToModFolderButton";
import { CustomTraitsPanel } from "./CustomTraitsPanel";
import {
  ScriptBlockEditor,
  EFFECT_TEMPLATES,
  CONDITION_TEMPLATES,
  extractSaveErrorMessage,
} from "@/components/shared/editors";
import { DisplayNameLocsTabs } from "@/components/shared/editors/DisplayNameLocsTabs";
import { DescriptionLocsTabs } from "@/components/shared/editors/DescriptionLocsTabs";
import { DEFAULT_EXPORT_LANGUAGES } from "@/components/shared/editors/displayNameLocsHelpers";
import { InfoTooltip, TooltipProvider } from "@/components/ui/tooltip";
import { API_BASE, charactersApi } from "@/lib/api";
import {
  characterTagOwners,
  characterTagTaken,
  isCharacterTagConflict,
} from "./characterTagTaken";
import { SourceEditorDialog } from "@/shared/dialogs/SourceEditorDialog";
import { useImportedSourceGate } from "@/shared/dialogs/useImportedSourceGate";
import { useProjectStore } from "@/lib/store/projectStore";
import { useBaseModVocabulary } from "@/lib/hooks/useBaseModVocabulary";
import {
  useBaseModIdCollision,
  BaseModIdCollisionNotice,
} from "./baseModIdCollision";

interface CharacterEditorProps {
  projectId: number;
  character: Character;
  onSave: (data: Partial<Character>) => Promise<void>;
  onRefresh: () => void;
  onDraftChange?: (draft: Character) => void;

  onFlipBack?: () => Promise<void>;

  onForked?: (forked: Character) => void;

  projectCharacters?: readonly Character[];
}

interface KnownTraitCatalogCustomEntry {
  trait_id: string;
  name_en: string;
  name_zh: string;
  metadata: TraitApplicabilityMetadata;
}

interface KnownTraitCatalog {
  vanilla: string[];
  vanilla_metadata: Record<string, TraitApplicabilityMetadata>;
  custom: KnownTraitCatalogCustomEntry[];
}

const LOCALISATION_KEY_RE = /^[A-Za-z0-9_.'-]+$/;

function deriveLocalisationKey(tag: string): string {
  const folded = (tag || "")
    .trim()
    .normalize("NFKD")
    .replace(/[\u0300-\u036f]/g, "");
  return folded.replace(/[^A-Za-z0-9_.'-]/g, "_");
}

const EMPTY_KNOWN_TRAIT_CATALOG: KnownTraitCatalog = {
  vanilla: [],
  vanilla_metadata: {},
  custom: [],
};

export function CharacterEditor({
  projectId,
  character,
  onSave,
  onRefresh,
  onDraftChange,
  onFlipBack,
  onForked,
  projectCharacters,
}: CharacterEditorProps) {
  const t = useTranslations("characterEditor");

  const tEnums = useTranslations("enums");

  const tIdiom = useTranslations("idiomWarning");

  const tCommon = useTranslations("common");
  const baseModVocab = useBaseModVocabulary(projectId);

  const locale = useContentLocaleStore((s) => s.locale);

  const exportLanguages = useProjectStore(
    (s) => s.currentProject?.export_languages ?? DEFAULT_EXPORT_LANGUAGES,
  );

  const { subOptions: ideologySubOptions, knownSubIds: knownIdeologySubIds } =
    useIdeologyOptions();
  const [ideologyCustomMode, setIdeologyCustomMode] = useState(false);

  const [advisorSlotCustomMode, setAdvisorSlotCustomMode] = useState(false);
  const [form, setForm] = useState(character);

  const baseModId = useBaseModIdCollision("character", form.tag || "");
  const tagOwners = useMemo(
    () => characterTagOwners(projectCharacters ?? []),
    [projectCharacters],
  );
  const tagTaken = characterTagTaken(tagOwners, form.tag || "", character.id);

  const vanillaAdvisorSlots = useMemo(
    () => ADVISOR_SLOT_OPTIONS.map((o) => o.value as string),
    [],
  );
  const baseModAdvisorSlots = useMemo(
    () => baseModOnlyOptions(vanillaAdvisorSlots, baseModVocab.advisor_slots),
    [vanillaAdvisorSlots, baseModVocab.advisor_slots],
  );
  const advisorSlotSel = resolveVocabularySelection(
    form.advisor_slot,
    vanillaAdvisorSlots,
    baseModVocab.advisor_slots,
    advisorSlotCustomMode,
  );
  const [saving, setSaving] = useState(false);
  const [saveStatus, setSaveStatus] = useState<"idle" | "success" | "error">(
    "idle",
  );

  const [saveError, setSaveError] = useState<string | null>(null);

  const isUnsaved = useMemo(
    () => JSON.stringify(form) !== JSON.stringify(character),
    [form, character],
  );

  useRegisterDirty("characters", isUnsaved);
  const [showAdvancedAdvisor, setShowAdvancedAdvisor] = useState(false);
  const [customTraits, setCustomTraits] = useState<CustomTrait[]>([]);
  const [customTraitsOpen, setCustomTraitsOpen] = useState(false);
  const [guideDismissed, setGuideDismissed] = useState(
    () =>
      typeof window !== "undefined" &&
      localStorage.getItem("char-guide-dismissed") === "1",
  );
  const handleDismissGuide = () => {
    localStorage.setItem("char-guide-dismissed", "1");
    setGuideDismissed(true);
  };

  const [knownTraitCatalogs, setKnownTraitCatalogs] = useState<{
    country_leader: KnownTraitCatalog;
    unit_leader: KnownTraitCatalog;
  }>({
    country_leader: EMPTY_KNOWN_TRAIT_CATALOG,
    unit_leader: EMPTY_KNOWN_TRAIT_CATALOG,
  });
  const [traitSearch, setTraitSearch] = useState("");
  const [advisorTraitSearch, setAdvisorTraitSearch] = useState("");

  const fetchCustomTraits = useCallback(async () => {
    try {
      const res = await fetch(
        `${API_BASE}/go/projects/${projectId}/custom-traits`,
      );
      if (res.ok) {
        const data: CustomTrait[] = await res.json();
        setCustomTraits(data);
      }
    } catch {}
  }, [projectId]);

  const characterFingerprint = useCallback(
    (c: Character) =>
      stableFingerprint(c, [
        "large_portrait",
        "small_portrait",
        "is_portrait_modified",
        "warnings",
      ]),
    [],
  );
  const notifyReseed = useApplyReseedNotice();
  const seededCharRef = useRef({
    id: character.id,
    fingerprint: characterFingerprint(character),
  });
  useEffect(() => {
    const decision = nextDraftOnEntityChange<Character, number>({
      prevSeededId: seededCharRef.current.id,
      prevFingerprint: seededCharRef.current.fingerprint,
      entityId: character.id,
      entity: character,
      currentDraft: form,
      fingerprintOf: characterFingerprint,
      syncFieldsOnSameId: [
        "large_portrait",
        "small_portrait",
        "is_portrait_modified",
      ],
    });
    seededCharRef.current = {
      id: decision.seededId,
      fingerprint: decision.fingerprint,
    };
    if (decision.reseeded || decision.draft !== form) {
      setForm(decision.draft);
    }
    notifyReseed(decision);
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [character, characterFingerprint, notifyReseed]);

  useEffect(() => {
    onDraftChange?.(form);
  }, [form, onDraftChange]);

  useEffect(() => {
    fetchCustomTraits();
  }, [fetchCustomTraits]);

  useEffect(() => {
    let cancelled = false;
    (async () => {
      try {
        const [countryLeaderRes, unitLeaderRes] = await Promise.all([
          fetch(
            `/api/go/projects/${projectId}/custom-traits/known?trait_type=country_leader`,
          ),
          fetch(
            `/api/go/projects/${projectId}/custom-traits/known?trait_type=unit_leader`,
          ),
        ]);
        if (!countryLeaderRes.ok || !unitLeaderRes.ok) return;
        const [countryLeader, unitLeader] = await Promise.all([
          countryLeaderRes.json(),
          unitLeaderRes.json(),
        ]);
        if (cancelled) return;
        setKnownTraitCatalogs({
          country_leader: {
            vanilla: countryLeader.vanilla || [],
            vanilla_metadata: countryLeader.vanilla_metadata || {},
            custom: countryLeader.custom || [],
          },
          unit_leader: {
            vanilla: unitLeader.vanilla || [],
            vanilla_metadata: unitLeader.vanilla_metadata || {},
            custom: unitLeader.custom || [],
          },
        });
      } catch {}
    })();
    return () => {
      cancelled = true;
    };
  }, [projectId, customTraitsOpen]);

  useEffect(() => {
    if (!customTraitsOpen) {
      fetchCustomTraits();
    }
  }, [customTraitsOpen, fetchCustomTraits]);

  const update = useCallback((field: string, value: unknown) => {
    setForm((prev) => ({ ...prev, [field]: value }));
  }, []);

  const sourceGate = useImportedSourceGate<Partial<Character>>({
    entitySource: character.source,
    performSave: async (payload) => {
      setSaving(true);
      setSaveStatus("idle");
      setSaveError(null);
      try {
        await onSave(payload);
        setSaveStatus("success");
        setTimeout(() => setSaveStatus("idle"), 2000);
      } catch (err) {
        console.error("Save failed:", err);
        setSaveError(
          isCharacterTagConflict(err)
            ? t("tagTaken", { tag: (payload.tag || "").trim() })
            : extractSaveErrorMessage(err),
        );
        setSaveStatus("error");
      } finally {
        setSaving(false);
      }
    },
    performFork: onForked
      ? async (newTag) => {
          setSaving(true);
          setSaveStatus("idle");
          setSaveError(null);
          try {
            const forked = await charactersApi.fork(
              projectId,
              character.id,
              newTag,
            );
            setSaveStatus("success");
            setTimeout(() => setSaveStatus("idle"), 2000);
            onForked(forked);
          } catch (err) {
            console.error("Fork failed:", err);
            setSaveError(extractSaveErrorMessage(err, "Fork failed"));
            setSaveStatus("error");
          } finally {
            setSaving(false);
          }
        }
      : undefined,
    resetForm: () => setForm(character),
  });

  const handleSave = async () => {
    const localisationKey = (form.localisation_key || "").trim();
    if (localisationKey && !LOCALISATION_KEY_RE.test(localisationKey)) {
      setSaveError(t("localisationKeyInvalid"));
      setSaveStatus("error");
      return;
    }

    if (await baseModId.verify((form.tag || "").trim())) return;
    if (tagTaken) {
      setSaveError(t("tagTaken", { tag: (form.tag || "").trim() }));
      setSaveStatus("error");
      return;
    }
    const payload: Partial<Character> = {
      ...form,
      tag: (form.tag || "").trim(),
      localisation_key: localisationKey,
    };
    if (sourceGate.interceptSave(payload)) return;
    setSaving(true);
    setSaveStatus("idle");
    setSaveError(null);
    try {
      await onSave(payload);
      setSaveStatus("success");
      setTimeout(() => setSaveStatus("idle"), 2000);
    } catch (err) {
      console.error("Save failed:", err);
      setSaveError(
        isCharacterTagConflict(err)
          ? t("tagTaken", { tag: payload.tag || "" })
          : extractSaveErrorMessage(err),
      );
      setSaveStatus("error");
    } finally {
      setSaving(false);
    }
  };

  const isMilitary = ["general", "field_marshal", "admiral"].includes(
    form.role_type,
  );
  const isAdmiral = form.role_type === "admiral";
  const isScientist = form.role_type === "scientist";
  const isCountryLeader = form.role_type === "country_leader";
  const isPureAdvisor = form.role_type === "advisor";

  const isOperative = form.role_type === "operative";

  const legacyAlwaysShowOnActions = String(
    form.raw_extras?.["advisor_always_show_on_actions_tooltip"] ?? "",
  )
    .trim()
    .toLowerCase();

  const scientistCustomTraits = useMemo(
    () => customTraits.filter((trait) => trait.trait_type === "scientist"),
    [customTraits],
  );

  const primaryUsesUnitCatalog = isMilitary || isOperative;
  const primaryCatalog = primaryUsesUnitCatalog
    ? knownTraitCatalogs.unit_leader
    : knownTraitCatalogs.country_leader;
  const primaryVanillaTraits = useMemo(() => {
    if (!primaryUsesUnitCatalog) return primaryCatalog.vanilla;
    return filterUnitLeaderTraitsForRole(
      form.role_type,
      primaryCatalog.vanilla,
      primaryCatalog.vanilla_metadata,
    );
  }, [form.role_type, primaryCatalog, primaryUsesUnitCatalog]);
  const primaryCustomTraits = useMemo(() => {
    if (!primaryUsesUnitCatalog) return primaryCatalog.custom;
    return primaryCatalog.custom.filter((trait) =>
      isUnitLeaderTraitEligibleForRole(form.role_type, trait.metadata),
    );
  }, [form.role_type, primaryCatalog, primaryUsesUnitCatalog]);
  const selectedPrimaryTraits = useMemo(
    () => (isOperative ? form.operative_traits || [] : form.traits || []),
    [form.operative_traits, form.traits, isOperative],
  );
  const primaryKnownTraitSet = useMemo(
    () =>
      new Set([
        ...primaryVanillaTraits,
        ...primaryCustomTraits.map((trait) => trait.trait_id),
      ]),
    [primaryCustomTraits, primaryVanillaTraits],
  );
  const invalidPrimaryTraits = useMemo(() => {
    if (isScientist) return new Set<string>();
    return new Set(
      selectedPrimaryTraits.filter((trait) => !primaryKnownTraitSet.has(trait)),
    );
  }, [isScientist, primaryKnownTraitSet, selectedPrimaryTraits]);

  const filteredKnownTraits = useMemo(() => {
    const q = traitSearch.trim().toLowerCase();
    const matchVanilla = getVisibleVanillaTraitsForPicker(
      form.role_type,
      primaryVanillaTraits,
      q,
    );
    const matchCustom = primaryCustomTraits.filter(
      (trait) => !q || trait.trait_id.toLowerCase().includes(q),
    );
    return { vanilla: matchVanilla, custom: matchCustom };
  }, [form.role_type, primaryCustomTraits, primaryVanillaTraits, traitSearch]);

  const advisorKnownTraitSet = useMemo(
    () =>
      new Set([
        ...knownTraitCatalogs.country_leader.vanilla,
        ...knownTraitCatalogs.country_leader.custom.map(
          (trait) => trait.trait_id,
        ),
      ]),
    [knownTraitCatalogs.country_leader],
  );
  const invalidAdvisorTraits = useMemo(
    () =>
      new Set(
        (form.advisor_traits || []).filter(
          (trait) => !advisorKnownTraitSet.has(trait),
        ),
      ),
    [advisorKnownTraitSet, form.advisor_traits],
  );
  const filteredAdvisorTraits = useMemo(() => {
    const q = advisorTraitSearch.trim().toLowerCase();
    const custom = knownTraitCatalogs.country_leader.custom;
    return {
      vanilla: getVisibleVanillaTraitsForPicker(
        "advisor",
        knownTraitCatalogs.country_leader.vanilla,
        q,
      ),
      custom: custom.filter(
        (trait) => !q || trait.trait_id.toLowerCase().includes(q),
      ),
    };
  }, [advisorTraitSearch, knownTraitCatalogs.country_leader]);

  const toggleTrait = (trait: string) => {
    const field = isOperative ? "operative_traits" : "traits";
    const current = isOperative
      ? form.operative_traits || []
      : form.traits || [];
    const next = current.includes(trait)
      ? current.filter((t) => t !== trait)
      : [...current, trait];
    update(field, next);
  };

  const toggleAdvisorTrait = (trait: string) => {
    const current = form.advisor_traits || [];
    const next = current.includes(trait)
      ? current.filter((t) => t !== trait)
      : [...current, trait];
    update("advisor_traits", next);
  };

  const isDualRole = form.has_advisor_role && form.role_type !== "advisor";
  const primaryRoleOpt = ROLE_TYPE_OPTIONS.find(
    (r) => r.value === form.role_type,
  );
  const primaryRoleLabel = primaryRoleOpt
    ? tEnums(primaryRoleOpt.label_i18n_key)
    : form.role_type;

  const legacyEnglishName =
    (form.name_en || "").trim().toLocaleLowerCase() === "new character"
      ? ""
      : form.name_en;
  const explicitLocalisationKey = (form.localisation_key || "").trim();
  const localisationKeyInvalid = Boolean(
    explicitLocalisationKey &&
    !LOCALISATION_KEY_RE.test(explicitLocalisationKey),
  );
  const derivedLocalisationKey = deriveLocalisationKey(form.tag);
  const localisationStatusMessages = {
    imported: t("localisationStatusImported"),
    saved: t("localisationStatusSaved"),
    legacy: t("localisationStatusLegacy"),
    missing: t("localisationStatusMissing"),
  };

  return (
    <TooltipProvider>
      <div className="p-6 space-y-6 max-w-2xl mx-auto">
        {(saving || saveStatus !== "idle" || isUnsaved) && (
          <div
            className={`
            sticky top-0 z-10 -mx-6 -mt-6 px-6 py-1.5 text-xs font-medium flex items-center gap-2 shrink-0
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
                      : ""
            }
          `}
          >
            <span>
              {saving
                ? t("saving")
                : saveStatus === "success"
                  ? t("saveSuccess")
                  : saveStatus === "error"
                    ? (saveError ?? t("saveError"))
                    : isUnsaved
                      ? t("unsavedChanges")
                      : ""}
            </span>
          </div>
        )}
        {isDualRole && (
          <div className="rounded-md border border-amber-500/20 bg-amber-500/5 px-4 py-3 text-xs">
            <p className="font-medium text-amber-700 dark:text-amber-400">
              {t("dualRoleTitle")}
            </p>
            <p className="mt-1 text-muted-foreground">
              {t("primaryRoleLabel")}{" "}
              <span className="font-mono">{primaryRoleLabel}</span>
              {t("dualRoleDesc")}
            </p>
          </div>
        )}

        <section className="space-y-3">
          <h3 className="text-xs font-semibold text-foreground uppercase tracking-widest pl-2 border-l border-accent/40 mb-2">
            {t("basicInfoSection")}
          </h3>

          <div className="space-y-1">
            <Label>{t("displayNameLabel")}</Label>
            <DisplayNameLocsTabs
              entityKey={`character:${character.id}:display-name`}
              value={form.display_name_locs || {}}
              exportLanguages={exportLanguages}
              legacyFallback={{ en: legacyEnglishName, zh: form.name_zh }}
              onChange={(next) => update("display_name_locs", next)}
              placeholder={t("displayNamePlaceholder")}
              ariaLabel={t("displayNameAria")}
              valueSource={
                character.source === "imported" ? "imported" : "saved"
              }
              statusMessages={localisationStatusMessages}
            />
          </div>

          <div className="space-y-1">
            <Label>{t("descriptionLabel")}</Label>
            <DescriptionLocsTabs
              entityKey={`character:${character.id}:description`}
              value={
                (form as { description_locs?: Record<string, string> })
                  .description_locs || {}
              }
              exportLanguages={exportLanguages}
              legacyFallback={{ en: form.desc_en, zh: form.desc_zh }}
              onChange={(next) => update("description_locs", next)}
              placeholder={t("descriptionPlaceholder")}
              ariaLabel={t("descriptionAria")}
              rows={3}
              valueSource={
                character.source === "imported" ? "imported" : "saved"
              }
              statusMessages={localisationStatusMessages}
            />
          </div>
          <div className="space-y-1">
            <Label>{t("tagLabel")}</Label>
            <Input
              value={form.tag}
              onChange={(e) => update("tag", e.target.value)}
              placeholder={t("tagPlaceholder")}
              aria-invalid={baseModId.collision || tagTaken ? true : undefined}
              aria-describedby={tagTaken ? "character-tag-taken" : undefined}
              className={`font-mono text-xs ${baseModId.collision || tagTaken ? "border-destructive" : ""}`}
            />
            <BaseModIdCollisionNotice
              entityType="character"
              collision={baseModId.collision}
            />
            {tagTaken && (
              <p id="character-tag-taken" className="text-xs text-destructive">
                {t("tagTaken", { tag: (form.tag || "").trim() })}
              </p>
            )}
          </div>
          <div className="space-y-1">
            <Label>{t("localisationKeyLabel")}</Label>
            <Input
              value={form.localisation_key || ""}
              onChange={(e) => update("localisation_key", e.target.value)}
              placeholder={
                derivedLocalisationKey || t("localisationKeyPlaceholder")
              }
              aria-invalid={localisationKeyInvalid}
              className="font-mono text-xs"
            />
            <p
              className={
                localisationKeyInvalid
                  ? "text-micro text-destructive"
                  : "text-micro text-muted-foreground"
              }
            >
              {localisationKeyInvalid
                ? t("localisationKeyInvalid")
                : explicitLocalisationKey
                  ? t("localisationKeyExplicitHint")
                  : t("localisationKeyGeneratedHint", {
                      key:
                        derivedLocalisationKey ||
                        t("localisationKeyPlaceholder"),
                    })}
            </p>
          </div>
          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-1">
              <Label>{t("roleTypeLabel")}</Label>
              <Select
                value={form.role_type}
                onChange={(e) =>
                  update("role_type", e.target.value as RoleType)
                }
              >
                {ROLE_TYPE_OPTIONS.map((opt) => (
                  <option key={opt.value} value={opt.value}>
                    {tEnums(opt.label_i18n_key)}
                  </option>
                ))}
              </Select>
            </div>
            <div className="space-y-1">
              <Label>{t("genderLabel")}</Label>
              <Select
                value={form.gender || ""}
                onChange={(e) => update("gender", e.target.value)}
              >
                <option value="">{t("genderMaleDefault")}</option>
                <option value="female">{t("genderFemale")}</option>
              </Select>
              {!form.gender && (form.large_portrait || form.small_portrait) && (
                <p className="text-xs text-amber-600 dark:text-amber-400">
                  {t("genderHint")}
                </p>
              )}
            </div>
          </div>

          <div className="space-y-1">
            <label className="flex items-center gap-2 cursor-pointer">
              <input
                type="checkbox"
                checked={form.can_be_captured !== false}
                onChange={(e) => update("can_be_captured", e.target.checked)}
                className="rounded"
              />
              <span className="text-sm">{t("canBeCapturedLabel")}</span>
            </label>
            <p className="text-xs text-muted-foreground ml-6">
              {form.can_be_captured !== false
                ? t("canBeCapturedOn")
                : t("canBeCapturedOff")}
            </p>
          </div>

          <div className="space-y-1">
            <label className="flex items-center gap-2 cursor-pointer">
              <input
                type="checkbox"
                checked={form.is_replacement || false}
                onChange={(e) => update("is_replacement", e.target.checked)}
                className="rounded"
              />
              <span className="text-sm">{t("replaceVanillaLabel")}</span>
            </label>
            <p className="text-xs text-muted-foreground ml-6">
              {form.is_replacement
                ? t("replaceVanillaOn")
                : t("replaceVanillaOff")}
            </p>
          </div>

          <div className="space-y-1">
            <label className="flex items-center gap-2 cursor-pointer">
              <input
                type="checkbox"
                checked={form.replaces_vanilla_id || false}
                onChange={(e) =>
                  update("replaces_vanilla_id", e.target.checked)
                }
                className="rounded"
              />
              <span className="text-sm">{t("overlayVanillaLabel")}</span>
            </label>
            <p className="text-xs text-muted-foreground ml-6">
              {form.replaces_vanilla_id
                ? t("overlayVanillaOn")
                : t.raw("overlayVanillaOff")}
            </p>
          </div>

          {form.replaces_vanilla_id && (
            <div className="space-y-1">
              <label className="flex items-center gap-2 cursor-pointer">
                <input
                  type="checkbox"
                  checked={
                    (form.raw_extras?.["emit_trait_overlay_event"] as
                      boolean | undefined) !== false
                  }
                  onChange={(e) => {
                    const next = { ...(form.raw_extras || {}) };
                    if (e.target.checked) {
                      delete next["emit_trait_overlay_event"];
                    } else {
                      next["emit_trait_overlay_event"] = false;
                    }
                    update("raw_extras", next);
                  }}
                  className="rounded"
                />
                <span className="text-sm">{t("emitTraitOverlayLabel")}</span>
              </label>
              <p className="text-xs text-muted-foreground ml-6">
                {(form.raw_extras?.["emit_trait_overlay_event"] as
                  boolean | undefined) !== false
                  ? t.raw("emitTraitOverlayOn")
                  : t.raw("emitTraitOverlayOff")}
              </p>
            </div>
          )}
        </section>

        <Separator />

        {isMilitary && (
          <section className="space-y-3">
            <h3 className="text-xs font-semibold text-foreground uppercase tracking-widest pl-2 border-l border-accent/40 mb-2">
              {isAdmiral ? t("navyStatsSection") : t("armyStatsSection")}
            </h3>

            <div className="grid grid-cols-3 gap-3">
              <div className="space-y-1">
                <Label>{t("skillLabel")}</Label>
                <Input
                  type="number"
                  min={0}
                  max={9}
                  className="w-20"
                  value={form.skill}
                  onChange={(e) => update("skill", Number(e.target.value))}
                />
              </div>
              <div className="space-y-1">
                <Label>{t("attackLabel")}</Label>
                <Input
                  type="number"
                  min={0}
                  max={9}
                  className="w-20"
                  value={form.attack_skill}
                  onChange={(e) =>
                    update("attack_skill", Number(e.target.value))
                  }
                />
              </div>
              <div className="space-y-1">
                <Label>{t("defenseLabel")}</Label>
                <Input
                  type="number"
                  min={0}
                  max={9}
                  className="w-20"
                  value={form.defense_skill}
                  onChange={(e) =>
                    update("defense_skill", Number(e.target.value))
                  }
                />
              </div>
              {isAdmiral ? (
                <>
                  <div className="space-y-1">
                    <Label>{t("maneuveringLabel")}</Label>
                    <Input
                      type="number"
                      min={0}
                      max={9}
                      className="w-20"
                      value={form.maneuvering_skill}
                      onChange={(e) =>
                        update("maneuvering_skill", Number(e.target.value))
                      }
                    />
                  </div>
                  <div className="space-y-1">
                    <Label>{t("coordinationLabel")}</Label>
                    <Input
                      type="number"
                      min={0}
                      max={9}
                      className="w-20"
                      value={form.coordination_skill}
                      onChange={(e) =>
                        update("coordination_skill", Number(e.target.value))
                      }
                    />
                  </div>
                </>
              ) : (
                <>
                  <div className="space-y-1">
                    <Label>{t("planningLabel")}</Label>
                    <Input
                      type="number"
                      min={0}
                      max={9}
                      className="w-20"
                      value={form.planning_skill}
                      onChange={(e) =>
                        update("planning_skill", Number(e.target.value))
                      }
                    />
                  </div>
                  <div className="space-y-1">
                    <Label>{t("logisticsLabel")}</Label>
                    <Input
                      type="number"
                      min={0}
                      max={9}
                      className="w-20"
                      value={form.logistics_skill}
                      onChange={(e) =>
                        update("logistics_skill", Number(e.target.value))
                      }
                    />
                  </div>
                </>
              )}
            </div>

            <div className="space-y-1">
              <Label>{t("militaryVisibleLabel")}</Label>
              <textarea
                value={form.military_visible || ""}
                onChange={(e) => update("military_visible", e.target.value)}
                placeholder={
                  'has_dlc = "No Step Back"\nhas_country_flag = my_flag'
                }
                className="w-full min-h-20 px-2 py-1 text-xs font-mono border rounded-md bg-background"
              />
              <p className="text-xs text-muted-foreground">
                {t("militaryVisibleHint")}
              </p>
            </div>
          </section>
        )}

        {isCountryLeader && (
          <section className="space-y-3">
            <h3 className="text-xs font-semibold text-foreground uppercase tracking-widest pl-2 border-l border-accent/40 mb-2">
              {t("countryLeaderSection")}
            </h3>
            <div className="grid grid-cols-2 gap-3">
              <div className="space-y-1">
                <Label>{t("subIdeologyLabel")}</Label>
                {(() => {
                  const ideologyIsCustom =
                    ideologyCustomMode ||
                    (!!form.ideology &&
                      !knownIdeologySubIds.has(form.ideology));
                  return (
                    <>
                      <Select
                        value={
                          ideologyIsCustom
                            ? CUSTOM_IDEOLOGY_SENTINEL
                            : form.ideology || ""
                        }
                        onChange={(e) => {
                          const v = e.target.value;
                          if (v === CUSTOM_IDEOLOGY_SENTINEL) {
                            setIdeologyCustomMode(true);
                          } else {
                            setIdeologyCustomMode(false);
                            update("ideology", v);
                          }
                        }}
                      >
                        <option value="">
                          {t("ideologySelectPlaceholder")}
                        </option>
                        {ideologySubOptions.map((opt) => (
                          <option key={opt.value} value={opt.value}>
                            {opt.label}
                          </option>
                        ))}
                        <option value={CUSTOM_IDEOLOGY_SENTINEL}>
                          {t("ideologyCustomOption")}
                        </option>
                      </Select>
                      {ideologyIsCustom && (
                        <Input
                          value={form.ideology || ""}
                          onChange={(e) => update("ideology", e.target.value)}
                          placeholder="national_populist"
                          className="mt-1 font-mono text-xs"
                        />
                      )}
                    </>
                  );
                })()}
                <Link
                  href={`/project/${projectId}/ideologies`}
                  onNavigate={(event) => {
                    if (!confirmDiscardIfDirty()) event.preventDefault();
                  }}
                  className="text-xs text-muted-foreground hover:text-foreground underline underline-offset-2 ui-interactive"
                >
                  {t("manageCustomIdeologies")}
                </Link>
              </div>
              <div className="space-y-1">
                <Label>{t("expireDateLabel")}</Label>
                <Input
                  value={form.expire || ""}
                  onChange={(e) => update("expire", e.target.value)}
                  placeholder={t("expireDatePlaceholder")}
                  className="font-mono text-xs"
                />
              </div>
            </div>
          </section>
        )}

        {isScientist && (
          <section className="space-y-3">
            <h3 className="text-xs font-semibold text-foreground uppercase tracking-widest pl-2 border-l border-accent/40 mb-2">
              {t("scientistSection")}
            </h3>
            <div className="grid grid-cols-2 gap-3">
              <div className="space-y-1">
                <Label>{t("specializationLabel")}</Label>
                <Select
                  value={
                    form.scientist_data?.specializations?.[0]?.spec ||
                    "specialization_land"
                  }
                  onChange={(e) => {
                    const newSpec = e.target.value as ScientistSpecialization;
                    const oldSpec = (form.scientist_data?.specializations?.[0]
                      ?.spec ||
                      "specialization_land") as ScientistSpecialization;

                    const oldSpecTraits = SCIENTIST_SPEC_TRAITS[oldSpec] || [];
                    const newSpecTraits = SCIENTIST_SPEC_TRAITS[newSpec] || [];
                    const cleanedTraits = (form.traits || []).filter(
                      (t) =>
                        !oldSpecTraits.includes(t) || newSpecTraits.includes(t),
                    );
                    update("traits", cleanedTraits);
                    update("scientist_data", {
                      specializations: [
                        {
                          spec: newSpec,
                          level:
                            form.scientist_data?.specializations?.[0]?.level ||
                            2,
                        },
                      ],
                      allow_multiple: false,
                    });
                  }}
                >
                  {SCIENTIST_SPECIALIZATION_OPTIONS.map((opt) => (
                    <option key={opt.value} value={opt.value}>
                      {tEnums(opt.label_i18n_key)}
                    </option>
                  ))}
                </Select>
              </div>
              <div className="space-y-1">
                <Label>{t("scientistLevelLabel")}</Label>
                <Input
                  type="number"
                  min={1}
                  max={5}
                  value={form.scientist_data?.specializations?.[0]?.level || 2}
                  onChange={(e) => {
                    const n = Number(e.target.value);
                    const level = Number.isFinite(n) && n >= 1 ? n : 1;
                    update("scientist_data", {
                      specializations: [
                        {
                          spec:
                            form.scientist_data?.specializations?.[0]?.spec ||
                            "specialization_land",
                          level,
                        },
                      ],
                      allow_multiple: false,
                    });
                  }}
                />
              </div>
            </div>

            <div className="space-y-1">
              <Label>{t("scientistVisibleLabel")}</Label>
              <textarea
                value={form.scientist_visible || ""}
                onChange={(e) => update("scientist_visible", e.target.value)}
                placeholder={
                  'has_dlc = "By Blood Alone"\nhas_country_flag = my_flag'
                }
                className="w-full min-h-20 px-2 py-1 text-xs font-mono border rounded-md bg-background"
              />
              <p className="text-xs text-muted-foreground">
                {t("scientistVisibleHint")}
              </p>
            </div>
          </section>
        )}

        {isOperative && (
          <section className="space-y-3">
            <h3 className="text-sm font-semibold text-muted-foreground uppercase tracking-wide">
              {t("operativeSection")}
            </h3>
            <div className="grid grid-cols-2 gap-3">
              <div className="space-y-1">
                <Label>{t("operativeSkillLevelLabel")}</Label>
                <Input
                  type="number"
                  min={1}
                  max={5}
                  value={form.operative_skill_level ?? 1}
                  onChange={(e) =>
                    update("operative_skill_level", Number(e.target.value) || 1)
                  }
                />
              </div>
              <div className="space-y-1">
                <Label>{t("operativeBgLabel")}</Label>
                <Input
                  value={form.operative_bg || ""}
                  onChange={(e) => update("operative_bg", e.target.value)}
                  placeholder="GFX_operative_bg_USA"
                />
              </div>
            </div>
            <div className="space-y-1">
              <Label>{t("nationalitiesLabel")}</Label>
              <textarea
                value={(form.operative_nationalities || []).join(" ")}
                onChange={(e) =>
                  update(
                    "operative_nationalities",
                    e.target.value
                      .split(/[\s,]+/)
                      .map((s) => s.trim().toUpperCase())
                      .filter(Boolean),
                  )
                }
                placeholder="USA ENG FRA"
                className="w-full min-h-12 px-2 py-1 text-xs font-mono border rounded-md bg-background"
              />
              <p className="text-xs text-muted-foreground">
                {t("nationalitiesHint")}
              </p>
            </div>
            <div className="space-y-1">
              <Label>{t("operativeTraitsLabel")}</Label>
              <p className="text-xs text-muted-foreground">
                {t("operativeTraitsCatalogHint")}
              </p>
            </div>
            <div className="space-y-1">
              <Label>{t("operativeAvailableLabel")}</Label>
              <textarea
                value={form.operative_available || ""}
                onChange={(e) => update("operative_available", e.target.value)}
                placeholder={t.raw("operativeAvailablePlaceholder")}
                className="w-full min-h-20 px-2 py-1 text-xs font-mono border rounded-md bg-background"
              />
              <p className="text-xs text-muted-foreground">
                {t("operativeAvailableHint")}
              </p>
            </div>
          </section>
        )}

        {!isPureAdvisor && isMilitary && !guideDismissed && (
          <div className="rounded-md border border-accent/40 bg-accent/5 p-3 text-xs space-y-1.5 relative">
            <button
              onClick={handleDismissGuide}
              className="absolute top-2 right-2 w-5 h-5 flex items-center justify-center rounded text-muted-foreground hover:text-foreground hover:bg-muted transition-colors"
              title={t("guideDismissTitle")}
              aria-label={t("guideDismissAria")}
            >
              ✕
            </button>
            <p className="font-semibold text-foreground">
              {t("guideTitle", { role: primaryRoleLabel })}
            </p>
            <p className="text-muted-foreground">{t("guideIntro")}</p>
            <ol className="list-decimal list-inside space-y-0.5 text-muted-foreground">
              <li>
                {t.rich("guideStep1", {
                  b: (c) => <strong className="text-foreground">{c}</strong>,
                  role: primaryRoleLabel,
                })}
              </li>
              <li>
                {t.rich("guideStep2", {
                  b: (c) => <strong className="text-foreground">{c}</strong>,
                })}
              </li>
              <li>
                {t.rich("guideStep3", {
                  b: (c) => <strong className="text-foreground">{c}</strong>,
                })}
              </li>
              <li>
                {t.rich("guideStep4", {
                  b: (c) => <strong className="text-foreground">{c}</strong>,
                })}
              </li>
            </ol>
            <p className="text-muted-foreground italic">{t("guideOutro")}</p>
          </div>
        )}

        {!isPureAdvisor && (
          <section className="space-y-3">
            <label className="flex items-center gap-2 cursor-pointer">
              <input
                type="checkbox"
                checked={form.has_advisor_role}
                onChange={(e) => update("has_advisor_role", e.target.checked)}
                className="rounded"
              />
              <span className="text-sm">
                {t("hasAdvisorRoleLabel", { role: primaryRoleLabel })}
              </span>
            </label>
            <p className="text-xs text-muted-foreground pl-6">
              {t("hasAdvisorRoleHint")}
            </p>
          </section>
        )}

        {(isPureAdvisor || form.has_advisor_role) && (
          <>
            <Separator />
            <section className="space-y-3">
              <h3 className="text-xs font-semibold text-foreground uppercase tracking-widest pl-2 border-l border-accent/40 mb-2">
                {t("advisorSettingsSection")}
              </h3>
              <div className="grid grid-cols-2 gap-3">
                <div className="space-y-1">
                  <Label>{t("advisorSlotLabel")}</Label>

                  <Select
                    value={advisorSlotSel.selectValue || "political_advisor"}
                    trailingChip={
                      advisorSlotSel.isCustom
                        ? tCommon("vocabulary.customChip")
                        : undefined
                    }
                    onChange={(e) => {
                      const v = e.target.value;
                      if (v === CUSTOM_VOCAB_SENTINEL) {
                        setAdvisorSlotCustomMode(true);
                      } else {
                        setAdvisorSlotCustomMode(false);
                        update("advisor_slot", v);
                      }
                    }}
                  >
                    {baseModAdvisorSlots.length > 0 ? (
                      <optgroup label={tCommon("vocabulary.vanillaGroup")}>
                        {ADVISOR_SLOT_OPTIONS.map((opt) => (
                          <option key={opt.value} value={opt.value}>
                            {tEnums(opt.label_i18n_key)}
                          </option>
                        ))}
                      </optgroup>
                    ) : (
                      ADVISOR_SLOT_OPTIONS.map((opt) => (
                        <option key={opt.value} value={opt.value}>
                          {tEnums(opt.label_i18n_key)}
                        </option>
                      ))
                    )}
                    {baseModAdvisorSlots.length > 0 && (
                      <optgroup label={tCommon("vocabulary.baseModGroup")}>
                        {baseModAdvisorSlots.map((slot) => (
                          <option key={slot} value={slot}>
                            {slot}
                          </option>
                        ))}
                      </optgroup>
                    )}
                    <option value={CUSTOM_VOCAB_SENTINEL}>
                      {tCommon("vocabulary.customOption")}
                    </option>
                  </Select>
                  {advisorSlotSel.freeText && (
                    <Input
                      value={form.advisor_slot || ""}
                      onChange={(e) => update("advisor_slot", e.target.value)}
                      placeholder="economy_advisor"
                      className="mt-1 font-mono text-xs"
                    />
                  )}

                  {advisorSlotSel.showUnknownNote && (
                    <p className="text-xs text-muted-foreground">
                      {tIdiom("advisorSlotNote", {
                        value: form.advisor_slot || "",
                        list: baseModVocab.advisor_slots.join(", "),
                      })}
                    </p>
                  )}
                </div>
                <div className="space-y-1">
                  <Label>{t("advisorCostLabel")}</Label>
                  <Input
                    type="number"
                    min={0}
                    value={form.advisor_cost}
                    onChange={(e) =>
                      update("advisor_cost", Number(e.target.value))
                    }
                  />
                </div>
              </div>

              <div className="space-y-1">
                <label className="flex items-center gap-2 cursor-pointer">
                  <input
                    type="checkbox"
                    checked={
                      form.always_show_on_actions_tooltip === true ||
                      (form.always_show_on_actions_tooltip == null &&
                        legacyAlwaysShowOnActions === "yes")
                    }
                    onChange={(e) =>
                      update(
                        "always_show_on_actions_tooltip",
                        e.target.checked
                          ? true
                          : typeof character.always_show_on_actions_tooltip ===
                                "boolean" || legacyAlwaysShowOnActions !== ""
                            ? false
                            : character.always_show_on_actions_tooltip,
                      )
                    }
                    className="rounded"
                  />
                  <span className="text-sm">
                    {t("alwaysShowOnActionsLabel")}
                  </span>
                </label>
                <p className="text-xs text-muted-foreground ml-6">
                  {form.always_show_on_actions_tooltip === true
                    ? t("alwaysShowOnActionsOn")
                    : form.always_show_on_actions_tooltip == null &&
                        legacyAlwaysShowOnActions !== ""
                      ? t("alwaysShowOnActionsImported")
                      : t("alwaysShowOnActionsOff")}
                </p>
              </div>

              {!isPureAdvisor && (
                <div className="space-y-2">
                  <Label>{t("advisorTraitsLabel")}</Label>
                  <Input
                    value={advisorTraitSearch}
                    onChange={(event) =>
                      setAdvisorTraitSearch(event.target.value)
                    }
                    placeholder={t("traitSearchPlaceholderRole", {
                      role: tEnums("advisor"),
                    })}
                    className="h-8 text-xs"
                  />
                  {(filteredAdvisorTraits.vanilla.length > 0 ||
                    filteredAdvisorTraits.custom.length > 0) && (
                    <div className="flex flex-wrap gap-1 max-h-28 overflow-y-auto rounded-md border border-border/40 bg-muted/20 p-2">
                      {filteredAdvisorTraits.custom.map((trait) => (
                        <Badge
                          key={`advisor-custom-${trait.trait_id}`}
                          variant={
                            (form.advisor_traits || []).includes(trait.trait_id)
                              ? "secondary"
                              : "outline"
                          }
                          className={`cursor-pointer text-xs ${(form.advisor_traits || []).includes(trait.trait_id) ? "" : "ui-chip-ghost hover:bg-muted hover:text-foreground"}`}
                          onClick={() => toggleAdvisorTrait(trait.trait_id)}
                          title={localizedName(
                            locale,
                            trait.name_en,
                            trait.name_zh,
                            t("customCountryLeaderTraitTitle"),
                          )}
                        >
                          ★ {trait.trait_id}
                        </Badge>
                      ))}
                      {filteredAdvisorTraits.vanilla.map((trait) => (
                        <Badge
                          key={`advisor-vanilla-${trait}`}
                          variant={
                            (form.advisor_traits || []).includes(trait)
                              ? "default"
                              : "outline"
                          }
                          className={`cursor-pointer text-xs ${(form.advisor_traits || []).includes(trait) ? "" : "ui-chip-ghost hover:bg-muted hover:text-foreground"}`}
                          onClick={() => toggleAdvisorTrait(trait)}
                          title={t("vanillaDbTraitTitle")}
                        >
                          {trait}
                        </Badge>
                      ))}
                    </div>
                  )}
                  {(form.advisor_traits || []).length > 0 && (
                    <div className="flex flex-wrap gap-1">
                      {(form.advisor_traits || []).map((trait) => (
                        <Badge
                          key={`advisor-selected-${trait}`}
                          variant={
                            invalidAdvisorTraits.has(trait)
                              ? "destructive"
                              : "secondary"
                          }
                          className="cursor-pointer text-xs"
                          onClick={() => toggleAdvisorTrait(trait)}
                          title={
                            invalidAdvisorTraits.has(trait)
                              ? t("invalidTraitTitle")
                              : t("deleteTraitTitle")
                          }
                        >
                          {invalidAdvisorTraits.has(trait) && "⚠ "}
                          {trait} ×
                        </Badge>
                      ))}
                    </div>
                  )}
                  {invalidAdvisorTraits.size > 0 && (
                    <p className="text-xs text-warning">
                      {t("invalidRoleTraitsWarn", {
                        count: invalidAdvisorTraits.size,
                        role: tEnums("advisor"),
                      })}
                    </p>
                  )}
                </div>
              )}

              <button
                className="flex items-center gap-1 text-xs text-muted-foreground hover:text-foreground transition-colors"
                onClick={() => setShowAdvancedAdvisor(!showAdvancedAdvisor)}
              >
                {showAdvancedAdvisor ? (
                  <ChevronDown className="h-3 w-3" />
                ) : (
                  <ChevronRight className="h-3 w-3" />
                )}
                {t("advancedAdvisorSettings")}
              </button>
              {showAdvancedAdvisor && (
                <div className="space-y-3 pl-3 border-l-2 border-muted">
                  <div className="space-y-1">
                    <div className="flex items-center gap-1">
                      <Label>{t("removalCostLabel")}</Label>
                      <InfoTooltip
                        title={t("removalCostTooltipTitle")}
                        content={t.raw("removalCostTooltipContent")}
                      />
                    </div>
                    <Input
                      type="number"
                      min={-1}
                      value={form.advisor_removal_cost ?? -1}
                      onChange={(e) =>
                        update("advisor_removal_cost", Number(e.target.value))
                      }
                      className="w-32"
                    />
                  </div>
                  <ScriptBlockEditor
                    label={t("advisorAvailableLabel")}
                    labelTooltip={t.raw("advisorAvailableTooltip")}
                    value={form.advisor_available || ""}
                    onChange={(v) => update("advisor_available", v)}
                    templates={CONDITION_TEMPLATES}
                    minHeight="min-h-12"
                  />
                  <ScriptBlockEditor
                    label={t("advisorVisibleLabel")}
                    labelTooltip={t.raw("advisorVisibleTooltip")}
                    value={form.advisor_visible || ""}
                    onChange={(v) => update("advisor_visible", v)}
                    templates={CONDITION_TEMPLATES}
                    minHeight="min-h-12"
                  />
                  <ScriptBlockEditor
                    label={t("advisorModifierLabel")}
                    labelTooltip={t("advisorModifierTooltip")}
                    value={form.advisor_modifier || ""}
                    onChange={(v) => update("advisor_modifier", v)}
                    placeholder="political_power_gain = 0.05"
                    minHeight="min-h-12"
                  />
                  <ScriptBlockEditor
                    label={t("advisorOnAddLabel")}
                    labelTooltip={t.raw("advisorOnAddTooltip")}
                    value={form.advisor_on_add || ""}
                    onChange={(v) => update("advisor_on_add", v)}
                    templates={EFFECT_TEMPLATES}
                    minHeight="min-h-12"
                  />
                  <ScriptBlockEditor
                    label={t("advisorOnRemoveLabel")}
                    labelTooltip={t("advisorOnRemoveTooltip")}
                    value={form.advisor_on_remove || ""}
                    onChange={(v) => update("advisor_on_remove", v)}
                    templates={EFFECT_TEMPLATES}
                    minHeight="min-h-12"
                  />
                </div>
              )}
            </section>
          </>
        )}

        <Separator />

        <section className="space-y-3">
          <div className="flex items-center justify-between">
            <h3 className="text-xs font-semibold text-foreground uppercase tracking-widest pl-2 border-l border-accent/40 mb-2">
              {isPureAdvisor ? t("advisorTraitsLabel") : t("traitsSection")}
            </h3>

            <Button
              variant="outline"
              size="xs"
              onClick={() => setCustomTraitsOpen(true)}
            >
              {t("manageCustomTraits")}
            </Button>
          </div>

          {isScientist ? (
            <div className="space-y-4">
              <div>
                <p className="text-xs text-muted-foreground mb-2">
                  {t("genericTraitsLabel")}
                </p>
                <div className="flex flex-wrap gap-1">
                  {SCIENTIST_GENERIC_TRAITS.map((trait) => {
                    const selected = (form.traits || []).includes(trait);
                    return (
                      <Badge
                        key={trait}
                        variant={selected ? "default" : "outline"}
                        className={`cursor-pointer text-xs ${selected ? "" : "ui-chip-ghost hover:bg-muted hover:text-foreground"}`}
                        onClick={() => toggleTrait(trait)}
                      >
                        {trait.replace("scientist_trait_", "")}
                      </Badge>
                    );
                  })}
                </div>
              </div>
              <div>
                <p className="text-xs text-muted-foreground mb-2">
                  {(() => {
                    const o = SCIENTIST_SPECIALIZATION_OPTIONS.find(
                      (x) =>
                        x.value ===
                        (form.scientist_data?.specializations?.[0]?.spec ||
                          "specialization_land"),
                    );
                    return o
                      ? tEnums(o.label_i18n_key)
                      : t("specializationFallback");
                  })()}{" "}
                  {t("specializationTraitsSuffix")}
                </p>
                <div className="flex flex-wrap gap-1">
                  {(
                    SCIENTIST_SPEC_TRAITS[
                      (form.scientist_data?.specializations?.[0]?.spec ||
                        "specialization_land") as ScientistSpecialization
                    ] || []
                  ).map((trait) => {
                    const selected = (form.traits || []).includes(trait);
                    return (
                      <Badge
                        key={trait}
                        variant={selected ? "default" : "outline"}
                        className={`cursor-pointer text-xs ${selected ? "" : "ui-chip-ghost hover:bg-muted hover:text-foreground"}`}
                        onClick={() => toggleTrait(trait)}
                      >
                        {trait.replace("scientist_trait_", "")}
                      </Badge>
                    );
                  })}
                </div>
              </div>

              {scientistCustomTraits.length > 0 && (
                <div>
                  <p className="text-xs text-muted-foreground mb-2">
                    {t("customScientistTraitsLabel")}
                  </p>
                  <div className="flex flex-wrap gap-1">
                    {scientistCustomTraits.map((ct) => {
                      const selected = (form.traits || []).includes(
                        ct.trait_id,
                      );
                      return (
                        <Badge
                          key={`custom-${ct.id}`}
                          variant={selected ? "secondary" : "outline"}
                          className={`cursor-pointer text-xs ${selected ? "" : "ui-chip-ghost hover:bg-muted hover:text-foreground"}`}
                          onClick={() => toggleTrait(ct.trait_id)}
                          title={resolveDisplayName(
                            locale,
                            ct.display_name_locs,
                            ct.name_en,
                            ct.name_zh,
                            t("customScientistTraitTitle"),
                          )}
                        >
                          ★ {ct.trait_id}
                        </Badge>
                      );
                    })}
                  </div>
                </div>
              )}
            </div>
          ) : (
            <div className="space-y-2">
              <Input
                type="text"
                value={traitSearch}
                onChange={(e) => setTraitSearch(e.target.value)}
                placeholder={t("traitSearchPlaceholderRole", {
                  role: primaryRoleLabel,
                })}
                className="h-8 text-xs"
              />
              <p className="text-xs text-muted-foreground">
                {t("roleTraitCatalogHint", { role: primaryRoleLabel })}
              </p>
              {(filteredKnownTraits.vanilla.length > 0 ||
                filteredKnownTraits.custom.length > 0) && (
                <div className="border border-border/40 rounded-md p-2 bg-muted/20 max-h-32 overflow-y-auto space-y-1">
                  {filteredKnownTraits.custom.map((trait) => {
                    const selected = selectedPrimaryTraits.includes(
                      trait.trait_id,
                    );
                    return (
                      <Badge
                        key={`role-custom-${trait.trait_id}`}
                        variant={selected ? "secondary" : "outline"}
                        className={`cursor-pointer text-xs mr-1 ${selected ? "" : "ui-chip-ghost hover:bg-muted hover:text-foreground"}`}
                        onClick={() => toggleTrait(trait.trait_id)}
                        title={localizedName(
                          locale,
                          trait.name_en,
                          trait.name_zh,
                          primaryUsesUnitCatalog
                            ? t("customUnitLeaderTraitTitle")
                            : t("customCountryLeaderTraitTitle"),
                        )}
                      >
                        ★ {trait.trait_id}
                      </Badge>
                    );
                  })}
                  {filteredKnownTraits.vanilla.map((trait) => {
                    const selected = selectedPrimaryTraits.includes(trait);
                    return (
                      <Badge
                        key={`role-vanilla-${trait}`}
                        variant={selected ? "default" : "outline"}
                        className={`cursor-pointer text-xs mr-1 ${selected ? "" : "ui-chip-ghost hover:bg-muted hover:text-foreground"}`}
                        onClick={() => toggleTrait(trait)}
                        title={t("vanillaDbTraitTitle")}
                      >
                        {trait}
                      </Badge>
                    );
                  })}
                </div>
              )}
              {traitSearch.trim() &&
                filteredKnownTraits.vanilla.length === 0 &&
                filteredKnownTraits.custom.length === 0 && (
                  <p className="text-xs text-muted-foreground">
                    {t("noMatchHint")}
                  </p>
                )}
            </div>
          )}

          {selectedPrimaryTraits.length > 0 && (
            <div className="space-y-1">
              <p className="text-xs text-muted-foreground">
                {t("selectedTraitsLabel", {
                  count: selectedPrimaryTraits.length,
                })}
              </p>
              <div className="flex flex-wrap gap-1">
                {selectedPrimaryTraits.map((trait) => {
                  const isInvalid = invalidPrimaryTraits.has(trait);
                  return (
                    <Badge
                      key={`selected-${trait}`}
                      variant={isInvalid ? "destructive" : "secondary"}
                      className="cursor-pointer text-xs"
                      onClick={() => toggleTrait(trait)}
                      title={
                        isInvalid
                          ? t("invalidTraitTitle")
                          : t("deleteTraitTitle")
                      }
                    >
                      {isInvalid && "⚠ "}
                      {trait} ×
                    </Badge>
                  );
                })}
              </div>
              {invalidPrimaryTraits.size > 0 && (
                <p className="text-xs text-warning">
                  {t("invalidRoleTraitsWarn", {
                    count: invalidPrimaryTraits.size,
                    role: primaryRoleLabel,
                  })}
                </p>
              )}
            </div>
          )}
        </section>

        <Separator />

        <PortraitUpload
          projectId={projectId}
          characterId={character.id}
          largePortrait={character.large_portrait}
          smallPortrait={character.small_portrait}
          isPortraitModified={character.is_portrait_modified}
          characterTag={character.tag}

          showAdvisor={isPureAdvisor || form.has_advisor_role}
          onUploaded={onRefresh}
        />

        <Separator />

        <div className="flex items-center gap-2 pt-2 border-t border-border">
          <Button
            onClick={handleSave}
            disabled={saving || !!baseModId.collision || tagTaken}
            size="sm"
          >
            {saving ? t("saving") : t("saveCharacter")}
          </Button>
        </div>

        {character.id ? (
          <div className="flex justify-center">
            <SaveToModFolderButton
              projectId={projectId}
              characterId={character.id}
              characterTag={character.tag}
            />
          </div>
        ) : null}

        {character.id && form.replaces_vanilla_id && onFlipBack ? (
          <div className="rounded-md border border-amber-500/40 bg-amber-500/5 p-3 space-y-2">
            <h4 className="text-sm font-semibold text-amber-700 dark:text-amber-400">
              {t("restoreVanillaTitle")}
            </h4>
            <p className="text-xs text-muted-foreground">
              {t("restoreVanillaIntro")}{" "}
              <code className="text-amber-700 dark:text-amber-400">
                {form.tag}
              </code>{" "}
              {t("restoreVanillaIntroSuffix")}
            </p>
            <ul className="text-xs text-muted-foreground list-disc ml-5 space-y-0.5">
              <li>{t("restoreVanillaItem1")}</li>
              <li>{t("restoreVanillaItem2")}</li>
              <li>{t("restoreVanillaItem3")}</li>
            </ul>
            <p className="text-xs text-red-600 dark:text-red-400">
              {t("restoreVanillaWarn")}
            </p>
            <Button
              variant="destructive"
              size="sm"
              onClick={async () => {
                await onFlipBack();
              }}
            >
              {t("restoreVanillaButton", { tag: form.tag })}
            </Button>
          </div>
        ) : null}

        <CustomTraitsPanel
          projectId={projectId}
          open={customTraitsOpen}
          onOpenChange={setCustomTraitsOpen}
          onTraitsChanged={fetchCustomTraits}
        />
      </div>
      <SourceEditorDialog
        open={sourceGate.showDialog}
        entityType="character"
        entityId={character.tag}
        entityLabel={character.name_en || character.tag}
        hideForkOption={!onForked}
        onChoice={sourceGate.handleChoice}
        onCancel={sourceGate.handleCancel}
      />
    </TooltipProvider>
  );
}
