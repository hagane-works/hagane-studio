import { request } from "./_client";

export type PanelContractSeverity = "error" | "warning" | "info";

export interface PanelContractEntityRef {
  kind: string;
  entity_id: string | null;
  name: string | null;
  country_tag: string | null;
}

export interface PanelContractFinding {
  severity: PanelContractSeverity;
  code: string;
  title: string;
  reason: string;
  suggestion: string;
  entity_ref: PanelContractEntityRef | null;
  file: string | null;
  detail: Record<string, unknown>;
}

export interface PanelContractReport {
  findings: PanelContractFinding[];
  totals: Record<PanelContractSeverity, number>;
  count: number;
  scanned_panels?: number;
}

export const panelContractApi = {
  scan: (projectId: number) =>
    request<PanelContractReport>(`/go/lint/panel-contract-core/${projectId}`),
};
