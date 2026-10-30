"use client";

import { useMemo, useState } from "react";
import { useTranslations } from "use-intl";
import { Plus, X } from "lucide-react";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { subunitOptionLabel } from "@/lib/i18n/subunitGloss";
import { useVanillaUnitNames } from "@/lib/vanillaUnitNames";
import type { DivisionRegiment } from "@/types";
import type { DivisionSubunitOption } from "./divisionSubunitOptions";
import {
  GRID_COLS,
  GRID_ROWS,
  SUPPORT_SLOTS,
  buildRegimentGrid,
  buildSupportColumn,
} from "./divisionGridModel";

type PickerTarget =
  | { kind: "regiments"; index: number | null; x: number; y: number }
  | { kind: "support"; index: number | null };

export function DivisionDesignerGrid({
  regiments,
  support,
  subunits,
  regimentTokens,
  supportTokens,
  locale,
  onChangeRegiments,
  onChangeSupport,
}: {
  regiments: DivisionRegiment[];
  support: DivisionRegiment[];
  subunits: Record<string, DivisionSubunitOption>;
  regimentTokens: string[];
  supportTokens: string[];
  locale: Parameters<typeof subunitOptionLabel>[1];
  onChangeRegiments: (rows: DivisionRegiment[]) => void;
  onChangeSupport: (rows: DivisionRegiment[]) => void;
}) {
  const t = useTranslations("divisionTemplate");
  const unitNames = useVanillaUnitNames();
  const [target, setTarget] = useState<PickerTarget | null>(null);
  const [search, setSearch] = useState("");

  const grid = useMemo(() => buildRegimentGrid(regiments), [regiments]);
  const column = useMemo(() => buildSupportColumn(support), [support]);

  const pickerTokens = useMemo(() => {
    if (!target) return [];
    const pool = target.kind === "regiments" ? regimentTokens : supportTokens;
    const needle = search.trim().toLowerCase();
    if (!needle) return pool;
    return pool.filter((token) =>
      `${token} ${subunitOptionLabel(token, locale, unitNames.subunits)}`
        .toLowerCase()
        .includes(needle),
    );
  }, [target, regimentTokens, supportTokens, search, locale, unitNames]);

  const openPicker = (next: PickerTarget) => {
    setSearch("");
    setTarget(next);
  };

  const applyToken = (token: string) => {
    if (!target) return;
    if (target.kind === "regiments") {
      if (target.index === null) {
        onChangeRegiments([...regiments, { token, x: target.x, y: target.y }]);
      } else {
        onChangeRegiments(
          regiments.map((row, i) =>
            i === target.index ? { ...row, token } : row,
          ),
        );
      }
    } else if (target.index === null) {
      onChangeSupport([...support, { token, x: 0, y: support.length }]);
    } else {
      onChangeSupport(
        support.map((row, i) => (i === target.index ? { ...row, token } : row)),
      );
    }
    setTarget(null);
  };

  const removeCurrent = () => {
    if (!target || target.index === null) return;
    if (target.kind === "regiments") {
      onChangeRegiments(regiments.filter((_, i) => i !== target.index));
    } else {
      onChangeSupport(support.filter((_, i) => i !== target.index));
    }
    setTarget(null);
  };

  const cellText = (token: string) =>
    (subunits[token]?.abbreviation || token.slice(0, 3)).toUpperCase();

  const filledCell = (
    kind: "regiments" | "support",
    index: number,
    token: string,
    onClick: () => void,
  ) => (
    <button
      key={`${kind}-${index}`}
      type="button"
      onClick={onClick}
      title={subunitOptionLabel(token, locale, unitNames.subunits)}
      className="ui-interactive relative flex h-14 w-full flex-col items-center justify-center rounded-[4px] border border-ingame-edge/70 bg-ingame-band/85 text-ingame-title hover:ring-1 hover:ring-accent/70"
    >
      <span className="font-mono text-sm font-semibold text-ingame-gold">
        {cellText(token)}
      </span>
      <span className="text-micro max-w-full truncate px-1 text-ingame-text/80">
        {token}
      </span>
    </button>
  );

  const emptyCell = (key: string, label: string, onClick: () => void) => (
    <button
      key={key}
      type="button"
      onClick={onClick}
      title={label}
      aria-label={label}
      className="ui-interactive flex h-14 w-full items-center justify-center rounded-[4px] border border-dashed border-ingame-edge/45 text-ingame-text/50 hover:border-accent/60 hover:text-accent"
    >
      <Plus className="h-4 w-4" />
    </button>
  );

  return (
    <section className="rounded-md border border-ingame-edge/60 bg-gradient-to-b from-ingame-panel/90 to-ingame-panel-deep/95 p-3 shadow-sm">
      <div className="flex items-start gap-4">
        <div className="grid flex-1 grid-cols-5 gap-1.5">
          {Array.from({ length: GRID_COLS }, (_, x) => (
            <div key={x} className="space-y-1.5">
              {Array.from({ length: GRID_ROWS }, (_, y) => {
                const index = grid.cells[x][y];
                return index !== null
                  ? filledCell("regiments", index, regiments[index].token, () =>
                      openPicker({ kind: "regiments", index, x, y }),
                    )
                  : emptyCell(`empty-${x}-${y}`, t("emptyCellAdd"), () =>
                      openPicker({ kind: "regiments", index: null, x, y }),
                    );
              })}
            </div>
          ))}
        </div>
        <div className="w-24 shrink-0 space-y-1.5 border-l border-ingame-edge/40 pl-3">
          <div className="text-micro font-semibold uppercase tracking-widest text-ingame-gold/90">
            {t("gridSupportColumn")}
          </div>
          {Array.from({ length: SUPPORT_SLOTS }, (_, slot) => {
            const index = column.slots[slot];
            return index !== null
              ? filledCell("support", index, support[index].token, () =>
                  openPicker({ kind: "support", index }),
                )
              : emptyCell(`support-${slot}`, t("emptySupportAdd"), () =>
                  openPicker({ kind: "support", index: null }),
                );
          })}
        </div>
      </div>

      <Dialog
        open={Boolean(target)}
        onOpenChange={(open) => {
          if (!open) setTarget(null);
        }}
      >
        <DialogContent className="grid max-h-[min(640px,calc(100vh-2rem))] max-w-md grid-rows-[auto_auto_minmax(0,1fr)_auto] gap-0 overflow-hidden p-0">
          <DialogHeader className="border-b px-4 py-3">
            <DialogTitle className="text-base">
              {target?.kind === "support"
                ? t("pickerSupportTitle")
                : t("pickerRegimentTitle")}
            </DialogTitle>
          </DialogHeader>
          <div className="border-b p-2">
            <Input
              autoFocus
              value={search}
              onChange={(event) => setSearch(event.target.value)}
              placeholder={t("pickerSearch")}
            />
          </div>
          <div className="min-h-0 overflow-y-auto p-1.5">
            {pickerTokens.length ? (
              <div className="space-y-0.5">
                {pickerTokens.map((token) => (
                  <button
                    key={token}
                    type="button"
                    onClick={() => applyToken(token)}
                    className="ui-interactive flex w-full items-center gap-2 rounded px-2.5 py-1.5 text-left text-sm hover:bg-muted"
                  >
                    <span className="w-10 shrink-0 font-mono text-xs text-muted-foreground">
                      {cellText(token)}
                    </span>
                    <span className="min-w-0 flex-1 truncate">
                      {subunitOptionLabel(token, locale, unitNames.subunits)}
                    </span>
                    {subunits[token]?.source === "project" && (
                      <span className="shrink-0 text-micro text-muted-foreground">
                        {t("projectSubunit")}
                      </span>
                    )}
                  </button>
                ))}
              </div>
            ) : (
              <p className="p-6 text-center text-xs text-muted-foreground">
                {t("pickerEmpty")}
              </p>
            )}
          </div>
          {target?.index !== null && target !== null && (
            <div className="border-t p-2">
              <Button
                type="button"
                variant="destructive"
                size="sm"
                className="w-full"
                onClick={removeCurrent}
              >
                <X className="h-4 w-4" />
                {target.kind === "support"
                  ? t("pickerRemoveSupport")
                  : t("pickerRemove")}
              </Button>
            </div>
          )}
        </DialogContent>
      </Dialog>
    </section>
  );
}
