"use client";

import { useMemo, useState } from "react";
import { useTranslations } from "use-intl";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import type { DynamicModifier } from "@/types";
import { useActiveCountry, useProjectStore } from "@/lib/store/projectStore";
import {
  useVanillaCollisions,
  suggestNonCollidingId,
} from "@/lib/vanillaCollisions";
import { DisplayNameLocsTabs } from "@/components/shared/editors/DisplayNameLocsTabs";
import { DEFAULT_EXPORT_LANGUAGES } from "@/components/shared/editors/displayNameLocsHelpers";
import { useRegisterDirty } from "@/lib/hooks/useUnsavedGuard";

interface DraftDM {
  dm_id: string;
  name_en: string;
  name_zh: string;

  display_name_locs: Record<string, string>;
  icon: string;
  enable: string;
  remove_trigger: string;

  modifierPairs: { key: string; value: string }[];
}

function dmToDraft(dm: DynamicModifier): DraftDM {
  return {
    dm_id: dm.dm_id || "",
    name_en: dm.name_en || "",
    name_zh: dm.name_zh || "",
    display_name_locs: { ...(dm.display_name_locs ?? {}) },
    icon: dm.icon || "",
    enable: dm.enable || "",
    remove_trigger: dm.remove_trigger || "",
    modifierPairs: Object.entries(dm.modifier || {}).map(([k, v]) => ({
      key: k,
      value: String(v),
    })),
  };
}

function pairsToDict(
  pairs: { key: string; value: string }[],
): Record<string, string | number> {
  const out: Record<string, string | number> = {};
  for (const { key, value } of pairs) {
    if (!key.trim()) continue;
    const num = Number(value);
    out[key.trim()] = !isNaN(num) && value.trim() !== "" ? num : value;
  }
  return out;
}

export function DynamicModifierEditor({
  dm,
  onSave,
}: {
  dm: DynamicModifier;

  onSave: (data: Partial<DynamicModifier>) => Promise<void>;
}) {
  const activeCountry = useActiveCountry();

  const exportLanguages = useProjectStore(
    (s) => s.currentProject?.export_languages ?? DEFAULT_EXPORT_LANGUAGES,
  );
  const t = useTranslations("dynamicModifiers");

  const [draft, setDraft] = useState<DraftDM>(() => dmToDraft(dm));
  const [savedDraft, setSavedDraft] = useState<DraftDM>(() => dmToDraft(dm));
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
      await onSave({
        dm_id: draft.dm_id.trim(),
        name_en: draft.name_en,
        name_zh: draft.name_zh,
        display_name_locs: draft.display_name_locs,
        icon: draft.icon,
        enable: draft.enable,
        remove_trigger: draft.remove_trigger,
        modifier: pairsToDict(draft.modifierPairs),
      });
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

  const vanillaCollisionsData = useVanillaCollisions();
  const dmCollision = useMemo(() => {
    const trimmed = (draft.dm_id || "").trim();
    const isColliding =
      vanillaCollisionsData.loaded &&
      !!trimmed &&
      vanillaCollisionsData.dynamicModifierIds.has(trimmed);
    const suggested = suggestNonCollidingId(trimmed, activeCountry || "");
    return { isColliding, suggested };
  }, [vanillaCollisionsData, draft.dm_id, activeCountry]);

  const isUnsaved = JSON.stringify(draft) !== JSON.stringify(savedDraft);

  useRegisterDirty("dynamic-modifiers", isUnsaved);

  return (
    <div className="p-4 max-w-3xl space-y-4">
      <div>
        <h2 className="text-sm font-semibold mb-1">{t("editorHeading")}</h2>
        <p className="text-xs text-muted-foreground">
          {t("landing")}{" "}
          <span className="font-mono">
            common/dynamic_modifiers/{`{TAG}`}_dynamic_modifiers.txt
          </span>
        </p>
      </div>

      <div className="grid grid-cols-2 gap-3">
        <div className="space-y-1">
          <Label>{t("dmIdLabel")}</Label>
          <Input
            className={`font-mono text-xs ${
              dmCollision.isColliding
                ? "border-destructive focus-visible:ring-destructive/40"
                : ""
            }`}
            value={draft.dm_id}
            onChange={(e) => setDraft({ ...draft, dm_id: e.target.value })}
          />

          {dmCollision.isColliding && (
            <div className="p-2 rounded border border-destructive/40 bg-destructive/5 text-xs space-y-1">
              <p className="text-destructive font-medium">
                {t.rich("collisionWarning", {
                  id: draft.dm_id.trim(),
                  code: (c) => <code className="font-mono">{c}</code>,
                })}
              </p>
              <button
                type="button"
                onClick={() =>
                  setDraft({ ...draft, dm_id: dmCollision.suggested })
                }
                className="text-xs underline decoration-dotted text-destructive/80 hover:text-destructive"
              >
                {t("renameTo")}{" "}
                <code className="font-mono">{dmCollision.suggested}</code>{" "}
                {t("recommended")}
              </button>
            </div>
          )}
        </div>
        <div className="space-y-1">
          <Label>Icon · GFX sprite name</Label>
          <Input
            className="font-mono text-xs"
            placeholder="GFX_idea_unknown"
            value={draft.icon}
            onChange={(e) => setDraft({ ...draft, icon: e.target.value })}
          />
        </div>
      </div>

      <div className="space-y-1">
        <Label>{t("nameLabel")}</Label>
        <DisplayNameLocsTabs
          entityKey={`dynamic-modifier:${dm.id}:display-name`}
          value={draft.display_name_locs}
          exportLanguages={exportLanguages}
          legacyFallback={{ en: draft.name_en, zh: draft.name_zh }}
          onChange={(next) => setDraft({ ...draft, display_name_locs: next })}
          ariaLabel={t("nameLabel")}
        />
      </div>

      <div className="space-y-1">
        <Label>{t("enableLabel")}</Label>
        <Textarea
          className="font-mono text-xs min-h-20"
          placeholder="always = yes"
          value={draft.enable}
          onChange={(e) => setDraft({ ...draft, enable: e.target.value })}
        />
      </div>

      <div className="space-y-1">
        <Label>{t("removeTriggerLabel")}</Label>
        <Textarea
          className="font-mono text-xs min-h-20"
          placeholder="has_country_flag = some_flag"
          value={draft.remove_trigger}
          onChange={(e) =>
            setDraft({ ...draft, remove_trigger: e.target.value })
          }
        />
      </div>

      <div className="space-y-2">
        <div className="flex items-center justify-between">
          <Label>
            Modifier scalar fields · vanilla key = value(political_power_factor
            / stability_factor / ...)
          </Label>
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
              placeholder="political_power_factor"
              className="font-mono text-xs"
              value={p.key}
              onChange={(e) => {
                const next = [...draft.modifierPairs];
                next[i] = { ...next[i], key: e.target.value };
                setDraft({ ...draft, modifierPairs: next });
              }}
            />
            <Input
              placeholder="0.10"
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
              onClick={() => {
                const next = draft.modifierPairs.filter((_, j) => j !== i);
                setDraft({ ...draft, modifierPairs: next });
              }}
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
          {`effect = {
  add_dynamic_modifier = {
    modifier = ${draft.dm_id}
    days = 90      # ${t("usageComment")}
  }
}`}
        </pre>
      </details>
    </div>
  );
}
