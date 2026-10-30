"use client";

import { useCallback, useState } from "react";
import { useTranslations } from "use-intl";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { intelligenceAgenciesApi } from "@/lib/api";
import type {
  IntelligenceAgencyRow,
  IntelAgencyBranch,
  IntelAgencyUpgrade,
} from "@/lib/api";
import { useBaseModDepKeys, gfxUrl } from "@/lib/api/basemod";
import { useProjectStore } from "@/lib/store/projectStore";
import { DisplayNameLocsTabs } from "@/components/shared/editors/DisplayNameLocsTabs";
import { DEFAULT_EXPORT_LANGUAGES } from "@/components/shared/editors/displayNameLocsHelpers";
import { ImageUpload } from "@/shared/upload/ImageUpload";
import { useRegisterDirty } from "@/lib/hooks/useUnsavedGuard";
import { IntelSaveBar, useIntelSave } from "./IntelSaveBar";
import { IntelUpgradeFields } from "./IntelUpgradeEditor";

const LOGO_SIZE = { width: 117, height: 119 };

export type IntelAgencySaveData = Parameters<
  typeof intelligenceAgenciesApi.update
>[2];

interface DraftAgency {
  agency_id: string;
  picture: string;

  names: string[];
  default_trigger: string;
  available: string;

  upgrade_branches: IntelAgencyBranch[];
  name_en: string;
  name_zh: string;
  display_name_locs: Record<string, string>;
}

function agencyToDraft(a: IntelligenceAgencyRow): DraftAgency {
  return {
    agency_id: a.agency_id || "",
    picture: a.picture || "",
    names: Array.isArray(a.names) ? [...a.names] : [],
    default_trigger: a.default_trigger || "",
    available: a.available || "",
    upgrade_branches: Array.isArray(a.upgrade_branches)
      ? JSON.parse(JSON.stringify(a.upgrade_branches))
      : [],
    name_en: a.name_en || "",
    name_zh: a.name_zh || "",
    display_name_locs: { ...(a.display_name_locs ?? {}) },
  };
}

