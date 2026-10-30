import { readFileSync, readdirSync } from "node:fs";
import { join } from "node:path";
import { describe, expect, it } from "vitest";
import {
  gameViewHref,
  gameViewTargetOf,
  gameViewWindowName,
  withWorkbenchCountry,
  workbenchHrefForScreen,
  workbenchTargetOf,
  workbenchWindowName,
} from "./gameViewLinks";

describe("links between the workbench and the game view", () => {
  it("opens the game view on a screen, a country and an item", () => {
    expect(gameViewHref(2)).toBe("/game-view?project_id=2");
    expect(
      gameViewHref(2, { tag: " ger ", panel: "countrypoliticsview" }),
    ).toBe("/game-view?project_id=2&tag=GER&panel=countrypoliticsview");
    expect(gameViewHref(2, { panel: "EventWindow", event: "germany.5" })).toBe(
      "/game-view?project_id=2&panel=EventWindow&event=germany.5",
    );
    expect(
      gameViewHref(2, { tag: "GER", panel: "countrystateview", state: 64 }),
    ).toBe("/game-view?project_id=2&tag=GER&panel=countrystateview&state=64");
    expect(gameViewHref(2, { superEvent: 7, tag: null, state: 0 })).toBe(
      "/game-view?project_id=2&super_event=7",
    );
  });

  it("goes back to the workbench a screen is made in, else the start page", () => {
    expect(workbenchHrefForScreen(2, "nationalfocusview")).toBe(
      "/project/2/focus-tree",
    );
    expect(workbenchHrefForScreen(2, "EventWindow")).toBe("/project/2/events");
    expect(workbenchHrefForScreen(2, "GER_SE_main_container")).toBe(
      "/project/2/super-events",
    );
    expect(workbenchHrefForScreen(2, "topbar")).toBe("/project/2/start");
    expect(workbenchHrefForScreen(2, null)).toBe("/project/2/start");
  });

  it("only names workbenches that exist", () => {
    const routes = new Set(
      readdirSync(join(__dirname, "..", "app", "[locale]", "project", "[id]")),
    );
    const screens = [
      "countrypoliticsview",
      "political_ideas_window",
      "nationalfocusview",
      "countrytechnologyview",
      "facilities_view",
      "countrydecisionview",
      "EventWindow",
      "countrystateview",
      "countrydeploymentview",
      "countrydivisiondesignerview",
      "countryproductionlineview",
      "countryofficercorpview",
      "industrial_organisation_list_window",
      "countryintelligenceagencyview",
      "country_doctrine_view",
      "countryarmyview",
      "countrydiplomacyview",
      "manage_subjects_window",
      "gamesetup_scenario_window",
      "X_SE_main_container",
    ];
    for (const screen of screens) {
      const slug = workbenchHrefForScreen(2, screen).split("/").pop() as string;
      expect(routes.has(slug), `${screen} → ${slug}`).toBe(true);
    }
  });
});

describe("the workbench opens on the game view's country", () => {
  it("adds the country to any workbench link, keeping its query and fragment", () => {
    expect(withWorkbenchCountry("/project/7/start", "ger")).toBe(
      "/project/7/start?country=GER",
    );
    expect(
      withWorkbenchCountry("/project/7/focus-tree?tree=3&focus=9", "GER"),
    ).toBe("/project/7/focus-tree?tree=3&focus=9&country=GER");
    expect(
      withWorkbenchCountry("/project/7/events?country=ENG#top", "GER"),
    ).toBe("/project/7/events?country=GER#top");
    expect(withWorkbenchCountry("/project/7/start", null)).toBe(
      "/project/7/start",
    );
    expect(withWorkbenchCountry("/project/7/start", " ")).toBe(
      "/project/7/start",
    );
  });
});

describe("the two tabs", () => {
  it("names one game view tab per project and the workbench tab, and reads them back from links", () => {
    expect(gameViewWindowName(2)).toBe("hoi4-game-view-2");
    expect(workbenchWindowName(2)).toBe("hoi4-workbench-2");
    expect(gameViewTargetOf(gameViewHref(7, { panel: "EventWindow" }))).toBe(
      "hoi4-game-view-7",
    );
    expect(workbenchTargetOf("/project/7/events?namespace=1&event=2")).toBe(
      "hoi4-workbench-7",
    );
  });
});

