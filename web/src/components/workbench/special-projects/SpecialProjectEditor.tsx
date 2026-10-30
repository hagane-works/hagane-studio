"use client";

import { useState } from "react";
import { useTranslations } from "use-intl";
import { Trash2 } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { Select } from "@/components/ui/select";
import { useContentLocaleStore } from "@/lib/store/contentLocaleStore";
import { resolveDisplayName } from "@/lib/i18n/localizedName";
import type { Specialization, SpecialProject } from "@/types";
import { SPECIAL_PROJECT_COMPLEXITY_OPTIONS } from "@/types";

const COMPLEXITY_LABEL_KEYS: Record<string, string> = {
  "sp_complexity.small": "complexitySmall",
  "sp_complexity.medium": "complexityMedium",
  "sp_complexity.large": "complexityLarge",
  "sp_complexity.insane": "complexityInsane",
};

interface ProjectDraft {
  project_token: string;
  name_en: string;
  name_zh: string;
  desc_en: string;
  desc_zh: string;
  display_name_locs: Record<string, string>;
  description_locs: Record<string, string>;
  specialization: string;
  complexity: string;
  prototype_time: string;
  icon: string;
  project_tags: string;
  special_project_parent: string[];
  allowed: string;
  visible: string;
  available: string;
  resource_cost: string;
  project_output: string;
  generic_prototype_rewards: string;
}

function projToDraft(p: SpecialProject): ProjectDraft {
  return {
    project_token: p.project_token,
    name_en: p.name_en,
    name_zh: p.name_zh,
    desc_en: p.desc_en,
    desc_zh: p.desc_zh,
    display_name_locs: { ...(p.display_name_locs ?? {}) },
    description_locs: { ...(p.description_locs ?? {}) },
    specialization: p.specialization,
    complexity: p.complexity,
    prototype_time: p.prototype_time,
    icon: p.icon,
    project_tags: (p.project_tags || []).join(" "),
    special_project_parent: p.special_project_parent || [],
    allowed: p.allowed,
    visible: p.visible,
    available: p.available,
    resource_cost: p.resource_cost,
    project_output: p.project_output,
    generic_prototype_rewards: (p.generic_prototype_rewards || []).join(" "),
  };
}

function draftToPayload(d: ProjectDraft): Partial<SpecialProject> {
  return {
    project_token: d.project_token.trim(),
    name_en: d.name_en,
    name_zh: d.name_zh,
    desc_en: d.desc_en,
    desc_zh: d.desc_zh,
    display_name_locs: d.display_name_locs,
    description_locs: d.description_locs,
    specialization: d.specialization,
    complexity: d.complexity,
    prototype_time: d.prototype_time,
    icon: d.icon,
    project_tags: d.project_tags.split(/\s+/).filter(Boolean),
    special_project_parent: d.special_project_parent,
    allowed: d.allowed,
    visible: d.visible,
    available: d.available,
    resource_cost: d.resource_cost,
    project_output: d.project_output,
    generic_prototype_rewards: d.generic_prototype_rewards
      .split(/\s+/)
      .filter(Boolean),
  };
}

