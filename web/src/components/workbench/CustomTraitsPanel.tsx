"use client";

import { useState, useEffect, useCallback, useMemo } from "react";
import { useTranslations } from "use-intl";
import { useLocaleStore } from "@/lib/store/localeStore";
import { useContentLocaleStore } from "@/lib/store/contentLocaleStore";
import { localizedOptionLabel } from "@/lib/i18n/optionLabel";
import { resolveDisplayName } from "@/lib/i18n/localizedName";
import { toast } from "sonner";
import { useConfirm } from "@/components/shared/ConfirmDialogProvider";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { Select } from "@/components/ui/select";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import type { CustomTrait, CustomTraitType } from "@/types";
import {
  CUSTOM_TRAIT_TYPE_OPTIONS,
  UNIT_LEADER_TRAIT_TYPE_OPTIONS,
  UNIT_LEADER_SKILL_KEYS,
  UNIT_LEADER_SKILL_FACTOR_KEYS,
  UNIT_LEADER_ADVANCED_INT_FIELDS,
  UNIT_LEADER_ADVANCED_STRING_FIELDS,
  UNIT_LEADER_ADVANCED_BOOL_FIELDS,
  UNIT_LEADER_ADVANCED_SCRIPT_FIELDS,
  UNIT_LEADER_ADVANCED_MODIFIER_FIELDS,
  SCIENTIST_TRAIT_SPECIALIZATION_OPTIONS,
} from "@/types";
import { API_BASE, customTraitsApi } from "@/lib/api";
import { ImageUpload } from "@/shared/upload/ImageUpload";
import { useActiveCountry } from "@/lib/store/projectStore";
import { TraitTreeGraph, type TraitTreePatch } from "./trait-graph";
import { OpinionModifiersPanel } from "./OpinionModifiersPanel";
import {
  customTraitSupportsTargetedModifier,
  findTargetedModifierKeyCollision,
  normalizeTargetedModifiers,
  targetedModifierKeyConflicts,
  type TargetedModifierEntry,
} from "./targetedModifierForm";

const TRAIT_ICON_SIZE = { width: 23, height: 33 } as const;

const PERCENT_MODIFIER_RE = /(_factor|_mult|_multiplier)$/;

function formatModifierValue(key: string, value: number | string): string {
  if (typeof value !== "number" || !Number.isFinite(value)) {
    return String(value);
  }
  const sign = value > 0 ? "+" : "";
  if (PERCENT_MODIFIER_RE.test(key)) {
    const pct = value * 100;
    const pctStr = Number.isInteger(pct)
      ? String(pct)
      : String(parseFloat(pct.toFixed(2)));
    return `${sign}${pctStr}%`;
  }
  const numStr = Number.isInteger(value)
    ? String(value)
    : String(parseFloat(value.toFixed(3)));
  return `${sign}${numStr}`;
}

function modifierTone(value: number | string): string {
  if (typeof value !== "number" || !Number.isFinite(value) || value === 0) {
    return "text-ingame-text/85";
  }
  return value > 0 ? "text-emerald-300" : "text-rose-300";
}

interface CustomTraitPreviewProps {
  name: string;
  desc: string;
  modifier: Record<string, number | string>;
  typeLabel: string;
}

const MODIFIER_RENDER_CAP = 40;

function CustomTraitPreview({
  name,
  desc,
  modifier,
  typeLabel,
}: CustomTraitPreviewProps) {
  const t = useTranslations("customTraitsPanel");
  const entries = Object.entries(modifier).filter(
    ([k]) => k && !k.startsWith("__"),
  );
  const shown = entries.slice(0, MODIFIER_RENDER_CAP);
  const hiddenCount = entries.length - shown.length;

  return (
    <div className="rounded-md border border-ingame-edge/60 bg-gradient-to-b from-ingame-panel/85 to-ingame-panel-deep/95 text-ingame-title shadow-sm overflow-hidden">
      <div className="flex items-center gap-2 px-3 py-1.5 bg-ingame-band/80 border-b border-ingame-edge/40">
        <span className="text-micro font-mono uppercase tracking-wide text-ingame-gold/90 truncate">
          {typeLabel}
        </span>
        <span className="ml-auto text-micro font-mono text-ingame-text/70">
          {t("previewApprox")}
        </span>
      </div>

      <div className="px-3 py-3 space-y-2">
        <p className="text-sm font-semibold text-ingame-title leading-snug">
          {name}
        </p>

        {desc && (
          <p className="text-xs text-ingame-text/90 whitespace-pre-wrap leading-snug">
            {desc}
          </p>
        )}

        {shown.length > 0 ? (
          <ul className="space-y-0.5 pt-1">
            {shown.map(([key, value]) => (
              <li
                key={key}
                className="flex items-baseline justify-between gap-3 text-xs font-mono"
              >
                <span className="text-ingame-text/80 truncate">{key}</span>
                <span className={`shrink-0 ${modifierTone(value)}`}>
                  {formatModifierValue(key, value)}
                </span>
              </li>
            ))}
            {hiddenCount > 0 && (
              <li className="text-micro font-mono text-ingame-text/60 pt-0.5">
                {t("previewMoreModifiers", { count: hiddenCount })}
              </li>
            )}
          </ul>
        ) : (
          <p className="text-micro font-mono text-ingame-text/60 italic pt-1">
            {t("previewNoModifiers")}
          </p>
        )}
      </div>
    </div>
  );
}

interface CustomTraitsPanelProps {
  projectId: number;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onTraitsChanged?: () => void;
}

type Draft = Omit<
  CustomTrait,
  "id" | "project_id" | "raw_extras" | "country_tag"
>;

