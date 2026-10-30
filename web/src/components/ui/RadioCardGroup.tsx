"use client";

import { useId, useRef, type KeyboardEvent, type ReactNode } from "react";
import { cn } from "@/lib/utils";

export type RadioCardOption<T extends string> = {
  value: T;
  title: ReactNode;

  consequence: ReactNode;

  price?: ReactNode;
} & (
  { disabled?: false; reason?: never } | { disabled: true; reason: ReactNode }
);

export interface RadioCardGroupProps<T extends string> {
  value: T;
  onChange: (value: T) => void;
  options: ReadonlyArray<RadioCardOption<T>>;

  label: string;

  orientation?: "vertical" | "horizontal";
  className?: string;
}

export function RadioCardGroup<T extends string>({
  value,
  onChange,
  options,
  label,
  orientation = "vertical",
  className,
}: RadioCardGroupProps<T>) {
  const containerRef = useRef<HTMLDivElement>(null);
  const idPrefix = useId();

  const selectedOption = options.find((option) => option.value === value);
  const firstEnabledValue = options.find((option) => !option.disabled)?.value;
  const tabStopValue =
    selectedOption && !selectedOption.disabled
      ? selectedOption.value
      : firstEnabledValue;

  const handleKeyDown = (event: KeyboardEvent<HTMLDivElement>) => {
    if (
      !["ArrowDown", "ArrowRight", "ArrowUp", "ArrowLeft"].includes(event.key)
    )
      return;
    if (!options.some((option) => !option.disabled)) return;
    event.preventDefault();
    const forward = event.key === "ArrowDown" || event.key === "ArrowRight";
    const step = forward ? 1 : -1;

    const from = options.findIndex((option) => option.value === value);
    const start = from === -1 ? 0 : from;
    for (let hop = 1; hop <= options.length; hop += 1) {
      const index =
        (((start + step * hop) % options.length) + options.length) %
        options.length;
      const candidate = options[index];
      if (candidate.disabled) continue;
      onChange(candidate.value);

      const nodes =
        containerRef.current?.querySelectorAll<HTMLButtonElement>(
          "[data-rung]",
        );
      Array.from(nodes ?? [])
        .find((node) => node.dataset.rung === candidate.value)
        ?.focus();
      return;
    }
  };

  return (
    <div
      ref={containerRef}
      role="radiogroup"
      aria-orientation={orientation}
      aria-label={label}
      onKeyDown={handleKeyDown}
      className={cn(
        "flex flex-col gap-1.5",

        orientation === "horizontal" &&
          "xl:grid xl:grid-flow-col xl:auto-cols-fr xl:gap-2",
        className,
      )}
    >
      {options.map((option) => {
        const selected = option.value === value;
        const reasonId = option.disabled
          ? `${idPrefix}-${option.value}-why`
          : undefined;
        return (
          <button
            key={option.value}
            type="button"
            role="radio"
            aria-checked={selected}
            aria-disabled={option.disabled || undefined}
            aria-describedby={reasonId}

            tabIndex={option.value === tabStopValue ? 0 : -1}
            data-rung={option.value}
            onClick={() => {
              if (!option.disabled) onChange(option.value);
            }}
            className={cn(
              "ui-interactive outline-none focus-visible:ring-2 focus-visible:ring-accent/60",
              "flex w-full items-start gap-2.5 rounded-md border px-3 py-2.5 text-left",

              selected
                ? "border-border bg-card shadow-sm"
                : "border-transparent hover:bg-muted/60",
              option.disabled && "cursor-not-allowed opacity-50",
            )}
          >
            <span
              aria-hidden="true"
              className={cn(
                "mt-0.5 h-3.5 w-3.5 shrink-0 rounded-full border",

                selected
                  ? "border-accent bg-accent/25"
                  : "border-muted-foreground/60",
              )}
            />
            <span className="min-w-0 flex-1">
              <span className="flex flex-wrap items-baseline gap-x-2 text-sm text-foreground">
                {option.title}
                {option.price ? (
                  <span className="tech-tabular text-xs text-muted-foreground">
                    {option.price}
                  </span>
                ) : null}
              </span>

              <span className="mt-0.5 block text-xs text-muted-foreground">
                {option.consequence}
              </span>
              {reasonId ? (
                <span id={reasonId} className="mt-1 block text-xs text-warning">
                  {option.reason}
                </span>
              ) : null}
            </span>
          </button>
        );
      })}
    </div>
  );
}
