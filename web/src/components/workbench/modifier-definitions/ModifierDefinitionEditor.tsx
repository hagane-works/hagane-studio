"use client";

import { useTranslations } from "use-intl";
import { Save, Trash2 } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  MODIFIER_CATEGORIES,
  MODIFIER_COLOR_TYPES,
  MODIFIER_POSTFIXES,
  MODIFIER_VALUE_TYPES,
  type ModifierCategory,
  type ModifierColorType,
  type ModifierDefinitionDraft,
  type ModifierPostfix,
  type ModifierValueType,
} from "@/lib/api/modifierDefinitions";

type SaveStatus = "idle" | "success" | "error";

interface ModifierDefinitionEditorProps {
  value: ModifierDefinitionDraft;
  onChange: (value: ModifierDefinitionDraft) => void;
  onSave: () => void;
  onDelete: () => void;
  saving?: boolean;
  dirty?: boolean;
  saveStatus?: SaveStatus;
}

export function ModifierDefinitionEditor({
  value,
  onChange,
  onSave,
  onDelete,
  saving = false,
  dirty = false,
  saveStatus = "idle",
}: ModifierDefinitionEditorProps) {
  const t = useTranslations("modifierDefinitions");

  const toggleCategory = (category: ModifierCategory) => {
    const selected = value.categories.includes(category);
    if (selected && value.categories.length === 1) return;
    onChange({
      ...value,
      categories: selected
        ? value.categories.filter((item) => item !== category)
        : [...value.categories, category],
    });
  };

  return (
    <div className="space-y-6 p-6">
      <div>
        <h2 className="text-lg font-semibold tracking-tight">
          {t("editorTitle")}
        </h2>
        <p className="mt-1 text-xs text-muted-foreground">
          {t("editorDescription")}
        </p>
      </div>

      <div className="space-y-2">
        <Label htmlFor="modifier-definition-id">{t("modifierIdLabel")}</Label>
        <Input
          id="modifier-definition-id"
          className="font-mono"
          value={value.modifier_id}
          onChange={(event) =>
            onChange({ ...value, modifier_id: event.target.value })
          }
          placeholder="custom_readiness"
        />
        <p className="text-xs text-muted-foreground">{t("modifierIdHint")}</p>
      </div>

      <div className="grid grid-cols-2 gap-4">
        <div className="space-y-2">
          <Label htmlFor="modifier-definition-color">
            {t("colorTypeLabel")}
          </Label>
          <select
            id="modifier-definition-color"
            className="ui-focus-ring h-9 w-full rounded-md border border-input bg-card px-3 text-sm"
            value={value.color_type}
            onChange={(event) =>
              onChange({
                ...value,
                color_type: event.target.value as ModifierColorType,
              })
            }
          >
            {MODIFIER_COLOR_TYPES.map((item) => (
              <option key={item} value={item}>
                {t(`colorOptions.${item}`)}
              </option>
            ))}
          </select>
          <p className="text-xs text-muted-foreground">{t("colorTypeHint")}</p>
        </div>

        <div className="space-y-2">
          <Label htmlFor="modifier-definition-value">
            {t("valueTypeLabel")}
          </Label>
          <select
            id="modifier-definition-value"
            className="ui-focus-ring h-9 w-full rounded-md border border-input bg-card px-3 text-sm"
            value={value.value_type}
            onChange={(event) =>
              onChange({
                ...value,
                value_type: event.target.value as ModifierValueType,
              })
            }
          >
            {MODIFIER_VALUE_TYPES.map((item) => (
              <option key={item} value={item}>
                {t(`valueOptions.${item}`)}
              </option>
            ))}
          </select>
          <p className="text-xs text-muted-foreground">{t("valueTypeHint")}</p>
        </div>

        <div className="space-y-2">
          <Label htmlFor="modifier-definition-precision">
            {t("precisionLabel")}
          </Label>
          <Input
            id="modifier-definition-precision"
            type="number"
            min={0}
            max={3}
            step={1}
            value={value.precision}
            onChange={(event) => {
              const next = Number(event.target.value);
              onChange({ ...value, precision: Math.min(3, Math.max(0, next)) });
            }}
          />
          <p className="text-xs text-muted-foreground">{t("precisionHint")}</p>
        </div>

        <div className="space-y-2">
          <Label htmlFor="modifier-definition-postfix">
            {t("postfixLabel")}
          </Label>
          <select
            id="modifier-definition-postfix"
            className="ui-focus-ring h-9 w-full rounded-md border border-input bg-card px-3 text-sm"
            value={value.postfix}
            onChange={(event) =>
              onChange({
                ...value,
                postfix: event.target.value as ModifierPostfix,
              })
            }
          >
            {MODIFIER_POSTFIXES.map((item) => (
              <option key={item} value={item}>
                {t(`postfixOptions.${item}`)}
              </option>
            ))}
          </select>
          <p className="text-xs text-muted-foreground">{t("postfixHint")}</p>
        </div>
      </div>

      <fieldset className="space-y-3">
        <legend className="text-sm font-medium">{t("categoriesLabel")}</legend>
        <p className="text-xs text-muted-foreground">{t("categoriesHint")}</p>
        <div className="grid grid-cols-2 gap-2 rounded-lg border border-border bg-card p-4">
          {MODIFIER_CATEGORIES.map((category) => {
            const checked = value.categories.includes(category);
            const lastSelected = checked && value.categories.length === 1;
            return (
              <label
                key={category}
                className="ui-interactive flex min-h-8 items-center gap-2 rounded-md px-2 text-xs hover:bg-muted"
              >
                <input
                  type="checkbox"
                  checked={checked}
                  disabled={lastSelected}
                  aria-label={t(`categoryOptions.${category}`)}
                  onChange={() => toggleCategory(category)}
                />
                <span>{t(`categoryOptions.${category}`)}</span>
                <code className="text-muted-foreground">{category}</code>
              </label>
            );
          })}
        </div>
      </fieldset>

      {(saving || dirty || saveStatus !== "idle") && (
        <div
          className={`border-y px-3 py-2 text-xs font-medium transition-[background-color,border-color,color] duration-300 ${
            saving
              ? "border-border bg-muted text-muted-foreground"
              : saveStatus === "success"
                ? "border-success/20 bg-success/10 text-success"
                : saveStatus === "error"
                  ? "border-destructive/20 bg-destructive/10 text-destructive"
                  : "border-warning/20 bg-warning/10 text-warning"
          }`}
        >
          {saving
            ? t("saving")
            : saveStatus === "success"
              ? t("saved")
              : saveStatus === "error"
                ? t("saveFailed")
                : t("unsaved")}
        </div>
      )}

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
    </div>
  );
}
