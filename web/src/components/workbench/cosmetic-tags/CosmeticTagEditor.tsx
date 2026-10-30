"use client";

import { useEffect, useId, useState } from "react";
import { useLocale, useTranslations } from "use-intl";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { ContentLanguageSwitcher } from "@/components/i18n/ContentLanguageSwitcher";
import { useRegisterDirty } from "@/lib/hooks/useUnsavedGuard";
import {
  useContentLocaleStore,
  type ContentLocale,
} from "@/lib/store/contentLocaleStore";
import { useProjectStore } from "@/lib/store/projectStore";
import {
  COSMETIC_LOCALES,
  type CosmeticRGB,
  type CosmeticRawOverride,
  type CosmeticSuffix,
  type CosmeticTagView,
} from "@/lib/api/cosmeticTags";
import { useCosmeticTagDraft } from "./useCosmeticTagDraft";

interface Props {
  projectId: number;
  tag: string;
  countryTag?: string;
  countryNames?: Record<string, string>;
  onViewChange?: (view: CosmeticTagView) => void;
}

export function CosmeticTagEditor(props: Props) {
  const t = useTranslations("historyFilesDialog.cosmetic");
  if (!props.tag) return null;
  if (
    props.tag === "." ||
    props.tag === ".." ||
    !/^[A-Za-z0-9_.-]{1,100}$/.test(props.tag)
  ) {
    return (
      <p role="alert" className="text-xs text-destructive">
        {t("invalidTag")}
      </p>
    );
  }
  return (
    <CosmeticTagSession key={`${props.projectId}:${props.tag}`} {...props} />
  );
}

