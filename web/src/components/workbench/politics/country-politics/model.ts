import type { Character } from "@/types";
import type { HistoryFileRow } from "@/lib/api";
import type { IdeologyOption } from "@/lib/ideology/registry";

export interface CountryPoliticsDraft {
  rulingParty: string;
  lastElection: string;
  electionFrequency: number;
  electionsAllowed: boolean | null;
  popularities: Record<string, number>;
}

export function historyToPoliticsDraft(
  history: HistoryFileRow | null,
  roots: readonly IdeologyOption[],
): CountryPoliticsDraft {
  const stored = history?.set_popularities ?? {};
  return {
    rulingParty: history?.set_politics_ruling_party ?? "",
    lastElection: history?.set_politics_last_election ?? "",
    electionFrequency: history?.set_politics_election_frequency ?? 0,
    electionsAllowed: history?.set_politics_elections_allowed ?? null,
    popularities: Object.fromEntries(
      roots.map((root) => [root.value, Number(stored[root.value] ?? 0)]),
    ),
  };
}

export function politicsHistoryPayload(
  draft: CountryPoliticsDraft,
): Pick<
  HistoryFileRow,
  | "set_politics_ruling_party"
  | "set_politics_last_election"
  | "set_politics_election_frequency"
  | "set_politics_elections_allowed"
  | "set_popularities"
> {
  return {
    set_politics_ruling_party: draft.rulingParty,
    set_politics_last_election: draft.lastElection,
    set_politics_election_frequency: draft.electionFrequency,
    set_politics_elections_allowed: draft.electionsAllowed,

    set_popularities: Object.fromEntries(
      Object.entries(draft.popularities).filter(([, support]) => support !== 0),
    ),
  };
}

export function leaderSubtypeDrafts(
  leaders: readonly Pick<Character, "id" | "ideology">[],
): Record<number, string> {
  return Object.fromEntries(
    leaders.map((leader) => [leader.id, leader.ideology]),
  );
}

export function validatePopularities(values: Record<string, number>): {
  valid: boolean;
  total: number;
  reason: "range" | "total" | null;
} {
  const numbers = Object.values(values);
  const total = numbers.reduce((sum, value) => sum + Number(value || 0), 0);
  if (
    numbers.some(
      (value) => !Number.isInteger(value) || value < 0 || value > 100,
    )
  ) {
    return { valid: false, total, reason: "range" };
  }
  if (numbers.some((value) => value !== 0) && total !== 100) {
    return { valid: false, total, reason: "total" };
  }
  return { valid: true, total, reason: null };
}

export interface PopularityPieSegment {
  id: string;
  value: number;
  share: number;
  offset: number;
}

export function popularityPieSegments(
  popularities: Record<string, number>,
  rootIds: readonly string[],
): PopularityPieSegment[] {
  const values = rootIds.map((id) => ({
    id,
    value: Number.isFinite(popularities[id])
      ? Math.max(0, popularities[id] ?? 0)
      : 0,
  }));
  const total = values.reduce((sum, item) => sum + item.value, 0);
  let offset = 0;
  return values.map((item) => {
    const share = total > 0 ? (item.value / total) * 100 : 0;
    const segment = { ...item, share, offset };
    offset += share;
    return segment;
  });
}

export function subtypeRoot(
  subtypeId: string,
  subtypes: readonly IdeologyOption[],
): string {
  return subtypes.find((subtype) => subtype.value === subtypeId)?.group ?? "";
}

export function countryLeaders(
  characters: readonly Character[],
  countryTag: string,
): Character[] {
  return characters.filter(
    (character) =>
      character.role_type === "country_leader" &&
      character.country_tag.toUpperCase() === countryTag.toUpperCase(),
  );
}

export type CountryLeaderRole = "current" | "candidate";

export interface CountryLeaderClassification {
  currentLeaderId: number | null;
  roleByLeaderId: Readonly<Record<number, CountryLeaderRole>>;
}

export function classifyCountryLeaders(
  leaders: readonly Pick<Character, "id" | "ideology">[],
  savedRulingRoot: string,
  subtypes: readonly IdeologyOption[],
): CountryLeaderClassification {
  const matchingLeaderIds = savedRulingRoot
    ? leaders
        .filter(
          (leader) =>
            subtypeRoot(leader.ideology, subtypes) === savedRulingRoot,
        )
        .map((leader) => leader.id)
    : [];
  const currentLeaderId =
    matchingLeaderIds.length === 1 ? matchingLeaderIds[0] : null;

  return {
    currentLeaderId,
    roleByLeaderId: Object.fromEntries(
      leaders.map((leader) => [
        leader.id,
        leader.id === currentLeaderId ? "current" : "candidate",
      ]),
    ),
  };
}

export function isCurrentLeaderRulingMismatch(
  classification: CountryLeaderClassification,
  leaderId: number,
  draftedLeaderRoot: string,
  draftedRulingRoot: string,
): boolean {
  return Boolean(
    classification.currentLeaderId === leaderId &&
    draftedLeaderRoot &&
    draftedRulingRoot &&
    draftedLeaderRoot !== draftedRulingRoot,
  );
}

export function politicsReadbackMatches(
  row: HistoryFileRow,
  draft: CountryPoliticsDraft,
): boolean {
  const canonicalPopularities = row.set_popularities ?? {};
  return (
    row.set_politics_ruling_party === draft.rulingParty &&
    row.set_politics_last_election === draft.lastElection &&
    row.set_politics_election_frequency === draft.electionFrequency &&
    row.set_politics_elections_allowed === draft.electionsAllowed &&
    Object.entries(draft.popularities).every(
      ([root, support]) => Number(canonicalPopularities[root] ?? 0) === support,
    )
  );
}
