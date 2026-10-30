"use client";

import { useCallback, useRef, useState } from "react";
import { useTranslations } from "use-intl";
import { Plus, Trash2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Separator } from "@/components/ui/separator";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { useContentLocaleStore } from "@/lib/store/contentLocaleStore";
import { useProjectStore } from "@/lib/store/projectStore";
import { DisplayNameLocsTabs } from "@/components/shared/editors/DisplayNameLocsTabs";
import { DEFAULT_EXPORT_LANGUAGES } from "@/components/shared/editors/displayNameLocsHelpers";
import { resolveDisplayName } from "@/lib/i18n/localizedName";
import type { State, StateUpdate } from "@/types";
import {
  draftToStateUpdate,
  isStateFormDirty,
  resolvedStateName,
  seedStateForm,
  validateProvinceDraft,
  type ProvinceBuildingRow,
  type StateFormDraft,
} from "@/components/workbench/stateFormState";
import { useRegisterDirty } from "@/lib/hooks/useUnsavedGuard";
import {
  checkImpassableSave,
  type ImpassableSaveGuardResult,
} from "@/lib/states/impassableSaveGuard";

const STATE_CATEGORY_HINTS = [
  "wasteland",
  "small_island",
  "tiny_island",
  "pastoral",
  "rural",
  "town",
  "large_town",
  "city",
  "large_city",
  "metropolis",
  "megalopolis",
  "enclave",
];

const STATE_BUILDING_HINTS = [
  "infrastructure",
  "arms_factory",
  "industrial_complex",
  "dockyard",
  "air_base",
  "anti_air_building",
  "synthetic_refinery",
  "fuel_silo",
  "rocket_site",
  "nuclear_reactor",
  "radar_station",
  "supply_node",
];
const RESOURCE_HINTS = [
  "oil",
  "aluminium",
  "rubber",
  "tungsten",
  "steel",
  "chromium",
];

interface StateEditorProps {
  state: State;
  countryTags: string[];
  onSave: (id: number, data: StateUpdate) => Promise<State>;
}

const VANILLA_IMPASSABLE_CATEGORIES: ReadonlySet<string> = new Set([
  "wasteland",
  "enclave",
]);

