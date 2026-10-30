"use client";

import { useId, useMemo, useState } from "react";
import { useTranslations } from "use-intl";
import { ArrowLeft } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Separator } from "@/components/ui/separator";
import { Select } from "@/components/ui/select";
import type { CountryRow, HistoryFileRow } from "@/lib/api";
import type { CosmeticTagView } from "@/lib/api/cosmeticTags";
import {
  useIdeologyOptions,
  CUSTOM_IDEOLOGY_SENTINEL,
} from "@/lib/ideology/useIdeologyOptions";
import { useProjectStore } from "@/lib/store/projectStore";
import { CosmeticTagEditor } from "./cosmetic-tags/CosmeticTagEditor";
import { CountryAppearanceSettings } from "./CountryAppearanceSettings";
import { OobPassthroughPanel } from "@/components/workbench/OobPassthroughPanel";
import { HistoryDateBlocksEditor } from "@/components/workbench/HistoryDateBlocksEditor";
import { DialogSaveStatusBar } from "@/components/workbench/DialogSaveStatusBar";

export const blankHistoryFileDraft = (): Omit<
  HistoryFileRow,
  "id" | "project_id" | "created_at" | "updated_at" | "source"
> => ({
  country_tag: "",
  capital: 0,
  set_politics_ruling_party: "",
  set_politics_last_election: "",
  set_politics_election_frequency: 0,
  set_politics_elections_allowed: null,
  set_popularities_democratic: 0,
  set_popularities_communism: 0,
  set_popularities_fascism: 0,
  set_popularities_neutrality: 0,
  set_popularities: {},
  set_research_slots: 0,
  set_convoys: 0,
  set_stability: -1,
  set_war_support: -1,
  add_ideas_extra: [],
  remove_ideas: [],
  complete_national_focus_ids: [],
  set_technology_ids: [],
  set_cosmetic_tag: "",
  set_oob: "",
  set_air_oob: "",
  set_naval_oob: "",
  startup_flags: [] as string[],
  give_guarantee_tags: [] as string[],
  add_to_faction_tags: [] as string[],
  add_to_faction: "",
  set_autonomy: [] as { target: string; autonomous_state: string }[],
  diplomatic_relation: [] as {
    country: string;
    relation: string;
    active: boolean;
  }[],
  date_blocks: [] as { date: string; script: string }[],
  raw_script: "",
});

export type HistoryFileDraft = ReturnType<typeof blankHistoryFileDraft>;

export function historyRowToDraft(row: HistoryFileRow): HistoryFileDraft {
  return {
    country_tag: row.country_tag,
    capital: row.capital ?? 0,
    set_politics_ruling_party: row.set_politics_ruling_party,
    set_politics_last_election: row.set_politics_last_election,
    set_politics_election_frequency: row.set_politics_election_frequency,
    set_politics_elections_allowed: row.set_politics_elections_allowed,
    set_popularities_democratic: row.set_popularities_democratic,
    set_popularities_communism: row.set_popularities_communism,
    set_popularities_fascism: row.set_popularities_fascism,
    set_popularities_neutrality: row.set_popularities_neutrality,

    set_popularities:
      Object.keys(row.set_popularities || {}).length > 0
        ? row.set_popularities
        : Object.fromEntries(
            [
              ["democratic", row.set_popularities_democratic],
              ["communism", row.set_popularities_communism],
              ["fascism", row.set_popularities_fascism],
              ["neutrality", row.set_popularities_neutrality],
            ].filter(([, value]) => Number(value) > 0),
          ),
    set_research_slots: row.set_research_slots,
    set_convoys: row.set_convoys ?? 0,
    set_stability: row.set_stability,
    set_war_support: row.set_war_support,
    add_ideas_extra: row.add_ideas_extra || [],
    remove_ideas: row.remove_ideas || [],
    complete_national_focus_ids: row.complete_national_focus_ids || [],
    set_technology_ids: row.set_technology_ids || [],
    set_cosmetic_tag: row.set_cosmetic_tag || "",
    set_oob: row.set_oob || "",
    set_air_oob: row.set_air_oob || "",
    set_naval_oob: row.set_naval_oob || "",
    startup_flags: row.startup_flags || [],
    give_guarantee_tags: row.give_guarantee_tags || [],
    add_to_faction_tags:
      row.add_to_faction_tags?.length > 0
        ? [...row.add_to_faction_tags]
        : row.add_to_faction
          ? [row.add_to_faction]
          : [],
    add_to_faction: "",
    set_autonomy: row.set_autonomy || [],
    diplomatic_relation: row.diplomatic_relation || [],
    date_blocks: row.date_blocks || [],
    raw_script: row.raw_script,
  };
}