describe("the workbench and the game view are wired to each other", () => {
  const read = (...parts: string[]) =>
    readFileSync(join(__dirname, "..", ...parts), "utf8");

  it("the top bar opens the game view and a bare GameView URL stays on the project's country", () => {
    const layout = read("app", "[locale]", "project", "[id]", "layout.tsx");
    expect(layout).toContain(
      'gameViewHref(projectId, { tag: activeCountry || currentProject.country_tag, panel: "countrypoliticsview" })',
    );

    expect(layout).toContain(
      'countries.length === 0 ? gameViewHref(projectId, { panel: "gamesetup_scenario_window" })',
    );

    const addCountry = layout.indexOf(
      '<Button size="sm" onClick={openAddCountry}>',
    );
    const entry = layout.indexOf('{t("navGameView")}');
    expect(addCountry).toBeGreaterThan(0);
    expect(entry).toBeGreaterThan(addCountry);
    expect(entry).toBeLessThan(layout.indexOf("</header>", addCountry));
    expect(layout).toContain("target={gameViewWindowName(projectId)}");
    expect(layout).toContain("window.name = workbenchWindowName(projectId);");
    const page = read("app", "[locale]", "game-view", "page.tsx");

    expect(page).not.toContain(
      "href={workbenchHrefForScreen(projectId, base)}",
    );

    expect(page).toContain("onClick={() => void refreshGameView()}");
    expect(page).toContain(
      'document.addEventListener("visibilitychange", onVisible);',
    );
    expect(page).toContain(
      'const requestedSuperEventId = Number(params.get("super_event")) || null;',
    );
    expect(page).toContain(
      "useState<number | null>(() => requestedSuperEventId)",
    );
    expect(page).toContain('chosenTag || routeTag || projectTag.data || "CHI"');
  });

  it("each workbench's preview column opens its screen, and each game view panel opens its workbench row", () => {
    expect(read("components", "workbench", "WorkbenchShell.tsx")).toContain(
      "<OpenInGameLink href={gameViewHref}",
    );
    expect(read("components", "workbench", "OpenInGameLink.tsx")).toContain(
      '{t("openInGame")}',
    );
    for (const page of [
      "events",
      "characters",
      "decisions",
      "ideas",
      "technology",
      "states",
      "division-templates",
      "deployments",
      "intelligence",
      "autonomy-states",
      "equipment",
      "sub-units",
      "wargoals",
      "dynamic-modifiers",
    ]) {
      expect(
        read("app", "[locale]", "project", "[id]", page, "page.tsx"),
        page,
      ).toContain("gameViewHref={gameViewHref(projectId,");
    }
    for (const file of [
      "BookmarksContent.tsx",
      "MIOsDialog.tsx",
      "DoctrinesContent.tsx",
      "HistoryFilesContent.tsx",
    ]) {
      expect(read("components", "workbench", file), file).toContain(
        "gameViewHref={gameViewHref(projectId,",
      );
    }
    expect(read("components", "workbench", "SuperEventsContent.tsx")).toContain(
      "superEvent: activeRow.id",
    );

    const ownHeaders = [
      ["app", "[locale]", "project", "[id]", "focus-tree", "page.tsx"],
      ["app", "[locale]", "project", "[id]", "special-projects", "page.tsx"],
      ["components", "workbench", "oob-units", "OobUnitsEditor.tsx"],
      [
        "components",
        "workbench",
        "country-profile",
        "CountryProfileWorkbench.tsx",
      ],
      ["components", "workbench", "politics", "PoliticsWorkspaceNav.tsx"],
    ];
    for (const parts of ownHeaders)
      expect(read(...parts), parts.join("/")).toContain("<OpenInGameLink");
    expect(
      read("app", "[locale]", "project", "[id]", "focus-tree", "page.tsx"),
    ).toContain('panel: "nationalfocusview", tree: currentTree?.tree_id');

    const panels: Record<string, string> = {
      "EventEditPanel.tsx":
        "/events?namespace=${row.data.namespace_id}&event=${row.data.id}",
      "CharacterEditPanel.tsx": "/characters?c=${row.id}",
      "StateEditPanel.tsx": "/states?state=${shown.id}",
      "SuperEventEditPanel.tsx": "/super-events?se=${row.id}",
      "FocusEditPanel.tsx": "/focus-tree?tree=${tree.data.id}&focus=${node.id}",
      "DecisionEditPanel.tsx":
        "/decisions?category=${decisionRow.data.category_id}&decision=${decisionRow.data.id}",
      "TechEditPanel.tsx": "/technology?tech=${row.id}",
      "SpiritEditPanel.tsx": "/ideas?idea=${found.row.id}",
      "SpiritStartPanel.tsx": "/history-files?hf=${row.data.id}",
      "DiplomacyEditPanel.tsx": "/history-files?hf=${row.data.id}",
      "IdeologyEditPanel.tsx": "/country-politics",
      "CountryProfilePanel.tsx": "/country-profile",
      "BookmarkEditPanel.tsx": "/bookmarks?bm=${row.id}",
      "ScientistEditPanel.tsx": "/characters?c=${row.id}",
      "EquipmentEditPanel.tsx": "/equipment?equipment=${row.id}",
      "MioTraitEditPanel.tsx": "/mio?mio=${row.id}",
      "DeploymentEditPanel.tsx": "/deployments",
      "WarGoalEditPanel.tsx": "/project/${projectId}/wargoals",
    };
    for (const [file, path] of Object.entries(panels))
      expect(read("app", "[locale]", "game-view", file), file).toContain(path);

    for (const file of ["SubUnitEditPanel.tsx", "OOBUnitEditPanel.tsx"]) {
      expect(read("app", "[locale]", "game-view", file), file).not.toContain(
        "<OpenInWorkbench",
      );
    }
  });

  it("the workbenches select what the links name, and an import offers the game view", () => {
    expect(
      read("app", "[locale]", "project", "[id]", "states", "page.tsx"),
    ).toContain('searchParams?.get("state")');
    expect(
      read("app", "[locale]", "project", "[id]", "technology", "page.tsx"),
    ).toContain('searchParams?.get("tech")');
    expect(
      read("app", "[locale]", "project", "[id]", "ideas", "page.tsx"),
    ).toContain('searchParams?.get("idea")');
    const decisions = read(
      "app",
      "[locale]",
      "project",
      "[id]",
      "decisions",
      "page.tsx",
    );

    expect(decisions.indexOf("selectCategory(category);")).toBeLessThan(
      decisions.indexOf("if (decision) selectDecision(decision);"),
    );

    const linked: Array<[string[], string]> = [
      [
        ["components", "workbench", "SuperEventsContent.tsx"],
        'useLinkedRowSelection("se", rows, selectRow)',
      ],
      [
        ["components", "workbench", "BookmarksContent.tsx"],
        'useLinkedRowSelection("bm", rows, selectRow)',
      ],
      [
        ["components", "workbench", "MIOsDialog.tsx"],
        'useLinkedRowSelection("mio", rows, selectRow)',
      ],
      [
        ["components", "workbench", "HistoryFilesContent.tsx"],
        'useLinkedRowSelection("hf", rows, selectRow)',
      ],
      [
        ["app", "[locale]", "project", "[id]", "equipment", "page.tsx"],
        'useLinkedRowSelection("equipment", list,',
      ],
    ];
    for (const [parts, call] of linked) {
      const source = read(...parts);
      expect(source, parts.join("/")).toContain(call);
      expect(source, parts.join("/")).not.toMatch(
        /useState<[^>]*>\(\(\) => Number\(searchParams/,
      );
    }

    const events = read(
      "app",
      "[locale]",
      "project",
      "[id]",
      "events",
      "page.tsx",
    );
    expect(events).not.toContain(
      ".events.some((event) => event.id === targetEvent.id)",
    );
    expect(
      read("components", "workbench", "import-wizard", "ImportSuccessCard.tsx"),
    ).toContain('{t("whatsNextGameViewTitle")}');
  });
});
