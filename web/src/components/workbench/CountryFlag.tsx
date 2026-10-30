"use client";

import { Globe } from "lucide-react";
import { cn } from "@/lib/utils";

export function isResolvedCountryName(
  name: string | null | undefined,
  tag: string | null | undefined,
): boolean {
  const n = (name ?? "").trim();
  if (!n) return false;
  const t = (tag ?? "").trim();
  if (!t) return true;
  const lettersOf = (s: string) => s.replace(/[^a-z]/gi, "").toUpperCase();
  return lettersOf(n) !== lettersOf(t);
}

interface CountryFlagProps {
  color?: number[] | null;

  className?: string;
}

export function CountryFlag({ color, className }: CountryFlagProps) {
  if (color && color.length === 3) {
    return (
      <span
        aria-hidden
        className={cn(
          "inline-block w-3.5 h-3.5 rounded-sm shrink-0 border border-border/60",
          className,
        )}
        style={{ backgroundColor: `rgb(${color.join(", ")})` }}
      />
    );
  }
  return (
    <Globe
      aria-hidden
      className={cn("w-3.5 h-3.5 shrink-0 text-muted-foreground", className)}
    />
  );
}