function CosmeticTagSession({
  projectId,
  tag,
  countryTag,
  countryNames = {},
  onViewChange,
}: Props) {
  const t = useTranslations("historyFilesDialog.cosmetic");
  const appLocale = useLocale();
  const locale = useContentLocaleStore((s) => s.locale);
  const header = COSMETIC_LOCALES[locale];
  const editor = useCosmeticTagDraft(projectId, tag, header);
  const id = useId();
  const [advanced, setAdvanced] = useState(false);
  const busy = editor.status === "saving";
  useRegisterDirty(
    `cosmetic-tag:${projectId}:${tag}`,
    editor.dirty || busy || editor.status === "readbackFailed",
  );
  const view = editor.view;
  useEffect(() => {
    if (view) onViewChange?.(view);
  }, [view, onViewChange]);
  const draft = editor.draft;
  const preview = editor.previewLocale === header ? editor.preview : null;
  const countryName = (country: string) =>
    countryNames[country]?.trim() || country;
  const usingCountries = [...new Set(view?.referenced_by ?? [])];
  const impact =
    usingCountries.length > 1
      ? t("sharedImpact", {
          countries: new Intl.ListFormat(appLocale, {
            style: "long",
            type: "conjunction",
          }).format(usingCountries.map(countryName)),
        })
      : usingCountries.length === 1
        ? t("singleImpact", { country: countryName(usingCountries[0]) })
        : t("unassignedImpact");

  const fieldLabel = (suffix: CosmeticSuffix): string => {
    if (suffix === "") return t("mainName");
    if (suffix === "ADJ") return t("genericAdjective");
    if (suffix === "DEF") return t("genericDefinite");
    const [group, form] = suffix.split("_");
    const ideology = t(`ideologies.${group}`);
    if (form === "ADJ") return t("ideologyAdjective", { ideology });
    if (form === "DEF") return t("ideologyDefinite", { ideology });
    return t("ideologyName", { ideology });
  };

  function nameField(suffix: CosmeticSuffix) {
    const inputId = `${id}-${suffix || "main"}`;
    const resolved = preview?.names[suffix];
    const imported = view?.imported_name_locs?.[header]?.[suffix];
    const saved = view?.name_locs?.[header]?.[suffix];
    return (
      <div key={suffix} className="space-y-1.5">
        <label htmlFor={inputId} className="text-sm font-medium">
          {fieldLabel(suffix)}
        </label>
        <Input
          id={inputId}
          value={draft?.name_locs[header]?.[suffix] ?? ""}
          placeholder={resolved?.value || t("inheritPlaceholder")}
          aria-describedby={`${inputId}-source`}
          className="ui-interactive"
          onChange={(event) =>
            editor.changeName(header, suffix, event.target.value)
          }
        />
        <div
          id={`${inputId}-source`}
          className="space-y-0.5 text-xs text-muted-foreground"
        >
          <p>
            {resolved
              ? t("effectiveValue", {
                  value: resolved.value || t("emptyValue"),
                })
              : t("previewPending")}
          </p>
          {editor.status === "review" && (
            <p>{t("savedValue", { value: saved ?? t("noOverride") })}</p>
          )}
        </div>
        <details className="text-xs text-muted-foreground">
          <summary className="cursor-pointer ui-focus-ring ui-interactive">
            {t("nameDetails")}
          </summary>
          <div className="mt-1 space-y-1">
            {resolved && (
              <p>
                {resolved.source === "authored"
                  ? resolved.inherited_from === suffix
                    ? t("authored")
                    : t("inherited", {
                        field: fieldLabel(
                          resolved.inherited_from as CosmeticSuffix,
                        ),
                      })
                  : resolved.source === "imported"
                    ? t("imported")
                    : t("generated")}
              </p>
            )}
            {imported !== undefined && (
              <p>
                {t("importedValue", { value: imported || t("emptyValue") })}
              </p>
            )}
          </div>
        </details>
      </div>
    );
  }

  return (
    <section
      aria-label={t("title")}
      className="space-y-4 rounded-lg border border-border bg-card p-4"
    >
      <div className="flex items-start justify-between gap-4">
        <div className="min-w-0 space-y-1">
          <h3 className="text-base font-semibold">{t("title")}</h3>
          {countryTag && (
            <p className="text-xs text-muted-foreground">
              {t("openedFrom", { country: countryName(countryTag) })}
            </p>
          )}
        </div>
        {view && <ContentLanguageSwitcher />}
      </div>

      {!view && (
        <div className="space-y-3 rounded-md bg-muted p-4">
          <p role="status" className="text-sm">
            {editor.status === "missing"
              ? t("notCreated")
              : editor.status === "failed"
                ? t("loadFailed")
                : t("loading")}
          </p>
          {editor.status === "missing" && (
            <Button
              type="button"
              size="sm"
              onClick={() => void editor.create()}
            >
              {t("create")}
            </Button>
          )}
          {editor.status === "failed" && (
            <Button
              type="button"
              variant="outline"
              size="sm"
              onClick={editor.retryLoad}
            >
              {t("retry")}
            </Button>
          )}
        </div>
      )}

      {view && draft && (
        <>
          <p className="rounded-md bg-muted p-3 text-sm" aria-live="polite">
            {impact}
          </p>
          <p className="text-xs text-muted-foreground">{t("clearHelp")}</p>
          {nameField("")}
          <Button
            type="button"
            variant="ghost"
            size="sm"
            aria-expanded={advanced}
            aria-controls={`${id}-advanced`}
            onClick={() => setAdvanced(!advanced)}
          >
            {advanced ? t("hideAdvanced") : t("advanced")}
          </Button>
          {advanced && (
            <div
              id={`${id}-advanced`}
              className="space-y-4 border-t border-border pt-4"
            >
              <div className="grid grid-cols-2 gap-4">
                {nameField("ADJ")}
                {nameField("DEF")}
              </div>
              {(
                ["democratic", "fascism", "neutrality", "communism"] as const
              ).map((group) => (
                <fieldset
                  key={group}
                  className="space-y-3 rounded-md border border-border p-3"
                >
                  <legend className="px-1 text-sm font-medium">
                    {t(`ideologies.${group}`)}
                  </legend>
                  {nameField(group)}
                  <div className="grid grid-cols-2 gap-4">
                    {nameField(`${group}_ADJ`)}
                    {nameField(`${group}_DEF`)}
                  </div>
                </fieldset>
              ))}
            </div>
          )}

          <div className="grid grid-cols-2 gap-4 border-t border-border pt-4">
            {(["color", "color_ui"] as const).map((property) => (
              <ColorField
                key={property}
                label={property === "color" ? t("mapColor") : t("uiColor")}
                value={draft[property]}
                resolved={preview?.[property]}
                saved={editor.status === "review" ? view[property] : undefined}
                imported={
                  property === "color"
                    ? view.imported_color
                    : view.imported_color_ui
                }
                onChange={(value) =>
                  editor.edit((current) => ({ ...current, [property]: value }))
                }
              />
            ))}
          </div>
          {editor.previewState === "failed" && (
            <div
              role="alert"
              className="flex items-center gap-2 text-xs text-destructive"
            >
              <span>{t("previewFailed")}</span>
              <Button
                type="button"
                variant="outline"
                size="sm"
                onClick={editor.retryPreview}
              >
                {t("retry")}
              </Button>
            </div>
          )}

          {(view.raw_overrides?.length ?? 0) > 0 && (
            <RawOverrideReview
              entries={view.raw_overrides ?? []}
              onCopy={async (entries) => {
                for (const language of new Set(
                  entries.map((entry) => entry.locale),
                )) {
                  const code = Object.entries(COSMETIC_LOCALES).find(
                    ([, value]) => value === language,
                  )?.[0] as ContentLocale | undefined;
                  const projectState = useProjectStore.getState();
                  if (!code || projectState.currentProject?.id !== projectId)
                    throw new Error("Project or content language changed");
                  if (
                    !projectState.currentProject.export_languages?.includes(
                      code,
                    )
                  ) {
                    await projectState.setContentLanguageEnabled(code, true);
                  }
                  if (
                    useProjectStore.getState().currentProject?.id !== projectId
                  )
                    throw new Error("Project changed");
                }
                editor.edit((current) => {
                  const names = structuredClone(current.name_locs);
                  for (const entry of entries) {
                    names[entry.locale] = { ...names[entry.locale] };
                    if (entry.value.trim() === "")
                      delete names[entry.locale][entry.suffix];
                    else names[entry.locale][entry.suffix] = entry.value;
                  }
                  return { ...current, name_locs: names };
                });
              }}
            />
          )}

          <details className="rounded-md border border-border p-3 text-xs text-muted-foreground">
            <summary className="cursor-pointer ui-focus-ring ui-interactive">
              {t("technicalDetails")}
            </summary>
            <div className="mt-2 space-y-1">
              <p className="break-all font-mono">
                {t("tagIdentifier", { tag })}
              </p>
              <p>{t("shared")}</p>
            </div>
          </details>

          <div className="sticky bottom-0 flex flex-wrap items-center justify-between gap-3 rounded-md border border-border bg-card p-3">
            <p
              role={
                editor.status === "failed" ||
                editor.status === "conflict" ||
                editor.status === "readbackFailed"
                  ? "alert"
                  : "status"
              }
              className="min-w-0 flex-1 text-xs"
            >
              {busy
                ? t("saving")
                : editor.status === "conflict"
                  ? t("conflict")
                  : editor.status === "readbackFailed"
                    ? t("readbackFailed")
                    : editor.status === "failed"
                      ? t("saveFailed")
                      : editor.status === "review"
                        ? t("reviewHelp")
                        : editor.dirty
                          ? t("unsaved")
                          : editor.status === "saved"
                            ? t("saved")
                            : t("noChanges")}
            </p>
            {(editor.status === "conflict" ||
              editor.status === "readbackFailed") && (
              <Button
                type="button"
                size="sm"
                variant="outline"
                onClick={() => void editor.reviewLatest()}
              >
                {t("reviewLatest")}
              </Button>
            )}
            <Button
              type="button"
              size="sm"
              variant="ghost"
              disabled={!editor.dirty || busy}
              onClick={() => {
                if (window.confirm(t("discardConfirm"))) editor.discard();
              }}
            >
              {t("discard")}
            </Button>
            <Button
              type="button"
              size="sm"
              disabled={
                !editor.dirty ||
                busy ||
                editor.status === "conflict" ||
                editor.status === "readbackFailed"
              }
              onClick={() => void editor.save()}
            >
              {busy ? t("saving") : t("save")}
            </Button>
          </div>
        </>
      )}
    </section>
  );
}

