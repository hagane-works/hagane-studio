"use client";

import { useTranslations } from "use-intl";
import { Plus, Save, Trash2 } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import {
  PLAN_BLOCK_FIELDS,
  type AiStrategyPlanDraft,
  type AiTemplateDraft,
  type PlanBlockField,
} from "@/lib/api/aiPlanning";

type SaveStatus = "idle" | "success" | "error";
interface EditorActions {
  onSave: () => void;
  onDelete: () => void;
  saving?: boolean;
  dirty?: boolean;
  saveStatus?: SaveStatus;
}

function SaveState({
  saving,
  dirty,
  saveStatus,
}: Pick<EditorActions, "saving" | "dirty" | "saveStatus">) {
  const t = useTranslations("aiPlanning");
  if (!saving && !dirty && saveStatus === "idle") return null;
  const tone = saving
    ? "border-border bg-muted text-muted-foreground"
    : saveStatus === "success"
      ? "border-success/20 bg-success/10 text-success"
      : saveStatus === "error"
        ? "border-destructive/20 bg-destructive/10 text-destructive"
        : "border-warning/20 bg-warning/10 text-warning";
  return (
    <div className={`border-y px-3 py-2 text-xs font-medium ${tone}`}>
      {saving
        ? t("saving")
        : saveStatus === "success"
          ? t("saved")
          : saveStatus === "error"
            ? t("saveFailed")
            : t("unsaved")}
    </div>
  );
}

function EditorButtons({ onSave, onDelete, saving, dirty }: EditorActions) {
  const t = useTranslations("aiPlanning");
  return (
    <div className="flex items-center justify-between border-t border-border pt-4">
      <Button variant="destructive" size="sm" onClick={onDelete}>
        <Trash2 className="h-4 w-4" />
        {t("deleteButton")}
      </Button>
      <Button size="sm" onClick={onSave} disabled={saving || !dirty}>
        <Save className="h-4 w-4" />
        {t("saveButton")}
      </Button>
    </div>
  );
}

function removeNthMarker(
  order: string[],
  marker: string,
  occurrence: number,
): string[] {
  let seen = 0;
  return order.filter((item) => {
    if (item !== marker) return true;
    const remove = seen === occurrence;
    seen += 1;
    return !remove;
  });
}

