"use client";

import { useState, useEffect, useCallback, useMemo, memo } from "react";
import { useBaseModDepKeys, gfxUrl } from "@/lib/api/basemod";
import { useTranslations } from "use-intl";
import { ArrowLeft } from "lucide-react";
import { useProjectStore } from "@/lib/store/projectStore";
import { DisplayNameLocsTabs } from "@/components/shared/editors/DisplayNameLocsTabs";
import { DescriptionLocsTabs } from "@/components/shared/editors/DescriptionLocsTabs";
import { DEFAULT_EXPORT_LANGUAGES } from "@/components/shared/editors/displayNameLocsHelpers";
import {
  useIdeologyOptions,
  CUSTOM_IDEOLOGY_SENTINEL,
} from "@/lib/ideology/useIdeologyOptions";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Separator } from "@/components/ui/separator";
import { Select } from "@/components/ui/select";
import {
  bookmarksApi,
  focusTreesApi,
  ideasApi,
  type BookmarkRow,
  type BookmarkSelectableCountry,
  type BookmarkFilterChoice119,
  BOOKMARK_FILTER_CHOICES_1_19,
  isBookmarkFilterChoice119,
  updateBookmarkFilterChoice119,
} from "@/lib/api";
import type { FocusNode, FocusTree, Idea } from "@/types";
import type { SaveDiagnostic } from "@/lib/saveDiagnostics";
import { DialogSaveStatusBar } from "@/components/workbench/DialogSaveStatusBar";
import { ImageUpload } from "@/shared/upload/ImageUpload";
import { BookmarkLabelInput } from "@/components/workbench/BookmarkLabelInput";
import { SaveDiagnostics } from "./save-diagnostics/SaveDiagnostics";
import { includeDefaultCountry } from "./save-diagnostics/bookmarkDefaultCountry";

const CUSTOM_IDEOLOGY_LS_KEY = "bookmark_custom_ideologies:v1";
const CUSTOM_IDEOLOGY_MAX_RECENT = 5;

function _loadRecentCustomIdeologies(): string[] {
  if (typeof window === "undefined") return [];
  try {
    const raw = window.localStorage.getItem(CUSTOM_IDEOLOGY_LS_KEY);
    if (!raw) return [];
    const parsed = JSON.parse(raw);
    return Array.isArray(parsed)
      ? parsed
          .filter((v): v is string => typeof v === "string")
          .slice(0, CUSTOM_IDEOLOGY_MAX_RECENT)
      : [];
  } catch {
    return [];
  }
}

function _saveRecentCustomIdeology(value: string): string[] {
  if (typeof window === "undefined" || !value) return [];
  const prior = _loadRecentCustomIdeologies();

  const filtered = prior.filter((v) => v.toLowerCase() !== value.toLowerCase());
  const next = [value, ...filtered].slice(0, CUSTOM_IDEOLOGY_MAX_RECENT);
  try {
    window.localStorage.setItem(CUSTOM_IDEOLOGY_LS_KEY, JSON.stringify(next));
  } catch {}
  return next;
}

const MAX_FOCUSES_PER_ROW = 3;

const INITIAL_RENDER_CAP = 50;

function parseCsvIds(raw: string): string[] {
  return raw
    .split(/[,\n]+/)
    .map((s) => s.trim())
    .filter(Boolean);
}

function rawExtraStr(entry: BookmarkSelectableCountry, key: string): string {
  const v = (entry.raw_extras as Record<string, unknown> | undefined)?.[key];
  if (typeof v === "string") return v;
  return v == null ? "" : String(v);
}

function rowHasLegacyNewContent(entry: BookmarkSelectableCountry): boolean {
  return rawExtraStr(entry, "new_content").trim() !== "";
}

const BOOKMARK_VERSION_TOKENS = [
  "base_game",
  "no_compromise_no_surrender",
  "trial_of_allegiance",
  "thunder_at_our_gates",
  "graveyard_of_empires",
  "gotterdammerung",
  "arms_against_tyranny",
  "by_blood_alone",
  "peace_for_our_time",
  "no_step_back",
  "man_the_guns",
  "la_resistance",
] as const;

function rowHasAdvancedFields(entry: BookmarkSelectableCountry): boolean {
  return Boolean(
    (entry.ideology && entry.ideology.trim()) ||
    (entry.ideas && entry.ideas.length > 0) ||
    (entry.focuses && entry.focuses.length > 0) ||
    rawExtraStr(entry, "override_leader_portrait").trim() ||
    rawExtraStr(entry, "version").trim() ||
    rowHasLegacyNewContent(entry),
  );
}

function autoOpenedRows(
  countries: BookmarkSelectableCountry[],
  known: Set<string>,
): { expanded: Set<number>; custom: Set<number> } {
  const expanded = new Set<number>();
  const custom = new Set<number>();
  countries.forEach((entry, idx) => {
    if (rowHasAdvancedFields(entry)) expanded.add(idx);

    const ideology = (entry.ideology || "").trim();
    if (ideology && !known.has(ideology)) custom.add(idx);
  });
  return { expanded, custom };
}

