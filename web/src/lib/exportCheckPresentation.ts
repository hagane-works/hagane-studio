import { EXPORT_FINDINGS, type FindingMeta } from "@/lib/exportFindingsMap";
import type { CtdIssue } from "@/lib/api/export";
import {
  buildVanillaDlcNotice,
  type DlcNoticeState,
  type VanillaDlcNoticeOptions,
} from "@/lib/vanillaDlcNotice";
import {
  buildImpassableCategoryNotice,
  type ImpassableNoticeState,
} from "@/lib/impassableCategoryNotice";
import {
  buildImpassableForceLinkNotice,
  type ImpassableForceLinkState,
} from "@/lib/impassableForceLinkNotice";

export const MAP_INTEGRITY_LEGACY_UNVERIFIED_CODES: ReadonlySet<string> =
  new Set([
    "map_integrity.final_membership_not_verified",

    "map_integrity.final_membership_legacy_unverified",
    "map_integrity.final_membership_evidence_missing",
    "map_integrity.overlay_not_verified",
    "map_integrity.custom_map_unverified",
    "map_integrity.building_geometry_unverified",
    "map_integrity.verified_overlay_unavailable",
  ]);

export const RAW_ICU_FINDING_KEYS: ReadonlySet<string> = new Set([
  "clausewitz_unclosed_block",
  "clausewitz_stray_closing_brace",
]);

export interface PresentedFinding {
  issue: CtdIssue;

  meta: FindingMeta | null;
}

export interface LegacyMapAggregate {
  count: number;
  issues: CtdIssue[];
}

export interface CodeGroup<T> {
  code: string;
  rows: T[];

  representative: T;

  count: number;

  hasBlocker: boolean;
}

export type FindingGroup = CodeGroup<PresentedFinding>;

export interface CheckPresentation {
  attention: PresentedFinding[];

  attentionGroups: FindingGroup[];

  auto: PresentedFinding[];

  health: PresentedFinding[];

  pass: PresentedFinding[];

  legacyMap: LegacyMapAggregate | null;
}

const SEVERITY_RANK: Record<string, number> = {
  blocker: 0,
  warning: 1,
  info: 2,
};

function severityRank(issue: CtdIssue): number {
  return SEVERITY_RANK[String(issue.severity)] ?? 1;
}

export function legacyMapStateCount(issues: CtdIssue[]): number {
  const ids = new Set<number>();
  for (const issue of issues) {
    for (const bag of [issue.evidence, issue.detail]) {
      if (!bag || typeof bag !== "object") continue;
      const raw = (bag as Record<string, unknown>).state_ids;
      if (!Array.isArray(raw)) continue;
      for (const value of raw) {
        if (typeof value === "number" && Number.isFinite(value)) ids.add(value);
      }
    }
  }
  return ids.size > 0 ? ids.size : issues.length;
}

export function countDistinctEntities(issues: readonly CtdIssue[]): number {
  const seen = new Set<string>();
  let unattributed = 0;
  for (const issue of issues) {
    const ref = issue.entity_ref;
    const id = ref?.id;
    if (!ref || id === undefined || id === null || String(id) === "") {
      unattributed += 1;
      continue;
    }
    seen.add(`${String(ref.type ?? "")}\u0000${String(id)}`);
  }
  return seen.size + unattributed;
}

export function groupByFindingCode<T>(
  rows: readonly T[],
  toFinding: (row: T) => PresentedFinding,
): CodeGroup<T>[] {
  const order: string[] = [];
  const buckets = new Map<string, T[]>();
  for (const row of rows) {
    const code = String(toFinding(row).issue.code ?? "");
    const bucket = buckets.get(code);
    if (bucket) {
      bucket.push(row);
      continue;
    }
    buckets.set(code, [row]);
    order.push(code);
  }
  return order.map((code) => {
    const groupRows = buckets.get(code) as T[];
    const issues = groupRows.map((row) => toFinding(row).issue);
    return {
      code,
      rows: groupRows,
      representative: groupRows[0],
      count: countDistinctEntities(issues),
      hasBlocker: issues.some((issue) => String(issue.severity) === "blocker"),
    };
  });
}

export function groupAttentionFindings(
  findings: readonly PresentedFinding[],
): FindingGroup[] {
  return groupByFindingCode(findings, (finding) => finding);
}

export interface PresentCheckOptions extends VanillaDlcNoticeOptions {
  states?:
    | readonly (DlcNoticeState &
        ImpassableNoticeState &
        ImpassableForceLinkState)[]
    | null;
}

export function synthesizeClientFindings(
  issues: readonly CtdIssue[],
  options: PresentCheckOptions | undefined,
): CtdIssue[] {
  if (!options) return [];
  const backendCodes = new Set(issues.map((issue) => String(issue.code ?? "")));
  const synthesized: CtdIssue[] = [];

  const vanillaDlc = buildVanillaDlcNotice(options.states, options);
  if (vanillaDlc && !backendCodes.has(vanillaDlc.code)) {
    synthesized.push(vanillaDlc);
  }

  const impassableCategory = buildImpassableCategoryNotice(options.states);
  if (impassableCategory && !backendCodes.has(impassableCategory.code)) {
    synthesized.push(impassableCategory);
  }

  const impassableForceLink = buildImpassableForceLinkNotice(options.states);
  if (impassableForceLink && !backendCodes.has(impassableForceLink.code)) {
    synthesized.push(impassableForceLink);
  }
  return synthesized;
}

export function presentCheckIssues(
  issues: CtdIssue[],
  options?: PresentCheckOptions,
): CheckPresentation {
  const attention: PresentedFinding[] = [];
  const auto: PresentedFinding[] = [];
  const health: PresentedFinding[] = [];
  const pass: PresentedFinding[] = [];
  const legacyIssues: CtdIssue[] = [];

  const synthesized = synthesizeClientFindings(issues, options);
  const allIssues =
    synthesized.length > 0 ? [...issues, ...synthesized] : issues;

  for (const issue of allIssues) {
    const code = String(issue.code ?? "");
    if (MAP_INTEGRITY_LEGACY_UNVERIFIED_CODES.has(code)) {
      legacyIssues.push(issue);
      continue;
    }
    const meta = EXPORT_FINDINGS[code] ?? null;
    if (!meta) {
      if (issue.severity === "info") auto.push({ issue, meta: null });
      else attention.push({ issue, meta: null });
      continue;
    }
    switch (meta.tier) {
      case "attn":
        attention.push({ issue, meta });
        break;
      case "auto":
        auto.push({ issue, meta });
        break;
      case "health":
        health.push({ issue, meta });
        break;
      case "pass":
        pass.push({ issue, meta });
        break;
    }
  }

  attention.sort((a, b) => severityRank(a.issue) - severityRank(b.issue));

  return {
    attention,
    attentionGroups: groupAttentionFindings(attention),
    auto,
    health,
    pass,
    legacyMap:
      legacyIssues.length > 0
        ? { count: legacyMapStateCount(legacyIssues), issues: legacyIssues }
        : null,
  };
}

export function formatRawFindingTemplate(
  template: string,
  values: Record<string, string | number>,
): string {
  return template.replace(/\{(\w+)\}/g, (whole, name: string) =>
    name in values ? String(values[name]) : whole,
  );
}