export function AiStrategyPlanEditor({
  value,
  onChange,
  ...actions
}: EditorActions & {
  value: AiStrategyPlanDraft;
  onChange: (value: AiStrategyPlanDraft) => void;
}) {
  const t = useTranslations("aiPlanning");
  const changeBlocks = (
    field: PlanBlockField,
    blocks: string[],
    order = value.field_order,
  ) => {
    onChange({ ...value, [field]: blocks, field_order: order });
  };
  return (
    <div className="space-y-6 p-6">
      <div>
        <h2 className="text-lg font-semibold tracking-tight">
          {t("planEditorTitle")}
        </h2>
        <p className="mt-1 text-xs text-muted-foreground">
          {t("fidelityDisclosure")}
        </p>
      </div>
      <div className="space-y-2">
        <Label htmlFor="ai-plan-id">{t("planIdLabel")}</Label>
        <Input
          id="ai-plan-id"
          className="font-mono"
          value={value.plan_id}
          onChange={(event) =>
            onChange({ ...value, plan_id: event.target.value })
          }
        />
      </div>
      <div className="space-y-2">
        <Label htmlFor="ai-plan-name">name</Label>
        <Input
          id="ai-plan-name"
          value={value.name}
          onChange={(event) => onChange({ ...value, name: event.target.value })}
        />
      </div>
      <div className="space-y-2">
        <Label htmlFor="ai-plan-desc">desc</Label>
        <Textarea
          id="ai-plan-desc"
          value={value.desc}
          onChange={(event) => onChange({ ...value, desc: event.target.value })}
        />
      </div>
      <div className="rounded-lg border border-border bg-card p-4 text-xs text-muted-foreground">
        {t("rawBlockHint")}
      </div>
      {PLAN_BLOCK_FIELDS.map((field) => (
        <section
          key={field}
          className="space-y-3 rounded-lg border border-border p-4"
        >
          <div className="flex items-center justify-between gap-3">
            <div>
              <h3 className="font-mono text-sm font-semibold">{field}</h3>
              <p className="mt-1 text-xs text-muted-foreground">
                {t(`planFieldHelp.${field}`)}
              </p>
            </div>
            <Button
              type="button"
              variant="outline"
              size="sm"
              onClick={() =>
                changeBlocks(
                  field,
                  [...value[field], "{\n}"],
                  [...value.field_order, field],
                )
              }
            >
              <Plus className="h-4 w-4" />
              {t("addBlock")}
            </Button>
          </div>
          {value[field].length === 0 ? (
            <p className="rounded-md border border-dashed border-border p-3 text-xs text-muted-foreground">
              {t("optionalBlockEmpty")}
            </p>
          ) : (
            value[field].map((block, index) => (
              <div key={`${field}-${index}`} className="space-y-2">
                <div className="flex items-center justify-between">
                  <Label htmlFor={`ai-plan-${field}-${index}`}>
                    {t("blockNumber", { count: index + 1 })}
                  </Label>
                  <Button
                    type="button"
                    variant="ghost"
                    size="sm"
                    onClick={() =>
                      changeBlocks(
                        field,
                        value[field].filter(
                          (_, itemIndex) => itemIndex !== index,
                        ),
                        removeNthMarker(value.field_order, field, index),
                      )
                    }
                  >
                    <Trash2 className="h-4 w-4" />
                    {t("removeBlock")}
                  </Button>
                </div>
                <Textarea
                  id={`ai-plan-${field}-${index}`}
                  className="min-h-28 font-mono text-xs"
                  value={block}
                  onChange={(event) =>
                    changeBlocks(
                      field,
                      value[field].map((item, itemIndex) =>
                        itemIndex === index ? event.target.value : item,
                      ),
                    )
                  }
                />
              </div>
            ))
          )}
        </section>
      ))}
      <SaveState {...actions} />
      <EditorButtons {...actions} />
    </div>
  );
}

