"use client";

import { useMemo } from "react";
import { useTranslations } from "use-intl";
import useSWR from "swr";
import { Save, Trash2 } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { equipmentApi, vanillaDataApi } from "@/lib/api";
import { equipmentTokenLabel } from "@/lib/i18n/equipmentGloss";
import { useVanillaUnitNames } from "@/lib/vanillaUnitNames";
import { useLocaleStore } from "@/lib/store/localeStore";
import type { Equipment } from "@/types";
import type { OobSource, OobUnitsSnapshot } from "@/types/oobUnits";

import { localizeOobWarning, oobRowWarnings } from "./oobWarnings";
import {
  buildOobEquipmentSuggestions,
  isKnownEquipmentToken,
  type OobEntityDraft,
} from "./oobForm";

type EntityKind = OobEntityDraft["kind"];

export type OobSaveState = "idle" | "saving" | "saved" | "error";

const sourceForDraft = (
  snapshot: OobUnitsSnapshot,
  draft: OobEntityDraft,
): OobSource => {
  const collections: Record<
    EntityKind,
    Array<{ id: number; source: OobSource }>
  > = {
    "air-wing-group": snapshot.air_wing_groups,
    "air-wing": snapshot.air_wings,
    fleet: snapshot.fleets,
    "task-force": snapshot.task_forces,
    ship: snapshot.ships,
  };
  return (
    collections[draft.kind].find((row) => row.id === draft.rowId)?.source ??
    "created"
  );
};

const warningsForDraft = (
  snapshot: OobUnitsSnapshot,
  draft: OobEntityDraft,
): string[] => {
  if (draft.rowId === 0) return [];
  const collections: Record<EntityKind, Array<{ id: number }>> = {
    "air-wing-group": snapshot.air_wing_groups,
    "air-wing": snapshot.air_wings,
    fleet: snapshot.fleets,
    "task-force": snapshot.task_forces,
    ship: snapshot.ships,
  };
  return oobRowWarnings(
    collections[draft.kind].find((row) => row.id === draft.rowId),
  );
};

interface OobEntityEditorProps {
  projectId: number;
  draft: OobEntityDraft;

  snapshot: OobUnitsSnapshot;

  isUnsaved: boolean;
  saveState: OobSaveState;

  error: string | null;
  onFieldChange: (field: string, value: string) => void;
  onSave: () => void;

  onDelete?: () => void;
}

