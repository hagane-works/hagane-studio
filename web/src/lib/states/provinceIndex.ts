export interface IndexableState {
  id: number;

  state_id: number;

  name?: string | null;
  provinces?: number[] | null;
}

export interface ProvinceOwner {
  rowId: number;
  stateId: number;
  name: string;
}

export type ProvinceIndex = ReadonlyMap<number, readonly ProvinceOwner[]>;

export interface ProvinceCollision {
  provinceId: number;

  owners: ProvinceOwner[];
}

export interface DonorEdit {
  rowId: number;
  stateId: number;
  name: string;

  nextProvinces: number[];

  prevProvinces: number[];

  taken: number[];
}

export interface ProvinceMovePlan {
  targetRowId: number;
  targetStateId: number;
  targetName: string;
  targetNextProvinces: number[];
  targetPrevProvinces: number[];

  donors: DonorEdit[];

  movedProvinceIds: number[];
}

export function normalizeProvinces(
  ids: readonly number[] | null | undefined,
): number[] {
  const out: number[] = [];
  const seen = new Set<number>();
  for (const raw of ids ?? []) {
    if (typeof raw !== "number" || !Number.isFinite(raw)) continue;
    const id = Math.trunc(raw);
    if (seen.has(id)) continue;
    seen.add(id);
    out.push(id);
  }
  return out;
}

export function buildProvinceIndex(
  states: readonly IndexableState[],
): ProvinceIndex {
  const index = new Map<number, ProvinceOwner[]>();
  for (const st of states) {
    const owner: ProvinceOwner = {
      rowId: st.id,
      stateId: st.state_id,
      name: st.name ?? "",
    };
    for (const pid of normalizeProvinces(st.provinces)) {
      const bucket = index.get(pid);
      if (bucket) bucket.push(owner);
      else index.set(pid, [owner]);
    }
  }
  return index;
}

export function detectCollisions(index: ProvinceIndex): ProvinceCollision[] {
  const out: ProvinceCollision[] = [];
  for (const [provinceId, owners] of index) {
    if (owners.length < 2) continue;
    const byState = [...owners].sort((a, b) => a.stateId - b.stateId);
    out.push({ provinceId, owners: byState });
  }
  return out.sort((a, b) => a.provinceId - b.provinceId);
}

export function planProvinceMove(
  states: readonly IndexableState[],
  targetRowId: number,
  nextProvinces: readonly number[],
): ProvinceMovePlan | null {
  const target = states.find((s) => s.id === targetRowId);
  if (!target) return null;

  const next = normalizeProvinces(nextProvinces);
  const nextSet = new Set(next);
  const donors: DonorEdit[] = [];
  const moved = new Set<number>();

  for (const other of states) {
    if (other.id === targetRowId) continue;
    const prev = normalizeProvinces(other.provinces);
    const taken = prev.filter((pid) => nextSet.has(pid));
    if (taken.length === 0) continue;
    donors.push({
      rowId: other.id,
      stateId: other.state_id,
      name: other.name ?? "",
      prevProvinces: prev,
      nextProvinces: prev.filter((pid) => !nextSet.has(pid)),
      taken: [...taken].sort((a, b) => a - b),
    });
    for (const pid of taken) moved.add(pid);
  }

  return {
    targetRowId,
    targetStateId: target.state_id,
    targetName: target.name ?? "",
    targetNextProvinces: next,
    targetPrevProvinces: normalizeProvinces(target.provinces),
    donors: donors.sort((a, b) => a.stateId - b.stateId),
    movedProvinceIds: [...moved].sort((a, b) => a - b),
  };
}
