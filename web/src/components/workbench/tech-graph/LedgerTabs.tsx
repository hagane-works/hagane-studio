"use client";

import { useMemo } from "react";
import { useTranslations } from "use-intl";
import type { Technology } from "@/types";
import {
  LEDGER_ORDER,
  LEDGER_LABEL_I18N_KEYS,
  resolveLedger,
  resolveDisplayTechLine,
  DLC_TECH_LINE_TO_BASE,
  type LedgerKey,
} from "@/lib/techGraph";

export interface TechLineOption {
  value: string;
  label: string;
  folder: string;
}

interface LedgerTabsProps {
  techLines: TechLineOption[];
  techs: Technology[];
  activeTechLine: string;
  onSelectTechLine: (value: string) => void;
}

export function LedgerTabs({
  techLines,
  techs,
  activeTechLine,
  onSelectTechLine,
}: LedgerTabsProps) {
  const t = useTranslations("ledgerTabs");
  const tTech = useTranslations("tech");
  const activeLedger = resolveLedger(activeTechLine);

  const displayTechLines = useMemo(
    () => techLines.filter((line) => !(line.value in DLC_TECH_LINE_TO_BASE)),
    [techLines],
  );

  const linesByLedger = useMemo(() => {
    const map: Record<LedgerKey, TechLineOption[]> = {
      army: [],
      air: [],
      navy: [],
      civilian: [],
      special: [],
    };
    for (const line of displayTechLines) {
      const ledger = resolveLedger(line.value);
      map[ledger].push(line);
    }
    return map;
  }, [displayTechLines]);

  const countByLine = useMemo(() => {
    const map: Record<string, number> = {};
    for (const tech of techs) {
      const display = resolveDisplayTechLine(tech.tech_line);
      map[display] = (map[display] ?? 0) + 1;
    }
    return map;
  }, [techs]);

  const countByLedger = useMemo(() => {
    const map: Record<LedgerKey, number> = {
      army: 0,
      air: 0,
      navy: 0,
      civilian: 0,
      special: 0,
    };
    for (const tech of techs) {
      const ledger = resolveLedger(tech.tech_line);
      map[ledger] += 1;
    }
    return map;
  }, [techs]);

  const activeSubLines = linesByLedger[activeLedger] ?? [];

  const handleLedgerClick = (ledger: LedgerKey) => {
    const lines = linesByLedger[ledger];
    if (lines.length === 0) return;
    const firstWithTechs = lines.find((l) => (countByLine[l.value] ?? 0) > 0);
    onSelectTechLine(firstWithTechs?.value ?? lines[0].value);
  };

  return (
    <div className="border-b border-border bg-card flex flex-col shrink-0">
      <div className="flex items-center gap-1 px-3 py-2 border-b border-border/40 overflow-x-auto">
        {LEDGER_ORDER.map((ledger) => {
          const subCount = linesByLedger[ledger].length;
          if (subCount === 0) return null;
          const isActive = activeLedger === ledger;
          const count = countByLedger[ledger];
          return (
            <button
              key={ledger}
              onClick={() => handleLedgerClick(ledger)}
              className={`px-3 py-1.5 rounded-md text-sm font-medium transition-colors whitespace-nowrap ${
                isActive
                  ? "bg-primary text-primary-foreground shadow-sm"
                  : "bg-transparent text-muted-foreground hover:bg-muted hover:text-foreground"
              }`}
              title={`${tTech(LEDGER_LABEL_I18N_KEYS[ledger])} · ${count} tech in ${subCount} folder${subCount > 1 ? "s" : ""}`}
            >
              <span>{tTech(LEDGER_LABEL_I18N_KEYS[ledger])}</span>
              {count > 0 && (
                <span
                  className={`ml-1.5 text-micro px-1 py-0.5 rounded font-mono ${
                    isActive ? "bg-primary-foreground/20" : "bg-muted/60"
                  }`}
                >
                  {count}
                </span>
              )}
            </button>
          );
        })}
      </div>

      <div className="flex items-center gap-0.5 px-3 py-1.5 overflow-x-auto">
        {activeSubLines.map((line) => {
          const isActive = activeTechLine === line.value;
          const count = countByLine[line.value] ?? 0;
          const isEmpty = count === 0;
          return (
            <button
              key={line.value}
              onClick={() => onSelectTechLine(line.value)}
              className={`px-2.5 py-1 rounded text-xs whitespace-nowrap transition-colors ${
                isActive
                  ? "bg-primary/15 text-primary border border-primary/40"
                  : isEmpty
                    ? "text-muted-foreground/50 hover:text-muted-foreground hover:bg-muted/30 border border-transparent"
                    : "text-foreground hover:bg-muted/50 border border-transparent"
              }`}
              title={
                isEmpty
                  ? t("emptyTitle", { label: line.label })
                  : `${line.label} · ${count} tech`
              }
            >
              {line.label}
              {!isEmpty && (
                <span className="ml-1 text-micro opacity-60 font-mono">
                  {count}
                </span>
              )}
            </button>
          );
        })}
      </div>
    </div>
  );
}