export function IntelAgencyEditor({
  projectId,
  agency,
  onSave,
  onLogoUploaded,
}: {
  projectId: number;

  agency: IntelligenceAgencyRow;

  onSave: (data: IntelAgencySaveData) => Promise<void>;

  onLogoUploaded?: (picture: string) => void;
}) {
  const t = useTranslations("intelligenceAgencies");
  const exportLanguages = useProjectStore(
    (s) => s.currentProject?.export_languages ?? DEFAULT_EXPORT_LANGUAGES,
  );

  const baseModDepKeys = useBaseModDepKeys();
  const [draft, setDraft] = useState<DraftAgency>(() => agencyToDraft(agency));
  const [savedDraft, setSavedDraft] = useState<DraftAgency>(draft);
  const { saving, status, error, run } = useIntelSave();

  const update = useCallback(
    <K extends keyof DraftAgency>(key: K, value: DraftAgency[K]) => {
      setDraft((d) => ({ ...d, [key]: value }));
    },
    [],
  );

  const handleSave = () =>
    run(async () => {
      await onSave({
        agency_id: draft.agency_id.trim(),
        picture: draft.picture,
        names: draft.names.map((n) => n.trim()).filter((n) => n.length > 0),
        default_trigger: draft.default_trigger,
        available: draft.available,
        upgrade_branches: draft.upgrade_branches,
        name_en: draft.name_en,
        name_zh: draft.name_zh,
        display_name_locs: draft.display_name_locs,
      });
      setSavedDraft(draft);
    });

  const isUnsaved = JSON.stringify(draft) !== JSON.stringify(savedDraft);

  useRegisterDirty("intelligence", isUnsaved);

  const setName = (i: number, value: string) => {
    const next = [...draft.names];
    next[i] = value;
    update("names", next);
  };
  const addName = () => update("names", [...draft.names, ""]);
  const removeName = (i: number) =>
    update(
      "names",
      draft.names.filter((_, j) => j !== i),
    );

  const addBranch = () =>
    update("upgrade_branches", [
      ...draft.upgrade_branches,
      { branch: "branch_intelligence", upgrades: [] },
    ]);
  const setBranchName = (bi: number, value: string) => {
    const next = draft.upgrade_branches.map((b, j) =>
      j === bi ? { ...b, branch: value } : b,
    );
    update("upgrade_branches", next);
  };
  const removeBranch = (bi: number) =>
    update(
      "upgrade_branches",
      draft.upgrade_branches.filter((_, j) => j !== bi),
    );
  const addUpgrade = (bi: number) => {
    const next = draft.upgrade_branches.map((b, j) =>
      j === bi
        ? {
            ...b,
            upgrades: [
              ...b.upgrades,
              { token: "", picture: "", levels: [""] } as IntelAgencyUpgrade,
            ],
          }
        : b,
    );
    update("upgrade_branches", next);
  };
  const setUpgrade = (bi: number, ui: number, value: IntelAgencyUpgrade) => {
    const next = draft.upgrade_branches.map((b, j) => {
      if (j !== bi) return b;
      const upgrades = b.upgrades.map((u, k) => (k === ui ? value : u));
      return { ...b, upgrades };
    });
    update("upgrade_branches", next);
  };
  const removeUpgrade = (bi: number, ui: number) => {
    const next = draft.upgrade_branches.map((b, j) =>
      j === bi ? { ...b, upgrades: b.upgrades.filter((_, k) => k !== ui) } : b,
    );
    update("upgrade_branches", next);
  };

  const isManagedLogo =
    draft.picture === `GFX_intelligence_agency_logo_${draft.agency_id}`;

  return (
    <div className="p-4 max-w-3xl space-y-4">
      <div>
        <h2 className="text-sm font-semibold mb-1">{t("title")}</h2>
        <p className="text-xs text-muted-foreground">
          {t("landing")}{" "}
          <span className="font-mono">
            common/intelligence_agencies/{`{TAG}`}_intelligence_agencies.txt
          </span>
        </p>
        <p className="text-xs text-muted-foreground mt-1">
          {t("anonymousNote")}
        </p>
      </div>

      <div className="grid grid-cols-2 gap-3">
        <div className="space-y-1">
          <Label>{t("agencyIdLabel")}</Label>
          <Input
            className="font-mono text-xs"
            value={draft.agency_id}
            onChange={(e) => update("agency_id", e.target.value)}
          />
          <p className="text-micro text-muted-foreground">
            {t("agencyIdHint")}
          </p>
        </div>
        <div className="space-y-1">
          <Label>{t("pictureLabel")}</Label>
          <Input
            className="font-mono text-xs"
            placeholder="GFX_intelligence_agency_logo_generic_1"
            value={draft.picture}
            onChange={(e) => update("picture", e.target.value)}
          />
        </div>
      </div>

      <ImageUpload
        label={t("logoUploadLabel")}
        targetSize={LOGO_SIZE}
        previewUrl={
          isManagedLogo
            ? intelligenceAgenciesApi.logoUrl(projectId, agency.id)
            : null
        }
        hint={t("logoUploadHint")}
        gfxName={draft.picture || undefined}
        baseModFallbackUrls={
          draft.picture && baseModDepKeys.length
            ? baseModDepKeys.map((dk) => gfxUrl(projectId, dk, draft.picture))
            : []
        }
        onUpload={(file, crop) =>
          intelligenceAgenciesApi.uploadLogo(projectId, agency.id, file, crop)
        }
        onUploaded={(result) => {
          const pic = result["picture"];
          if (typeof pic === "string") {
            update("picture", pic);
            onLogoUploaded?.(pic);
          }
        }}
      />

      <div className="space-y-1">
        <Label>{t("displayNameLabel")}</Label>
        <DisplayNameLocsTabs
          entityKey={`intelligence-agency:${agency.id}:display-name`}
          value={draft.display_name_locs}
          exportLanguages={exportLanguages}
          legacyFallback={{ en: draft.name_en, zh: draft.name_zh }}
          onChange={(next) => update("display_name_locs", next)}
          ariaLabel={t("displayNameLabel")}
        />
      </div>

      <div className="space-y-2">
        <div className="flex items-center justify-between">
          <Label>{t("namesLabel")}</Label>
          <Button
            size="sm"
            variant="outline"
            className="text-xs"
            onClick={addName}
          >
            {t("addName")}
          </Button>
        </div>
        {draft.names.length === 0 && (
          <p className="text-xs text-muted-foreground italic">{t("noNames")}</p>
        )}
        {draft.names.map((n, i) => (
          <div key={i} className="grid grid-cols-[1fr_auto] gap-2">
            <Input
              placeholder="Office of Strategic Services"
              value={n}
              onChange={(e) => setName(i, e.target.value)}
            />
            <Button
              size="sm"
              variant="ghost"
              className="px-2 text-muted-foreground hover:text-destructive"
              onClick={() => removeName(i)}
              aria-label={t("removeNameAria")}
              title={t("removeNameAria")}
            >
              ×
            </Button>
          </div>
        ))}
      </div>

      <div className="space-y-1">
        <Label>{t("defaultTriggerLabel")}</Label>
        <Textarea
          className="font-mono text-xs min-h-20"
          placeholder={"tag = USA"}
          value={draft.default_trigger}
          onChange={(e) => update("default_trigger", e.target.value)}
        />
        <p className="text-micro text-muted-foreground">
          {t("defaultTriggerHint")}
        </p>
      </div>

      <div className="space-y-1">
        <Label>{t("availableLabel")}</Label>
        <Textarea
          className="font-mono text-xs min-h-20"
          placeholder={"original_tag = USA"}
          value={draft.available}
          onChange={(e) => update("available", e.target.value)}
        />
        <p className="text-micro text-muted-foreground">{t("availableHint")}</p>
      </div>

      <div className="space-y-2 border-t border-border pt-3">
        <div className="flex items-center justify-between">
          <Label>{t("upgradesLabel")}</Label>
          <Button
            size="sm"
            variant="outline"
            className="text-xs"
            onClick={addBranch}
          >
            {t("addBranch")}
          </Button>
        </div>
        <p className="text-micro text-muted-foreground">{t("upgradesHint")}</p>
        {draft.upgrade_branches.length === 0 && (
          <p className="text-xs text-muted-foreground italic">
            {t("noUpgrades")}
          </p>
        )}
        {draft.upgrade_branches.map((b, bi) => (
          <div key={bi} className="rounded border border-border p-2 space-y-2">
            <div className="grid grid-cols-[1fr_auto] gap-2 items-center">
              <Input
                placeholder="branch_intelligence"
                className="font-mono text-xs"
                value={b.branch}
                onChange={(e) => setBranchName(bi, e.target.value)}
              />
              <Button
                size="sm"
                variant="ghost"
                className="px-2 text-muted-foreground hover:text-destructive"
                onClick={() => removeBranch(bi)}
                aria-label={t("removeBranchAria")}
                title={t("removeBranchAria")}
              >
                ×
              </Button>
            </div>
            {b.upgrades.map((u, ui) => (
              <IntelUpgradeFields
                key={ui}
                upgrade={u}
                onChange={(next) => setUpgrade(bi, ui, next)}
                onRemove={() => removeUpgrade(bi, ui)}
              />
            ))}
            <Button
              size="sm"
              variant="outline"
              className="text-xs w-full"
              onClick={() => addUpgrade(bi)}
            >
              {t("addUpgrade")}
            </Button>
          </div>
        ))}
      </div>

      <IntelSaveBar
        saving={saving}
        status={status}
        error={error}
        unsaved={isUnsaved}
        onSave={() => void handleSave()}
      />
    </div>
  );
}