export function StateEditor({ state, countryTags, onSave }: StateEditorProps) {
  const t = useTranslations("states");
  const tc = useTranslations("common");
  const tContent = useTranslations("decisions");
  const locale = useContentLocaleStore((s) => s.locale);

  const exportLanguages = useProjectStore(
    (s) => s.currentProject?.export_languages ?? DEFAULT_EXPORT_LANGUAGES,
  );

  const [draft, setDraft] = useState<StateFormDraft>(() =>
    seedStateForm(state),
  );
  const [saving, setSaving] = useState(false);

  const [saveStatus, setSaveStatus] = useState<"idle" | "success" | "error">(
    "idle",
  );
  const [saveError, setSaveError] = useState<string | null>(null);

  const [blocked, setBlocked] = useState<{
    field: string;
    message: string;
  } | null>(null);
  const forceLinkRef = useRef<HTMLInputElement>(null);

  const seededIdRef = useRef(state.id);
  if (seededIdRef.current !== state.id) {
    seededIdRef.current = state.id;
    setDraft(seedStateForm(state));
    setSaveStatus("idle");
    setSaveError(null);
    setBlocked(null);
  }

  const dirty = isStateFormDirty(draft, state);
  const provinceValidation = validateProvinceDraft(draft.provinces);

  useRegisterDirty("states", dirty);

  const patch = useCallback((p: Partial<StateFormDraft>) => {
    setDraft((d) => ({ ...d, ...p }));

    if ("impassable" in p || "force_link_ownership_to" in p) setBlocked(null);
  }, []);

  const handleSave = async () => {
    if (!provinceValidation.valid) return;

    const guard: ImpassableSaveGuardResult = checkImpassableSave({
      impassable: draft.impassable,
      force_link_ownership_to: draft.force_link_ownership_to,
    });
    if (!guard.ok) {
      setBlocked({ field: guard.field!, message: t(guard.messageKey!) });
      setSaveStatus("idle");
      setSaveError(null);

      forceLinkRef.current?.focus();
      return;
    }
    setBlocked(null);
    setSaving(true);
    setSaveStatus("idle");
    setSaveError(null);
    try {
      await onSave(state.id, draftToStateUpdate(draft, state));
      setSaveStatus("success");
      setTimeout(() => setSaveStatus("idle"), 2000);
    } catch (e) {
      setSaveError(e instanceof Error ? e.message : String(e));
      setSaveStatus("error");
    } finally {
      setSaving(false);
    }
  };

  const handleRevert = () => {
    setDraft(seedStateForm(state));
    setSaveStatus("idle");
    setSaveError(null);
    setBlocked(null);
  };

  const resolvedName = resolveDisplayName(
    locale,
    state.display_name ?? state.display_name_locs,
    null,
    null,
    "",
  );

  return (
    <div className="p-4 space-y-5 max-w-2xl mx-auto">
      {(saving || saveStatus !== "idle" || blocked || dirty) && (
        <div
          className={`
            sticky top-0 z-10 -mx-4 -mt-4 px-4 py-1.5 text-xs font-medium flex items-center gap-2 shrink-0
            transition-[opacity,background-color,border-color] duration-300
            ${
              saving
                ? "bg-muted/80 text-muted-foreground border-b border-border"
                : blocked
                  ? "bg-destructive/10 text-destructive border-b border-destructive/20"
                  : saveStatus === "success"
                    ? "bg-success/10 text-success border-b border-success/20"
                    : saveStatus === "error"
                      ? "bg-destructive/10 text-destructive border-b border-destructive/20"
                      : dirty
                        ? "bg-warning/10 text-warning border-b border-warning/20"
                        : ""
            }
          `}
        >
          <span role={blocked ? "alert" : undefined}>
            {saving
              ? t("saving")
              : blocked
                ? blocked.message
                : saveStatus === "success"
                  ? t("savedShort")
                  : saveStatus === "error"
                    ? saveError
                      ? t("saveFailedDetail", { msg: saveError })
                      : t("saveFailed")
                    : dirty
                      ? t("unsavedChanges")
                      : ""}
          </span>
        </div>
      )}

      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0">
          <div className="flex items-center gap-2">
            <h3 className="text-base font-semibold tracking-tight">
              {t("stateNo", { id: state.state_id })}
            </h3>
            <Badge variant="outline" className="text-micro font-mono">
              {state.source === "imported"
                ? t("badgeImported")
                : t("badgeCreated")}
            </Badge>
          </div>
          <p
            className="text-xs text-muted-foreground truncate"
            title={state.name}
          >
            {resolvedName ? (
              <>
                <span>{resolvedName}</span>
                <span className="font-mono ml-2 opacity-70">{state.name}</span>
              </>
            ) : (
              <span className="font-mono">{state.name || "—"}</span>
            )}
          </p>
        </div>
        <div className="flex items-center gap-2 shrink-0">
          {dirty && (
            <Button
              size="sm"
              variant="ghost"
              onClick={handleRevert}
              disabled={saving}
            >
              {tc("revert")}
            </Button>
          )}
          <Button
            size="sm"
            onClick={handleSave}
            disabled={!dirty || saving || !provinceValidation.valid}
          >
            {saving ? tc("saving") : tc("save")}
          </Button>
        </div>
      </div>

      <section className="space-y-3">
        <h3 className="text-sm font-semibold text-muted-foreground uppercase tracking-wide">
          {t("sectionPrimary")}
        </h3>

        <Field label={t("displayName")} hint={t("displayNameHint")}>
          <DisplayNameLocsTabs
            entityKey={`state:${state.id}:display-name`}
            value={draft.displayNameLocs}
            exportLanguages={exportLanguages}
            legacyFallback={{
              en: resolvedStateName(state, "en"),
              zh: resolvedStateName(state, "zh"),
            }}
            onChange={(next) => patch({ displayNameLocs: next })}
            ariaLabel={tContent("contentNameLabel", {
              locale: locale.toUpperCase(),
            })}
          />
        </Field>

        <Field label={t("provinces")} hint={t("provincesHint")}>
          <Textarea
            value={draft.provinces}
            onChange={(event) => patch({ provinces: event.target.value })}
            placeholder={t("provincesPlaceholder")}
            spellCheck={false}
            rows={3}
            aria-invalid={!provinceValidation.valid}
            className={listTextareaCls}
          />
          <p className="text-micro text-muted-foreground">
            {t("provinceDraftCount", {
              count: provinceValidation.values.length,
            })}
          </p>
          {provinceValidation.invalidTokens.length > 0 && (
            <p role="alert" className="text-xs text-destructive">
              {t("provinceInvalidTokens", {
                tokens: provinceValidation.invalidTokens.join(", "),
              })}
            </p>
          )}
          {provinceValidation.duplicateIds.length > 0 && (
            <p role="alert" className="text-xs text-destructive">
              {t("provinceDuplicateIds", {
                ids: provinceValidation.duplicateIds.join(", "),
              })}
            </p>
          )}
          {provinceValidation.empty && (
            <p className="text-xs text-warning">{t("provinceEmptyDraft")}</p>
          )}
        </Field>

        <MembershipProvenance state={state} />

        <Field label={t("owner")} hint={t("ownerHint")}>
          <Input
            list="state-owner-tags"
            value={draft.owner}
            onChange={(e) => patch({ owner: e.target.value })}
            placeholder="GER"
            className={tagInputCls}
          />
          <datalist id="state-owner-tags">
            {countryTags.map((tg) => (
              <option key={tg} value={tg} />
            ))}
          </datalist>
        </Field>

        <Field label={t("cores")} hint={t("coresHint")}>
          <ListEditor
            values={draft.cores}
            onChange={(cores) => patch({ cores })}
            placeholder="TAG"
            addLabel={t("addCore")}
            datalistId="state-core-tags"
            datalistOptions={countryTags}
          />
        </Field>

        <Field label={t("claims")} hint={t("claimsHint")}>
          <ListEditor
            values={draft.claims}
            onChange={(claims) => patch({ claims })}
            placeholder="TAG"
            addLabel={t("addClaim")}
            datalistId="state-core-tags"
            datalistOptions={countryTags}
          />
        </Field>

        <Field label={t("removeCores")} hint={t("removeCoresHint")}>
          <ListEditor
            values={draft.removeCores}
            onChange={(removeCores) => patch({ removeCores })}
            placeholder="TAG"
            addLabel={t("addRemoveCore")}
            datalistId="state-core-tags"
            datalistOptions={countryTags}
          />
        </Field>

        <Field label={t("manpower")}>
          <Input
            type="number"
            value={draft.manpower}
            onChange={(e) => patch({ manpower: e.target.value })}
            className={numInputCls}
          />
        </Field>

        <Field label={t("buildings")} hint={t("buildingsHint")}>
          <KvEditor
            rows={draft.buildingRows}
            onChange={(buildingRows) => patch({ buildingRows })}
            keyHints={STATE_BUILDING_HINTS}
            keyPlaceholder={t("buildingKeyPh")}
            addLabel={t("addBuilding")}
            datalistId="state-building-keys"
          />
        </Field>

        <Field label={t("provinceBuildings")} hint={t("provinceBuildingsHint")}>
          <ProvinceBuildingsEditor
            rows={draft.provinceBuildings}
            onChange={(provinceBuildings) => patch({ provinceBuildings })}
            t={t}
          />
        </Field>
      </section>

      <Separator />

      <section className="space-y-3">
        <h3 className="text-sm font-semibold text-muted-foreground uppercase tracking-wide">
          {t("sectionSecondary")}
        </h3>

        <Field label={t("controller")} hint={t("controllerHint")}>
          <Input
            list="state-owner-tags"
            value={draft.controller}
            onChange={(e) => patch({ controller: e.target.value })}
            placeholder={t("controllerPh")}
            className={tagInputCls}
          />
        </Field>

        <Field label={t("victoryPoints")}>
          <VictoryPointsEditor
            rows={draft.victoryPoints}
            onChange={(victoryPoints) => patch({ victoryPoints })}
            t={t}
          />
        </Field>

        <Field
          label={t("provinceControllers")}
          hint={t("provinceControllersHint")}
        >
          <ProvinceControllersEditor
            rows={draft.provinceControllers}
            onChange={(provinceControllers) => patch({ provinceControllers })}
            countryTags={countryTags}
            t={t}
          />
        </Field>

        <Field label={t("resources")}>
          <KvEditor
            rows={draft.resourceRows}
            onChange={(resourceRows) => patch({ resourceRows })}
            keyHints={RESOURCE_HINTS}
            keyPlaceholder={t("resourceKeyPh")}
            addLabel={t("addResource")}
            datalistId="state-resource-keys"
          />
        </Field>

        <Field label={t("stateCategory")} hint={t("stateCategoryHint")}>
          <Input
            list="state-category-hints"
            value={draft.state_category}
            onChange={(e) => patch({ state_category: e.target.value })}
            placeholder="rural"
            className={tagInputCls}
          />
          <datalist id="state-category-hints">
            {STATE_CATEGORY_HINTS.map((c) => (
              <option key={c} value={c} />
            ))}
          </datalist>
        </Field>

        <div className="grid grid-cols-2 gap-3 items-end">
          <Field label={t("localSupplies")} hint={t("localSuppliesHint")}>
            <Input
              type="number"
              step="0.1"
              value={draft.local_supplies}
              onChange={(e) => patch({ local_supplies: e.target.value })}
              placeholder="—"
              className={numInputCls}
            />
          </Field>
          <Field
            label={t("buildingsMaxLevelFactor")}
            hint={t("buildingsMaxLevelFactorHint")}
          >
            <Input
              type="number"
              step="0.1"
              value={draft.buildings_max_level_factor}
              onChange={(e) =>
                patch({ buildings_max_level_factor: e.target.value })
              }
              placeholder="—"
              className={numInputCls}
            />
          </Field>
          <Field label={t("impassable")} hint={t("impassableHint")}>
            <label className="flex items-center gap-2 h-8 text-sm">
              <input
                type="checkbox"
                checked={draft.impassable}
                onChange={(e) => patch({ impassable: e.target.checked })}
                className="h-4 w-4"
              />
              <span className="text-muted-foreground">
                {t("impassableLabel")}
              </span>
            </label>

            {draft.impassable &&
              !VANILLA_IMPASSABLE_CATEGORIES.has(
                (draft.state_category || "").trim().toLowerCase(),
              ) && (
                <p className="pt-1 text-micro leading-snug text-muted-foreground">
                  {t("impassableCategoryMismatch", {
                    category: draft.state_category || "—",
                  })}
                </p>
              )}
          </Field>
        </div>

        <Field
          label={t("forceLinkOwnershipTo")}
          hint={t("forceLinkOwnershipToHint")}
        >
          <Input
            ref={forceLinkRef}
            type="number"
            step="1"
            min="0"
            value={draft.force_link_ownership_to}
            onChange={(e) => patch({ force_link_ownership_to: e.target.value })}
            placeholder="—"
            aria-invalid={blocked?.field === "force_link_ownership_to"}
            className={numInputCls}
          />

          {blocked?.field === "force_link_ownership_to" && (
            <p role="alert" className="text-xs text-destructive">
              {blocked.message}
            </p>
          )}
        </Field>

        <Field
          label={t("impassableIgnoredLinks")}
          hint={t("impassableIgnoredLinksHint")}
        >
          <Textarea
            value={draft.impassable_ignored_links}
            onChange={(e) =>
              patch({ impassable_ignored_links: e.target.value })
            }
            placeholder="—"
            spellCheck={false}
            rows={2}
            className={listTextareaCls}
          />
        </Field>
      </section>

      <Separator />

      <section className="space-y-2">
        <h3 className="text-sm font-semibold text-muted-foreground uppercase tracking-wide">
          {t("sectionReadonly")}
        </h3>
        <p className="text-xs text-muted-foreground">{t("readonlyHint")}</p>
        <ReadonlyRow label={t("stateId")}>
          <span className="font-mono">{state.state_id}</span>
        </ReadonlyRow>
        <ReadonlyRow label={t("name")}>
          <span className="font-mono">{state.name || "—"}</span>
          <span className="ml-2 text-micro text-muted-foreground italic">
            {t("nameLocNote")}
          </span>
        </ReadonlyRow>
        {Object.keys(state.raw_extras ?? {}).length > 0 && (
          <ReadonlyRow label={t("rawExtras")}>
            <span className="text-xs text-muted-foreground">
              {t("rawExtrasNote", {
                count: Object.keys(state.raw_extras).length,
              })}
            </span>
          </ReadonlyRow>
        )}
      </section>
    </div>
  );
}

