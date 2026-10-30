"use client";

import { useState } from "react";
import { useTranslations } from "use-intl";
import { ChevronDown, ChevronRight } from "lucide-react";

import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import type {
  LegacyMapAggregate,
  PresentedFinding,
} from "@/lib/exportCheckPresentation";
import { useFindingCopy } from "@/components/export/FindingCopy";

function TierChip({ kind }: { kind: "auto" | "health" | "pass" }) {
  const t = useTranslations("export");
  const label =
    kind === "auto"
      ? `ℹ️ ${t("ui.dialogCheck.autoTier")}`
      : kind === "health"
        ? `🩺 ${t("ui.dialogCheck.healthTier")}`
        : `✅ ${t("ui.dialogCheck.passTier")}`;
  const tone =
    kind === "pass"
      ? "bg-success/15 text-success"
      : kind === "health"
        ? "border border-dashed border-border bg-muted text-muted-foreground"
        : "bg-muted text-muted-foreground";
  return (
    <span
      className={`inline-flex shrink-0 items-center gap-1 self-start rounded-full px-2.5 py-0.5 text-micro font-semibold ${tone}`}
    >
      {label}
    </span>
  );
}

function FindingRow({
  finding,
  chip,
}: {
  finding: PresentedFinding;
  chip: "auto" | "health";
}) {
  const copyOf = useFindingCopy();
  const copy = copyOf(finding);
  return (
    <div className="flex items-start justify-between gap-3 border-b border-border py-3 last:border-b-0">
      <div className="min-w-0">
        <p className="text-sm font-medium text-foreground">{copy.what}</p>
        {copy.impact ? (
          <p className="mt-0.5 text-xs text-muted-foreground">{copy.impact}</p>
        ) : null}
      </div>
      <TierChip kind={chip} />
    </div>
  );
}

export function CheckDetailsDialog({
  open,
  onOpenChange,
  auto,
  health,
  pass,
  legacyMap,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  auto: PresentedFinding[];
  health: PresentedFinding[];
  pass: PresentedFinding[];
  legacyMap: LegacyMapAggregate | null;
}) {
  const t = useTranslations("export");
  const copyOf = useFindingCopy();
  const [passOpen, setPassOpen] = useState(false);

  const subtitle =
    health.length > 0
      ? t("ui.entries.checkSubHealth", {
          auto: auto.length,
          pass: pass.length,
          health: health.length,
        })
      : t("ui.entries.checkSub", { auto: auto.length, pass: pass.length });

  const empty =
    auto.length === 0 &&
    health.length === 0 &&
    pass.length === 0 &&
    legacyMap === null;

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-xl">
        <DialogHeader>
          <DialogTitle>{t("ui.dialogCheck.title")}</DialogTitle>
          <DialogDescription>{subtitle}</DialogDescription>
        </DialogHeader>

        <div className="max-h-[56vh] overflow-y-auto pr-1">
          {empty ? (
            <p className="py-3 text-sm text-muted-foreground">
              {t("ui.dialogCheck.empty")}
            </p>
          ) : (
            <>
              {auto.map((finding, i) => (
                <FindingRow key={`auto-${i}`} finding={finding} chip="auto" />
              ))}

              {legacyMap ? (
                <div
                  data-testid="legacy-map-aggregate"
                  className="flex items-start justify-between gap-3 border-b border-border py-3 last:border-b-0"
                >
                  <p className="min-w-0 text-sm text-foreground">
                    {t("ui.dialogCheck.legacyMapAggregate", {
                      count: legacyMap.count,
                    })}
                  </p>
                  <TierChip kind="auto" />
                </div>
              ) : null}

              {health.map((finding, i) => (
                <FindingRow
                  key={`health-${i}`}
                  finding={finding}
                  chip="health"
                />
              ))}

              {pass.length > 0 ? (
                <div className="py-3">
                  <button
                    type="button"
                    onClick={() => setPassOpen((v) => !v)}
                    aria-expanded={passOpen}
                    className="flex w-full items-center justify-between gap-3 text-left ui-interactive ui-focus-ring rounded"
                  >
                    <span className="text-sm font-medium text-foreground">
                      {t("ui.dialogCheck.passLabel", { count: pass.length })}
                    </span>
                    <span className="inline-flex items-center gap-1">
                      <TierChip kind="pass" />
                      {passOpen ? (
                        <ChevronDown
                          className="h-3.5 w-3.5 text-muted-foreground"
                          aria-hidden
                        />
                      ) : (
                        <ChevronRight
                          className="h-3.5 w-3.5 text-muted-foreground"
                          aria-hidden
                        />
                      )}
                    </span>
                  </button>
                  {passOpen ? (
                    <ul className="mt-2 space-y-0.5">
                      {pass.map((finding, i) => (
                        <li
                          key={`pass-${i}`}
                          className="border-b border-dashed border-border py-1.5 text-xs text-muted-foreground last:border-b-0"
                        >
                          {copyOf(finding).what}
                        </li>
                      ))}
                    </ul>
                  ) : null}
                </div>
              ) : null}
            </>
          )}
        </div>

        <p className="text-micro text-muted-foreground">
          {t("ui.dialogCheck.scopeNote")}
        </p>

        <DialogFooter>
          <Button
            variant="outline"
            size="sm"
            onClick={() => onOpenChange(false)}
          >
            {t("ui.dialogCheck.close")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
