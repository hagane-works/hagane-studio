"use client";

import { useState } from "react";
import type { ReactNode } from "react";
import { useTranslations } from "use-intl";
import { ChevronLeft, ChevronRight } from "lucide-react";
import { ScrollArea } from "@/components/ui/scroll-area";
import { OpenInGameLink } from "@/components/workbench/OpenInGameLink";

interface WorkbenchShellProps {
  listPanel: ReactNode;

  editorPanel: ReactNode;

  previewPanel: ReactNode;

  validationPanel?: ReactNode;

  listWidth?: string;
  previewWidth?: string;

  gameViewHref?: string | null;
}

export function WorkbenchShell({
  listPanel,
  editorPanel,
  previewPanel,
  validationPanel,

  listWidth = "w-72",
  previewWidth = "w-80",
  gameViewHref = null,
}: WorkbenchShellProps) {
  const t = useTranslations("workbenchShell");
  const [previewCollapsed, setPreviewCollapsed] = useState(false);

  return (
    <div className="flex flex-col h-full min-h-0">
      <div className="flex flex-1 min-h-0">
        <div
          className={`${listWidth} border-r border-border flex flex-col shrink-0 min-h-0 bg-muted`}
        >
          <ScrollArea className="flex-1 min-h-0">{listPanel}</ScrollArea>
        </div>

        <div className="flex-1 min-w-0 min-h-0 flex flex-col bg-background">
          <ScrollArea className="flex-1 min-h-0">{editorPanel}</ScrollArea>
        </div>

        {previewCollapsed ? (
          <div className="border-l border-border flex flex-col items-center bg-card shrink-0 min-h-0 w-8">
            <button
              className="w-full py-3 flex flex-col items-center gap-1 text-muted-foreground hover:text-foreground hover:bg-muted transition-colors"
              onClick={() => setPreviewCollapsed(false)}
              title={t("expandPreview")}
            >
              <ChevronLeft className="h-4 w-4" />
              <span
                className="text-micro font-medium uppercase tracking-wider opacity-60 mt-1"
                style={{ writingMode: "vertical-rl" }}
              >
                {t("previewLabel")}
              </span>
            </button>
          </div>
        ) : (
          <div
            className={`${previewWidth} border-l border-border flex flex-col shrink-0 min-h-0 bg-card`}
          >
            <div className="flex items-center justify-between px-3 py-2 border-b border-border shrink-0">
              <span className="text-micro uppercase font-semibold tracking-widest text-muted-foreground">
                {t("previewLabel")}
              </span>
              <span className="flex-1" />
              {gameViewHref && (
                <OpenInGameLink href={gameViewHref} className="mr-1" />
              )}
              <button
                className="text-muted-foreground hover:text-foreground transition-colors p-0.5 rounded hover:bg-muted"
                onClick={() => setPreviewCollapsed(true)}
                title={t("collapsePreview")}
              >
                <ChevronRight className="h-4 w-4" />
              </button>
            </div>
            <ScrollArea className="flex-1 min-h-0">{previewPanel}</ScrollArea>
          </div>
        )}
      </div>

      {validationPanel && (
        <div className="border-t border-border h-40 shrink-0">
          <ScrollArea className="h-full">{validationPanel}</ScrollArea>
        </div>
      )}
    </div>
  );
}