function MembershipProvenance({ state }: { state: State }) {
  const t = useTranslations("states");
  const provenance = state.membership_provenance;
  const known = provenance?.status === "known";
  const current = provenance?.current ?? state.provinces ?? [];
  const source = known ? (provenance.source ?? []) : null;
  const sourcePath =
    state.source_snapshot?.source_path ?? state.source_file ?? null;
  const formatIds = (values: number[] | null) =>
    values == null
      ? t("membershipUnknownValue")
      : values.length
        ? values.join(" ")
        : "—";
  const formatFlag = (value: boolean | null | undefined) =>
    value == null
      ? t("membershipFlagUnknown")
      : value
        ? t("membershipFlagChanged")
        : t("membershipFlagUnchanged");

  return (
    <div className="space-y-2 rounded-lg border border-border bg-muted/30 p-3">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <p className="text-xs font-semibold text-foreground">
          {t("membershipEvidenceTitle")}
        </p>
        <Badge variant="outline" className="text-micro">
          {known
            ? t("membershipStatusKnown")
            : t("membershipStatusLegacyUnknown")}
        </Badge>
      </div>

      {!known && (
        <details className="text-xs text-muted-foreground">
          <summary className="cursor-pointer select-none">
            {t("membershipLegacyUnknownSummary")}
          </summary>
          <p className="pt-1 leading-relaxed">
            {t("membershipLegacyUnknownHint")}
          </p>
        </details>
      )}
      {sourcePath && (
        <p className="break-all font-mono text-micro text-muted-foreground">
          {t("membershipSourcePath", { path: sourcePath })}
        </p>
      )}
      <div className="grid gap-1.5 text-xs sm:grid-cols-2">
        <EvidenceRow label={t("membershipSource")} value={formatIds(source)} />
        <EvidenceRow
          label={t("membershipCurrent")}
          value={formatIds(current)}
        />
        <EvidenceRow
          label={t("membershipAdded")}
          value={formatIds(known ? (provenance?.added ?? []) : null)}
        />
        <EvidenceRow
          label={t("membershipRemoved")}
          value={formatIds(known ? (provenance?.removed ?? []) : null)}
        />
        <EvidenceRow
          label={t("membershipChanged")}
          value={formatIds(known ? (provenance?.changed ?? []) : null)}
        />
      </div>
      <div className="grid gap-1.5 border-t border-border/60 pt-2 text-xs sm:grid-cols-3">
        <EvidenceRow
          label={t("membershipFlag")}
          value={formatFlag(provenance?.membership_changed)}
        />
        <EvidenceRow
          label={t("ownerFlag")}
          value={formatFlag(provenance?.owner_changed)}
        />
        <EvidenceRow
          label={t("controllerFlag")}
          value={formatFlag(provenance?.controller_changed)}
        />
      </div>
    </div>
  );
}

