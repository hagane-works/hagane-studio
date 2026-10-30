import { readFileSync } from "node:fs";
import { resolve } from "node:path";
import { describe, expect, it } from "vitest";

import {
  parseRunCheckEntityDeepLink,
  resolveDeepLinkScopeReveal,
} from "./runCheckDeepLink";

describe("parseRunCheckEntityDeepLink", () => {
  it("accepts a complete pair of positive database ids", () => {
    const params = new URLSearchParams("namespace=12&event=34");
    expect(parseRunCheckEntityDeepLink(params, "namespace", "event")).toEqual({
      containerId: 12,
      entityId: 34,
    });
  });

  it.each([
    "namespace=12",
    "event=34",
    "namespace=0&event=34",
    "namespace=12&event=-1",
    "namespace=12.5&event=34",
    "namespace=tree&event=34",
  ])(
    "ignores incomplete or malformed targets without partial selection: %s",
    (query) => {
      expect(
        parseRunCheckEntityDeepLink(
          new URLSearchParams(query),
          "namespace",
          "event",
        ),
      ).toBeNull();
    },
  );
});

describe("resolveDeepLinkScopeReveal", () => {
  it("keeps the current country when the linked entity is already visible", () => {
    expect(
      resolveDeepLinkScopeReveal({ scope_kind: "global" }, "USA", [
        "USA",
        "GER",
      ]),
    ).toEqual({ kind: "unchanged" });
  });

  it("switches to the linked owner instead of opening every country", () => {
    expect(
      resolveDeepLinkScopeReveal(
        { scope_kind: "own", country_tag: "GER" },
        "USA",
        ["USA", "GER"],
      ),
    ).toEqual({ kind: "country", countryTag: "GER" });
  });

  it("uses all-country view when no project country can reveal the target", () => {
    expect(
      resolveDeepLinkScopeReveal(
        { scope_kind: "explicit", shared_country_tags: ["FRA"] },
        "USA",
        ["USA", "GER"],
      ),
    ).toEqual({ kind: "all" });
  });

  it("requires one country to reveal both a linked container and its entity", () => {
    expect(
      resolveDeepLinkScopeReveal(
        [
          { scope_kind: "own", country_tag: "USA" },
          { scope_kind: "own", country_tag: "GER" },
        ],
        "USA",
        ["USA", "GER"],
      ),
    ).toEqual({ kind: "all" });
  });
});

function sourceWithoutComments(relativePath: string): string {
  return readFileSync(resolve(relativePath), "utf8")
    .replace(/\/\*[\s\S]*?\*\//g, "")
    .replace(/\/\/.*$/gm, "");
}

describe("Run Check entity deep-link wiring", () => {
  it("validates and selects the requested event within its namespace", () => {
    const source = sourceWithoutComments(
      "src/app/[locale]/project/[id]/events/page.tsx",
    );
    expect(source).toContain(
      'parseRunCheckEntityDeepLink(searchParams, "namespace", "event")',
    );
    expect(source).toContain(
      "eventsApi.get(projectId, runCheckTarget.entityId)",
    );
    expect(source).toContain("targetEvent.namespace_id !== targetNamespace.id");
    expect(source).toContain("selectEvent(targetEvent.id)");
  });

  it("requires the requested focus node to belong to the requested tree", () => {
    const source = sourceWithoutComments(
      "src/app/[locale]/project/[id]/focus-tree/page.tsx",
    );
    expect(source).toContain(
      'parseRunCheckEntityDeepLink(searchParams, "tree", "focus")',
    );
    expect(source).toContain("targetTree?.nodes.find(");
    expect(source).toContain("node.id === runCheckTarget.entityId");
    expect(source).toContain("selectNode(targetNode.id)");
  });
});
