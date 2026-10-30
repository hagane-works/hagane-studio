import type { VanillaStateIndex } from "@/lib/vanillaStateIndex";

export const STATE_INDEX_FIXTURE: VanillaStateIndex = {
  content: {
    1: { navalBases: 1 },
    2: {
      navalBases: 3,
      landmarks: ["landmark_colosseum"],
      datedBlocks: 1,
      hasDlc: 1,
    },
    379: { otherContent: 1 },
    608: {
      navalBases: 1,
      landmarks: ["landmark_forbidden_city"],
      datedBlocks: 2,
      hasDlc: 2,
    },
    609: {
      navalBases: 1,
      landmarks: ["landmark_great_wall_section"],
      datedBlocks: 1,
      hasDlc: 2,
    },
    987: { resistance: 3 },
    1040: { datedBlocks: 2, hasDlc: 2 },
    1057: { resistance: 2 },
  },
  dlc: {
    2: ["Gotterdammerung"],
    608: ["No Compromise, No Surrender", "No Compromise, No Surrender"],
    609: ["No Compromise, No Surrender", "No Compromise, No Surrender"],
    1040: ["No Compromise, No Surrender", "Waking the Tiger"],
  },
};