export function AiTemplateEditor({
  value,
  onChange,
  ...actions
}: EditorActions & {
  value: AiTemplateDraft;
  onChange: (value: AiTemplateDraft) => void;
}) {
  const t = useTranslations("aiPlanning");
  const blockFields = [
    "blocked_for",
    "available_for",
    "upgrade_prio",
    "enable",
  ] as const;
  const renameTarget = (index: number, id: string) => {
    const previous = value.target_templates[index].id;
    onChange({
      ...value,
      target_templates: value.target_templates.map((item, itemIndex) =>
        itemIndex === index ? { ...item, id } : item,
      ),
      field_order: value.field_order.map((marker) =>
        marker === `target:${previous}` ? `target:${id}` : marker,
      ),
    });
  };
  return (
    <div className="space-y-6 p-6">
      <div>
        <h2 className="text-lg font-semibold tracking-tight">
          {t("templateEditorTitle")}
        </h2>
        <p className="mt-1 text-xs text-muted-foreground">
          {t("templateFidelityDisclosure")}
        </p>
      </div>
      <div className="grid grid-cols-2 gap-4">
        <div className="space-y-2">
          <Label htmlFor="ai-template-id">{t("templateIdLabel")}</Label>
          <Input
            id="ai-template-id"
            className="font-mono"
            value={value.template_id}
            onChange={(event) =>
              onChange({ ...value, template_id: event.target.value })
            }
          />
        </div>
        <div className="space-y-2">
          <Label htmlFor="ai-template-role">role</Label>
          <Input
            id="ai-template-role"
            className="font-mono"
            value={value.role}
            onChange={(event) =>
              onChange({ ...value, role: event.target.value })
            }
          />
        </div>
      </div>
      <div className="space-y-2">
        <Label htmlFor="ai-template-front-role">front_role_override</Label>
        <Input
          id="ai-template-front-role"
          className="font-mono"
          value={value.front_role_override}
          onChange={(event) =>
            onChange({ ...value, front_role_override: event.target.value })
          }
        />
        <p className="text-xs text-muted-foreground">{t("frontRoleHint")}</p>
      </div>
      {blockFields.map((field) => (
        <div key={field} className="space-y-2">
          <Label htmlFor={`ai-template-${field}`} className="font-mono">
            {field}
          </Label>
          <Textarea
            id={`ai-template-${field}`}
            className="min-h-24 font-mono text-xs"
            value={value[field]}
            placeholder="optional"
            onChange={(event) =>
              onChange({ ...value, [field]: event.target.value })
            }
          />
          <p className="text-xs text-muted-foreground">
            {t(`templateFieldHelp.${field}`)}
          </p>
        </div>
      ))}
      <section className="space-y-4 rounded-lg border border-border p-4">
        <div className="flex items-center justify-between gap-3">
          <div>
            <h3 className="text-sm font-semibold">
              {t("targetTemplatesTitle")}
            </h3>
            <p className="mt-1 text-xs text-muted-foreground">
              {t("targetOrderHint")}
            </p>
          </div>
          <Button
            type="button"
            variant="outline"
            size="sm"
            onClick={() => {
              let suffix = value.target_templates.length + 1;
              let id = `target_${suffix}`;
              while (value.target_templates.some((item) => item.id === id)) {
                suffix += 1;
                id = `target_${suffix}`;
              }
              onChange({
                ...value,
                target_templates: [
                  ...value.target_templates,
                  {
                    id,
                    body: "{\n\tupgrade_prio = {\n\t\tfactor = 1\n\t}\n\ttarget_template = {\n\t\tsupport = {\n\t\t}\n\t\tregiments = {\n\t\t}\n\t}\n}",
                  },
                ],
                field_order: [...value.field_order, `target:${id}`],
              });
            }}
          >
            <Plus className="h-4 w-4" />
            {t("addTarget")}
          </Button>
        </div>
        {value.target_templates.map((target, index) => (
          <div
            key={`${target.id}-${index}`}
            className="space-y-3 rounded-md border border-border bg-card p-3"
          >
            <div className="flex items-end gap-2">
              <div className="flex-1 space-y-2">
                <Label htmlFor={`ai-target-id-${index}`}>
                  {t("targetIdLabel", { count: index + 1 })}
                </Label>
                <Input
                  id={`ai-target-id-${index}`}
                  className="font-mono"
                  value={target.id}
                  onChange={(event) => renameTarget(index, event.target.value)}
                />
              </div>
              <Button
                type="button"
                variant="ghost"
                size="sm"
                disabled={value.target_templates.length === 1}
                onClick={() =>
                  onChange({
                    ...value,
                    target_templates: value.target_templates.filter(
                      (_, itemIndex) => itemIndex !== index,
                    ),
                    field_order: removeNthMarker(
                      value.field_order,
                      `target:${target.id}`,
                      0,
                    ),
                  })
                }
              >
                <Trash2 className="h-4 w-4" />
                {t("removeTarget")}
              </Button>
            </div>
            <Textarea
              aria-label={t("targetBodyAria", { id: target.id })}
              className="min-h-64 font-mono text-xs"
              value={target.body}
              onChange={(event) =>
                onChange({
                  ...value,
                  target_templates: value.target_templates.map(
                    (item, itemIndex) =>
                      itemIndex === index
                        ? { ...item, body: event.target.value }
                        : item,
                  ),
                })
              }
            />
          </div>
        ))}
      </section>
      <SaveState {...actions} />
      <EditorButtons {...actions} />
    </div>
  );
}