function setsEqual(a: Set<number>, b: Set<number>): boolean {
  if (a.size !== b.size) return false;
  for (const v of a) {
    if (!b.has(v)) return false;
  }
  return true;
}

interface CountryKeys {
  bookmarkKey: string;
  next: number;
  keys: string[];
}

function withCountryKeys(
  current: CountryKeys,
  bookmarkKey: string,
  count: number,
): CountryKeys {
  const same = current.bookmarkKey === bookmarkKey;
  if (same && current.keys.length >= count) return current;
  const keys = same ? [...current.keys] : [];
  let next = same ? current.next : 0;
  while (keys.length < count)
    keys.push(`bookmark:${bookmarkKey}:country:${next++}`);
  return { bookmarkKey, next, keys };
}

export const blankBookmarkDraft = (): {
  edit_revision?: string;
  bookmark_id: string;
  name_en: string;
  name_zh: string;
  desc_en: string;
  desc_zh: string;

  display_name_locs: Record<string, string>;
  description_locs: Record<string, string>;
  date: string;
  picture: string;
  default_country: string;
  default_flag: boolean;
  selectable_countries: BookmarkSelectableCountry[];
  filters: string[];
} => ({
  bookmark_id: "",
  name_en: "",
  name_zh: "",
  desc_en: "",
  desc_zh: "",
  display_name_locs: {},
  description_locs: {},
  date: "1936.1.1.12",
  picture: "GFX_select_date_1936",
  default_country: "",
  default_flag: false,
  selectable_countries: [],
  filters: [],
});

export type BookmarkDraft = ReturnType<typeof blankBookmarkDraft>;

export function bookmarkRowToDraft(row: BookmarkRow): BookmarkDraft {
  return {
    edit_revision: row.edit_revision,
    bookmark_id: row.bookmark_id,
    name_en: row.name_en,
    name_zh: row.name_zh,
    desc_en: row.desc_en,
    desc_zh: row.desc_zh,
    display_name_locs: { ...(row.display_name_locs ?? {}) },
    description_locs: { ...(row.description_locs ?? {}) },
    date: row.date,
    picture: row.picture,
    default_country: row.default_country,
    default_flag: row.default_flag,
    selectable_countries: Array.isArray(row.selectable_countries)
      ? row.selectable_countries
      : [],
    filters: Array.isArray(row.filters) ? row.filters : [],
  };
}

interface SelectableCountryRowProps {
  entityKey: string;
  entry: BookmarkSelectableCountry;
  idx: number;
  expanded: boolean;
  customIdeology: boolean;
  ideologyOptions: { value: string; label: string }[];
  knownIdeologyValues: Set<string>;
  recentCustomIdeologies: string[];
  projectIdeas: Idea[];
  projectFocuses: Array<FocusNode & { _country_tag?: string }>;
  exportLanguages: string[];
  onPatch: (idx: number, patch: Partial<BookmarkSelectableCountry>) => void;
  onRemove: (idx: number) => void;
  onToggleExpanded: (idx: number) => void;
  onEnterCustomIdeology: (idx: number) => void;
  onExitCustomIdeology: (idx: number) => void;
  onSaveRecentIdeology: (value: string) => void;
}