function EvidenceRow({ label, value }: { label: string; value: string }) {
  return (
    <div className="min-w-0">
      <span className="text-micro text-muted-foreground">{label}</span>
      <p className="break-all font-mono text-xs text-foreground">{value}</p>
    </div>
  );
}

const inputBaseCls = "";
const tagInputBaseCls = "font-mono";
const tagInputCls = "font-mono";
const numInputCls = "";

const listTextareaCls = "font-mono resize-y";

function Field({
  label,
  hint,
  children,
}: {
  label: string;
  hint?: string;
  children: React.ReactNode;
}) {
  return (
    <div className="space-y-1">
      <label className="text-xs font-medium text-foreground">{label}</label>
      {hint && <p className="text-micro text-muted-foreground">{hint}</p>}
      {children}
    </div>
  );
}

function ReadonlyRow({
  label,
  children,
}: {
  label: string;
  children: React.ReactNode;
}) {
  return (
    <div className="flex items-baseline gap-2 text-sm">
      <span className="text-xs text-muted-foreground w-24 shrink-0">
        {label}
      </span>
      <span className="min-w-0">{children}</span>
    </div>
  );
}

function ListEditor({
  values,
  onChange,
  placeholder,
  addLabel,
  datalistId,
  datalistOptions,
}: {
  values: string[];
  onChange: (v: string[]) => void;
  placeholder: string;
  addLabel: string;
  datalistId?: string;
  datalistOptions?: string[];
}) {
  return (
    <div className="space-y-1.5">
      {values.map((v, i) => (
        <div key={i} className="flex items-center gap-1.5">
          <Input
            list={datalistId}
            value={v}
            onChange={(e) => {
              const next = [...values];
              next[i] = e.target.value;
              onChange(next);
            }}
            placeholder={placeholder}
            className={tagInputCls}
          />
          <RemoveBtn
            onClick={() => onChange(values.filter((_, j) => j !== i))}
          />
        </div>
      ))}
      {datalistId && datalistOptions && (
        <datalist id={datalistId}>
          {datalistOptions.map((o) => (
            <option key={o} value={o} />
          ))}
        </datalist>
      )}
      <AddBtn label={addLabel} onClick={() => onChange([...values, ""])} />
    </div>
  );
}

