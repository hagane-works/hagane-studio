"use client";

import { useState, useEffect, useCallback, useMemo, useRef } from "react";
import { useTranslations } from "use-intl";
import useSWR from "swr";
import { Input } from "@/components/ui/input";
import { Select } from "@/components/ui/select";
import { Label } from "@/components/ui/label";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Separator } from "@/components/ui/separator";
import { Textarea } from "@/components/ui/textarea";
import { DisplayNameLocsTabs } from "@/components/shared/editors/DisplayNameLocsTabs";
import { DescriptionLocsTabs } from "@/components/shared/editors/DescriptionLocsTabs";
import { DEFAULT_EXPORT_LANGUAGES } from "@/components/shared/editors/displayNameLocsHelpers";
import {
  nextDraftOnEntityChange,
  stableFingerprint,
} from "@/components/shared/editors/draftReseed";
import { useApplyReseedNotice } from "@/components/shared/editors/useApplyReseedNotice";
import { useProjectStore } from "@/lib/store/projectStore";
import { useRegisterDirty } from "@/lib/hooks/useUnsavedGuard";
import {
  InfoTooltip,
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import {
  SectionHeader,
  extractSaveErrorMessage,
  validateRequiredId,
} from "@/components/shared/editors";
import type { Technology } from "@/types";
import { equipmentApi, workbenchIconsApi, technologiesApi } from "@/lib/api";
import { useBaseModDepKeys, gfxUrl } from "@/lib/api/basemod";
import {
  ImageUpload,
  type ImageUploadResult,
} from "@/shared/upload/ImageUpload";
import { TECH_ICON_SIZE } from "@/shared/upload/sizes";
import {
  useVanillaCollisions,
  suggestNonCollidingTechId,
} from "@/lib/vanillaCollisions";
import { SourceEditorDialog } from "@/shared/dialogs/SourceEditorDialog";
import { useConfirm } from "@/components/shared/ConfirmDialogProvider";
import { useImportedSourceGate } from "@/shared/dialogs/useImportedSourceGate";
import {
  parseTokenLines,
  buildTechSavePayload,
  parseDependencyLines,
  formatDependencyLines,
} from "./techFormState";
import { RawExtrasNotice } from "./RawExtrasNotice";
import {
  getTechnologyFolderPlacements,
  getSelectedTechnologyPlacement,
  resolveTechnologyPlacementIndex,
  updateTechnologyFolderPlacement,
} from "@/lib/techGraph";

export const TECH_LINE_OPTIONS: {
  value: string;
  label: string;
  folder: string;
  i18nKey: string;
}[] = [
  {
    value: "infantry",
    label: "Infantry / 步兵",
    folder: "infantry_folder",
    i18nKey: "infantry",
  },
  {
    value: "support",
    label: "Support / 支援",
    folder: "support_folder",
    i18nKey: "support",
  },
  {
    value: "artillery",
    label: "Artillery / 火炮",
    folder: "artillery_folder",
    i18nKey: "artillery",
  },
  {
    value: "armor",
    label: "Armor / 装甲",
    folder: "armour_folder",
    i18nKey: "armor",
  },
  {
    value: "nsb_armor",
    label: "NSB Armor / NSB 装甲(NSB DLC)",
    folder: "nsb_armour_folder",
    i18nKey: "nsbArmor",
  },
  {
    value: "air_techs",
    label: "Air / 空军",
    folder: "air_techs_folder",
    i18nKey: "air",
  },
  {
    value: "bba_air_techs",
    label: "BBA Air / BBA 空军(BBA DLC)",
    folder: "bba_air_techs_folder",
    i18nKey: "bbaAir",
  },
  {
    value: "naval",
    label: "Naval / 海军",
    folder: "naval_folder",
    i18nKey: "naval",
  },
  {
    value: "mtg_naval",
    label: "MTG Naval / MTG 海军(MTG DLC)",
    folder: "mtgnavalfolder",
    i18nKey: "mtgNaval",
  },
  {
    value: "naval_support",
    label: "Naval Support / 海军支援(MTG DLC · ASW/鱼雷/雷区)",
    folder: "mtgnavalsupportfolder",
    i18nKey: "navalSupport",
  },
  {
    value: "industry",
    label: "Industry / 工业",
    folder: "industry_folder",
    i18nKey: "industry",
  },
  {
    value: "electronic_mechanical_engineering",
    label: "Electronics / 电子",
    folder: "electronics_folder",
    i18nKey: "electronics",
  },
  {
    value: "special_forces_doctrine",
    label: "Special Forces / 特种部队学说",
    folder: "special_forces_doctrine_folder",
    i18nKey: "specialForces",
  },
];

const KNOWN_TECH_FOLDERS: string[] = Array.from(
  new Set(TECH_LINE_OPTIONS.map((o) => o.folder)),
);

const COMMON_CATEGORIES = [
  "infantry_weapons",
  "infantry_tech",
  "cat_light_infantry",
  "motorized_equipment",
  "cat_mechanized_equipment",
  "armor",
  "cat_light_armor",
  "cat_medium_armor",
  "cat_heavy_armor",
  "air_equipment",
  "cat_fighter",
  "cat_cas",
  "cat_bomber",
  "naval_equipment",
  "cat_submarines",
  "cat_destroyers",
  "cat_capital_ships",
  "electronics",
  "radar_tech",
  "computing_tech",
  "encryption_tech",
  "industry",
  "cat_construction_tech",
  "cat_production",
  "support_tech",
  "cat_hospital",
  "cat_logistics_tech",
];

const MODIFIER_CATEGORY_OPTIONS: {
  value: string;
  label: string;
  i18nKey: string;
}[] = [
  {
    value: "category_light_infantry",
    label: "轻步兵 (Light Infantry)",
    i18nKey: "lightInfantry",
  },
  { value: "category_infantry", label: "步兵 (Infantry)", i18nKey: "infantry" },
  { value: "category_cavalry", label: "骑兵 (Cavalry)", i18nKey: "cavalry" },
  {
    value: "category_vehicle_infantry",
    label: "摩托化/机械化步兵 (Motorized/Mechanized Infantry)",
    i18nKey: "motorizedMechanizedInfantry",
  },
  {
    value: "category_special_forces",
    label: "特种部队 (Special Forces)",
    i18nKey: "specialForcesUnit",
  },
  {
    value: "category_all_infantry",
    label: "所有步兵类 (All Infantry)",
    i18nKey: "allInfantry",
  },
  {
    value: "category_front_line",
    label: "所有前线部队 (All Front-line)",
    i18nKey: "allFrontLine",
  },
  { value: "category_army", label: "所有陆军 (All Army)", i18nKey: "allArmy" },
  {
    value: "category_light_armor",
    label: "轻型装甲 (Light Armor)",
    i18nKey: "lightArmor",
  },
  {
    value: "category_medium_armor",
    label: "中型装甲 (Medium Armor)",
    i18nKey: "mediumArmor",
  },
  {
    value: "category_heavy_armor",
    label: "重型装甲 (Heavy Armor)",
    i18nKey: "heavyArmor",
  },
  {
    value: "category_all_armor",
    label: "所有装甲 (All Armor)",
    i18nKey: "allArmor",
  },
  { value: "category_fighter", label: "战斗机 (Fighter)", i18nKey: "fighter" },
  { value: "category_cas", label: "近距支援 (CAS)", i18nKey: "cas" },
  { value: "category_bomber", label: "轰炸机 (Bomber)", i18nKey: "bomber" },
  {
    value: "category_submarines",
    label: "潜艇 (Submarines)",
    i18nKey: "submarines",
  },
  {
    value: "category_destroyers",
    label: "驱逐舰 (Destroyers)",
    i18nKey: "destroyers",
  },
  {
    value: "category_capital_ships",
    label: "主力舰 (Capital Ships)",
    i18nKey: "capitalShips",
  },
];

const MODIFIER_STAT_OPTIONS: {
  value: string;
  label: string;
  i18nKey: string;
  isPercent?: boolean;
}[] = [
  { value: "soft_attack", label: "软攻 (Soft Attack)", i18nKey: "softAttack" },
  { value: "hard_attack", label: "硬攻 (Hard Attack)", i18nKey: "hardAttack" },
  { value: "defense", label: "防御 (Defense)", i18nKey: "defense" },
  {
    value: "breakthrough",
    label: "突破 (Breakthrough)",
    i18nKey: "breakthrough",
  },
  { value: "armor_value", label: "装甲值 (Armor)", i18nKey: "armorValue" },
  { value: "ap_attack", label: "穿甲 (Piercing)", i18nKey: "piercing" },
  {
    value: "max_organisation",
    label: "组织度 (Organization)",
    i18nKey: "organization",
  },
  { value: "max_strength", label: "血量 (HP)", i18nKey: "hp" },
  {
    value: "reliability",
    label: "可靠性 (Reliability)",
    i18nKey: "reliability",
    isPercent: true,
  },
  { value: "maximum_speed", label: "速度 (Speed)", i18nKey: "speed" },
  { value: "build_cost_ic", label: "生产花费 (IC Cost)", i18nKey: "icCost" },
  {
    value: "suppression",
    label: "镇压值 (Suppression)",
    i18nKey: "suppression",
  },
  { value: "air_attack", label: "对空攻击 (Air Attack)", i18nKey: "airAttack" },
  { value: "naval_speed", label: "航速 (Naval Speed)", i18nKey: "navalSpeed" },
  {
    value: "surface_detection",
    label: "水面侦测 (Surface Detection)",
    i18nKey: "surfaceDetection",
  },
  {
    value: "sub_detection",
    label: "反潜侦测 (Sub Detection)",
    i18nKey: "subDetection",
  },
];

interface TechEditorProps {
  tech: Technology;
  allTechs: Technology[];
  onSave: (data: Partial<Technology>) => Promise<void>;
  projectId: number;
  onDraftChange?: (draft: Technology) => void;

  onIconUploaded?: (result: ImageUploadResult) => void;

  onDelete?: (techDbId: number) => Promise<void>;

  selectedPlacementIndex?: number;
  onSelectedPlacementIndexChange?: (index: number) => void;

  layoutMode?: boolean;
}

export function TechEditor({
  tech,
  allTechs,
  onSave,
  projectId,
  onDraftChange,
  onIconUploaded,
  onDelete,
  selectedPlacementIndex = 0,
  onSelectedPlacementIndexChange,
  layoutMode = false,
}: TechEditorProps) {
  const t = useTranslations("techEditor");
  const confirmDialog = useConfirm();

  const tTech = useTranslations("tech");

  const exportLanguages = useProjectStore(
    (s) => s.currentProject?.export_languages ?? DEFAULT_EXPORT_LANGUAGES,
  );
  const [iconCacheBust, setIconCacheBust] = useState(0);
  const [form, setForm] = useState(tech);
  const [projectEquipmentIds, setProjectEquipmentIds] = useState<string[]>([]);

  const baseModDepKeys = useBaseModDepKeys();
  const baseModIconFallbacks = useMemo(
    () =>
      form.tech_id && baseModDepKeys.length
        ? baseModDepKeys.map((dk) =>
            gfxUrl(projectId, dk, `GFX_${form.tech_id}_medium`),
          )
        : [],
    [projectId, baseModDepKeys, form.tech_id],
  );

  const iconSizeTechId = tech.tech_id || form.tech_id;
  const { data: iconSizeInfo } = useSWR(
    iconSizeTechId ? ["tech-icon-size", projectId, iconSizeTechId] : null,
    () => workbenchIconsApi.techIconSize(projectId, iconSizeTechId),
    { revalidateOnFocus: false },
  );
  const iconSize = iconSizeInfo
    ? { width: iconSizeInfo.width, height: iconSizeInfo.height }
    : TECH_ICON_SIZE;
  const [saving, setSaving] = useState(false);
  const [saveStatus, setSaveStatus] = useState<"idle" | "success" | "error">(
    "idle",
  );

  const [saveError, setSaveError] = useState<string | null>(null);
  const [idError, setIdError] = useState<string | null>(null);

  const techFingerprint = useCallback(
    (t2: Technology) => stableFingerprint(t2),
    [],
  );
  const notifyReseed = useApplyReseedNotice();
  const seededTechRef = useRef({
    id: tech.id,
    fingerprint: techFingerprint(tech),
  });
  useEffect(() => {
    const decision = nextDraftOnEntityChange<Technology, number>({
      prevSeededId: seededTechRef.current.id,
      prevFingerprint: seededTechRef.current.fingerprint,
      entityId: tech.id,
      entity: tech,
      currentDraft: form,
      fingerprintOf: techFingerprint,
    });
    seededTechRef.current = {
      id: decision.seededId,
      fingerprint: decision.fingerprint,
    };
    if (decision.reseeded) setForm(decision.draft);
    if (decision.reason === "switch") setSaveStatus("idle");
    notifyReseed(decision);
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [tech, techFingerprint, notifyReseed]);

  useEffect(() => {
    onDraftChange?.(form);
  }, [form, onDraftChange]);

  const update = useCallback((field: string, value: unknown) => {
    setForm((prev) => ({ ...prev, [field]: value }));
  }, []);

  const placements = useMemo(() => getTechnologyFolderPlacements(form), [form]);
  const resolvedPlacementIndex = useMemo(
    () => resolveTechnologyPlacementIndex(form, selectedPlacementIndex),
    [form, selectedPlacementIndex],
  );
  const selectedPlacement = useMemo(
    () => getSelectedTechnologyPlacement(form, resolvedPlacementIndex),
    [form, resolvedPlacementIndex],
  );
  const updateSelectedPlacement = useCallback(
    (patch: Parameters<typeof updateTechnologyFolderPlacement>[2]) => {
      setForm((prev) =>
        updateTechnologyFolderPlacement(prev, selectedPlacementIndex, patch),
      );
    },
    [selectedPlacementIndex],
  );

  useEffect(() => {
    let alive = true;
    equipmentApi
      .list(projectId)
      .then((rows) => {
        if (alive) setProjectEquipmentIds(rows.map((row) => row.equipment_id));
      })
      .catch(() => {
        if (alive) setProjectEquipmentIds([]);
      });
    return () => {
      alive = false;
    };
  }, [projectId]);

  const equipmentOptions = useMemo(
    () =>
      Array.from(
        new Set([...(form.enable_equipments || []), ...projectEquipmentIds]),
      ).sort(),
    [form.enable_equipments, projectEquipmentIds],
  );

  const vanillaCollisionsData = useVanillaCollisions();
  const vanillaCollision = useMemo(() => {
    const isColliding =
      vanillaCollisionsData.loaded &&
      !!form.tech_id &&
      vanillaCollisionsData.techIds.has(form.tech_id);
    const suggested = suggestNonCollidingTechId(
      form.tech_id || "",
      form.country_tag || "",
    );
    return { isColliding, suggested };
  }, [vanillaCollisionsData, form.tech_id, form.country_tag]);

  const sourceGate = useImportedSourceGate<Partial<Technology>>({
    entitySource: tech.source,
    performSave: async (payload) => {
      setSaving(true);
      setSaveStatus("idle");
      setSaveError(null);
      try {
        await onSave(payload);
        setSaveStatus("success");
        setTimeout(() => setSaveStatus("idle"), 2000);
      } catch (err) {
        console.error("Save failed:", err);
        setSaveError(extractSaveErrorMessage(err));
        setSaveStatus("error");
      } finally {
        setSaving(false);
      }
    },
    resetForm: () => setForm(tech),
  });

  const handleSave = async () => {
    const trimmedId = (form.tech_id || "").trim();

    const idValidation = validateRequiredId(trimmedId, "Tech ID");
    if (idValidation) {
      setIdError(idValidation);
      setSaveError(idValidation);
      setSaveStatus("error");
      return;
    }
    setIdError(null);
    const payload: Partial<Technology> = buildTechSavePayload(form);
    if (sourceGate.interceptSave(payload)) return;
    setSaving(true);
    setSaveStatus("idle");
    setSaveError(null);
    try {
      await onSave(payload);
      setSaveStatus("success");
      setTimeout(() => setSaveStatus("idle"), 2000);
    } catch (err) {
      console.error("Save failed:", err);
      setSaveError(extractSaveErrorMessage(err));
      setSaveStatus("error");
    } finally {
      setSaving(false);
    }
  };

  const toggleCategory = (cat: string) => {
    const current = form.categories || [];
    const next = current.includes(cat)
      ? current.filter((c) => c !== cat)
      : [...current, cat];
    update("categories", next);
  };

  const addPrereq = (techId: string) => {
    if (!techId) return;
    const current = form.prerequisites || [];
    if (!current.includes(techId)) {
      update("prerequisites", [...current, techId]);
    }
  };

  const removePrereq = (techId: string) => {
    const current = form.prerequisites || [];
    update(
      "prerequisites",
      current.filter((p) => p !== techId),
    );
  };

  const [modCategory, setModCategory] = useState("");
  const [modStat, setModStat] = useState("");
  const [modValue, setModValue] = useState("0.05");

  const addModifier = () => {
    if (!modCategory || !modStat) return;

    const num = parseFloat(modValue);
    if (!Number.isFinite(num)) return;
    const current = { ...(form.modifiers || {}) };
    if (!current[modCategory]) current[modCategory] = {};
    (current[modCategory] as Record<string, number>)[modStat] = num;
    update("modifiers", current);
    setModStat("");
  };

  const removeModifier = (cat: string, stat: string) => {
    const current = { ...(form.modifiers || {}) };
    if (current[cat]) {
      const catMods = { ...(current[cat] as Record<string, number>) };
      delete catMods[stat];
      if (Object.keys(catMods).length === 0) {
        delete current[cat];
      } else {
        current[cat] = catMods;
      }
      update("modifiers", current);
    }
  };

  const otherTechs = allTechs.filter((t) => t.id !== tech.id);

  const isUnsaved = useMemo(
    () => JSON.stringify(form) !== JSON.stringify(tech),
    [form, tech],
  );

  useRegisterDirty("technology", isUnsaved);

  return (
    <TooltipProvider>
      <div className="p-6 space-y-6 max-w-2xl mx-auto">
        {(saving || saveStatus !== "idle" || isUnsaved) && (
          <div
            className={`
            sticky top-0 z-10 -mx-6 -mt-6 px-6 py-1.5 text-xs font-medium flex items-center gap-2 shrink-0
            transition-[opacity,background-color,border-color] duration-300
            ${
              saving
                ? "bg-muted/80 text-muted-foreground border-b border-border"
                : saveStatus === "success"
                  ? "bg-success/10 text-success border-b border-success/20"
                  : saveStatus === "error"
                    ? "bg-destructive/10 text-destructive border-b border-destructive/20"
                    : isUnsaved
                      ? "bg-warning/10 text-warning border-b border-warning/20"
                      : ""
            }
          `}
          >
            <span>
              {saving
                ? t("statusSaving")
                : saveStatus === "success"
                  ? t("statusSaved")
                  : saveStatus === "error"
                    ? (saveError ?? t("statusError"))
                    : isUnsaved
                      ? t("statusUnsaved")
                      : ""}
            </span>
          </div>
        )}

        <RawExtrasNotice rawExtras={form.raw_extras} />

        <section className="space-y-3">
          <SectionHeader title={t("basicInfo")} />

          <div className="space-y-1">
            <Label>{t("displayName")}</Label>
            <DisplayNameLocsTabs
              entityKey={`technology:${tech.id}:display-name`}
              value={form.display_name_locs || {}}
              exportLanguages={exportLanguages}
              legacyFallback={{ en: form.name_en, zh: form.name_zh }}
              onChange={(next) => update("display_name_locs", next)}
              ariaLabel="Technology display name"
            />
          </div>
          <div className="space-y-1">
            <div className="flex items-center gap-1">
              <Label>{t("techIdLabel")}</Label>
              <InfoTooltip
                title={t("techId.tooltipTitle")}
                content={t.raw("techId.tooltipContent")}
              />
            </div>

            <Input
              value={form.tech_id}
              onChange={(e) => {
                if (idError) setIdError(null);
                update("tech_id", e.target.value);
              }}
              aria-invalid={idError ? true : undefined}
              className={`font-mono text-xs ${
                idError || vanillaCollision.isColliding
                  ? "border-destructive focus-visible:ring-destructive/40"
                  : ""
              }`}
              placeholder="e.g. infantry_weapons1"
            />
            {idError && <p className="text-xs text-destructive">{idError}</p>}

            {vanillaCollision.isColliding && (
              <div className="mt-1 p-2 rounded border border-destructive/40 bg-destructive/5 text-xs space-y-1">
                <p className="text-destructive font-medium">
                  ⚠ <code className="font-mono">{form.tech_id}</code>{" "}
                  {t("collision.sameName")}
                </p>
                <button
                  type="button"
                  onClick={() => update("tech_id", vanillaCollision.suggested)}
                  className="text-xs underline decoration-dotted text-destructive/80 hover:text-destructive"
                >
                  {t("collision.renameTo")}{" "}
                  <code className="font-mono">
                    {vanillaCollision.suggested}
                  </code>{" "}
                  {t("collision.recommended")}
                </button>
              </div>
            )}
          </div>

          {form.tech_id ? (
            <ImageUpload
              label="Tech Icon"
              targetSize={iconSize}
              previewUrl={`${workbenchIconsApi.techIconUrl(projectId, form.tech_id)}?t=${iconCacheBust}`}
              baseModFallbackUrls={baseModIconFallbacks}
              gfxName={`GFX_${form.tech_id}_medium`}
              hint={`PNG / JPG / DDS · ${iconSize.width} × ${iconSize.height}${iconSizeInfo?.base_game ? " (the base game's size for this tech)" : ""} · auto-converted to DDS at export · GFX_<tech_id>_medium`}
              onUpload={async (file, crop) =>
                technologiesApi.uploadIcon(projectId, tech.id, file, crop)
              }
              onUploaded={(result) => {
                setIconCacheBust(Date.now());
                onIconUploaded?.(result);
              }}
            />
          ) : (
            <p className="text-xs text-muted-foreground p-3 bg-muted/30 rounded border border-dashed border-border">
              Set Tech ID first — the upload uses it as the on-disk filename.
            </p>
          )}

          <div className="space-y-2">
            <div className="flex items-center gap-2">
              <input
                type="checkbox"
                id="tech-is-banner"
                checked={form.is_banner ?? false}
                onChange={(e) => update("is_banner", e.target.checked)}
                className="rounded border-border"
              />
              <Label htmlFor="tech-is-banner" className="cursor-pointer">
                {t("banner.showAs")}
              </Label>
              <InfoTooltip
                title={t("banner.tooltipTitle")}
                content={t.raw("banner.tooltipContent")}
              />
            </div>

            <div className="flex items-center gap-2">
              <input
                type="checkbox"
                id="tech-force-small-layout"
                checked={form.force_use_small_tech_layout ?? false}
                onChange={(e) =>
                  update("force_use_small_tech_layout", e.target.checked)
                }
                className="rounded border-border"
              />
              <Label
                htmlFor="tech-force-small-layout"
                className="cursor-pointer"
              >
                {t("forceSmallLayout.showAs")}
              </Label>
              <InfoTooltip
                title={t("forceSmallLayout.tooltipTitle")}
                content={t.raw("forceSmallLayout.tooltipContent")}
              />
            </div>
          </div>

          <div className="grid grid-cols-3 gap-3">
            <div className="space-y-1">
              <div className="flex items-center gap-1">
                <Label>{t("researchCost.label")}</Label>
                <InfoTooltip
                  title={t("researchCost.tooltipTitle")}
                  content={t.raw("researchCost.tooltipContent")}
                />
              </div>
              <Input
                type="number"
                min={0.1}
                step={0.1}
                value={form.research_cost}
                onChange={(e) => {
                  const v = parseFloat(e.target.value);
                  update(
                    "research_cost",
                    Number.isFinite(v) ? v : form.research_cost,
                  );
                }}
              />
            </div>
            <div className="space-y-1">
              <div className="flex items-center gap-1">
                <Label>{t("startYear.label")}</Label>
                <InfoTooltip
                  title={t("startYear.tooltipTitle")}
                  content={t.raw("startYear.tooltipContent")}
                />
              </div>
              <Input
                type="number"
                value={form.start_year}
                onChange={(e) => update("start_year", Number(e.target.value))}
              />
              <div className="text-micro leading-tight text-warning">
                {t("startYear.balanceWarn")}
              </div>
            </div>
            <div className="space-y-1">
              <div className="flex items-center gap-1">
                <Label>{t("xPosition.label")}</Label>
                <InfoTooltip
                  title={t("xPosition.tooltipTitle")}
                  content={t.raw("xPosition.tooltipContent")}
                />
              </div>
              <Input
                type="number"
                value={selectedPlacement?.x ?? form.position_x}
                disabled={!layoutMode}
                onChange={(e) => {
                  const x = Number(e.target.value);
                  if (selectedPlacement) updateSelectedPlacement({ x });
                  else update("position_x", x);
                }}
              />
            </div>
          </div>

          {placements.length > 1 && (
            <div className="space-y-1">
              <Label>{t("folder.label")}</Label>
              <Select
                value={String(resolvedPlacementIndex)}
                aria-label={t("folder.label")}
                onChange={(e) =>
                  onSelectedPlacementIndexChange?.(Number(e.target.value))
                }
              >
                {placements.map((placement, index) => (
                  <option
                    key={`${placement.folder}-${placement.x}-${placement.y}-${index}`}
                    value={String(index)}
                  >
                    {`${placement.folder} (${placement.x}, ${placement.y})`}
                  </option>
                ))}
              </Select>
            </div>
          )}
          <div className="space-y-1">
            <div className="flex items-center gap-1">
              <Label>{t("folder.label")}</Label>
              <InfoTooltip
                title={t("folder.tooltipTitle")}
                content={t.raw("folder.tooltipContent")}
              />
            </div>
            <Select
              value={(selectedPlacement?.folder ?? form.folder) || ""}
              disabled={!layoutMode}
              onChange={(e) => {
                if (selectedPlacement)
                  updateSelectedPlacement({ folder: e.target.value });
                else update("folder", e.target.value);
              }}
            >
              <option value="">{t("folder.none")}</option>
              {KNOWN_TECH_FOLDERS.map((f) => (
                <option key={f} value={f}>
                  {f}
                </option>
              ))}
              {(selectedPlacement?.folder ?? form.folder) &&
                !KNOWN_TECH_FOLDERS.includes(
                  selectedPlacement?.folder ?? form.folder,
                ) && (
                  <option value={selectedPlacement?.folder ?? form.folder}>
                    {t("folder.customCurrent", {
                      folder: selectedPlacement?.folder ?? form.folder,
                    })}
                  </option>
                )}
            </Select>
          </div>
        </section>

        <Separator />

        <section className="space-y-3">
          <SectionHeader
            title={t("prerequisites.heading")}
            tooltip={t("prerequisites.tooltipContent")}
          />
          <Select
            onChange={(e) => {
              addPrereq(e.target.value);
              e.target.value = "";
            }}
            defaultValue=""
          >
            <option value="" disabled>
              {t("addPrerequisitePlaceholder")}
            </option>
            {otherTechs
              .filter((t) => !(form.prerequisites || []).includes(t.tech_id))
              .map((t) => (
                <option key={t.id} value={t.tech_id}>
                  {t.tech_id} ({t.start_year})
                </option>
              ))}
          </Select>
          <div className="flex flex-wrap gap-1">
            {(form.prerequisites || []).map((pid) => (
              <Badge
                key={pid}
                variant="default"
                className="cursor-pointer text-xs"
                onClick={() => removePrereq(pid)}
              >
                {pid} x
              </Badge>
            ))}
          </div>
        </section>

        <Separator />

        <section className="space-y-3">
          <SectionHeader
            title={t("effects.title")}
            tooltip={t("effects.tooltipContent")}
          />
          <p className="text-xs text-muted-foreground">{t("effects.hint")}</p>
          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-1">
              <div className="flex items-center gap-1">
                <Label>{t("enableEquipments.heading")}</Label>
                <InfoTooltip
                  title={t("enableEquipments.tooltipTitle")}
                  content={t.raw("enableEquipments.tooltipContent")}
                />
              </div>
              <div
                className="max-h-40 overflow-y-auto rounded-md border border-input p-2 space-y-1"
                role="group"
                aria-label={t("enableEquipments.heading")}
              >
                {equipmentOptions.map((equipmentId) => {
                  const checked = (form.enable_equipments || []).includes(
                    equipmentId,
                  );
                  return (
                    <label
                      key={equipmentId}
                      className="flex items-center gap-2 text-xs font-mono"
                    >
                      <input
                        type="checkbox"
                        checked={checked}
                        onChange={() =>
                          update(
                            "enable_equipments",
                            checked
                              ? (form.enable_equipments || []).filter(
                                  (id) => id !== equipmentId,
                                )
                              : [
                                  ...(form.enable_equipments || []),
                                  equipmentId,
                                ],
                          )
                        }
                        className="rounded border-border"
                      />
                      {equipmentId}
                    </label>
                  );
                })}
              </div>
            </div>
            <div className="space-y-1">
              <div className="flex items-center gap-1">
                <Label>{t("enableSubunits.heading")}</Label>
                <InfoTooltip
                  title={t("enableSubunits.tooltipTitle")}
                  content={t.raw("enableSubunits.tooltipContent")}
                />
              </div>
              <Textarea
                value={(form.enable_subunits || []).join("\n")}
                onChange={(e) => {
                  update("enable_subunits", parseTokenLines(e.target.value));
                }}
                className="font-mono text-xs min-h-12"
                placeholder={"infantry\nmotorized"}
              />
            </div>
          </div>
          <div className="space-y-1">
            <div className="flex items-center gap-1">
              <Label>{t("enableModules.heading")}</Label>
              <InfoTooltip
                title={t("enableModules.tooltipTitle")}
                content={t.raw("enableModules.tooltipContent")}
              />
            </div>
            <Textarea
              value={(form.enable_equipment_modules || []).join("\n")}
              onChange={(e) => {
                update(
                  "enable_equipment_modules",
                  parseTokenLines(e.target.value),
                );
              }}
              className="font-mono text-xs min-h-12"
              placeholder={"tank_heavy_machine_gun\nadditional_machine_guns"}
            />
          </div>
        </section>

        <section className="space-y-3">
          <SectionHeader
            title={t("globalModifier.title")}
            tooltip={t("globalModifier.tooltipContent")}
          />
          <p className="text-xs text-muted-foreground">
            {t("globalModifier.hint")}
          </p>
          <Textarea
            value={
              ((form.modifiers as Record<string, unknown>)
                ?.__modifier as string) || ""
            }
            onChange={(e) => {
              const m = { ...(form.modifiers || {}) };
              if (e.target.value)
                (m as Record<string, unknown>).__modifier = e.target.value;
              else delete (m as Record<string, unknown>).__modifier;
              update("modifiers", m);
            }}
            className="font-mono text-xs min-h-16"
            placeholder={
              "research_speed_factor = 0.03\nproduction_factory_efficiency_gain_factor = 0.1"
            }
          />
        </section>

        <Separator />

        <section className="space-y-3">
          <SectionHeader
            title={t("researchScripts.title")}
            tooltip={t("researchScripts.tooltipContent")}
          />
          <div className="space-y-1">
            <div className="flex items-center gap-1">
              <Label>{t("onResearchComplete.heading")}</Label>
              <InfoTooltip
                title={t("onResearchComplete.tooltipTitle")}
                content={t.raw("onResearchComplete.tooltipContent")}
              />
            </div>
            <p className="text-xs text-muted-foreground">
              {t("onResearchComplete.hint")}
            </p>
            <Textarea
              value={form.on_research_complete || ""}
              onChange={(e) => update("on_research_complete", e.target.value)}
              className="font-mono text-xs min-h-12"
              placeholder={
                'hidden_effect = {\n\tload_oob = "unlock_paratroopers"\n}\ncustom_effect_tooltip = UNLOCK_DIVISION_TEMPLATE'
              }
            />
          </div>
          <div className="space-y-1">
            <div className="flex items-center gap-1">
              <Label>{t("onResearchCompleteLimit.heading")}</Label>
              <InfoTooltip
                title={t("onResearchCompleteLimit.tooltipTitle")}
                content={t.raw("onResearchCompleteLimit.tooltipContent")}
              />
            </div>
            <p className="text-xs text-muted-foreground">
              {t("onResearchCompleteLimit.hint")}
            </p>
            <Textarea
              value={form.on_research_complete_limit || ""}
              onChange={(e) =>
                update("on_research_complete_limit", e.target.value)
              }
              className="font-mono text-xs min-h-12"
              placeholder={
                "NOT = { has_template_containing_unit = paratrooper }"
              }
            />
          </div>
          <div className="space-y-1">
            <div className="flex items-center gap-1">
              <Label>{t("allow.heading")}</Label>
              <InfoTooltip
                title={t("allow.tooltipTitle")}
                content={t.raw("allow.tooltipContent")}
              />
            </div>
            <p className="text-xs text-muted-foreground">{t("allow.hint")}</p>
            <Textarea
              value={form.allow || ""}
              onChange={(e) => update("allow", e.target.value)}
              className="font-mono text-xs min-h-12"
              placeholder={"always = no"}
            />
          </div>
        </section>

        <Separator />

        <section className="space-y-3">
          <details className="group">
            <summary className="cursor-pointer text-sm font-semibold text-muted-foreground uppercase tracking-widest select-none">
              {t("advancedFieldsLabel")}
            </summary>
            <div className="mt-3 space-y-3">
              <div className="grid grid-cols-2 gap-3">
                <div className="space-y-1">
                  <div className="flex items-center gap-1">
                    <Label>enable_building</Label>
                    <InfoTooltip
                      title="enable_building"
                      content={t.raw("enableBuilding.tooltipContent")}
                    />
                  </div>
                  <Input
                    type="text"
                    value={form.enable_building || ""}
                    onChange={(e) => update("enable_building", e.target.value)}
                    className="font-mono text-xs"
                    placeholder="synthetic_refinery"
                  />
                </div>
                <div className="space-y-1">
                  <div className="flex items-center gap-1">
                    <Label>enable_tactic</Label>
                    <InfoTooltip
                      title="enable_tactic"
                      content={t.raw("enableTactic.tooltipContent")}
                    />
                  </div>
                  <Input
                    type="text"
                    value={form.enable_tactic || ""}
                    onChange={(e) => update("enable_tactic", e.target.value)}
                    className="font-mono text-xs"
                    placeholder="encirclement"
                  />
                </div>
              </div>

              <div className="space-y-1">
                <div className="flex items-center gap-1">
                  <Label>{t("xor.heading")}</Label>
                  <InfoTooltip
                    title={t("xor.tooltipTitle")}
                    content={t.raw("xor.tooltipContent")}
                  />
                </div>
                <Textarea
                  value={(form.xor || []).join("\n")}
                  onChange={(e) => {
                    update("xor", parseTokenLines(e.target.value));
                  }}
                  className="font-mono text-xs min-h-12"
                  placeholder={"alternative_tech_1\nalternative_tech_2"}
                />
              </div>

              <div className="space-y-1">
                <div className="flex items-center gap-1">
                  <Label>{t("subTechnologies.heading")}</Label>
                  <InfoTooltip
                    title={t("subTechnologies.tooltipTitle")}
                    content={t.raw("subTechnologies.tooltipContent")}
                  />
                </div>
                <Textarea
                  value={(form.sub_technologies || []).join("\n")}
                  onChange={(e) => {
                    update("sub_technologies", parseTokenLines(e.target.value));
                  }}
                  className="font-mono text-xs min-h-12"
                  placeholder={"infantry_at\ninfantry_aa"}
                />
              </div>

              <div className="space-y-1">
                <div className="flex items-center gap-1">
                  <Label>{t("dependencies.heading")}</Label>
                  <InfoTooltip
                    title={t("dependencies.tooltipTitle")}
                    content={t.raw("dependencies.tooltipContent")}
                  />
                </div>
                <Textarea
                  value={formatDependencyLines(form.dependencies)}
                  onChange={(e) => {
                    update(
                      "dependencies",
                      parseDependencyLines(e.target.value),
                    );
                  }}
                  className="font-mono text-xs min-h-12"
                  placeholder={"some_other_tech = 1"}
                />
              </div>

              <div className="space-y-1">
                <div className="flex items-center gap-1">
                  <Label>{t("allowBranch.heading")}</Label>
                  <InfoTooltip
                    title={t("allowBranch.tooltipTitle")}
                    content={t.raw("allowBranch.tooltipContent")}
                  />
                </div>
                <Textarea
                  value={form.allow_branch || ""}
                  onChange={(e) => update("allow_branch", e.target.value)}
                  className="font-mono text-xs min-h-12"
                  placeholder={'has_dlc = "Man the Guns"'}
                />
              </div>

              <div className="grid grid-cols-2 gap-3">
                <div className="space-y-1">
                  <div className="flex items-center gap-1">
                    <Label>{t("xpResearchType.heading")}</Label>
                    <InfoTooltip
                      title={t("xpResearchType.tooltipTitle")}
                      content={t.raw("xpResearchType.tooltipContent")}
                    />
                  </div>
                  <Select
                    className="px-2 text-xs font-mono"
                    value={form.xp_research_type || ""}
                    onChange={(e) => update("xp_research_type", e.target.value)}
                  >
                    <option value="">{t("xpResearchType.optNone")}</option>
                    <option value="land">{t("xpResearchType.optLand")}</option>
                    <option value="naval">
                      {t("xpResearchType.optNaval")}
                    </option>
                    <option value="air">{t("xpResearchType.optAir")}</option>
                  </Select>
                </div>
                <div className="space-y-1">
                  <div className="flex items-center gap-1">
                    <Label>xp_boost_cost</Label>
                    <InfoTooltip
                      title={t("xpBoostCost.tooltipTitle")}
                      content={t.raw("xpBoostCost.tooltipContent")}
                    />
                  </div>
                  <Input
                    type="number"
                    min={0}
                    max={1000}
                    value={form.xp_boost_cost ?? 0}
                    onChange={(e) =>
                      update("xp_boost_cost", parseInt(e.target.value) || 0)
                    }
                    className="font-mono text-xs"
                  />
                </div>
              </div>

              <div className="space-y-1">
                <div className="flex items-center gap-1">
                  <Label>{t("xpResearchBonus.heading")}</Label>
                  <InfoTooltip
                    title={t("xpResearchBonus.tooltipTitle")}
                    content={t.raw("xpResearchBonus.tooltipContent")}
                  />
                </div>
                {(() => {
                  const xpb = form.xp_research_bonus;
                  const isBlock =
                    !!xpb &&
                    typeof xpb === "object" &&
                    !("__scalar" in xpb) &&
                    Object.keys(xpb).length > 0;
                  const scalar =
                    xpb && "__scalar" in xpb
                      ? (xpb as Record<string, number>)["__scalar"]
                      : undefined;
                  return isBlock ? (
                    <p className="text-xs text-muted-foreground">
                      {t("xpResearchBonus.blockFormNote")}
                    </p>
                  ) : (
                    <Input
                      type="number"
                      step="0.05"
                      min={0}
                      value={scalar ?? ""}
                      onChange={(e) => {
                        const v = e.target.value;
                        if (v === "") {
                          update("xp_research_bonus", {});
                        } else {
                          const n = parseFloat(v);
                          update(
                            "xp_research_bonus",
                            Number.isFinite(n) ? { __scalar: n } : {},
                          );
                        }
                      }}
                      className="font-mono text-xs"
                      placeholder="1.50"
                    />
                  );
                })()}
              </div>

              <div className="flex items-center gap-2 pt-1">
                <input
                  type="checkbox"
                  id={`tech-doctrine-${form.id}`}
                  checked={Boolean(form.doctrine)}
                  onChange={(e) => update("doctrine", e.target.checked)}
                  className="size-4 accent-accent"
                />
                <Label
                  htmlFor={`tech-doctrine-${form.id}`}
                  className="cursor-pointer"
                >
                  {t("doctrine.heading")}
                </Label>
                <InfoTooltip
                  title="doctrine · legacy"
                  content={t.raw("doctrine.tooltipContent")}
                />
              </div>

              {form.path && form.path.length > 0 && (
                <div className="space-y-1">
                  <Label>{t("advanced.pathField")}</Label>
                  <div className="text-xs font-mono space-y-0.5">
                    {form.path.map((p, i) => (
                      <div key={i}>
                        → {p.leads_to_tech} (×{p.research_cost_coeff})
                      </div>
                    ))}
                  </div>
                </div>
              )}
            </div>
          </details>
        </section>

        <Separator />

        <section className="space-y-3">
          <SectionHeader
            title={t("statBonuses.title")}
            tooltip={t("statBonuses.tooltipContent")}
          />
          <p className="text-xs text-muted-foreground">
            {t("statBonuses.hint")}
          </p>
          <div className="flex gap-2 items-end">
            <div className="flex-1 space-y-1">
              <Label>{t("statBonuses.category")}</Label>
              <Select
                className="px-2 text-xs"
                value={modCategory}
                onChange={(e) => setModCategory(e.target.value)}
              >
                <option value="">{t("statBonuses.categoryPlaceholder")}</option>
                {MODIFIER_CATEGORY_OPTIONS.map((opt) => (
                  <option key={opt.value} value={opt.value}>
                    {tTech(opt.i18nKey)}
                  </option>
                ))}
              </Select>
            </div>
            <div className="flex-1 space-y-1">
              <Label>{t("statBonuses.stat")}</Label>
              <Select
                className="px-2 text-xs"
                value={modStat}
                onChange={(e) => setModStat(e.target.value)}
              >
                <option value="">{t("statBonuses.statPlaceholder")}</option>
                {MODIFIER_STAT_OPTIONS.map((opt) => (
                  <option
                    key={opt.value}
                    value={opt.value}
                    title={t(`statDesc.${opt.value}`)}
                  >
                    {tTech(opt.i18nKey)}
                  </option>
                ))}
              </Select>
            </div>
            <div className="w-20 space-y-1">
              <Label>{t("statBonuses.value")}</Label>
              <Input
                type="number"

                step={
                  MODIFIER_STAT_OPTIONS.find((o) => o.value === modStat)
                    ?.isPercent
                    ? 0.01
                    : 1
                }
                value={modValue}
                onChange={(e) => setModValue(e.target.value)}
              />
            </div>
            <Button
              size="sm"
              variant="outline"
              onClick={addModifier}
              className="shrink-0"
            >
              +
            </Button>
          </div>
          {Object.entries(form.modifiers || {})
            .filter(([k]) => !k.startsWith("__"))
            .map(([cat, stats]) => (
              <div key={cat} className="text-xs">
                <span className="font-mono text-muted-foreground">
                  {(() => {
                    const o = MODIFIER_CATEGORY_OPTIONS.find(
                      (x) => x.value === cat,
                    );
                    return o ? tTech(o.i18nKey) : cat;
                  })()}
                  :
                </span>
                <div className="flex flex-wrap gap-1 mt-1">
                  {Object.entries(stats as Record<string, number>).map(
                    ([stat, val]) => {
                      const opt = MODIFIER_STAT_OPTIONS.find(
                        (o) => o.value === stat,
                      );

                      const display = opt?.isPercent
                        ? `${val > 0 ? "+" : ""}${(val * 100).toFixed(0)}%`
                        : `${val > 0 ? "+" : ""}${val}`;
                      const badge = (
                        <Badge
                          variant="outline"
                          className="cursor-pointer text-xs"
                          onClick={() => removeModifier(cat, stat)}
                        >
                          {opt ? tTech(opt.i18nKey) : stat} = {display} x
                        </Badge>
                      );
                      if (!opt) return <span key={stat}>{badge}</span>;
                      return (
                        <Tooltip key={stat}>
                          <TooltipTrigger render={<span>{badge}</span>} />
                          <TooltipContent>
                            <div className="font-medium mb-1">
                              {tTech(opt.i18nKey)}
                            </div>
                            <div className="text-muted-foreground whitespace-pre-line">
                              {t(`statDesc.${opt.value}`)}
                            </div>
                          </TooltipContent>
                        </Tooltip>
                      );
                    },
                  )}
                </div>
              </div>
            ))}
        </section>

        <Separator />

        <section className="space-y-3">
          <h3 className="text-sm font-semibold text-muted-foreground uppercase tracking-wide">
            {t("section.categories")}
          </h3>
          <div className="flex flex-wrap gap-1 max-h-32 overflow-y-auto">
            {COMMON_CATEGORIES.map((cat) => (
              <Badge
                key={cat}
                variant={
                  (form.categories || []).includes(cat) ? "default" : "outline"
                }
                className="cursor-pointer text-xs"
                onClick={() => toggleCategory(cat)}
              >
                {cat}
              </Badge>
            ))}
          </div>
        </section>

        <Separator />

        <section className="space-y-3">
          <h3 className="text-sm font-semibold text-muted-foreground uppercase tracking-wide">
            {t("section.aiWillDo")}
          </h3>
          <Textarea
            value={form.ai_will_do}
            onChange={(e) => update("ai_will_do", e.target.value)}
            className="font-mono text-xs min-h-12"
            placeholder="factor = 1"
          />
        </section>

        <Separator />

        <section className="space-y-3">
          <h3 className="text-sm font-semibold text-muted-foreground uppercase tracking-wide">
            {t("section.description")}
          </h3>
          <DescriptionLocsTabs
            entityKey={`technology:${tech.id}:description`}
            value={form.description_locs || {}}
            exportLanguages={exportLanguages}
            legacyFallback={{ en: form.desc_en, zh: form.desc_zh }}
            onChange={(next) => update("description_locs", next)}
            ariaLabel="Technology description"
            rows={3}
          />
        </section>

        <Separator />

        <div className="space-y-2">
          <Button onClick={handleSave} disabled={saving} className="w-full">
            {t("saveTechButton")}
          </Button>
          {onDelete && (
            <Button
              variant="destructive"
              size="sm"
              className="w-full"
              onClick={async () => {
                if (
                  await confirmDialog({
                    body: `Delete tech "${form.name_en || form.name || form.tech_id}" (${form.tech_id})?`,
                    danger: true,
                  })
                ) {
                  await onDelete(tech.id);
                }
              }}
            >
              {t("deleteTechButton")}
            </Button>
          )}
        </div>
      </div>
      <SourceEditorDialog
        open={sourceGate.showDialog}
        entityType="tech"
        entityId={tech.tech_id}
        entityLabel={tech.name_en || tech.tech_id}
        hideForkOption
        onChoice={sourceGate.handleChoice}
        onCancel={sourceGate.handleCancel}
      />
    </TooltipProvider>
  );
}
