import type {
  StateBulkDetail,
  StateBulkResult,
  StateRestoreEntry,
} from "@/types";

export interface BulkRemovedTag {
  tag: string;

  count: number;
}

export interface BulkPreviewSummary {
  changed: number;

  skipped: number;

  ownerChanged: number;

  coresChanged: number;

  removedTags: BulkRemovedTag[];
}

function sameList(a: readonly string[], b: readonly string[]): boolean {
  if (a.length !== b.length) return false;
  return a.every((v, i) => v === b[i]);
}

export function summarizeBulk(
  result: StateBulkResult | null,
): BulkPreviewSummary {
  const empty: BulkPreviewSummary = {
    changed: 0,
    skipped: 0,
    ownerChanged: 0,
    coresChanged: 0,
    removedTags: [],
  };
  if (!result) return empty;

  const removed = new Map<string, number>();
  let ownerChanged = 0;
  let coresChanged = 0;

  for (const d of result.details) {
    if (!d.changed) continue;
    if (d.before_owner !== d.after_owner) ownerChanged += 1;
    if (!sameList(d.before_cores ?? [], d.after_cores ?? [])) coresChanged += 1;
    for (const tag of d.removed_cores ?? []) {
      removed.set(tag, (removed.get(tag) ?? 0) + 1);
    }
  }

  const removedTags = [...removed.entries()]
    .map(([tag, count]) => ({ tag, count }))

    .sort((a, b) => b.count - a.count || a.tag.localeCompare(b.tag));

  return {
    changed: result.changed,
    skipped: result.skipped,
    ownerChanged,
    coresChanged,
    removedTags,
  };
}

export function toRestoreEntries(
  result: StateBulkResult | null,
): StateRestoreEntry[] {
  if (!result) return [];
  return result.details
    .filter((d: StateBulkDetail) => d.changed)

    .map((d) => ({
      id: d.id,
      owner: d.before_owner,
      add_core_of: d.before_cores ?? [],
      controller: d.before_controller ?? "",
      add_claim_by: d.before_claims ?? [],
      remove_core_of: d.before_remove_cores ?? [],
      ...(d.before_source ? { source: d.before_source } : {}),
    }));
}

export interface BulkCurrentValues {
  owners: { tag: string; count: number }[];

  cores: string[];
}

export function summarizeCurrentValues(
  rows: { owner?: string | null; add_core_of?: string[] | null }[],
): BulkCurrentValues {
  const ownerCount = new Map<string, number>();
  const cores = new Set<string>();
  for (const r of rows) {
    const owner = (r.owner ?? "").trim();
    ownerCount.set(owner, (ownerCount.get(owner) ?? 0) + 1);
    for (const c of r.add_core_of ?? []) {
      const tag = (c ?? "").trim();
      if (tag) cores.add(tag);
    }
  }
  const owners = [...ownerCount.entries()]
    .map(([tag, count]) => ({ tag, count }))
    .sort((a, b) => b.count - a.count || a.tag.localeCompare(b.tag));
  return { owners, cores: [...cores].sort() };
}
