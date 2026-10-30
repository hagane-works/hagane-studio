"use client";

import type { ReactNode } from "react";
import { InfoTooltip } from "@/components/ui/tooltip";
import { cn } from "@/lib/utils";

interface SectionHeaderProps {
  title: string;
  tooltip?: string;
  className?: string;
  children?: ReactNode;
}

export function SectionHeader({
  title,
  tooltip,
  className,
  children,
}: SectionHeaderProps) {
  return (
    <div className={cn("flex items-center justify-between", className)}>
      <h3 className="text-sm font-semibold text-muted-foreground uppercase tracking-wide flex items-center gap-1.5">
        {title}
        {tooltip && <InfoTooltip title={title} content={tooltip} />}
      </h3>
      {children}
    </div>
  );
}
