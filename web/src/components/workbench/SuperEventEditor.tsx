"use client";

import { useState } from "react";
import { useTranslations } from "use-intl";
import { ArrowLeft } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Separator } from "@/components/ui/separator";
import {
  superEventsApi,
  type SuperEventRow,
  type SuperEventOption,
} from "@/lib/api";
import { useProjectStore } from "@/lib/store/projectStore";
import { DialogSaveStatusBar } from "@/components/workbench/DialogSaveStatusBar";
import { DisplayNameLocsTabs } from "@/components/shared/editors/DisplayNameLocsTabs";
import { DescriptionLocsTabs } from "@/components/shared/editors/DescriptionLocsTabs";
import { DEFAULT_EXPORT_LANGUAGES } from "@/components/shared/editors/displayNameLocsHelpers";
import {
  EFFECT_TEMPLATES,
  ScriptBlockEditor,
} from "@/components/shared/editors";
import { parseSuperEventAiChance } from "./superEventAiChance";

export const blankSuperEventDraft = (number: number) => ({
  number,
  country_tag: "",
  title_en: "",
  title_zh: "",
  desc_en: "",
  desc_zh: "",
  quote_en: "",
  quote_zh: "",
  btn_en: "",
  btn_zh: "",

  display_name_locs: {} as Record<string, string>,

  description_locs: {} as Record<string, string>,
  quote_locs: {} as Record<string, string>,
  button_locs: {} as Record<string, string>,
  song_token: "wow_starsstripes",
  auto_fire_on_startup: true,
  auto_fire_after_days: 1,
  options: [] as SuperEventOption[],
});

export type SuperEventDraft = ReturnType<typeof blankSuperEventDraft>;

export function superEventToDraft(row: SuperEventRow): SuperEventDraft {
  return {
    number: row.number,
    country_tag: row.country_tag,
    title_en: row.title_en,
    title_zh: row.title_zh,
    desc_en: row.desc_en,
    desc_zh: row.desc_zh,
    quote_en: row.quote_en,
    quote_zh: row.quote_zh,
    btn_en: row.btn_en,
    btn_zh: row.btn_zh,
    display_name_locs: row.display_name_locs ?? {},
    description_locs: row.description_locs ?? {},
    quote_locs: row.quote_locs ?? {},
    button_locs: row.button_locs ?? {},
    song_token: row.song_token,
    auto_fire_on_startup: row.auto_fire_on_startup,
    auto_fire_after_days: row.auto_fire_after_days,
    options: Array.isArray(row.options) ? row.options : [],
  };
}

