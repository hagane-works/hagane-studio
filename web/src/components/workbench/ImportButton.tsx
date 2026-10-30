"use client";

import { useState, useRef } from "react";
import { useTranslations } from "use-intl";
import { Button } from "@/components/ui/button";
import type { ImportResult } from "@/lib/api";

interface ImportButtonProps {
  size?: "default" | "sm" | "xs" | "lg";
  label?: string;
  accept?: string;
  onImport: (file: File) => Promise<ImportResult>;
  onComplete: () => void;
}

export function useImportButton({
  label,
  accept = ".txt",
  onImport,
  onComplete,

  size = "sm",
}: ImportButtonProps) {
  const t = useTranslations("import");
  const displayLabel = label ?? t("defaultLabel");
  const [importing, setImporting] = useState(false);
  const [result, setResult] = useState<ImportResult | null>(null);
  const fileRef = useRef<HTMLInputElement>(null);

  const handleFileChange = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;

    setImporting(true);
    setResult(null);
    try {
      const res = await onImport(file);

      setResult({
        ...res,
        preserved_fields: res.preserved_fields ?? {},
        warnings: res.warnings ?? [],
      });
      onComplete();
    } catch (err) {
      console.error("Import failed:", err);
      setResult({
        total: 0,
        preserved_fields: {},
        warnings: [
          t("importFailed", {
            error: err instanceof Error ? err.message : "Unknown error",
          }),
        ],
      });
    } finally {
      setImporting(false);
      if (fileRef.current) fileRef.current.value = "";
    }
  };

  const preservedCount = result
    ? Object.values(result.preserved_fields).reduce((a, b) => a + b, 0)
    : 0;

  const trigger = (
    <>
      <input
        ref={fileRef}
        type="file"
        accept={accept}
        onChange={handleFileChange}
        className="hidden"
      />
      <Button
        size={size}
        variant="outline"
        disabled={importing}
        onClick={() => fileRef.current?.click()}
      >
        {importing ? t("importing") : displayLabel}
      </Button>
    </>
  );

  const resultPanel = result ? (
    <div className="rounded-md border p-3 text-xs space-y-1">
      <div className="flex items-start justify-between gap-2">
        <p className="font-medium">
          {result.total > 0
            ? t("importedItems", {
                count: result.created ?? result.created_nodes ?? result.total,
              })
            : t("noItemsFound")}
        </p>
        <button
          className="shrink-0 leading-none text-muted-foreground hover:text-foreground ui-interactive"
          onClick={() => setResult(null)}
          aria-label={t("dismissResult")}
        >
          ×
        </button>
      </div>
      {preservedCount > 0 && (
        <div>
          <p className="text-amber-600">
            {preservedCount} {t("preservedFieldsLabel")}
          </p>
          <div className="flex flex-wrap gap-1 mt-1">
            {Object.entries(result.preserved_fields).map(([field, count]) => (
              <span
                key={field}
                className="inline-block bg-amber-500/10 text-amber-400 rounded px-1.5 py-0.5 text-xs"
              >
                {field} ({count})
              </span>
            ))}
          </div>
        </div>
      )}
      {result.warnings.length > 0 && (
        <div className="text-red-500">
          {result.warnings.map((w, i) => (
            <p key={i}>{w}</p>
          ))}
        </div>
      )}
    </div>
  ) : null;

  return { trigger, resultPanel };
}

export function ImportButton(props: ImportButtonProps) {
  const { trigger, resultPanel } = useImportButton(props);
  return (
    <div className="space-y-2">
      {trigger}
      {resultPanel}
    </div>
  );
}