function KvEditor({
  rows,
  onChange,
  keyHints,
  keyPlaceholder,
  addLabel,
  datalistId,
}: {
  rows: [string, string][];
  onChange: (r: [string, string][]) => void;
  keyHints: string[];
  keyPlaceholder: string;
  addLabel: string;
  datalistId: string;
}) {
  return (
    <div className="space-y-1.5">
      {rows.map(([k, v], i) => (
        <div key={i} className="flex items-center gap-1.5">
          <Input
            list={datalistId}
            value={k}
            onChange={(e) => {
              const next = [...rows];
              next[i] = [e.target.value, v];
              onChange(next);
            }}
            placeholder={keyPlaceholder}
            className={tagInputBaseCls + " min-w-0 flex-1"}
          />
          <Input
            type="number"
            step={1}
            min={0}
            value={v}
            onChange={(e) => {
              const next = [...rows];
              next[i] = [k, e.target.value];
              onChange(next);
            }}
            className={inputBaseCls + " w-24 shrink-0"}
          />
          <RemoveBtn onClick={() => onChange(rows.filter((_, j) => j !== i))} />
        </div>
      ))}
      <datalist id={datalistId}>
        {keyHints.map((h) => (
          <option key={h} value={h} />
        ))}
      </datalist>
      <AddBtn label={addLabel} onClick={() => onChange([...rows, ["", "0"]])} />
    </div>
  );
}

