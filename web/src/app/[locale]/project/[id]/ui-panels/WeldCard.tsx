"use client";

import { useEffect, useRef, useState } from "react";
import useSWR from "swr";
import { useTranslations } from "use-intl";
import { Zap } from "lucide-react";

import { panelContractApi, uiPanelsApi } from "@/lib/api";
import type { UiPanel, UiPanelControl } from "@/types";
import {
  buildWeldRows,
  fillBlankTextByName,
  fillBlankTooltipByName,
  findingsForPanel,
  summarizeRows,
  type WeldRow,
} from "@/components/workbench/ui-panels/weld/weldModel";
import { WeldTargetRow } from "@/components/workbench/ui-panels/weld/WeldTargetRow";
import {
  LivePreviewToggle,
  MockVarsTable,
} from "@/components/workbench/ui-panels/weld/MockVarsEditor";
import { useMockVarsStore } from "@/components/workbench/ui-panels/mockVars";

export interface WeldCardProps {
  projectId: number;
  panel: UiPanel;
  controls: UiPanelControl[];

  onControlsChange?: (next: UiPanelControl[]) => void;
}

export function WeldCard({
  projectId,
  panel,
  controls,
  onControlsChange,
}: WeldCardProps) {
  const t = useTranslations("uiPanelsWeld");
  const mockEnabled = useMockVarsStore((s) => s.enabled);

  const { data: files } = useSWR(
    ["ui-panels/files", projectId, panel.id, panel.updated_at],
    () => uiPanelsApi.files(projectId, panel.id),
  );

  const [scanNonce, setScanNonce] = useState(0);
  const controlsRef = useRef(controls);
  useEffect(() => {
    if (controlsRef.current === controls) return;
    controlsRef.current = controls;
    const timer = setTimeout(() => setScanNonce((n) => n + 1), 600);
    return () => clearTimeout(timer);
  }, [controls]);
  const {
    data: report,
    error: scanError,
    isValidating,
  } = useSWR(
    ["panel-contract", projectId, panel.updated_at, scanNonce],
    () => panelContractApi.scan(projectId),
    { keepPreviousData: true },
  );

  const rows = buildWeldRows(
    panel,
    controls,
    findingsForPanel(report?.findings, panel),
    t("tooltipNotSet"),
  );
  const { welded, total, pending } = summarizeRows(rows);

  const handleFix = (row: WeldRow) => {
    if (!onControlsChange || !row.controlName) return;
    const fill =
      row.kind === "tooltip" ? fillBlankTooltipByName : fillBlankTextByName;
    const fallbackText =
      row.controlName.replace(/[_-]+/g, " ").trim() || row.controlName;
    onControlsChange(fill(controls, row.controlName, fallbackText));
  };

  return (
    <div className="rounded-lg border border-success/30 overflow-hidden">
      <div className="flex items-center gap-2.5 px-3 py-2 border-b border-success/20 bg-success/5 flex-wrap">
        <Zap className="w-4 h-4 text-success shrink-0" aria-hidden />
        <div className="min-w-0">
          <h3 className="text-xs font-semibold tracking-tight">{t("title")}</h3>
          <p className="text-micro text-muted-foreground">{t("subtitle")}</p>
        </div>
        <div className="ml-auto flex items-center gap-3 shrink-0">
          <LivePreviewToggle />
          {isValidating && !report ? (
            <span className="text-micro text-muted-foreground">
              {t("scanning")}
            </span>
          ) : scanError && !report ? (
            <span className="text-micro text-muted-foreground">
              {t("scanFailed")}
            </span>
          ) : pending > 0 ? (
            <span className="text-xs font-semibold text-warning">
              {t("sealWelded", { welded, total })} ·{" "}
              {t("sealPending", { count: pending })}
            </span>
          ) : (
            <span className="text-xs font-semibold text-success">
              {t("sealWelded", { welded, total })} · {t("sealAllWelded")}
            </span>
          )}
        </div>
      </div>

      <div className="p-3 space-y-3">
        {mockEnabled ? <MockVarsTable controls={controls} /> : null}

        <div className="flex gap-4 items-start">
          <div className="w-40 shrink-0 rounded-md border border-border bg-card px-2.5 py-2 text-center">
            <p className="text-xs font-mono truncate">{panel.token}</p>
            <p className="text-micro text-muted-foreground mt-0.5">
              {t("sourceNote")}
            </p>
          </div>
          <div className="flex-1 min-w-0 space-y-1.5">
            {!report ? (
              <p className="text-xs text-muted-foreground px-1 py-2">
                {scanError ? t("scanFailed") : t("scanning")}
              </p>
            ) : (
              rows.map((row) => (
                <WeldTargetRow
                  key={row.id}
                  row={row}
                  onFix={onControlsChange ? handleFix : undefined}
                />
              ))
            )}
          </div>
        </div>

        {files ? (
          <div className="flex flex-wrap gap-2">
            {[files.gui_path, files.txt_path, ...files.loc_paths].map((p) => (
              <span
                key={p}
                className="text-micro font-mono px-2 py-1 rounded-md border border-border bg-card text-muted-foreground"
              >
                {p}
              </span>
            ))}
          </div>
        ) : null}

        <p className="text-micro text-muted-foreground">{t("scanNote")}</p>

        <p className="text-micro text-muted-foreground">
          {t("notCheckedNote")}
        </p>
      </div>
    </div>
  );
}
