"use client";

import { useRef, type KeyboardEvent, type ReactNode } from "react";
import { cn } from "@/lib/utils";

export interface SegmentedOption<T extends string> {
  value: T;
  label: ReactNode;

  icon?: ReactNode;

  count?: number;
  title?: string;
  disabled?: boolean;
}

export interface SegmentedControlProps<T extends string> {
  value: T;
  onChange: (value: T) => void;
  options: ReadonlyArray<SegmentedOption<T>>;

  label: string;

  size?: "sm" | "default";
  className?: string;
}

export function SegmentedControl<T extends string>({
  value,
  onChange,
  options,
  label,
  size = "default",
  className,
}: SegmentedControlProps<T>) {
  const containerRef = useRef<HTMLDivElement>(null);

  const handleKeyDown = (event: KeyboardEvent<HTMLDivElement>) => {
    const keys = ["ArrowRight", "ArrowDown", "ArrowLeft", "ArrowUp"];
    if (!keys.includes(event.key)) return;
    const usable = options.filter((option) => !option.disabled);
    if (usable.length < 2) return;
    event.preventDefault();
    const forward = event.key === "ArrowRight" || event.key === "ArrowDown";
    const current = usable.findIndex((option) => option.value === value);
    const nextIndex =
      (current + (forward ? 1 : -1) + usable.length) % usable.length;
    const next = usable[nextIndex];
    onChange(next.value);
    containerRef.current
      ?.querySelector<HTMLButtonElement>(
        `[data-segment="${CSS.escape(next.value)}"]`,
      )
      ?.focus();
  };

  return (
    <div
      ref={containerRef}
      role="radiogroup"
      aria-label={label}
      onKeyDown={handleKeyDown}
      className={cn(
        "inline-grid w-full auto-cols-fr grid-flow-col gap-0.5 rounded-md bg-muted p-0.5",
        className,
      )}
    >
      {options.map((option) => {
        const selected = option.value === value;
        return (
          <button
            key={option.value}
            type="button"
            role="radio"
            aria-checked={selected}

            tabIndex={selected ? 0 : -1}
            data-segment={option.value}
            disabled={option.disabled}
            title={option.title}
            onClick={() => !option.disabled && onChange(option.value)}
            className={cn(
              "ui-interactive outline-none focus-visible:ring-2 focus-visible:ring-accent/60 flex min-w-0 items-center justify-center gap-1.5 rounded border px-2.5",

              size === "sm" ? "h-7 text-xs" : "h-8 text-sm",
              selected
                ? "border-border bg-card shadow-sm"
                : "border-transparent text-muted-foreground hover:bg-muted/60",
              option.disabled && "cursor-not-allowed opacity-50",
            )}
          >
            {option.icon}
            <span className="truncate">{option.label}</span>
            {typeof option.count === "number" && (
              <span className="tech-tabular shrink-0 text-muted-foreground">
                {option.count}
              </span>
            )}
          </button>
        );
      })}
    </div>
  );
}
