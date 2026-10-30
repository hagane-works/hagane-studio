import type { State, StateUpdate } from "@/types";

import { normalizeInnerList } from "./rawInnerList";

export interface ProvinceBuildingRow {
  province_id: string;
  building: string;
  level: string;
}

export interface StateFormDraft {
  provinces: string;
  manpower: string;
  state_category: string;

  displayNameLocs: Record<string, string>;
  owner: string;
  controller: string;
  impassable: boolean;

  impassable_ignored_links: string;

  force_link_ownership_to: string;
  local_supplies: string;
  buildings_max_level_factor: string;

  resourceRows: [string, string][];
  buildingRows: [string, string][];
  cores: string[];
  claims: string[];
  removeCores: string[];
  victoryPoints: { province_id: string; value: string }[];
  provinceControllers: { province_id: string; controller: string }[];

  provinceBuildings: ProvinceBuildingRow[];
}

export interface ProvinceDraftValidation {
  values: number[];
  invalidTokens: string[];
  duplicateIds: number[];
  valid: boolean;
  empty: boolean;
}

export function validateProvinceDraft(raw: string): ProvinceDraftValidation {
  const tokens = raw.trim() ? raw.trim().split(/\s+/) : [];
  const values: number[] = [];
  const invalidTokens: string[] = [];
  const duplicateIds = new Set<number>();
  const seen = new Set<number>();
  for (const token of tokens) {
    if (!/^[1-9]\d*$/.test(token)) {
      invalidTokens.push(token);
      continue;
    }
    const value = Number(token);
    if (!Number.isSafeInteger(value)) {
      invalidTokens.push(token);
      continue;
    }
    if (seen.has(value)) duplicateIds.add(value);
    else seen.add(value);
    values.push(value);
  }
  return {
    values,
    invalidTokens,
    duplicateIds: [...duplicateIds].sort((a, b) => a - b),
    valid: invalidTokens.length === 0 && duplicateIds.size === 0,
    empty: tokens.length === 0,
  };
}

function provinceDraftFingerprint(raw: string): string {
  const validation = validateProvinceDraft(raw);
  return validation.valid
    ? validation.values.join(" ")
    : raw.trim().replace(/\s+/g, " ");
}

export function resolvedStateName(state: State, lang: string): string {
  return (
    (state.display_name ?? {})[lang] ??
    (state.display_name_locs ?? {})[lang] ??
    ""
  );
}

export function seedStateForm(state: State): StateFormDraft {
  return {
    provinces: (state.provinces ?? []).join(" "),
    manpower: String(state.manpower ?? 0),
    state_category: state.state_category ?? "",
    displayNameLocs: { ...(state.display_name_locs ?? {}) },
    owner: state.owner ?? "",
    controller: state.controller ?? "",
    impassable: state.impassable ?? false,
    impassable_ignored_links: state.impassable_ignored_links ?? "",
    force_link_ownership_to:
      state.force_link_ownership_to == null
        ? ""
        : String(state.force_link_ownership_to),
    local_supplies:
      state.local_supplies == null ? "" : String(state.local_supplies),
    buildings_max_level_factor:
      state.buildings_max_level_factor == null
        ? ""
        : String(state.buildings_max_level_factor),
    resourceRows: Object.entries(state.resources ?? {}).map(
      ([k, v]) => [k, String(v)] as [string, string],
    ),
    buildingRows: Object.entries(state.buildings ?? {}).map(
      ([k, v]) => [k, String(v)] as [string, string],
    ),
    cores: [...(state.add_core_of ?? [])],
    claims: [...(state.add_claim_by ?? [])],
    removeCores: [...(state.remove_core_of ?? [])],
    victoryPoints: (state.victory_points ?? []).map((vp) => ({
      province_id: String(vp.province_id),
      value: String(vp.value),
    })),
    provinceControllers: (state.set_province_controller ?? []).map((pc) => ({
      province_id: String(pc.province_id),
      controller: pc.controller ?? "",
    })),

    provinceBuildings: Object.entries(state.province_buildings ?? {}).flatMap(
      ([pid, blk]) =>
        Object.entries(blk ?? {}).map((entry) => ({
          province_id: pid,
          building: entry[0],
          level: String(entry[1]),
        })),
    ),
  };
}