function hex(rgb: CosmeticRGB): string {
  return `#${rgb.map((v) => v.toString(16).padStart(2, "0")).join("")}`;
}

function ColorField({
  label,
  value,
  resolved,
  imported,
  saved,
  onChange,
}: {
  label: string;
  value: CosmeticRGB | null;
  resolved?: CosmeticRGB;
  imported: CosmeticRGB | null;
  saved?: CosmeticRGB | null;
  onChange: (value: CosmeticRGB | null) => void;
}) {
  const t = useTranslations("historyFilesDialog.cosmetic");
  const id = useId();

  const displayed = value ?? resolved;
  return (
    <fieldset className="space-y-2">
      <legend className="text-sm font-medium">{label}</legend>
      <div className="flex flex-wrap items-center gap-2">
        {displayed && (
          <input
            id={id}
            type="color"
            aria-label={label}
            value={hex(displayed)}
            className="h-9 w-12 rounded-md border border-input bg-card p-1 ui-focus-ring ui-interactive"
            onChange={(event) =>
              onChange(
                [1, 3, 5].map((start) =>
                  parseInt(event.target.value.slice(start, start + 2), 16),
                ) as CosmeticRGB,
              )
            }
          />
        )}
        <Button
          type="button"
          size="sm"
          variant="ghost"
          disabled={value === null}
          onClick={() => onChange(null)}
        >
          {t("resetColor")}
        </Button>
      </div>
      <p className="text-xs text-muted-foreground" aria-live="polite">
        {displayed
          ? t("colorResult", { value: hex(displayed).toUpperCase() })
          : t("previewPending")}
      </p>
      {saved !== undefined && (
        <p className="text-xs text-muted-foreground">
          {t("savedValue", {
            value: saved ? hex(saved).toUpperCase() : t("noOverride"),
          })}
        </p>
      )}
      <details className="text-xs text-muted-foreground">
        <summary className="cursor-pointer ui-focus-ring ui-interactive">
          {t("colorDetails")}
        </summary>
        <div className="mt-1 space-y-1">
          <p>
            {value
              ? t("authored")
              : imported
                ? t("imported")
                : t("colorFallback")}
          </p>
          {imported && (
            <p>{t("importedValue", { value: hex(imported).toUpperCase() })}</p>
          )}
        </div>
      </details>
    </fieldset>
  );
}

