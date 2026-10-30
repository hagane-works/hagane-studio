"use client";

import { useState } from "react";
import { useTranslations } from "use-intl";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import type { AutonomyState } from "@/types";
import { useProjectStore } from "@/lib/store/projectStore";
import { DisplayNameLocsTabs } from "@/components/shared/editors/DisplayNameLocsTabs";
import { DEFAULT_EXPORT_LANGUAGES } from "@/components/shared/editors/displayNameLocsHelpers";
import { useRegisterDirty } from "@/lib/hooks/useUnsavedGuard";

interface KVPair {
  key: string;
  value: string;
}

interface DraftAS {
  autonomy_id: string;
  name_en: string;
  name_zh: string;
  display_name_locs: Record<string, string>;
  is_puppet: boolean;

  is_default: boolean;
  manpower_influence: number | null;

  min_freedom_level: number;
  rule: string;

  aiWeightPairs: KVPair[];

  modifierPairs: KVPair[];
}

function asToDraft(s: AutonomyState): DraftAS {
  return {
    autonomy_id: s.autonomy_id || "",
    name_en: s.name_en || "",
    name_zh: s.name_zh || "",
    display_name_locs: { ...(s.display_name_locs ?? {}) },
    is_puppet: !!s.is_puppet,
    is_default: !!s.is_default,
    manpower_influence: s.manpower_influence ?? null,
    min_freedom_level: Number(s.min_freedom_level || 0),
    rule: s.rule || "",
    aiWeightPairs: Object.entries(s.ai_weights || {}).map(([k, v]) => ({
      key: k,
      value: typeof v === "string" ? v : String(v ?? ""),
    })),
    modifierPairs: Object.entries(s.modifier || {}).map(([k, v]) => ({
      key: k,
      value: String(v),
    })),
  };
}

function pairsToNumericDict(pairs: KVPair[]): Record<string, string | number> {
  const out: Record<string, string | number> = {};
  for (const { key, value } of pairs) {
    if (!key.trim()) continue;
    const num = Number(value);
    out[key.trim()] = !isNaN(num) && value.trim() !== "" ? num : value;
  }
  return out;
}

function pairsToStringDict(pairs: KVPair[]): Record<string, string> {
  const out: Record<string, string> = {};
  for (const { key, value } of pairs) {
    if (!key.trim()) continue;
    out[key.trim()] = value;
  }
  return out;
}

function draftToWrite(draft: DraftAS): Partial<AutonomyState> {
  return {
    autonomy_id: draft.autonomy_id.trim(),
    name_en: draft.name_en,
    name_zh: draft.name_zh,
    display_name_locs: draft.display_name_locs,
    is_puppet: draft.is_puppet,
    is_default: draft.is_default,
    manpower_influence: draft.manpower_influence,
    min_freedom_level: Math.round(Number(draft.min_freedom_level) || 0),
    rule: draft.rule,
    ai_weights: pairsToStringDict(draft.aiWeightPairs),
    modifier: pairsToNumericDict(draft.modifierPairs),
  };
}

