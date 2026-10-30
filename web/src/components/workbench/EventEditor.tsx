"use client";

import React, {
  useState,
  useEffect,
  useCallback,
  useMemo,
  useRef,
} from "react";
import { useTranslations } from "use-intl";
import { ChevronDown, ChevronRight } from "lucide-react";
import { Input } from "@/components/ui/input";
import { Select } from "@/components/ui/select";
import { Label } from "@/components/ui/label";
import { Button } from "@/components/ui/button";
import { Separator } from "@/components/ui/separator";
import type { GameEvent, EventOption } from "@/types";
import { EVENT_TYPE_OPTIONS, EVENT_ON_ACTION_HOOKS } from "@/types";
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
import { Hoi4TextMarkupHint } from "@/components/shared/Hoi4TextMarkupHint";
import { DEFAULT_EXPORT_LANGUAGES } from "@/components/shared/editors/displayNameLocsHelpers";
import {
  nextDraftOnEntityChange,
  stableFingerprint,
} from "@/components/shared/editors/draftReseed";
import { useApplyReseedNotice } from "@/components/shared/editors/useApplyReseedNotice";
import { useProjectStore } from "@/lib/store/projectStore";
import { IdeaReferenceDropdown } from "./IdeaReferenceDropdown";
import { EventReferenceDropdown } from "./EventReferenceDropdown";
import { DecisionReferenceDropdown } from "./DecisionReferenceDropdown";
import { FocusReferenceDropdown } from "./FocusReferenceDropdown";
import { CharacterReferenceDropdown } from "./CharacterReferenceDropdown";
import { eventsApi } from "@/lib/api";
import { useBaseModDepKeys, gfxUrl } from "@/lib/api/basemod";
import { TooltipProvider } from "@/components/ui/tooltip";
import {
  ImageUpload,
  type ImageUploadResult,
} from "@/shared/upload/ImageUpload";
import {
  EVENT_PICTURE_SIZE,
  NEWS_EVENT_PICTURE_SIZE,
} from "@/shared/upload/sizes";
import { SourceEditorDialog } from "@/shared/dialogs/SourceEditorDialog";
import { useImportedSourceGate } from "@/shared/dialogs/useImportedSourceGate";
import type { EventPreviewTarget } from "@/components/workbench/EventInGamePreview";
import { useContentLocaleStore } from "@/lib/store/contentLocaleStore";
import { resolveDisplayName } from "@/lib/i18n/localizedName";

function emptyOption(): EventOption {
  return {
    name: "",
    display_name_locs: {},
    effect_tooltip_locs: {},
    trigger: "",
    effects: "",
    hidden_effects: "",
    ai_chance: {},
  };
}

interface OptionEditorProps {
  entityKey: string;
  option: EventOption;
  index: number;
  onChange: (index: number, option: EventOption) => void;
  onRemove: (index: number) => void;
  projectId: number;
  countryTag: string;
  exportLanguages: string[];

  rowRef?: (node: HTMLDivElement | null) => void;

  highlighted?: boolean;

  expandSignal?: number;
}