function RawOverrideReview({
  entries,
  onCopy,
}: {
  entries: CosmeticRawOverride[];
  onCopy: (entries: CosmeticRawOverride[]) => Promise<void>;
}) {
  const t = useTranslations("historyFilesDialog.cosmetic");
  const [selected, setSelected] = useState<Set<string>>(new Set());
  const [copied, setCopied] = useState(false);
  const [copying, setCopying] = useState(false);
  const [failed, setFailed] = useState(false);
  const keyFor = (entry: CosmeticRawOverride) =>
    `${entry.file_id}:${entry.locale}:${entry.suffix}`;
  return (
    <details className="rounded-md border border-border p-3">
      <summary className="cursor-pointer text-sm ui-focus-ring ui-interactive">
        {t("rawTitle", { count: entries.length })}
      </summary>
      <p className="mt-2 text-xs text-muted-foreground">{t("rawHelp")}</p>
      <div className="my-3 space-y-3">
        {entries.map((entry) => (
          <label key={keyFor(entry)} className="flex items-start gap-2 text-xs">
            <input
              type="checkbox"
              className="mt-0.5 ui-focus-ring"
              disabled={copying}
              checked={selected.has(keyFor(entry))}
              onChange={(event) => {
                const next = new Set(selected);
                if (event.target.checked) {
                  for (const candidate of entries) {
                    if (
                      candidate.locale === entry.locale &&
                      candidate.suffix === entry.suffix
                    )
                      next.delete(keyFor(candidate));
                  }
                  next.add(keyFor(entry));
                } else next.delete(keyFor(entry));
                setSelected(next);
                setCopied(false);
              }}
            />
            <span className="min-w-0 space-y-1">
              <span className="block break-all font-mono">{entry.path}</span>
              <span className="block">
                {entry.locale} · {entry.suffix || t("mainName")}
              </span>
              <span className="block break-words">
                {entry.value || t("emptyValue")}
              </span>
            </span>
          </label>
        ))}
      </div>
      <Button
        type="button"
        variant="outline"
        size="sm"
        disabled={selected.size === 0 || copying}
        onClick={async () => {
          setCopying(true);
          setFailed(false);
          setCopied(false);
          try {
            await onCopy(
              entries.filter((entry) => selected.has(keyFor(entry))),
            );
            setCopied(true);
          } catch {
            setFailed(true);
          } finally {
            setCopying(false);
          }
        }}
      >
        {copying ? t("loading") : t("rawCopy", { count: selected.size })}
      </Button>
      {failed && (
        <p role="alert" className="mt-2 text-xs text-destructive">
          {t("rawCopyFailed")}
        </p>
      )}
      {copied && (
        <p role="status" className="mt-2 text-xs text-muted-foreground">
          {t("rawCopied")}
        </p>
      )}
    </details>
  );
}
