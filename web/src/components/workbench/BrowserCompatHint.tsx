"use client";

import { useSyncExternalStore } from "react";
import { Info } from "lucide-react";
import { useTranslations } from "use-intl";
import { hasDirectoryPicker } from "@/lib/fsAccess";
import { cn } from "@/lib/utils";

interface BrowserCompatHintProps {
  mode: "folderUpload" | "folderWrite";
  className?: string;
}

const subscribeToHydration = () => () => {};
const getClientHydrated = () => true;
const getServerHydrated = () => false;

export function BrowserCompatHint({ mode, className }: BrowserCompatHintProps) {
  const t = useTranslations("browserCompat");
  const mounted = useSyncExternalStore(
    subscribeToHydration,
    getClientHydrated,
    getServerHydrated,
  );

  if (!mounted || hasDirectoryPicker()) return null;

  return (
    <div
      className={cn(
        "flex items-start gap-2 rounded-md border border-border bg-muted/30 px-3 py-2 text-xs text-muted-foreground",
        className,
      )}
    >
      <Info className="mt-0.5 h-3.5 w-3.5 shrink-0" aria-hidden />
      <span className="leading-snug">
        {mode === "folderUpload" ? t("uploadHint") : t("writeHint")}
      </span>
    </div>
  );
}
