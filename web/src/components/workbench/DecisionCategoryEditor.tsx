"use client";

import { useState } from "react";
import { useTranslations } from "use-intl";
import { Eye } from "lucide-react";
import { useBaseModDepKeys, gfxUrl } from "@/lib/api/basemod";
import { decisionsApi } from "@/lib/api";
import { categoryIconSprite } from "@/app/[locale]/game-view/decisionEdit";
import { useContentLocaleStore } from "@/lib/store/contentLocaleStore";
import { resolveDisplayName } from "@/lib/i18n/localizedName";
import { ImageUpload } from "@/shared/upload/ImageUpload";
import { DECISION_ICON_SIZE } from "@/shared/upload/sizes";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import type { DecisionCategory } from "@/types";

export type DecisionCategoryForm = Pick<
  DecisionCategory,
  | "icon"
  | "picture"
  | "on_map_area"
  | "name_en"
  | "name_zh"
  | "desc_en"
  | "desc_zh"
  | "display_name_locs"
  | "description_locs"
  | "visibility"
>;

function formOf(category: DecisionCategory): DecisionCategoryForm {
  return {
    icon: category.icon || "",

    picture: category.picture || "",

    on_map_area: category.on_map_area || "",
    name_en: category.name_en || "",
    name_zh: category.name_zh || "",
    desc_en: category.desc_en || "",
    desc_zh: category.desc_zh || "",
    display_name_locs: category.display_name_locs || {},
    description_locs: category.description_locs || {},

    visibility: category.visibility || "visible",
  };
}

