import type { CtdIssue } from "@/lib/api/export";

export const NATIVE_WARNING_CODE = "export.native_warning";

export function nativeWarningIssues(
  warnings: readonly string[] | null | undefined,
  label?: string,
): CtdIssue[] {
  if (!warnings || warnings.length === 0) return [];
  const out: CtdIssue[] = [];
  for (const raw of warnings) {
    const text = String(raw ?? "").trim();
    if (!text) continue;
    out.push({
      code: NATIVE_WARNING_CODE,
      entity_ref: null,
      severity: "info",
      message: label ? `${label}: ${text}` : text,
      fix_hint: "",
      source: "go_native_export",
      detail: { native_warning: text },
    });
  }
  return out;
}