export function AutonomyStateEditor({
  state,
  onSave,
}: {
  state: AutonomyState;

  onSave: (data: Partial<AutonomyState>) => Promise<void>;
}) {
  const t = useTranslations("autonomyStates");
  const exportLanguages = useProjectStore(
    (s) => s.currentProject?.export_languages ?? DEFAULT_EXPORT_LANGUAGES,
  );
  const [draft, setDraft] = useState<DraftAS>(() => asToDraft(state));
  const [savedDraft, setSavedDraft] = useState<DraftAS>(draft);
  const [saving, setSaving] = useState(false);
  const [saveStatus, setSaveStatus] = useState<"idle" | "success" | "error">(
    "idle",
  );
  const [error, setError] = useState<string | null>(null);

  const handleSave = async () => {
    setSaving(true);
    setSaveStatus("idle");
    setError(null);
    try {
      await onSave(draftToWrite(draft));
      setSavedDraft(draft);
      setSaveStatus("success");
      setTimeout(() => setSaveStatus("idle"), 2000);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
      setSaveStatus("error");
    } finally {
      setSaving(false);
    }
  };

  const isUnsaved = JSON.stringify(draft) !== JSON.stringify(savedDraft);
  useRegisterDirty("autonomy-states", isUnsaved);

  const freedomScale = (Number(draft.min_freedom_level) || 0) / 1000;

  return (
    <div className="p-4 max-w-3xl space-y-4">
      <div>
        <h2 className="text-sm font-semibold mb-1">{t("editorHeading")}</h2>
        <p className="text-xs text-muted-foreground">
          {t("landing")}{" "}
          <span className="font-mono">
            common/autonomous_states/{`{TAG}`}_autonomous_states.txt
          </span>
        </p>
      </div>

      <div className="grid grid-cols-2 gap-3">
        <div className="space-y-1">
          <Label>{t("autonomyIdLabel")}</Label>
          <Input
            className="font-mono text-xs"
            value={draft.autonomy_id}
            onChange={(e) =>
              setDraft({ ...draft, autonomy_id: e.target.value })
            }
          />
        </div>
        <div className="space-y-1">
          <Label>{t("isPuppetLabel")}</Label>
          <label className="flex items-center gap-2 h-9 text-xs cursor-pointer">
            <input
              type="checkbox"
              className="h-4 w-4"
              checked={draft.is_puppet}
              onChange={(e) =>
                setDraft({ ...draft, is_puppet: e.target.checked })
              }
            />
            <span className="text-muted-foreground">{t("isPuppetHint")}</span>
          </label>
        </div>
      </div>

      <div className="space-y-1">
        <Label>{t("nameLabel")}</Label>
        <DisplayNameLocsTabs
          entityKey={`autonomy-state:${state.id}:display-name`}
          value={draft.display_name_locs}
          exportLanguages={exportLanguages}
          legacyFallback={{ en: draft.name_en, zh: draft.name_zh }}
          onChange={(next) => setDraft({ ...draft, display_name_locs: next })}
          ariaLabel={t("nameLabel")}
        />
      </div>

      <div className="space-y-1">
        <Label>{t("minFreedomLevelLabel")}</Label>
        <div className="flex items-center gap-3">
          <Input
            type="number"
            min={0}
            max={1000}
            step={50}
            className="font-mono text-xs w-40"
            value={String(draft.min_freedom_level)}
            onChange={(e) =>
              setDraft({ ...draft, min_freedom_level: Number(e.target.value) })
            }
          />
          <span className="text-xs text-muted-foreground">
            {t("freedomScale", { scale: freedomScale.toFixed(3) })}
          </span>
        </div>
        <p className="text-xs text-muted-foreground">
          {t("minFreedomLevelHint")}
        </p>
      </div>

      <div className="grid grid-cols-2 gap-3">
        <div className="space-y-1">
          <Label>{t("isDefaultLabel")}</Label>
          <label className="flex items-center gap-2 h-9 text-xs cursor-pointer">
            <input
              type="checkbox"
              className="h-4 w-4"
              checked={draft.is_default}
              onChange={(e) =>
                setDraft({ ...draft, is_default: e.target.checked })
              }
            />
            <span className="text-muted-foreground">{t("isDefaultHint")}</span>
          </label>
        </div>
        <div className="space-y-1">
          <Label>{t("manpowerInfluenceLabel")}</Label>
          <Input
            type="number"
            min={0}
            max={1}
            step={0.05}
            className="font-mono text-xs w-40"
            value={draft.manpower_influence ?? ""}
            placeholder={t("manpowerInfluencePlaceholder")}
            onChange={(e) =>
              setDraft({
                ...draft,
                manpower_influence:
                  e.target.value === "" ? null : Number(e.target.value),
              })
            }
          />
          <p className="text-xs text-muted-foreground">
            {t("manpowerInfluenceHint")}
          </p>
        </div>
      </div>

      <div className="space-y-1">
        <Label>{t("ruleLabel")}</Label>
        <Textarea
          className="font-mono text-xs min-h-24"
          placeholder={
            "can_not_declare_war = yes\ncan_decline_call_to_war = no"
          }
          value={draft.rule}
          onChange={(e) => setDraft({ ...draft, rule: e.target.value })}
        />
        <p className="text-xs text-muted-foreground">{t("ruleHint")}</p>
      </div>

      <div className="space-y-2">
        <div className="flex items-center justify-between">
          <Label>{t("aiWeightsLabel")}</Label>
          <Button
            size="sm"
            variant="outline"
            className="text-xs"
            onClick={() =>
              setDraft({
                ...draft,
                aiWeightPairs: [...draft.aiWeightPairs, { key: "", value: "" }],
              })
            }
          >
            {t("addField")}
          </Button>
        </div>
        {draft.aiWeightPairs.length === 0 && (
          <p className="text-xs text-muted-foreground italic">
            {t("noAiWeights")}
          </p>
        )}
        {draft.aiWeightPairs.map((p, i) => (
          <div key={i} className="grid grid-cols-[1fr_1fr_auto] gap-2">
            <Input
              placeholder="ai_subject_wants_higher"
              className="font-mono text-xs"
              value={p.key}
              onChange={(e) => {
                const next = [...draft.aiWeightPairs];
                next[i] = { ...next[i], key: e.target.value };
                setDraft({ ...draft, aiWeightPairs: next });
              }}
            />
            <Input
              placeholder="factor = 1.0"
              className="font-mono text-xs"
              value={p.value}
              onChange={(e) => {
                const next = [...draft.aiWeightPairs];
                next[i] = { ...next[i], value: e.target.value };
                setDraft({ ...draft, aiWeightPairs: next });
              }}
            />
            <Button
              size="sm"
              variant="ghost"
              className="px-2 text-muted-foreground hover:text-destructive"
              onClick={() =>
                setDraft({
                  ...draft,
                  aiWeightPairs: draft.aiWeightPairs.filter((_, j) => j !== i),
                })
              }
              aria-label={t("removeFieldAria")}
              title={t("removeFieldAria")}
            >
              ×
            </Button>
          </div>
        ))}
      </div>

      <div className="space-y-2">
        <div className="flex items-center justify-between">
          <Label>{t("modifierLabel")}</Label>
          <Button
            size="sm"
            variant="outline"
            className="text-xs"
            onClick={() =>
              setDraft({
                ...draft,
                modifierPairs: [...draft.modifierPairs, { key: "", value: "" }],
              })
            }
          >
            {t("addField")}
          </Button>
        </div>
        {draft.modifierPairs.length === 0 && (
          <p className="text-xs text-muted-foreground italic">
            {t("noFields")}
          </p>
        )}
        {draft.modifierPairs.map((p, i) => (
          <div key={i} className="grid grid-cols-[1fr_1fr_auto] gap-2">
            <Input
              placeholder="autonomy_manpower_share"
              className="font-mono text-xs"
              value={p.key}
              onChange={(e) => {
                const next = [...draft.modifierPairs];
                next[i] = { ...next[i], key: e.target.value };
                setDraft({ ...draft, modifierPairs: next });
              }}
            />
            <Input
              placeholder="0.9"
              className="font-mono text-xs"
              value={p.value}
              onChange={(e) => {
                const next = [...draft.modifierPairs];
                next[i] = { ...next[i], value: e.target.value };
                setDraft({ ...draft, modifierPairs: next });
              }}
            />
            <Button
              size="sm"
              variant="ghost"
              className="px-2 text-muted-foreground hover:text-destructive"
              onClick={() =>
                setDraft({
                  ...draft,
                  modifierPairs: draft.modifierPairs.filter((_, j) => j !== i),
                })
              }
              aria-label={t("removeFieldAria")}
              title={t("removeFieldAria")}
            >
              ×
            </Button>
          </div>
        ))}
      </div>

      {(saving || saveStatus !== "idle" || isUnsaved) && (
        <div
          className={`
            -mx-4 px-4 py-1.5 text-xs font-medium flex items-center gap-2
            transition-[opacity,background-color,border-color] duration-300
            ${
              saving
                ? "bg-muted/80 text-muted-foreground border-y border-border"
                : saveStatus === "success"
                  ? "bg-success/10 text-success border-y border-success/20"
                  : saveStatus === "error"
                    ? "bg-destructive/10 text-destructive border-y border-destructive/20"
                    : isUnsaved
                      ? "bg-warning/10 text-warning border-y border-warning/20"
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
                  ? (error ?? t("saveFailed"))
                  : isUnsaved
                    ? t("unsavedChanges")
                    : ""}
          </span>
        </div>
      )}
      <div className="flex items-center gap-2 pt-2 border-t border-border">
        <Button onClick={handleSave} disabled={saving} size="sm">
          {t("saveButton")}
        </Button>
      </div>

      <details className="text-xs text-muted-foreground">
        <summary className="cursor-pointer">{t("usageExample")}</summary>
        <pre className="mt-2 p-2 bg-muted rounded text-xs overflow-x-auto">
          {`# ${t("usageComment")}
TAG = {
  set_autonomy = {
    target = SUBJECT_TAG
    autonomy_state = ${draft.autonomy_id}
  }
}`}
        </pre>
      </details>
    </div>
  );
}