export function stateDirtyFingerprint(d: StateFormDraft): string {
  return JSON.stringify({
    provinces: provinceDraftFingerprint(d.provinces),
    manpower: d.manpower.trim(),
    state_category: d.state_category.trim(),
    displayNameLocs: Object.fromEntries(
      Object.entries(d.displayNameLocs).map(([lang, value]) => [
        lang,
        value.trim(),
      ]),
    ),
    owner: d.owner.trim(),
    controller: d.controller.trim(),
    impassable: d.impassable,

    impassable_ignored_links: normalizeInnerList(d.impassable_ignored_links),
    force_link_ownership_to: d.force_link_ownership_to.trim(),
    local_supplies: d.local_supplies.trim(),
    buildings_max_level_factor: d.buildings_max_level_factor.trim(),

    resourceRows: d.resourceRows.filter(([k]) => k.trim()),
    buildingRows: d.buildingRows.filter(([k]) => k.trim()),
    cores: d.cores.filter((c) => c.trim()),
    claims: d.claims.filter((c) => c.trim()),
    removeCores: d.removeCores.filter((c) => c.trim()),
    victoryPoints: d.victoryPoints.filter((vp) => vp.province_id.trim()),
    provinceControllers: d.provinceControllers.filter(
      (pc) => pc.province_id.trim() && pc.controller.trim(),
    ),

    provinceBuildings: d.provinceBuildings.filter(
      (r) => r.province_id.trim() && r.building.trim(),
    ),
  });
}

export function isStateFormDirty(d: StateFormDraft, state: State): boolean {
  return (
    stateDirtyFingerprint(d) !== stateDirtyFingerprint(seedStateForm(state))
  );
}

export function draftToStateUpdate(
  d: StateFormDraft,
  original?: State,
): StateUpdate {
  const provinceValidation = validateProvinceDraft(d.provinces);
  if (!provinceValidation.valid) {
    throw new Error("province membership draft is invalid");
  }

  const resources: Record<string, number> = {};
  for (const [k, v] of d.resourceRows) {
    const key = k.trim();
    if (key) resources[key] = Math.trunc(Number(v) || 0);
  }
  const buildings: Record<string, number> = {};
  for (const [k, v] of d.buildingRows) {
    const key = k.trim();
    if (key) buildings[key] = Math.trunc(Number(v) || 0);
  }
  const ls = d.local_supplies.trim();
  const bmlf = d.buildings_max_level_factor.trim();

  const iil = normalizeInnerList(d.impassable_ignored_links);

  const forceLinkRaw = d.force_link_ownership_to.trim();
  const forceLink = /^-?\d+$/.test(forceLinkRaw) ? Number(forceLinkRaw) : null;

  const province_buildings: Record<string, Record<string, number>> = {};
  for (const r of d.provinceBuildings) {
    const pid = r.province_id.trim();
    const bld = r.building.trim();
    if (!pid || !bld) continue;
    (province_buildings[pid] ??= {})[bld] = Math.trunc(Number(r.level) || 0);
  }

  const includeImpassable =
    d.impassable === true || (original != null && original.impassable != null);

  const display_name_locs: Record<string, string> = Object.fromEntries(
    Object.entries(d.displayNameLocs)
      .map(([lang, value]) => [lang, value.trim()])
      .filter(([, value]) => Boolean(value)),
  );
  const out: StateUpdate = {
    provinces: provinceValidation.values,
    display_name_locs,
    manpower: Math.trunc(Number(d.manpower) || 0),
    state_category: d.state_category.trim(),
    owner: d.owner.trim(),
    controller: d.controller.trim() || null,
    local_supplies: ls === "" ? null : Number(ls),
    buildings_max_level_factor: bmlf === "" ? null : Number(bmlf),
    impassable_ignored_links: iil === "" ? null : iil,
    force_link_ownership_to: forceLink,
    resources,
    buildings,
    province_buildings,
    add_core_of: d.cores.map((c) => c.trim()).filter(Boolean),
    add_claim_by: d.claims.map((c) => c.trim()).filter(Boolean),
    remove_core_of: d.removeCores.map((c) => c.trim()).filter(Boolean),
    victory_points: d.victoryPoints
      .filter((vp) => vp.province_id.trim())
      .map((vp) => ({
        province_id: Math.trunc(Number(vp.province_id) || 0),
        value: Math.trunc(Number(vp.value) || 0),
      })),
    set_province_controller: d.provinceControllers
      .filter((pc) => pc.province_id.trim() && pc.controller.trim())
      .map((pc) => ({
        province_id: Math.trunc(Number(pc.province_id) || 0),
        controller: pc.controller.trim(),
      })),
  };
  if (includeImpassable) out.impassable = d.impassable;
  return out;
}
