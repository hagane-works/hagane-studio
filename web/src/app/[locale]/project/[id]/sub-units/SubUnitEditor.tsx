"use client";

import { useMemo, useState } from "react";
import { useTranslations } from "use-intl";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { DisplayNameLocsTabs } from "@/components/shared/editors/DisplayNameLocsTabs";
import { DEFAULT_EXPORT_LANGUAGES } from "@/components/shared/editors/displayNameLocsHelpers";
import { useRegisterDirty } from "@/lib/hooks/useUnsavedGuard";
import { useProjectStore } from "@/lib/store/projectStore";
import type { SubUnit } from "@/types";
import {
  SUB_UNIT_BOOLEAN_FIELDS,
  SUB_UNIT_INTERNAL_TYPES,
  buildSubUnitUpdate,
  getSubUnitDraftIssue,
  subUnitToDraft,
  type SubUnitBooleanKey,
  type SubUnitDraft,
  type SubUnitJsonKey,
  type TriState,
} from "./subUnitForm";

const BOOLEAN_LABELS: Record<SubUnitBooleanKey, string> = {
  active: "active",
  cavalry: "cavalry",
  special_forces: "specialForces",
  marines: "marines",
  can_exfiltrate_from_coast: "canExfiltrateFromCoast",
  mountaineers: "mountaineers",
  can_be_parachuted: "canBeParachuted",
  affects_speed: "affectsSpeed",
};

const JSON_LABELS: Record<SubUnitJsonKey, string> = {
  need: "need",
  stats: "stats",
  terrain_modifiers: "terrainModifiers",
  raw_extras: "rawExtras",
};

