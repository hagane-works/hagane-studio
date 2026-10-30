import { beforeEach, describe, expect, it, vi } from "vitest";

vi.mock("./_client", () => ({
  request: vi.fn(),
}));

import { request } from "./_client";
import {
  panelContractApi,
  type PanelContractFinding,
  type PanelContractReport,
} from "./panelContract";

function finding(
  code: string,
  severity: PanelContractFinding["severity"],
): PanelContractFinding {
  return {
    severity,
    code,
    title: code,
    reason: "reason",
    suggestion: "suggestion",
    entity_ref: null,
    file: null,
    detail: {},
  };
}

function report(findings: PanelContractFinding[]): PanelContractReport {
  return {
    findings,
    count: findings.length,
    scanned_panels: 2,
    totals: { error: 0, warning: 0, info: 0 },
  };
}

describe("panel-contract scan", () => {
  beforeEach(() => vi.mocked(request).mockReset());

  it("asks only the Go core and returns its report", async () => {
    const core = report([
      finding("panel.token_collision", "error"),
      finding("panel.sprite_empty", "info"),
    ]);
    vi.mocked(request).mockResolvedValueOnce(core);

    const scanned = await panelContractApi.scan(182);

    expect(vi.mocked(request).mock.calls.map(([path]) => path)).toEqual([
      "/go/lint/panel-contract-core/182",
    ]);
    expect(scanned).toBe(core);
  });
});
