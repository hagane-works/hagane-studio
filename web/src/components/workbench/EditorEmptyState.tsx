"use client";

import type { ReactNode } from "react";
import { cn } from "@/lib/utils";

interface EditorEmptyStateProps {
  icon?: ReactNode;
  title: string;
  description?: string;

  actions?: ReactNode;
  className?: string;

  variant?: "compact" | "hero";
}

export function EditorEmptyState({
  icon,
  title,
  description,
  actions,
  className,
  variant = "compact",
}: EditorEmptyStateProps) {
  const hero = variant === "hero";
  return (
    <div
      className={cn(
        "flex flex-col items-center text-center",

        hero
          ? "gap-4 rounded-2xl border border-border bg-card p-12"
          : "gap-2 rounded-lg border border-dashed bg-muted/20 p-6",
        className,
      )}
    >
      {icon &&
        (hero ? (
          <div className="flex h-14 w-14 items-center justify-center rounded-xl bg-muted">
            {icon}
          </div>
        ) : (
          icon
        ))}
      <p
        className={cn(
          "font-medium text-foreground",

          hero ? "text-2xl font-semibold tracking-tight" : "text-xs",
        )}
      >
        {title}
      </p>
      {description && (
        <p
          className={cn(
            "text-muted-foreground",
            hero ? "max-w-md text-sm leading-relaxed" : "text-xs",
          )}
        >
          {description}
        </p>
      )}
      {actions && (
        <div
          className={cn(
            "flex flex-wrap items-center justify-center gap-2",
            hero ? "mt-3" : "mt-2",
          )}
        >
          {actions}
        </div>
      )}
    </div>
  );
}
