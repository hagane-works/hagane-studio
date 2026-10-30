"use client";

import { useMemo } from "react";
import { useTranslations } from "use-intl";
import { Label } from "@/components/ui/label";
import { oobStarterScript } from "@/lib/oob/oobDefaults";

interface OobPassthroughPanelProps {
  value: string;

  onChange: (next: string) => void;

  countryTag?: string | null;

  oobFile?: string | null;

  startDate?: string | null;
}

const OOB_TOKENS: { key: string; test: (s: string) => boolean }[] = [
  { key: "set_oob", test: (s) => /(^|\n)\s*set_oob\b/.test(s) },
  { key: "set_air_oob", test: (s) => /(^|\n)\s*set_air_oob\b/.test(s) },
  { key: "set_naval_oob", test: (s) => /(^|\n)\s*set_naval_oob\b/.test(s) },
  { key: "units", test: (s) => /(^|\n)\s*units\s*=\s*\{/.test(s) },
  {
    key: "division_template",
    test: (s) => /(^|\n)\s*division_template\s*=\s*\{/.test(s),
  },
  {
    key: "equipment",
    test: (s) =>
      /\bequipment\s*=\s*\{/.test(s) ||
      /\badd_equipment_production\b/.test(s) ||
      /\binstant_effect\s*=\s*\{/.test(s),
  },
];

export function OobPassthroughPanel({
  value,
  onChange,
  countryTag,
  oobFile,
  startDate,
}: OobPassthroughPanelProps) {
  const t = useTranslations("oobRaw");

  const detected = useMemo(() => {
    const text = value || "";
    return OOB_TOKENS.filter((tok) => tok.test(text)).map((tok) => tok.key);
  }, [value]);

  const hasOob = detected.length > 0;
  const hasAnyRaw = (value || "").trim().length > 0;

  const insertStarter = () => {
    const starter = oobStarterScript({
      comment: t("starterComment"),
      unknownTagComment: t("starterNeedsTag"),
      countryTag,
      oobFile,
      startDate,
    });
    const base = (value || "").trimEnd();
    onChange(base ? `${base}\n\n${starter}\n` : `${starter}\n`);
  };

  return (
    <div className="ui-panel p-6 space-y-4">
      <div className="flex items-start justify-between gap-3">
        <div className="space-y-1">
          <h3 className="text-base font-semibold tracking-tight">
            {t("title")}
          </h3>
          <p className="text-xs text-muted-foreground leading-snug">
            {t("subtitle")}
          </p>
        </div>
        {hasOob && (
          <span className="shrink-0 rounded-full border border-success/40 bg-success/10 px-2 py-1 text-xs text-success">
            {t("preservedBadge")}
          </span>
        )}
      </div>

      {hasOob && (
        <div className="space-y-2">
          <p className="ui-ide-section-header text-micro font-semibold uppercase tracking-widest text-muted-foreground">
            {t("detectedHeading")}
          </p>
          <div className="flex flex-wrap gap-2">
            {detected.map((key) => (
              <span
                key={key}
                className="inline-flex items-center gap-1 rounded-md border border-border bg-muted/40 px-2 py-1 text-xs text-foreground"
              >
                <span
                  aria-hidden
                  className="h-1.5 w-1.5 rounded-full bg-success"
                />
                {t(`stat.${key}`)}
              </span>
            ))}
          </div>
        </div>
      )}

      {!hasOob && !hasAnyRaw ? (
        <div className="ui-card rounded-md border border-border/60 bg-muted/20 p-6 text-center space-y-3">
          <p className="text-sm font-medium text-foreground">
            {t("emptyTitle")}
          </p>
          <p className="text-xs text-muted-foreground leading-relaxed max-w-prose mx-auto">
            {t("emptyBody")}
          </p>
          <button
            type="button"
            onClick={insertStarter}
            className="ui-interactive ui-focus-ring inline-flex h-8 items-center rounded-md border border-border bg-transparent px-3 text-xs text-foreground hover:bg-muted"
          >
            {t("emptyInsert")}
          </button>
        </div>
      ) : (
        <div className="space-y-2">
          <Label className="text-sm font-medium">{t("editLabel")}</Label>
          <textarea
            value={value}
            onChange={(e) => onChange(e.target.value)}
            spellCheck={false}
            className="ui-focus-ring w-full min-h-40 rounded-md border border-input bg-background px-2 py-1 font-mono text-xs leading-relaxed"
          />
          <p className="text-xs text-muted-foreground leading-snug">
            {t("editHelp")}
          </p>
        </div>
      )}
    </div>
  );
}
