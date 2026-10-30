"use client";

import { useState } from "react";
import { useTranslations } from "use-intl";
import { FileUp } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { oobUnitsApi } from "@/lib/api/oobUnits";
import type { OobUnitsImportResult } from "@/types/oobUnits";

import { localizeOobWarning } from "./oobWarnings";

interface OobImportPanelProps {
  projectId: number;
  defaultOobFile: string;
  beforeImport?: () => boolean | Promise<boolean>;
  onImported: (result: OobUnitsImportResult) => void | Promise<void>;
}

export function OobImportPanel({
  projectId,
  defaultOobFile,
  beforeImport,
  onImported,
}: OobImportPanelProps) {
  const t = useTranslations("oobUnitsEditor");
  const [oobFile, setOobFile] = useState(defaultOobFile);
  const [sourceFile, setSourceFile] = useState(
    defaultOobFile ? `history/units/${defaultOobFile}.txt` : "",
  );
  const [countryTag, setCountryTag] = useState("");
  const [policy, setPolicy] = useState<"overwrite" | "skip">("overwrite");
  const [text, setText] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [result, setResult] = useState<OobUnitsImportResult | null>(null);
  const [error, setError] = useState<string | null>(null);

  const handleImport = async () => {
    if (beforeImport && !(await beforeImport())) return;
    setSubmitting(true);
    setResult(null);
    setError(null);
    try {
      const next = await oobUnitsApi.importText(projectId, {
        oob_file: oobFile.trim(),
        source_file: sourceFile.trim().replaceAll("\\", "/"),
        country_tag: countryTag.trim().toUpperCase(),
        text,
        policy,
      });
      setResult(next);
      setText("");
      await onImported(next);
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : String(caught));
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <section
      className="border-b border-border bg-muted/30 p-4"
      aria-labelledby="oob-import-title"
    >
      <div className="mx-auto max-w-5xl space-y-4">
        <div className="flex items-start gap-3">
          <FileUp className="mt-0.5 h-5 w-5 shrink-0 text-accent" aria-hidden />
          <div>
            <h2 id="oob-import-title" className="text-sm font-semibold">
              {t("importTitle")}
            </h2>
            <p className="mt-1 text-xs text-muted-foreground">
              {t("importHint")}
            </p>
          </div>
        </div>

        <div className="grid grid-cols-4 gap-3">
          <div className="space-y-1">
            <Label htmlFor="oob-import-file" className="text-xs">
              {t("oobFile")}
            </Label>
            <Input
              id="oob-import-file"
              className="font-mono text-xs"
              value={oobFile}
              onChange={(event) => setOobFile(event.target.value)}
            />
          </div>
          <div className="col-span-2 space-y-1">
            <Label htmlFor="oob-import-source" className="text-xs">
              {t("sourceFile")}
            </Label>
            <Input
              id="oob-import-source"
              className="font-mono text-xs"
              value={sourceFile}
              onChange={(event) => setSourceFile(event.target.value)}
            />
          </div>
          <div className="space-y-1">
            <Label htmlFor="oob-import-tag" className="text-xs">
              {t("countryTag")}
            </Label>
            <Input
              id="oob-import-tag"
              className="font-mono text-xs uppercase"
              maxLength={10}
              value={countryTag}
              onChange={(event) => setCountryTag(event.target.value)}
            />
          </div>
        </div>

        <div className="space-y-1">
          <Label htmlFor="oob-import-script" className="text-xs">
            {t("scriptText")}
          </Label>
          <Textarea
            id="oob-import-script"
            className="min-h-40 font-mono text-xs"
            value={text}
            onChange={(event) => setText(event.target.value)}
          />
        </div>

        <div className="flex items-center gap-3">
          <Label htmlFor="oob-import-policy" className="text-xs">
            {t("policy")}
          </Label>
          <select
            id="oob-import-policy"
            className="h-9 rounded-md border border-input bg-card px-3 text-xs ui-focus-ring"
            value={policy}
            onChange={(event) =>
              setPolicy(event.target.value as "overwrite" | "skip")
            }
          >
            <option value="overwrite">{t("policyOverwrite")}</option>
            <option value="skip">{t("policySkip")}</option>
          </select>
          <Button
            size="sm"
            className="ml-auto gap-2 ui-interactive"
            disabled={
              submitting ||
              !text.trim() ||
              !oobFile.trim() ||
              !sourceFile.trim()
            }
            onClick={() => void handleImport()}
          >
            <FileUp className="h-4 w-4" aria-hidden />
            {submitting ? t("importing") : t("importSubmit")}
          </Button>
        </div>

        {error && (
          <p
            role="alert"
            className="border-y border-destructive/20 bg-destructive/10 px-3 py-2 text-xs text-destructive"
          >
            {t("importError", { error })}
          </p>
        )}
        {result && (
          <div className="border-y border-success/20 bg-success/10 px-3 py-2 text-xs text-success">
            <p>
              {t("importResult", {
                groups: result.air_wing_groups,
                wings: result.air_wings,
                fleets: result.fleets,
                forces: result.task_forces,
                ships: result.ships,
                skipped: result.skipped,
              })}
            </p>
            {result.warnings.length > 0 && (
              <ul className="mt-2 list-disc space-y-1 pl-4 text-warning">
                {result.warnings.map((warning, index) => (
                  <li key={`${index}-${warning}`}>
                    {localizeOobWarning(warning, t)}
                  </li>
                ))}
              </ul>
            )}
          </div>
        )}
      </div>
    </section>
  );
}