function VictoryPointsEditor({
  rows,
  onChange,
  t,
}: {
  rows: { province_id: string; value: string }[];
  onChange: (r: { province_id: string; value: string }[]) => void;
  t: ReturnType<typeof useTranslations>;
}) {
  return (
    <div className="space-y-1.5">
      {rows.map((vp, i) => (
        <div key={i} className="flex items-center gap-1.5">
          <Input
            type="number"
            value={vp.province_id}
            onChange={(e) => {
              const next = [...rows];
              next[i] = { ...vp, province_id: e.target.value };
              onChange(next);
            }}
            placeholder={t("vpProvincePh")}
            className={inputBaseCls + " min-w-0 flex-1"}
          />
          <Input
            type="number"
            value={vp.value}
            onChange={(e) => {
              const next = [...rows];
              next[i] = { ...vp, value: e.target.value };
              onChange(next);
            }}
            placeholder={t("vpValuePh")}
            className={inputBaseCls + " w-24 shrink-0"}
          />
          <RemoveBtn onClick={() => onChange(rows.filter((_, j) => j !== i))} />
        </div>
      ))}
      <AddBtn
        label={t("addVictoryPoint")}
        onClick={() => onChange([...rows, { province_id: "", value: "1" }])}
      />
    </div>
  );
}

