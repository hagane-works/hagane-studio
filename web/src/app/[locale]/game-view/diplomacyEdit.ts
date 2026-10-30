import type { HistoryFileRow } from "@/lib/api/scenario";

export type DiplomacyFields = Pick<
  HistoryFileRow,
  | "give_guarantee_tags"
  | "add_to_faction_tags"
  | "set_autonomy"
  | "diplomatic_relation"
>;

export const RELATION_TYPES = [
  "non_aggression_pact",
  "military_access",
  "guarantee",
  "docking_rights",
] as const;

const TAG = /^[A-Z][A-Z0-9]{2}$/;

const tagOf = (tag: string) => tag.trim().toUpperCase();

export type DiplomacyProblem =
  | { key: "notTags"; values: { tags: string } }
  | { key: "self" | "subjectAutonomy" | "relationType"; values?: undefined };

export function diplomacyProblem(
  fields: DiplomacyFields,
  self: string,
): DiplomacyProblem | null {
  const guarantees = (fields.give_guarantee_tags ?? []).map(tagOf);
  const subjects = (fields.set_autonomy ?? []).map((s) => tagOf(s.target));
  const relations = (fields.diplomatic_relation ?? []).map((r) =>
    tagOf(r.country),
  );
  const tags = [
    ...guarantees,
    ...(fields.add_to_faction_tags ?? []).map(tagOf),
    ...subjects,
    ...relations,
  ];
  const bad = tags.filter((tag) => !TAG.test(tag));
  if (bad.length) return { key: "notTags", values: { tags: bad.join(", ") } };

  if ([...guarantees, ...subjects, ...relations].includes(self.toUpperCase()))
    return { key: "self" };
  if ((fields.set_autonomy ?? []).some((s) => !s.autonomous_state.trim()))
    return { key: "subjectAutonomy" };
  if ((fields.diplomatic_relation ?? []).some((r) => !r.relation.trim()))
    return { key: "relationType" };
  return null;
}
