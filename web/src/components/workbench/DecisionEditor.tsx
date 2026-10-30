"use client";

import { useState, useEffect, useCallback, useMemo, useRef } from "react";
import { Lightbulb } from "lucide-react";
import { useTranslations } from "use-intl";
import { Input } from "@/components/ui/input";
import { Select } from "@/components/ui/select";
import { Label } from "@/components/ui/label";
import { Button } from "@/components/ui/button";
import { Separator } from "@/components/ui/separator";
import { Textarea } from "@/components/ui/textarea";
import type { Decision } from "@/types";
import {
  ScriptBlockEditor,
  EFFECT_TEMPLATES,
  CONDITION_TEMPLATES,
  extractSaveErrorMessage,
  validateRequiredId,
} from "@/components/shared/editors";
import { useRegisterDirty } from "@/lib/hooks/useUnsavedGuard";
import { DisplayNameLocsTabs } from "@/components/shared/editors/DisplayNameLocsTabs";
import { DescriptionLocsTabs } from "@/components/shared/editors/DescriptionLocsTabs";
import { DEFAULT_EXPORT_LANGUAGES } from "@/components/shared/editors/displayNameLocsHelpers";
import {
  nextDraftOnEntityChange,
  stableFingerprint,
} from "@/components/shared/editors/draftReseed";
import { useApplyReseedNotice } from "@/components/shared/editors/useApplyReseedNotice";
import { useProjectStore } from "@/lib/store/projectStore";
import { IdeaReferenceDropdown } from "./IdeaReferenceDropdown";
import { EventReferenceDropdown } from "./EventReferenceDropdown";
import { FocusReferenceDropdown } from "./FocusReferenceDropdown";
import { CharacterReferenceDropdown } from "./CharacterReferenceDropdown";
import { decisionsApi } from "@/lib/api";
import { useBaseModDepKeys, gfxUrl } from "@/lib/api/basemod";
import { decisionIconSprite } from "@/app/[locale]/game-view/decisionEdit";
import { TooltipProvider } from "@/components/ui/tooltip";
import {
  ImageUpload,
  type ImageUploadResult,
} from "@/shared/upload/ImageUpload";
import { DECISION_ICON_SIZE } from "@/shared/upload/sizes";
import { useCwtoolRules } from "@/shared/editors/cwtoolStore";
import {
  useQiuqiModifierTranslations,
  useQiuqiLibraryStatus,
} from "@/shared/qiuqi/qiuqiModifierStore";
import { SourceEditorDialog } from "@/shared/dialogs/SourceEditorDialog";
import { useImportedSourceGate } from "@/shared/dialogs/useImportedSourceGate";
import {
  useBaseModIdCollision,
  BaseModIdCollisionNotice,
} from "./baseModIdCollision";

const MODIFIER_BLOCK_SKIP_KEYS = new Set([
  "factor",
  "base",
  "add",
  "tag",
  "original_tag",
  "custom_modifier_tooltip",
]);

function extractUnknownModifierKeys(
  body: string,
  knownModifiers: readonly string[] | null,
): string[] {
  if (!body || !body.trim() || !knownModifiers || knownModifiers.length === 0) {
    return [];
  }
  const knownSet = new Set(knownModifiers);
  const kvRe = /^\s*([A-Za-z_][A-Za-z_0-9]*)\s*=/;
  const seen = new Set<string>();
  const out: string[] = [];
  let depth = 0;
  for (const rawLine of body.split("\n")) {
    const line = rawLine.split("#", 1)[0];
    if (depth === 0) {
      const m = kvRe.exec(line);
      if (m) {
        const key = m[1];
        if (
          !MODIFIER_BLOCK_SKIP_KEYS.has(key) &&
          !knownSet.has(key) &&
          !seen.has(key)
        ) {
          seen.add(key);
          out.push(key);
        }
      }
    }
    let opens = 0;
    let closes = 0;
    for (const ch of line) {
      if (ch === "{") opens++;
      else if (ch === "}") closes++;
    }
    depth += opens - closes;
    if (depth < 0) depth = 0;
  }
  return out;
}

