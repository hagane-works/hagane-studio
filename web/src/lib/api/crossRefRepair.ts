import { request } from "./_client";
import { revisionRequest } from "@/components/workbench/save-diagnostics/revisionRequest";
import type { EditRevisions } from "@/components/workbench/save-diagnostics/editRevisions";

export type CrossRefRepairEntityKind = "focus" | "event" | "decision" | "idea";

export interface CrossRefRepairSourceSpan {
  start_offset: number;
  end_offset: number;
  start_line: number;
  start_column: number;
  end_line: number;
  end_column: number;
}

export interface CrossRefRepairRequest {
  entity_kind: CrossRefRepairEntityKind;
  entity_id: number;
  field: string;
  target_id: string;
  target_start_offset: number;
  target_end_offset: number;
  replacement_id?: string | null;
  delete_assignment?: boolean;
  source_sha256?: string | null;
}

export interface CrossRefRepairPreview {
  edit_revisions?: EditRevisions;
  entity_kind: CrossRefRepairEntityKind;
  entity_id: number;
  field: string;
  target_id: string;
  action: "replace_target" | "delete_assignment";
  applicable: boolean;
  before: string;
  after: string;
  source_span: CrossRefRepairSourceSpan;
  source_sha256: string;
  reason: string;
  replacement_id: string | null;
  boolean_ancestry: string[];
  applied: false;
}

export interface CrossRefRepairApplyResult extends Omit<
  CrossRefRepairPreview,
  "applied"
> {
  applied: true;
  new_source_sha256: string;
  remaining_target_count: number;
  reparse_ok: true;
}

export type CrossRefRepairApplyRequest = CrossRefRepairRequest & {
  source_sha256: string;
};

export const crossRefRepairsApi = {
  preview: (projectId: number, body: CrossRefRepairRequest) =>
    request<CrossRefRepairPreview>(
      `/go/projects/${projectId}/cross-ref-repairs/preview`,
      { method: "POST", body: JSON.stringify(body) },
    ),

  apply: (
    projectId: number,
    body: CrossRefRepairApplyRequest,
    revisions: EditRevisions = {},
  ) =>
    revisionRequest<CrossRefRepairApplyResult>(
      projectId,
      `/go/projects/${projectId}/cross-ref-repairs/apply`,
      { method: "POST", body: JSON.stringify(body) },
      revisions,
    ),
};
