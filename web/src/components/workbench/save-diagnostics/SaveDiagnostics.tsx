"use client";

import { useFindingCopy } from "@/components/export/FindingCopy";
import { EXPORT_FINDINGS } from "@/lib/exportFindingsMap";
import { GATE_PARITY_FINDINGS } from "@/lib/gateParityFindingMetadata";
import {
  currentSaveDiagnostics,
  type SaveDiagnostic,
} from "@/lib/saveDiagnostics";

export function SaveDiagnostics({
  diagnostics,
}: {
  diagnostics?: readonly SaveDiagnostic[];
}) {
  const copyFor = useFindingCopy();
  const current = currentSaveDiagnostics(diagnostics ?? []);
  if (!current.length) return null;
  return (
    <ul
      className="space-y-2 border-l-2 border-warning/40 pl-3 text-xs"
      aria-live="polite"
    >
      {current.map((item) => {
        const copy = copyFor({
          issue: {
            code: item.code,
            severity: item.severity === "info" ? "info" : "warning",
            message: item.code,
            entity_ref: item.entity_ref,
            evidence: item.evidence,
            detail: item.evidence,
            fix_hint: item.repair_route,
          },
          meta:
            EXPORT_FINDINGS[item.code] ??
            GATE_PARITY_FINDINGS[item.code] ??
            null,
        });
        return (
          <li key={item.key} className="space-y-1">
            <p className="font-medium text-foreground">{copy.what}</p>
            {copy.impact && (
              <p className="text-muted-foreground">{copy.impact}</p>
            )}
            {copy.action && (
              <p className="text-muted-foreground">{copy.action}</p>
            )}
            {copy.jump && <p className="text-muted-foreground">{copy.jump}</p>}
          </li>
        );
      })}
    </ul>
  );
}