export type HistoryFileSection = "ideas" | "diplomacy";

const listToCsv = (arr: string[]) => arr.join(", ");
const csvToList = (csv: string) =>
  csv
    .split(/[,\s\n]+/)
    .map((s) => s.trim())
    .filter(Boolean);

const noAppearanceView = () => {};

export function HistoryFileEditor({
  projectId,
  activeId,
  draft,
  onChange,
  rows,
  section,
  countries = [],
  countryNames = {},
  onSelectCountry,
  savedAppearance = "",
  latestAppearance = null,
  onAppearanceView = noAppearanceView,
  onAppearanceChange,
  onManageIdeologies,
  onCancelNew,
  diplomacyOptions,
  saving,
  deleting = false,
  saved,
  dirty,
  error,
  onSave,
  onDelete,
}: {
  projectId: number;

  activeId: number | "new";
  draft: HistoryFileDraft;
  onChange: <K extends keyof HistoryFileDraft>(
    key: K,
    value: HistoryFileDraft[K],
  ) => void;

  rows: readonly HistoryFileRow[];

  section?: HistoryFileSection;

  countries?: readonly CountryRow[];
  countryNames?: Record<string, string>;

  onSelectCountry?: (tag: string) => void;

  savedAppearance?: string;
  latestAppearance?: CosmeticTagView | null;
  onAppearanceView?: (view: CosmeticTagView | null) => void;

  onAppearanceChange?: (tag: string) => Promise<boolean>;

  onManageIdeologies?: () => void;

  onCancelNew?: () => void;

  diplomacyOptions?: {
    autonomyLevels: readonly { id: string; label?: string }[];
    relationTypes: readonly string[];
  };
  saving: boolean;

  deleting?: boolean;

  saved: boolean;

  dirty: boolean;

  error: string | null;
  onSave: () => void;

  onDelete?: () => void;
}) {
  const t = useTranslations("historyFilesDialog");

  const projectStartDate = useProjectStore(
    (s) => s.currentProject?.start_date ?? null,
  );

  const { groupOptions } = useIdeologyOptions();
  const knownRulingPartyValues = useMemo(
    () => new Set(groupOptions.map((o) => o.value)),
    [groupOptions],
  );
  const isExisting = typeof activeId === "number";
  const savedRow = isExisting ? rows.find((r) => r.id === activeId) : undefined;

  const [typedChoice, setTypedChoice] = useState<{
    row: HistoryFileRow | undefined;
    known: Set<string>;
    typed: boolean;
  } | null>(null);
  const savedRulingParty = (savedRow?.set_politics_ruling_party || "").trim();
  const rulingPartyCustom =
    typedChoice &&
    typedChoice.row === savedRow &&
    typedChoice.known === knownRulingPartyValues
      ? typedChoice.typed
      : !!savedRulingParty && !knownRulingPartyValues.has(savedRulingParty);
  const setRulingPartyCustom = (typed: boolean) =>
    setTypedChoice({ row: savedRow, known: knownRulingPartyValues, typed });

  const popularityGroups = useMemo(() => {
    const byRoot = new Map(
      groupOptions.map((option) => [option.value, option]),
    );
    for (const root of Object.keys(draft.set_popularities)) {
      if (!byRoot.has(root))
        byRoot.set(root, { value: root, label: root, group: root });
    }
    return Array.from(byRoot.values());
  }, [draft.set_popularities, groupOptions]);
  const popularitiesTotal = useMemo(
    () =>
      Object.values(draft.set_popularities).reduce(
        (total, value) => total + Number(value || 0),
        0,
      ),
    [draft.set_popularities],
  );
  const updatePopularity = (root: string, rawValue: number) => {
    const value = Number.isFinite(rawValue)
      ? Math.max(0, Math.min(100, Math.trunc(rawValue)))
      : 0;
    const next = { ...draft.set_popularities };
    if (value === 0) delete next[root];
    else next[root] = value;
    onChange("set_popularities", next);
  };

  const listId = useId();
  const autonomyListId = `${listId}-autonomy`;
  const relationListId = `${listId}-relation`;

  const ideasSection = (
    <div className="space-y-2">
      <Label className="text-sm font-semibold">{t("ideasTitle")}</Label>
      <p className="text-xs text-muted-foreground">
        {t.rich("ideasHelp", {
          mono: (c) => <span className="font-mono">{c}</span>,
        })}
      </p>
      <div className="grid grid-cols-2 gap-3">
        <div className="space-y-1">
          <Label className="text-muted-foreground">{t("addIdeasLabel")}</Label>
          <textarea
            value={listToCsv(draft.add_ideas_extra)}
            onChange={(e) =>
              onChange("add_ideas_extra", csvToList(e.target.value))
            }
            placeholder="national_spirit_1, national_spirit_2"
            className="w-full min-h-16 px-2 py-1 text-xs font-mono border rounded-md bg-background"
          />
        </div>
        <div className="space-y-1">
          <Label className="text-muted-foreground">
            {t("removeIdeasLabel")}
          </Label>
          <textarea
            value={listToCsv(draft.remove_ideas)}
            onChange={(e) =>
              onChange("remove_ideas", csvToList(e.target.value))
            }
            placeholder="undisturbed_isolation, great_depression, disarmed_nation, free_trade, home_of_the_free"
            className="w-full min-h-16 px-2 py-1 text-xs font-mono border rounded-md bg-background"
          />
        </div>
      </div>
    </div>
  );

  const diplomacySection = (
    <div className="space-y-3">
      {diplomacyOptions && (
        <>
          <datalist id={autonomyListId}>
            {diplomacyOptions.autonomyLevels.map((level) => (
              <option
                key={level.id}
                value={level.id}
                label={level.label || level.id}
              />
            ))}
          </datalist>
          <datalist id={relationListId}>
            {diplomacyOptions.relationTypes.map((type) => (
              <option key={type} value={type} />
            ))}
          </datalist>
        </>
      )}

      <Label className="text-sm font-semibold">{t("diplomacyLabel")}</Label>

      <div className="space-y-1">
        <Label className="text-muted-foreground">
          {t("giveGuaranteeLabel")}
        </Label>
        <textarea
          value={listToCsv(draft.give_guarantee_tags)}
          onChange={(e) =>
            onChange(
              "give_guarantee_tags",
              csvToList(e.target.value).map((s) => s.toUpperCase()),
            )
          }
          placeholder="POL, BEL"
          className="w-full min-h-12 px-2 py-1 text-xs font-mono border rounded-md bg-background"
        />
        <p className="text-xs text-muted-foreground">
          {t.raw("giveGuaranteeHelp")}
        </p>
      </div>

      <div className="space-y-1">
        <Label className="text-muted-foreground">
          {t("addToFactionLabel")}
        </Label>
        <textarea
          value={listToCsv(draft.add_to_faction_tags)}
          onChange={(e) =>
            onChange(
              "add_to_faction_tags",
              csvToList(e.target.value).map((s) => s.toUpperCase()),
            )
          }
          placeholder="HUN, ROM, BUL"
          className="w-full min-h-12 px-2 py-1 text-xs font-mono border rounded-md bg-background"
        />
        <p className="text-xs text-muted-foreground">
          {t.raw("addToFactionHelp")}
        </p>
      </div>

      <div className="space-y-1">
        <Label className="text-muted-foreground">{t("setAutonomyLabel")}</Label>
        {draft.set_autonomy.map((row, i) => (
          <div key={i} className="flex gap-2 items-center">
            <Input
              value={row.target}
              onChange={(e) => {
                const next = [...draft.set_autonomy];
                next[i] = {
                  ...next[i],
                  target: e.target.value.trim().toUpperCase(),
                };
                onChange("set_autonomy", next);
              }}
              placeholder={t("setAutonomyTargetLabel")}
              className="font-mono text-xs h-8"
            />
            <Input
              value={row.autonomous_state}
              list={diplomacyOptions ? autonomyListId : undefined}
              onChange={(e) => {
                const next = [...draft.set_autonomy];
                next[i] = {
                  ...next[i],
                  autonomous_state: e.target.value.trim(),
                };
                onChange("set_autonomy", next);
              }}
              placeholder={t("setAutonomyStateLabel")}
              className="font-mono text-xs h-8"
            />
            <Button
              type="button"
              variant="ghost"
              size="sm"
              className="px-2 text-xs shrink-0"
              onClick={() =>
                onChange(
                  "set_autonomy",
                  draft.set_autonomy.filter((_, j) => j !== i),
                )
              }
            >
              {t("removeRow")}
            </Button>
          </div>
        ))}
        <Button
          type="button"
          variant="outline"
          size="sm"
          className="h-7 px-2 text-xs"
          onClick={() =>
            onChange("set_autonomy", [
              ...draft.set_autonomy,
              { target: "", autonomous_state: "" },
            ])
          }
        >
          {t("addAutonomyRow")}
        </Button>
      </div>

      <div className="space-y-1">
        <Label className="text-muted-foreground">
          {t("diplomaticRelationLabel")}
        </Label>
        {draft.diplomatic_relation.map((row, i) => (
          <div key={i} className="flex gap-2 items-center">
            <Input
              value={row.country}
              onChange={(e) => {
                const next = [...draft.diplomatic_relation];
                next[i] = {
                  ...next[i],
                  country: e.target.value.trim().toUpperCase(),
                };
                onChange("diplomatic_relation", next);
              }}
              placeholder={t("diplomaticRelationCountryLabel")}
              className="font-mono text-xs h-8"
            />
            <Input
              value={row.relation}
              list={diplomacyOptions ? relationListId : undefined}
              onChange={(e) => {
                const next = [...draft.diplomatic_relation];
                next[i] = {
                  ...next[i],
                  relation: e.target.value.trim(),
                };
                onChange("diplomatic_relation", next);
              }}
              placeholder={t("diplomaticRelationRelationLabel")}
              className="font-mono text-xs h-8"
            />
            <Select
              value={row.active === false ? "no" : "yes"}
              onChange={(e) => {
                const next = [...draft.diplomatic_relation];
                next[i] = {
                  ...next[i],
                  active: e.target.value === "yes",
                };
                onChange("diplomatic_relation", next);
              }}
              className="h-8 px-2 text-xs shrink-0"
            >
              <option value="yes">{t("yes")}</option>
              <option value="no">{t("no")}</option>
            </Select>
            <Button
              type="button"
              variant="ghost"
              size="sm"
              className="px-2 text-xs shrink-0"
              onClick={() =>
                onChange(
                  "diplomatic_relation",
                  draft.diplomatic_relation.filter((_, j) => j !== i),
                )
              }
            >
              {t("removeRow")}
            </Button>
          </div>
        ))}
        <Button
          type="button"
          variant="outline"
          size="sm"
          className="h-7 px-2 text-xs"
          onClick={() =>
            onChange("diplomatic_relation", [
              ...draft.diplomatic_relation,
              { country: "", relation: "", active: true },
            ])
          }
        >
          {t("addRelationRow")}
        </Button>
      </div>
    </div>
  );

  const saveBar = (
    <div className="sticky bottom-0 space-y-2 rounded-md border border-border bg-card p-3">
      {dirty && !saving && !error && (
        <p role="status" className="text-xs">
          {t("unsaved")}
        </p>
      )}

      <DialogSaveStatusBar
        saving={saving}
        saved={saved}
        error={error}
        savingLabel={t("saving")}
        savedLabel={t("saved")}
      />

      <div className="flex gap-2">
        <Button
          onClick={onSave}
          disabled={saving || !draft.country_tag}
          size="sm"
        >
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

  if (section) {
    return (
      <fieldset disabled={deleting} className="p-4 space-y-3">
        {section === "ideas" ? ideasSection : diplomacySection}
        {saveBar}
      </fieldset>
    );
  }

  return (
    <fieldset disabled={deleting} className="p-4 space-y-3">
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
      <div className="space-y-2">
        <Label htmlFor="history-country">{t("appearance.selectCountry")}</Label>
        {isExisting ? (
          <p className="text-sm">
            {countryNames[draft.country_tag] || draft.country_tag}
          </p>
        ) : (
          <Select
            id="history-country"
            value={draft.country_tag}
            onChange={(event) => {
              const nextCountry = event.target.value;
              if (onSelectCountry) onSelectCountry(nextCountry);
              else onChange("country_tag", nextCountry);
            }}
          >
            <option value="">{t("appearance.noCountry")}</option>
            {draft.country_tag &&
              !countries.some(
                (country) => country.tag === draft.country_tag,
              ) && (
                <option value={draft.country_tag}>{draft.country_tag}</option>
              )}
            {countries
              .filter(
                (country) =>
                  !rows.some((row) => row.country_tag === country.tag),
              )
              .map((country) => (
                <option key={country.tag} value={country.tag}>
                  {countryNames[country.tag]}
                </option>
              ))}
          </Select>
        )}
      </div>
      <CountryAppearanceSettings
        key={`${projectId}:${activeId}:${draft.country_tag}`}
        projectId={projectId}
        countryTag={draft.country_tag}
        value={draft.set_cosmetic_tag}
        savedValue={savedAppearance}
        latestView={latestAppearance}
        disabled={saving}
        onChange={async (nextTag) => {
          if (onAppearanceChange) return onAppearanceChange(nextTag);
          onChange("set_cosmetic_tag", nextTag);
          return true;
        }}
      />
      {draft.set_cosmetic_tag &&
        (isExisting && draft.set_cosmetic_tag === savedAppearance ? (
          <CosmeticTagEditor
            key={`${projectId}:${draft.set_cosmetic_tag}`}
            projectId={projectId}
            tag={draft.set_cosmetic_tag}
            countryTag={draft.country_tag}
            countryNames={countryNames}
            onViewChange={onAppearanceView}
          />
        ) : (
          <p className="text-sm text-muted-foreground">
            {t("appearance.saveFirst")}
          </p>
        ))}
      <Separator />

      <div className="space-y-2">
        <Label className="text-sm font-semibold">{t("capitalLabel")}</Label>
        <Input
          type="number"
          min={0}
          value={draft.capital}
          onChange={(e) => onChange("capital", Number(e.target.value))}
          placeholder="361"
          className="text-xs h-8 w-40 font-mono"
        />
        <p className="text-xs text-muted-foreground">{t("capitalHelp")}</p>
      </div>

      <Separator />

      <div className="space-y-2">
        <div className="flex flex-wrap items-center justify-between gap-2">
          <Label className="text-sm font-semibold">{t("politicsLabel")}</Label>
          {onManageIdeologies && (
            <Button
              type="button"
              variant="ghost"
              size="sm"
              onClick={onManageIdeologies}
            >
              {t("manageIdeologies")}
            </Button>
          )}
        </div>
        <p className="text-xs text-muted-foreground">{t("ideologySyncHelp")}</p>
        <div className="grid grid-cols-2 gap-3">
          <div className="space-y-1">
            <Label>{t("rulingPartyLabel")}</Label>
            {rulingPartyCustom ? (
              <div className="flex gap-1 items-center">
                <Input
                  aria-label={t("rulingPartyLabel")}
                  list="hf-ruling-party-presets"
                  value={draft.set_politics_ruling_party}
                  onChange={(e) =>
                    onChange("set_politics_ruling_party", e.target.value)
                  }
                  placeholder="socialist"
                  className="font-mono text-xs h-8"
                />
                <datalist id="hf-ruling-party-presets">
                  {groupOptions.map((opt) => (
                    <option key={opt.value} value={opt.value} />
                  ))}
                </datalist>
                <Button
                  type="button"
                  variant="ghost"
                  size="sm"
                  className="px-2 text-xs shrink-0"
                  onClick={() => {
                    setRulingPartyCustom(false);
                    onChange("set_politics_ruling_party", "");
                  }}
                >
                  {t("rulingPartyPresets")}
                </Button>
              </div>
            ) : (
              <Select
                aria-label={t("rulingPartyLabel")}
                value={draft.set_politics_ruling_party}
                onChange={(e) => {
                  if (e.target.value === CUSTOM_IDEOLOGY_SENTINEL) {
                    setRulingPartyCustom(true);
                    onChange("set_politics_ruling_party", "");
                  } else {
                    onChange("set_politics_ruling_party", e.target.value);
                  }
                }}
                className="h-8 px-2 text-xs"
              >
                <option value="">{t("keepVanilla")}</option>
                {groupOptions.map((opt) => (
                  <option key={opt.value} value={opt.value}>
                    {opt.label}
                  </option>
                ))}
                <option value={CUSTOM_IDEOLOGY_SENTINEL}>
                  {t("rulingPartyCustom")}
                </option>
              </Select>
            )}
          </div>
          <div className="space-y-1">
            <Label>{t("lastElectionLabel")}</Label>
            <Input
              value={draft.set_politics_last_election}
              onChange={(e) =>
                onChange("set_politics_last_election", e.target.value)
              }
              placeholder="1936.11.3"
              className="font-mono text-xs h-8"
            />
          </div>
        </div>
        <div className="grid grid-cols-2 gap-3">
          <div className="space-y-1">
            <Label>{t("electionFreqLabel")}</Label>
            <Input
              type="number"
              min={0}
              value={draft.set_politics_election_frequency}
              onChange={(e) =>
                onChange(
                  "set_politics_election_frequency",
                  Number(e.target.value),
                )
              }
              className="text-xs h-8"
            />
          </div>
          <div className="space-y-1">
            <Label>{t("electionsAllowedLabel")}</Label>
            <Select
              value={
                draft.set_politics_elections_allowed === null
                  ? ""
                  : draft.set_politics_elections_allowed
                    ? "yes"
                    : "no"
              }
              onChange={(e) =>
                onChange(
                  "set_politics_elections_allowed",
                  e.target.value === "" ? null : e.target.value === "yes",
                )
              }
              className="h-8 px-2 text-xs"
            >
              <option value="">{t("keepVanilla")}</option>
              <option value="yes">{t("yes")}</option>
              <option value="no">{t("no")}</option>
            </Select>
          </div>
        </div>
      </div>

      <Separator />

      <div className="space-y-2">
        <div className="flex items-baseline justify-between gap-3">
          <Label className="text-sm font-semibold">
            {t("popularitiesLabel")}
          </Label>
          <span
            className={
              popularitiesTotal === 0 || popularitiesTotal === 100
                ? "text-xs text-muted-foreground"
                : "text-xs text-destructive font-medium"
            }
          >
            {popularitiesTotal}/100
          </span>
        </div>
        <p className="text-xs text-muted-foreground">
          {t("popularitiesRootHelp")}
        </p>
        <div className="grid grid-cols-2 gap-2">
          {popularityGroups.map((option) => (
            <div key={option.value} className="space-y-1">
              <Label
                htmlFor={`hf-popularity-${option.value}`}
                className="text-muted-foreground truncate"
                title={option.value}
              >
                {option.label}
              </Label>
              <Input
                id={`hf-popularity-${option.value}`}
                type="number"
                min={0}
                max={100}
                value={draft.set_popularities[option.value] ?? 0}
                onChange={(e) =>
                  updatePopularity(option.value, Number(e.target.value))
                }
                className="text-xs h-8"
              />
            </div>
          ))}
        </div>
      </div>

      <Separator />

      <div className="grid grid-cols-3 gap-3">
        <div className="space-y-1">
          <Label>{t("researchSlotsLabel")}</Label>
          <Input
            type="number"
            min={0}
            max={10}
            value={draft.set_research_slots}
            onChange={(e) =>
              onChange("set_research_slots", Number(e.target.value))
            }
            className="text-xs h-8"
          />
        </div>
        <div className="space-y-1">
          <Label>{t("stabilityLabel")}</Label>
          <Input
            type="number"
            step="0.05"
            min={-1}
            max={1}
            value={draft.set_stability}
            onChange={(e) => onChange("set_stability", Number(e.target.value))}
            className="text-xs h-8"
          />
        </div>
        <div className="space-y-1">
          <Label>{t("warSupportLabel")}</Label>
          <Input
            type="number"
            step="0.05"
            min={-1}
            max={1}
            value={draft.set_war_support}
            onChange={(e) =>
              onChange("set_war_support", Number(e.target.value))
            }
            className="text-xs h-8"
          />
        </div>
      </div>

      <div className="space-y-1">
        <Label>{t("setConvoysLabel")}</Label>
        <Input
          type="number"
          min={0}
          value={draft.set_convoys}
          onChange={(e) => onChange("set_convoys", Number(e.target.value))}
          placeholder="100"
          className="text-xs h-8 w-40 font-mono"
        />
        <p className="text-xs text-muted-foreground">
          {t.raw("setConvoysHelp")}
        </p>
      </div>

      <Separator />

      {ideasSection}

      <Separator />

      <div className="space-y-2">
        <Label className="text-sm font-semibold">{t("focusListLabel")}</Label>
        <textarea
          value={listToCsv(draft.complete_national_focus_ids)}
          onChange={(e) =>
            onChange("complete_national_focus_ids", csvToList(e.target.value))
          }
          placeholder="USA_war_department, USA_air_war_plans_division"
          className="w-full min-h-12 px-2 py-1 text-xs font-mono border rounded-md bg-background"
        />
        <p className="text-xs text-muted-foreground">{t("focusHelp")}</p>
      </div>

      <Separator />

      <div className="space-y-2">
        <Label className="text-sm font-semibold">
          {t("setTechnologyLabel")}
        </Label>
        <textarea
          value={listToCsv(draft.set_technology_ids)}
          onChange={(e) =>
            onChange("set_technology_ids", csvToList(e.target.value))
          }
          placeholder="infantry_weapons, fighter1, basic_light_tank"
          className="w-full min-h-16 px-2 py-1 text-xs font-mono border rounded-md bg-background"
        />
        <p className="text-xs text-muted-foreground">
          {t.raw("setTechnologyHelp")}
        </p>
      </div>

      <Separator />

      <div className="space-y-2">
        <Label className="text-sm font-semibold">{t("setOobLabel")}</Label>
        <div className="grid grid-cols-3 gap-3">
          <div className="space-y-1">
            <Label className="text-muted-foreground">
              {t("setArmyOobLabel")}
            </Label>
            <Input
              value={draft.set_oob}
              onChange={(e) => onChange("set_oob", e.target.value.trim())}
              placeholder="USA_1936"
              className="font-mono text-xs h-8"
            />
          </div>
          <div className="space-y-1">
            <Label className="text-muted-foreground">
              {t("setAirOobLabel")}
            </Label>
            <Input
              value={draft.set_air_oob}
              onChange={(e) => onChange("set_air_oob", e.target.value.trim())}
              placeholder="USA_1936_air"
              className="font-mono text-xs h-8"
            />
          </div>
          <div className="space-y-1">
            <Label className="text-muted-foreground">
              {t("setNavalOobLabel")}
            </Label>
            <Input
              value={draft.set_naval_oob}
              onChange={(e) => onChange("set_naval_oob", e.target.value.trim())}
              placeholder="USA_1936_naval"
              className="font-mono text-xs h-8"
            />
          </div>
        </div>
        <p className="text-xs text-muted-foreground">{t.raw("setOobHelp")}</p>
      </div>

      <Separator />

      <div className="space-y-2">
        <Label className="text-sm font-semibold">
          {t("startupFlagsLabel")}
        </Label>
        <textarea
          value={listToCsv(draft.startup_flags)}
          onChange={(e) => onChange("startup_flags", csvToList(e.target.value))}
          placeholder={"modern_usa_arrived\nmodusa_starting_state"}
          className="w-full min-h-16 px-2 py-1 text-xs font-mono border rounded-md bg-background"
        />
        <p className="text-xs text-muted-foreground">
          {t.rich("startupFlagsHelp", {
            id: "<id>",
            mono: (c) => <span className="font-mono">{c}</span>,
          })}
        </p>
      </div>

      <Separator />

      {diplomacySection}

      <Separator />

      <details className="rounded-md border border-border p-3">
        <summary className="cursor-pointer text-sm ui-focus-ring ui-interactive">
          {t("advancedScriptDetails")}
        </summary>
        <div className="space-y-3 pt-3">
          <HistoryDateBlocksEditor
            blocks={draft.date_blocks}
            onChange={(dateBlocks) => onChange("date_blocks", dateBlocks)}
            disabled={saving}
            labels={{
              title: t("dateBlocksTitle"),
              description: t("dateBlocksDescription"),
              add: t("dateBlocksAdd"),
              empty: t("dateBlocksEmpty"),
              date: t("dateBlocksDate"),
              script: t("dateBlocksScript"),
              moveUp: t("dateBlocksMoveUp"),
              moveDown: t("dateBlocksMoveDown"),
              delete: t("dateBlocksDelete"),
              validation: {
                format: t("dateBlocksValidationFormat"),
                range: t("dateBlocksValidationRange"),
                hour: t("dateBlocksValidationHour"),
              },
            }}
          />

          <Separator />

          <OobPassthroughPanel
            value={draft.raw_script}
            onChange={(next) => onChange("raw_script", next)}

            countryTag={draft.country_tag}
            oobFile={draft.set_oob}
            startDate={projectStartDate}
          />

          <Separator />
        </div>
      </details>
      {saveBar}
    </fieldset>
  );
}
