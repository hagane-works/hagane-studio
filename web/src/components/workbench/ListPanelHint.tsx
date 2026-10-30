"use client";

import type { ReactNode } from "react";
import { cn } from "@/lib/utils";

export function ListPanelHint({
  children,
  className,
}: {
  children: ReactNode;
  className?: string;
}) {
  return (
    <div
      className={cn(
        "rounded-[11px] border border-border bg-muted px-4 py-3",
        "text-xs leading-relaxed text-muted-foreground",
        className,
      )}
    >
      {children}
    </div>
  );
}

export function ListPanelEmpty({
  children,
  className,
}: {
  children: ReactNode;
  className?: string;
}) {
  return (
    <div
      className={cn(
        "flex flex-1 items-center justify-center px-4 py-10 text-center",
        "text-sm italic text-muted-foreground",
        className,
      )}
    >
      {children}
    </div>
  );
}