export function SubUnitEditor({
  subUnit,
  onSave,
  error = null,
}: {
  subUnit: SubUnit;

  onSave: (update: Partial<SubUnit>) => Promise<void>;

  error?: string | null;
}) {
  const t = useTranslations("subUnitEditor");
  const exportLanguages = useProjectStore(
    (state) =>
      state.currentProject?.export_languages ?? DEFAULT_EXPORT_LANGUAGES,
  );

  const [draft, setDraft] = useState<SubUnitDraft>(() =>
    subUnitToDraft(subUnit),
  );
  const [savedDraft, setSavedDraft] = useState<SubUnitDraft>(draft);
  const [saving, setSaving] = useState(false);
  const [saveStatus, setSaveStatus] = useState<"idle" | "success" | "error">(
    "idle",
  );
  const [saveError, setSaveError] = useState<string | null>(null);

  const isUnsaved = useMemo(
    () => JSON.stringify(draft) !== JSON.stringify(savedDraft),
    [draft, savedDraft],
  );
  const draftIssue = useMemo(() => getSubUnitDraftIssue(draft), [draft]);
  const draftIssueMessage = draftIssue
    ? t(
        draftIssue.kind === "booleanField"
          ? "unitTypeBooleanError"
          : draftIssue.kind === "unsupported"
            ? "unitTypeUnsupportedError"
            : "categoryTokenError",
        { token: draftIssue.token },
      )
    : null;
  useRegisterDirty("sub-units", isUnsaved);

  const save = async () => {
    if (draftIssueMessage) {
      setSaveError(draftIssueMessage);
      setSaveStatus("error");
      return;
    }
    setSaving(true);
    setSaveStatus("idle");
    setSaveError(null);
    try {
      await onSave(buildSubUnitUpdate(subUnit, draft));
      setSavedDraft(structuredClone(draft));
      setSaveStatus("success");
      window.setTimeout(() => setSaveStatus("idle"), 2000);
    } catch (caught) {
      const message =
        caught instanceof SyntaxError ||
        (caught instanceof Error &&
          caught.message.includes("must be a JSON object"))
          ? t("invalidJson")
          : caught instanceof Error
            ? caught.message
            : String(caught);
      setSaveError(message);
      setSaveStatus("error");
    } finally {
      setSaving(false);
    }
  };

  const setField = <K extends keyof SubUnitDraft>(
    field: K,
    value: SubUnitDraft[K],
  ) => {
    setDraft({ ...draft, [field]: value });
  };

  const renderTriState = (field: SubUnitBooleanKey) => {
    const value = draft.booleans[field];
    return (
      <div key={field} className="space-y-1">
        <Label className="text-xs">{t(BOOLEAN_LABELS[field])}</Label>
        <div className="flex gap-1">
          {(["unset", "yes", "no"] as TriState[]).map((option) => (
            <button
              key={option}
              type="button"
              onClick={() =>
                setDraft({
                  ...draft,
                  booleans: { ...draft.booleans, [field]: option },
                })
              }
              className={`h-7 px-2 rounded-md border text-xs ui-interactive ui-focus-ring ${
                value === option
                  ? "border-accent bg-accent text-accent-foreground font-medium"
                  : "border-border text-muted-foreground hover:bg-muted"
              }`}
            >
              {t(
                option === "unset"
                  ? "triUnset"
                  : option === "yes"
                    ? "triYes"
                    : "triNo",
              )}
            </button>
          ))}
        </div>
      </div>
    );
  };

  return (
    <div className="min-w-0 p-4 space-y-5">
      <section className="space-y-3">
        <h3 className="text-xs font-semibold uppercase tracking-wide text-muted-foreground">
          {t("identity")}
        </h3>
        <div className="grid grid-cols-2 gap-3">
          <div className="space-y-1">
            <Label className="text-xs">{t("id")}</Label>
            <Input
              className="font-mono text-xs"
              value={draft.sub_unit_id}
              onChange={(event) => setField("sub_unit_id", event.target.value)}
            />
          </div>
          <div className="space-y-1">
            <Label className="text-xs">{t("countryTag")}</Label>
            <Input
              className="font-mono text-xs uppercase"
              maxLength={3}
              value={draft.country_tag}
              onChange={(event) =>
                setField("country_tag", event.target.value.toUpperCase())
              }
            />
          </div>
          <div className="space-y-1 col-span-2">
            <Label className="text-xs">{t("sourceFile")}</Label>
            <Input
              className="font-mono text-xs"
              readOnly
              value={draft.source_file || t("sourceFileManagedValue")}
            />
            <p className="text-xs text-muted-foreground">
              {subUnit.source === "imported"
                ? t("sourceFileImportedHint")
                : t("sourceFileManagedHint")}
            </p>
          </div>
          <div className="space-y-1">
            <Label className="text-xs">{t("ordering")}</Label>
            <Input
              type="number"
              className="font-mono text-xs"
              value={draft.ordering}
              onChange={(event) => setField("ordering", event.target.value)}
            />
          </div>
          <div className="space-y-1">
            <Label className="text-xs">{t("abbreviation")}</Label>
            <Input
              className="font-mono text-xs"
              value={draft.abbreviation}
              onChange={(event) => setField("abbreviation", event.target.value)}
            />
          </div>
        </div>
        <div className="space-y-1">
          <Label className="text-xs">{t("displayName")}</Label>
          <DisplayNameLocsTabs
            entityKey={`sub-unit:${subUnit.id}:display-name`}
            value={draft.display_name_locs}
            exportLanguages={exportLanguages}
            legacyFallback={{ en: draft.name_en, zh: draft.name_zh }}
            onChange={(value) => setField("display_name_locs", value)}
            ariaLabel={t("displayName")}
          />
        </div>
      </section>

      <section className="space-y-3">
        <h3 className="text-xs font-semibold uppercase tracking-wide text-muted-foreground">
          {t("classification")}
        </h3>
        <div className="grid grid-cols-2 gap-3">
          {(["sprite", "map_icon_category", "transport", "group"] as const).map(
            (field) => (
              <div key={field} className="space-y-1">
                <Label className="text-xs">
                  {t(field === "map_icon_category" ? "mapIconCategory" : field)}
                </Label>
                <Input
                  className="font-mono text-xs"
                  value={draft[field]}
                  onChange={(event) => setField(field, event.target.value)}
                />
              </div>
            ),
          )}
          <div className="space-y-1">
            <Label className="text-xs">{t("priority")}</Label>
            <Input
              type="number"
              className="font-mono text-xs"
              value={draft.priority}
              onChange={(event) => setField("priority", event.target.value)}
            />
          </div>
          <div className="space-y-1">
            <Label className="text-xs">{t("aiPriority")}</Label>
            <Input
              type="number"
              className="font-mono text-xs"
              value={draft.ai_priority}
              onChange={(event) => setField("ai_priority", event.target.value)}
            />
          </div>
        </div>
        <div className="grid grid-cols-2 gap-3">
          {SUB_UNIT_BOOLEAN_FIELDS.map(renderTriState)}
        </div>
      </section>

      <section className="space-y-3">
        <h3 className="text-xs font-semibold uppercase tracking-wide text-muted-foreground">
          {t("tokenLists")}
        </h3>
        <div className="space-y-1">
          <Label className="text-xs">{t("unitTypes")}</Label>
          <Input
            aria-invalid={draftIssue?.field === "unit_types"}
            aria-describedby="sub-unit-types-help"
            className="font-mono text-xs"
            value={draft.unit_types}
            onChange={(event) => setField("unit_types", event.target.value)}
          />
          <p id="sub-unit-types-help" className="text-xs text-muted-foreground">
            {t("unitTypesHint", { types: SUB_UNIT_INTERNAL_TYPES.join(" · ") })}
          </p>
          {draftIssue?.field === "unit_types" && (
            <p className="text-xs text-destructive" role="alert">
              {draftIssueMessage}
            </p>
          )}
        </div>
        <div className="space-y-1">
          <Label className="text-xs">{t("categories")}</Label>
          <Input
            aria-invalid={draftIssue?.field === "categories"}
            className="font-mono text-xs"
            value={draft.categories}
            onChange={(event) => setField("categories", event.target.value)}
          />
          <p className="text-xs text-muted-foreground">{t("categoriesHint")}</p>
          {draftIssue?.field === "categories" && (
            <p className="text-xs text-destructive" role="alert">
              {draftIssueMessage}
            </p>
          )}
        </div>
        <div className="space-y-1">
          <Label className="text-xs">{t("essential")}</Label>
          <Input
            className="font-mono text-xs"
            value={draft.essential}
            onChange={(event) => setField("essential", event.target.value)}
          />
        </div>
      </section>

      <section className="space-y-3">
        <h3 className="text-xs font-semibold uppercase tracking-wide text-muted-foreground">
          {t("advanced")}
        </h3>
        <div className="grid grid-cols-2 gap-3">
          {(Object.keys(JSON_LABELS) as SubUnitJsonKey[]).map((field) => (
            <div key={field} className="space-y-1">
              <Label className="text-xs">{t(JSON_LABELS[field])}</Label>
              <Textarea
                className="min-h-36 font-mono text-xs"
                value={draft.json[field]}
                onChange={(event) =>
                  setDraft({
                    ...draft,
                    json: { ...draft.json, [field]: event.target.value },
                  })
                }
              />
            </div>
          ))}
        </div>
      </section>

      {(saving || saveStatus !== "idle" || isUnsaved) && (
        <div
          className={`-mx-4 px-4 py-2 border-y text-xs font-medium transition-[opacity,background-color,border-color] duration-300 ${
            saving
              ? "border-border bg-muted text-muted-foreground"
              : saveStatus === "success"
                ? "border-success/20 bg-success/10 text-success"
                : saveStatus === "error"
                  ? "border-destructive/20 bg-destructive/10 text-destructive"
                  : "border-warning/20 bg-warning/10 text-warning"
          }`}
        >
          {saving
            ? t("saving")
            : saveStatus === "success"
              ? t("saved")
              : saveStatus === "error"
                ? (saveError ?? t("saveFailed"))
                : t("unsaved")}
        </div>
      )}
      {error && saveStatus !== "error" && (
        <p className="text-xs text-destructive" role="alert">
          {error}
        </p>
      )}
      <div className="pt-2 border-t border-border">
        <Button
          size="sm"
          onClick={save}
          disabled={saving || !isUnsaved || Boolean(draftIssue)}
        >
          {t("save")}
        </Button>
      </div>
    </div>
  );
}
