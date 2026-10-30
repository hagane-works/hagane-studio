import { describe, expect, it } from "vitest";
import { gameViewScopeChange, type GameViewScope } from "./viewScope";

const scope: GameViewScope = {
  projectId: 69,
  tag: "PER",
  base: "nationalfocusview",
  panel: "nationalfocusview",
};

describe("gameViewScopeChange", () => {
  it("does not discard a pick while establishing or rerendering the same scope", () => {
    expect(gameViewScopeChange(null, scope)).toBe("none");
    expect(gameViewScopeChange(scope, { ...scope })).toBe("none");
  });

  it.each<[string, Partial<GameViewScope>, string]>([
    ["project", { projectId: 70 }, "project"],
    ["country", { tag: "CHI" }, "country"],
    ["base screen", { base: "countrypoliticsview" }, "surface"],
    ["open window", { panel: "idea_view" }, "surface"],
  ])("classifies a %s change", (_label, change, expected) => {
    expect(gameViewScopeChange(scope, { ...scope, ...change })).toBe(expected);
  });

  it("gives a project or country change precedence over its resulting surface change", () => {
    expect(
      gameViewScopeChange(scope, {
        ...scope,
        projectId: 70,
        tag: "CHI",
        panel: null,
      }),
    ).toBe("project");
    expect(
      gameViewScopeChange(scope, { ...scope, tag: "CHI", panel: null }),
    ).toBe("country");
  });
});