export function SpecialProjectEditor({
  project,
  specs,
  onSave,
  onDelete,
}: {
  project: SpecialProject;

  specs: Specialization[];

  onSave: (data: Partial<SpecialProject>) => Promise<void>;

  onDelete?: () => void;
}) {
  const t = useTranslations("specialProjects");
  const tContent = useTranslations("decisions");
  const contentLocale = useContentLocaleStore((s) => s.locale);

  const [draft, setDraft] = useState<ProjectDraft>(() => projToDraft(project));
  const [shown, setShown] = useState(project);
  if (shown !== project) {
    setShown(project);
    setDraft(projToDraft(project));
  }
  const [saving, setSaving] = useState(false);
  const [saveError, setSaveError] = useState<string | null>(null);

  const save = async () => {
    setSaving(true);
    setSaveError(null);
    try {
      await onSave(draftToPayload(draft));
    } catch (err) {
      setSaveError(err instanceof Error ? err.message : String(err));
    } finally {
      setSaving(false);
    }
  };

  const setDraftField = <K extends keyof ProjectDraft>(
    key: K,
    value: ProjectDraft[K],
  ) => setDraft((d) => ({ ...d, [key]: value }));

  return (
    <div className="p-3 space-y-3">
      <div>
        <Label>{t("fieldToken")}</Label>
        <Input
          value={draft.project_token}
          onChange={(e) => setDraftField("project_token", e.target.value)}
          className="font-mono text-xs"
        />
      </div>
      <div>
        <Label>
          {tContent("contentNameLabel", {
            locale: contentLocale.toUpperCase(),
          })}
        </Label>
        <Input
          value={resolveDisplayName(
            contentLocale,
            draft.display_name_locs,
            draft.name_en,
            draft.name_zh,
            "",
          )}
          onChange={(e) =>
            setDraftField("display_name_locs", {
              ...draft.display_name_locs,
              [contentLocale]: e.target.value,
            })
          }
        />
      </div>
      <div>
        <Label>
          {tContent("contentDescriptionLabel", {
            locale: contentLocale.toUpperCase(),
          })}
        </Label>
        <Textarea
          value={resolveDisplayName(
            contentLocale,
            draft.description_locs,
            draft.desc_en,
            draft.desc_zh,
            "",
          )}
          onChange={(e) =>
            setDraftField("description_locs", {
              ...draft.description_locs,
              [contentLocale]: e.target.value,
            })
          }
          rows={3}
        />
      </div>
      <div>
        <Label>{t("fieldSpecialization")}</Label>
        <Select
          className="px-2"
          value={draft.specialization}
          onChange={(e) => setDraftField("specialization", e.target.value)}
        >
          <option value="">—</option>
          {specs.map((s) => (
            <option key={s.id} value={s.specialization_id}>
              {s.specialization_id}
            </option>
          ))}
        </Select>
      </div>
      <div>
        <Label>{t("fieldComplexity")}</Label>
        <Select
          className="px-2"
          value={draft.complexity}
          onChange={(e) => setDraftField("complexity", e.target.value)}
        >
          <option value="">{t("fieldComplexityNone")}</option>
          {SPECIAL_PROJECT_COMPLEXITY_OPTIONS.map((o) => (
            <option key={o.value} value={o.value}>
              {COMPLEXITY_LABEL_KEYS[o.value]
                ? t(COMPLEXITY_LABEL_KEYS[o.value])
                : o.label}
            </option>
          ))}
        </Select>
      </div>
      <div>
        <Label>{t("fieldPrototypeTime")}</Label>
        <Input
          value={draft.prototype_time}
          onChange={(e) => setDraftField("prototype_time", e.target.value)}
          placeholder="sp_time.prototype.long"
          className="font-mono text-xs"
        />
      </div>
      <div>
        <Label>{t("fieldIcon")}</Label>
        <Input
          value={draft.icon}
          onChange={(e) => setDraftField("icon", e.target.value)}
          placeholder="GFX_sp_nuclear_reactor"
          className="font-mono text-xs"
        />
      </div>
      <div>
        <Label>{t("fieldProjectTags")}</Label>
        <Input
          value={draft.project_tags}
          onChange={(e) => setDraftField("project_tags", e.target.value)}
          placeholder="sp_tag_nuclear_power"
          className="font-mono text-xs"
        />
      </div>
      <div>
        <Label>{t("fieldParents")}</Label>
        <p className="text-micro text-muted-foreground mb-1">
          {t("fieldParentsHint")}
        </p>
        <Input
          value={draft.special_project_parent.join(" ")}
          onChange={(e) =>
            setDraftField(
              "special_project_parent",
              e.target.value.split(/\s+/).filter(Boolean),
            )
          }
          className="font-mono text-xs"
        />
      </div>
      <div>
        <Label>{t("fieldAllowed")}</Label>
        <Textarea
          value={draft.allowed}
          onChange={(e) => setDraftField("allowed", e.target.value)}
          rows={3}
          className="font-mono text-xs"
        />
      </div>
      <div>
        <Label>{t("fieldVisible")}</Label>
        <Textarea
          value={draft.visible}
          onChange={(e) => setDraftField("visible", e.target.value)}
          rows={3}
          className="font-mono text-xs"
        />
      </div>
      <div>
        <Label>{t("fieldAvailable")}</Label>
        <Textarea
          value={draft.available}
          onChange={(e) => setDraftField("available", e.target.value)}
          rows={3}
          className="font-mono text-xs"
        />
      </div>
      <div>
        <Label>{t("fieldResourceCost")}</Label>
        <Textarea
          value={draft.resource_cost}
          onChange={(e) => setDraftField("resource_cost", e.target.value)}
          rows={2}
          className="font-mono text-xs"
        />
      </div>
      <div>
        <Label>{t("fieldProjectOutput")}</Label>
        <Textarea
          value={draft.project_output}
          onChange={(e) => setDraftField("project_output", e.target.value)}
          rows={4}
          className="font-mono text-xs"
        />
      </div>
      <div>
        <Label>{t("fieldGenericRewards")}</Label>
        <Input
          value={draft.generic_prototype_rewards}
          onChange={(e) =>
            setDraftField("generic_prototype_rewards", e.target.value)
          }
          className="font-mono text-xs"
        />
      </div>

      <div className="flex gap-2 pt-1">
        <Button
          onClick={() => void save()}
          disabled={saving}
          className="flex-1"
        >
          {saving ? t("saving") : t("save")}
        </Button>
        {onDelete && (
          <Button
            variant="destructive"
            size="icon"
            onClick={onDelete}
            aria-label={t("delete")}
          >
            <Trash2 className="w-4 h-4" />
          </Button>
        )}
      </div>
      {saveError && <p className="text-xs text-destructive">{saveError}</p>}
    </div>
  );
}