const SelectableCountryRow = memo(function SelectableCountryRow({
  entityKey,
  entry,
  idx,
  expanded,
  customIdeology,
  ideologyOptions,
  knownIdeologyValues,
  recentCustomIdeologies,
  projectIdeas,
  projectFocuses,
  exportLanguages,
  onPatch,
  onRemove,
  onToggleExpanded,
  onEnterCustomIdeology,
  onExitCustomIdeology,
  onSaveRecentIdeology,
}: SelectableCountryRowProps) {
  const t = useTranslations("bookmarksDialog");

  const overrideLeaderPortrait = rawExtraStr(entry, "override_leader_portrait");
  const countryVersion = rawExtraStr(entry, "version");

  const hasLegacyNewContent = rowHasLegacyNewContent(entry);

  const labelJoined = (entry.label || []).join(" ");

  const patchRawExtra = (key: string, value: string) => {
    const next: Record<string, unknown> = {
      ...((entry.raw_extras as Record<string, unknown> | undefined) ?? {}),
    };
    if (value.trim()) next[key] = value;
    else delete next[key];
    onPatch(idx, { raw_extras: next });
  };

  const advCount =
    ((entry.ideology || "").trim() ? 1 : 0) +
    ((entry.ideas || []).length > 0 ? 1 : 0) +
    ((entry.focuses || []).length > 0 ? 1 : 0) +
    (overrideLeaderPortrait.trim() ? 1 : 0) +
    (countryVersion.trim() ? 1 : 0) +
    (hasLegacyNewContent ? 1 : 0);
  const filteredIdeas = projectIdeas.filter(
    (i) =>
      !entry.tag ||
      !i.country_tag ||
      i.country_tag.toUpperCase() === entry.tag.toUpperCase(),
  );

  return (
    <div className="rounded-md border p-2 bg-muted/10 space-y-2">
      <div className="flex items-center justify-between gap-2">
        <Label className="font-semibold">
          {t("countryHeading", { num: idx + 1 })}
        </Label>
        <div className="flex items-center gap-3">
          <label className="flex items-center gap-1 text-xs text-muted-foreground cursor-pointer">
            <input
              type="checkbox"
              checked={entry.minor}
              onChange={(e) => onPatch(idx, { minor: e.target.checked })}
              className="accent-accent h-3 w-3"
            />
            {t("minorLabel")}
          </label>
          <Button size="sm" variant="ghost" onClick={() => onRemove(idx)}>
            {t("delete")}
          </Button>
        </div>
      </div>
      <div className="grid grid-cols-[100px_1fr] gap-2">
        <div className="space-y-1">
          <Label className="text-muted-foreground">TAG</Label>
          <Input
            value={entry.tag}
            onChange={(e) =>
              onPatch(idx, { tag: e.target.value.toUpperCase() })
            }
            placeholder="USA"
            className="h-7 text-xs font-mono"
          />
        </div>
        <div className="space-y-1 min-w-0">
          <Label className="text-muted-foreground">{t("hoverEnLabel")}</Label>
          <DescriptionLocsTabs
            entityKey={`${entityKey}:history`}
            value={entry.history_locs ?? {}}
            exportLanguages={exportLanguages}
            onChange={(next) => onPatch(idx, { history_locs: next })}
            legacyFallback={{ en: entry.history_en, zh: entry.history_zh }}
            placeholder={t("hoverEnPlaceholder")}
            ariaLabel={t("hoverEnLabel")}
            rows={2}
          />
        </div>
      </div>

      {entry.tag.trim() && !(entry.ideology || "").trim() ? (
        <div className="rounded-md border border-warning/40 bg-warning/5 px-2 py-1.5 text-xs text-warning">
          <p className="font-medium">
            {t("ideologyMissingTitle", { tag: entry.tag })}
          </p>
          <p className="mt-0.5 text-muted-foreground">
            {t("ideologyMissingImpact")}
          </p>
        </div>
      ) : null}

      <div className="space-y-1">
        <Label className="text-muted-foreground">
          {t.raw("countryAvailableLabel")}
        </Label>
        <textarea
          value={entry.available || ""}
          onChange={(e) => onPatch(idx, { available: e.target.value })}
          placeholder={t("countryAvailablePlaceholder")}
          className="w-full min-h-10 px-2 py-1 text-xs border rounded-md bg-background font-mono"
        />
        <p className="text-xs text-muted-foreground leading-snug">
          {t("countryAvailableHint")}
        </p>
      </div>

      <div className="space-y-1">
        <Label className="text-muted-foreground">
          {t("countryLabelLabel")}
        </Label>
        <BookmarkLabelInput
          value={labelJoined}
          onTokensChange={(tokens) => onPatch(idx, { label: tokens })}
          placeholder="label_1 label_2"
          className="h-7 text-xs font-mono"
        />
      </div>

      <div className="pt-1">
        <button
          type="button"
          onClick={() => onToggleExpanded(idx)}
          className="flex items-center gap-1 text-xs text-muted-foreground hover:text-foreground"
        >
          <span>{expanded ? "▾" : "▸"}</span>
          <span>
            {t("advancedFields")}
            {advCount > 0 ? (
              <span className="ml-1 inline-flex items-center justify-center min-w-4 h-4 px-1 rounded-full bg-accent/20 text-accent text-micro font-semibold">
                {advCount}
              </span>
            ) : null}
            )
          </span>
        </button>
        {expanded && (
          <div className="mt-2 pl-3 border-l-2 border-accent/30 space-y-2">
            <div className="space-y-1">
              <Label className="text-muted-foreground">
                {t("startIdeology")}
              </Label>
              {customIdeology ? (
                <>
                  <Input
                    type="text"
                    list={`bookmark-custom-ideology-${idx}`}
                    value={entry.ideology || ""}
                    onChange={(e) => onPatch(idx, { ideology: e.target.value })}
                    onBlur={() => onSaveRecentIdeology(entry.ideology || "")}
                    placeholder="rexism_extreme / alt_anarcho_syndicalism / ..."
                    className="h-7 text-xs font-mono"
                  />
                  <datalist id={`bookmark-custom-ideology-${idx}`}>
                    {Array.from(knownIdeologyValues).map((v) => (
                      <option key={`van-${v}`} value={v} />
                    ))}
                    {recentCustomIdeologies.map((v) => (
                      <option key={`rec-${v}`} value={v} />
                    ))}
                  </datalist>
                  <button
                    type="button"
                    onClick={() => onExitCustomIdeology(idx)}
                    className="text-xs text-muted-foreground hover:text-foreground underline"
                  >
                    {t("switchBackVanilla")}
                  </button>
                </>
              ) : (
                <Select
                  value={entry.ideology || ""}
                  onChange={(e) => {
                    if (e.target.value === CUSTOM_IDEOLOGY_SENTINEL) {
                      onEnterCustomIdeology(idx);
                    } else {
                      onPatch(idx, { ideology: e.target.value });
                    }
                  }}
                  className="h-7 px-2 text-xs"
                >
                  {ideologyOptions.map((opt) => (
                    <option key={opt.value} value={opt.value}>
                      {opt.label}
                    </option>
                  ))}
                </Select>
              )}
            </div>

            <div className="space-y-1">
              <Label className="text-muted-foreground">{t("startNS")}</Label>
              <textarea
                value={(entry.ideas || []).join(", ")}
                onChange={(e) =>
                  onPatch(idx, { ideas: parseCsvIds(e.target.value) })
                }
                placeholder="modusa_great_again, modusa_arrived"
                className="w-full min-h-8 px-2 py-1 text-xs border rounded-md bg-background font-mono"
              />
              {filteredIdeas.length > 0 && (
                <div className="flex flex-wrap gap-1 pt-1">
                  <span className="text-xs text-muted-foreground">
                    {t("available")}
                  </span>
                  {filteredIdeas.slice(0, 10).map((idea) => (
                    <button
                      key={idea.id}
                      type="button"
                      onClick={() => {
                        const cur = entry.ideas || [];
                        if (cur.includes(idea.idea_id)) return;
                        onPatch(idx, { ideas: [...cur, idea.idea_id] });
                      }}
                      className="text-xs px-1.5 py-0.5 rounded border border-muted-foreground/30 hover:bg-muted font-mono"
                      title={t("insertTooltip", { id: idea.idea_id })}
                    >
                      {idea.idea_id}
                    </button>
                  ))}
                  {filteredIdeas.length > 10 && (
                    <span className="text-xs text-muted-foreground">
                      ... +{filteredIdeas.length - 10}
                    </span>
                  )}
                </div>
              )}
            </div>

            <div className="space-y-1">
              <Label className="text-muted-foreground">
                {t("focusPreviewLabel", { max: MAX_FOCUSES_PER_ROW })}
              </Label>
              <textarea
                value={(entry.focuses || []).join(", ")}
                onChange={(e) => {
                  const parsed = parseCsvIds(e.target.value);
                  onPatch(idx, {
                    focuses: parsed.slice(0, MAX_FOCUSES_PER_ROW),
                  });
                }}
                placeholder="USA_F1, USA_F2, USA_F3"
                className="w-full min-h-8 px-2 py-1 text-xs border rounded-md bg-background font-mono"
              />
              {(() => {
                const filteredFocuses = projectFocuses.filter(
                  (n) =>
                    !entry.tag ||
                    !n._country_tag ||
                    n._country_tag === entry.tag.toUpperCase(),
                );
                if (filteredFocuses.length === 0) return null;
                const currentSet = new Set(
                  (entry.focuses || []).map((f) => f.toLowerCase()),
                );
                const atLimit =
                  (entry.focuses || []).length >= MAX_FOCUSES_PER_ROW;
                return (
                  <div className="flex flex-wrap gap-1 pt-1">
                    <span className="text-xs text-muted-foreground">
                      {t("available")}
                    </span>
                    {filteredFocuses.slice(0, 10).map((node) => {
                      const already = currentSet.has(
                        node.focus_id.toLowerCase(),
                      );
                      return (
                        <button
                          key={node.id}
                          type="button"
                          disabled={already || atLimit}
                          onClick={() => {
                            const cur = entry.focuses || [];
                            if (cur.includes(node.focus_id)) return;
                            if (cur.length >= MAX_FOCUSES_PER_ROW) return;
                            onPatch(idx, {
                              focuses: [...cur, node.focus_id],
                            });
                          }}
                          className={`text-xs px-1.5 py-0.5 rounded border font-mono ${
                            already
                              ? "border-success/40 text-success/80 bg-success/10"
                              : atLimit
                                ? "border-muted-foreground/20 text-muted-foreground/50 cursor-not-allowed"
                                : "border-muted-foreground/30 hover:bg-muted"
                          }`}
                          title={
                            already
                              ? t("alreadyAdded")
                              : atLimit
                                ? t("atLimitTooltip", {
                                    max: MAX_FOCUSES_PER_ROW,
                                  })
                                : t("insertTooltip", { id: node.focus_id })
                          }
                        >
                          {node.focus_id}
                        </button>
                      );
                    })}
                    {filteredFocuses.length > 10 && (
                      <span className="text-xs text-muted-foreground">
                        ... +{filteredFocuses.length - 10}
                      </span>
                    )}
                  </div>
                );
              })()}
              {(entry.focuses || []).length > MAX_FOCUSES_PER_ROW - 1 && (
                <p className="text-xs text-warning">
                  {t("limitWarning", { max: MAX_FOCUSES_PER_ROW })}
                </p>
              )}
            </div>

            <div className="space-y-1">
              <Label className="text-muted-foreground">
                {t("overrideLeaderPortraitLabel")}
              </Label>
              <Input
                type="text"
                value={overrideLeaderPortrait}
                onChange={(e) =>
                  patchRawExtra("override_leader_portrait", e.target.value)
                }
                placeholder="JAP_emperor_hirohito"
                className="h-7 text-xs font-mono"
              />
              <p className="text-xs text-muted-foreground leading-snug">
                {t("overrideLeaderPortraitHint")}
              </p>
            </div>

            <div className="space-y-1">
              <Label className="text-muted-foreground">
                {t("versionLabel")}
              </Label>
              <Input
                type="text"
                list={`bookmark-version-${entityKey}`}
                value={countryVersion}
                onChange={(e) => patchRawExtra("version", e.target.value)}
                placeholder="base_game"
                className="h-7 text-xs font-mono"
              />
              <datalist id={`bookmark-version-${entityKey}`}>
                {BOOKMARK_VERSION_TOKENS.map((token) => (
                  <option key={token} value={token} />
                ))}
              </datalist>
              <p className="text-xs text-muted-foreground leading-snug">
                {t("versionHint")}
              </p>
            </div>

            {hasLegacyNewContent && (
              <p className="text-xs text-warning leading-snug pt-1">
                {t("newContentRetiredNotice")}
              </p>
            )}
          </div>
        )}
      </div>
    </div>
  );
});

