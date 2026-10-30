"use client";

import * as React from "react";
import { BookOpen } from "lucide-react";
import { useTranslations } from "use-intl";
import { Tooltip as TooltipPrimitive } from "@base-ui/react/tooltip";

import { cn } from "@/lib/utils";

function TooltipProvider({
  delay = 200,
  closeDelay = 100,
  ...props
}: TooltipPrimitive.Provider.Props) {
  return (
    <TooltipPrimitive.Provider
      delay={delay}
      closeDelay={closeDelay}
      {...props}
    />
  );
}

function Tooltip({ ...props }: TooltipPrimitive.Root.Props) {
  return <TooltipPrimitive.Root {...props} />;
}

function TooltipTrigger({ ...props }: TooltipPrimitive.Trigger.Props) {
  return <TooltipPrimitive.Trigger data-slot="tooltip-trigger" {...props} />;
}

function TooltipContent({
  className,
  side = "top",
  sideOffset = 6,
  align = "center",
  interactive = false,
  children,
  ...props
}: TooltipPrimitive.Popup.Props &
  Pick<TooltipPrimitive.Positioner.Props, "side" | "sideOffset" | "align"> & {
    interactive?: boolean;
  }) {
  return (
    <TooltipPrimitive.Portal>
      <TooltipPrimitive.Positioner
        side={side}
        sideOffset={sideOffset}
        align={align}
        className={cn("isolate z-50", !interactive && "pointer-events-none")}
      >
        <TooltipPrimitive.Popup
          data-slot="tooltip-content"
          className={cn(
            "z-50 max-w-sm rounded-md bg-popover px-3 py-2 text-xs leading-relaxed text-popover-foreground shadow-md border border-border outline-hidden",
            "data-open:animate-in data-open:fade-in-0 data-open:zoom-in-95 data-closed:animate-out data-closed:fade-out-0 data-closed:zoom-out-95",
            className,
          )}
          {...props}
        >
          {children}
        </TooltipPrimitive.Popup>
      </TooltipPrimitive.Positioner>
    </TooltipPrimitive.Portal>
  );
}

function InfoTooltip({
  title,
  content,
  link,
  linkLabel,
  className,
}: {
  title?: string;
  content: string;
  link?: string;
  linkLabel?: string;
  className?: string;
}) {
  const t = useTranslations("tooltip");
  return (
    <Tooltip>
      <TooltipTrigger
        render={
          <span
            className={cn(
              "cursor-help text-xs leading-none select-none opacity-60 hover:opacity-100 transition",
              className,
            )}
            aria-label={
              title ? t("ariaLabelTitled", { title }) : t("ariaLabel")
            }
          >
            ⓘ
          </span>
        }
      />
      <TooltipContent interactive={!!link}>
        {title && <div className="font-medium mb-1">{title}</div>}
        <div className="text-muted-foreground whitespace-pre-line">
          {content}
        </div>
        {link && (
          <a
            href={link}
            target="_blank"
            rel="noopener noreferrer"
            className="mt-2 inline-block text-accent-foreground bg-accent/20 hover:bg-accent/30 rounded px-2 py-0.5 text-xs"
          >
            <BookOpen
              className="inline h-3 w-3 mr-0.5 align-[-2px]"
              aria-hidden
            />
            {linkLabel ?? t("linkFallback")} ↗
          </a>
        )}
      </TooltipContent>
    </Tooltip>
  );
}

export {
  Tooltip,
  TooltipTrigger,
  TooltipContent,
  TooltipProvider,
  InfoTooltip,
};