const OptionEditor = React.memo(function OptionEditor({
  entityKey,
  option,
  index,
  onChange,
  onRemove,
  projectId,
  countryTag,
  exportLanguages,
  rowRef,
  highlighted = false,
  expandSignal = 0,
}: OptionEditorProps) {
  const t = useTranslations("eventEditor");

  const tc = useTranslations("characterReferenceDropdown");
  const contentLocale = useContentLocaleStore((s) => s.locale);
  const [expansion, setExpansion] = useState({
    expanded: true,
    signal: expandSignal,
  });

  if (expansion.signal !== expandSignal) {
    setExpansion({
      expanded: expandSignal > 0 ? true : expansion.expanded,
      signal: expandSignal,
    });
  }
  const expanded = expansion.expanded;

  const update = (field: keyof EventOption, value: unknown) => {
    onChange(index, { ...option, [field]: value });
  };

  const extra = option as unknown as Record<string, unknown>;
  const optStr = (key: string): string =>
    typeof extra[key] === "string" ? (extra[key] as string) : "";
  const optLocs = (key: "display_name_locs" | "effect_tooltip_locs") => {
    const value = extra[key];
    return value && typeof value === "object" && !Array.isArray(value)
      ? (value as Record<string, string>)
      : {};
  };
  const updateLocs = (
    key: "display_name_locs" | "effect_tooltip_locs",
    value: Record<string, string>,
  ) => {
    onChange(index, { ...option, [key]: value } as EventOption);
  };
  const optionDisplayName = resolveDisplayName(
    contentLocale,
    optLocs("display_name_locs"),
    optStr("name_en"),
    optStr("name_zh"),
    option.name,
  );

  return (
    <div
      ref={rowRef}
      className={`border rounded-md overflow-hidden scroll-mt-16 transition-[box-shadow,background-color] duration-200 ${
        highlighted ? "ring-2 ring-accent/70 bg-accent/5" : ""
      }`}
    >
      <div
        className="flex items-center gap-2 px-3 py-2 bg-muted/50 cursor-pointer min-w-0"
        onClick={() =>
          setExpansion({ expanded: !expanded, signal: expandSignal })
        }
      >
        <span className="text-muted-foreground w-6 shrink-0 flex items-center">
          {expanded ? (
            <ChevronDown className="h-3 w-3" />
          ) : (
            <ChevronRight className="h-3 w-3" />
          )}
        </span>
        <span className="text-sm font-medium flex-1 truncate">
          Option {index + 1}
          {optionDisplayName
            ? `: ${optionDisplayName}`
            : option.name
              ? `: ${option.name}`
              : ""}
        </span>
        <Button
          size="sm"
          variant="ghost"
          className="h-6 w-6 p-0 text-xs text-destructive hover:text-destructive/80"
          onClick={(e) => {
            e.stopPropagation();
            onRemove(index);
          }}
        >
          ×
        </Button>
      </div>

      {expanded && (
        <div className="p-3 space-y-3">
          <div className="space-y-1">
            <Label>{t("optNameLabel")}</Label>
            <Input
              value={option.name}
              onChange={(e) => update("name", e.target.value)}
              className="font-mono text-xs"
              placeholder="e.g. france.1.a"
            />
          </div>

          <div className="space-y-1">
            <div className="flex items-center gap-1">
              <Label>{t("optDisplayTextLabel")}</Label>
              <Hoi4TextMarkupHint />
            </div>
            <DisplayNameLocsTabs
              entityKey={`${entityKey}:display-name`}
              value={optLocs("display_name_locs")}
              exportLanguages={exportLanguages}
              onChange={(next) => updateLocs("display_name_locs", next)}
              legacyFallback={{ en: optStr("name_en"), zh: optStr("name_zh") }}
              placeholder={t("optDisplayTextEnPlaceholder")}
              ariaLabel={t("optDisplayTextEnAria")}
            />
            <p className="text-micro text-muted-foreground">
              {t("optDisplayTextHint")}
            </p>
          </div>

          <ScriptBlockEditor
            label={t("optTriggerLabel")}
            value={option.trigger}
            onChange={(v) => update("trigger", v)}
            templates={CONDITION_TEMPLATES}
            minHeight="min-h-12"
          />

          <ScriptBlockEditor
            label={t("optEffectsLabel")}
            value={option.effects}
            onChange={(v) => update("effects", v)}
            templates={EFFECT_TEMPLATES}
            minHeight="min-h-16"
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
            <Label>{t("optEffectTooltipLabel")}</Label>
            <DisplayNameLocsTabs
              entityKey={`${entityKey}:effect-tooltip`}
              value={optLocs("effect_tooltip_locs")}
              exportLanguages={exportLanguages}
              onChange={(next) => updateLocs("effect_tooltip_locs", next)}
              legacyFallback={{
                en: optStr("effect_tooltip_en"),
                zh: optStr("effect_tooltip_zh"),
              }}
              inputComponent="textarea"
              placeholder={t("optEffectTooltipEnPlaceholder")}
              ariaLabel={t("optEffectTooltipEnAria")}
            />
            <p className="text-micro text-muted-foreground">
              {t("optEffectTooltipHint")}
            </p>
          </div>

          <ScriptBlockEditor
            label={t("optHiddenEffectLabel")}
            value={option.hidden_effects}
            onChange={(v) => update("hidden_effects", v)}
            templates={EFFECT_TEMPLATES}
            minHeight="min-h-12"
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

          <div className="space-y-1">
            <Label>{t("optAiChanceLabel")}</Label>
            <Input
              type="number"
              value={option.ai_chance?.base ?? ""}
              onChange={(e) =>
                update("ai_chance", {
                  ...option.ai_chance,
                  base: e.target.value ? Number(e.target.value) : undefined,
                })
              }
              className="w-32"
              placeholder="base = 50"
            />
          </div>
        </div>
      )}
    </div>
  );
});

