import type { CtdIssue } from "@/lib/api/export";
import {
  groupByFindingCode,
  presentCheckIssues,
  type CodeGroup,
  type PresentedFinding,
} from "@/lib/exportCheckPresentation";
import { TYPE_TO_ROUTE } from "@/components/workbench/mod-check/validationIssuePresentation";
import { gateParityRepairRoute } from "@/lib/gateParityFindingMetadata";

export interface RepairQueueItem {
  key: string;
  finding: PresentedFinding;

  resolved: boolean;

  addedByRecheck: boolean;
}

export interface RepairReconcileResult {
  items: RepairQueueItem[];

  newlyResolved: string[];

  added: string[];
}

export function repairItemKey(issue: CtdIssue): string {
  const ref = issue.entity_ref;
  const detail = issue.detail ?? {};
  const detailTarget =
    detail.reference ?? detail.prereq ?? detail.mutex ?? detail.focus_id ?? "";
  return [
    String(issue.code ?? ""),
    String(ref?.type ?? ""),
    String(ref?.id ?? ""),
    String(detailTarget ?? ""),
  ].join("\u0000");
}

export function repairHash(issue: CtdIssue): string {
  const code = String(issue.code ?? "");
  const rawId = issue.entity_ref?.id;
  const id = rawId === undefined || rawId === null ? "" : String(rawId);
  return id === "" ? code : `${code}-${encodeURIComponent(id)}`;
}

function hashCandidates(issue: CtdIssue): string[] {
  const code = String(issue.code ?? "");
  const rawId = issue.entity_ref?.id;
  const id = rawId === undefined || rawId === null ? "" : String(rawId);
  if (id === "") return [code];
  return [`${code}-${encodeURIComponent(id)}`, `${code}-${id}`];
}

export function normalizeRepairHash(hash: string | null | undefined): string {
  return String(hash ?? "")
    .replace(/^#/, "")
    .trim();
}

export function findRepairIndexByHash(
  items: RepairQueueItem[],
  hash: string | null | undefined,
): number {
  const wanted = normalizeRepairHash(hash);
  if (!wanted) return -1;
  const exact = items.findIndex((item) =>
    hashCandidates(item.finding.issue).includes(wanted),
  );
  if (exact >= 0) return exact;
  return items.findIndex((item) => String(item.finding.issue.code) === wanted);
}

export function repairRoutePath(issue: CtdIssue): string | null {
  const ref = issue.entity_ref;
  if (!ref) return null;
  return (
    gateParityRepairRoute(issue.code, String(ref.type)) ??
    TYPE_TO_ROUTE[String(ref.type)] ??
    null
  );
}

function withKeys(findings: PresentedFinding[]): RepairQueueItem[] {
  const seen = new Map<string, number>();
  return findings.map((finding) => {
    const base = repairItemKey(finding.issue);
    const nth = seen.get(base) ?? 0;
    seen.set(base, nth + 1);
    return {
      key: nth === 0 ? base : `${base}\u0000#${nth}`,
      finding,
      resolved: false,
      addedByRecheck: false,
    };
  });
}

export function buildRepairQueue(issues: CtdIssue[]): RepairQueueItem[] {
  return withKeys(presentCheckIssues(issues).attention);
}

export function reconcileRepairQueue(
  previous: RepairQueueItem[],
  issues: CtdIssue[],
): RepairReconcileResult {
  const fresh = buildRepairQueue(issues);
  const freshByKey = new Map(fresh.map((item) => [item.key, item]));
  const items: RepairQueueItem[] = [];
  const newlyResolved: string[] = [];
  const added: string[] = [];
  const seen = new Set<string>();

  for (const prev of previous) {
    seen.add(prev.key);
    const match = freshByKey.get(prev.key);
    if (match) {
      items.push({ ...prev, finding: match.finding, resolved: false });
      continue;
    }
    if (!prev.resolved) newlyResolved.push(prev.key);
    items.push({ ...prev, resolved: true });
  }

  for (const item of fresh) {
    if (seen.has(item.key)) continue;
    seen.add(item.key);
    added.push(item.key);
    items.push({ ...item, addedByRecheck: true });
  }

  return { items, newlyResolved, added };
}

export interface RepairQueueGroup extends CodeGroup<RepairQueueItem> {
  resolved: boolean;

  openCount: number;
}

export function countGroupsWithEntryPoint(
  groups: readonly CodeGroup<PresentedFinding>[],
): number {
  return groups.filter(
    (group) => repairRoutePath(group.representative.issue) !== null,
  ).length;
}

export function groupRepairQueue(
  items: readonly RepairQueueItem[],
): RepairQueueGroup[] {
  return groupByFindingCode(items, (item) => item.finding).map((group) => {
    const openCount = group.rows.filter((item) => !item.resolved).length;
    return { ...group, resolved: openCount === 0, openCount };
  });
}

export interface RepairProgress {
  total: number;
  done: number;
  remaining: number;

  percent: number;
}

export function repairProgress(items: RepairQueueItem[]): RepairProgress {
  const total = items.length;
  const done = items.filter((item) => item.resolved).length;
  return {
    total,
    done,
    remaining: total - done,
    percent: total === 0 ? 0 : Math.round((done / total) * 100),
  };
}

export function nextUnresolvedIndex(
  items: RepairQueueItem[],
  from = 0,
): number {
  if (items.length === 0) return -1;
  const start = Math.max(0, Math.min(from, items.length - 1));
  for (let offset = 0; offset < items.length; offset += 1) {
    const index = (start + offset) % items.length;
    if (!items[index].resolved) return index;
  }
  return -1;
}