const PROVINCE_BUILDING_HINTS = [
  "naval_base",
  "coastal_bunker",
  "bunker",
  "floating_harbor",
];

function ProvinceBuildingsEditor({
  rows,
  onChange,
  t,
}: {
  rows: ProvinceBuildingRow[];
  onChange: (r: ProvinceBuildingRow[]) => void;
  t: ReturnType<typeof useTranslations>;
}) {
  return (
    <div className="space-y-1.5">
      {rows.map((r, i) => (
        <div key={i} className="flex items-center gap-1.5">
          <Input
            type="number"
            value={r.province_id}
            onChange={(e) => {
              const next = [...rows];
              next[i] = { ...r, province_id: e.target.value };
              onChange(next);
            }}
            placeholder={t("pbProvincePh")}
            className={inputBaseCls + " w-24 shrink-0"}
          />
          <Input
            list="state-province-building-keys"
            value={r.building}
            onChange={(e) => {
              const next = [...rows];
              next[i] = { ...r, building: e.target.value };
              onChange(next);
            }}
            placeholder={t("pbBuildingPh")}
            className={tagInputBaseCls + " min-w-0 flex-1"}
          />
          <Input
            type="number"
            step={1}
            min={0}
            value={r.level}
            onChange={(e) => {
              const next = [...rows];
              next[i] = { ...r, level: e.target.value };
              onChange(next);
            }}
            placeholder={t("pbLevelPh")}
            className={inputBaseCls + " w-20 shrink-0"}
          />
          <RemoveBtn onClick={() => onChange(rows.filter((_, j) => j !== i))} />
        </div>
      ))}
      <datalist id="state-province-building-keys">
        {PROVINCE_BUILDING_HINTS.map((h) => (
          <option key={h} value={h} />
        ))}
      </datalist>
      <AddBtn
        label={t("addProvinceBuilding")}
        onClick={() =>
          onChange([...rows, { province_id: "", building: "", level: "1" }])
        }
      />
    </div>
  );
}

function ProvinceControllersEditor({
  rows,
  onChange,
  countryTags,
  t,
}: {
  rows: { province_id: string; controller: string }[];
  onChange: (r: { province_id: string; controller: string }[]) => void;
  countryTags: string[];
  t: ReturnType<typeof useTranslations>;
}) {
  return (
    <div className="space-y-1.5">
      {rows.map((pc, i) => (
        <div key={i} className="flex items-center gap-1.5">
          <Input
            type="number"
            value={pc.province_id}
            onChange={(e) => {
              const next = [...rows];
              next[i] = { ...pc, province_id: e.target.value };
              onChange(next);
            }}
            placeholder={t("pcProvincePh")}
            className={inputBaseCls + " min-w-0 flex-1"}
          />
          <Input
            list="state-province-controller-tags"
            value={pc.controller}
            onChange={(e) => {
              const next = [...rows];
              next[i] = { ...pc, controller: e.target.value };
              onChange(next);
            }}
            placeholder={t("pcControllerPh")}
            className={tagInputBaseCls + " w-28 shrink-0"}
          />
          <RemoveBtn onClick={() => onChange(rows.filter((_, j) => j !== i))} />
        </div>
      ))}
      <datalist id="state-province-controller-tags">
        {countryTags.map((tg) => (
          <option key={tg} value={tg} />
        ))}
      </datalist>
      <AddBtn
        label={t("addProvinceController")}
        onClick={() => onChange([...rows, { province_id: "", controller: "" }])}
      />
    </div>
  );
}

function AddBtn({ label, onClick }: { label: string; onClick: () => void }) {
  return (
    <button
      type="button"
      onClick={onClick}
      className="flex items-center gap-1 text-xs text-muted-foreground hover:text-foreground ui-interactive"
    >
      <Plus className="h-3 w-3" /> {label}
    </button>
  );
}

function RemoveBtn({ onClick }: { onClick: () => void }) {
  const t = useTranslations("states");
  return (
    <button
      type="button"
      onClick={onClick}
      className="text-muted-foreground hover:text-destructive shrink-0 ui-interactive"
      title={t("remove")}
    >
      <Trash2 className="h-3.5 w-3.5" />
    </button>
  );
}
