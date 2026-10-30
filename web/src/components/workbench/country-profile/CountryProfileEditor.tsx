"use client";

import {
  useCallback,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from "react";
import { Flag, MapPinned, Palette, Trash2 } from "lucide-react";
import { useTranslations } from "use-intl";

import { DisplayNameLocsTabs } from "@/components/shared/editors/DisplayNameLocsTabs";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select } from "@/components/ui/select";
import {
  GRAPHICAL_CULTURES,
  deriveGraphicalCulture2d,
  hexToRgb,
  rgbToHex,
} from "@/lib/graphicalCultures";
import {
  countriesApi,
  historyFilesApi,
  type CountryRow,
  type HistoryFileRow,
} from "@/lib/api";
import { useRegisterDirty } from "@/lib/hooks/useUnsavedGuard";
import type { IdeologyOption } from "@/lib/ideology/registry";

import { DeleteCountryDialog } from "@/components/workbench/DeleteCountryDialog";

import { CountryFlagPreview } from "./CountryFlagPreview";
import {
  countryProfilePatch,
  countryToProfileDraft,
  legacyIdeologyCountryName,
  profileCountryPatchMatches,
  visibleCountryName,
  type CountryProfileDraft,
  type EffectiveCountryProfile,
} from "./model";

const FALLBACK_MAP_COLOR = rgbToHex([111, 102, 88]);

export interface CountryProfileRows {
  country: CountryRow;

  history: HistoryFileRow | null;
}

export interface CountryProfileWrite {
  country: Partial<CountryRow>;
  capital?: number;
}

export function countryProfileWrite(
  draft: CountryProfileDraft,
): Partial<CountryRow> {
  return {
    name_en: draft.nameEn,
    name_zh: draft.nameZh,
    display_name_locs: draft.displayNameLocs,
    adjective_en: draft.adjectiveEn,
    adjective_zh: draft.adjectiveZh,
    adjective_locs: draft.adjectiveLocs,
    ideology_name_locs: draft.ideologyNameLocs,
    color: draft.color,
    color_ui: draft.colorUi,
    graphical_culture: draft.graphicalCulture,
    graphical_culture_2d: deriveGraphicalCulture2d(draft.graphicalCulture),
  };
}

export interface CountryProfileEditorProps {
  projectId: number;

  country: CountryRow;
  roots: readonly IdeologyOption[];
  exportLanguages: string[];
  contentLocale: string;
  baseModDepKey: string | null;

  effectiveProfile?: EffectiveCountryProfile | null;

  onSave: (
    write: CountryProfileWrite,
    rows: CountryProfileRows,
  ) => Promise<HistoryFileRow | null>;

  onCountryReadback?: (country: CountryRow) => void;

  onProfileSaved?: (rows: {
    country: CountryRow;
    history: HistoryFileRow | null;
  }) => void;

  onFlagUploaded?: (root: string | undefined) => void;
  onOpenPolitics?: () => void;

  headerAction?: ReactNode;

  embedded?: boolean;
}