const EMPTY_DRAFT: Draft = {
  trait_type: "country_leader",
  trait_id: "",
  name_en: "",
  name_zh: "",
  desc_en: "",
  desc_zh: "",

  display_name_locs: {},
  description_locs: {},
  modifier: {},
  targeted_modifier: [],
  ai_will_do: "",
  random: true,
  extra_fields: {},
};

export function CustomTraitsPanel({
  projectId,
  open,
  onOpenChange,
  onTraitsChanged,
}: CustomTraitsPanelProps) {
  const t = useTranslations("customTraitsPanel");
  const tContent = useTranslations("decisions");
  const confirmDialog = useConfirm();

  const tEnums = useTranslations("enums");
  const appLocale = useLocaleStore((s) => s.locale);
  const contentLocale = useContentLocaleStore((s) => s.locale);
  const countryTag = useActiveCountry();
  const [activeType, setActiveType] =
    useState<CustomTraitType>("country_leader");
  const [traits, setTraits] = useState<CustomTrait[]>([]);
  const [selectedId, setSelectedId] = useState<number | null>(null);
  const [draft, setDraft] = useState<Draft>(EMPTY_DRAFT);
  const [modifierText, setModifierText] = useState("{}");
  const [modifierError, setModifierError] = useState<string | null>(null);
  const [targetedModifierError, setTargetedModifierError] = useState<
    string | null
  >(null);
  const [saving, setSaving] = useState(false);
  const [status, setStatus] = useState<string | null>(null);
  const [opinionModifiersOpen, setOpinionModifiersOpen] = useState(false);

  const [iconUploaded, setIconUploaded] = useState(false);

  const [viewMode, setViewMode] = useState<"form" | "tree">("form");

  const fetchTraits = useCallback(async () => {
    const res = await fetch(
      `${API_BASE}/go/projects/${projectId}/custom-traits?trait_type=${activeType}`,
    );
    if (!res.ok) return;
    const data: CustomTrait[] = await res.json();
    setTraits(data);
  }, [projectId, activeType]);

  const persistTraitPatch = useCallback(
    async (dbId: number, patch: TraitTreePatch) => {
      const res = await fetch(
        `${API_BASE}/go/projects/${projectId}/custom-traits/${dbId}`,
        {
          method: "PUT",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify(patch),
        },
      );
      if (!res.ok) {
        const err = await res.json().catch(() => ({ detail: "Save failed" }));
        throw new Error(err.detail || "Save failed");
      }
      await fetchTraits();
      onTraitsChanged?.();
    },
    [projectId, fetchTraits, onTraitsChanged],
  );

  const visibleTraits = useMemo(() => {
    if (!countryTag) return traits;
    return traits.filter((t) => !t.country_tag || t.country_tag === countryTag);
  }, [traits, countryTag]);

  useEffect(() => {
    if (open) fetchTraits();
  }, [open, fetchTraits]);

  const selectTrait = (trait: CustomTrait) => {
    const targetedModifier = trait.targeted_modifier ?? [];
    const collision = findTargetedModifierKeyCollision(targetedModifier);
    setSelectedId(trait.id);
    setDraft({
      trait_type: trait.trait_type,
      trait_id: trait.trait_id,
      name_en: trait.name_en,
      name_zh: trait.name_zh,
      desc_en: trait.desc_en,
      desc_zh: trait.desc_zh,

      display_name_locs: trait.display_name_locs ?? {},
      description_locs: trait.description_locs ?? {},
      modifier: trait.modifier,

      targeted_modifier: targetedModifier,
      ai_will_do: trait.ai_will_do,
      random: trait.random,
      extra_fields: trait.extra_fields,
    });
    setModifierText(JSON.stringify(trait.modifier, null, 2));
    setModifierError(null);
    setTargetedModifierError(
      collision
        ? t("targetedModifierDuplicateKey", { key: collision.key })
        : null,
    );
    setStatus(null);
    setIconUploaded(false);
  };

  const startNew = useCallback(() => {
    setSelectedId(null);
    setDraft({ ...EMPTY_DRAFT, trait_type: activeType });
    setModifierText("{}");
    setModifierError(null);
    setTargetedModifierError(null);
    setStatus(null);
    setIconUploaded(false);
  }, [activeType]);

  useEffect(() => {
    if (open) {
      startNew();
    }
  }, [activeType, open, startNew]);

  useEffect(() => {
    if (activeType !== "unit_leader") setViewMode("form");
  }, [activeType]);

  const updateField = <K extends keyof Draft>(field: K, value: Draft[K]) => {
    setDraft((prev) => ({ ...prev, [field]: value }));
  };

  const replaceTargetedModifiers = (entries: TargetedModifierEntry[]) => {
    const collision = findTargetedModifierKeyCollision(entries);
    setDraft((previous) => ({ ...previous, targeted_modifier: entries }));
    setTargetedModifierError(
      collision
        ? t("targetedModifierDuplicateKey", { key: collision.key })
        : null,
    );
  };

  const updateTargetedModifier = (
    targetIndex: number,
    updater: (entry: TargetedModifierEntry) => TargetedModifierEntry,
  ) => {
    replaceTargetedModifiers(
      draft.targeted_modifier.map((entry, index) =>
        index === targetIndex ? updater(entry) : entry,
      ),
    );
  };

  const updateTargetedModifierKey = (
    targetIndex: number,
    modifierIndex: number,
    nextKey: string,
  ) => {
    const editedEntry = draft.targeted_modifier[targetIndex];
    if (
      editedEntry &&
      targetedModifierKeyConflicts(editedEntry, modifierIndex, nextKey)
    ) {
      setTargetedModifierError(
        t("targetedModifierDuplicateKey", { key: nextKey.trim() }),
      );
      return;
    }
    const next = draft.targeted_modifier.map((entry, index) =>
      index === targetIndex
        ? {
            ...entry,
            modifier: Object.fromEntries(
              Object.entries(entry.modifier).map(([key, value], index) =>
                index === modifierIndex ? [nextKey, value] : [key, value],
              ),
            ),
          }
        : entry,
    );
    const collision = findTargetedModifierKeyCollision(next);
    if (collision) {
      setTargetedModifierError(
        t("targetedModifierDuplicateKey", { key: collision.key }),
      );
      return;
    }
    replaceTargetedModifiers(next);
  };

  const updateTargetedModifierValue = (
    targetIndex: number,
    modifierIndex: number,
    nextValue: string,
  ) => {
    updateTargetedModifier(targetIndex, (entry) => ({
      ...entry,
      modifier: Object.fromEntries(
        Object.entries(entry.modifier).map(([key, value], index) =>
          index === modifierIndex ? [key, nextValue] : [key, value],
        ),
      ),
    }));
  };

  const updateExtra = (key: string, value: unknown) => {
    setDraft((prev) => ({
      ...prev,
      extra_fields: { ...prev.extra_fields, [key]: value },
    }));
  };

  const readExtraString = (key: string, fallback: string): string => {
    const v = draft.extra_fields?.[key];
    return typeof v === "string" ? v : fallback;
  };

  const readExtraNumber = (key: string): number => {
    const v = draft.extra_fields?.[key];
    return typeof v === "number" ? v : 0;
  };

  const readExtraBool = (key: string): boolean => {
    const v = draft.extra_fields?.[key];
    return typeof v === "boolean" ? v : false;
  };

  const readExtraFloat = (key: string): string => {
    const v = draft.extra_fields?.[key];
    if (typeof v === "number") return String(v);
    return "";
  };

  const readExtraModifier = (key: string): string => {
    const v = draft.extra_fields?.[key];
    if (v && typeof v === "object" && !Array.isArray(v)) {
      return JSON.stringify(v, null, 2);
    }
    return "";
  };

  const updateExtraModifier = (key: string, jsonText: string) => {
    const trimmed = jsonText.trim();
    if (!trimmed) {
      setDraft((prev) => {
        const next = { ...prev.extra_fields };
        delete next[key];
        return { ...prev, extra_fields: next };
      });
      return;
    }
    try {
      const parsed = JSON.parse(trimmed);
      if (
        typeof parsed !== "object" ||
        parsed === null ||
        Array.isArray(parsed)
      ) {
        return;
      }
      updateExtra(key, parsed);
    } catch {}
  };

  const [advancedOpen, setAdvancedOpen] = useState(false);

  const advancedExtrasCount = useMemo(() => {
    if (activeType !== "unit_leader") return 0;
    const ALL_ADVANCED_KEYS = [
      ...UNIT_LEADER_SKILL_FACTOR_KEYS,
      ...UNIT_LEADER_ADVANCED_INT_FIELDS.map((f) => f.key),
      ...UNIT_LEADER_ADVANCED_STRING_FIELDS.map((f) => f.key),
      ...UNIT_LEADER_ADVANCED_BOOL_FIELDS.map((f) => f.key),
      ...UNIT_LEADER_ADVANCED_SCRIPT_FIELDS.map((f) => f.key),
      ...UNIT_LEADER_ADVANCED_MODIFIER_FIELDS.map((f) => f.key),
    ];
    let n = 0;
    for (const k of ALL_ADVANCED_KEYS) {
      const v = draft.extra_fields?.[k];
      if (v === undefined || v === null) continue;
      if (typeof v === "string" && !v.trim()) continue;
      if (
        typeof v === "number" &&
        v === 0 &&
        !["gui_row", "gui_column"].includes(k)
      )
        continue;
      if (typeof v === "object" && Object.keys(v as object).length === 0)
        continue;
      n += 1;
    }
    return n;
  }, [draft.extra_fields, activeType]);

  const previewName = resolveDisplayName(
    contentLocale,
    draft.display_name_locs,
    draft.name_en,
    draft.name_zh,
    draft.trait_id || t("previewUnnamed"),
  );
  const previewDesc = resolveDisplayName(
    contentLocale,
    draft.description_locs,
    draft.desc_en,
    draft.desc_zh,
    "",
  );

  const previewModifier = useMemo<Record<string, number | string>>(() => {
    try {
      const parsed = JSON.parse(modifierText);
      if (parsed && typeof parsed === "object" && !Array.isArray(parsed)) {
        return parsed as Record<string, number | string>;
      }
    } catch {}
    return draft.modifier ?? {};
  }, [modifierText, draft.modifier]);
  const previewTypeLabel = useMemo(() => {
    const opt = CUSTOM_TRAIT_TYPE_OPTIONS.find((o) => o.value === activeType);
    return opt ? tEnums(opt.label_i18n_key) : activeType;
  }, [activeType, tEnums]);

  const parseModifier = (): Record<string, number | string> | null => {
    try {
      const parsed = JSON.parse(modifierText);
      if (
        typeof parsed !== "object" ||
        parsed === null ||
        Array.isArray(parsed)
      ) {
        throw new Error("Modifier must be a JSON object");
      }
      setModifierError(null);
      return parsed;
    } catch (e) {
      setModifierError(e instanceof Error ? e.message : String(e));
      return null;
    }
  };

  const save = async () => {
    if (targetedModifierError) return;
    const modifier = parseModifier();
    if (modifier === null) return;
    const targetedCollision = findTargetedModifierKeyCollision(
      draft.targeted_modifier,
    );
    if (targetedCollision) {
      setTargetedModifierError(
        t("targetedModifierDuplicateKey", { key: targetedCollision.key }),
      );
      return;
    }

    const trimmedTraitId = (draft.trait_id || "").trim();
    if (!trimmedTraitId) {
      setStatus("trait_id is required");
      return;
    }
    setSaving(true);
    setStatus(null);
    try {
      const isNew = selectedId === null;
      const targetedModifier = normalizeTargetedModifiers(
        draft.targeted_modifier,
      );
      const body = isNew
        ? {
            ...draft,
            trait_id: trimmedTraitId,
            trait_type: activeType,
            modifier,
            targeted_modifier: targetedModifier,
            country_tag: countryTag,
          }
        : {
            ...draft,
            trait_id: trimmedTraitId,
            trait_type: activeType,
            modifier,
            targeted_modifier: targetedModifier,
          };
      const url = isNew
        ? `${API_BASE}/go/projects/${projectId}/custom-traits`
        : `${API_BASE}/go/projects/${projectId}/custom-traits/${selectedId}`;
      const res = await fetch(url, {
        method: isNew ? "POST" : "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      });
      if (!res.ok) {
        const err = await res.json().catch(() => ({ detail: "Save failed" }));
        throw new Error(err.detail || "Save failed");
      }
      const saved: CustomTrait = await res.json();
      setStatus(isNew ? "Created" : "Saved");
      await fetchTraits();
      setSelectedId(saved.id);

      const advancedN = activeType === "unit_leader" ? advancedExtrasCount : 0;
      toast.success(
        isNew
          ? `Trait "${saved.trait_id}" created${advancedN ? ` · ${advancedN} advanced fields` : ""}`
          : `Trait "${saved.trait_id}" saved${advancedN ? ` · ${advancedN} advanced fields` : ""}`,
      );
      onTraitsChanged?.();
    } catch (e) {
      const msg = e instanceof Error ? e.message : String(e);
      setStatus(msg);
      toast.error(t("saveFailed", { msg }));
    } finally {
      setSaving(false);
    }
  };

  const deleteTrait = async () => {
    if (selectedId === null) return;
    if (
      !(await confirmDialog({
        body: t("confirmDelete", { traitId: draft.trait_id }),
        danger: true,
      }))
    )
      return;
    setSaving(true);
    try {
      const res = await fetch(
        `${API_BASE}/go/projects/${projectId}/custom-traits/${selectedId}`,
        { method: "DELETE" },
      );
      if (!res.ok) throw new Error("Delete failed");
      await fetchTraits();
      startNew();
      onTraitsChanged?.();
    } catch (e) {
      setStatus(e instanceof Error ? e.message : String(e));
    } finally {
      setSaving(false);
    }
  };

  return (
    <>
      <Dialog open={open} onOpenChange={onOpenChange} disablePointerDismissal>
        <DialogContent className="max-w-5xl w-[95vw] sm:max-w-5xl p-0 gap-0">
          <DialogHeader className="p-4 pb-3 border-b">
            <div className="flex items-center justify-between gap-3">
              <DialogTitle>{t("title")}</DialogTitle>
              <Button
                type="button"
                size="sm"
                variant="outline"
                onClick={() => setOpinionModifiersOpen(true)}
              >
                {t("openOpinionModifiers")}
              </Button>
            </div>
            <p className="text-xs text-muted-foreground">
              {t("opinionVsTargetedHelp")}
            </p>
          </DialogHeader>

          <div className="flex gap-1 px-4 pt-3 border-b">
            {CUSTOM_TRAIT_TYPE_OPTIONS.map((opt) => (
              <button
                key={opt.value}
                disabled={!opt.enabled}
                onClick={() => opt.enabled && setActiveType(opt.value)}
                className={`px-3 py-2 text-xs rounded-t-md border-b-2 transition-colors ${
                  activeType === opt.value && opt.enabled
                    ? "border-primary text-foreground font-medium"
                    : "border-transparent text-muted-foreground hover:text-foreground"
                } ${!opt.enabled ? "opacity-50 cursor-not-allowed" : "cursor-pointer"}`}
              >
                {tEnums(opt.label_i18n_key)}
              </button>
            ))}

            {activeType === "unit_leader" && (
              <div className="ml-auto flex items-center gap-0.5 self-center rounded-md border p-0.5">
                {(["form", "tree"] as const).map((mode) => (
                  <button
                    key={mode}
                    onClick={() => setViewMode(mode)}
                    className={`px-2 py-1 text-xs rounded transition-colors cursor-pointer ${
                      viewMode === mode
                        ? "bg-muted text-foreground font-medium"
                        : "text-muted-foreground hover:text-foreground"
                    }`}
                  >
                    {mode === "form" ? t("viewForm") : t("viewTree")}
                  </button>
                ))}
              </div>
            )}
          </div>

          <div className="grid grid-cols-[260px_1fr] h-[70vh]">
            <aside className="border-r overflow-y-auto">
              <div className="p-3 border-b">
                <Button size="sm" className="w-full" onClick={startNew}>
                  {t("newTrait")}
                </Button>
              </div>
              <ul className="divide-y">
                {visibleTraits.length === 0 && (
                  <li className="p-3 text-xs text-muted-foreground">
                    {traits.length === 0
                      ? t("emptyTraits")
                      : t("emptyCountryTraits", {
                          country: countryTag || t("thisCountry"),
                        })}
                  </li>
                )}
                {visibleTraits.map((trait) => {
                  let extraN = 0;
                  if (
                    trait.trait_type === "unit_leader" &&
                    trait.extra_fields
                  ) {
                    for (const [k, v] of Object.entries(trait.extra_fields)) {
                      if (
                        UNIT_LEADER_SKILL_KEYS.includes(k as never) ||
                        k === "type"
                      )
                        continue;
                      if (v === undefined || v === null) continue;
                      if (typeof v === "string" && !v.trim()) continue;
                      if (
                        typeof v === "number" &&
                        v === 0 &&
                        !["gui_row", "gui_column"].includes(k)
                      )
                        continue;
                      if (
                        typeof v === "object" &&
                        Object.keys(v as object).length === 0
                      )
                        continue;
                      extraN += 1;
                    }
                  }
                  return (
                    <li
                      key={trait.id}
                      onClick={() => selectTrait(trait)}
                      className={`p-3 cursor-pointer text-sm hover:bg-muted/50 ${
                        selectedId === trait.id ? "bg-muted" : ""
                      }`}
                    >
                      <div className="flex items-center justify-between gap-2">
                        <span className="font-mono text-xs truncate">
                          {trait.trait_id}
                        </span>
                        {extraN > 0 && (
                          <span className="shrink-0 rounded-full bg-accent/15 text-accent border border-accent/30 px-1.5 py-0.5 text-micro">
                            {t("advancedBadge", { count: extraN })}
                          </span>
                        )}
                      </div>
                      <div className="text-xs text-muted-foreground truncate">
                        {resolveDisplayName(
                          contentLocale,
                          trait.display_name_locs,
                          trait.name_en,
                          trait.name_zh,
                          t("noDisplayName"),
                        )}
                      </div>
                    </li>
                  );
                })}
              </ul>
            </aside>

            {viewMode === "tree" ? (
              <div className="flex flex-col min-h-0">
                <p className="px-4 py-2 text-xs text-muted-foreground border-b leading-relaxed">
                  {t("canvasHint")}
                </p>
                <div className="flex-1 min-h-0">
                  <TraitTreeGraph
                    traits={visibleTraits}
                    selectedTraitDbId={selectedId}
                    onSelectTrait={(dbId) => {
                      const found =
                        dbId === null
                          ? null
                          : traits.find((tr) => tr.id === dbId);
                      if (found) selectTrait(found);
                      else setSelectedId(null);
                    }}
                    onPersist={persistTraitPatch}
                    locale={contentLocale}
                  />
                </div>
              </div>
            ) : (
              <div className="overflow-y-auto p-4 space-y-4">
                <div className="grid grid-cols-2 gap-3">
                  <div className="space-y-1">
                    <Label>Trait ID (code name)</Label>
                    <Input
                      value={draft.trait_id}
                      onChange={(e) => updateField("trait_id", e.target.value)}
                      placeholder={
                        activeType === "unit_leader"
                          ? "my_brilliant_strategist"
                          : activeType === "scientist"
                            ? "my_rocket_genius"
                            : "my_industrial_expert"
                      }
                      className="font-mono text-xs"
                      disabled={selectedId !== null}
                    />
                    {selectedId !== null && (
                      <p className="text-xs text-muted-foreground">
                        Trait ID is immutable after creation.
                      </p>
                    )}
                  </div>
                  {activeType === "country_leader" && (
                    <div className="space-y-1">
                      <Label>Random (auto-assign)</Label>
                      <div className="flex items-center gap-2 h-9">
                        <input
                          type="checkbox"
                          checked={draft.random}
                          onChange={(e) =>
                            updateField("random", e.target.checked)
                          }
                        />
                        <span className="text-xs text-muted-foreground">
                          HOI4 may auto-assign to random country leaders
                        </span>
                      </div>
                    </div>
                  )}
                  {activeType === "unit_leader" && (
                    <div className="space-y-1">
                      <Label>Type (which commanders)</Label>
                      <Select
                        value={readExtraString("type", "land")}
                        onChange={(e) => updateExtra("type", e.target.value)}
                        className="px-2 text-xs"
                      >
                        {UNIT_LEADER_TRAIT_TYPE_OPTIONS.map((opt) => (
                          <option key={opt.value} value={opt.value}>
                            {localizedOptionLabel(opt.label, appLocale)}
                          </option>
                        ))}
                      </Select>
                      <p className="text-xs text-muted-foreground">
                        land = army (general/marshal); navy = admiral; all =
                        both
                      </p>
                    </div>
                  )}
                  {activeType === "scientist" && (
                    <div className="space-y-1">
                      <Label>Specialization (special-project family)</Label>
                      <Select
                        value={readExtraString("specialization", "")}
                        onChange={(e) =>
                          updateExtra("specialization", e.target.value)
                        }
                        className="px-2 text-xs"
                      >
                        {SCIENTIST_TRAIT_SPECIALIZATION_OPTIONS.map((opt) => (
                          <option key={opt.value} value={opt.value}>
                            {localizedOptionLabel(opt.label, appLocale)}
                          </option>
                        ))}
                      </Select>
                      <p className="text-xs text-muted-foreground">
                        Limits which Special Project families this trait applies
                        to. Generic = no constraint.
                      </p>
                    </div>
                  )}
                </div>
                {activeType === "scientist" && (
                  <div className="rounded-md border border-amber-300/50 bg-amber-50/50 dark:bg-amber-950/20 p-2 text-xs text-amber-900 dark:text-amber-200">
                    <strong>{t("scientistConstraintLabel")}</strong>{" "}
                    {t("scientistConstraintIntro")}{" "}
                    <em>{t("scientistConstraintSpecialProjects")}</em>{" "}
                    {t("scientistConstraintDlc")}{" "}
                    <code className="font-mono">
                      special_project_speed_factor
                    </code>
                    ,{" "}
                    <code className="font-mono">
                      scientist_breakthrough_bonus_factor
                    </code>
                    ,{" "}
                    <code className="font-mono">
                      sp_tag_rocket_speed_factor
                    </code>
                    . {t("scientistConstraintOutro")}
                  </div>
                )}

                {activeType === "unit_leader" && (
                  <div className="space-y-2 border rounded-md p-3 bg-muted/30">
                    <Label className="font-medium">{t("skillsSection")}</Label>
                    <div className="grid grid-cols-3 gap-2">
                      {UNIT_LEADER_SKILL_KEYS.map((skillKey) => (
                        <div key={skillKey} className="space-y-1">
                          <Label className="text-micro uppercase text-muted-foreground">
                            {skillKey.replace("_skill", "")}
                          </Label>
                          <Input
                            type="number"
                            min={0}
                            max={20}
                            step={1}
                            value={readExtraNumber(skillKey)}
                            onChange={(e) => {
                              const n = parseInt(e.target.value, 10);
                              updateExtra(skillKey, Number.isFinite(n) ? n : 0);
                            }}
                            className="h-8 text-xs"
                          />
                        </div>
                      ))}
                    </div>
                  </div>
                )}

                {activeType === "unit_leader" && (
                  <details
                    open={advancedOpen}
                    onToggle={(e) =>
                      setAdvancedOpen((e.target as HTMLDetailsElement).open)
                    }
                    className="rounded-md border bg-muted/20"
                  >
                    <summary className="cursor-pointer px-3 py-2 text-xs font-medium select-none flex items-center gap-2">
                      <span>{t("advancedFieldsTitle")}</span>
                      {advancedExtrasCount > 0 && (
                        <span className="ml-1 rounded-full bg-accent/15 text-accent border border-accent/30 px-2 py-0.5 text-micro">
                          {t("advancedSet", { count: advancedExtrasCount })}
                        </span>
                      )}
                      <span className="ml-auto text-muted-foreground text-micro">
                        {advancedOpen ? t("collapse") : t("expand")}
                      </span>
                    </summary>

                    <div className="p-3 space-y-4 border-t">
                      <div className="space-y-2">
                        <Label className="font-medium">
                          {t("skillFactorsSection")}
                        </Label>
                        <div className="grid grid-cols-3 gap-2">
                          {UNIT_LEADER_SKILL_FACTOR_KEYS.map((fkey) => (
                            <div key={fkey} className="space-y-1">
                              <Label className="text-micro uppercase text-muted-foreground">
                                {fkey.replace("_skill_factor", "")} ×
                              </Label>
                              <Input
                                type="number"
                                step={0.05}
                                value={readExtraFloat(fkey)}
                                placeholder="0.0"
                                onChange={(e) => {
                                  const raw = e.target.value;
                                  if (raw === "") {
                                    setDraft((prev) => {
                                      const next = { ...prev.extra_fields };
                                      delete next[fkey];
                                      return { ...prev, extra_fields: next };
                                    });
                                    return;
                                  }
                                  const n = parseFloat(raw);
                                  if (Number.isFinite(n)) updateExtra(fkey, n);
                                }}
                                className="h-8 text-xs"
                              />
                            </div>
                          ))}
                        </div>
                      </div>

                      <div className="space-y-2">
                        <Label className="font-medium">
                          {t("treePositionCostSection")}
                        </Label>
                        <div className="grid grid-cols-2 gap-2">
                          {UNIT_LEADER_ADVANCED_INT_FIELDS.map((f) => (
                            <div key={f.key} className="space-y-1">
                              <Label
                                className="text-micro text-muted-foreground"
                                title={f.hint}
                              >
                                {f.label}
                              </Label>
                              <Input
                                type="number"
                                step={1}
                                value={readExtraFloat(f.key)}
                                placeholder="(unset)"
                                onChange={(e) => {
                                  const raw = e.target.value;
                                  if (raw === "") {
                                    setDraft((prev) => {
                                      const next = { ...prev.extra_fields };
                                      delete next[f.key];
                                      return { ...prev, extra_fields: next };
                                    });
                                    return;
                                  }
                                  const n = parseInt(raw, 10);
                                  if (Number.isFinite(n)) updateExtra(f.key, n);
                                }}
                                className="h-8 text-xs"
                              />
                            </div>
                          ))}
                        </div>
                      </div>

                      <div className="space-y-2">
                        <Label className="font-medium">
                          {t("stringFieldsSection")}
                        </Label>
                        <div className="grid grid-cols-2 gap-2">
                          {UNIT_LEADER_ADVANCED_STRING_FIELDS.map((f) => (
                            <div key={f.key} className="space-y-1">
                              <Label
                                className="text-micro text-muted-foreground"
                                title={f.hint}
                              >
                                {f.label}
                              </Label>
                              <Input
                                value={readExtraString(f.key, "")}
                                placeholder={f.hint.split(" · ")[0]}
                                onChange={(e) =>
                                  updateExtra(f.key, e.target.value)
                                }
                                className="h-8 text-xs font-mono"
                              />
                            </div>
                          ))}
                        </div>
                      </div>

                      <div className="space-y-1">
                        {UNIT_LEADER_ADVANCED_BOOL_FIELDS.map((f) => (
                          <label
                            key={f.key}
                            className="flex items-center gap-2 text-xs cursor-pointer"
                          >
                            <input
                              type="checkbox"
                              checked={readExtraBool(f.key)}
                              onChange={(e) =>
                                updateExtra(f.key, e.target.checked)
                              }
                            />
                            <span className="font-mono">{f.label}</span>
                            <span className="text-muted-foreground text-micro">
                              {f.hint}
                            </span>
                          </label>
                        ))}
                      </div>

                      <div className="space-y-2">
                        <Label className="font-medium">
                          {t("modifierVariantsSection")}
                        </Label>
                        {UNIT_LEADER_ADVANCED_MODIFIER_FIELDS.map((f) => (
                          <div key={f.key} className="space-y-1">
                            <Label
                              className="text-micro text-muted-foreground"
                              title={f.hint}
                            >
                              {f.label}{" "}
                              <span className="text-muted-foreground/70">
                                · {f.hint}
                              </span>
                            </Label>
                            <Textarea
                              value={readExtraModifier(f.key)}
                              onChange={(e) =>
                                updateExtraModifier(f.key, e.target.value)
                              }
                              rows={3}
                              placeholder={`{"army_speed_factor": 0.05}`}
                              className="font-mono text-xs"
                            />
                          </div>
                        ))}
                      </div>

                      <div className="space-y-2">
                        <Label className="font-medium">
                          {t("scriptBlocksSection")}{" "}
                          <code className="font-mono">key = {`{ ... }`}</code>
                        </Label>
                        {UNIT_LEADER_ADVANCED_SCRIPT_FIELDS.map((f) => (
                          <div key={f.key} className="space-y-1">
                            <Label
                              className="text-micro text-muted-foreground"
                              title={f.hint}
                            >
                              {f.label}{" "}
                              <span className="text-muted-foreground/70">
                                · {f.hint}
                              </span>
                            </Label>
                            <Textarea
                              value={readExtraString(f.key, "")}
                              onChange={(e) =>
                                updateExtra(f.key, e.target.value)
                              }
                              rows={2}
                              placeholder={
                                f.key.startsWith("on_") ||
                                f.key === "daily_effect"
                                  ? "set_unit_leader_flag = my_flag"
                                  : "has_trait = some_prereq"
                              }
                              className="font-mono text-xs"
                            />
                          </div>
                        ))}
                      </div>

                      <p className="text-micro text-muted-foreground">
                        The trait-tree blocks (
                        <code className="font-mono">parent</code> /{" "}
                        <code className="font-mono">any_parent</code> /{" "}
                        <code className="font-mono">all_parents</code> /{" "}
                        <code className="font-mono">sub_unit_modifiers</code> /{" "}
                        <code className="font-mono">trait_xp_factor</code> /{" "}
                        <code className="font-mono">new_commander_weight</code>)
                        survive import and ARE written back on export. They are
                        kept verbatim, so this form does not edit them — use the{" "}
                        <strong>Trait tree</strong> view above to add or remove{" "}
                        <code className="font-mono">any_parent</code>{" "}
                        prerequisites. Typing one of these keys into{" "}
                        <code className="font-mono">extra_fields</code> here has
                        no effect: the exporter reads them only from the
                        preserved import data.
                      </p>
                    </div>
                  </details>
                )}

                <div className="space-y-1">
                  <Label>
                    {tContent("contentNameLabel", {
                      locale: contentLocale.toUpperCase(),
                    })}
                  </Label>
                  <Input
                    value={resolveDisplayName(
                      contentLocale,
                      draft.display_name_locs,
                      draft.name_en,
                      draft.name_zh,
                      "",
                    )}
                    onChange={(e) =>
                      updateField("display_name_locs", {
                        ...draft.display_name_locs,
                        [contentLocale]: e.target.value,
                      })
                    }
                    placeholder={t("namePlaceholderEn")}
                  />
                  <p className="text-xs text-muted-foreground">
                    {t("nameHint")}
                  </p>
                </div>

                <div className="space-y-1">
                  <Label>
                    {tContent("contentDescriptionLabel", {
                      locale: contentLocale.toUpperCase(),
                    })}
                  </Label>
                  <Textarea
                    value={resolveDisplayName(
                      contentLocale,
                      draft.description_locs,
                      draft.desc_en,
                      draft.desc_zh,
                      "",
                    )}
                    onChange={(e) =>
                      updateField("description_locs", {
                        ...draft.description_locs,
                        [contentLocale]: e.target.value,
                      })
                    }
                    rows={2}
                  />
                </div>

                {(activeType === "unit_leader" ||
                  activeType === "scientist") && (
                  <div className="space-y-2 border rounded-md p-3 bg-muted/20">
                    <p className="text-xs text-muted-foreground">
                      {activeType === "unit_leader"
                        ? "Assignable traits (set Trait Type = assignable + a GUI row) show this icon in the commander trait tree."
                        : "Shown on the scientist's trait in the Special Projects screen."}
                    </p>
                    {selectedId === null ? (
                      <p className="text-xs text-muted-foreground italic">
                        Save the trait first to upload an icon.
                      </p>
                    ) : (
                      <ImageUpload
                        label={t("traitIconLabel")}
                        targetSize={TRAIT_ICON_SIZE}
                        previewUrl={
                          iconUploaded
                            ? customTraitsApi.iconUrl(projectId, selectedId)
                            : null
                        }
                        hint="PNG / JPG / DDS · small icon (e.g. 23 × 33 / square ~64px) · auto-converted to DDS at export · GFX name auto-generated"
                        onUpload={async (file, crop) =>
                          customTraitsApi.uploadIcon(
                            projectId,
                            selectedId,
                            file,
                            crop,
                          )
                        }
                        onUploaded={() => {
                          setIconUploaded(true);
                          fetchTraits();
                        }}
                      />
                    )}
                  </div>
                )}

                <div className="space-y-1">
                  <Label>
                    Modifier (JSON object — e.g. {"{"}
                    &quot;political_power_factor&quot;: 0.25,
                    &quot;stability_weekly&quot;: 0.001{"}"})
                  </Label>
                  <Textarea
                    value={modifierText}
                    onChange={(e) => setModifierText(e.target.value)}
                    onBlur={parseModifier}
                    rows={6}
                    className="font-mono text-xs"
                  />
                  {modifierError && (
                    <p className="text-xs text-destructive">
                      ⚠ {modifierError}
                    </p>
                  )}
                </div>

                {customTraitSupportsTargetedModifier(activeType) && (
                  <div className="space-y-3 rounded-md border border-border p-3">
                    <div className="space-y-1">
                      <Label>{t("targetedModifierTitle")}</Label>
                      <p className="text-xs text-muted-foreground">
                        {t("targetedModifierHelp")}
                      </p>
                    </div>

                    {draft.targeted_modifier.length === 0 && (
                      <p className="text-xs text-muted-foreground">
                        {t("targetedModifierEmpty")}
                      </p>
                    )}

                    {targetedModifierError && (
                      <p className="text-xs text-destructive" role="alert">
                        {targetedModifierError}
                      </p>
                    )}

                    {draft.targeted_modifier.map((target, targetIndex) => {
                      const modifierEntries = Object.entries(target.modifier);
                      return (
                        <div
                          key={targetIndex}
                          className="space-y-2 rounded-md border border-border bg-muted/20 p-3"
                        >
                          <div className="flex items-end gap-2">
                            <div className="flex-1 space-y-1">
                              <Label
                                htmlFor={`targeted-country-${targetIndex}`}
                              >
                                {t("targetedModifierCountry")}
                              </Label>
                              <Input
                                id={`targeted-country-${targetIndex}`}
                                value={target.tag}
                                onChange={(event) =>
                                  updateTargetedModifier(
                                    targetIndex,
                                    (entry) => ({
                                      ...entry,
                                      tag: event.target.value,
                                    }),
                                  )
                                }
                                placeholder={t(
                                  "targetedModifierCountryPlaceholder",
                                )}
                                className="font-mono uppercase"
                                maxLength={10}
                              />
                            </div>
                            <Button
                              type="button"
                              variant="ghost"
                              size="sm"
                              onClick={() =>
                                replaceTargetedModifiers(
                                  draft.targeted_modifier.filter(
                                    (_, index) => index !== targetIndex,
                                  ),
                                )
                              }
                            >
                              {t("targetedModifierRemoveTarget")}
                            </Button>
                          </div>

                          {modifierEntries.map(
                            ([modifierKey, modifierValue], modifierIndex) => (
                              <div
                                key={modifierIndex}
                                className="grid grid-cols-[1fr_160px_auto] items-end gap-2"
                              >
                                <div className="space-y-1">
                                  <Label>{t("targetedModifierKey")}</Label>
                                  <Input
                                    value={modifierKey}
                                    onChange={(event) =>
                                      updateTargetedModifierKey(
                                        targetIndex,
                                        modifierIndex,
                                        event.target.value,
                                      )
                                    }
                                    placeholder={t(
                                      "targetedModifierKeyPlaceholder",
                                    )}
                                    className="font-mono"
                                  />
                                </div>
                                <div className="space-y-1">
                                  <Label>{t("targetedModifierValue")}</Label>
                                  <Input
                                    value={String(modifierValue)}
                                    onChange={(event) =>
                                      updateTargetedModifierValue(
                                        targetIndex,
                                        modifierIndex,
                                        event.target.value,
                                      )
                                    }
                                    placeholder={t(
                                      "targetedModifierValuePlaceholder",
                                    )}
                                    className="font-mono"
                                  />
                                </div>
                                <Button
                                  type="button"
                                  variant="ghost"
                                  size="sm"
                                  onClick={() =>
                                    updateTargetedModifier(
                                      targetIndex,
                                      (entry) => ({
                                        ...entry,
                                        modifier: Object.fromEntries(
                                          Object.entries(entry.modifier).filter(
                                            (_, index) =>
                                              index !== modifierIndex,
                                          ),
                                        ),
                                      }),
                                    )
                                  }
                                >
                                  {t("targetedModifierRemoveValue")}
                                </Button>
                              </div>
                            ),
                          )}

                          <Button
                            type="button"
                            variant="outline"
                            size="sm"
                            disabled={Object.prototype.hasOwnProperty.call(
                              target.modifier,
                              "",
                            )}
                            onClick={() =>
                              updateTargetedModifier(targetIndex, (entry) => ({
                                ...entry,
                                modifier: { ...entry.modifier, "": "" },
                              }))
                            }
                          >
                            {t("targetedModifierAddValue")}
                          </Button>
                        </div>
                      );
                    })}

                    <Button
                      type="button"
                      variant="outline"
                      size="sm"
                      onClick={() =>
                        replaceTargetedModifiers([
                          ...draft.targeted_modifier,
                          { tag: "", modifier: { "": "" } },
                        ])
                      }
                    >
                      {t("targetedModifierAddTarget")}
                    </Button>
                  </div>
                )}

                <div className="space-y-1">
                  <Label>{t("previewLabel")}</Label>
                  <CustomTraitPreview
                    name={previewName}
                    desc={previewDesc}
                    modifier={previewModifier}
                    typeLabel={previewTypeLabel}
                  />
                </div>

                {activeType !== "scientist" && (
                  <div className="space-y-1">
                    <Label>ai_will_do (raw script, optional)</Label>
                    <Textarea
                      value={draft.ai_will_do}
                      onChange={(e) =>
                        updateField("ai_will_do", e.target.value)
                      }
                      placeholder="factor = 1"
                      rows={3}
                      className="font-mono text-xs"
                    />
                  </div>
                )}

                <div className="flex items-center gap-2 pt-2 border-t">
                  <Button onClick={save} disabled={saving}>
                    {saving
                      ? "Saving..."
                      : selectedId === null
                        ? "Create"
                        : "Save"}
                  </Button>
                  {selectedId !== null && (
                    <Button
                      variant="destructive"
                      onClick={deleteTrait}
                      disabled={saving}
                    >
                      Delete
                    </Button>
                  )}
                  {status && (
                    <span className="text-xs text-muted-foreground">
                      {status}
                    </span>
                  )}
                </div>
              </div>
            )}
          </div>
        </DialogContent>
      </Dialog>
      <OpinionModifiersPanel
        projectId={projectId}
        open={opinionModifiersOpen}
        onOpenChange={setOpinionModifiersOpen}
        onChanged={onTraitsChanged}
      />
    </>
  );
}