interface EventEditorProps {
  event: GameEvent;
  onSave: (data: Partial<GameEvent>) => Promise<void>;
  projectId: number;
  countryTag: string;

  onPictureUploaded?: (result: ImageUploadResult) => void;

  onForked?: (forked: GameEvent) => void;
  onDraftChange?: (draft: GameEvent) => void;
  focusTarget?: EventPreviewTarget | null;

  focusOptionIndex?: number | null;
  onFocusTargetConsumed?: () => void;
}

export function EventEditor({
  event,
  onSave,
  projectId,
  countryTag,
  onPictureUploaded,
  onForked,
  onDraftChange,
  focusTarget,
  focusOptionIndex,
  onFocusTargetConsumed,
}: EventEditorProps) {
  const t = useTranslations("eventEditor");

  const tc = useTranslations("characterReferenceDropdown");

  const tEnums = useTranslations("enums");

  const exportLanguages = useProjectStore(
    (s) => s.currentProject?.export_languages ?? DEFAULT_EXPORT_LANGUAGES,
  );
  const [form, setForm] = useState(event);
  const optionIdentityRef = useRef<{
    eventId: number;
    next: number;
    keys: string[];
  }>({ eventId: event.id, next: 0, keys: [] });
  if (optionIdentityRef.current.eventId !== event.id) {
    optionIdentityRef.current = { eventId: event.id, next: 0, keys: [] };
  }
  while (optionIdentityRef.current.keys.length < (form.options || []).length) {
    optionIdentityRef.current.keys.push(
      `event:${event.id}:option:${optionIdentityRef.current.next++}`,
    );
  }

  const baseModDepKeys = useBaseModDepKeys();
  const baseModPictureFallbacks = useMemo(
    () =>
      form.picture && baseModDepKeys.length
        ? baseModDepKeys.map((dk) => gfxUrl(projectId, dk, form.picture))
        : [],
    [projectId, baseModDepKeys, form.picture],
  );
  const [saving, setSaving] = useState(false);
  const [saveStatus, setSaveStatus] = useState<"idle" | "success" | "error">(
    "idle",
  );

  const [saveError, setSaveError] = useState<string | null>(null);

  const [idError, setIdError] = useState<string | null>(null);

  const [lintWarnings, setLintWarnings] = useState<
    { code: string; message: string; severity: "warning" | "error" | "info" }[]
  >([]);
  const pictureSectionRef = useRef<HTMLElement | null>(null);
  const optionsSectionRef = useRef<HTMLElement | null>(null);
  const titleSectionRef = useRef<HTMLDivElement | null>(null);
  const descriptionSectionRef = useRef<HTMLDivElement | null>(null);
  const [highlightTarget, setHighlightTarget] =
    useState<EventPreviewTarget | null>(null);

  const optionRowRefs = useRef<Map<number, HTMLDivElement>>(new Map());

  const optionRowRefSetters = useRef<
    Map<number, (node: HTMLDivElement | null) => void>
  >(new Map());
  const getOptionRowRefSetter = useCallback((index: number) => {
    const cached = optionRowRefSetters.current.get(index);
    if (cached) return cached;
    const setter = (node: HTMLDivElement | null) => {
      if (node) optionRowRefs.current.set(index, node);
      else optionRowRefs.current.delete(index);
    };
    optionRowRefSetters.current.set(index, setter);
    return setter;
  }, []);
  const [highlightOptionIndex, setHighlightOptionIndex] = useState<
    number | null
  >(null);
  const [optionExpandSignal, setOptionExpandSignal] = useState<{
    index: number;
    nonce: number;
  } | null>(null);

  const isUnsaved = useMemo(
    () => JSON.stringify(form) !== JSON.stringify(event),
    [form, event],
  );

  useRegisterDirty("events", isUnsaved);

  const eventFingerprint = useCallback(
    (e: GameEvent) => stableFingerprint(e, ["picture", "updated_at"]),
    [],
  );
  const notifyReseed = useApplyReseedNotice();
  const seededEventRef = useRef({
    id: event.id,
    fingerprint: eventFingerprint(event),
  });
  useEffect(() => {
    const decision = nextDraftOnEntityChange<GameEvent, number>({
      prevSeededId: seededEventRef.current.id,
      prevFingerprint: seededEventRef.current.fingerprint,
      entityId: event.id,
      entity: event,
      currentDraft: form,
      fingerprintOf: eventFingerprint,
      syncFieldsOnSameId: ["picture"],
    });
    seededEventRef.current = {
      id: decision.seededId,
      fingerprint: decision.fingerprint,
    };
    if (decision.reseeded || decision.draft !== form) {
      setForm(decision.draft);
    }
    if (decision.reason === "switch") setSaveStatus("idle");
    notifyReseed(decision);
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [event, eventFingerprint, notifyReseed]);

  useEffect(() => {
    onDraftChange?.(form);
  }, [form, onDraftChange]);

  useEffect(() => {
    if (!focusTarget) return;

    if (focusTarget === "options" && typeof focusOptionIndex === "number") {
      setOptionExpandSignal({ index: focusOptionIndex, nonce: Date.now() });

      const raf = window.requestAnimationFrame(() => {
        const row = optionRowRefs.current.get(focusOptionIndex);
        row?.scrollIntoView({ block: "center", behavior: "smooth" });
      });
      setHighlightOptionIndex(focusOptionIndex);
      const timer = window.setTimeout(
        () => setHighlightOptionIndex(null),
        1400,
      );
      onFocusTargetConsumed?.();
      return () => {
        window.cancelAnimationFrame(raf);
        window.clearTimeout(timer);
      };
    }
    const node =
      focusTarget === "picture"
        ? pictureSectionRef.current
        : focusTarget === "options"
          ? optionsSectionRef.current
          : focusTarget === "title"
            ? titleSectionRef.current
            : descriptionSectionRef.current;
    node?.scrollIntoView({ block: "center", behavior: "smooth" });
    setHighlightTarget(focusTarget);
    const timer = window.setTimeout(() => setHighlightTarget(null), 1400);
    onFocusTargetConsumed?.();
    return () => window.clearTimeout(timer);
  }, [focusTarget, focusOptionIndex, onFocusTargetConsumed]);

  const targetSectionClass = (base: string, target: EventPreviewTarget) =>
    `${base} scroll-mt-16 rounded-md transition-[box-shadow,background-color] duration-200 ${
      highlightTarget === target ? "ring-2 ring-accent/70 bg-accent/5" : ""
    }`;

  useEffect(() => {
    let cancelled = false;
    eventsApi
      .lint(projectId, event.id)
      .then((r) => {
        if (!cancelled) setLintWarnings(r.warnings ?? []);
      })
      .catch(() => {
        if (!cancelled) setLintWarnings([]);
      });
    return () => {
      cancelled = true;
    };
  }, [projectId, event.id, event.event_type, event.trigger, event.immediate]);

  const update = useCallback((field: string, value: unknown) => {
    setForm((prev) => ({ ...prev, [field]: value }));
  }, []);

  const sourceGate = useImportedSourceGate<Partial<GameEvent>>({
    entitySource: event.source,
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
      ? async (newEventId) => {
          setSaving(true);
          setSaveStatus("idle");
          setSaveError(null);
          try {
            const forked = await eventsApi.fork(
              projectId,
              event.id,
              newEventId,
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
    resetForm: () => setForm(event),
  });

  const handleSave = async () => {
    const trimmedId = (form.event_id || "").trim();

    const idValidation = validateRequiredId(trimmedId, "Event ID");
    if (idValidation) {
      setIdError(idValidation);
      setSaveError(idValidation);
      setSaveStatus("error");
      return;
    }
    setIdError(null);
    const payload: Partial<GameEvent> = {
      ...form,
      event_id: trimmedId,
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

  const updateOption = useCallback((index: number, option: EventOption) => {
    setForm((prev) => {
      const opts = [...(prev.options || [])];
      opts[index] = option;
      return { ...prev, options: opts };
    });
  }, []);

  const addOption = useCallback(() => {
    setForm((prev) => {
      const opts = [...(prev.options || [])];
      const ns = prev.event_id.split(".")[0] || "event";
      const num = prev.event_number || 1;
      const letter = String.fromCharCode(97 + opts.length);
      const newOpt = emptyOption();
      newOpt.name = `${ns}.${num}.${letter}`;
      opts.push(newOpt);
      return { ...prev, options: opts };
    });
  }, []);

  const removeOption = useCallback((index: number) => {
    optionIdentityRef.current.keys.splice(index, 1);
    setForm((prev) => {
      const opts = [...(prev.options || [])];
      opts.splice(index, 1);
      return { ...prev, options: opts };
    });
  }, []);

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

        {lintWarnings.length > 0 && (
          <div
            role="status"
            className="rounded-md border border-warning/40 bg-warning/10 px-3 py-2 text-xs text-warning space-y-1"
          >
            {lintWarnings.map((w, i) => (
              <div key={`${w.code}-${i}`} className="flex items-start gap-2">
                <span aria-hidden="true">⚠</span>
                <span className="flex-1">{w.message}</span>
              </div>
            ))}
          </div>
        )}

        <section
          ref={pictureSectionRef}
          className={targetSectionClass("space-y-3", "picture")}
        >
          <h3 className="text-sm font-semibold text-muted-foreground uppercase tracking-wide">
            {t("basicInfoSection")}
          </h3>
          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-1">
              <Label>{t("eventIdLabel")}</Label>
              <Input
                value={form.event_id}
                onChange={(e) => {
                  if (idError) setIdError(null);
                  update("event_id", e.target.value);
                }}
                aria-invalid={idError ? true : undefined}
                className={`font-mono text-xs ${idError ? "border-destructive" : ""}`}
                placeholder={t("eventIdPlaceholder")}
              />
              {idError && <p className="text-xs text-destructive">{idError}</p>}
            </div>
            <div className="space-y-1">
              <Label>{t("eventNumberLabel")}</Label>
              <Input
                type="number"
                value={form.event_number}
                onChange={(e) => update("event_number", Number(e.target.value))}
              />
            </div>
          </div>
          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-1">
              <Label>{t("eventTypeLabel")}</Label>
              <Select
                value={form.event_type}
                onChange={(e) => update("event_type", e.target.value)}
              >
                {EVENT_TYPE_OPTIONS.map((opt) => (
                  <option key={opt.value} value={opt.value}>
                    {tEnums(opt.label_i18n_key)}
                  </option>
                ))}
              </Select>
            </div>
            <div className="space-y-1">
              <Label>{t("pictureGfxLabel")}</Label>

              <Input
                value={form.picture}
                onChange={(e) => update("picture", e.target.value)}
                className="font-mono text-xs"
                placeholder={t("pictureGfxPlaceholder")}
              />
            </div>
          </div>

          <ImageUpload
            label={t("eventPictureUploadLabel")}
            targetSize={
              form.event_type === "news_event"
                ? NEWS_EVENT_PICTURE_SIZE
                : EVENT_PICTURE_SIZE
            }
            previewUrl={
              form.picture ? eventsApi.pictureUrl(projectId, event.id) : null
            }
            baseModFallbackUrls={baseModPictureFallbacks}
            gfxName={form.picture || undefined}
            hint={t("eventPictureUploadHint")}
            onUpload={async (file, crop) =>
              eventsApi.uploadPicture(projectId, event.id, file, crop)
            }
            onUploaded={(result) => {
              if (result.picture) {
                setForm((f) => ({ ...f, picture: result.picture as string }));
              }
              onPictureUploaded?.(result);
            }}
          />
        </section>

        <Separator />

        <section className="space-y-3">
          <h3 className="text-sm font-semibold text-muted-foreground uppercase tracking-wide">
            Flags
          </h3>
          <div className="flex flex-wrap gap-4">
            {(
              [
                ["is_triggered_only", "Is Triggered Only"],
                ["fire_only_once", "Fire Only Once"],
                ["hidden", "Hidden"],
                ["major", "Major (news)"],
              ] as const
            ).map(([field, label]) => (
              <label
                key={field}
                className="flex items-center gap-2 text-sm cursor-pointer"
              >
                <input
                  type="checkbox"
                  checked={Boolean(form[field])}
                  onChange={(e) => update(field, e.target.checked)}
                  className="accent-accent h-4 w-4"
                />
                {label}
              </label>
            ))}
          </div>
          {form.timeout_days !== undefined && (
            <div className="space-y-1">
              <Label>{t("timeoutDaysLabel")}</Label>
              <Input
                type="number"
                min={0}
                value={form.timeout_days}
                onChange={(e) => update("timeout_days", Number(e.target.value))}
                className="w-32"
              />
            </div>
          )}

          <div className="space-y-2 pt-2 border-t border-border/40">
            <label className="flex items-center gap-2 text-sm cursor-pointer">
              <input
                type="checkbox"
                checked={Boolean(form.auto_fire_on_startup)}
                onChange={(e) =>
                  update("auto_fire_on_startup", e.target.checked)
                }
                className="accent-accent h-4 w-4"
              />
              <span>{t("autoFireStartup")}</span>
            </label>
            {form.auto_fire_on_startup && (
              <div className="ml-6 space-y-1">
                <Label>{t("autoFireAfterDaysLabel")}</Label>
                <Input
                  type="number"
                  min={0}
                  value={form.auto_fire_after_days ?? 0}
                  onChange={(e) =>
                    update("auto_fire_after_days", Number(e.target.value))
                  }
                  className="w-32"
                />
              </div>
            )}
          </div>

          <div className="space-y-1 pt-2 border-t border-border/40">
            <Label>{t("onActionAttachLabel")}</Label>
            <Select
              value={form.on_action_attach ?? ""}
              onChange={(e) => update("on_action_attach", e.target.value)}
            >
              {EVENT_ON_ACTION_HOOKS.map((h) => (
                <option key={h.value || "_blank"} value={h.value}>
                  {tEnums(h.label_i18n_key)}
                </option>
              ))}
            </Select>
            <p className="text-xs text-muted-foreground leading-tight">
              {t("onActionHint")}
            </p>
          </div>
        </section>

        <Separator />

        <section className="space-y-4">
          <h3 className="text-sm font-semibold text-muted-foreground uppercase tracking-wide">
            {t("scriptBlocksSection")}
          </h3>

          <ScriptBlockEditor
            label={t("triggerLabel")}
            value={form.trigger}
            onChange={(v) => update("trigger", v)}
            placeholder="tag = FRA"
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

          {!form.is_triggered_only && (
            <ScriptBlockEditor
              label={t("mtthLabel")}
              value={form.mean_time_to_happen}
              onChange={(v) => update("mean_time_to_happen", v)}
              placeholder="days = 365"
              minHeight="min-h-12"
            />
          )}

          <ScriptBlockEditor
            label={t("immediateLabel")}
            value={form.immediate}
            onChange={(v) => update("immediate", v)}
            templates={EFFECT_TEMPLATES}
          >
            {(insertCode) => (
              <>
                <IdeaReferenceDropdown
                  projectId={projectId}
                  countryTag={countryTag}
                  effectType="add_ideas"
                  buttonLabel={t("addIdea")}
                  onSelect={insertCode}
                />
                <EventReferenceDropdown
                  projectId={projectId}
                  effectType="country_event"
                  buttonLabel={t("triggerEvent")}
                  onSelect={insertCode}
                />
                <DecisionReferenceDropdown
                  projectId={projectId}
                  effectType="activate_decision"
                  buttonLabel={t("activateDecision")}
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
        </section>

        <Separator />

        <section
          ref={optionsSectionRef}
          className={targetSectionClass("space-y-3", "options")}
        >
          <div className="flex items-center justify-between">
            <h3 className="text-sm font-semibold text-muted-foreground uppercase tracking-wide">
              {t("optionsSection")} ({(form.options || []).length})
            </h3>
            <Button size="sm" variant="outline" onClick={addOption}>
              + {t("addOption")}
            </Button>
          </div>

          {(form.options || []).length === 0 && (
            <p className="text-xs text-muted-foreground italic">
              {t("optionsEmpty")}
            </p>
          )}

          <div className="space-y-2">
            {(form.options || []).map((opt, i) => (
              <OptionEditor
                key={optionIdentityRef.current.keys[i]}
                entityKey={optionIdentityRef.current.keys[i]}
                option={opt}
                index={i}
                onChange={updateOption}
                onRemove={removeOption}
                projectId={projectId}
                countryTag={countryTag}
                exportLanguages={exportLanguages}
                rowRef={getOptionRowRefSetter(i)}
                highlighted={highlightOptionIndex === i}
                expandSignal={
                  optionExpandSignal?.index === i ? optionExpandSignal.nonce : 0
                }
              />
            ))}
          </div>
        </section>

        <Separator />

        <section className="space-y-4">
          <ScriptBlockEditor
            label={t("afterLabel")}
            value={form.after}
            onChange={(v) => update("after", v)}
            templates={EFFECT_TEMPLATES}
            minHeight="min-h-12"
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
        </section>

        <Separator />

        <section className="space-y-3">
          <div className="flex items-center gap-1.5">
            <h3 className="text-sm font-semibold text-muted-foreground uppercase tracking-wide">
              Localization
            </h3>

            <Hoi4TextMarkupHint />
          </div>

          <div
            ref={titleSectionRef}
            className={targetSectionClass("space-y-1", "title")}
          >
            <Label>{t("titleLabel")}</Label>
            <DisplayNameLocsTabs
              entityKey={`event:${event.id}:display-name`}
              value={form.display_name_locs || {}}
              exportLanguages={exportLanguages}
              legacyFallback={{ en: form.title_en, zh: form.title_zh }}
              onChange={(next) => update("display_name_locs", next)}
              placeholder={t("titlePlaceholder")}
              ariaLabel={t("titleAria")}
            />
          </div>

          <div
            ref={descriptionSectionRef}
            className={targetSectionClass("space-y-1", "description")}
          >
            <Label>{t("descriptionLabel")}</Label>
            <DescriptionLocsTabs
              entityKey={`event:${event.id}:description`}
              value={form.description_locs || {}}
              exportLanguages={exportLanguages}
              legacyFallback={{ en: form.desc_en, zh: form.desc_zh }}
              onChange={(next) => update("description_locs", next)}
              placeholder={t("descPlaceholder")}
              ariaLabel={t("descriptionAria")}
              rows={3}
            />
          </div>
        </section>

        <Separator />

        <Button onClick={handleSave} disabled={saving} className="w-full">
          {t("saveEvent")}
        </Button>
      </div>
      <SourceEditorDialog
        open={sourceGate.showDialog}
        entityType="event"
        entityId={event.event_id}
        entityLabel={event.title_en || event.event_id}
        hideForkOption={!onForked}
        onChoice={sourceGate.handleChoice}
        onCancel={sourceGate.handleCancel}
      />
    </TooltipProvider>
  );
}