function extractAllModifierKeys(body: string): string[] {
  if (!body || !body.trim()) return [];
  const kvRe = /^\s*([A-Za-z_][A-Za-z_0-9]*)\s*=/;
  const seen = new Set<string>();
  const out: string[] = [];
  let depth = 0;
  for (const rawLine of body.split("\n")) {
    const line = rawLine.split("#", 1)[0];
    if (depth === 0) {
      const m = kvRe.exec(line);
      if (m) {
        const key = m[1].toLowerCase();
        if (!MODIFIER_BLOCK_SKIP_KEYS.has(key) && !seen.has(key)) {
          seen.add(key);
          out.push(key);
        }
      }
    }
    let opens = 0;
    let closes = 0;
    for (const ch of line) {
      if (ch === "{") opens++;
      else if (ch === "}") closes++;
    }
    depth += opens - closes;
    if (depth < 0) depth = 0;
  }
  return out;
}

interface DecisionEditorProps {
  decision: Decision;
  onSave: (data: Partial<Decision>) => Promise<void>;
  projectId: number;
  countryTag: string;
  onDraftChange?: (draft: Decision) => void;

  onIconUploaded?: (result: ImageUploadResult) => void;

  onForked?: (forked: Decision) => void;
}

export function DecisionEditor({
  decision,
  onSave,
  projectId,
  countryTag,
  onDraftChange,
  onIconUploaded,
  onForked,
}: DecisionEditorProps) {
  const t = useTranslations("decisionEditor");

  const tc = useTranslations("characterReferenceDropdown");

  const exportLanguages = useProjectStore(
    (s) => s.currentProject?.export_languages ?? DEFAULT_EXPORT_LANGUAGES,
  );
  const [form, setForm] = useState(decision);

  const baseModDepKeys = useBaseModDepKeys();
  const baseModIconFallbacks = useMemo(
    () =>
      form.icon && baseModDepKeys.length
        ? baseModDepKeys.map((dk) =>
            gfxUrl(projectId, dk, decisionIconSprite(form.icon)),
          )
        : [],
    [projectId, baseModDepKeys, form.icon],
  );

  const seedTargetedModifierText = (d: Decision): string =>
    d.targeted_modifier && Object.keys(d.targeted_modifier).length > 0
      ? JSON.stringify(d.targeted_modifier, null, 2)
      : "";
  const [targetedModifierText, setTargetedModifierText] = useState(() =>
    seedTargetedModifierText(decision),
  );
  const [targetedModifierError, setTargetedModifierError] = useState<
    string | null
  >(null);

  const baseModId = useBaseModIdCollision("decision", form.decision_id || "");
  const [saving, setSaving] = useState(false);
  const [saveStatus, setSaveStatus] = useState<"idle" | "success" | "error">(
    "idle",
  );

  const [saveError, setSaveError] = useState<string | null>(null);
  const [idError, setIdError] = useState<string | null>(null);

  const cwtoolRules = useCwtoolRules();
  const unknownModifierKeys = extractUnknownModifierKeys(
    form.modifier || "",
    cwtoolRules?.known_modifiers ?? null,
  );

  const qiuqiTranslations = useQiuqiModifierTranslations();

  const qiuqiLibrary = useQiuqiLibraryStatus();
  const modifierKeysInBody = extractAllModifierKeys(form.modifier || "");
  const qiuqiHits = modifierKeysInBody
    .map((key) => ({ key, entry: qiuqiTranslations?.[key] }))
    .filter(
      (
        item,
      ): item is {
        key: string;
        entry: { name_zh: string; desc_zh: string };
      } => !!item.entry && !!item.entry.name_zh,
    );

  const isUnsaved = JSON.stringify(form) !== JSON.stringify(decision);

  useRegisterDirty("decisions", isUnsaved);

  const decisionFingerprint = useCallback(
    (d: Decision) => stableFingerprint(d, ["icon", "updated_at"]),
    [],
  );
  const notifyReseed = useApplyReseedNotice();
  const reseedDecisionForm = useCallback((d: Decision) => {
    setForm(d);
    setTargetedModifierText(seedTargetedModifierText(d));
    setTargetedModifierError(null);
  }, []);
  const seededDecisionRef = useRef({
    id: decision.id,
    fingerprint: decisionFingerprint(decision),
  });
  useEffect(() => {
    const d = nextDraftOnEntityChange<Decision, number>({
      prevSeededId: seededDecisionRef.current.id,
      prevFingerprint: seededDecisionRef.current.fingerprint,
      entityId: decision.id,
      entity: decision,
      currentDraft: form,
      fingerprintOf: decisionFingerprint,
      syncFieldsOnSameId: ["icon"],
    });
    seededDecisionRef.current = { id: d.seededId, fingerprint: d.fingerprint };
    if (d.reseeded) {
      reseedDecisionForm(d.draft);
    } else if (d.draft !== form) {
      setForm(d.draft);
    }
    if (d.reason === "switch") setSaveStatus("idle");
    notifyReseed(d);
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [decision, decisionFingerprint, reseedDecisionForm, notifyReseed]);

  useEffect(() => {
    onDraftChange?.(form);
  }, [form, onDraftChange]);

  const update = useCallback((field: string, value: unknown) => {
    setForm((prev) => ({ ...prev, [field]: value }));
  }, []);

  const sourceGate = useImportedSourceGate<Partial<Decision>>({
    entitySource: decision.source,
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
        setSaveError(extractSaveErrorMessage(err));
        setSaveStatus("error");
      } finally {
        setSaving(false);
      }
    },
    performFork: onForked
      ? async (newDecisionId) => {
          setSaving(true);
          setSaveStatus("idle");
          setSaveError(null);
          try {
            const forked = await decisionsApi.fork(
              projectId,
              decision.id,
              newDecisionId,
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
    resetForm: () => {
      setForm(decision);
      setTargetedModifierText(seedTargetedModifierText(decision));
      setTargetedModifierError(null);
    },
  });

  const handleSave = async () => {
    const trimmedId = (form.decision_id || "").trim();

    const idValidation = validateRequiredId(trimmedId, "Decision ID");
    if (idValidation) {
      setIdError(idValidation);
      setSaveError(idValidation);
      setSaveStatus("error");
      return;
    }
    setIdError(null);

    if (await baseModId.verify(trimmedId)) return;
    const payload: Partial<Decision> = {
      ...form,
      decision_id: trimmedId,
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
      setSaveError(extractSaveErrorMessage(err));
      setSaveStatus("error");
    } finally {
      setSaving(false);
    }
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
                  ? t("saved")
                  : saveStatus === "error"
                    ? (saveError ?? t("saveFailed"))
                    : isUnsaved
                      ? t("unsavedChanges")
                      : ""}
            </span>
          </div>
        )}

        <section className="space-y-3">
          <h3 className="text-sm font-semibold text-muted-foreground uppercase tracking-wide">
            {t("basicInfoSection")}
          </h3>
          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-1">
              <Label>{t("decisionIdLabel")}</Label>
              <Input
                value={form.decision_id}
                onChange={(e) => {
                  if (idError) setIdError(null);
                  update("decision_id", e.target.value);
                }}
                aria-invalid={idError || baseModId.collision ? true : undefined}
                className={`font-mono text-xs ${idError || baseModId.collision ? "border-destructive" : ""}`}
                placeholder="e.g. USA_homeland_defense"
              />
              {idError && <p className="text-xs text-destructive">{idError}</p>}
              <BaseModIdCollisionNotice
                entityType="decision"
                collision={baseModId.collision}
              />
            </div>
            <div className="space-y-1">
              <Label>{t("iconLabel")}</Label>

              <Input
                value={form.icon}
                onChange={(e) => update("icon", e.target.value)}
                className="font-mono text-xs"
                placeholder="generic_prepare_civil_war (or upload below)"
              />
            </div>
          </div>

          <ImageUpload
            label={t("decisionIconLabel")}
            targetSize={DECISION_ICON_SIZE}
            previewUrl={
              form.icon ? decisionsApi.iconUrl(projectId, decision.id) : null
            }
            baseModFallbackUrls={baseModIconFallbacks}
            gfxName={form.icon || undefined}
            hint="PNG / JPG / DDS · 33 × 30 · auto-converted to DDS at export · GFX name auto-generated"
            onUpload={async (file, crop) =>
              decisionsApi.uploadIcon(projectId, decision.id, file, crop)
            }
            onUploaded={(result) => {
              const icon = result["icon"];
              if (typeof icon === "string") {
                setForm((f) => ({ ...f, icon }));
              }
              onIconUploaded?.(result);
            }}
          />
        </section>

        <Separator />

        <section className="space-y-3">
          <h3 className="text-sm font-semibold text-muted-foreground uppercase tracking-wide">
            {t("costTimingSection")}
          </h3>
          <div className="grid grid-cols-3 gap-3">
            <div className="space-y-1">
              <Label>{t("costPpLabel")}</Label>
              <Input
                type="number"
                min={0}
                value={form.cost}
                onChange={(e) => update("cost", Number(e.target.value))}
              />
            </div>
            <div className="space-y-1">
              <Label>{t("durationDaysLabel")}</Label>
              <Input
                type="number"
                min={0}
                value={form.days_remove}
                onChange={(e) => update("days_remove", Number(e.target.value))}
                placeholder="0 = instant"
              />
            </div>
            <div className="space-y-1">
              <Label>{t("cooldownDaysLabel")}</Label>
              <Input
                type="number"
                min={0}
                value={form.days_re_enable}
                onChange={(e) =>
                  update("days_re_enable", Number(e.target.value))
                }
                placeholder="0 = no cooldown"
              />
            </div>
          </div>
          <label className="flex items-center gap-2 text-sm cursor-pointer">
            <input
              type="checkbox"
              checked={form.fire_only_once}
              onChange={(e) => update("fire_only_once", e.target.checked)}
              className="accent-accent h-4 w-4"
            />
            {t("fireOnlyOnceLabel")}
          </label>

          <label className="flex items-center gap-2 text-sm cursor-pointer">
            <input
              type="checkbox"
              checked={form.cancel_if_not_visible ?? false}
              onChange={(e) =>
                update("cancel_if_not_visible", e.target.checked)
              }
              className="accent-accent h-4 w-4"
            />
            {t("cancelIfNotVisibleLabel")}
          </label>
        </section>

        <Separator />

        <section className="space-y-4">
          <h3 className="text-sm font-semibold text-muted-foreground uppercase tracking-wide">
            {t("conditionsSection")}
          </h3>

          <ScriptBlockEditor
            label={t("allowedLabel")}
            value={form.allowed}
            onChange={(v) => update("allowed", v)}
            placeholder="original_tag = USA"
            minHeight="min-h-12"
          />

          <ScriptBlockEditor
            label={t("availableLabel")}
            value={form.available}
            onChange={(v) => update("available", v)}
            templates={CONDITION_TEMPLATES}
          >
            {(insertCode) => (
              <>
                <FocusReferenceDropdown
                  projectId={projectId}
                  effectType="has_completed_focus"
                  buttonLabel={t("completedFocus")}
                  onSelect={insertCode}
                />
                <IdeaReferenceDropdown
                  projectId={projectId}
                  countryTag={countryTag}
                  effectType="has_idea"
                  buttonLabel={t("hasIdea")}
                  onSelect={insertCode}
                />
              </>
            )}
          </ScriptBlockEditor>

          <ScriptBlockEditor
            label={t("visibleLabel")}
            value={form.visible}
            onChange={(v) => update("visible", v)}
            templates={CONDITION_TEMPLATES}
          >
            {(insertCode) => (
              <>
                <p className="text-xs text-warning mb-2 px-2 py-1.5 rounded bg-warning/5 border border-warning/20 leading-relaxed">
                  <Lightbulb
                    className="inline h-3.5 w-3.5 mr-1 align-[-3px]"
                    aria-hidden
                  />
                  {t("flagChainHintLead")}{" "}
                  <code className="font-mono bg-background/40 px-1 rounded">
                    has_country_flag = X
                  </code>{" "}
                  {t("flagChainHintPrefix")}{" "}
                  <code className="font-mono bg-background/40 px-1 rounded">
                    set_country_flag = X
                  </code>{" "}
                  {t("flagChainHintSuffix")}
                </p>
                <FocusReferenceDropdown
                  projectId={projectId}
                  effectType="has_completed_focus"
                  buttonLabel={t("completedFocus")}
                  onSelect={insertCode}
                />
              </>
            )}
          </ScriptBlockEditor>
        </section>

        <Separator />

        <section className="space-y-4">
          <h3 className="text-sm font-semibold text-muted-foreground uppercase tracking-wide">
            {t("effectsSection")}
          </h3>

          <ScriptBlockEditor
            label={t("completeEffectLabel")}
            value={form.complete_effect}
            onChange={(v) => update("complete_effect", v)}
            templates={EFFECT_TEMPLATES}
            minHeight="min-h-24"
          >
            {(insertCode) => (
              <>
                <EventReferenceDropdown
                  projectId={projectId}
                  effectType="country_event"
                  buttonLabel={t("triggerEvent")}
                  onSelect={insertCode}
                />
                <IdeaReferenceDropdown
                  projectId={projectId}
                  countryTag={countryTag}
                  effectType="add_ideas"
                  buttonLabel={t("addIdea")}
                  onSelect={insertCode}
                />
                <CharacterReferenceDropdown
                  projectId={projectId}
                  effectType="promote_character"
                  buttonLabel={tc("promote")}
                  onSelect={insertCode}
                />
                <CharacterReferenceDropdown
                  projectId={projectId}
                  effectType="recruit_character"
                  buttonLabel={tc("recruit")}
                  onSelect={insertCode}
                />
                <CharacterReferenceDropdown
                  projectId={projectId}
                  effectType="retire_character"
                  buttonLabel={tc("retire")}
                  onSelect={insertCode}
                />
              </>
            )}
          </ScriptBlockEditor>

          <div className="space-y-1">
            <Label>{t("effectTooltipLabel")}</Label>
            <DescriptionLocsTabs
              entityKey={`decision:${decision.id}:effect-tooltip`}
              value={form.effect_tooltip_locs || {}}
              exportLanguages={exportLanguages}
              onChange={(next) => update("effect_tooltip_locs", next)}
              ariaLabel="Decision effect tooltip"
              rows={2}
            />
            <p className="text-xs text-muted-foreground">
              {t("effectTooltipHint")}
            </p>
          </div>

          {form.days_remove > 0 && (
            <>
              <ScriptBlockEditor
                label={t("removeEffectLabel")}
                value={form.remove_effect}
                onChange={(v) => update("remove_effect", v)}
                templates={EFFECT_TEMPLATES}
              >
                {(insertCode) => (
                  <EventReferenceDropdown
                    projectId={projectId}
                    effectType="country_event"
                    buttonLabel={t("triggerEvent")}
                    onSelect={insertCode}
                  />
                )}
              </ScriptBlockEditor>

              <ScriptBlockEditor
                label={t("modifierLabel")}
                value={form.modifier}
                onChange={(v) => update("modifier", v)}
                placeholder="war_support_weekly = 0.005"
                minHeight="min-h-12"
              />

              {unknownModifierKeys.length > 0 && (
                <div
                  className="inline-flex items-center gap-1 rounded-md border border-accent bg-accent/15 px-2 py-1 text-xs text-foreground"
                  title={t("cwtoolTip")}
                >
                  <span className="font-medium">
                    cwtool β unknown modifier:
                  </span>
                  <span className="font-mono">
                    {unknownModifierKeys.join(", ")}
                  </span>
                </div>
              )}

              {qiuqiHits.length > 0 && (
                <div className="flex flex-wrap gap-1">
                  {qiuqiHits.map(({ key, entry }) => (
                    <span
                      key={key}
                      className="inline-flex items-center gap-1 rounded-md border border-border bg-muted/40 px-2 py-1 text-xs text-foreground"
                      title={entry.desc_zh || entry.name_zh}
                    >
                      <span className="font-medium">{entry.name_zh}</span>
                      <span className="font-mono text-muted-foreground">
                        ({key})
                      </span>
                    </span>
                  ))}
                </div>
              )}

              {qiuqiLibrary.empty && modifierKeysInBody.length > 0 && (
                <p className="text-xs text-muted-foreground">
                  {t("qiuqiLibraryNotLoaded")}
                </p>
              )}

              <ScriptBlockEditor
                label={t("cancelTriggerLabel")}
                value={form.cancel_trigger}
                onChange={(v) => update("cancel_trigger", v)}
                templates={CONDITION_TEMPLATES}
                minHeight="min-h-12"
              />

              <ScriptBlockEditor
                label={t("cancelEffectLabel")}
                value={form.cancel_effect}
                onChange={(v) => update("cancel_effect", v)}
                templates={EFFECT_TEMPLATES}
                minHeight="min-h-12"
              />
            </>
          )}

          <ScriptBlockEditor
            label="ai_will_do"
            value={form.ai_will_do}
            onChange={(v) => update("ai_will_do", v)}
            placeholder="factor = 1"
            minHeight="min-h-12"
          />
        </section>

        <Separator />

        <details className="group rounded-md border border-border bg-card/50">
          <summary className="cursor-pointer select-none px-3 py-2 text-sm font-semibold text-muted-foreground uppercase tracking-wide hover:bg-card/80 rounded-md">
            {t("missionFlowSection")}
            <span className="ml-2 text-xs font-normal normal-case text-muted-foreground">
              {t("missionModeHint")}
            </span>
          </summary>
          <div className="space-y-4 p-3 border-t border-border">
            <div className="grid grid-cols-3 gap-3">
              <div className="space-y-1">
                <Label>days_mission_timeout</Label>
                <Input
                  type="number"
                  min={0}
                  value={form.days_mission_timeout || 0}
                  onChange={(e) =>
                    update("days_mission_timeout", Number(e.target.value))
                  }
                  placeholder="0 = no timeout"
                />
                <p className="text-xs text-muted-foreground">
                  {t("timeoutCountdownHint")}
                </p>
              </div>
              <label className="flex items-center gap-2 text-sm cursor-pointer mt-5">
                <input
                  type="checkbox"
                  checked={form.selectable_mission || false}
                  onChange={(e) =>
                    update("selectable_mission", e.target.checked)
                  }
                  className="accent-accent h-4 w-4"
                />
                selectable_mission = no
              </label>
              <label className="flex items-center gap-2 text-sm cursor-pointer mt-5">
                <input
                  type="checkbox"
                  checked={form.is_good || false}
                  onChange={(e) => update("is_good", e.target.checked)}
                  className="accent-accent h-4 w-4"
                />
                {t("isGoodLabel")}
              </label>
            </div>

            <ScriptBlockEditor
              label={t("activationLabel")}
              value={form.activation || ""}
              onChange={(v) => update("activation", v)}
              templates={CONDITION_TEMPLATES}
              placeholder="has_country_flag = chapter_2"
              minHeight="min-h-12"
            />

            <ScriptBlockEditor
              label={t("timeoutEffectLabel")}
              value={form.timeout_effect || ""}
              onChange={(v) => update("timeout_effect", v)}
              templates={EFFECT_TEMPLATES}
              placeholder="add_stability = -0.1"
              minHeight="min-h-12"
            />
          </div>
        </details>

        <details className="group rounded-md border border-border bg-card/50">
          <summary className="cursor-pointer select-none px-3 py-2 text-sm font-semibold text-muted-foreground uppercase tracking-wide hover:bg-card/80 rounded-md">
            {t("targetingCustomCostAdvanced")}
            <span className="ml-2 text-xs font-normal normal-case text-muted-foreground">
              (targeted decision / state target / custom cost / priority /
              war_with_*)
            </span>
          </summary>
          <div className="space-y-4 p-3 border-t border-border">
            <div className="grid grid-cols-3 gap-3">
              <div className="space-y-1">
                <Label>state_target</Label>
                <Input
                  value={form.state_target || ""}
                  onChange={(e) => update("state_target", e.target.value)}
                  placeholder="state id (e.g. 123 or var:foo)"
                  className="font-mono text-xs"
                />
                <p className="text-xs text-muted-foreground">
                  {t("stateTargetHint")}
                </p>
              </div>
              <div className="space-y-1">
                <Label>priority</Label>
                <Input
                  type="number"
                  min={0}
                  max={100}
                  value={form.priority || 0}
                  onChange={(e) => update("priority", Number(e.target.value))}
                  placeholder="0-100"
                />
                <p className="text-xs text-muted-foreground">
                  {t("priorityHint")}
                </p>
              </div>
              <div className="space-y-1">
                <Label>custom_cost_text</Label>
                <Input
                  value={form.custom_cost_text || ""}
                  onChange={(e) => update("custom_cost_text", e.target.value)}
                  placeholder="COST_LABEL_KEY"
                  className="font-mono text-xs"
                />
                <p className="text-xs text-muted-foreground">
                  {t("customCostTextHint")}
                </p>
              </div>
            </div>

            <div className="grid grid-cols-2 gap-3">
              <div className="space-y-1">
                <Label>war_with_on_add</Label>
                <Input
                  value={form.war_with_on_add || ""}
                  onChange={(e) =>
                    update("war_with_on_add", e.target.value.toUpperCase())
                  }
                  placeholder="3-letter TAG"
                  maxLength={10}
                  className="font-mono text-xs"
                />
              </div>
              <div className="space-y-1">
                <Label>war_with_on_remove</Label>
                <Input
                  value={form.war_with_on_remove || ""}
                  onChange={(e) =>
                    update("war_with_on_remove", e.target.value.toUpperCase())
                  }
                  placeholder="3-letter TAG"
                  maxLength={10}
                  className="font-mono text-xs"
                />
              </div>
            </div>

            <div className="grid grid-cols-2 gap-3">
              <div className="space-y-1">
                <Label>{t("targetsLabel")}</Label>
                <Input
                  value={(form.targets || []).join(", ")}
                  onChange={(e) =>
                    update(
                      "targets",
                      e.target.value
                        .split(",")
                        .map((s) => s.trim())
                        .filter((s) => s.length > 0),
                    )
                  }
                  placeholder="CHI, JAP, GER"
                  className="font-mono text-xs"
                />
              </div>
              <div className="space-y-1">
                <Label>target_array (scripted-var array)</Label>
                <Input
                  value={(form.target_array || []).join(", ")}
                  onChange={(e) =>
                    update(
                      "target_array",
                      e.target.value
                        .split(",")
                        .map((s) => s.trim())
                        .filter((s) => s.length > 0),
                    )
                  }
                  placeholder="@modern_usa_targets"
                  className="font-mono text-xs"
                />
              </div>
            </div>

            <div className="grid grid-cols-2 gap-3">
              <div className="space-y-1">
                <Label>{t("highlightStatesLabel")}</Label>
                <Input
                  value={(form.highlight_states || []).join(", ")}
                  onChange={(e) =>
                    update(
                      "highlight_states",
                      e.target.value
                        .split(",")
                        .map((s) => s.trim())
                        .filter((s) => s.length > 0),
                    )
                  }
                  placeholder="123, 321, 1008"
                  className="font-mono text-xs"
                />
                <p className="text-xs text-muted-foreground">
                  {t("highlightStatesHint")}
                </p>
              </div>
              <div className="space-y-1">
                <Label>on_map_mode</Label>
                <Select
                  className="px-2 text-xs font-mono"
                  value={form.on_map_mode || ""}
                  onChange={(e) => update("on_map_mode", e.target.value)}
                >
                  <option value="">{t("onMapModeOptDefault")}</option>
                  <option value="map_only">map_only</option>
                  <option value="map_and_decisions_view">
                    map_and_decisions_view
                  </option>
                  <option value="decision_view_only">decision_view_only</option>
                </Select>
                <p className="text-xs text-muted-foreground">
                  {t("onMapModeHint")}
                </p>
              </div>
            </div>

            <div className="space-y-1">
              <Label>
                targeted_modifier (JSON · {`{tag, modifier:{key:value}}`})
              </Label>
              <Textarea
                value={targetedModifierText}
                onChange={(e) => {
                  const text = e.target.value;
                  setTargetedModifierText(text);
                  const raw = text.trim();
                  if (!raw) {
                    setTargetedModifierError(null);
                    update("targeted_modifier", {});
                    return;
                  }
                  try {
                    update("targeted_modifier", JSON.parse(raw));
                    setTargetedModifierError(null);
                  } catch (err) {
                    setTargetedModifierError(
                      err instanceof Error ? err.message : String(err),
                    );
                  }
                }}
                placeholder='{"tag": "CHI", "modifier": {"trade_opinion_factor": -0.5}}'
                className={`font-mono text-xs min-h-20 ${
                  targetedModifierError
                    ? "border-warning focus-visible:ring-warning/40"
                    : ""
                }`}
              />
              {targetedModifierError && (
                <p className="text-xs text-warning">
                  {t("targetedModifierParseError", {
                    error: targetedModifierError,
                  })}
                </p>
              )}
            </div>

            <ScriptBlockEditor
              label={t("targetRootTriggerLabel")}
              value={form.target_root_trigger || ""}
              onChange={(v) => update("target_root_trigger", v)}
              templates={CONDITION_TEMPLATES}
              placeholder="tag = USA"
              minHeight="min-h-12"
            />
            <ScriptBlockEditor
              label={t("targetTriggerLabel")}
              value={form.target_trigger || ""}
              onChange={(v) => update("target_trigger", v)}
              templates={CONDITION_TEMPLATES}
              placeholder="NOT = { has_war_with = ROOT }"
              minHeight="min-h-12"
            />
            <ScriptBlockEditor
              label={t("customCostTriggerLabel")}
              value={form.custom_cost_trigger || ""}
              onChange={(v) => update("custom_cost_trigger", v)}
              templates={CONDITION_TEMPLATES}
              placeholder="has_political_power > 100"
              minHeight="min-h-12"
            />
          </div>
        </details>

        <Separator />

        <section className="space-y-3">
          <h3 className="text-sm font-semibold text-muted-foreground uppercase tracking-wide">
            {t("localizationSection")}
          </h3>

          <div className="space-y-1">
            <Label>{t("nameLabel")}</Label>
            <DisplayNameLocsTabs
              entityKey={`decision:${decision.id}:display-name`}
              value={form.display_name_locs || {}}
              exportLanguages={exportLanguages}
              legacyFallback={{ en: form.name_en, zh: form.name_zh }}
              onChange={(next) => update("display_name_locs", next)}
              placeholder={t("namePlaceholder")}
              ariaLabel="Decision name"
            />
            <p className="text-xs text-muted-foreground">{t("nameHint")}</p>
          </div>

          <div className="space-y-1">
            <Label>{t("descriptionLabel")}</Label>
            <DescriptionLocsTabs
              entityKey={`decision:${decision.id}:description`}
              value={form.description_locs || {}}
              exportLanguages={exportLanguages}
              legacyFallback={{ en: form.desc_en, zh: form.desc_zh }}
              onChange={(next) => update("description_locs", next)}
              ariaLabel="Decision description"
              rows={3}
            />
          </div>
        </section>

        <Separator />

        <Button
          onClick={handleSave}
          disabled={saving || !!baseModId.collision}
          className="w-full"
        >
          {t("saveButton")}
        </Button>
      </div>
      <SourceEditorDialog
        open={sourceGate.showDialog}
        entityType="decision"
        entityId={decision.decision_id}
        entityLabel={decision.name_en || decision.decision_id}
        hideForkOption={!onForked}
        onChoice={sourceGate.handleChoice}
        onCancel={sourceGate.handleCancel}
      />
    </TooltipProvider>
  );
}
