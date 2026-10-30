"use client";

import { useTranslations } from "use-intl";
import { AlertTriangle, Check, Info } from "lucide-react";

import type { WeldRow, WeldRowKind } from "./weldModel";

const KIND_LABEL_KEY: Record<WeldRowKind, string> = {
  window: "catWindow",
  panelVisible: "catPanelVisible",
  visible: "catVisible",
  loc: "catLoc",
  sprite: "catSprite",
  raw: "catRaw",
  click: "catClick",
  clickEnabled: "catClickEnabled",
  tooltip: "catTooltip",
};

const STATIC_CODES = new Set([
  "panel.token_collision",
  "panel.control_collision",
  "panel.raw_dirty_invalid",
  "panel.unknown_scripted",
  "panel.sprite_empty",
  "panel.loc_blank",
]);

function codeKey(code: string): string {
  return `code_${code.slice("panel.".length)}`;
}

export function WeldTargetRow({
  row,
  onFix,
}: {
  row: WeldRow;

  onFix?: (row: WeldRow) => void;
}) {
  const t = useTranslations("uiPanelsWeld");

  const infoOnly = row.worst === null && row.findings.length > 0;
  const neutral = row.pendingSave || infoOnly || Boolean(row.infoNote);

  const band =
    row.worst === "error"
      ? "border-destructive/40 bg-destructive/5"
      : row.worst === "warning"
        ? "border-warning/40 bg-warning/10"
        : neutral
          ? "border-border bg-muted/20"
          : "border-success/30 bg-success/10";
  const accent =
    row.worst === "error"
      ? "text-destructive"
      : row.worst === "warning"
        ? "text-warning"
        : neutral
          ? "text-muted-foreground"
          : "text-success";

  return (
    <div className={`rounded-md border px-2.5 py-1.5 ${band}`}>
      <div className="flex items-center gap-2 min-w-0">
        <span
          className={`w-4 h-4 rounded-full grid place-items-center shrink-0 ${accent}`}
          aria-hidden
        >
          {row.worst !== null ? (
            <AlertTriangle className="w-3 h-3" />
          ) : neutral ? (
            <Info className="w-3 h-3" />
          ) : (
            <Check className="w-3 h-3" />
          )}
        </span>
        <span className="text-xs font-mono truncate">{row.label}</span>
        <span className="text-micro text-muted-foreground shrink-0">
          {t(KIND_LABEL_KEY[row.kind])}
        </span>
        <span className="ml-auto flex items-center gap-2 shrink-0">
          {row.pendingSave ? (
            <span className="text-micro text-muted-foreground">
              {t("pendingSave")}
            </span>
          ) : null}
          {row.infoNote ? (
            <span className="text-micro text-muted-foreground">
              {row.infoNote}
            </span>
          ) : null}
          {row.file ? (
            <span className="text-micro font-mono text-muted-foreground hidden sm:inline">
              {row.file}
            </span>
          ) : null}
        </span>
      </div>

      {row.findings.map((f, i) => {
        const known = STATIC_CODES.has(f.code);
        const actionable = f.severity !== "info";
        const color = !actionable
          ? "text-muted-foreground"
          : f.severity === "error"
            ? "text-destructive"
            : "text-warning";
        return (
          <div key={`${f.code}-${i}`} className="mt-1 pl-6 space-y-0.5">
            <p className={`text-xs ${color}`}>
              {known ? t(`${codeKey(f.code)}_title`) : f.title}
              {f.code === "panel.loc_blank" && row.fixableLocKey && onFix ? (
                <button
                  type="button"
                  onClick={() => onFix(row)}
                  className="ml-2 inline-flex items-center h-5 px-1.5 rounded border border-warning/40 text-warning text-micro ui-interactive ui-focus-ring"
                >
                  {t("fixBlankLoc")}
                </button>
              ) : null}
            </p>
            <p className="text-micro text-muted-foreground">
              {known
                ? t(`${codeKey(f.code)}_suggestion`)
                : f.suggestion || f.reason}
            </p>
          </div>
        );
      })}
    </div>
  );
}
