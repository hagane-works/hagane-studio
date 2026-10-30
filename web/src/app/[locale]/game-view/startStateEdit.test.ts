import { describe, expect, it } from "vitest";
import { withAdvisor, withChosenIdea } from "./startStateEdit";

const germany = [
  "sour_loser",
  "limited_exports",
  "limited_conscription",
  "partial_economic_mobilisation",
];
const conscription = [
  "disarmed_nation",
  "volunteer_only",
  "limited_conscription",
  "extensive_conscription",
];

describe("withChosenIdea", () => {
  it("puts the chosen law where its group's law was", () => {
    expect(
      withChosenIdea(germany, conscription, "extensive_conscription"),
    ).toEqual([
      "sour_loser",
      "limited_exports",
      "extensive_conscription",
      "partial_economic_mobilisation",
    ]);
  });

  it("adds it at the end when the list held none of the group", () => {
    expect(
      withChosenIdea(["sour_loser"], conscription, "volunteer_only"),
    ).toEqual(["sour_loser", "volunteer_only"]);
  });

  it("leaves a list that already holds it as it was", () => {
    expect(
      withChosenIdea(germany, conscription, "limited_conscription"),
    ).toEqual(germany);
  });
});

describe("withAdvisor", () => {
  it("hires by adding the idea token, once", () => {
    expect(withAdvisor(germany, "GER_franz_seldte", true)).toEqual([
      ...germany,
      "GER_franz_seldte",
    ]);
    expect(
      withAdvisor([...germany, "GER_franz_seldte"], "GER_franz_seldte", true),
    ).toEqual([...germany, "GER_franz_seldte"]);
  });

  it("lets go by taking it out", () => {
    expect(
      withAdvisor([...germany, "GER_franz_seldte"], "GER_franz_seldte", false),
    ).toEqual(germany);
  });
});
