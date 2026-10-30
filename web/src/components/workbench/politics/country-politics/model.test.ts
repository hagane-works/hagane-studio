import { describe, expect, it } from "vitest";

import {
  classifyCountryLeaders,
  isCurrentLeaderRulingMismatch,
  popularityPieSegments,
  politicsHistoryPayload,
  subtypeRoot,
  validatePopularities,
} from "./model";

const subtypes = [
  { value: "liberal", label: "Liberal", group: "democratic" },
  { value: "conservative", label: "Conservative", group: "democratic" },
  { value: "fascist", label: "Fascist", group: "fascism" },
];

describe("country politics model", () => {
  it("accepts an arbitrary fifth ROOT when support totals 100", () => {
    expect(
      validatePopularities({ democratic: 40, military_government: 60 }),
    ).toEqual({ valid: true, total: 100, reason: null });
  });

  it("rejects fractional or incomplete support", () => {
    expect(validatePopularities({ democratic: 99.5 })).toMatchObject({
      valid: false,
      reason: "range",
    });
    expect(validatePopularities({ democratic: 70 })).toMatchObject({
      valid: false,
      reason: "total",
    });
  });

  it("writes an all-zero editor expansion as an empty popularity map", () => {
    const payload = politicsHistoryPayload({
      rulingParty: "",
      lastElection: "",
      electionFrequency: 1,
      electionsAllowed: null,
      popularities: {
        democratic: 0,
        communism: 0,
        fascism: 0,
        neutrality: 0,
      },
    });

    expect(payload.set_popularities).toEqual({});
  });

  it("builds dynamic pie segments in project ROOT order", () => {
    expect(
      popularityPieSegments(
        { democratic: 20, technocracy: 65, neutrality: 15 },
        ["democratic", "neutrality", "technocracy"],
      ),
    ).toEqual([
      { id: "democratic", value: 20, share: 20, offset: 0 },
      { id: "neutrality", value: 15, share: 15, offset: 20 },
      { id: "technocracy", value: 65, share: 65, offset: 35 },
    ]);
  });

  it("derives a leader subtype relationship without treating the subtype as ROOT", () => {
    expect(
      subtypeRoot("national_military_government", [
        {
          value: "national_military_government",
          label: "National military government",
          group: "military_government",
        },
      ]),
    ).toBe("military_government");
  });

  it("identifies the unique persisted leader under the saved ruling ROOT", () => {
    expect(
      classifyCountryLeaders(
        [
          { id: 11, ideology: "fascist" },
          { id: 22, ideology: "liberal" },
        ],
        "democratic",
        subtypes,
      ),
    ).toEqual({
      currentLeaderId: 22,
      roleByLeaderId: { 11: "candidate", 22: "current" },
    });
  });

  it("does not depend on API list order", () => {
    const forwards = classifyCountryLeaders(
      [
        { id: 11, ideology: "fascist" },
        { id: 22, ideology: "liberal" },
      ],
      "democratic",
      subtypes,
    );
    const reversed = classifyCountryLeaders(
      [
        { id: 22, ideology: "liberal" },
        { id: 11, ideology: "fascist" },
      ],
      "democratic",
      subtypes,
    );

    expect(reversed.currentLeaderId).toBe(forwards.currentLeaderId);
    expect(reversed.roleByLeaderId).toEqual(forwards.roleByLeaderId);
  });

  it("does not guess an incumbent when no persisted leader matches", () => {
    expect(
      classifyCountryLeaders(
        [
          { id: 11, ideology: "fascist" },
          { id: 22, ideology: "fascist" },
        ],
        "democratic",
        subtypes,
      ),
    ).toEqual({
      currentLeaderId: null,
      roleByLeaderId: { 11: "candidate", 22: "candidate" },
    });
  });

  it("does not guess an incumbent when multiple persisted leaders match", () => {
    expect(
      classifyCountryLeaders(
        [
          { id: 11, ideology: "liberal" },
          { id: 22, ideology: "conservative" },
        ],
        "democratic",
        subtypes,
      ),
    ).toEqual({
      currentLeaderId: null,
      roleByLeaderId: { 11: "candidate", 22: "candidate" },
    });
  });

  it("keeps incumbent identity tied to the saved ROOT while its draft changes", () => {
    const savedRulingRoot = "democratic";
    const leaders = [
      { id: 11, ideology: "fascist" },
      { id: 22, ideology: "liberal" },
    ];
    const beforeDraftEdit = classifyCountryLeaders(
      leaders,
      savedRulingRoot,
      subtypes,
    );
    const draftedRulingRoot = "fascism";
    const afterDraftEdit = classifyCountryLeaders(
      leaders,
      savedRulingRoot,
      subtypes,
    );

    expect(draftedRulingRoot).not.toBe(savedRulingRoot);
    expect(afterDraftEdit.currentLeaderId).toBe(
      beforeDraftEdit.currentLeaderId,
    );
  });

  it("offers ruling mismatch repair only for the classified incumbent", () => {
    const classification = classifyCountryLeaders(
      [
        { id: 11, ideology: "fascist" },
        { id: 22, ideology: "liberal" },
      ],
      "democratic",
      subtypes,
    );

    expect(
      isCurrentLeaderRulingMismatch(
        classification,
        22,
        "democratic",
        "fascism",
      ),
    ).toBe(true);
    expect(
      isCurrentLeaderRulingMismatch(
        classification,
        11,
        "democratic",
        "fascism",
      ),
    ).toBe(false);

    const ambiguous = classifyCountryLeaders(
      [
        { id: 22, ideology: "liberal" },
        { id: 33, ideology: "conservative" },
      ],
      "democratic",
      subtypes,
    );
    expect(
      isCurrentLeaderRulingMismatch(ambiguous, 22, "democratic", "fascism"),
    ).toBe(false);
  });
});