export function OobEntityEditor({
  projectId,
  draft,
  snapshot,
  isUnsaved,
  saveState,
  error,
  onFieldChange,
  onSave,
  onDelete,
}: OobEntityEditorProps) {
  const t = useTranslations("oobUnitsEditor");
  const locale = useLocaleStore((state) => state.locale);
  const unitNames = useVanillaUnitNames();

  const { data: equipmentData, isLoading: projectEquipmentLoading } = useSWR<
    Equipment[]
  >(projectId ? ["equipment/for-oob-units", projectId] : null, () =>
    equipmentApi.list(projectId),
  );
  const { data: vanillaEquipmentData, isLoading: vanillaEquipmentLoading } =
    useSWR("vanilla/equipment-archetypes/for-oob-units", () =>
      vanillaDataApi.equipmentArchetypes(),
    );
  const projectEquipment = useMemo(
    () => (Array.isArray(equipmentData) ? equipmentData : []),
    [equipmentData],
  );
  const vanillaEquipment = useMemo(
    () =>
      Array.isArray(vanillaEquipmentData?.archetypes)
        ? vanillaEquipmentData.archetypes
        : [],
    [vanillaEquipmentData],
  );
  const airEquipmentSuggestions = useMemo(
    () =>
      buildOobEquipmentSuggestions("air", projectEquipment, vanillaEquipment),
    [projectEquipment, vanillaEquipment],
  );
  const navalEquipmentSuggestions = useMemo(
    () =>
      buildOobEquipmentSuggestions("naval", projectEquipment, vanillaEquipment),
    [projectEquipment, vanillaEquipment],
  );
  const equipmentCatalogLoading =
    projectEquipmentLoading || vanillaEquipmentLoading;

  const displayedSaveState =
    isUnsaved && saveState === "saved" ? "idle" : saveState;

  const draftTitle =
    draft.kind === "air-wing-group"
      ? t("airGroup")
      : draft.kind === "air-wing"
        ? t("airWing")
        : draft.kind === "fleet"
          ? t("fleet")
          : draft.kind === "task-force"
            ? t("taskForce")
            : t("ship");

  const draftWarnings = warningsForDraft(snapshot, draft);

  const field = (
    label: string,
    key: string,
    value: string,
    options: { mono?: boolean; maxLength?: number } = {},
  ) => (
    <div className="space-y-1">
      <Label htmlFor={`oob-${draft.kind}-${key}`} className="text-xs">
        {label}
      </Label>
      <Input
        id={`oob-${draft.kind}-${key}`}
        className={options.mono ? "font-mono text-xs" : "text-xs"}
        maxLength={options.maxLength}
        value={value}
        onChange={(event) => onFieldChange(key, event.target.value)}
      />
    </div>
  );

  const equipmentField = (domain: "air" | "naval", value: string) => {
    const suggestions =
      domain === "air" ? airEquipmentSuggestions : navalEquipmentSuggestions;
    const listId = `oob-${domain}-equipment-options`;
    const isUnknown =
      !equipmentCatalogLoading && !isKnownEquipmentToken(value, suggestions);
    return (
      <div className="space-y-1">
        <Label htmlFor={`oob-${draft.kind}-equipment_type`} className="text-xs">
          {t("equipmentType")}
        </Label>
        <Input
          id={`oob-${draft.kind}-equipment_type`}
          list={listId}
          className="font-mono text-xs"
          value={value}
          onChange={(event) =>
            onFieldChange("equipment_type", event.target.value)
          }
        />
        <datalist id={listId}>
          {suggestions.map((token) => (
            <option key={token} value={token}>
              {equipmentTokenLabel(token, locale, unitNames.equipment)}
            </option>
          ))}
        </datalist>
        <p className="text-xs text-muted-foreground">
          {t("equipmentCatalogHint")}
        </p>
        {isUnknown && (
          <p className="text-xs text-warning">
            {t("unknownEquipmentPreserved", { token: value })}
          </p>
        )}
      </div>
    );
  };

  const parentSelect = (
    label: string,
    key: string,
    value: string,
    rows: Array<{ id: number; label: string }>,
  ) => (
    <div className="space-y-1">
      <Label htmlFor={`oob-${draft.kind}-${key}`} className="text-xs">
        {label}
      </Label>
      <select
        id={`oob-${draft.kind}-${key}`}
        className="h-9 w-full rounded-md border border-input bg-card px-3 font-mono text-xs ui-focus-ring"
        value={value}
        onChange={(event) => onFieldChange(key, event.target.value)}
      >
        {rows.map((row) => (
          <option key={row.id} value={row.id}>
            {row.label}
          </option>
        ))}
      </select>
    </div>
  );

  const renderFormFields = () => {
    switch (draft.kind) {
      case "air-wing-group":
        return (
          <>
            {field(t("oobFile"), "oob_file", draft.value.oob_file, {
              mono: true,
            })}
            {field(t("sourceFile"), "source_file", draft.value.source_file, {
              mono: true,
            })}
            {field(t("countryTag"), "country_tag", draft.value.country_tag, {
              mono: true,
              maxLength: 10,
            })}
            {field(t("location"), "location", draft.value.location, {
              mono: true,
            })}
            {field(t("ordering"), "ordering", draft.value.ordering, {
              mono: true,
            })}
          </>
        );
      case "air-wing":
        return (
          <>
            {parentSelect(
              t("parentAirGroup"),
              "group_id",
              draft.value.group_id,
              snapshot.air_wing_groups.map((row) => ({
                id: row.id,
                label: row.location || `#${row.id}`,
              })),
            )}
            {equipmentField("air", draft.value.equipment_type)}
            {field(t("name"), "name", draft.value.name)}
            {field(t("owner"), "owner", draft.value.owner, { mono: true })}
            {field(t("amount"), "amount", draft.value.amount, { mono: true })}
            {field(t("ordering"), "ordering", draft.value.ordering, {
              mono: true,
            })}
          </>
        );
      case "fleet":
        return (
          <>
            {field(t("oobFile"), "oob_file", draft.value.oob_file, {
              mono: true,
            })}
            {field(t("sourceFile"), "source_file", draft.value.source_file, {
              mono: true,
            })}
            {field(t("countryTag"), "country_tag", draft.value.country_tag, {
              mono: true,
              maxLength: 10,
            })}
            {field(t("name"), "name", draft.value.name)}
            {field(t("navalBase"), "naval_base", draft.value.naval_base, {
              mono: true,
            })}
            {field(t("ordering"), "ordering", draft.value.ordering, {
              mono: true,
            })}
          </>
        );
      case "task-force":
        return (
          <>
            {parentSelect(
              t("parentFleet"),
              "fleet_id",
              draft.value.fleet_id,
              snapshot.fleets.map((row) => ({
                id: row.id,
                label: row.name || `#${row.id}`,
              })),
            )}
            {field(t("name"), "name", draft.value.name)}
            {field(t("location"), "location", draft.value.location, {
              mono: true,
            })}
            {field(t("ordering"), "ordering", draft.value.ordering, {
              mono: true,
            })}
          </>
        );
      case "ship":
        return (
          <>
            {parentSelect(
              t("parentTaskForce"),
              "task_force_id",
              draft.value.task_force_id,
              snapshot.task_forces.map((row) => ({
                id: row.id,
                label: row.name || `#${row.id}`,
              })),
            )}
            {field(t("name"), "name", draft.value.name)}
            {field(t("definition"), "definition", draft.value.definition, {
              mono: true,
            })}
            {field(
              t("startExperienceFactor"),
              "start_experience_factor",
              draft.value.start_experience_factor,
              { mono: true },
            )}
            {equipmentField("naval", draft.value.equipment_type)}
            {field(t("amount"), "amount", draft.value.amount, { mono: true })}
            {field(t("owner"), "owner", draft.value.owner, { mono: true })}
            {field(t("versionName"), "version_name", draft.value.version_name)}
            {field(t("ordering"), "ordering", draft.value.ordering, {
              mono: true,
            })}
          </>
        );
    }
  };

  return (
    <div className="mx-auto max-w-3xl p-6">
      <div className="mb-6 flex items-center gap-3 border-b border-border pb-4">
        <div className="min-w-0 flex-1">
          <h2 className="text-lg font-semibold tracking-tight">{draftTitle}</h2>
          <p className="mt-1 font-mono text-xs text-muted-foreground">
            {draft.rowId === 0 ? t("newDraft") : `#${draft.rowId}`}
          </p>
        </div>
        <span className="rounded-full bg-muted px-2 py-1 text-xs text-muted-foreground">
          {sourceForDraft(snapshot, draft) === "imported"
            ? t("sourceImported")
            : t("sourceCreated")}
        </span>
        {draft.rowId !== 0 && onDelete && (
          <Button
            type="button"
            size="icon"
            variant="ghost"
            className="h-9 w-9 text-muted-foreground hover:text-destructive ui-interactive"
            title={t("delete")}
            aria-label={t("delete")}
            onClick={onDelete}
          >
            <Trash2 className="h-4 w-4" aria-hidden />
          </Button>
        )}
      </div>

      <div className="grid grid-cols-2 gap-4">{renderFormFields()}</div>

      {draftWarnings.length > 0 && (
        <ul
          role="alert"
          className="mt-4 list-disc space-y-1 rounded-md border border-warning/20 bg-warning/10 py-2 pl-10 pr-4 text-xs text-warning"
        >
          {draftWarnings.map((warning, index) => (
            <li key={`${index}-${warning}`}>
              {localizeOobWarning(warning, t)}
            </li>
          ))}
        </ul>
      )}

      {(saveState !== "idle" || isUnsaved) && (
        <div
          className={`-mx-6 mt-6 flex items-center border-y px-6 py-2 text-xs font-medium transition-[background-color,border-color,color] duration-300 ${
            displayedSaveState === "saving"
              ? "border-border bg-muted/80 text-muted-foreground"
              : displayedSaveState === "saved"
                ? "border-success/20 bg-success/10 text-success"
                : displayedSaveState === "error"
                  ? "border-destructive/20 bg-destructive/10 text-destructive"
                  : "border-warning/20 bg-warning/10 text-warning"
          }`}
        >
          {displayedSaveState === "saving"
            ? t("saving")
            : displayedSaveState === "saved"
              ? t("saved")
              : displayedSaveState === "error" && error
                ? t("saveError", { error })
                : t("unsaved")}
        </div>
      )}

      <div className="mt-4 flex justify-end border-t border-border pt-4">
        <Button
          type="button"
          size="sm"
          className="gap-2 ui-interactive"
          disabled={saveState === "saving" || !isUnsaved}
          onClick={onSave}
        >
          <Save className="h-4 w-4" aria-hidden />
          {t("save")}
        </Button>
      </div>
    </div>
  );
}
