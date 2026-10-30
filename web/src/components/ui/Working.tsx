"use client";

import { cn } from "@/lib/utils";

export function Working({
  className,
  label,
}: {
  className?: string;
  label?: string;
}) {
  return (
    <span
      role={label ? "status" : undefined}
      aria-label={label}
      aria-hidden={label ? undefined : true}
      className={cn(
        "animate-pulse-ring inline-block h-2 w-2 shrink-0 rounded-full bg-accent",
        className,
      )}
    />
  );
}
