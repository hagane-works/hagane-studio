"use client";

import { useMemo, useState } from "react";
import useSWR from "swr";
import { useTranslations } from "use-intl";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { DisplayNameLocsTabs } from "@/components/shared/editors/DisplayNameLocsTabs";
import { DEFAULT_EXPORT_LANGUAGES } from "@/components/shared/editors/displayNameLocsHelpers";
import { vanillaDataApi } from "@/lib/api";
import { equipmentTokenLabel } from "@/lib/i18n/equipmentGloss";
import { useVanillaUnitNames } from "@/lib/vanillaUnitNames";
import { useRegisterDirty } from "@/lib/hooks/useUnsavedGuard";
import { useLocaleStore } from "@/lib/store/localeStore";
import { useProjectStore } from "@/lib/store/projectStore";
import type { Equipment } from "@/types";
import {
  AIR_FIELDS,
  DEFENSIVE_FIELDS,
  ECONOMY_FIELDS,
  NAVY_FIELDS,
  OFFENSIVE_FIELDS,
  equipmentDraftToPayload,
  equipmentToDraft,
  type EquipmentDraft,
  type EquipmentNumericKey,
  type TriState,
} from "./equipmentForm";

const SUGGESTED_EQUIPMENT_TYPES = [
  "infantry",
  "motorized",
  "mechanized",
  "armor",
  "artillery",
  "anti_air",
  "anti_tank",
  "rocket",
  "support_equipment",
  "light_tank_chassis",
  "medium_tank_chassis",
  "heavy_tank_chassis",
  "ship",
  "submarine",
  "convoy",
  "fighter",
  "cas",
  "naval_bomber",
  "tactical_bomber",
  "strategic_bomber",
  "scout_plane",
];

const STAT_LABEL: Record<EquipmentNumericKey, string> = {
  soft_attack: "softAttack",
  hard_attack: "hardAttack",
  ap_attack: "apAttack",
  air_attack: "airAttack",
  breakthrough: "breakthrough",
  defense: "defense",
  armor_value: "armorValue",
  hardness: "hardness",
  max_strength: "maxStrength",
  build_cost_ic: "buildCostIc",
  manpower: "manpower",
  reliability: "reliability",
  maximum_speed: "maximumSpeed",
  naval_speed: "navalSpeed",
  fire_range: "fireRange",
  lg_armor_piercing: "lgArmorPiercing",
  lg_attack: "lgAttack",
  hg_armor_piercing: "hgArmorPiercing",
  hg_attack: "hgAttack",
  torpedo_attack: "torpedoAttack",
  anti_air_attack: "antiAirAttack",
  shore_bombardment: "shoreBombardment",
  evasion: "evasion",
  surface_detection: "surfaceDetection",
  sub_attack: "subAttack",
  sub_detection: "subDetection",
  surface_visibility: "surfaceVisibility",
  sub_visibility: "subVisibility",
  naval_range: "navalRange",
  port_capacity_usage: "portCapacityUsage",
  search_and_destroy_coordination: "searchAndDestroyCoordination",
  convoy_raiding_coordination: "convoyRaidingCoordination",
  air_defence: "airDefence",
  air_range: "airRange",
  air_agility: "airAgility",
  air_ground_attack: "airGroundAttack",
  air_bombing: "airBombing",
  air_superiority: "airSuperiority",
  naval_strike_attack: "navalStrikeAttack",
  naval_strike_targetting: "navalStrikeTargetting",
  carrier_size: "carrierSize",
  default_carrier_composition_weight: "defaultCarrierCompositionWeight",
};

const EMPTY_ARCHETYPES: readonly string[] = [];

