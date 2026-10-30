"use client";

import { useId } from "react";
import { useTranslations } from "use-intl";

import { Button } from "@/components/ui/button";
import {
  INTEGRATED_DLCS,
  SELECTABLE_DLCS,
  normalizeDlcChoice,
} from "@/lib/dlcCatalog";

interface DlcPickerProps {
  value: string[] | null;
  onChange: (next: string[] | null) => void;
  disabled?: boolean;
}

export function DlcPicker({ value, onChange, disabled }: DlcPickerProps) {
  const t = useTranslations("dlcPicker");
  const titleId = useId();
  const owned = new Set(value ?? SELECTABLE_DLCS);
  const count = SELECTABLE_DLCS.filter((name) => owned.has(name)).length;
  const summary =
    value == null || count === SELECTABLE_DLCS.length
      ? t("summaryAll")
      : count === 0
        ? t("summaryNone")
        : t("summarySome", { count, total: SELECTABLE_DLCS.length });

  const toggle = (name: string) => {
    const next = new Set(owned);
    if (next.has(name)) next.delete(name);
    else next.add(name);
    onChange(
      next.size === SELECTABLE_DLCS.length
        ? null
        : normalizeDlcChoice([...next]),
    );
  };

  return (
    <div className="space-y-2" role="group" aria-labelledby={titleId}>
      <div className="flex items-center justify-between gap-2">
        <div className="flex items-baseline gap-2">
          <span id={titleId} className="text-sm font-medium">
            {t("title")}
          </span>
          <span className="text-xs text-muted-foreground">{summary}</span>
        </div>
        <div className="flex gap-1">
          <Button
            type="button"
            variant="ghost"
            size="sm"
            disabled={disabled}
            onClick={() => onChange(null)}
          >
            {t("selectAll")}
          </Button>
          <Button
            type="button"
            variant="ghost"
            size="sm"
            disabled={disabled}
            onClick={() => onChange([])}
          >
            {t("selectNone")}
          </Button>
        </div>
      </div>
      <div className="grid grid-cols-2 gap-1.5">
        {SELECTABLE_DLCS.map((name) => {
          const checked = owned.has(name);
          return (
            <label
              key={name}
              className={
                "flex items-center gap-1.5 rounded-md border px-2 py-1 text-xs cursor-pointer ui-interactive " +
                (checked
                  ? "border-primary bg-primary/10 text-foreground"
                  : "border-input bg-background hover:bg-muted text-muted-foreground") +
                (disabled ? " opacity-80 cursor-default" : "")
              }
            >
              <input
                type="checkbox"
                checked={checked}
                disabled={disabled}
                onChange={() => toggle(name)}
                className="accent-accent h-3.5 w-3.5"
              />
              <span className="truncate" title={name}>
                {name}
              </span>
            </label>
          );
        })}
      </div>
      <p className="text-xs text-muted-foreground">
        {t("integrated", { names: INTEGRATED_DLCS.join(", ") })}
      </p>
      <p className="text-xs text-muted-foreground">{t("hint")}</p>
    </div>
  );
}
