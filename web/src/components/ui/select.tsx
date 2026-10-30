"use client";

import * as React from "react";
import { Select as SelectPrimitive } from "@base-ui/react/select";
import { Check, ChevronDown } from "lucide-react";

import { cn } from "@/lib/utils";
import { collectSections, type SelectOpt as Opt } from "./selectOptions";

function renderItem(o: Opt) {
  return (
    <SelectPrimitive.Item
      key={o.value}
      value={o.value}
      disabled={o.disabled}
      className={cn(
        "relative flex cursor-pointer select-none items-center gap-2 rounded-md py-1.5 pl-2 pr-7 text-sm outline-none",

        "data-highlighted:bg-accent/15 data-highlighted:text-foreground",
        "data-disabled:pointer-events-none data-disabled:opacity-50",
      )}
    >
      <SelectPrimitive.ItemText className="min-w-0 flex-1 truncate">
        {o.node}
      </SelectPrimitive.ItemText>
      <SelectPrimitive.ItemIndicator className="absolute right-2 flex items-center text-accent">
        <Check className="h-3.5 w-3.5" aria-hidden />
      </SelectPrimitive.ItemIndicator>
    </SelectPrimitive.Item>
  );
}

type SelectExtraProps = { trailingChip?: React.ReactNode };

function Select({
  className,
  children,
  value,
  defaultValue,
  onChange,
  disabled,
  name,
  trailingChip,
  "aria-label": ariaLabel,
  title,
  ...rest
}: React.ComponentProps<"select"> & SelectExtraProps) {
  const sections = React.useMemo(() => collectSections(children), [children]);
  const options = React.useMemo(
    () => sections.flatMap((s) => s.options),
    [sections],
  );
  const selected = options.find(
    (o) => o.value === String(value ?? defaultValue ?? ""),
  );

  return (
    <SelectPrimitive.Root
      value={value != null ? String(value) : undefined}
      defaultValue={defaultValue != null ? String(defaultValue) : undefined}
      disabled={disabled}
      name={name}
      onValueChange={(next) => {
        onChange?.({
          target: { value: next == null ? "" : String(next) },
        } as unknown as React.ChangeEvent<HTMLSelectElement>);
      }}
    >
      <SelectPrimitive.Trigger
        aria-label={ariaLabel}
        title={title}
        className={cn(
          "h-9 w-full min-w-0 rounded-lg border border-input bg-card px-3 py-1.5 pr-8 text-sm transition-colors outline-none focus-visible:ring-1 focus-visible:ring-ring focus-visible:ring-offset-0 disabled:pointer-events-none disabled:cursor-not-allowed disabled:opacity-50 aria-invalid:border-destructive",
          "relative flex items-center justify-between gap-2 text-left cursor-pointer hover:bg-muted/40",
          className,
        )}
        {...(rest as Record<string, unknown>)}
      >
        <span className="min-w-0 flex-1 truncate">
          {selected ? (
            selected.node
          ) : (
            <span className="text-muted-foreground">&nbsp;</span>
          )}
        </span>
        {trailingChip ? (
          <span className="shrink-0 rounded-full border border-border bg-muted px-1.5 text-micro font-semibold tracking-wide text-muted-foreground">
            {trailingChip}
          </span>
        ) : null}
        <SelectPrimitive.Icon className="pointer-events-none absolute right-2.5 flex shrink-0 text-muted-foreground">
          <ChevronDown className="h-4 w-4" aria-hidden />
        </SelectPrimitive.Icon>
      </SelectPrimitive.Trigger>

      <SelectPrimitive.Portal>
        <SelectPrimitive.Positioner
          alignItemWithTrigger={false}
          side="bottom"
          align="start"
          sideOffset={4}
          className="isolate z-50"
        >
          <SelectPrimitive.Popup
            className={cn(
              "z-50 max-h-[min(24rem,var(--available-height))] min-w-(--anchor-width) origin-(--transform-origin)",
              "overflow-y-auto rounded-lg border border-border bg-popover p-1 text-sm text-popover-foreground shadow-md outline-hidden",
              "duration-100 data-open:animate-in data-open:fade-in-0 data-open:zoom-in-95 data-closed:animate-out data-closed:fade-out-0 data-closed:zoom-out-95",
            )}
          >
            <SelectPrimitive.List>
              {sections.map((section, si) =>
                section.label == null ? (
                  <React.Fragment key={`s${si}`}>
                    {section.options.map(renderItem)}
                  </React.Fragment>
                ) : (
                  <SelectPrimitive.Group key={`s${si}`}>
                    <SelectPrimitive.GroupLabel className="px-2 pt-2 pb-1 text-micro font-semibold uppercase tracking-widest text-muted-foreground">
                      {section.label}
                    </SelectPrimitive.GroupLabel>
                    {section.options.map(renderItem)}
                  </SelectPrimitive.Group>
                ),
              )}
            </SelectPrimitive.List>
          </SelectPrimitive.Popup>
        </SelectPrimitive.Positioner>
      </SelectPrimitive.Portal>
    </SelectPrimitive.Root>
  );
}

export { Select };
