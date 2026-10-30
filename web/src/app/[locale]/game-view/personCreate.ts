import type { Character, RoleType } from "@/types";

export const PERSON_KINDS: readonly {
  value: RoleType;
  labelKey: "advisor" | "general" | "fieldMarshal" | "admiral" | "leader";
}[] = [
  { value: "advisor", labelKey: "advisor" },
  { value: "general", labelKey: "general" },
  { value: "field_marshal", labelKey: "fieldMarshal" },
  { value: "admiral", labelKey: "admiral" },
  { value: "country_leader", labelKey: "leader" },
];

export const NEW_ADVISOR_COST = 150;

export interface NewPerson {
  tag: string;
  name: string;
  kind: RoleType;

  slot: string;

  ideology: string;
}

export function personToken(tag: string, name: string): string {
  return `${tag.toUpperCase()}_${slugOf(name) || "new_person"}`;
}

export function slugOf(name: string): string {
  return name
    .normalize("NFKD")
    .replace(/[\u0300-\u036f]/g, "")
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, "_")
    .replace(/^_+|_+$/g, "");
}

export type NewPersonProblem =
  "nameRequired" | "nameInvalid" | "slotRequired" | "ideologyRequired";

export function newPersonProblem(person: NewPerson): NewPersonProblem | null {
  const name = person.name.trim();
  if (!name) return "nameRequired";
  if (/["\r\n]/.test(name)) return "nameInvalid";
  if (person.kind === "advisor" && !person.slot) return "slotRequired";
  if (person.kind === "country_leader" && !person.ideology)
    return "ideologyRequired";
  return null;
}

export function newPersonRow(person: NewPerson): Partial<Character> {
  const name = person.name.trim();
  const advisor = person.kind === "advisor";
  return {
    name,
    name_en: name,
    tag: personToken(person.tag, name),
    localisation_key: "",
    role_type: person.kind,
    has_advisor_role: advisor,
    country_tag: person.tag.toUpperCase(),
    ...(advisor
      ? { advisor_slot: person.slot, advisor_cost: NEW_ADVISOR_COST }
      : {}),
    ...(person.kind === "country_leader" ? { ideology: person.ideology } : {}),
  };
}

export function personDeletable(
  row: Pick<Character, "replaces_vanilla_id" | "source_file">,
): boolean {
  return !row.replaces_vanilla_id && !row.source_file;
}
