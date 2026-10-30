"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { useTranslations } from "use-intl";
import { FileCode2, RefreshCw } from "lucide-react";

import { Button } from "@/components/ui/button";
import { oobUnitsApi } from "@/lib/api/oobUnits";

interface OobScriptPreviewProps {
  projectId: number;
  oobFile: string;
  refreshToken: number;
}

export function OobScriptPreview({
  projectId,
  oobFile,
  refreshToken,
}: OobScriptPreviewProps) {
  const t = useTranslations("oobUnitsEditor");
  const requestSequence = useRef(0);
  const [script, setScript] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const load = useCallback(async () => {
    const sequence = ++requestSequence.current;
    setLoading(true);
    setError(null);
    setScript("");
    try {
      const result = await oobUnitsApi.preview(
        projectId,
        oobFile ? { oobFile } : undefined,
      );
      if (requestSequence.current === sequence) setScript(result.script);
    } catch (caught) {
      if (requestSequence.current === sequence) {
        setError(caught instanceof Error ? caught.message : String(caught));
      }
    } finally {
      if (requestSequence.current === sequence) setLoading(false);
    }
  }, [oobFile, projectId]);

  useEffect(() => {
    void load();
  }, [load, refreshToken]);

  return (
    <aside className="flex h-full w-96 shrink-0 flex-col border-l border-border bg-card">
      <div className="flex min-h-14 items-center gap-3 border-b border-border px-4">
        <FileCode2 className="h-4 w-4 text-muted-foreground" aria-hidden />
        <div className="min-w-0 flex-1">
          <h2 className="text-sm font-semibold">{t("previewTitle")}</h2>
          <p className="truncate text-xs text-muted-foreground">
            {oobFile ? `history/units/${oobFile}.txt` : t("allFiles")}
          </p>
        </div>
        <Button
          type="button"
          size="icon"
          variant="ghost"
          className="h-8 w-8 ui-interactive"
          title={t("previewReload")}
          aria-label={t("previewReload")}
          disabled={loading}
          onClick={() => void load()}
        >
          <RefreshCw className="h-4 w-4" aria-hidden />
        </Button>
      </div>

      <div className="min-h-0 flex-1 overflow-auto p-4">
        {loading && (
          <div className="animate-pulse-ring rounded-md border border-accent p-3 text-xs text-muted-foreground">
            {t("previewLoading")}
          </div>
        )}
        {error && (
          <div
            role="alert"
            className="border-y border-destructive/20 bg-destructive/10 px-3 py-2 text-xs text-destructive"
          >
            {t("previewError", { error })}
          </div>
        )}
        {!loading && !error && !script && (
          <div className="rounded-md border border-dashed border-border p-6 text-center text-xs text-muted-foreground">
            {t("previewEmpty")}
          </div>
        )}
        {!loading && !error && script && (
          <pre className="whitespace-pre-wrap break-words font-mono text-xs leading-relaxed text-foreground">
            {script}
          </pre>
        )}
      </div>
    </aside>
  );
}
