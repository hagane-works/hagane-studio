"use client";

import { AlertTriangle } from "lucide-react";
import { useTranslations } from "use-intl";
import { useState } from "react";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import { EditorEmptyState } from "./EditorEmptyState";

interface WorkbenchLoadErrorProps {
  onRetry: () => unknown | Promise<unknown>;
  className?: string;
  variant?: "compact" | "hero";
}

export function WorkbenchLoadError({
  onRetry,
  className,
  variant = "compact",
}: WorkbenchLoadErrorProps) {
  const t = useTranslations("workbenchLoadError");
  const [retrying, setRetrying] = useState(false);

  const retry = async () => {
    if (retrying) return;
    setRetrying(true);
    try {
      await onRetry();
    } catch {
    } finally {
      setRetrying(false);
    }
  };

  return (
    <div role="alert">
      <EditorEmptyState
        variant={variant}
        className={cn("border-destructive/40 bg-destructive/5", className)}
        icon={
          <AlertTriangle className="h-6 w-6 text-destructive" aria-hidden />
        }
        title={t("title")}
        description={t("description")}
        actions={
          <Button
            size="sm"
            variant="outline"
            disabled={retrying}
            onClick={() => void retry()}
          >
            {t("retry")}
          </Button>
        }
      />
    </div>
  );
}
