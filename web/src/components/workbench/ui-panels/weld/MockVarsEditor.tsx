"use client";

import { Input } from "@/components/ui/input";
import { useMemo } from "react";
import { useTranslations } from "use-intl";

import type { UiPanelControl } from "@/types";
import {
  collectReferencedVars,
  useMockVarsStore,
} from "@/components/workbench/ui-panels/mockVars";

export function LivePreviewToggle() {
  const t = useTranslations("uiPanelsWeld");
  const enabled = useMockVarsStore((s) => s.enabled);
  const setEnabled = useMockVarsStore((s) => s.setEnabled);
  return (
    <button
      type="button"
      role="switch"
      aria-checked={enabled}
      onClick={() => setEnabled(!enabled)}
      className="inline-flex items-center gap-2 h-7 px-2.5 rounded-md border border-border bg-card text-xs ui-interactive ui-focus-ring"
    >
      <span
        className={`relative w-7 h-4 rounded-full transition-colors ${
          enabled ? "bg-accent" : "bg-muted"
        }`}
        aria-hidden
      >
        <span
          className={`absolute top-0.5 w-3 h-3 rounded-full transition-transform ${
            enabled
              ? "translate-x-3.5 bg-accent-foreground"
              : "translate-x-0.5 bg-muted-foreground"
          }`}
        />
      </span>
      <span className={enabled ? "text-foreground" : "text-muted-foreground"}>
        {t("livePreview")}
      </span>
    </button>
  );
}

export function MockVarsTable({ controls }: { controls: UiPanelControl[] }) {
  const t = useTranslations("uiPanelsWeld");
  const vars = useMockVarsStore((s) => s.vars);
  const setVar = useMockVarsStore((s) => s.setVar);
  const names = useMemo(() => collectReferencedVars(controls), [controls]);

  return (
    <div className="rounded-md border border-border bg-card p-2.5 space-y-1.5">
      <div className="flex items-baseline gap-2">
        <span className="text-micro uppercase tracking-widest font-semibold text-muted-foreground">
          {t("mockVarsTitle")}
        </span>
        <span className="text-micro text-muted-foreground">
          {t("mockVarsHint")}
        </span>
      </div>
      {names.length === 0 ? (
        <p className="text-xs text-muted-foreground">{t("mockVarsEmpty")}</p>
      ) : (
        <div className="flex flex-wrap gap-x-4 gap-y-1.5">
          {names.map((name) => (
            <label key={name} className="flex items-center gap-1.5 text-xs">
              <span className="font-mono text-muted-foreground">{name}</span>
              <Input
                type="number"
                value={vars[name] ?? 0}
                onChange={(e) => setVar(name, Number(e.target.value) || 0)}
                className="h-6 w-20 px-1.5 text-xs font-mono ui-focus-ring"
                aria-label={t("mockVarAria", { name })}
              />
            </label>
          ))}
        </div>
      )}
    </div>
  );
}
