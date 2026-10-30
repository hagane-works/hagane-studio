import { cn } from "@/lib/utils";

interface ProgressProps {
  value?: number | null;

  indeterminate?: boolean;
  className?: string;

  ariaLabel?: string;
}

export function Progress({
  value,
  indeterminate,
  className,
  ariaLabel,
}: ProgressProps) {
  const hasValue =
    typeof value === "number" && Number.isFinite(value) && !indeterminate;
  const pct = hasValue ? Math.max(0, Math.min(100, value as number)) : 100;

  return (
    <div
      role="progressbar"
      aria-label={ariaLabel ?? "progress"}
      aria-valuemin={0}
      aria-valuemax={100}
      aria-valuenow={hasValue ? pct : undefined}
      className={cn(
        "relative h-2.5 w-full overflow-hidden rounded-full bg-muted/40",
        className,
      )}
    >
      <div
        className={cn(
          "h-full rounded-full transition-[width] duration-300 ease-out",
          hasValue ? "bg-primary" : "bg-primary/70 animate-pulse",
        )}
        style={{ width: hasValue ? `${pct}%` : "100%" }}
      />
    </div>
  );
}
