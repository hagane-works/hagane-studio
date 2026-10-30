"use client";

import { useRef } from "react";
import { useTranslations } from "use-intl";
import { Button } from "@/components/ui/button";

export function FilePick({
  accept,
  disabled,
  fileName,
  onPick,
  className,
}: {
  accept?: string;
  disabled?: boolean;

  fileName?: string | null;

  onPick: (file: File | null) => void;
  className?: string;
}) {
  const t = useTranslations("common");
  const input = useRef<HTMLInputElement>(null);
  return (
    <span className={`flex min-w-0 items-center gap-2 ${className ?? ""}`}>
      <input
        ref={input}
        type="file"
        accept={accept}
        disabled={disabled}
        tabIndex={-1}
        aria-hidden
        className="sr-only"
        onChange={(e) => {
          const file = e.target.files?.[0] ?? null;
          e.target.value = "";
          onPick(file);
        }}
      />
      <Button
        type="button"
        size="sm"
        variant="outline"
        disabled={disabled}
        onClick={() => input.current?.click()}
      >
        {t("chooseFile")}
      </Button>
      {fileName !== undefined && (
        <span className="min-w-0 truncate text-xs text-muted-foreground">
          {fileName ?? t("noFileChosen")}
        </span>
      )}
    </span>
  );
}
