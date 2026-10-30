"use client";

import { useEffect } from "react";
import { useTranslations } from "use-intl";
import { AlertTriangle, RotateCcw, RefreshCw } from "lucide-react";
import { Button } from "@/components/ui/button";
import {
  claimChunkReload,
  isChunkLoadError,
  reloadPage,
} from "@/lib/chunkLoadRecovery";
import { useEditorDirtyStore } from "@/lib/store/editorDirtyStore";

const loggedErrors = new WeakSet<Error>();

export interface ErrorBoundaryFallbackProps {
  error: Error & { digest?: string };

  reset: () => void;

  scope: "workbench" | "root";
}

export function ErrorBoundaryFallback({
  error,
  reset,
  scope,
}: ErrorBoundaryFallbackProps) {
  const t = useTranslations("errorBoundary");

  useEffect(() => {
    if (loggedErrors.has(error)) return;
    loggedErrors.add(error);
    console.error("[error-boundary]", scope, error);
    if (isChunkLoadError(error)) {
      if (claimChunkReload(useEditorDirtyStore.getState().hasAnyDirty()))
        reloadPage();
    }
  }, [error, scope]);

  const chunkFailure = isChunkLoadError(error);

  const title = chunkFailure
    ? t("chunkTitle")
    : scope === "workbench"
      ? t("workbenchTitle")
      : t("rootTitle");
  const body = chunkFailure
    ? t("chunkBody")
    : scope === "workbench"
      ? t("workbenchBody")
      : t("rootBody");
  const detail = error?.message?.trim() || t("unknownError");

  return (
    <div className="flex h-full w-full items-center justify-center p-8 bg-background">
      <div className="ui-panel w-full max-w-md p-6">
        <div className="flex items-start gap-3">
          <span
            className="mt-0.5 flex h-9 w-9 shrink-0 items-center justify-center rounded-md bg-warning/10"
            aria-hidden
          >
            <AlertTriangle className="h-5 w-5 text-warning" strokeWidth={1.8} />
          </span>
          <div className="min-w-0 flex-1">
            <h2 className="text-lg font-semibold tracking-tight text-foreground">
              {title}
            </h2>
            <p className="mt-1 text-sm leading-normal text-muted-foreground">
              {body}
            </p>
          </div>
        </div>

        {!chunkFailure && (
          <p className="mt-4 text-xs leading-snug text-muted-foreground">
            {t("draftHint")}
          </p>
        )}

        <div className="mt-5 flex items-center gap-2">
          {!chunkFailure && (
            <Button onClick={reset} className="ui-interactive">
              <RotateCcw className="h-4 w-4" strokeWidth={1.8} />
              {t("retry")}
            </Button>
          )}
          <Button
            variant={chunkFailure ? "default" : "outline"}
            onClick={reloadPage}
            className="ui-interactive"
          >
            <RefreshCw className="h-4 w-4" strokeWidth={1.8} />
            {t("reload")}
          </Button>
        </div>

        <details className="mt-5 border-t border-border pt-3">
          <summary className="cursor-pointer text-xs text-muted-foreground hover:text-foreground ui-interactive select-none">
            {t("details")}
          </summary>
          <pre className="mt-2 max-h-40 overflow-auto whitespace-pre-wrap break-words rounded-md bg-muted/40 p-3 font-mono text-xs text-muted-foreground">
            {detail}
            {error?.digest ? `\n\ndigest: ${error.digest}` : ""}
          </pre>
        </details>
      </div>
    </div>
  );
}
