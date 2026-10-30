import { readFileSync } from "node:fs";
import { join } from "node:path";
import { describe, expect, it } from "vitest";

const source = (file: string) => readFileSync(join(__dirname, file), "utf8");
const compact = (value: string) => value.replace(/\s+/g, " ");

describe("GameView view-mode click matrix", () => {
  const page = source("page.tsx");
  const screenLayer = page.slice(
    page.indexOf("function ScreenLayer("),
    page.indexOf("export default function GameViewPage"),
  );

  it("keeps every content object hit wired when editMode is false", () => {
    const alwaysOnHits: Array<[string, string]> = [
      ["technology nodes", "onPick={onPickTech}"],
      ["decision/category rows", "onPick={onPickDecision}"],
      ["event fields/options", "onPickField={onPickEventField}"],
      ["equipment and designs", "onEditEquipment={onEditEquipment}"],
      ["equipment design slots", "onEditDesign={onEditDesign}"],
      ["war-goal kinds", "onEditWarGoal={onEditWarGoal}"],
      ["agency insignia", "onEditInsignia={onEditInsignia}"],
      ["agency upgrades", "onEditUpgrade={onEditIntelUpgrade}"],
      ["autonomy levels", "onEditLevel={onEditAutonomy}"],
      ["army divisions", "onEditDivision={onEditDivision}"],
      ["navy and air units", "onEditUnit={onEditUnit}"],
      ["scientists", "onEditScientist={onEditScientist}"],
      ["parties and ideologies", "onEditParties={onEditParties}"],
      ["characters", "onEditPerson={onEditPerson}"],
      ["national spirits", "onEditSpirit={onEditSpirit}"],
      ["MIO traits", "onEditTrait={onEditMioTrait}"],
    ];

    for (const [label, wiring] of alwaysOnHits) {
      expect(screenLayer, label).toContain(wiring);
      expect(
        screenLayer,
        `${label} must not be gated by editMode`,
      ).not.toContain(wiring.replace("={", "={editMode ? "));
    }

    expect(page).not.toContain("<ContentInspectPanel");
  });

  it("keeps choosing a first-day law or advisor edit-only", () => {
    expect(screenLayer).toContain(
      "onChoose={editMode ? (token) => void chooseStart(token) : undefined}",
    );
    expect(screenLayer).toContain(
      "const historyRow = useHistoryRow(projectId, tag ?? null, editMode && isPicker)",
    );
  });

  it("opens the technology tree from the engine-owned empty research rows", () => {
    const research = source("ResearchLayer.tsx");

    expect(research).toContain("onOpenTree?: () => void");
    expect(research).toContain("onClick={onOpenTree}");
    expect(research).toContain('appearance="inspect"');
    expect(screenLayer).toContain("onOpenTree={onOpenTechTree}");
    expect(page).toContain("onOpenTechTree={() => {");
    expect(page).toContain("setPanel(TECH_TREE);");
    expect(page).toContain(
      "new Set([DOCTRINE_VIEW, TECH_TREE, DEPLOYMENT_VIEW])",
    );
  });

  it("never leaves a nested engine button without a click result", () => {
    const canvas = source("../gui-renderer/GuiCanvas.tsx");

    expect(canvas).toContain(
      'GUI_UNHANDLED_ACTIVATE_EVENT = "gui-canvas:unhandled-activate"',
    );
    expect(canvas).toContain("!editMode && ctx.hits");
    expect(canvas).toContain("if (onActivate)");
    expect(canvas).toContain("semanticParent");
    expect(canvas).toContain(
      "window.dispatchEvent(new CustomEvent<GuiUnhandledActivateDetail>",
    );
    expect(page).toContain(
      "window.addEventListener(GUI_UNHANDLED_ACTIVATE_EVENT, announce)",
    );
  });

  it("keeps the deployment template checkbox stateful in view mode", () => {
    const deployment = source("DeploymentLayer.tsx");

    expect(deployment).toContain(
      "const [showDecommissioned, setShowDecommissioned] = useState(false)",
    );
    expect(deployment).toContain('aria-label="show_decommissioned_templates"');
    expect(deployment).toContain("aria-pressed={showDecommissioned}");
    expect(deployment).toContain("setShowDecommissioned((shown) => !shown)");
    expect(deployment).toContain("frame: 2");
    expect(deployment).toContain("onUnavailable?.(FOREIGN_TEMPLATES)");
    const companion = page.slice(
      page.indexOf("{companionName && companion.data && ("),
      page.indexOf('{panel === "nationalfocusview"'),
    );
    expect(companion).toContain("onUnsupportedControl=");
  });

  it("answers the construction screen's convert button outside edit mode", () => {
    const construction = source("ConstructionLayer.tsx");

    expect(construction).toContain('onUnavailable?.("convert factories")');
    expect(screenLayer).toContain("onUnavailable={onUnsupportedControl}");
  });

  it("makes overview category pills actual filters", () => {
    const overview = source("ArmyOverviewLayer.tsx");

    expect(overview).toContain("const [pillFilter, setPillFilter]");
    expect(overview).toContain("aria-pressed={effectiveFilter === item.key}");
    expect(overview).toContain("allTaskForces.filter");
    expect(overview).toContain("allWings.filter");
  });

  it("makes every visible diplomacy action clickable while only navigating supported actions", () => {
    const diplomacy = source("DiplomacyLayer.tsx");
    expect(diplomacy).toContain("actionList.map((entry, at) => (");
    expect(diplomacy).toContain("onClick={() => onAction(entry.id)}");

    expect(
      page.match(/if \(id !== "generate_wargoal"\) return;/g)?.length,
    ).toBe(2);
  });

  it("opens the inspected country's focus tree from diplomacy", () => {
    expect(page).toContain(
      "setFocusViewTag(owner === DIPLOMACY ? diplomacyTarget : null)",
    );

    expect(page).toContain(
      '["game-view-focus-tree", projectId, focusTreeId, focusTag, scenarioDate]',
    );
    expect(page).toContain(
      "guiRenderApi.focusTree(focusTreeId, focusTag, scenarioDate)",
    );
    expect(page).toContain("chosenTrees[focusTag.toUpperCase()]");
    expect(page).toContain("focusViewTag ? null : requestedFocusTree");
    expect(page).toContain("rememberFocusTree(focusTag, treeId)");
    expect(page).not.toContain("setChosenTree(");
  });

  it("does not swallow the admiral trait button while the navy trait tree is unavailable", () => {
    const roster = compact(source("CommanderRosterLayer.tsx"));

    expect(roster).toContain(
      'kind.kind === "army" && onDetails ? () => onDetails(leader.token) : onEditPerson ? () => onEditPerson(leader.token, leader.name) : undefined',
    );
  });

  it("treats omitted arrays in partial GUI payloads as empty", () => {
    expect(page).toContain("guiArray(editedWarGoals.data?.types).find");
    expect(page).toContain("guiArray(editedIntel.data?.insignia).find");
    expect(page).toContain("guiArray(editedAgency.data?.branches)");
    expect(source("CharacterLayer.tsx")).toContain(
      "guiArray(category.slots).map",
    );
  });

  it("opens a subject country from its redrawn flag", () => {
    const subjects = source("SubjectsLayer.tsx");

    expect(subjects).toContain('aria-label={"open country " + subject.tag}');
    expect(subjects).toContain("onClick={() => onOpenCountry(subject.tag)}");
    expect(screenLayer).toContain("onOpenCountry={onDiplomacyTarget}");
  });

  it("opens faction members from their dynamic cards", () => {
    const faction = source("FactionLayer.tsx");

    expect(faction).toContain(
      'aria-label={"open faction member " + member.tag}',
    );
    expect(screenLayer).toContain("onOpenCountry={onDiplomacyTarget}");
  });

  it("opens a trade seller from its flag", () => {
    const trade = source("TradeLayer.tsx");

    expect(trade).toContain('aria-label={"open trade country " + seller.tag}');
    expect(trade).toContain("onOpenCountry(seller.tag)");
    expect(screenLayer).toContain("onOpenCountry={onDiplomacyTarget}");
  });

  it("separates inspect appearance from content editing", () => {
    const fieldHit = source("FieldHit.tsx");
    const context = source("editContext.ts");

    expect(context).toContain("contentEditMode: boolean");
    expect(fieldHit).toContain(
      "const { contentEditMode } = useContext(EditContext)",
    );
    expect(fieldHit).toContain('appearance?: "edit" | "inspect"');
    expect(fieldHit).toContain(
      'appearance ?? (contentEditMode ? "edit" : "inspect")',
    );
    expect(fieldHit).toContain('pointerEvents: "auto"');
  });

  it("keeps persistent mutations behind editMode", () => {
    const mutations = [
      "onSave={editMode && designDirty && !designSaving ? () => void saveDesign() : undefined}",
      "onChoose={editMode ? (choice) => void chooseDoctrine(choice) : undefined}",
      "onChoose={editMode ? (token) => void chooseStart(token) : undefined}",
      "onAddSpirit={editMode ? onAddSpirit : undefined}",
      "onMove={editMode && !picking ? moveFocus : undefined}",
      "editMode && !uiEdit",
    ];
    for (const guard of mutations) expect(page, guard).toContain(guard);

    for (const panel of [
      "DecisionEditPanel",
      "CharacterEditPanel",
      "FocusEditPanel",
      "EquipmentEditPanel",
      "MioTraitEditPanel",
      "ScientistEditPanel",
      "AutonomyEditPanel",
    ]) {
      const at = page.indexOf(`<${panel}`);
      expect(at, `${panel} is rendered`).toBeGreaterThan(0);
      expect(
        page.slice(Math.max(0, at - 800), at),
        `${panel} edit guard`,
      ).toContain("editMode");
    }
  });

  it("restores ordinary clicks immediately after UI 编辑 exits", () => {
    expect(page).toContain("{uiEdit && <style>{UI_EDIT_STAGE_CSS}</style>}");
    expect(page).toContain('data-ui-edit-stage={uiEdit ? "" : undefined}');
    expect(page).toContain(
      "onPointerDownCapture={uiEdit ? pickInUiEdit : undefined}",
    );
    expect(page).toContain(
      "onClickCapture={uiEdit ? (e) => e.stopPropagation() : undefined}",
    );
    const scopeReset = page.slice(
      page.indexOf(
        'if (scopeChange === "project" || scopeChange === "country")',
      ),
      page.indexOf('if (scopeChange === "project")'),
    );
    expect(scopeReset).toContain('setEditMode(params.get("edit") === "1")');
    expect(scopeReset).toContain('setUiEdit(params.get("edit") === "ui")');
  });

  it("starts a fresh game surface when a link changes URL context", () => {
    expect(page).toContain("panelRequest.projectId !== projectId");
    expect(page).toContain("panelRequest.requestedPanel !== requestedPanel");
    expect(page).toContain("templateRequest.projectId !== projectId");
    expect(page).toContain(
      "templateRequest.requestedTemplate !== requestedTemplate",
    );
    expect(page).toContain("templateRequest.requestedTag !== requestedTag");
    expect(page).toContain("setPanel(requestedPanel)");
    expect(page).toContain("setTemplate(requestedTemplate)");
    expect(page).toContain("routeTargets.projectId !== projectId");
    expect(page).toContain("routeTargets.panel !== requestedPanel");
    expect(page).toContain("routeTargets.tag !== requestedTag");
    expect(page).toContain("routeTargets.template !== requestedTemplate");
    expect(page).toContain("routeTargets.focusTree !== requestedFocusTree");
    expect(page).toContain("setChosenEvent(null)");
    expect(page).toContain("setSuperEventId(requestedSuperEventId)");
    expect(page).toContain("setLeaderDetail(requestedLeader)");
    expect(page).toContain("setChosenState(null)");
    expect(page).toContain("setTechFolder(requestedTechFolder)");
    expect(page).toContain("setDoctrineFolder(requestedDoctrineFolder)");
    expect(page).toContain("setDiplomacyTarget(null)");
    expect(page).toContain("setChosenTrees({})");
  });

  it("discards project-local edits and picks when a link changes project", () => {
    const scopeReset = page.slice(
      page.indexOf('if (scopeChange !== "none")'),
      page.indexOf("const deployedOn"),
    );
    expect(scopeReset).toContain("setSelection(null)");

    const projectReset = page.slice(
      page.indexOf('if (scopeChange === "project")'),
      page.indexOf("const deployedOn"),
    );
    for (const reset of [
      "setEventSaved({})",
      "setDecisionSaved({})",
      "setTechSaved({})",
      "setLocEdits({})",
      "setSpriteVersions({})",
      "setDecisionPick(null)",
      "setPersonPick(null)",
      "setEquipmentPick(null)",
      "setMapProvince(0)",
    ]) {
      expect(projectReset, reset).toContain(reset);
    }

    const focusReset = page.slice(
      page.indexOf("const [focusEdits"),
      page.indexOf("const shownFocusTree"),
    );
    expect(focusReset).toContain('if (scopeChange === "project")');
    expect(focusReset).toContain("setFocusEdits({})");
    expect(focusReset).toContain("setFocusRemoved(new Set())");
    expect(focusReset).toContain("setPicking(null)");
  });

  it("gives technology its read-only details panel outside edit mode", () => {
    expect(compact(page)).toContain(
      "const clearInspectedContent = useCallback(() => { setSelectedFocus(null); setTechPick(null); }, []);",
    );
    expect(page).not.toContain("<ContentInspectPanel");
    expect(page).toMatch(
      /<TechInspectPanel[\s\S]{0,500}onClose=\{clearInspectedContent\}/,
    );
  });
});