export function EquipmentEditor({
  equipment,
  list,
  onSave,
  error = null,
}: {
  equipment: Equipment;

  list: readonly Equipment[];

  onSave: (payload: Partial<Equipment>) => Promise<void>;

  error?: string | null;
}) {
  const t = useTranslations("equipmentEditor");
  const locale = useLocaleStore((s) => s.locale);
  const unitNames = useVanillaUnitNames();
  const exportLanguages = useProjectStore(
    (s) => s.currentProject?.export_languages ?? DEFAULT_EXPORT_LANGUAGES,
  );
  const vanilla = useSWR("vanilla/equipment-archetypes", () =>
    vanillaDataApi.equipmentArchetypes(),
  );
  const vanillaArchetypes = vanilla.data?.archetypes ?? EMPTY_ARCHETYPES;

  const [draft, setDraft] = useState<EquipmentDraft>(() =>
    equipmentToDraft(equipment),
  );
  const [savedDraft, setSavedDraft] = useState<EquipmentDraft>(draft);
  const [saving, setSaving] = useState(false);
  const [saveStatus, setSaveStatus] = useState<"idle" | "success" | "error">(
    "idle",
  );
  const [saveError, setSaveError] = useState<string | null>(null);

  const isUnsaved = JSON.stringify(draft) !== JSON.stringify(savedDraft);
  useRegisterDirty("equipment", isUnsaved);

  const handleSave = async () => {
    setSaving(true);
    setSaveStatus("idle");
    setSaveError(null);
    try {
      await onSave(equipmentDraftToPayload(draft));
      setSavedDraft(draft);
      setSaveStatus("success");
      setTimeout(() => setSaveStatus("idle"), 2000);
    } catch (e) {
      setSaveError(e instanceof Error ? e.message : String(e));
      setSaveStatus("error");
    } finally {
      setSaving(false);
    }
  };

  const archetypeRow = useMemo(
    () =>
      draft.archetype
        ? list.find((e) => e.equipment_id === draft.archetype)
        : undefined,
    [draft.archetype, list],
  );
  const baselineOf = (k: EquipmentNumericKey): number | null => {
    if (!archetypeRow) return null;
    const v = (archetypeRow as unknown as Record<string, number | null>)[k];
    return v === undefined ? null : v;
  };

  const archetypeOptions = useMemo(() => {
    const own = list.filter((e) => e.is_archetype).map((e) => e.equipment_id);
    return Array.from(new Set([...own, ...vanillaArchetypes])).sort();
  }, [list, vanillaArchetypes]);

  const isVariant = !!draft.archetype && !draft.is_archetype;

  const renderStat = (k: EquipmentNumericKey) => {
    const value = draft.stats[k];
    const baseline = isVariant ? baselineOf(k) : null;
    const isOverride = isVariant && value.trim() !== "";
    return (
      <div key={k} className="space-y-1">
        <Label className="flex items-center gap-1.5">
          <span>{t(STAT_LABEL[k])}</span>
          {isOverride && (
            <span className="text-micro px-1 rounded bg-warning/15 text-warning font-semibold">
              {t("overrideBadge")}
            </span>
          )}
        </Label>
        <Input
          type="number"
          step="any"
          className={`font-mono text-xs ${isOverride ? "border-warning/60" : ""}`}
          placeholder={baseline !== null ? `↳ ${baseline}` : ""}
          value={value}
          onChange={(e) =>
            setDraft({
              ...draft,
              stats: { ...draft.stats, [k]: e.target.value },
            })
          }
        />
        {isVariant && value.trim() === "" && baseline !== null && (
          <p className="text-micro text-muted-foreground">
            {t("inheritedFrom")}: {baseline}
          </p>
        )}
      </div>
    );
  };

  const renderTri = (
    field: "is_buildable" | "can_license" | "carrier_capable",
    label: string,
  ) => {
    const v = draft[field];
    return (
      <div key={field} className="space-y-1">
        <Label>{label}</Label>
        <div className="flex gap-1">
          {(["unset", "yes", "no"] as TriState[]).map((opt) => (
            <button
              key={opt}
              type="button"
              onClick={() => setDraft({ ...draft, [field]: opt })}
              className={`px-2 py-1 rounded text-micro border ui-interactive ${
                v === opt
                  ? "bg-primary/10 text-primary border-primary/40 font-semibold"
                  : "border-border text-muted-foreground hover:bg-muted"
              }`}
            >
              {opt === "unset"
                ? t("triUnset")
                : opt === "yes"
                  ? t("triYes")
                  : t("triNo")}
            </button>
          ))}
        </div>
      </div>
    );
  };

  return (
    <div className="flex-1 min-w-0 p-4 max-w-3xl space-y-4">
      <div>
        <p className="text-xs text-muted-foreground">
          {t("landing")}{" "}
          <span className="font-mono">
            {draft.source_file || `common/units/equipment/{TAG}_equipment.txt`}
          </span>
        </p>
        {isVariant ? (
          <p className="text-xs text-warning mt-1">
            {t("variantHint")}{" "}
            <span className="font-mono">{draft.archetype}</span>
          </p>
        ) : draft.is_archetype ? (
          <p className="text-xs text-primary mt-1">{t("archetypeHint")}</p>
        ) : null}
      </div>

      <section className="space-y-3">
        <h3 className="text-xs font-semibold uppercase tracking-wide text-muted-foreground">
          {t("sectionIdentity")}
        </h3>
        <div className="grid grid-cols-2 gap-3">
          <div className="space-y-1">
            <Label>{t("idLabel")}</Label>
            <Input
              className="font-mono text-xs"
              value={draft.equipment_id}
              onChange={(e) =>
                setDraft({ ...draft, equipment_id: e.target.value })
              }
            />
          </div>
          <div className="space-y-1">
            <Label>{t("typeLabel")}</Label>
            <Input
              list="equipment-type-suggestions"
              className="font-mono text-xs"
              placeholder={t("typePlaceholder")}
              value={draft.equipment_type}
              onChange={(e) =>
                setDraft({ ...draft, equipment_type: e.target.value })
              }
            />
            <datalist id="equipment-type-suggestions">
              {SUGGESTED_EQUIPMENT_TYPES.map((type) => (
                <option
                  key={type}
                  value={type}
                  label={equipmentTokenLabel(type, locale, unitNames.equipment)}
                />
              ))}
            </datalist>
          </div>
          <div className="space-y-1">
            <Label>{t("yearLabel")}</Label>
            <Input
              type="number"
              className="font-mono text-xs"
              value={draft.year}
              onChange={(e) => setDraft({ ...draft, year: e.target.value })}
            />
          </div>
          <div className="space-y-1">
            <Label>{t("pictureLabel")}</Label>
            <Input
              className="font-mono text-xs"
              placeholder="archetype_infantry_equipment"
              value={draft.picture}
              onChange={(e) => setDraft({ ...draft, picture: e.target.value })}
            />
          </div>
          <div className="space-y-1">
            <Label>{t("archetypeLabel")}</Label>
            <Input
              list="eq-archetypes"
              className="font-mono text-xs"
              placeholder={t("archetypePlaceholder")}
              value={draft.archetype}
              onChange={(e) =>
                setDraft({ ...draft, archetype: e.target.value })
              }
            />
            <datalist id="eq-archetypes">
              {archetypeOptions.map((a) => (
                <option
                  key={a}
                  value={a}
                  label={equipmentTokenLabel(a, locale, unitNames.equipment)}
                />
              ))}
            </datalist>
          </div>
          <div className="space-y-1">
            <Label>{t("parentLabel")}</Label>
            <Input
              className="font-mono text-xs"
              value={draft.parent}
              onChange={(e) => setDraft({ ...draft, parent: e.target.value })}
            />
          </div>
          <div className="space-y-1 col-span-2">
            <Label className="text-xs">{t("sourceFileLabel")}</Label>
            <Input
              className="font-mono text-xs"
              placeholder={t("sourceFilePlaceholder")}
              value={draft.source_file}
              onChange={(event) =>
                setDraft({ ...draft, source_file: event.target.value })
              }
            />
          </div>
        </div>
        <div className="flex items-center gap-4">
          <label className="flex items-center gap-1.5 text-xs cursor-pointer">
            <input
              type="checkbox"
              checked={draft.is_archetype}
              onChange={(e) =>
                setDraft({ ...draft, is_archetype: e.target.checked })
              }
            />
            {t("isArchetypeLabel")}
          </label>
          {renderTri("is_buildable", t("isBuildableLabel"))}
          {renderTri("can_license", t("canLicense"))}
        </div>
        <div className="space-y-1">
          <Label>{t("nameLabel")}</Label>
          <DisplayNameLocsTabs
            entityKey={`equipment:${equipment.id}:display-name`}
            value={draft.display_name_locs}
            exportLanguages={exportLanguages}
            legacyFallback={{ en: draft.name_en, zh: draft.name_zh }}
            onChange={(next) => setDraft({ ...draft, display_name_locs: next })}
            ariaLabel={t("nameLabel")}
          />
        </div>
      </section>

      <section className="space-y-2">
        <h3 className="text-xs font-semibold uppercase tracking-wide text-muted-foreground">
          {t("sectionOffensive")}
        </h3>
        <div className="grid grid-cols-3 gap-3">
          {OFFENSIVE_FIELDS.map((k) => renderStat(k))}
        </div>
      </section>

      <section className="space-y-2">
        <h3 className="text-xs font-semibold uppercase tracking-wide text-muted-foreground">
          {t("sectionDefensive")}
        </h3>
        <div className="grid grid-cols-3 gap-3">
          {DEFENSIVE_FIELDS.map((k) => renderStat(k))}
        </div>
      </section>

      <section className="space-y-2">
        <h3 className="text-xs font-semibold uppercase tracking-wide text-muted-foreground">
          {t("sectionEconomy")}
        </h3>
        <div className="grid grid-cols-3 gap-3">
          {ECONOMY_FIELDS.map((k) => renderStat(k))}
        </div>
      </section>

      <section className="space-y-2">
        <h3 className="text-xs font-semibold uppercase tracking-wide text-muted-foreground">
          {t("sectionNavy")}
        </h3>
        <div className="grid grid-cols-3 gap-3">
          {NAVY_FIELDS.map((field) => renderStat(field))}
        </div>
      </section>

      <section className="space-y-2">
        <div className="flex items-end justify-between gap-4">
          <h3 className="text-xs font-semibold uppercase tracking-wide text-muted-foreground">
            {t("sectionAir")}
          </h3>
          {renderTri("carrier_capable", t("carrierCapable"))}
        </div>
        <div className="grid grid-cols-3 gap-3">
          {AIR_FIELDS.map((field) => renderStat(field))}
        </div>
      </section>

      <section className="space-y-2">
        <div className="flex items-center justify-between">
          <h3 className="text-xs font-semibold uppercase tracking-wide text-muted-foreground">
            {t("sectionResources")}
          </h3>
          <Button
            size="sm"
            variant="outline"
            className="text-xs"
            onClick={() =>
              setDraft({
                ...draft,
                resourcePairs: [...draft.resourcePairs, { key: "", value: "" }],
              })
            }
          >
            {t("addResource")}
          </Button>
        </div>
        {draft.resourcePairs.length === 0 && (
          <p className="text-xs text-muted-foreground italic">
            {t("noResources")}
          </p>
        )}
        {draft.resourcePairs.map((p, i) => (
          <div key={i} className="grid grid-cols-[1fr_1fr_auto] gap-2">
            <Input
              placeholder={t("resourceName")}
              className="font-mono text-xs"
              value={p.key}
              onChange={(e) => {
                const next = [...draft.resourcePairs];
                next[i] = { ...next[i], key: e.target.value };
                setDraft({ ...draft, resourcePairs: next });
              }}
            />
            <Input
              placeholder={t("resourceAmount")}
              type="number"
              className="font-mono text-xs"
              value={p.value}
              onChange={(e) => {
                const next = [...draft.resourcePairs];
                next[i] = { ...next[i], value: e.target.value };
                setDraft({ ...draft, resourcePairs: next });
              }}
            />
            <Button
              size="sm"
              variant="ghost"
              className="px-2 text-muted-foreground hover:text-destructive"
              aria-label={t("removeResourceAria")}
              title={t("removeResourceAria")}
              onClick={() =>
                setDraft({
                  ...draft,
                  resourcePairs: draft.resourcePairs.filter((_, j) => j !== i),
                })
              }
            >
              ×
            </Button>
          </div>
        ))}
      </section>

      {(saving || saveStatus !== "idle" || isUnsaved) && (
        <div
          className={`-mx-4 px-4 py-1.5 text-xs font-medium flex items-center gap-2 transition-[opacity,background-color,border-color] duration-300 ${
            saving
              ? "bg-muted/80 text-muted-foreground border-y border-border"
              : saveStatus === "success"
                ? "bg-success/10 text-success border-y border-success/20"
                : saveStatus === "error"
                  ? "bg-destructive/10 text-destructive border-y border-destructive/20"
                  : isUnsaved
                    ? "bg-warning/10 text-warning border-y border-warning/20"
                    : ""
          }`}
        >
          <span>
            {saving
              ? t("saving")
              : saveStatus === "success"
                ? t("saved")
                : saveStatus === "error"
                  ? (saveError ?? t("saveFailed"))
                  : isUnsaved
                    ? t("unsavedChanges")
                    : ""}
          </span>
        </div>
      )}
      <div className="flex items-center gap-2 pt-2 border-t border-border">
        <Button onClick={() => void handleSave()} disabled={saving} size="sm">
          {t("save")}
        </Button>
      </div>
      {error && <p className="text-xs text-destructive">{error}</p>}
    </div>
  );
}
