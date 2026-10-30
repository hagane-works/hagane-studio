"use client";

import { useMemo, useState } from "react";
import { useTranslations } from "use-intl";
import {
  ShieldAlert,
  ChevronDown,
  ChevronRight,
  ArrowUpRight,
} from "lucide-react";
import type { CtdIssue } from "@/lib/api/export";
import { useStateStore } from "@/lib/store/stateStore";
import {
  buildProvinceIndex,
  detectCollisions,
} from "@/lib/states/provinceIndex";
import {
  playerIssueCopy,
  type PlayerTranslator,
} from "@/components/workbench/mod-check/validationIssuePresentation";

const MAX_ROWS = 40;

function issueGeoStateId(issue: CtdIssue): number | null {
  const ref = issue.entity_ref;
  if (!ref) return null;
  if (ref.type === "state") {
    if (typeof ref.id === "number") return ref.id;
    if (typeof ref.id === "string" && /^\d+$/.test(ref.id))
      return Number(ref.id);
  }
  const sid = (ref as { state_id?: unknown }).state_id;
  if (typeof sid === "number") return sid;
  if (typeof sid === "string" && /^\d+$/.test(sid)) return Number(sid);
  return null;
}

export interface MapHealthLightProps {
  projectId?: number;

  onSelectStateGeoId?: (hoi4StateId: number) => void;
}

export function buildProvinceCollisionIssues(
  states: readonly {
    id: number;
    state_id: number;
    name?: string | null;
    provinces?: number[] | null;
  }[],
): CtdIssue[] {
  return detectCollisions(buildProvinceIndex(states)).map((collision) => {
    const stateIds = collision.owners.map((o) => o.stateId);
    return {
      code: "province_double_mapping",
      severity: "blocker" as const,
      entity_ref: {
        type: "state",
        id: stateIds[0],
        province_id: collision.provinceId,
        state_ids: stateIds,
      },
      message:
        `Province ${collision.provinceId} is assigned to ${stateIds.length} ` +
        `states (${stateIds.join(", ")}) — HOI4 will crash on load.`,
      fix_hint: `Remove province ${collision.provinceId} from every state except one.`,
    };
  });
}

export function MapHealthLight({ onSelectStateGeoId }: MapHealthLightProps) {
  const t = useTranslations("states");
  const tModCheck = useTranslations("modCheck") as unknown as PlayerTranslator;

  const states = useStateStore((s) => s.states);
  const [open, setOpen] = useState(false);

  const issues = useMemo(() => buildProvinceCollisionIssues(states), [states]);

  if (issues.length === 0) return null;

  const shown = issues.slice(0, MAX_ROWS);
  const hidden = issues.length - shown.length;

  return (
    <div className="flex flex-col items-end gap-1.5">
      <button
        type="button"
        onClick={() => setOpen((o) => !o)}
        aria-expanded={open}
        className="inline-flex items-center gap-1.5 rounded-full border border-destructive/40 bg-destructive/10 px-2.5 py-1 text-xs font-medium text-destructive shadow-sm backdrop-blur ui-interactive"
        title={t("mapHealthTitle")}
      >
        <ShieldAlert className="h-3.5 w-3.5" aria-hidden />
        <span>{t("mapHealthSerious", { count: issues.length })}</span>
        {open ? (
          <ChevronDown className="h-3 w-3" aria-hidden />
        ) : (
          <ChevronRight className="h-3 w-3" aria-hidden />
        )}
      </button>

      {open && (
        <div className="w-72 max-h-72 overflow-y-auto rounded-lg border border-border/60 bg-popover/95 p-2 shadow-md backdrop-blur space-y-1.5">
          {shown.map((issue, i) => {
            const geoId = issueGeoStateId(issue);
            const copy = playerIssueCopy(tModCheck, issue);
            return (
              <div
                key={`${issue.code}-${i}`}
                className="rounded-md border border-destructive/40 bg-destructive/5 p-2 text-xs"
              >
                <div className="flex items-start gap-1.5">
                  <ShieldAlert
                    className="h-3.5 w-3.5 shrink-0 mt-0.5 text-destructive"
                    aria-hidden
                  />
                  <div className="min-w-0 flex-1 space-y-1">
                    <p className="text-foreground">{copy.message}</p>
                    <p className="text-muted-foreground">
                      <span className="font-medium text-foreground/80">
                        {tModCheck("impactLabel")}:{" "}
                      </span>
                      {copy.impact}
                    </p>
                    <p className="text-muted-foreground">
                      <span className="font-medium text-foreground/80">
                        {tModCheck("fixHintLabel")}:{" "}
                      </span>
                      {copy.hint}
                    </p>
                  </div>
                </div>
                {geoId != null && onSelectStateGeoId && (
                  <button
                    type="button"
                    onClick={() => onSelectStateGeoId(geoId)}
                    className="mt-1 inline-flex items-center gap-1 pl-5 text-accent hover:underline ui-interactive"
                  >
                    <ArrowUpRight className="h-3 w-3 shrink-0" aria-hidden />
                    {t("mapHealthGoToState", { id: geoId })}
                  </button>
                )}
              </div>
            );
          })}
          {hidden > 0 && (
            <p className="px-1 text-micro italic text-muted-foreground">
              {t("mapHealthMore", { count: hidden })}
            </p>
          )}
        </div>
      )}
    </div>
  );
}
