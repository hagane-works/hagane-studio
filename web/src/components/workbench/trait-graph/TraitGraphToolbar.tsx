"use client";

import { type ComponentProps } from "react";
import { useTranslations } from "use-intl";
import { Panel, useReactFlow, useViewport } from "@xyflow/react";
import { ZoomIn, ZoomOut, Frame } from "lucide-react";

import { Button } from "@/components/ui/button";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";

const ICON_BTN = "h-8 w-8 p-0";

function Divider() {
  return <div className="mx-0.5 h-5 w-px bg-border" />;
}

function ToolbarButton({
  tip,
  children,
  ...buttonProps
}: { tip: string } & ComponentProps<typeof Button>) {
  return (
    <Tooltip>
      <TooltipTrigger
        render={
          <Button
            variant="ghost"
            size="sm"
            className={ICON_BTN}
            {...buttonProps}
          >
            {children}
          </Button>
        }
      />
      <TooltipContent>{tip}</TooltipContent>
    </Tooltip>
  );
}

export function TraitGraphToolbar() {
  const t = useTranslations("customTraitsPanel");
  const { zoomIn, zoomOut, zoomTo, fitView } = useReactFlow();
  const { zoom } = useViewport();

  return (
    <Panel position="bottom-center" className="mb-3!">
      <TooltipProvider>
        <div className="ui-glass flex items-center gap-0.5 rounded-full px-1.5 py-1">
          <ToolbarButton tip={t("toolbarZoomOut")} onClick={() => zoomOut()}>
            <ZoomOut className="h-4 w-4" />
          </ToolbarButton>
          <Tooltip>
            <TooltipTrigger
              render={
                <button
                  onClick={() => zoomTo(1, { duration: 200 })}
                  className="ui-interactive w-12 rounded text-center text-xs tech-tabular text-muted-foreground hover:text-foreground"
                >
                  {Math.round(zoom * 100)}%
                </button>
              }
            />
            <TooltipContent>{t("toolbarZoomReset")}</TooltipContent>
          </Tooltip>
          <ToolbarButton tip={t("toolbarZoomIn")} onClick={() => zoomIn()}>
            <ZoomIn className="h-4 w-4" />
          </ToolbarButton>

          <Divider />

          <ToolbarButton
            tip={t("toolbarFitView")}
            onClick={() => fitView({ padding: 0.2, duration: 200 })}
          >
            <Frame className="h-4 w-4" />
          </ToolbarButton>
        </div>
      </TooltipProvider>
    </Panel>
  );
}