export function SuperEventEditor({
  projectId,
  identity,
  draft,
  onChange,
  row,
  pictureBust,
  saving,
  uploading,
  saved,
  error,
  onSave,
  onDelete,
  onUpload,
  onBack,
}: {
  projectId: number;

  identity: number | "new";
  draft: SuperEventDraft;
  onChange: <K extends keyof SuperEventDraft>(
    key: K,
    value: SuperEventDraft[K],
  ) => void;

  row: SuperEventRow | null;

  pictureBust: number;
  saving: boolean;

  uploading: boolean;

  saved: boolean;

  error: string | null;
  onSave: () => void;

  onDelete?: () => void;

  onUpload: (file: File) => void;

  onBack?: () => void;
}) {
  const t = useTranslations("superEventsDialog");

  const exportLanguages = useProjectStore(
    (s) => s.currentProject?.export_languages ?? DEFAULT_EXPORT_LANGUAGES,
  );

  const [optionKeys, setOptionKeys] = useState<{
    superEventKey: string;
    next: number;
    keys: string[];
  }>(() => ({ superEventKey: String(identity), next: 0, keys: [] }));
  const superEventIdentity = String(identity);
  let shownOptionKeys = optionKeys;
  if (shownOptionKeys.superEventKey !== superEventIdentity) {
    shownOptionKeys = {
      superEventKey: superEventIdentity,
      next: 0,
      keys: [],
    };
  }
  if (shownOptionKeys.keys.length < draft.options.length) {
    const keys = [...shownOptionKeys.keys];
    let next = shownOptionKeys.next;
    while (keys.length < draft.options.length) {
      keys.push(`super-event:${superEventIdentity}:option:${next++}`);
    }
    shownOptionKeys = { ...shownOptionKeys, next, keys };
  }
  if (shownOptionKeys !== optionKeys) setOptionKeys(shownOptionKeys);

  return (
    <div className="p-4 space-y-3">
      {onBack && (
        <button
          type="button"
          onClick={onBack}
          className="ui-interactive inline-flex items-center gap-1 text-xs text-muted-foreground hover:text-foreground"
        >
          <ArrowLeft className="h-3.5 w-3.5" />
          {t("backFromNew")}
        </button>
      )}
      <div className="grid grid-cols-2 gap-3">
        <div className="space-y-1">
          <Label>{t("numberLabel")}</Label>
          <Input
            type="number"
            min={1}
            max={9999}
            value={draft.number}
            onChange={(e) => onChange("number", Number(e.target.value))}
          />
        </div>
        <div className="space-y-1">
          <Label>{t("tagLabel")}</Label>
          <Input
            value={draft.country_tag}
            onChange={(e) =>
              onChange("country_tag", e.target.value.toUpperCase())
            }
            placeholder="USA / GER / ..."
          />
        </div>
      </div>

      <Separator />

      <div className="space-y-2">
        <Label className="text-sm font-semibold">{t("segmentsTitle")}</Label>

        <div className="grid grid-cols-[120px_1fr] gap-2 items-start">
          <Label className="pt-1.5 text-muted-foreground">
            {t("segTitle")}
          </Label>
          <DisplayNameLocsTabs
            entityKey={`super-event:${identity}:display-name`}
            value={draft.display_name_locs}
            exportLanguages={exportLanguages}
            legacyFallback={{
              en: draft.title_en,
              zh: draft.title_zh,
            }}
            onChange={(next) => onChange("display_name_locs", next)}
            placeholder={t("seTitlePlaceholder")}
            ariaLabel={t("seTitlePlaceholder")}
          />
        </div>
        <div className="grid grid-cols-[120px_1fr] gap-2 items-start">
          <Label className="pt-1.5 text-muted-foreground">{t("segDesc")}</Label>
          <DescriptionLocsTabs
            entityKey={`super-event:${identity}:description`}
            value={draft.description_locs}
            exportLanguages={exportLanguages}
            legacyFallback={{
              en: draft.desc_en,
              zh: draft.desc_zh,
            }}
            onChange={(next) => onChange("description_locs", next)}
            placeholder={t("seDescPlaceholder")}
            ariaLabel={t("seDescPlaceholder")}
            rows={3}
          />
        </div>
        <div className="grid grid-cols-[120px_1fr] gap-2 items-start">
          <Label className="pt-1.5 text-muted-foreground">
            {t("segQuote")}
          </Label>
          <DescriptionLocsTabs
            entityKey={`super-event:${identity}:quote`}
            value={draft.quote_locs}
            exportLanguages={exportLanguages}
            legacyFallback={{ en: draft.quote_en, zh: draft.quote_zh }}
            onChange={(next) => onChange("quote_locs", next)}
            placeholder={t("segQuote")}
            ariaLabel={t("segQuote")}
            rows={2}
          />
        </div>
        <div className="grid grid-cols-[120px_1fr] gap-2 items-start">
          <Label className="pt-1.5 text-muted-foreground">{t("segBtn")}</Label>
          <DisplayNameLocsTabs
            entityKey={`super-event:${identity}:button`}
            value={draft.button_locs}
            exportLanguages={exportLanguages}
            legacyFallback={{ en: draft.btn_en, zh: draft.btn_zh }}
            onChange={(next) => onChange("button_locs", next)}
            placeholder={t("segBtn")}
            ariaLabel={t("segBtn")}
          />
        </div>
      </div>

      <Separator />

      <div className="grid grid-cols-2 gap-3">
        <div className="space-y-1">
          <Label>{t("songLabel")}</Label>
          <Input
            value={draft.song_token}
            onChange={(e) => onChange("song_token", e.target.value)}
            placeholder="wow_starsstripes / soviet_anthem / ..."
          />
        </div>
        <div className="space-y-1">
          <Label>{t("delayLabel")}</Label>
          <Input
            type="number"
            min={0}
            value={draft.auto_fire_after_days}
            onChange={(e) =>
              onChange("auto_fire_after_days", Number(e.target.value))
            }
            disabled={!draft.auto_fire_on_startup}
          />
        </div>
      </div>

      <label className="flex items-center gap-2 text-sm cursor-pointer">
        <input
          type="checkbox"
          checked={draft.auto_fire_on_startup}
          onChange={(e) => onChange("auto_fire_on_startup", e.target.checked)}
          className="accent-accent h-4 w-4"
        />
        <span>{t("autoFireLabel")}</span>
      </label>

      {!draft.auto_fire_on_startup && (
        <p className="text-xs text-warning rounded border border-warning/40 bg-warning/10 px-2 py-1">
          {t.rich("dormantWarning", {
            tag: "<TAG>",
            number: draft.number,
            mono: (c) => <span className="font-mono">{c}</span>,
          })}
        </p>
      )}

      <Separator />

      <div className="space-y-2">
        <div className="flex items-center justify-between">
          <Label className="text-sm font-semibold">
            {t("optionChainLabel")}
          </Label>
          <Button
            size="sm"
            variant="outline"
            onClick={() =>
              onChange("options", [
                ...draft.options,
                {
                  name_en: "",
                  name_zh: "",
                  display_name_locs: {},
                  effect: "",
                  ai_chance: 1,
                },
              ])
            }
          >
            {t("addButton")}
          </Button>
        </div>
        {draft.options.length === 0 ? (
          <p className="text-xs text-muted-foreground">
            {t.rich("optionsEmpty", {
              snippet: " country_event = { id = usa_se.2 days = 30 }",
              mono: (c) => <span className="font-mono">{c}</span>,
            })}
          </p>
        ) : (
          <div className="space-y-2">
            {draft.options.map((opt, idx) => (
              <div
                key={shownOptionKeys.keys[idx]}
                className="rounded-md border p-2 bg-muted/10 space-y-2"
              >
                <div className="flex items-center justify-between">
                  <Label className="font-semibold">
                    {t("optionHeading", { num: idx + 1 })}
                  </Label>
                  <Button
                    size="sm"
                    variant="ghost"
                    onClick={() => {
                      setOptionKeys({
                        ...shownOptionKeys,
                        keys: shownOptionKeys.keys.filter((_, i) => i !== idx),
                      });
                      onChange(
                        "options",
                        draft.options.filter((_, i) => i !== idx),
                      );
                    }}
                  >
                    {t("delete")}
                  </Button>
                </div>
                <div className="grid grid-cols-[1fr_80px] gap-2 items-start">
                  <div className="space-y-1 min-w-0">
                    <Label className="text-muted-foreground">
                      {t("optBtnEnLabel")}
                    </Label>
                    <DisplayNameLocsTabs
                      entityKey={`${shownOptionKeys.keys[idx]}:display-name`}
                      value={opt.display_name_locs ?? {}}
                      exportLanguages={exportLanguages}
                      legacyFallback={{ en: opt.name_en, zh: opt.name_zh }}
                      onChange={(value) => {
                        const next = [...draft.options];
                        next[idx] = { ...opt, display_name_locs: value };
                        onChange("options", next);
                      }}
                      placeholder={t("optionNamePlaceholder")}
                      ariaLabel={t("optBtnEnLabel")}
                    />
                  </div>
                  <div className="space-y-1">
                    <Label className="text-muted-foreground">ai_chance</Label>
                    <Input
                      type="number"
                      step="any"
                      min={0}
                      value={opt.ai_chance ?? 1}
                      onChange={(e) => {
                        const next = [...draft.options];

                        next[idx] = {
                          ...opt,
                          ai_chance: parseSuperEventAiChance(e.target.value),
                        };
                        onChange("options", next);
                      }}
                      className="h-7 text-xs"
                    />
                  </div>
                </div>
                <ScriptBlockEditor
                  label={t("effectLabel")}
                  labelTooltip={t("effectHint")}
                  value={opt.effect}
                  onChange={(effect) => {
                    const next = [...draft.options];
                    next[idx] = { ...opt, effect };
                    onChange("options", next);
                  }}
                  placeholder="country_event = { id = usa_se.2 days = 30 }"
                  templates={EFFECT_TEMPLATES}
                  minHeight="min-h-12"
                />
              </div>
            ))}
          </div>
        )}
      </div>

      <Separator />

      {row && (
        <div className="space-y-2">
          <Label className="text-sm font-semibold">{t("pictureLabel")}</Label>
          <div className="flex items-center gap-3">
            <div className="border rounded bg-muted/40 p-1">
              {row.picture_path ? (
                <img
                  src={`${superEventsApi.picturePreviewUrl(projectId, row.id)}?bust=${pictureBust}`}
                  alt={t("pictureAlt", { number: row.number })}
                  width={230}
                  height={155}
                  className="object-cover"
                />
              ) : (
                <div
                  className="bg-muted text-muted-foreground text-xs flex items-center justify-center"
                  style={{ width: 230, height: 155 }}
                >
                  {t("notUploaded")}
                </div>
              )}
            </div>
            <label
              className={`inline-flex items-center justify-center rounded-md border px-3 h-8 text-xs cursor-pointer transition-colors ${
                uploading ? "opacity-50 pointer-events-none" : "hover:bg-muted"
              }`}
            >
              <input
                type="file"

                accept=".png,.jpg,.jpeg,.dds,.tga,image/png,image/jpeg"
                className="hidden"
                disabled={uploading}
                onChange={(e) => {
                  const file = e.target.files?.[0];
                  if (file) onUpload(file);
                  e.target.value = "";
                }}
              />
              {uploading ? t("uploading") : t("uploadPng")}
            </label>
          </div>
        </div>
      )}

      <Separator />

      <DialogSaveStatusBar
        saving={saving || uploading}
        saved={saved}
        error={error}
        savingLabel={t("saving")}
        savedLabel={t("saved")}
      />

      <div className="flex gap-2">
        <Button onClick={onSave} disabled={saving} size="sm">
          {saving ? t("saving") : identity === "new" ? t("create") : t("save")}
        </Button>
        {onDelete && (
          <Button
            onClick={onDelete}
            disabled={saving}
            size="sm"
            variant="destructive"
          >
            {t("delete")}
          </Button>
        )}
      </div>
    </div>
  );
}