export function DecisionCategoryEditor({
  projectId,
  category,
  onSave,
}: {
  projectId: number;
  category: DecisionCategory;

  onSave: (data: DecisionCategoryForm) => Promise<void>;
}) {
  const t = useTranslations("decisions");
  const contentLocale = useContentLocaleStore((s) => s.locale);

  const baseModDepKeys = useBaseModDepKeys();
  const [form, setForm] = useState<DecisionCategoryForm>(() =>
    formOf(category),
  );
  const [saving, setSaving] = useState(false);
  const [saved, setSaved] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const identity = `${category.id}:${JSON.stringify(formOf(category))}`;
  const [seen, setSeen] = useState(identity);
  if (seen !== identity) {
    setSeen(identity);
    setForm(formOf(category));
    setSaved(false);
    setError(null);
  }

  const handleSave = async () => {
    setSaving(true);
    setSaved(false);
    setError(null);
    try {
      await onSave(form);
      setSaved(true);
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className="p-4 space-y-4 max-w-2xl mx-auto">
      <div>
        <h2 className="text-sm font-semibold mb-1">
          {t("categoryInfoHeading")}
        </h2>
        <p className="text-xs text-muted-foreground">
          {t("categoryInfoLocExportPrefix")}{" "}
          <span className="font-mono">{"{TAG}_decisions_l_<lang>.yml"}</span>
        </p>
      </div>

      <div className="space-y-2">
        <Label>{t("categoryIdReadOnlyLabel")}</Label>
        <Input
          value={category.category_id}
          disabled
          className="font-mono text-xs"
        />
      </div>

      <div className="space-y-2">
        <Label>{t("categoryIconLabel")}</Label>
        <Input
          value={form.icon}
          onChange={(e) => setForm((f) => ({ ...f, icon: e.target.value }))}
          className="font-mono text-xs"
          placeholder="GFX_decision_category_generic (or upload below)"
        />
        <ImageUpload
          label={t("categoryIconUploadLabel")}
          targetSize={DECISION_ICON_SIZE}
          previewUrl={
            form.icon
              ? decisionsApi.categoryIconUrl(projectId, category.id)
              : null
          }
          gfxName={form.icon || undefined}
          baseModFallbackUrls={
            form.icon && baseModDepKeys.length
              ? baseModDepKeys.map((dk) =>
                  gfxUrl(projectId, dk, categoryIconSprite(form.icon)),
                )
              : []
          }
          hint={t("categoryIconUploadHint")}
          onUpload={async (file, crop) =>
            decisionsApi.uploadCategoryIcon(projectId, category.id, file, crop)
          }
          onUploaded={(result) => {
            const icon = result["icon"];
            if (typeof icon === "string") {
              setForm((f) => ({ ...f, icon }));
            }
          }}
        />
      </div>

      <div className="space-y-2">
        <Label>{t("categoryPictureLabel")}</Label>
        <Input
          value={form.picture}
          onChange={(e) => setForm((f) => ({ ...f, picture: e.target.value }))}
          className="font-mono text-xs"
          placeholder="GFX_decision_cat_pic_political_actions"
        />
        <p className="text-xs text-muted-foreground">
          {t("categoryPictureHint")}
        </p>
      </div>

      <div className="space-y-2">
        <Label>{t("categoryMapAreaLabel")}</Label>
        <Textarea
          value={form.on_map_area}
          onChange={(e) =>
            setForm((f) => ({ ...f, on_map_area: e.target.value }))
          }
          className="font-mono text-xs min-h-16"
          placeholder={"state = 123\nname = my_map_area_key\nzoom = 650"}
        />
        <p className="text-xs text-muted-foreground">
          {t("categoryMapAreaHint")}
        </p>
      </div>

      <div className="space-y-2">
        <h3 className="text-sm font-semibold text-muted-foreground uppercase tracking-wide">
          {t("visibilityHeading")}
        </h3>
        <div className="flex items-center gap-2">
          {(["visible", "solo", "hidden"] as const).map((mode) => {
            const active = form.visibility === mode;
            const label =
              mode === "visible" ? (
                <>
                  <Eye className="h-3 w-3" aria-hidden />
                  {t("visShow")}
                </>
              ) : mode === "solo" ? (
                t("visSolo")
              ) : (
                t("visHide")
              );
            const title =
              mode === "visible"
                ? t("visVisibleTip")
                : mode === "solo"
                  ? t("visSoloTip")
                  : t("visHiddenTip");
            return (
              <button
                key={mode}
                type="button"
                onClick={() => setForm((f) => ({ ...f, visibility: mode }))}
                className={
                  "inline-flex h-7 shrink-0 items-center gap-1 whitespace-nowrap rounded-md border px-2 text-xs transition-colors " +
                  (active
                    ? mode === "hidden"
                      ? "bg-muted border-border text-muted-foreground"
                      : mode === "solo"
                        ? "bg-accent/15 border-accent text-accent font-medium"
                        : "bg-success/15 border-success/60 text-success"
                    : "border-border bg-card hover:bg-muted")
                }
                title={title}
              >
                {label}
              </button>
            );
          })}
        </div>
        <p className="text-xs text-muted-foreground">{t("hiddenCatHint")}</p>
      </div>

      <div className="space-y-3">
        <h3 className="text-sm font-semibold text-muted-foreground uppercase tracking-wide">
          {t("localizationHeading")}
        </h3>

        <div className="space-y-2">
          <Label>
            {t("contentNameLabel", { locale: contentLocale.toUpperCase() })}
          </Label>
          <Input
            value={resolveDisplayName(
              contentLocale,
              form.display_name_locs,
              form.name_en,
              form.name_zh,
              "",
            )}
            onChange={(e) =>
              setForm((f) => ({
                ...f,
                display_name_locs: {
                  ...f.display_name_locs,
                  [contentLocale]: e.target.value,
                },
              }))
            }
          />
        </div>
        <div className="space-y-2">
          <Label>
            {t("contentDescriptionLabel", {
              locale: contentLocale.toUpperCase(),
            })}
          </Label>
          <Textarea
            value={resolveDisplayName(
              contentLocale,
              form.description_locs,
              form.desc_en,
              form.desc_zh,
              "",
            )}
            onChange={(e) =>
              setForm((f) => ({
                ...f,
                description_locs: {
                  ...f.description_locs,
                  [contentLocale]: e.target.value,
                },
              }))
            }
            className="text-xs min-h-16"
          />
        </div>
      </div>

      <div className="flex items-center gap-2">
        <Button onClick={handleSave} disabled={saving} size="sm">
          {saving ? t("saving") : t("saveCategory")}
        </Button>
        {saved && <span className="text-xs text-success">{t("saved")}</span>}
        {error && (
          <span role="alert" className="text-xs text-destructive">
            {error}
          </span>
        )}
      </div>
    </div>
  );
}
