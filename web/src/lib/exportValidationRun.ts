import { ApiError } from "@/lib/api/_client";
import type { ExportValidationIssue } from "@/lib/api/export";

interface ValidationResponse<TSummary, TAssetAudit> {
  issues: ExportValidationIssue[];
  summary: TSummary;
  asset_audit?: TAssetAudit;
}

interface StructureResponse {
  files: string[];
  job_id?: string;
}

export interface ExportValidationRun<TSummary, TAssetAudit> {
  issues: ExportValidationIssue[];
  summary: TSummary;
  assetAudit: TAssetAudit | null;
  files: string[];
  structureStatus: "ready" | "blocked" | "failed";
  structureError: unknown | null;
  jobId: string | null;
}

function issueLevel(severity: unknown): string {
  return severity === "blocker"
    ? "error"
    : severity === "warning"
      ? "warning"
      : "info";
}

export function validationIssuesFromRequestError(
  error: unknown,
  fallbackMessage: string,
): ExportValidationIssue[] {
  if (
    error instanceof ApiError &&
    error.payload &&
    typeof error.payload === "object"
  ) {
    const payload = error.payload as {
      message?: unknown;
      blockers?: unknown;
    };
    if (Array.isArray(payload.blockers)) {
      const blockers = payload.blockers.flatMap((candidate) => {
        if (!candidate || typeof candidate !== "object") return [];
        const item = candidate as Record<string, unknown>;
        if (typeof item.message !== "string") return [];
        return [
          {
            ...item,
            level: issueLevel(item.severity),
            message: item.message,
          } as ExportValidationIssue,
        ];
      });
      if (blockers.length > 0) return blockers;
    }
    if (typeof payload.message === "string") {
      return [{ level: "error", message: payload.message }];
    }
  }
  return [{ level: "error", message: fallbackMessage }];
}

function mergeIssues(
  current: ExportValidationIssue[],
  recovered: ExportValidationIssue[],
): ExportValidationIssue[] {
  const fingerprint = (issue: ExportValidationIssue) =>
    JSON.stringify([
      issue.code ?? "",
      issue.message,
      issue.entity_ref ?? null,
      issue.source ?? null,
      issue.repair_route ?? null,
      issue.evidence ?? null,
    ]);
  const seen = new Set(current.map(fingerprint));
  return [
    ...current,
    ...recovered.filter((issue) => {
      const key = fingerprint(issue);
      if (seen.has(key)) return false;
      seen.add(key);
      return true;
    }),
  ];
}

export async function runExportValidationRequests<TSummary, TAssetAudit>(
  validate: () => Promise<ValidationResponse<TSummary, TAssetAudit>>,
  previewStructure: () => Promise<StructureResponse>,
  previewFailureMessage: (error: unknown) => string,
): Promise<ExportValidationRun<TSummary, TAssetAudit>> {
  const validation = await validate();
  if (validation.issues.some((issue) => issue.level === "error")) {
    return {
      issues: validation.issues,
      summary: validation.summary,
      assetAudit: validation.asset_audit ?? null,
      files: [],
      structureStatus: "blocked",
      structureError: null,
      jobId: null,
    };
  }

  try {
    const structure = await previewStructure();
    return {
      issues: validation.issues,
      summary: validation.summary,
      assetAudit: validation.asset_audit ?? null,
      files: structure.files,
      structureStatus: "ready",
      structureError: null,
      jobId: structure.job_id ?? null,
    };
  } catch (error) {
    return {
      issues: mergeIssues(
        validation.issues,
        validationIssuesFromRequestError(error, previewFailureMessage(error)),
      ),
      summary: validation.summary,
      assetAudit: validation.asset_audit ?? null,
      files: [],
      structureStatus: "failed",
      structureError: error,
      jobId: null,
    };
  }
}
