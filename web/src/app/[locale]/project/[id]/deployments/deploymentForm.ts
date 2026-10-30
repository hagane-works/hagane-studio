import type { DivisionDeployment } from "@/types";
import { tagFromOob } from "@/lib/oob/oobDefaults";

export interface DepDraft {
  oob_file: string;
  country_tag: string;
  division_template: string;

  state_id: number | null;
  location: number | null;
  count: string;
  division_name: string;
  start_experience_factor: string;
  start_equipment_factor: string;
}

export function depToDraft(dep: DivisionDeployment): DepDraft {
  return {
    oob_file: dep.oob_file || "",
    country_tag: dep.country_tag || tagFromOob(dep.oob_file || ""),
    division_template: dep.division_template || "",
    state_id: null,
    location: dep.location,
    count: dep.count == null ? "1" : String(dep.count),
    division_name: dep.division_name || "",
    start_experience_factor:
      dep.start_experience_factor == null
        ? ""
        : String(dep.start_experience_factor),
    start_equipment_factor:
      dep.start_equipment_factor == null
        ? ""
        : String(dep.start_equipment_factor),
  };
}

export function draftToPayload(d: DepDraft): Partial<DivisionDeployment> {
  return {
    oob_file: d.oob_file.trim(),
    country_tag: d.country_tag.trim().toUpperCase(),
    division_template: d.division_template.trim(),
    location: d.location,
    count:
      d.count.trim() === "" ? 1 : Math.max(1, Math.trunc(Number(d.count)) || 1),
    division_name: d.division_name.trim(),
    start_experience_factor:
      d.start_experience_factor.trim() === ""
        ? null
        : Number(d.start_experience_factor),
    start_equipment_factor:
      d.start_equipment_factor.trim() === ""
        ? null
        : Number(d.start_equipment_factor),
  };
}
