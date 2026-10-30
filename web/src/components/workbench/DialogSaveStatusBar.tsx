"use client";

interface DialogSaveStatusBarProps {
  saving: boolean;

  saved: boolean;

  error: string | null;

  savingLabel: string;

  savedLabel: string;
}

export function DialogSaveStatusBar({
  saving,
  saved,
  error,
  savingLabel,
  savedLabel,
}: DialogSaveStatusBarProps) {
  if (!saving && !saved && !error) return null;

  return (
    <div
      className={`-mx-4 px-4 py-1.5 text-xs font-medium flex items-center gap-2
        transition-[opacity,background-color,border-color] duration-300
        ${
          saving
            ? "bg-muted/80 text-muted-foreground border-y border-border"
            : saved
              ? "bg-success/10 text-success border-y border-success/20"
              : error
                ? "bg-destructive/10 text-destructive border-y border-destructive/20"
                : ""
        }`}
      role="status"
      aria-live="polite"
    >
      <span>
        {saving ? savingLabel : saved ? savedLabel : error ? error : ""}
      </span>
    </div>
  );
}
