import { describe, expect, it } from "vitest";

import {
  POLITICS_WORKSPACE_ROUTES,
  isPoliticsWorkspaceRoute,
  politicsWorkspaceHref,
} from "./politicsWorkspaceRoutes";

describe("politics workspace routes", () => {
  it("builds real project routes instead of hash-only prototype state", () => {
    expect(
      politicsWorkspaceHref(17, POLITICS_WORKSPACE_ROUTES.countryPolitics),
    ).toBe("/project/17/country-politics");
    expect(
      politicsWorkspaceHref(17, POLITICS_WORKSPACE_ROUTES.ideologies),
    ).toBe("/project/17/ideologies");
    expect(
      politicsWorkspaceHref(17, POLITICS_WORKSPACE_ROUTES.countryProfile),
    ).toBe("/project/17/country-profile");
  });

  it("groups only the two politics tabs under one sidebar entry", () => {
    expect(isPoliticsWorkspaceRoute("country-politics")).toBe(true);
    expect(isPoliticsWorkspaceRoute("ideologies")).toBe(true);
    expect(isPoliticsWorkspaceRoute("country-profile")).toBe(false);
  });
});