export function CountryProfileEditor({
  projectId,
  country,
  roots,
  exportLanguages,
  contentLocale,
  baseModDepKey,
  effectiveProfile,
  onSave,
  onCountryReadback,
  onProfileSaved,
  onFlagUploaded,
  onOpenPolitics,
  headerAction,
  embedded = false,
}: CountryProfileEditorProps) {
  const t = useTranslations("countrySettingsDialog.workbench.profile");
  const tDelete = useTranslations("countrySwitcher.deleteCountry");
  const [deleteOpen, setDeleteOpen] = useState(false);
  const [canonicalCountry, setCanonicalCountry] = useState(country);
  const [history, setHistory] = useState<HistoryFileRow | null>(null);
  const [draft, setDraft] = useState<CountryProfileDraft>(() =>
    countryToProfileDraft(country, null, effectiveProfile),
  );
  const [savedDraft, setSavedDraft] = useState<CountryProfileDraft>(() =>
    countryToProfileDraft(country, null, effectiveProfile),
  );
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [uploading, setUploading] = useState<string | null>(null);
  const [status, setStatus] = useState<"" | "saved">("");
  const [error, setError] = useState<string | null>(null);
  const [flagBust, setFlagBust] = useState(0);
  const [previewRoot, setPreviewRoot] = useState(roots[0]?.value ?? "");

  const rootIds = useMemo(() => roots.map((root) => root.value), [roots]);

  const loadHistory = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const histories = await historyFilesApi.list(projectId);
      const nextHistory =
        histories.find(
          (row) => row.country_tag.toUpperCase() === country.tag.toUpperCase(),
        ) ?? null;
      const nextDraft = countryToProfileDraft(
        country,
        nextHistory,
        effectiveProfile,
      );
      setCanonicalCountry(country);
      setHistory(nextHistory);
      setDraft(nextDraft);
      setSavedDraft(nextDraft);
    } catch (cause) {
      setError((cause as Error).message);
    } finally {
      setLoading(false);
    }
  }, [country, effectiveProfile, projectId]);

  useEffect(() => {
    void loadHistory();
  }, [loadHistory]);

  useEffect(() => {
    if (!rootIds.includes(previewRoot)) setPreviewRoot(rootIds[0] ?? "");
  }, [previewRoot, rootIds]);

  const isDirty = useMemo(
    () => JSON.stringify(draft) !== JSON.stringify(savedDraft),
    [draft, savedDraft],
  );
  useRegisterDirty("country-profile", isDirty);

  const updateDraft = <K extends keyof CountryProfileDraft>(
    key: K,
    value: CountryProfileDraft[K],
  ) => setDraft((current) => ({ ...current, [key]: value }));

  const save = async () => {
    setSaving(true);
    setError(null);
    setStatus("");
    try {
      const countryPatch = countryProfilePatch(
        canonicalCountry,
        savedDraft,
        draft,
      );
      const capitalPatch =
        draft.capital !== savedDraft.capital ? draft.capital : undefined;
      const historySaved = await onSave(
        { country: countryPatch, capital: capitalPatch },
        { country: canonicalCountry, history },
      );

      const [countryList, historyReadback] = await Promise.all([
        countriesApi.list(projectId),
        historySaved
          ? historyFilesApi.get(projectId, historySaved.id)
          : Promise.resolve(null),
      ]);
      const countryReadback = countryList.countries.find(
        (item) => item.id === canonicalCountry.id,
      );
      if (
        !countryReadback ||
        !profileCountryPatchMatches(countryReadback, countryPatch) ||
        (capitalPatch !== undefined &&
          historyReadback?.capital !== capitalPatch)
      ) {
        throw new Error(t("readbackMismatch"));
      }
      setCanonicalCountry(countryReadback);
      setHistory(historyReadback);
      const nextDraft = countryToProfileDraft(
        countryReadback,
        historyReadback,
        effectiveProfile,
      );
      setDraft(nextDraft);
      setSavedDraft(nextDraft);
      onCountryReadback?.(countryReadback);
      onProfileSaved?.({ country: countryReadback, history: historyReadback });
      setStatus("saved");
    } catch (cause) {
      setError((cause as Error).message);
    } finally {
      setSaving(false);
    }
  };

  const commitFlag = async (file: File, root?: string) => {
    await countriesApi.uploadFlag(projectId, canonicalCountry.id, file, root);
    const countryList = await countriesApi.list(projectId);
    const readback = countryList.countries.find(
      (item) => item.id === canonicalCountry.id,
    );
    if (
      !readback ||
      (root
        ? !readback.flag_paths_by_ideology?.[root]
        : !readback.flag_image_path)
    ) {
      throw new Error(t("flagReadbackMismatch"));
    }
    setCanonicalCountry(readback);
    onCountryReadback?.(readback);
    onFlagUploaded?.(root);
    setFlagBust((value) => value + 1);
    setStatus("saved");
  };

  const uploadFlag = async (file: File, root?: string) => {
    setUploading(root ?? "default");
    setError(null);
    setStatus("");
    try {
      await commitFlag(file, root);
    } catch (cause) {
      setError((cause as Error).message);
    } finally {
      setUploading(null);
    }
  };

  const previewName = visibleCountryName(
    canonicalCountry,
    draft,
    previewRoot,
    contentLocale,
  );
  const previewAdjective =
    draft.adjectiveLocs[contentLocale] ||
    (contentLocale === "zh" ? draft.adjectiveZh : draft.adjectiveEn);
  const mapColor =
    draft.color.length === 3 ? rgbToHex(draft.color) : FALLBACK_MAP_COLOR;

  if (loading) {
    return (
      <div className="animate-pulse-ring ui-panel p-6 text-sm">
        {t("loading")}
      </div>
    );
  }

  return (
    <div
      className={`grid min-w-0 gap-4 ${embedded ? "" : "xl:grid-cols-[minmax(0,1fr)_320px]"}`}
      data-testid="country-profile-workbench"
    >
      <div className="min-w-0 space-y-4">
        <section className="ui-panel p-5">
          <div className="mb-5 flex items-start justify-between gap-4">
            <div className="flex min-w-0 items-center gap-3">
              <Flag
                className="size-5 shrink-0 text-primary"
                aria-hidden="true"
              />
              <div className="min-w-0">
                <h2 className="text-lg font-semibold tracking-tight">
                  {t("title")}
                </h2>
                <p className="truncate text-xs text-muted-foreground">
                  {t("context", {
                    tag: canonicalCountry.tag,
                    name: canonicalCountry.name_en || canonicalCountry.name,
                  })}
                </p>
              </div>
              {headerAction}
            </div>
            <span className="rounded-md border border-border px-2 py-1 font-mono text-xs">
              {canonicalCountry.tag}
            </span>
          </div>

          <div
            className={`grid gap-5 ${embedded ? "grid-cols-[180px_minmax(0,1fr)]" : "lg:grid-cols-[180px_minmax(0,1fr)]"}`}
          >
            <div className="space-y-3">
              {embedded && effectiveProfile?.flag ? (
                <img
                  data-testid="country-profile-effective-flag"
                  src={`${effectiveProfile.flag.url}${effectiveProfile.flag.url.includes("?") ? "&" : "?"}v=${flagBust}`}
                  alt={`${canonicalCountry.tag} current effective flag`}
                  width={82}
                  height={52}
                  className="h-[52px] w-[82px] max-w-none rounded-md object-cover"
                />
              ) : (
                <CountryFlagPreview
                  projectId={projectId}
                  country={canonicalCountry}
                  roots={rootIds}
                  baseModDepKey={baseModDepKey}
                  bust={flagBust}
                  className="h-[52px] w-[82px] max-w-none rounded-md"
                />
              )}
              <Label className="ui-focus-ring flex cursor-pointer items-center justify-center rounded-md border border-input px-3 py-2 text-xs ui-interactive hover:bg-muted">
                {uploading === "default"
                  ? t("uploading")
                  : t("replaceDefaultFlag")}
                <input
                  type="file"
                  accept="image/png,image/jpeg,image/tga,.tga"
                  className="sr-only"
                  disabled={uploading !== null}
                  onChange={(event) => {
                    const file = event.target.files?.[0];
                    if (file) void uploadFlag(file);
                    event.target.value = "";
                  }}
                />
              </Label>
            </div>

            <div
              className={`grid gap-4 ${embedded ? "grid-cols-2" : "md:grid-cols-2"}`}
            >
              <div
                className={`space-y-1 ${embedded ? "col-span-2" : "md:col-span-2"}`}
              >
                <Label>{t("countryName")}</Label>
                <DisplayNameLocsTabs
                  entityKey={`${canonicalCountry.id}-name`}
                  value={draft.displayNameLocs}
                  onChange={(value) => updateDraft("displayNameLocs", value)}
                  exportLanguages={exportLanguages}
                  legacyFallback={{ en: draft.nameEn, zh: draft.nameZh }}
                  ariaLabel={t("countryName")}
                />
              </div>
              <div
                className={`space-y-1 ${embedded ? "col-span-2" : "md:col-span-2"}`}
              >
                <Label>{t("countryAdjective")}</Label>
                <DisplayNameLocsTabs
                  entityKey={`${canonicalCountry.id}-adjective`}
                  value={draft.adjectiveLocs}
                  onChange={(value) => updateDraft("adjectiveLocs", value)}
                  exportLanguages={exportLanguages}
                  legacyFallback={{
                    en: draft.adjectiveEn,
                    zh: draft.adjectiveZh,
                  }}
                  ariaLabel={t("countryAdjective")}
                />
              </div>
              <div className="space-y-1">
                <Label htmlFor="country-profile-capital">{t("capital")}</Label>
                <Input
                  id="country-profile-capital"
                  type="number"
                  min={0}
                  step={1}
                  value={draft.capital}
                  onChange={(event) =>
                    updateDraft("capital", Number(event.target.value))
                  }
                />
                <p className="text-micro text-muted-foreground">
                  {t("capitalCanonical")}
                </p>
              </div>
              <div className="space-y-1">
                <Label htmlFor="country-profile-color">{t("mapColor")}</Label>
                <div className="flex items-center gap-2">
                  <Input
                    id="country-profile-color"
                    type="color"
                    value={
                      draft.color.length === 3
                        ? rgbToHex(draft.color)
                        : FALLBACK_MAP_COLOR
                    }
                    className="w-14 p-1"
                    onChange={(event) =>
                      updateDraft("color", hexToRgb(event.target.value))
                    }
                  />
                  <code className="text-xs text-muted-foreground">
                    {mapColor.toUpperCase()}
                  </code>
                </div>
              </div>
            </div>
          </div>
        </section>

        <section className="ui-panel p-5">
          <div className="mb-4 flex items-center justify-between gap-3">
            <div>
              <h3 className="text-base font-semibold tracking-tight">
                {t("variants")}
              </h3>
              <p className="mt-1 text-xs text-muted-foreground">
                {t("variantsHint")}
              </p>
            </div>
            <span className="text-xs text-muted-foreground">
              {t("rootCount", { count: roots.length })}
            </span>
          </div>
          <div className="space-y-3">
            {roots.map((root) => {
              const hasVariantFlag = Boolean(
                canonicalCountry.flag_paths_by_ideology?.[root.value],
              );
              const hasVariantName = Boolean(
                Object.values(draft.ideologyNameLocs[root.value] ?? {}).some(
                  (value) => value.trim(),
                ),
              );
              return (
                <article
                  key={root.value}
                  className={`ui-card grid gap-4 p-4 ${
                    embedded
                      ? "grid-cols-[130px_minmax(0,1fr)_auto] items-center"
                      : "lg:grid-cols-[130px_minmax(0,1fr)_auto] lg:items-center"
                  }`}
                >
                  <div>
                    <CountryFlagPreview
                      projectId={projectId}
                      country={canonicalCountry}
                      root={root.value}
                      roots={rootIds}
                      baseModDepKey={baseModDepKey}
                      bust={flagBust}
                      className="h-[52px] w-[82px] max-w-none rounded-md"
                    />
                    <p className="mt-1 text-micro text-muted-foreground">
                      {hasVariantFlag
                        ? t("dedicatedFlag")
                        : t("genericFlagFallback")}
                    </p>
                  </div>
                  <div className="min-w-0 space-y-2">
                    <div className="flex items-baseline gap-2">
                      <h4 className="truncate text-sm font-semibold">
                        {root.label}
                      </h4>
                      <code className="text-micro text-muted-foreground">
                        {root.value}
                      </code>
                    </div>
                    <DisplayNameLocsTabs
                      entityKey={`${canonicalCountry.id}-${root.value}`}
                      value={draft.ideologyNameLocs[root.value] ?? {}}
                      legacyFallback={{
                        en: legacyIdeologyCountryName(
                          canonicalCountry,
                          root.value,
                          "en",
                        ),
                        zh: legacyIdeologyCountryName(
                          canonicalCountry,
                          root.value,
                          "zh",
                        ),
                      }}
                      onChange={(value) =>
                        updateDraft("ideologyNameLocs", {
                          ...draft.ideologyNameLocs,
                          [root.value]: value,
                        })
                      }
                      exportLanguages={exportLanguages}
                      placeholder={t("variantNamePlaceholder")}
                      ariaLabel={t("variantNameAria", { root: root.label })}
                    />
                    <p className="text-micro text-muted-foreground">
                      {hasVariantName
                        ? t("dedicatedName")
                        : t("genericNameFallback")}
                    </p>
                  </div>
                  <Label className="ui-focus-ring flex cursor-pointer items-center justify-center rounded-md border border-input px-3 py-2 text-xs ui-interactive hover:bg-muted">
                    {uploading === root.value
                      ? t("uploading")
                      : t("replaceVariantFlag")}
                    <input
                      type="file"
                      accept="image/png,image/jpeg,image/tga,.tga"
                      className="sr-only"
                      disabled={uploading !== null}
                      onChange={(event) => {
                        const file = event.target.files?.[0];
                        if (file) void uploadFlag(file, root.value);
                        event.target.value = "";
                      }}
                    />
                  </Label>
                </article>
              );
            })}
          </div>
        </section>

        <details className="ui-panel p-5">
          <summary className="cursor-pointer text-sm font-semibold">
            {t("advanced")}
          </summary>
          <div
            className={`mt-4 grid gap-4 ${embedded ? "grid-cols-2" : "md:grid-cols-2"}`}
          >
            <div className="space-y-1">
              <Label htmlFor="country-profile-ui-color">{t("uiColor")}</Label>
              <div className="flex items-center gap-2">
                <Input
                  id="country-profile-ui-color"
                  type="color"
                  value={
                    draft.colorUi.length === 3
                      ? rgbToHex(draft.colorUi)
                      : mapColor
                  }
                  className="w-14 p-1"
                  onChange={(event) =>
                    updateDraft("colorUi", hexToRgb(event.target.value))
                  }
                />
                <Button
                  type="button"
                  variant="ghost"
                  size="sm"
                  onClick={() => updateDraft("colorUi", [])}
                >
                  {t("useMapColor")}
                </Button>
              </div>
            </div>
            <div className="space-y-1">
              <Label htmlFor="country-profile-culture">
                {t("graphicalCulture")}
              </Label>
              <Select
                id="country-profile-culture"
                value={draft.graphicalCulture}
                onChange={(event) =>
                  updateDraft("graphicalCulture", event.target.value)
                }
              >
                <option value="">{t("cultureAuto")}</option>
                {GRAPHICAL_CULTURES.map((culture) => (
                  <option key={culture} value={culture}>
                    {culture}
                  </option>
                ))}
              </Select>
              <p className="text-micro text-muted-foreground">
                {deriveGraphicalCulture2d(draft.graphicalCulture) ||
                  t("cultureAuto")}
              </p>
            </div>
            <div
              className={`rounded-md border border-border bg-muted p-3 text-xs ${embedded ? "col-span-2" : "md:col-span-2"}`}
            >
              <p>
                <span className="text-muted-foreground">TAG · </span>
                <code>{canonicalCountry.tag}</code>
              </p>
              <p className="mt-1">
                <span className="text-muted-foreground">
                  {t("sourcePath")} ·{" "}
                </span>
                <code>common/countries/{canonicalCountry.tag}.txt</code>
              </p>
            </div>
          </div>
        </details>

        <div className="ui-panel flex items-center justify-between gap-4 p-4">
          <p className="text-xs text-muted-foreground">
            {t("identityNotPolitics")}
          </p>
          <div className="flex items-center gap-2">
            <Button
              type="button"
              variant="outline"
              className="text-destructive hover:text-destructive"
              onClick={() => setDeleteOpen(true)}
              disabled={saving}
            >
              <Trash2 aria-hidden />
              {tDelete("button")}
            </Button>
            {onOpenPolitics && (
              <Button type="button" variant="outline" onClick={onOpenPolitics}>
                {t("openPolitics")}
              </Button>
            )}
            <Button
              type="button"
              onClick={() => void save()}
              disabled={!isDirty || saving || uploading !== null}
            >
              {saving ? t("saving") : t("saveProfile")}
            </Button>
          </div>
        </div>
        <DeleteCountryDialog
          projectId={projectId}
          country={{
            id: canonicalCountry.id,
            tag: canonicalCountry.tag,
            name: canonicalCountry.name,
          }}
          open={deleteOpen}
          onOpenChange={setDeleteOpen}
        />
      </div>

      {!embedded && (
        <aside className="min-w-0">
          <div className="ui-panel sticky top-4 p-5">
            <div className="mb-4 flex items-center gap-2">
              <MapPinned className="size-5 text-primary" aria-hidden="true" />
              <div>
                <h3 className="text-base font-semibold tracking-tight">
                  {t("playerPreview")}
                </h3>
                <p className="text-micro text-muted-foreground">
                  {t("previewNotEngineProof")}
                </p>
              </div>
            </div>
            <div className="relative flex aspect-[4/3] items-center justify-center overflow-hidden rounded-md border border-border bg-muted">
              <div
                className="absolute inset-0 opacity-30"
                style={{ backgroundColor: mapColor }}
              />
              <span className="relative font-mono text-lg font-semibold tracking-wide">
                {canonicalCountry.tag}
              </span>
            </div>
            <div className="mt-4 flex gap-3">
              <CountryFlagPreview
                projectId={projectId}
                country={canonicalCountry}
                root={previewRoot}
                roots={rootIds}
                baseModDepKey={baseModDepKey}
                bust={flagBust}
                className="h-[52px] w-[82px] shrink-0 rounded object-cover"
              />
              <div className="min-w-0">
                <h4 className="truncate text-sm font-semibold">
                  {previewName.value}
                </h4>
                <p className="truncate text-xs text-muted-foreground">
                  {canonicalCountry.tag} · {previewAdjective}
                </p>
                <p className="text-micro text-muted-foreground">
                  {t("capitalPreview", { capital: draft.capital || "—" })}
                </p>
              </div>
            </div>
            <div className="mt-4 space-y-1">
              <Label htmlFor="country-profile-preview-root">
                {t("previewRoot")}
              </Label>
              <Select
                id="country-profile-preview-root"
                value={previewRoot}
                onChange={(event) => setPreviewRoot(event.target.value)}
              >
                {roots.map((root) => (
                  <option key={root.value} value={root.value}>
                    {root.label}
                  </option>
                ))}
              </Select>
              <p className="text-micro text-muted-foreground">
                {previewName.fallback
                  ? t("genericNameFallback")
                  : t("dedicatedName")}
              </p>
            </div>
            <div className="mt-4 space-y-2 border-t border-border pt-4 text-xs">
              <p className="flex items-center gap-2">
                <Palette
                  className="size-4 text-muted-foreground"
                  aria-hidden="true"
                />{" "}
                common/countries/{canonicalCountry.tag}.txt
              </p>
              <p className="flex items-center gap-2">
                <Flag
                  className="size-4 text-muted-foreground"
                  aria-hidden="true"
                />{" "}
                gfx/flags/{canonicalCountry.tag}_{previewRoot}.tga
              </p>
            </div>
          </div>
        </aside>
      )}

      {(status || error) && (
        <div
          role="status"
          aria-live="polite"
          className={`sticky bottom-0 z-10 rounded-md border p-3 text-sm ${embedded ? "" : "xl:col-span-2"} ${
            error
              ? "border-destructive/30 bg-destructive/10 text-destructive"
              : "border-success/30 bg-success/10 text-success"
          }`}
        >
          {error ?? t("savedReadback")}
        </div>
      )}
    </div>
  );
}