export function BookmarkEditor({
  projectId,
  activeId,
  draft,
  onEdit,
  rows,
  saving,
  saved,
  error,
  diagnostics,
  onSave,
  onDelete,
  onCancelNew,
  onPictureUploaded,
}: {
  projectId: number;

  activeId: number | "new";
  draft: BookmarkDraft;

  onEdit: (change: (current: BookmarkDraft) => BookmarkDraft) => void;

  rows: readonly BookmarkRow[];
  saving: boolean;

  saved: boolean;

  error: string | null;
  diagnostics?: readonly SaveDiagnostic[];
  onSave: () => void;

  onDelete?: () => void;

  onCancelNew?: () => void;

  onPictureUploaded?: () => void;
}) {
  const t = useTranslations("bookmarksDialog");

  const baseModDepKeys = useBaseModDepKeys();

  const exportLanguages = useProjectStore(
    (s) => s.currentProject?.export_languages ?? DEFAULT_EXPORT_LANGUAGES,
  );

  const { subOptions: ideologySubOptions, groupOptions: ideologyGroupOptions } =
    useIdeologyOptions();
  const ideologyOptions = useMemo(
    () => [
      { value: "", label: "空 · 不 emit (Empty · no emit)" },
      ...ideologyGroupOptions.map((o) => ({ value: o.value, label: o.label })),
      ...ideologySubOptions.map((o) => ({ value: o.value, label: o.label })),
      {
        value: CUSTOM_IDEOLOGY_SENTINEL,
        label:
          "自定义 · 输入 mod-defined ideology (Custom · enter a mod-defined ideology)",
      },
    ],
    [ideologyGroupOptions, ideologySubOptions],
  );

  const knownIdeologyValues = useMemo(() => {
    const s = new Set<string>();
    ideologyGroupOptions.forEach((o) => s.add(o.value));
    ideologySubOptions.forEach((o) => s.add(o.value));
    return s;
  }, [ideologyGroupOptions, ideologySubOptions]);

  const bookmarkIdentity = String(activeId);
  const [storedCountryKeys, setCountryKeys] = useState<CountryKeys>(() =>
    withCountryKeys(
      { bookmarkKey: bookmarkIdentity, next: 0, keys: [] },
      bookmarkIdentity,
      draft.selectable_countries.length,
    ),
  );
  const countryKeys = withCountryKeys(
    storedCountryKeys,
    bookmarkIdentity,
    draft.selectable_countries.length,
  );
  if (countryKeys !== storedCountryKeys) setCountryKeys(countryKeys);

  const [visibleCount, setVisibleCount] = useState(INITIAL_RENDER_CAP);
  const [visibleCountFor, setVisibleCountFor] = useState(activeId);
  if (visibleCountFor !== activeId) {
    setVisibleCountFor(activeId);
    setVisibleCount(INITIAL_RENDER_CAP);
  }

  const [expandedRows, setExpandedRows] = useState<Set<number>>(new Set());

  const [customIdeologyRows, setCustomIdeologyRows] = useState<Set<number>>(
    new Set(),
  );

  const [autoOpenedFor, setAutoOpenedFor] = useState<{
    countries: BookmarkSelectableCountry[];
    known: Set<string>;
  } | null>(null);
  if (
    !autoOpenedFor ||
    autoOpenedFor.countries !== draft.selectable_countries ||
    autoOpenedFor.known !== knownIdeologyValues
  ) {
    setAutoOpenedFor({
      countries: draft.selectable_countries,
      known: knownIdeologyValues,
    });
    const { expanded, custom } = autoOpenedRows(
      draft.selectable_countries,
      knownIdeologyValues,
    );

    setExpandedRows((prev) => (setsEqual(prev, expanded) ? prev : expanded));
    setCustomIdeologyRows((prev) => (setsEqual(prev, custom) ? prev : custom));
  }

  const [recentCustomIdeologies, setRecentCustomIdeologies] = useState<
    string[]
  >(_loadRecentCustomIdeologies);

  const [projectIdeas, setProjectIdeas] = useState<Idea[]>([]);

  const [projectFocuses, setProjectFocuses] = useState<
    Array<FocusNode & { _country_tag?: string }>
  >([]);

  useEffect(() => {
    let cancelled = false;
    (async () => {
      try {
        const list = await ideasApi.list(projectId);
        if (!cancelled) setProjectIdeas(list);
      } catch {
        if (!cancelled) setProjectIdeas([]);
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [projectId]);

  useEffect(() => {
    let cancelled = false;
    (async () => {
      try {
        const trees: FocusTree[] = await focusTreesApi.list(projectId);
        const flat: Array<FocusNode & { _country_tag?: string }> = [];
        for (const tree of trees) {
          const treeTag = (tree.country_tag || "").toUpperCase();
          for (const node of tree.nodes || []) {
            flat.push({ ...node, _country_tag: treeTag });
          }
        }
        if (!cancelled) setProjectFocuses(flat);
      } catch {
        if (!cancelled) setProjectFocuses([]);
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [projectId]);

  const toggleExpanded = useCallback((idx: number) => {
    setExpandedRows((prev) => {
      const next = new Set(prev);
      if (next.has(idx)) next.delete(idx);
      else next.add(idx);
      return next;
    });
  }, []);

  const update = useCallback(
    <K extends keyof BookmarkDraft>(key: K, value: BookmarkDraft[K]) => {
      onEdit((current) => ({ ...current, [key]: value }));
    },
    [onEdit],
  );

  const patchCountry = useCallback(
    (idx: number, patch: Partial<BookmarkSelectableCountry>) => {
      onEdit((d) => {
        const next = d.selectable_countries.slice();
        next[idx] = { ...next[idx], ...patch };
        return { ...d, selectable_countries: next };
      });
    },
    [onEdit],
  );

  const removeCountry = useCallback(
    (idx: number) => {
      setCountryKeys((current) => ({
        ...current,
        keys: current.keys.filter((_, i) => i !== idx),
      }));
      onEdit((current) => ({
        ...current,
        selectable_countries: current.selectable_countries.filter(
          (_, i) => i !== idx,
        ),
      }));
    },
    [onEdit],
  );

  const enterCustomIdeology = useCallback(
    (idx: number) => {
      setCustomIdeologyRows((prev) => {
        const next = new Set(prev);
        next.add(idx);
        return next;
      });
      patchCountry(idx, { ideology: "" });
    },
    [patchCountry],
  );

  const exitCustomIdeology = useCallback(
    (idx: number) => {
      setCustomIdeologyRows((prev) => {
        const next = new Set(prev);
        next.delete(idx);
        return next;
      });

      patchCountry(idx, { ideology: "" });
    },
    [patchCountry],
  );

  const saveRecentIdeology = useCallback(
    (value: string) => {
      const v = (value || "").trim();
      if (v && !knownIdeologyValues.has(v)) {
        setRecentCustomIdeologies(_saveRecentCustomIdeology(v));
      }
    },
    [knownIdeologyValues],
  );

  const otherDefaultBookmarks = rows.filter(
    (r) =>
      r.default_flag && (typeof activeId !== "number" || r.id !== activeId),
  );
  const showDefaultWarning =
    draft.default_flag && otherDefaultBookmarks.length > 0;

  const legacyFilterTokens = draft.filters.filter(
    (token) => !isBookmarkFilterChoice119(token),
  );
  const countryLabelValues = new Set(
    draft.selectable_countries.flatMap((entry) => entry.label ?? []),
  );
  const legacyTokensMatchingCountryLabels = legacyFilterTokens.filter((token) =>
    countryLabelValues.has(token),
  );
  const countriesHaveLabels = countryLabelValues.size > 0;

  const setFilterChoice = (
    choice: BookmarkFilterChoice119,
    enabled: boolean,
  ) => {
    update(
      "filters",
      updateBookmarkFilterChoice119(draft.filters, choice, enabled),
    );
  };

  const isExisting = typeof activeId === "number";

  return (
    <div className="p-4 space-y-3">
      {activeId === "new" && onCancelNew && (
        <button
          type="button"
          onClick={onCancelNew}
          className="ui-interactive inline-flex items-center gap-1 text-xs text-muted-foreground hover:text-foreground"
        >
          <ArrowLeft className="h-3.5 w-3.5" />
          {t("backFromNew")}
        </button>
      )}
      <div className="grid grid-cols-2 gap-3">
        <div className="space-y-1">
          <Label>{t("idLabel")}</Label>
          <Input
            value={draft.bookmark_id}
            onChange={(e) =>
              update(
                "bookmark_id",
                e.target.value.replace(/\s+/g, "_").toLowerCase(),
              )
            }
            placeholder="the_modern_usa_arrived"
            className="font-mono text-xs"
          />
        </div>
        <div className="space-y-1">
          <Label>{t("dateLabel")}</Label>
          <Input
            value={draft.date}
            onChange={(e) => update("date", e.target.value)}
            placeholder="1936.1.1.12"
            className="font-mono text-xs"
          />
        </div>
      </div>

      <Separator />

      <div className="space-y-1">
        <Label>{t("nameLabel")}</Label>
        <DisplayNameLocsTabs
          entityKey={`bookmark:${activeId}:display-name`}
          value={draft.display_name_locs}
          exportLanguages={exportLanguages}
          legacyFallback={{ en: draft.name_en, zh: draft.name_zh }}
          onChange={(next) => update("display_name_locs", next)}
          placeholder={t("namePlaceholder")}
          ariaLabel={t("nameLabel")}
        />
        <p className="text-xs text-muted-foreground">{t("nameHint")}</p>
      </div>

      <div className="space-y-1">
        <Label>{t("descLabel")}</Label>
        <DescriptionLocsTabs
          entityKey={`bookmark:${activeId}:description`}
          value={draft.description_locs}
          exportLanguages={exportLanguages}
          legacyFallback={{ en: draft.desc_en, zh: draft.desc_zh }}
          onChange={(next) => update("description_locs", next)}
          placeholder={t("descPlaceholder")}
          ariaLabel={t("descLabel")}
          rows={3}
        />
      </div>

      <Separator />

      <div className="space-y-1">
        <Label>{t("pictureLabel")}</Label>
        <Input
          value={draft.picture}
          onChange={(e) => update("picture", e.target.value)}
          placeholder="GFX_select_date_1936"
          className="font-mono text-xs"
        />

        <p className="text-xs text-muted-foreground leading-snug">
          {t.rich("pictureHint", {
            modname: "<modname>",
            mono: (c) => <span className="font-mono">{c}</span>,
          })}
        </p>

        <div className="pt-1">
          <ImageUpload
            key={`bookmark-backdrop:${activeId}`}
            label={t("backdropUploadLabel")}
            targetSize={{ width: 180, height: 104 }}
            previewUrl={
              isExisting && draft.picture.startsWith("GFX_hagane_select_date_")
                ? `${bookmarksApi.pictureUrl(projectId, activeId)}?v=${draft.picture}`
                : null
            }
            hint={
              isExisting
                ? t("backdropUploadHint")
                : t("backdropUploadCreateFirst")
            }
            disabled={!isExisting}
            gfxName={draft.picture}
            baseModFallbackUrls={
              draft.picture && baseModDepKeys.length
                ? baseModDepKeys.map((dk) =>
                    gfxUrl(projectId, dk, draft.picture),
                  )
                : []
            }
            onUpload={(file, crop) => {
              if (typeof activeId !== "number") {
                return Promise.reject(
                  new Error(t("backdropUploadCreateFirst")),
                );
              }
              return bookmarksApi.uploadPicture(
                projectId,
                activeId,
                file,
                crop,
                draft.edit_revision,
              );
            }}
            onUploaded={(result) => {
              if (result.picture) update("picture", result.picture);
              onPictureUploaded?.();
            }}
          />
        </div>
      </div>

      <div className="space-y-1">
        <Label>{t("defaultCountryLabel")}</Label>
        <Input
          value={draft.default_country}
          onChange={(e) =>
            update("default_country", e.target.value.toUpperCase())
          }
          onBlur={() => {
            const countries = includeDefaultCountry(
              draft.selectable_countries,
              draft.default_country,
            );
            if (countries !== draft.selectable_countries) {
              update("selectable_countries", countries);
              setVisibleCount((count) => Math.max(count, countries.length));
            }
          }}
          placeholder="USA"
          className="font-mono text-xs w-32"
        />
      </div>

      <label className="flex items-center gap-2 text-xs cursor-pointer">
        <input
          type="checkbox"
          checked={draft.default_flag}
          onChange={(e) => update("default_flag", e.target.checked)}
          className="accent-accent h-4 w-4"
        />
        <span>
          {t.rich("defaultBookmarkLabel", {
            mono: (c) => <span className="font-mono">{c}</span>,
          })}
        </span>
      </label>
      {showDefaultWarning && (
        <p className="text-xs text-warning">
          {t("defaultWarning", {
            count: otherDefaultBookmarks.length,
            list: otherDefaultBookmarks
              .map((r) => ` "${r.bookmark_id}"`)
              .join(","),
          })}
        </p>
      )}

      <div className="space-y-2">
        <Label>{t("filtersLabel")}</Label>
        <div
          className="flex flex-wrap gap-2"
          role="group"
          aria-label={t("filtersLabel")}
        >
          {BOOKMARK_FILTER_CHOICES_1_19.map((choice) => {
            const checked = draft.filters.includes(choice);
            return (
              <label
                key={choice}
                className={`ui-interactive ui-focus-ring flex h-8 cursor-pointer items-center gap-2 rounded-md border px-3 text-xs ${
                  checked
                    ? "border-accent bg-accent text-accent-foreground"
                    : "border-border bg-transparent text-foreground hover:bg-muted"
                }`}
              >
                <input
                  type="checkbox"
                  checked={checked}
                  onChange={(e) => setFilterChoice(choice, e.target.checked)}
                  className="sr-only"
                />
                <span className="font-mono">{choice}</span>
                <span>{t(`filterChoice_${choice}`)}</span>
              </label>
            );
          })}
        </div>
        <p className="text-xs text-muted-foreground leading-snug">
          {t("filtersHint")}
        </p>
        {legacyFilterTokens.length > 0 ? (
          <div className="rounded-md border border-warning/40 bg-warning/5 p-2">
            <p className="text-xs font-medium text-warning">
              {t("legacyFiltersTitle", { count: legacyFilterTokens.length })}
            </p>
            <div className="mt-1.5 flex flex-wrap gap-1">
              {legacyFilterTokens.map((token) => (
                <code
                  key={token}
                  className="rounded-md border border-border bg-card px-1.5 py-0.5 text-xs text-foreground"
                >
                  {token}
                </code>
              ))}
            </div>
            <p className="mt-1.5 text-xs text-muted-foreground">
              {t("legacyFiltersPreserved")}
            </p>
            {legacyTokensMatchingCountryLabels.length > 0 ? (
              <p className="mt-1 text-xs text-warning">
                {t("legacyFiltersLabelHint", {
                  tokens: legacyTokensMatchingCountryLabels.join(", "),
                })}
              </p>
            ) : null}
          </div>
        ) : null}
        {countriesHaveLabels && !draft.filters.includes("label") ? (
          <p className="rounded-md border border-warning/40 bg-warning/5 px-2 py-1.5 text-xs text-warning">
            {t("countryLabelsNeedFilter")}
          </p>
        ) : null}
      </div>

      <Separator />

      <div className="space-y-2">
        <div className="flex items-center justify-between">
          <Label className="text-sm font-semibold">
            {t("selectableLabel")}
          </Label>
          <Button
            size="sm"
            variant="outline"
            onClick={() => {
              update("selectable_countries", [
                ...draft.selectable_countries,
                {
                  tag: "",
                  history_en: "",
                  history_zh: "",
                  history_locs: {},
                  minor: false,
                },
              ]);

              setVisibleCount((c) =>
                Math.max(c, draft.selectable_countries.length + 1),
              );
            }}
          >
            {t("addCountry")}
          </Button>
        </div>
        {draft.selectable_countries.length === 0 ? (
          <p className="text-xs text-muted-foreground">
            {t("selectableEmpty", { tag: "<TAG>" })}
          </p>
        ) : (
          <div className="space-y-2">
            {draft.selectable_countries
              .slice(0, visibleCount)
              .map((entry, idx) => (
                <SelectableCountryRow
                  key={countryKeys.keys[idx]}
                  entityKey={countryKeys.keys[idx]}
                  entry={entry}
                  idx={idx}
                  expanded={expandedRows.has(idx)}
                  customIdeology={customIdeologyRows.has(idx)}
                  ideologyOptions={ideologyOptions}
                  knownIdeologyValues={knownIdeologyValues}
                  recentCustomIdeologies={recentCustomIdeologies}
                  projectIdeas={projectIdeas}
                  projectFocuses={projectFocuses}
                  exportLanguages={exportLanguages}
                  onPatch={patchCountry}
                  onRemove={removeCountry}
                  onToggleExpanded={toggleExpanded}
                  onEnterCustomIdeology={enterCustomIdeology}
                  onExitCustomIdeology={exitCustomIdeology}
                  onSaveRecentIdeology={saveRecentIdeology}
                />
              ))}
            {draft.selectable_countries.length > visibleCount && (
              <Button
                variant="outline"
                size="sm"
                className="w-full"
                onClick={() => setVisibleCount((c) => c + INITIAL_RENDER_CAP)}
              >
                {t("showMoreCountries", {
                  count: draft.selectable_countries.length - visibleCount,
                })}
              </Button>
            )}
          </div>
        )}
      </div>

      <Separator />

      <DialogSaveStatusBar
        saving={saving}
        saved={saved}
        error={error}
        savingLabel={t("saving")}
        savedLabel={t("saved")}
      />
      <SaveDiagnostics diagnostics={diagnostics} />

      <div className="flex gap-2">
        <Button onClick={onSave} disabled={saving} size="sm">
          {saving ? t("saving") : activeId === "new" ? t("create") : t("save")}
        </Button>
        {isExisting && onDelete && (
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
