"use client";

import {
  Fragment,
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
  useSyncExternalStore,
} from "react";
import { useSearchParams } from "@/platform/navigation";
import { useLocale, useTranslations } from "use-intl";
import useSWR, { mutate as globalMutate } from "swr";
import { toast } from "sonner";
import { focusTreesApi } from "@/lib/api/focus-tech";
import { detectFocusPrefix, suggestNewFocusId } from "@/lib/focusIdSuggest";
import { getGuiLoadSnapshot, useGuiLoadActivity } from "@/lib/guiLoadActivity";
import {
  guiRenderApi,
  setGameLanguage,
  type GuiBookmarks,
  type GuiCountryStats,
  type GuiNode,
  type GuiSprite,
  type GuiWindow,
} from "@/lib/guiRenderApi";
import {
  GUI_UNHANDLED_ACTIVATE_EVENT,
  GuiCanvas,
  assignIds,
  flattenTree,
  measureWindow,
  type GuiUnhandledActivateDetail,
  type LayoutOverride,
  type ResizeAxes,
} from "../gui-renderer/GuiCanvas";
import { GameTooltipHost } from "../gui-renderer/GameTooltip";
import {
  FocusTreeLayer,
  type FocusNode,
  type FocusTreePayload,
} from "./FocusTreeLayer";
import { CharacterLayer, useCharacters } from "./CharacterLayer";
import { applySwaps, type SpriteSwaps } from "./spriteSwaps";
import { EditPanel } from "./EditPanel";
import { FocusEditPanel } from "./FocusEditPanel";
import { FocusTreeSwitcher } from "./FocusTreeSwitcher";
import {
  freeCell,
  linkProblem,
  relativeDescendants,
  storedPosition,
  withPrereq,
  withoutPrereq,
} from "./focusEdit";
import {
  editTargetOf,
  guiTextKey,
  guiTextOverrides,
  locEditOverrides,
  removalOf,
  withHiddenMark,
  withOwnPicture,
  type EditSelection,
} from "./editModel";
import { fixedTextSources, leaderNamePrinted, pickedText } from "./textSources";
import {
  createPrintedRegistry,
  EditContext,
  withPrintedOver,
  type EditContextValue,
} from "./editContext";
import { pick, recordChange, rowUndo } from "./changeLog";
import { onCountryTakenIn, speakFor, takeInCountry } from "./countryTakeIn";
import { useVanillaTagName } from "@/lib/vanillaCountryTags";
import {
  addressable,
  restoreGuiLayout,
  saveGuiLayout,
} from "./guiLayoutOverlay";
import { GuiTextNotExportable } from "./guiTextOverlay";
import { buildNamePathMap } from "../gui-renderer/guiOverlay";
import {
  COUNTRY_PICKER,
  CountrySelectLayer,
  useBookmarks,
} from "./CountrySelectLayer";
import { NewCountryControl } from "./NewCountryControl";
import { SCENARIO_PICKER, ScenarioSelectLayer } from "./ScenarioSelectLayer";
import { onStartScreens, startScreensToggle } from "./startScreens";
import {
  DEPLOYMENT_VIEW,
  DeploymentLayer,
  useDivisions,
} from "./DeploymentLayer";
import { DeploymentEditPanel } from "./DeploymentEditPanel";
import { TopbarClock, TopbarLayer, useTopbar } from "./TopbarLayer";
import { AlertsLayer } from "./AlertsLayer";
import { IDLE_TOPBAR, type TopbarUi } from "./topbarModel";
import {
  MAPMODES_WINDOW,
  MAP_MODE_LAYER,
  MAP_MODE_SLOTS,
  mapModeSlotOf,
  mapmodesWindow,
  type MapModeId,
} from "./mapmodesModel";
import {
  artSelection,
  invalidateLayerSpriteMeta,
  layerSpriteMeta,
} from "./UiEditArt";
import { gameViewScopeChange, type GameViewScope } from "./viewScope";
import { LoadingBar } from "./LoadingBar";
import { GameLoadingOverlay, GameLoadingPill } from "./GameLoadingOverlay";
import {
  LOADING_MS,
  LOADING_STEPS,
  LOADING_TIP_KEYS,
  LOAD_SCREEN,
  MAIN_MENU,
  MENU_BACKGROUND,
  MENU_CHANGE_BACKGROUND,
  MENU_DLC_COLUMN,
  MENU_DLC_ITEM,
  MENU_MAIN_PANEL,
  MENU_SINGLE_PLAYER,
  MENU_SOCIAL,
  MENU_UPPER_LEFT,
  MENU_UPPER_RIGHT,
  dlcColumnSprites,
  dlcColumnWindow,
  loadScreenWindow,
  loadingStepAt,
  mainMenuWindow,
  mainPanelSize,
  socialWindow,
  menuActionOf,
  menuBackgroundWindow,
  pickTip,
  screenPictureSprite,
} from "./mainMenuModel";
import { pinFullSizes } from "./topbarLayout";
import { gameCursor, pickable } from "./spriteFrame";
import {
  DIVISION_DESIGNER,
  DesignerLayer,
  designerTextOverrides,
  designerTextSources,
  designerTooltipFor,
  designerWindow,
  designerKey,
  useDesigner,
  useDesignerBarTexts,
} from "./DesignerLayer";
import {
  EMPTY_DRAFT,
  NEW_TEMPLATE,
  draftOf,
  encodeCells,
  designerButtonsOff,
  designerSavePayload,
  sameDraft,
  saveCost,
  type DesignerDraft,
  type PickerState,
} from "./designerEdit";
import {
  ARMY_HQ_TEMPLATES,
  ArmyHQTemplatesLayer,
} from "./ArmyHQTemplatesLayer";
import {
  PageScroll,
  TECH_TREE,
  TechTreeLayer,
  scrollTechTree,
  techTreeWindow,
  useTechTree,
  type GuiTechnology,
  type TechChange,
} from "./TechTreeLayer";
import {
  withContainerScroll,
  type ScrollOffset,
} from "../gui-renderer/containerScroll";
import { PageScrollbar } from "./PageScrollbar";
import {
  DOCTRINE_RINGS,
  DOCTRINE_VIEW,
  DoctrineLayer,
  doctrineWindow,
  useDoctrines,
} from "./DoctrineLayer";
import { doctrinePatch, type DoctrineChoice } from "./doctrineEdit";
import { TechEditPanel } from "./TechEditPanel";
import { TechInspectPanel } from "./TechInspectPanel";
import {
  CONSTRUCTION,
  ConstructionLayer,
  constructionWindow,
  useBuildings,
  useConstruction,
} from "./ConstructionLayer";
import {
  DECISIONS,
  DecisionsLayer,
  decisionEditKey,
  useDecisions,
  type DecisionEdits,
  type DecisionPick,
} from "./DecisionsLayer";
import { DecisionEditPanel } from "./DecisionEditPanel";
import { CharacterEditPanel } from "./CharacterEditPanel";
import {
  CommanderRosterLayer,
  ROSTERS,
  rosterWindow,
  useCommanderRoster,
} from "./CommanderRosterLayer";
import {
  LEADER_DETAILS,
  LeaderDetailsLayer,
  leaderDetailsWindow,
  useLeaderDetails,
} from "./LeaderDetailsLayer";
import { guiArray } from "./guiResponse";
import {
  canSkipGameViewRefresh,
  clearOfMapModes,
  gameStageGeometry,
  isGameDataKey,
  isGameViewRefreshKey,
} from "./gameStage";
import {
  CHARACTER_SLOTS,
  IDEA_WINDOW,
  IdeaPickerLayer,
  advisorChoices,
  ideaWindowTexts,
  useIdeaChoices,
} from "./IdeaPickerLayer";
import {
  IndustrialOrgDetailLayer,
  MIO_DETAIL_WINDOW,
  MIO_TABS,
  mioDetailWindow,
  useIndustrialOrg,
  type MioTab,
} from "./IndustrialOrgDetailLayer";
import {
  IndustrialOrgLayer,
  MIO_FILTER_BUTTONS,
  MIO_LIST_WINDOW,
  MIO_SLOTS,
  mioListWindow,
  toggleFilter,
  type MioCategory,
} from "./IndustrialOrgLayer";
import { useHistoryRow, withAdvisor, withChosenIdea } from "./startStateEdit";
import { historyFilesApi } from "@/lib/api/scenario";
import { PRODUCTION, ProductionLayer, useProduction } from "./ProductionLayer";
import { productionTexts } from "./ProductionQueue";
import { TRADE, TradeLayer } from "./TradeLayer";
import { LOGISTICS, LogisticsLayer } from "./LogisticsLayer";
import {
  OFFICER_CORPS,
  OfficerCorpsLayer,
  scrollOfficerCorps,
  officerCorpsTextOverrides,
  officerCorpsTooltipFor,
  officerCorpsWindow,
  spiritEntryNames,
  useOfficerCorps,
  useOfficerTooltipTexts,
  useSpiritEntries,
} from "./OfficerCorpsLayer";
import { PersonSwitcher } from "./PersonSwitcher";
import { SpiritEditPanel } from "./SpiritEditPanel";
import { SpiritStartPanel } from "./SpiritStartPanel";
import { MioTraitEditPanel } from "./MioTraitEditPanel";
import { EDIT_SWITCHER_SPACE, EditSwitcherDock } from "./GamePanel";
import {
  ComponentPalette,
  DropGhostLayer,
  componentScroll,
  itemSize,
  panelLabel,
  PanelComponentEditor,
  panelScrollOf,
  refreshPanelWindows,
  setComponentScroll,
  setDropGhost,
  SnapGuides,
  screenOfMount,
  useComponentHistory,
  useComponentScrolls,
  useMoveComponent,
  useMoveWindow,
  useEditWords,
  usePlaceComponent,
  useScreenPanels,
  type DropGhost,
  type PaletteDrag,
  type PaletteItem,
} from "./ScreenComponents";
import { DlcBadge } from "./DlcBadge";
import { HoverTip } from "./HoverTip";
import {
  boxesOf,
  containerAt,
  controlByGuiName,
  TOPBAR_MOUNT,
  flagOpsOf,
  isWindowPanel,
  mountOf,
  openFlagOf,
  opensByButton,
  panelWindowName,
  pickedControlNames,
  snapBox,
  toggleFlagEffect,
  type Box,
  type Guide,
} from "./panelMounts";

const BASES_OVER_ALERTS: ReadonlySet<string> = new Set([
  DOCTRINE_VIEW,
  TECH_TREE,
  DEPLOYMENT_VIEW,
]);
import {
  DIPLOMACY,
  DiplomacyLayer,
  diplomacyListHidden,
  diplomacyTargetWindow,
  useDiplomacy,
} from "./DiplomacyLayer";
import { DiplomacyEditPanel } from "./DiplomacyEditPanel";
import { IdeologyEditPanel } from "./IdeologyEditPanel";
import { CountryProfilePanel } from "./CountryProfilePanel";
import { EquipmentEditPanel } from "./EquipmentEditPanel";
import { EquipmentDesignPanel } from "./EquipmentDesignPanel";
import {
  NO_WARGOAL,
  WARGOAL_WINDOW,
  WarGoalLayer,
  useWarGoals,
  warGoalWindow,
  type WarGoalPick,
} from "./WarGoalLayer";
import { WarGoalEditPanel } from "./WarGoalEditPanel";
import {
  INSIGNIA_WINDOW,
  INTEL_VIEW,
  IntelAgencyLayer,
  NO_INTEL_PICK,
  insigniaWindow,
  intelAgencyWindow,
  useIntelAgency,
  type IntelPick,
} from "./IntelAgencyLayer";
import { IntelAgencyEditPanel } from "./IntelAgencyEditPanel";
import {
  EXILES_WINDOW,
  SUBJECTS_WINDOW,
  SubjectsLayer,
  exilesWindow,
  subjectsWindow,
  useSubjects,
} from "./SubjectsLayer";
import {
  FACTION_VIEW,
  FactionLayer,
  factionTabOf,
  factionWindow,
  useFaction,
  type FactionTab,
} from "./FactionLayer";
import { AutonomyEditPanel } from "./AutonomyEditPanel";
import {
  FacilitiesLayer,
  researchTabWindow,
  useSpecialProjects,
  type GuiFacility,
  type ResearchTab,
} from "./FacilitiesLayer";
import { SpecialProjectEditPanel } from "./SpecialProjectEditPanel";
import {
  SCIENTISTS_WINDOW,
  ScientistsLayer,
  scientistsWindow,
  useScientists,
} from "./ScientistsLayer";
import { ScientistEditPanel } from "./ScientistEditPanel";
import {
  INTEL_UPGRADES_WINDOW,
  IntelUpgradesLayer,
  intelUpgradesWindow,
} from "./IntelUpgradesLayer";
import { IntelUpgradeEditPanel } from "./IntelUpgradeEditPanel";
import {
  ARMY_OVERVIEW,
  ArmyOverviewLayer,
  armyOverviewWindow,
  overviewMode,
  useAirOverview,
  useArmyOverview,
  useNavyOverview,
  type GuiArmyDivision,
  type OverviewMode,
} from "./ArmyOverviewLayer";
import { DivisionEditPanel } from "./DivisionEditPanel";
import { OOBUnitEditPanel, type OOBUnit } from "./OOBUnitEditPanel";
import { SubUnitEditPanel } from "./SubUnitEditPanel";
import { SuperEventEditPanel } from "./SuperEventEditPanel";
import {
  gameViewWindowName,
  withWorkbenchCountry,
  workbenchHrefForScreen,
  workbenchWindowName,
} from "@/lib/gameViewLinks";
import {
  isSuperEventWindow,
  superEventPictureUrl,
  superEventWindow,
  superEventWindowName,
  useProjectTag,
  useSuperEvent,
} from "./superEventModel";
import type { SuperEventRow } from "@/lib/api/scenario";
import type { UiPanel } from "@/types";
import {
  PROGRAM_WINDOW,
  ProgramWindowLayer,
  programWindow,
} from "./ProgramWindowLayer";
import { useProjectPage } from "./ProjectPageLayer";
import type { DesignDraft } from "./designEdit";
import {
  DESIGNER_VIEWS,
  EquipmentDesignerLayer,
  TANK_DESIGNER,
  equipmentDesignerWindow,
  isDesignerView,
  useEquipmentDesigner,
} from "./EquipmentDesignerLayer";
import { POLITICS, usePolitics } from "./politicsModel";
import { ResearchLayer, TECHNOLOGY } from "./ResearchLayer";
import { PoliticsLayer } from "./PoliticsLayer";
import { STATE_VIEW, StateLayer, useStates } from "./StateLayer";
import { StateEditPanel } from "./StateEditPanel";
import { BookmarkEditPanel } from "./BookmarkEditPanel";
import { EventLayer } from "./EventLayer";
import { EVENT_WINDOW, useEventWindow, type GuiEvent } from "./eventModel";
import { EventEditPanel } from "./EventEditPanel";
import { EventSwitcher } from "./EventSwitcher";
import { GameWorldMap, unitCounters, useGameWorldMap } from "./GameWorldMap";
import { ProvinceEditPanel } from "./ProvinceEditPanel";
import { MapStudio } from "./map-studio/MapStudio";
import type { PickScope } from "@/components/workbench/world-map/worldMapModel";

const idOf = (node: GuiNode) => (node as GuiNode & { _id?: string })._id ?? "";

const DESIGNER_HQ_SHOWN = [
  "hq_view",
  "hq_overlay_images",
  "subunits_hq",
  "hq_title",
  "hq_badge",
];

const DESIGNER_HQ_PICKER_SHOWN = ["hq_title", "hq_view"];
const DESIGNER_HQ_PICKER_HIDDEN = [
  "hq_view/division_equipment_btn",
  "hq_view/division_priority_bg",
  "hq_view/can_paradrop_icon",
  "hq_view/special_forces_icon",
  "hq_view/divisions",
];
const DESIGNER_HQ_HIDDEN = [
  "non_hq_view",
  "subunits",
  "division_names",
  "combat_width_view",
  "division_composition_label",
  "division_title",
  "template_emblem_slot/division_template_icon",
];
const DESIGNER_EMPTY_HIDDEN = [
  "non_hq_view/can_paradrop_icon",
  "non_hq_view/special_forces_icon",
];

const DESIGNER_PICKER_HIDDEN = [
  "non_hq_view",
  "division_names",
  "template_emblem_slot",
  "hardness_stat",
  "org_stat",
  "strength_stat",
  "speed_stat",
  "combat_width_view",
  "division_composition_label",
  "subunits",
];
import {
  ACCELERATED,
  COMPANIONS,
  DECELERATED,
  RUNTIME_HIDDEN,
  RUNTIME_PRINTED_OVER,
  RUNTIME_SHOWN,
  RUNTIME_TEXT,
  START_CLOCK,
  TOPBAR_TARGETS,
  basePanelOf,
  opensOverWith,
  hotkeyOf,
  isRuntimeEffect,
  STEADY_GLOWS,
  matchNamePath,
  nextClock,
  resolveActivation,
  restPlacement,
  shortcutsOf,
  topbarButtonFor,
  topbarControlOf,
  type TopbarControl,
} from "./gameNavigation";

const SCREEN_W = 1920;
const SCREEN_H = 1080;

const passedOn = new WeakSet<Event>();

function isBackdropHit(el: HTMLElement): boolean {
  return (el.getAttribute("title") ?? "").startsWith("background ");
}

const UI_EDIT_STAGE_CSS = [
  '[data-ui-edit-stage] [role="button"], [data-ui-edit-stage] button { pointer-events: none !important; }',
  '[data-ui-edit-stage] div[style*="background-image"]:not([data-sprite]) { pointer-events: none !important; }',
  "[data-ui-edit-stage] [data-sprite] { pointer-events: auto !important; cursor: pointer; outline: 1px dashed rgba(52,211,153,0.55); outline-offset: -1px; }",
  "[data-ui-edit-stage] [data-sprite]:hover { outline-color: rgba(52,211,153,0.95); }",
  "[data-ui-edit-stage] [data-ui-picked] { outline: 2px solid rgba(52,211,153,0.95) !important; outline-offset: 1px; }",
].join("\n");

const FOCUS_FRAME = {
  left: 20,
  top: 150,
  width: SCREEN_W - 60,
  height: SCREEN_H - 230,
};
const NO_SWAPS: SpriteSwaps = {};
const NO_EDITS: Readonly<Record<string, string>> = {};
const NO_MIO_FILTERS: ReadonlySet<MioCategory> = new Set();
const NO_TEXTS: Readonly<Record<string, string>> = {};

const MUSIC_SLIDE_MS = 450;

const PINNED_SIZES = new Set([
  "topbar",
  MIO_LIST_WINDOW,
  MIO_DETAIL_WINDOW,
  LEADER_DETAILS,
  LOGISTICS,
  PRODUCTION,
  POLITICS,

  DIPLOMACY,
  ...Object.values(TOPBAR_TARGETS),
]);
const PINNED_NEGATIVE = new Set([
  MIO_LIST_WINDOW,
  MIO_DETAIL_WINDOW,
  LOGISTICS,
  DIPLOMACY,
  ...Object.values(TOPBAR_TARGETS),
]);

function useGameWindow(projectId: number, name: string | null) {
  return useSWR<GuiWindow>(
    name ? ["game-view-window", projectId, name] : null,
    async () => {
      guiRenderApi.setProjectId(projectId);
      const fetched = await guiRenderApi.getWindow(name as string);
      const win = PINNED_SIZES.has(fetched.name)
        ? {
            ...fetched,
            node: pinFullSizes(fetched.node, SCREEN_W, SCREEN_H, {
              negative: PINNED_NEGATIVE.has(fetched.name),
            }),
          }
        : fetched;
      assignIds(win.node);
      return win;
    },
    { revalidateOnFocus: false, keepPreviousData: false },
  );
}

function defaultScenario(
  bookmarks: GuiBookmarks | undefined,
): { index: number; date: string } | null {
  const marks = bookmarks?.bookmarks ?? [];
  if (!marks.length) return null;
  const at = Math.max(
    0,
    marks.findIndex((mark) => (mark as { default?: boolean }).default),
  );
  return { index: at, date: marks[at].date };
}

function defaultStateHidden(
  win: GuiWindow,
  revealed: readonly string[] = [],
  shown: string | null = null,

  extra: readonly string[] = [],

  alsoShown: readonly string[] = [],
): Set<string> {
  const hidden = new Set<string>();
  for (const group of win.overlap_groups ?? []) {
    const chosen =
      group.members.find((member) => member.name === shown)?.id ??
      group.default;
    for (const member of group.members)
      if (member.id !== chosen) hidden.add(member.id);
  }

  for (const namePath of [...(RUNTIME_SHOWN[win.name] ?? []), ...alsoShown]) {
    for (const node of matchNamePath(win.node, namePath))
      hidden.delete(idOf(node));
  }
  for (const namePath of RUNTIME_HIDDEN[win.name] ?? []) {
    if (revealed.includes(namePath)) continue;
    for (const node of matchNamePath(win.node, namePath))
      hidden.add(idOf(node));
  }
  const steady = STEADY_GLOWS[win.name] ?? [];
  for (const row of flattenTree(win.node)) {
    if (isRuntimeEffect(row.node.name) && !steady.includes(row.node.name ?? ""))
      hidden.add(row.id);
  }
  for (const namePath of extra) {
    for (const node of matchNamePath(win.node, namePath))
      hidden.add(idOf(node));
  }

  for (const namePath of revealed) {
    for (const node of matchNamePath(win.node, namePath))
      hidden.delete(idOf(node));
  }
  return hidden;
}

function useRuntimeTexts(projectId: number, windowName: string) {
  const keys = useMemo(
    () => [...new Set(Object.values(RUNTIME_TEXT[windowName] ?? {}))].sort(),
    [windowName],
  );
  return useSWR<Record<string, string>>(
    keys.length ? ["game-view-loc", projectId, keys.join(",")] : null,
    async () => {
      guiRenderApi.setProjectId(projectId);
      return guiRenderApi.loc(keys);
    },
    { revalidateOnFocus: false },
  );
}

function runtimeTextOverrides(
  win: GuiWindow,
  texts: Record<string, string> | undefined,
): Record<string, string> {
  const out: Record<string, string> = {};
  if (!texts) return out;
  for (const [namePath, key] of Object.entries(RUNTIME_TEXT[win.name] ?? {})) {
    const value = texts[key];
    if (value === undefined) continue;
    for (const node of matchNamePath(win.node, namePath))
      out[idOf(node)] = value;
  }
  return out;
}

function subscribeResize(onChange: () => void) {
  window.addEventListener("resize", onChange);
  return () => window.removeEventListener("resize", onChange);
}
const viewportSnapshot = () => `${window.innerWidth}:${window.innerHeight}`;
const serverViewportSnapshot = () => `${SCREEN_W}:${SCREEN_H}`;

type MountedLayerProps = Omit<
  React.ComponentProps<typeof ScreenLayer>,
  "win" | "owner" | "slideIn"
>;

function MountedPanelLayer({
  windowName,
  parentX,
  parentY,
  onMove,
  editPropsFor,
  frame,
  onWords,
  ...layer
}: {
  windowName: string;
  parentX: number;
  parentY: number;

  onMove?: (
    win: GuiWindow,
    nodeName: string | null | undefined,
    to: LayoutOverride,
  ) => Promise<unknown>;

  editPropsFor?: (owner: string) => Partial<MountedLayerProps>;

  frame?: WindowFrame;

  onWords?: (
    win: GuiWindow,
    nodeName: string | null | undefined,
    words: string,
  ) => Promise<unknown>;
} & MountedLayerProps) {
  const tComp = useTranslations("gameViewEdit.components");
  const win = useGameWindow(layer.projectId, windowName);
  const shown = win.data ?? null;

  const [moving, setMoving] = useState<Record<string, LayoutOverride>>({});

  const [frameDrag, setFrameDrag] = useState<FrameDrag | null>(null);

  const [wording, setWording] = useState<{
    id: string;
    nodeName: string | null;
    box: Box;
    words: string;
    tall: boolean;
  } | null>(null);

  const [hint, setHint] = useState<{
    guides: Guide[];
    chip: { x: number; y: number; text: string };
    into: Box | null;
    box: Box;
  } | null>(null);

  const scrolls = useComponentScrolls();

  const room = layer.uiEdit ? 0.5 : 0;
  const drawn = useMemo(
    () =>
      shown
        ? { ...shown, node: asTheGameClips(shown.node, scrolls, room) }
        : null,
    [shown, scrolls, room],
  );

  const layout = useMemo(() => {
    if (!drawn) return null;
    const rects = measureWindow(
      drawn.node,
      drawn.sprites,
      SCREEN_W,
      SCREEN_H,
      undefined,
      { nativeButtonSprites: true },
    );
    const all = flattenTree(drawn.node);
    const rows = all.filter(
      (row) => row.depth > 0 && rects[row.id] && !rects[row.id].hidden,
    );
    const lists = all
      .filter(
        (row) => row.depth > 0 && scrollsItsContent(row.node) && rects[row.id],
      )
      .map((row) => {
        const box = rects[row.id];
        const content = all.find(
          (entry) =>
            entry.depth === row.depth + 1 &&
            entry.id.startsWith(`${row.id}/`) &&
            isWindowNode(entry.node),
        );
        const contentH = content
          ? Number(pointSize(content.node).h) || rects[content.id]?.h || 0
          : 0;
        return {
          name: row.node.name ?? "",
          box: { x: box.x, y: box.y, w: box.w, h: box.h },
          limit: Math.max(0, contentH - box.h) + Math.round(box.h * room),
        };
      });

    const grids = rows
      .filter(
        (row) =>
          row.node.type.toLowerCase() === "gridboxtype" &&
          row.node.name?.endsWith("_grid"),
      )
      .map((row) => {
        const raw = (row.node.raw ?? {}) as {
          slotsize?: { width?: number | string; height?: number | string };
          max_slots_horizontal?: number | string;
        };
        const box = rects[row.id];
        return {
          entry: `${(row.node.name ?? "").slice(0, -"_grid".length)}_entry`,
          box: { x: box.x, y: box.y, w: box.w, h: box.h },
          slot: {
            w: Number(raw.slotsize?.width) || box.w,
            h: Number(raw.slotsize?.height) || 30,
          },
          columns: Math.max(1, Number(raw.max_slots_horizontal) || 1),
        };
      });
    return {
      rects,
      rows,
      lists,
      grids,
      place: restPlacement(drawn.node, SCREEN_W, SCREEN_H),
    };
  }, [drawn, room]);
  const drop = (id: string) => {
    setHint(null);
    setMoving((current) => {
      const next = { ...current };
      delete next[id];
      return next;
    });
  };

  const holderOf = (id: string): string => {
    const parent = id.slice(0, id.lastIndexOf("/"));
    const grand = parent.slice(0, parent.lastIndexOf("/"));
    const holder = layout?.rows.find((row) => row.id === grand);
    return holder && scrollsItsContent(holder.node) ? grand : parent;
  };

  const snapped = (
    id: string,
    to: LayoutOverride,
  ): { to: LayoutOverride; hint: NonNullable<typeof hint> } | null => {
    const row = layout?.rows.find((entry) => entry.id === id);
    const rect = row && layout?.rects[id];
    if (!layout || !row || !rect) return null;
    const others = layout.rows
      .filter(
        (entry) =>
          entry.id !== id &&
          !entry.id.startsWith(`${id}/`) &&
          !isListContent(entry.id, layout.rows),
      )
      .map((entry) => layout.rects[entry.id]);
    const { x: wx, y: wy } = layout.place;
    if (to.x !== undefined || to.y !== undefined) {
      const ox = rect.x - (Number(row.node.position?.x) || 0);
      const oy = rect.y - (Number(row.node.position?.y) || 0);
      const box = snapBox(
        {
          x: ox + (to.x ?? (Number(row.node.position?.x) || 0)),
          y: oy + (to.y ?? (Number(row.node.position?.y) || 0)),
          w: rect.w,
          h: rect.h,
        },
        others,
        "move",
      );

      const own = holderOf(id);
      let into: Box | null = null;
      for (const entry of layout.rows) {
        if (
          entry.id === id ||
          entry.id.startsWith(`${id}/`) ||
          !isWindowNode(entry.node) ||
          isListContent(entry.id, layout.rows)
        )
          continue;
        const r = layout.rects[entry.id];
        const cx = box.x + box.w / 2;
        const cy = box.y + box.h / 2;
        if (
          cx < r.x ||
          cy < r.y ||
          cx >= r.x + r.w ||
          cy >= r.y + r.h ||
          r.w < box.w ||
          (!scrollsItsContent(entry.node) && r.h < box.h)
        )
          continue;
        into = entry.id === own ? null : r;
      }
      return {
        to: { x: box.x - ox, y: box.y - oy },
        hint: {
          guides: box.guides,
          chip: {
            x: box.x,
            y: box.y + box.h + 4,
            text: tComp("coords", { x: wx + box.x, y: wy + box.y }),
          },
          into,
          box,
        },
      };
    }
    const box = snapBox(
      { x: rect.x, y: rect.y, w: to.width ?? rect.w, h: to.height ?? rect.h },
      others,
      "resize",
    );
    return {
      to: {
        ...(to.width !== undefined ? { width: box.w } : {}),
        ...(to.height !== undefined ? { height: box.h } : {}),
      },
      hint: {
        guides: box.guides,
        chip: {
          x: box.x + box.w + 6,
          y: box.y + box.h - 10,
          text: tComp("size", { w: box.w, h: box.h }),
        },
        into: null,
        box,
      },
    };
  };
  if (!drawn || !layout) return null;
  const nodeName = (id: string) =>
    layout.rows.find((row) => row.id === id)?.node.name;
  const lists = layout.lists.filter((list) => list.limit > 0 && list.name);
  const moved = frameDrag?.kind === "move" ? frameDrag : null;
  return (
    <div
      data-mounted-panel={windowName}
      style={{
        position: "absolute",
        left: parentX + (moved?.dx ?? 0),
        top: parentY + (moved?.dy ?? 0),
        pointerEvents: "none",
      }}
    >
      <ScreenLayer
        win={drawn}
        owner={windowName}
        slideIn={false}
        {...layer}
        nativeButtonSprites
        containerHits
        onEditNode={
          onWords
            ? (id, node) => {
                const type = node.type.toLowerCase();
                if (
                  !type.endsWith("textboxtype") &&
                  !type.endsWith("buttontype")
                )
                  return;
                const rect = layout.rects[id];
                if (!rect) return;
                const words = node.text
                  ? (drawn.texts[node.text]?.display ?? "")
                  : "";

                const w = rect.w || Number(node.maxwidth) || 0;
                const h = rect.h || Number(node.maxheight) || 0;
                const tall = type.endsWith("textboxtype") && h > 40;
                setWording({
                  id,
                  nodeName: node.name ?? null,
                  box: { x: rect.x, y: rect.y, w, h },
                  words,
                  tall,
                });
              }
            : undefined
        }
        resizeAxes={onMove ? panelResizeAxes : undefined}
        layoutOverrides={moving}

        underlay={lists.map((list) => (
          <div
            key={`under-${list.name}`}
            data-scroll-list={list.name}
            style={{
              position: "absolute",
              left: list.box.x,
              top: list.box.y,
              width: list.box.w,
              height: list.box.h,
              pointerEvents: "auto",
            }}
          />
        ))}
        overlay={lists.map((list) => {
          const at = componentScroll(list.name);
          const to = (y: number) =>
            setComponentScroll(
              list.name,
              Math.round(Math.min(list.limit, Math.max(0, y))),
            );
          return (
            <Fragment key={`over-${list.name}`}>
              <PageScroll
                box={list.box}
                scroll={{ x: 0, y: at }}
                limits={{ x: 0, y: list.limit }}
                step={LIST_WHEEL_STEP}
                dragLeft={false}
                onScroll={(update) =>
                  to(update({ x: 0, y: componentScroll(list.name) }).y)
                }
              />
              <PageScrollbar
                box={list.box}
                scroll={at}
                limit={list.limit}
                onScroll={to}
              />
            </Fragment>
          );
        })}
        onLayoutPreview={
          onMove
            ? (id, to) => {
                const at = snapped(id, to);
                setMoving((current) => ({ ...current, [id]: at?.to ?? to }));
                setHint(at?.hint ?? null);
              }
            : undefined
        }
        onLayoutCommit={
          onMove
            ? (id, to) => {
                const at = snapped(id, to)?.to ?? to;
                setHint(null);
                setMoving((current) => ({ ...current, [id]: at }));
                void onMove(drawn, nodeName(id), at).finally(() => drop(id));
              }
            : undefined
        }
        onLayoutCancel={onMove ? drop : undefined}
      />
      {layout.grids.map((grid) => (
        <ListRows
          key={grid.entry}
          {...grid}
          left={layout.place.x + grid.box.x}
          top={layout.place.y + grid.box.y}
          layer={layer}
          editPropsFor={editPropsFor}
          onMove={onMove}
          onWords={onWords}
        />
      ))}
      {wording && onWords && (
        <InlineWords
          key={wording.id}
          box={{
            ...wording.box,
            x: layout.place.x + wording.box.x,
            y: layout.place.y + wording.box.y,
          }}
          words={wording.words}
          tall={wording.tall}
          onDone={(words) => {
            const at = wording;
            setWording(null);
            if (words !== null && words !== at.words)
              void onWords(drawn, at.nodeName, words);
          }}
        />
      )}
      {frame && layer.uiEdit && (
        <WindowFrameHandles
          frame={frame}
          place={layout.place}
          size={pointSize(drawn.node)}
          zoom={layer.zoom ?? 1}
          drag={frameDrag}
          onDrag={setFrameDrag}
        />
      )}
      {hint && (
        <div
          style={{
            position: "absolute",
            left: layout.place.x,
            top: layout.place.y,
            pointerEvents: "none",
          }}
        >
          {hint.into && (
            <div
              data-drop-into=""
              className="absolute rounded-sm bg-accent/15 outline-2 outline-offset-2 outline-dashed outline-accent"
              style={{
                left: hint.into.x,
                top: hint.into.y,
                width: hint.into.w,
                height: hint.into.h,
              }}
            />
          )}
          <div
            data-drag-box=""
            className="absolute rounded-sm outline-1 outline-dashed outline-accent"
            style={{
              left: hint.box.x,
              top: hint.box.y,
              width: hint.box.w,
              height: hint.box.h,
            }}
          />
          <SnapGuides guides={hint.guides} />
          <span
            data-drag-chip=""
            className="absolute whitespace-nowrap rounded bg-popover px-1.5 py-0.5 text-xs tabular-nums text-popover-foreground shadow"
            style={{ left: hint.chip.x, top: hint.chip.y }}
          >
            {hint.chip.text}
          </span>
        </div>
      )}
    </div>
  );
}

const NO_HIDDEN = new Set<string>();

function InlineWords({
  box,
  words,
  tall,
  onDone,
}: {
  box: Box;
  words: string;
  tall: boolean;
  onDone: (words: string | null) => void;
}) {
  const t = useTranslations("gameViewEdit.components");
  const [value, setValue] = useState(words);
  const done = useRef(false);
  const finish = (next: string | null) => {
    if (done.current) return;
    done.current = true;
    onDone(next);
  };
  const className =
    "absolute z-10 rounded-sm border-2 border-accent bg-popover px-1 text-sm text-popover-foreground shadow-lg outline-none";
  const keys = (e: React.KeyboardEvent) => {
    e.stopPropagation();
    if (e.key === "Escape") finish(null);
    else if (e.key === "Enter" && (!tall || e.ctrlKey || e.metaKey)) {
      e.preventDefault();
      finish(value);
    }
  };
  return tall ? (
    <textarea
      autoFocus
      data-inline-words=""
      aria-label={t("editWords")}
      value={value}
      onFocus={(e) => e.currentTarget.select()}
      onBlur={() => finish(value)}
      onPointerDown={(e) => e.stopPropagation()}
      onChange={(e) => setValue(e.target.value)}
      onKeyDown={keys}
      className={className}
      style={{
        left: box.x,
        top: box.y,
        width: Math.max(box.w, 120),
        height: Math.max(box.h, 48),
        pointerEvents: "auto",
      }}
    />
  ) : (
    <input
      autoFocus
      data-inline-words=""
      aria-label={t("editWords")}
      value={value}
      onFocus={(e) => e.currentTarget.select()}
      onBlur={() => finish(value)}
      onPointerDown={(e) => e.stopPropagation()}
      onChange={(e) => setValue(e.target.value)}
      onKeyDown={keys}
      className={className}
      style={{
        left: box.x,
        top: box.y,
        width: Math.max(box.w, 120),
        height: Math.max(box.h, 26),
        pointerEvents: "auto",
      }}
    />
  );
}

interface WindowFrame {
  label: string;

  picked: boolean;

  onPick: () => void;

  onMove: (x: number, y: number) => Promise<unknown>;
  onResize: (width: number, height: number) => Promise<unknown>;
}

interface FrameDrag {
  kind: "move" | "size";
  dx: number;
  dy: number;
}

function WindowFrameHandles({
  frame,
  place,
  size,
  zoom,
  drag,
  onDrag,
}: {
  frame: WindowFrame;
  place: { x: number; y: number };
  size: { w: number; h: number };
  zoom: number;
  drag: FrameDrag | null;
  onDrag: (drag: FrameDrag | null) => void;
}) {
  const t = useTranslations("gameViewEdit.components");
  const start = useRef<{
    x: number;
    y: number;
    kind: "move" | "size";
    pointerId: number;
  } | null>(null);
  const begin =
    (kind: "move" | "size") => (event: React.PointerEvent<HTMLElement>) => {
      if (event.button !== 0) return;
      event.stopPropagation();
      event.currentTarget.setPointerCapture?.(event.pointerId);
      start.current = {
        x: event.clientX,
        y: event.clientY,
        kind,
        pointerId: event.pointerId,
      };
      onDrag({ kind, dx: 0, dy: 0 });
    };
  const move = (event: React.PointerEvent<HTMLElement>) => {
    const at = start.current;
    if (!at || at.pointerId !== event.pointerId) return;
    onDrag({
      kind: at.kind,
      dx: Math.round((event.clientX - at.x) / zoom),
      dy: Math.round((event.clientY - at.y) / zoom),
    });
  };
  const end = (event: React.PointerEvent<HTMLElement>) => {
    const at = start.current;
    if (!at || at.pointerId !== event.pointerId) return;
    start.current = null;
    const dx = Math.round((event.clientX - at.x) / zoom);
    const dy = Math.round((event.clientY - at.y) / zoom);
    if (Math.abs(dx) < 3 && Math.abs(dy) < 3) {
      onDrag(null);
      if (at.kind === "move") frame.onPick();
      return;
    }
    const done =
      at.kind === "move"
        ? frame.onMove(place.x + dx, place.y + dy)
        : frame.onResize(Math.max(40, size.w + dx), Math.max(40, size.h + dy));
    void done.finally(() => onDrag(null));
  };
  const sizing = drag?.kind === "size" ? drag : null;
  const w = Math.max(40, size.w + (sizing?.dx ?? 0));
  const h = Math.max(40, size.h + (sizing?.dy ?? 0));
  return (
    <>
      {(frame.picked || sizing) && (
        <div
          data-window-outline=""
          className="absolute rounded-sm outline-2 outline-dashed outline-accent"
          style={{
            left: place.x,
            top: place.y,
            width: w,
            height: h,
            pointerEvents: "none",
          }}
        />
      )}
      <button
        type="button"
        data-window-grip={frame.label}
        title={t("windowGrip")}
        onPointerDown={begin("move")}
        onPointerMove={move}
        onPointerUp={end}
        onPointerCancel={() => {
          start.current = null;
          onDrag(null);
        }}
        className={`absolute flex max-w-60 touch-none items-center gap-1 truncate rounded border px-1.5 py-0.5 text-xs shadow ui-focus-ring ${
          frame.picked
            ? "border-accent bg-accent text-accent-foreground"
            : "border-border bg-popover text-popover-foreground"
        } cursor-move`}
        style={{ left: place.x, top: place.y - 26, pointerEvents: "auto" }}
      >
        <span aria-hidden>⠿</span>
        {frame.label}
      </button>
      <div
        role="presentation"
        data-window-resize=""
        onPointerDown={begin("size")}
        onPointerMove={move}
        onPointerUp={end}
        className="absolute touch-none rounded-sm border border-background bg-accent"
        style={{
          left: place.x + w - 7,
          top: place.y + h - 7,
          width: 12,
          height: 12,
          cursor: "nwse-resize",
          pointerEvents: "auto",
        }}
      />
    </>
  );
}

function ListRows({
  entry,
  box,
  slot,
  columns,
  left,
  top,
  layer,
  editPropsFor,
  onMove,
  onWords,
}: {
  entry: string;
  box: { w: number; h: number };
  slot: { w: number; h: number };
  columns: number;
  left: number;
  top: number;
  layer: MountedLayerProps;
  editPropsFor?: (owner: string) => Partial<MountedLayerProps>;
  onMove?: (
    win: GuiWindow,
    nodeName: string | null | undefined,
    to: LayoutOverride,
  ) => Promise<unknown>;
  onWords?: (
    win: GuiWindow,
    nodeName: string | null | undefined,
    words: string,
  ) => Promise<unknown>;
}) {
  const row = useGameWindow(layer.projectId, entry).data;
  const shownColumns = Math.max(
    1,
    Math.min(columns, Math.ceil(box.w / slot.w)),
  );
  const cells = Math.min(
    40,
    shownColumns * Math.max(1, Math.ceil(box.h / slot.h)),
  );
  return (
    <div
      data-list-rows={entry}
      style={{
        position: "absolute",
        left,
        top,
        width: box.w,
        height: box.h,
        overflow: "hidden",
        pointerEvents: "none",
      }}
    >
      {row &&
        Array.from({ length: cells - 1 }, (_, i) => i + 1).map((cell) => (
          <div
            key={cell}
            aria-hidden
            data-list-row-copy=""
            className="[&_*]:!pointer-events-none"
            style={{
              position: "absolute",
              left: (cell % shownColumns) * slot.w,
              top: Math.floor(cell / shownColumns) * slot.h,
              opacity: 0.45,
            }}
          >
            <GuiCanvas
              node={row.node}
              sprites={row.sprites}
              texts={row.texts}
              fonts={row.fonts}
              versions={{}}
              hidden={NO_HIDDEN}
              isolateId={null}
              selectedId={null}
              onSelect={() => {}}
              editMode={false}
              transparent
              baseW={SCREEN_W}
              baseH={SCREEN_H}
              nativeButtonSprites
            />
          </div>
        ))}
      <MountedPanelLayer
        windowName={entry}
        parentX={0}
        parentY={0}
        {...layer}
        {...editPropsFor?.(entry)}
        onMove={onMove}
        onWords={onWords}
        editPropsFor={editPropsFor}
      />
    </div>
  );
}

const LIST_WHEEL_STEP = 40;

function isWindowNode(node: GuiNode): boolean {
  return node.type.toLowerCase().endsWith("windowtype");
}

function scrollsItsContent(node: GuiNode): boolean {
  return (
    isWindowNode(node) &&
    Boolean(node.raw?.verticalscrollbar) &&
    Boolean(node.children?.some(isWindowNode))
  );
}

function isListContent(
  id: string,
  rows: readonly { id: string; node: GuiNode }[],
): boolean {
  const parent = id.slice(0, id.lastIndexOf("/"));
  const holder = rows.find((row) => row.id === parent);
  return Boolean(holder && scrollsItsContent(holder.node));
}

function pointSize(node: GuiNode): { w: number; h: number } {
  const size = node.size as
    | {
        width?: number | string;
        height?: number | string;
        x?: number | string;
        y?: number | string;
      }
    | undefined;
  return {
    w: Number(size?.width ?? size?.x) || 0,
    h: Number(size?.height ?? size?.y) || 0,
  };
}

function asTheGameClips(
  root: GuiNode,
  scrolls: ReadonlyMap<string, number>,
  room = 0,
): GuiNode {
  const lists: { name: string; limit: number }[] = [];
  let node = withContainerScroll(
    root,
    (n) => {
      if (
        n === root ||
        !isWindowNode(n) ||
        /^(no|false)$/i.test(String(n.raw?.clipping ?? ""))
      )
        return false;
      if (scrollsItsContent(n)) {
        const content = n.children?.find(isWindowNode);
        const view = pointSize(n).h;
        if (n.name)
          lists.push({
            name: n.name,
            limit:
              Math.max(0, (content ? pointSize(content).h : 0) - view) +
              Math.round(view * room),
          });
        return false;
      }
      return true;
    },
    { x: 0, y: 0 },
  );

  for (const { name, limit } of lists) {
    node = withContainerScroll(node, (n) => n.name === name, {
      x: 0,
      y: Math.min(limit, scrolls.get(name) ?? 0),
    });
  }
  return node;
}

function panelResizeAxes(
  node: GuiNode,
  sprite: GuiSprite | undefined,
): ResizeAxes {
  const type = node.type.toLowerCase();
  if (type.endsWith("windowtype") || type.endsWith("textboxtype")) return "xy";
  if (type.endsWith("buttontype")) return sprite?.cornered ? "xy" : "none";
  return "none";
}

function ScreenLayer({
  win,
  owner,
  slideIn,
  modal = false,
  projectId,
  onActivate,
  tag,
  versions,
  onPickCountry,
  bookmarkIndex,
  onPickScenario,
  onScenarioTile,
  date,
  template,
  onEditTemplate,
  onEditHQTemplate,
  techFolder,
  onPickFolder,
  doctrineFolder = null,
  onPickDoctrineFolder,
  eventId,
  eventSaved,
  superEvent = null,
  onPickEventField,
  decisionSaved,
  onPickDecision,
  techSaved,
  onPickTech,
  ideaGroup,
  onOpenIdeas,
  mioFilters = NO_MIO_FILTERS,
  onMioDetails,
  mioDetail = null,
  mioTab = "traits",
  onEditPerson,
  onEditSpirit,
  onAddSpirit,
  onEditParties,
  onEditCountry,
  onEditEquipment,
  onEditSubUnit,
  equipmentDesign = null,
  designDraft = null,
  onEditDesign,
  diplomacyTarget = null,
  onDiplomacyTarget,
  onDiplomacyAction,
  warGoalPick = NO_WARGOAL,
  onWarGoalPick,
  onCloseWarGoal,
  onEditWarGoal,
  intelPick = NO_INTEL_PICK,
  intelForming = false,
  onOpenInsignia,
  onOpenIntelUpgrades,
  onEditIntelUpgrade,
  overview = "army",
  onEditDivision,
  onEditUnit,
  onIntelPick,
  onIntelCreate,
  onIntelCancel,
  onCloseInsignia,
  onEditInsignia,
  onEditAutonomy,
  researchTab = "slots",
  onResearchTab,
  onOpenTechTree,
  facilityPick = null,
  onOpenFacility,
  onCloseProgram,
  onEditProject,
  projectPage = null,
  onOpenProjectPage,
  onOpenScientists,
  onEditScientist,
  onOpenEquipmentDesigner,
  onEditMioTrait,
  onLeaderDetails,
  leaderDetail = null,
  onOpenSeat,
  stateId,
  topbarUi = IDLE_TOPBAR,
  mapMode = "land",
  onOpenWindow,
  onUnsupportedControl,
  editMode = false,
  uiEdit = false,
  selectedId = null,
  onSelectElement,
  locEdits = NO_EDITS,
  guiTexts = NO_EDITS,
  layoutOverrides,
  onLayoutPreview,
  onLayoutCommit,
  onLayoutCancel,
  zoom,
  resizeAxes,
  nativeButtonSprites = false,
  containerHits = false,
  onEditNode,
  onMoveGuiNode,
  underlay,
  overlay,
}: {
  win: GuiWindow;
  owner: string;
  slideIn: boolean;

  modal?: boolean;
  projectId: number;
  onActivate: (owner: string, node: GuiNode) => void;

  tag?: string;
  versions?: SpriteSwaps;

  onPickCountry?: (tag: string, focusTree: string | null) => void;
  bookmarkIndex?: number;
  onPickScenario?: (bookmarkIndex: number, date: string) => void;

  onScenarioTile?: (bookmarkIndex: number) => void;

  date?: string | null;

  template?: string | null;
  onEditTemplate?: (name: string) => void;

  onEditHQTemplate?: (name: string) => void;

  techFolder?: string | null;
  onPickFolder?: (folder: string) => void;

  doctrineFolder?: string | null;
  onPickDoctrineFolder?: (folder: string) => void;

  eventId?: string | null;

  eventSaved?: Partial<GuiEvent>;

  superEvent?: SuperEventRow | null;

  onPickEventField?: (field: string) => void;

  decisionSaved?: DecisionEdits;

  onPickDecision?: (pick: DecisionPick) => void;

  techSaved?: Record<string, TechChange>;

  onPickTech?: (tech: GuiTechnology, baseTechCost: number | undefined) => void;

  ideaGroup?: string | null;

  onOpenIdeas?: (group: string) => void;

  mioFilters?: ReadonlySet<MioCategory>;

  onMioDetails?: (token: string) => void;

  mioDetail?: string | null;
  mioTab?: MioTab;

  onEditPerson?: (token: string, name: string) => void;

  onEditSpirit?: (spirit: { token: string; label?: string }) => void;

  onAddSpirit?: () => void;

  onEditParties?: (
    party: string | null,
    colors: Record<string, [number, number, number]>,
  ) => void;

  onEditCountry?: () => void;

  onEditEquipment?: (name: string) => void;

  onEditSubUnit?: (type: string) => void;

  equipmentDesign?: string | null;

  designDraft?: DesignDraft | null;

  diplomacyTarget?: string | null;
  onDiplomacyTarget?: (tag: string | null) => void;

  onDiplomacyAction?: (id: string) => void;

  warGoalPick?: WarGoalPick;
  onWarGoalPick?: (pick: WarGoalPick) => void;
  onCloseWarGoal?: () => void;

  onEditWarGoal?: (id: string) => void;

  intelPick?: IntelPick;
  intelForming?: boolean;
  onOpenInsignia?: () => void;

  onOpenIntelUpgrades?: () => void;

  onEditIntelUpgrade?: (id: string) => void;

  overview?: OverviewMode;

  onEditDivision?: (division: GuiArmyDivision) => void;

  onEditUnit?: (
    kind: "task_force" | "air_wing",
    oob: string,
    ordinal: number,
  ) => void;
  onIntelPick?: (pick: IntelPick) => void;
  onIntelCreate?: () => void;
  onIntelCancel?: () => void;
  onCloseInsignia?: () => void;

  onEditInsignia?: (item: { file: string; ordinal: number }) => void;

  onEditAutonomy?: (id: string) => void;

  researchTab?: ResearchTab;
  onResearchTab?: (tab: ResearchTab) => void;

  onOpenTechTree?: () => void;

  facilityPick?: GuiFacility | null;
  onOpenFacility?: (facility: GuiFacility) => void;
  onCloseProgram?: () => void;

  onEditProject?: (id: string) => void;

  projectPage?: string | null;
  onOpenProjectPage?: (id: string | null) => void;

  onOpenScientists?: () => void;

  onEditScientist?: (token: string) => void;

  onEditDesign?: (slot?: string) => void;

  onOpenEquipmentDesigner?: (name: string, kind: string) => void;

  onEditMioTrait?: (token: string) => void;

  onLeaderDetails?: (token: string) => void;

  leaderDetail?: string | null;

  onOpenSeat?: (slot: string) => void;

  stateId?: number | null;

  topbarUi?: TopbarUi;

  mapMode?: MapModeId;

  onOpenWindow?: (window: string) => void;

  onUnsupportedControl?: (control: string) => void;

  editMode?: boolean;

  uiEdit?: boolean;

  selectedId?: string | null;
  onSelectElement?: (selection: EditSelection) => void;

  locEdits?: Readonly<Record<string, string>>;

  guiTexts?: Readonly<Record<string, string>>;

  layoutOverrides?: Record<string, LayoutOverride>;
  onLayoutPreview?: (id: string, partial: LayoutOverride) => void;
  onLayoutCommit?: (id: string, partial: LayoutOverride) => void;
  onLayoutCancel?: (id: string) => void;
  zoom?: number;

  resizeAxes?: (node: GuiNode, sprite: GuiSprite | undefined) => ResizeAxes;
  nativeButtonSprites?: boolean;

  containerHits?: boolean;
  onEditNode?: (id: string, node: GuiNode) => void;

  onMoveGuiNode?: (
    win: GuiWindow,
    id: string,
    to: LayoutOverride,
  ) => Promise<unknown>;

  underlay?: React.ReactNode;
  overlay?: React.ReactNode;
}) {
  const tEdit = useTranslations("gameViewEdit");
  const place = restPlacement(win.node, SCREEN_W, SCREEN_H);

  const [techScroll, setTechScroll] = useState<
    { folder: string } & ScrollOffset
  >({ folder: "", x: 0, y: 0 });
  const techPage = techFolder ?? "";
  const techOffset = techScroll.folder === techPage ? techScroll : null;
  const onTechScroll = useCallback(
    (update: (now: ScrollOffset) => ScrollOffset) =>
      setTechScroll((now) => {
        const from =
          now.folder === techPage ? { x: now.x, y: now.y } : { x: 0, y: 0 };
        const next = update(from);
        return next.x === from.x && next.y === from.y && now.folder === techPage
          ? now
          : { folder: techPage, ...next };
      }),
    [techPage],
  );

  const [factionTab, setFactionTab] = useState<FactionTab>("goals");

  const designKey = `${tag}|${date}|${template}`;
  const [design, setDesign] = useState<{
    key: string;
    draft: DesignerDraft;
  } | null>(null);
  const [designPicker, setDesignPicker] = useState<{
    key: string;
    state: PickerState;
  } | null>(null);
  const draft = design?.key === designKey ? design.draft : null;
  const picker = designPicker?.key === designKey ? designPicker.state : null;

  const [equipmentOpen, setEquipmentOpen] = useState<string | null>(null);
  const equipmentView =
    win.name === DIVISION_DESIGNER && equipmentOpen === designKey;

  const leftCovered = !!picker || equipmentView;

  const [designPreview, setDesignPreview] = useState<{
    key: string;
    draft: DesignerDraft;
  } | null>(null);
  const [designSaving, setDesignSaving] = useState(false);
  const previewDraft =
    picker && designPreview?.key === designKey ? designPreview.draft : null;

  const designCells = draft ? encodeCells(draft) : null;
  const designer = useDesigner(
    projectId,
    win.name === DIVISION_DESIGNER ? (tag ?? null) : null,
    date ?? null,
    template ?? null,
    designCells,
  );
  const previewDesigner = useDesigner(
    projectId,
    win.name === DIVISION_DESIGNER && previewDraft ? (tag ?? null) : null,
    date ?? null,
    template ?? null,
    previewDraft ? encodeCells(previewDraft) : null,
  );

  const previewData = previewDraft ? previewDesigner.data : undefined;

  const savedDesigner = useDesigner(
    projectId,
    win.name === DIVISION_DESIGNER && (draft || previewDraft)
      ? (tag ?? null)
      : null,
    date ?? null,
    template ?? null,
  );
  const originalDraft =
    template === NEW_TEMPLATE
      ? EMPTY_DRAFT
      : savedDesigner.data
        ? draftOf(savedDesigner.data)
        : null;
  const costDraft = previewDraft ?? draft;
  const designCost =
    costDraft && originalDraft && designer.data?.costs
      ? saveCost(
          originalDraft,
          costDraft,
          designer.data.available ?? [],
          designer.data.costs,
        )
      : 0;

  const designEmpty =
    win.name === DIVISION_DESIGNER &&
    !!designer.data &&
    !designer.data.regiments.length &&
    !designer.data.support.length;
  const designHQ =
    win.name === DIVISION_DESIGNER && !!designer.data?.is_army_hq;
  const designBar = useDesignerBarTexts(
    projectId,
    win.name === DIVISION_DESIGNER,
  );
  const officerTips = useOfficerTooltipTexts(
    projectId,
    win.name === OFFICER_CORPS,
  );
  const designExperience = designer.data?.army_experience ?? 0;
  const designDirty =
    !!draft && !!originalDraft && !sameDraft(draft, originalDraft);
  const designOff = useMemo(() => {
    if (win.name !== DIVISION_DESIGNER) return undefined;
    const names = designerButtonsOff({
      dirty: designDirty && !designSaving,
      fresh: template === NEW_TEMPLATE,
      cost: designCost,
      experience: designExperience,
      empty: designEmpty,

      editing: true,
    });
    return new Set(
      names.flatMap((name) => matchNamePath(win.node, name).map(idOf)),
    );
  }, [
    win,
    template,
    designDirty,
    designSaving,
    designCost,
    designExperience,
    designEmpty,
  ]);

  const saveDesign = async () => {
    if (!draft || !tag || !designer.data || designSaving) return;
    const own = designer.data;
    const input = designerSavePayload(draft, {
      tag,
      date,
      template:
        template === NEW_TEMPLATE ? "" : (savedDesigner.data?.name ?? own.name),
      name: own.name,
      namesGroup: own.division_names_group ?? null,
      priority: own.priority,
    });
    setDesignSaving(true);
    try {
      guiRenderApi.setProjectId(projectId);
      const result = await guiRenderApi.saveDesigner(input);

      const savedName = result.template.template_name;
      try {
        const fresh = await guiRenderApi.designer(
          tag,
          date || undefined,
          savedName,
        );
        await globalMutate(
          designerKey(projectId, tag, date ?? null, savedName),
          fresh,
          { revalidate: false },
        );
      } catch {}
      const saved = {
        name: result.template.template_name,
        file: result.oob_file,
      };
      toast.success(
        result.started_oob
          ? tEdit("page.designSavedStartedOob", saved)
          : result.imported
            ? tEdit("page.designSavedImported", saved)
            : tEdit("page.designSaved", saved),
      );
      setDesign(null);
      setDesignPicker(null);
      setDesignPreview(null);

      (designHQ ? onEditHQTemplate : onEditTemplate)?.(
        result.template.template_name,
      );
      void globalMutate(isGameDataKey);
    } catch (err) {
      toast.error(
        tEdit("page.saveFailed", {
          error: err instanceof Error ? err.message : String(err),
        }),
      );
    } finally {
      setDesignSaving(false);
    }
  };
  const baseHidden = useMemo(
    () =>
      defaultStateHidden(
        win,
        [
          ...(picker ? ["designerregimentswindow"] : []),

          ...(equipmentView
            ? [
                "division_equipment_view",
                "division_equipment_view/division_title",
              ]
            : []),
          ...(designHQ
            ? leftCovered
              ? DESIGNER_HQ_PICKER_SHOWN
              : DESIGNER_HQ_SHOWN
            : []),
        ],
        techFolder ?? null,
        [
          ...(designEmpty ? DESIGNER_EMPTY_HIDDEN : []),
          ...(leftCovered ? DESIGNER_PICKER_HIDDEN : []),
          ...(designHQ ? DESIGNER_HQ_HIDDEN : []),
          ...(designHQ && leftCovered ? DESIGNER_HQ_PICKER_HIDDEN : []),
        ],
        designHQ
          ? leftCovered
            ? DESIGNER_HQ_PICKER_SHOWN
            : DESIGNER_HQ_SHOWN
          : [],
      ),
    [
      win,
      techFolder,
      picker,
      equipmentView,
      leftCovered,
      designEmpty,
      designHQ,
    ],
  );
  const runtimeTexts = useRuntimeTexts(projectId, win.name);

  const stats = useSWR<GuiCountryStats>(
    tag &&
      (win.name === "topbar" ||
        win.name === PRODUCTION ||
        win.name === TRADE ||
        win.name === LOGISTICS ||
        win.name === DIPLOMACY ||
        win.name === POLITICS ||
        win.name === TECHNOLOGY)
      ? ["game-view-stats", projectId, tag, date]
      : null,
    () => {
      guiRenderApi.setProjectId(projectId);
      return guiRenderApi.countryStats(tag as string, date || undefined);
    },
    { revalidateOnFocus: false },
  );

  const topbar = useTopbar({
    win,
    projectId,
    stats: stats.data,
    date,
    ui: topbarUi,
  });

  const politics = usePolitics({ win, projectId, stats: stats.data });

  const event = useEventWindow({
    win,
    projectId,
    eventId,
    tag,
    date,
    saved: eventSaved,
  });

  const mio = useMemo(
    () =>
      win.name === MIO_LIST_WINDOW ? mioListWindow(win, mioFilters) : null,
    [win, mioFilters],
  );

  const detailData = useIndustrialOrg(
    projectId,
    win.name === MIO_DETAIL_WINDOW ? (tag ?? null) : null,
    date,
    mioDetail,
  );
  const detail = useMemo(
    () =>
      win.name === MIO_DETAIL_WINDOW
        ? mioDetailWindow(win, detailData.data, mioTab)
        : null,
    [win, detailData.data, mioTab],
  );

  const rosterKind = ROSTERS[win.name]?.kind;
  const rosterData = useCommanderRoster(
    projectId,
    rosterKind ? (tag ?? null) : null,
    date,
    rosterKind,
  );
  const roster = useMemo(
    () => (rosterKind ? rosterWindow(win, rosterData.data) : null),
    [win, rosterKind, rosterData.data],
  );

  const leaderData = useLeaderDetails(
    projectId,
    win.name === LEADER_DETAILS ? (tag ?? null) : null,
    date,
    leaderDetail,
  );
  const leader = useMemo(
    () =>
      win.name === LEADER_DETAILS
        ? leaderDetailsWindow(win, leaderData.data)
        : null,
    [win, leaderData.data],
  );

  const doctrineData = useDoctrines(
    projectId,
    win.name === DOCTRINE_VIEW ? (tag ?? null) : null,
    date,
    doctrineFolder,
  );
  const doctrine = useMemo(
    () =>
      win.name === DOCTRINE_VIEW
        ? doctrineWindow(win, doctrineData.data)
        : null,
    [win, doctrineData.data],
  );

  const tankData = useEquipmentDesigner(
    projectId,
    isDesignerView(win.name) ? (tag ?? null) : null,
    date,
    equipmentDesign,
    designDraft,
    editMode,
  );
  const draftName = designDraft?.name;
  const tankDesigner = useMemo(
    () =>
      isDesignerView(win.name)
        ? equipmentDesignerWindow(win, tankData.data, draftName)
        : null,
    [win, tankData.data, draftName],
  );

  const viewedCountry =
    win.name === DIPLOMACY && diplomacyTarget && diplomacyTarget !== tag
      ? diplomacyTarget
      : null;
  const viewedDiplomacy = useDiplomacy(projectId, viewedCountry, date ?? null);
  const diplomacyTargetView = useMemo(
    () =>
      viewedCountry ? diplomacyTargetWindow(win, viewedDiplomacy.data) : null,
    [win, viewedCountry, viewedDiplomacy.data],
  );
  const diplomacyListHides = useMemo(
    () =>
      win.name === DIPLOMACY && !viewedCountry
        ? diplomacyListHidden(win)
        : null,
    [win, viewedCountry],
  );

  const techTreeData = useTechTree(
    projectId,
    win.name === TECH_TREE ? (tag ?? null) : null,
    date ?? null,
    techFolder ?? "",
  );
  const techFolderNames = useMemo(
    () => techTreeData.data?.folders.map((entry) => entry.name),
    [techTreeData.data],
  );
  const techTreeLaid = useMemo(
    () =>
      win.name === TECH_TREE ? techTreeWindow(win, techFolderNames) : null,
    [win, techFolderNames],
  );
  const techScrollX = techOffset?.x ?? 0;
  const techScrollY = techOffset?.y ?? 0;
  const techTree = useMemo(
    () =>
      techTreeLaid
        ? scrollTechTree(techTreeLaid, { x: techScrollX, y: techScrollY })
        : null,
    [techTreeLaid, techScrollX, techScrollY],
  );

  const designerLaid = useMemo(
    () => (win.name === DIVISION_DESIGNER ? designerWindow(win) : null),
    [win],
  );

  const warGoals = useWarGoals(
    projectId,
    win.name === WARGOAL_WINDOW ? (tag ?? null) : null,
    diplomacyTarget,
    date,
  );
  const warGoalView = useMemo(
    () =>
      win.name === WARGOAL_WINDOW
        ? warGoalWindow(win, warGoals.data, warGoalPick)
        : null,
    [win, warGoals.data, warGoalPick],
  );

  const intelOn =
    win.name === INTEL_VIEW ||
    win.name === INSIGNIA_WINDOW ||
    win.name === INTEL_UPGRADES_WINDOW;
  const intel = useIntelAgency(projectId, intelOn ? (tag ?? null) : null, date);
  const intelView = useMemo(
    () =>
      win.name === INTEL_VIEW
        ? intelAgencyWindow(win, intel.data, intelForming)
        : null,
    [win, intel.data, intelForming],
  );
  const insigniaView = useMemo(
    () =>
      win.name === INSIGNIA_WINDOW
        ? insigniaWindow(win, intel.data, intelPick)
        : null,
    [win, intel.data, intelPick],
  );

  const armyOverview = useArmyOverview(
    projectId,
    win.name === ARMY_OVERVIEW && overview === "army" ? (tag ?? null) : null,
    date,
  );
  const navyOverview = useNavyOverview(
    projectId,
    win.name === ARMY_OVERVIEW && overview === "navy" ? (tag ?? null) : null,
    date,
  );
  const airOverview = useAirOverview(
    projectId,
    win.name === ARMY_OVERVIEW && overview === "air" ? (tag ?? null) : null,
    date,
  );
  const armyView = useMemo(
    () =>
      win.name === ARMY_OVERVIEW
        ? armyOverviewWindow(
            win,
            overview,
            armyOverview.data,
            navyOverview.data,
            airOverview.data,
          )
        : null,
    [win, overview, armyOverview.data, navyOverview.data, airOverview.data],
  );

  const superEventView = useMemo(
    () =>
      isSuperEventWindow(win.name) && superEvent
        ? superEventWindow(
            win,
            superEvent,
            superEventPictureUrl(projectId, superEvent),
          )
        : null,
    [win, superEvent, projectId],
  );

  const intelUpgradesView = useMemo(
    () =>
      win.name === INTEL_UPGRADES_WINDOW
        ? intelUpgradesWindow(win, intel.data)
        : null,
    [win, intel.data],
  );

  const researchView = useMemo(
    () =>
      win.name === TECHNOLOGY ? researchTabWindow(win, researchTab) : null,
    [win, researchTab],
  );

  const scientistRoster = useScientists(
    projectId,
    win.name === SCIENTISTS_WINDOW ? (tag ?? null) : null,
    date,
  );
  const scientistsView = useMemo(
    () =>
      win.name === SCIENTISTS_WINDOW
        ? scientistsWindow(win, scientistRoster.data)
        : null,
    [win, scientistRoster.data],
  );

  const programView = useMemo(
    () =>
      win.name === PROGRAM_WINDOW
        ? programWindow(win, facilityPick, Boolean(projectPage))
        : null,
    [win, facilityPick, projectPage],
  );

  const subjectsView = useMemo(
    () => (win.name === SUBJECTS_WINDOW ? subjectsWindow(win) : null),
    [win],
  );

  const exilesView = useMemo(
    () => (win.name === EXILES_WINDOW ? exilesWindow(win) : null),
    [win],
  );

  const faction = useFaction(
    projectId,
    win.name === FACTION_VIEW ? (tag ?? null) : null,
    date,
  );
  const factionView = useMemo(
    () =>
      win.name === FACTION_VIEW
        ? factionWindow(win, faction.data, factionTab)
        : null,
    [win, faction.data, factionTab],
  );

  const productionData = useProduction(
    projectId,
    win.name === PRODUCTION ? (tag ?? null) : null,
    date ?? null,
  );
  const productionView = useMemo(
    () =>
      win.name === PRODUCTION
        ? { texts: productionTexts(win, productionData.data?.queue) }
        : null,
    [win, productionData.data],
  );

  const constructionData = useConstruction(
    projectId,
    win.name === CONSTRUCTION ? (tag ?? null) : null,
    date,
  );
  const constructionBuildings = useBuildings(
    projectId,
    win.name === CONSTRUCTION ? (tag ?? null) : null,
    date ?? null,
  );
  const constructionView = useMemo(
    () =>
      win.name === CONSTRUCTION
        ? constructionWindow(
            win,
            constructionData.data,
            constructionBuildings.data?.buildings,
          )
        : null,
    [win, constructionData.data, constructionBuildings.data],
  );

  const corps = useOfficerCorps(
    projectId,
    win.name === OFFICER_CORPS ? (tag ?? null) : null,
    date ?? null,
  );
  const spiritEntries = useSpiritEntries(
    projectId,
    win.name === OFFICER_CORPS ? spiritEntryNames(corps.data) : [],
  );

  const [officerScroll, setOfficerScroll] = useState<{
    tag: string;
    y: number;
  }>({ tag: "", y: 0 });
  const officerY = officerScroll.tag === (tag ?? "") ? officerScroll.y : 0;
  const onOfficerScroll = useCallback(
    (update: (now: ScrollOffset) => ScrollOffset) =>
      setOfficerScroll((now) => {
        const from = now.tag === (tag ?? "") ? now.y : 0;
        const next = update({ x: 0, y: from }).y;
        return next === from && now.tag === (tag ?? "")
          ? now
          : { tag: tag ?? "", y: next };
      }),
    [tag],
  );
  const officers = useMemo(
    () =>
      win.name === OFFICER_CORPS
        ? scrollOfficerCorps(
            officerCorpsWindow(win, corps.data, true, spiritEntries.data),
            officerY,
          )
        : null,
    [win, corps.data, spiritEntries.data, officerY],
  );

  const mapmodes = useMemo(
    () => (win.name === MAPMODES_WINDOW ? mapmodesWindow(win, mapMode) : null),
    [win, mapMode],
  );
  const shown =
    mapmodes?.win ??
    topbar?.win ??
    politics?.win ??
    event?.win ??
    mio?.win ??
    detail?.win ??
    roster?.win ??
    leader?.win ??
    officers ??
    techTree ??
    designerLaid ??
    doctrine?.win ??
    tankDesigner?.win ??
    diplomacyTargetView?.win ??
    warGoalView?.win ??
    intelView?.win ??
    insigniaView?.win ??
    subjectsView?.win ??
    exilesView?.win ??
    researchView?.win ??
    programView?.win ??
    scientistsView?.win ??
    intelUpgradesView?.win ??
    armyView?.win ??
    factionView?.win ??
    constructionView?.win ??
    superEventView?.win ??
    win;

  const [gameMoving, setGameMoving] = useState<Record<string, LayoutOverride>>(
    {},
  );
  const dropGameMove = useCallback(
    (id: string) =>
      setGameMoving((current) => {
        if (!(id in current)) return current;
        const next = { ...current };
        delete next[id];
        return next;
      }),
    [],
  );
  const layoutEditable =
    !!onMoveGuiNode &&
    !shown.ui_panel &&
    !!shown.file &&
    !!(shown.scaffold_hash || shown.current_scaffold_hash);
  const hidden = useMemo(() => {
    if (
      !mapmodes &&
      !topbar?.hidden.size &&
      !politics &&
      !mio &&
      !detail &&
      !leader &&
      !doctrine &&
      !tankDesigner &&
      !diplomacyTargetView &&
      !diplomacyListHides?.length &&
      !warGoalView &&
      !intelView &&
      !researchView?.hidden.length &&
      !armyView &&
      !superEventView &&
      !factionView &&
      !constructionView
    ) {
      return baseHidden;
    }
    const merged = new Set(baseHidden);
    mapmodes?.hidden.forEach((id) => merged.add(id));
    mapmodes?.shown.forEach((id) => merged.delete(id));
    topbar?.hidden.forEach((id) => merged.add(id));
    politics?.shown.forEach((id) => merged.delete(id));
    politics?.hidden.forEach((id) => merged.add(id));
    mio?.hidden.forEach((id) => merged.add(id));
    detail?.hidden.forEach((id) => merged.add(id));
    detail?.shown.forEach((id) => merged.delete(id));
    leader?.hidden.forEach((id) => merged.add(id));
    leader?.shown.forEach((id) => merged.delete(id));
    doctrine?.hidden.forEach((id) => merged.add(id));
    tankDesigner?.hidden.forEach((id) => merged.add(id));
    diplomacyTargetView?.hidden.forEach((id) => merged.add(id));
    diplomacyListHides?.forEach((id) => merged.add(id));
    warGoalView?.hidden.forEach((id) => merged.add(id));
    intelView?.hidden.forEach((id) => merged.add(id));
    researchView?.hidden.forEach((id) => merged.add(id));
    armyView?.hidden.forEach((id) => merged.add(id));
    superEventView?.hidden.forEach((id) => merged.add(id));
    factionView?.hidden.forEach((id) => merged.add(id));
    constructionView?.hidden.forEach((id) => merged.add(id));
    return merged;
  }, [
    baseHidden,
    mapmodes,
    topbar,
    politics,
    mio,
    detail,
    leader,
    doctrine,
    tankDesigner,
    diplomacyTargetView,
    diplomacyListHides,
    warGoalView,
    intelView,
    researchView,
    armyView,
    superEventView,
    factionView,
    constructionView,
  ]);

  const printedOverIds = useMemo(
    () =>
      uiEdit
        ? (RUNTIME_PRINTED_OVER[win.name] ?? []).flatMap((namePath) =>
            matchNamePath(win.node, namePath).map(idOf),
          )
        : [],
    [uiEdit, win],
  );
  const canvasHidden = useMemo(() => {
    if (!printedOverIds.length) return hidden;
    const kept = new Set(hidden);
    for (const id of printedOverIds) kept.delete(id);
    return kept;
  }, [hidden, printedOverIds]);

  const isPicker = win.name === IDEA_WINDOW;
  const seat =
    isPicker && ideaGroup && CHARACTER_SLOTS.has(ideaGroup) ? ideaGroup : null;
  const lawChoices = useIdeaChoices(
    projectId,
    isPicker && !seat ? (tag ?? null) : null,
    date,
    ideaGroup ?? null,
  );
  const seatPeople = useCharacters(
    projectId,
    seat ? (tag ?? null) : null,
    date ?? null,
  );
  const ideaChoices = useMemo(() => {
    if (!seat) return { data: lawChoices.data };
    const data = seatPeople.data;
    if (!data) return { data: undefined };
    const slot = data.categories
      .flatMap((category) => category.slots)
      .find((candidate) => candidate.name === seat);
    return {
      data: advisorChoices(
        data.advisors,
        seat,
        slot?.label ?? seat,
        editMode,
        slot?.select_text,
      ),
    };
  }, [seat, lawChoices.data, seatPeople.data, editMode]);

  const historyRow = useHistoryRow(
    projectId,
    tag ?? null,
    editMode && isPicker,
  );
  const chooseStart = async (token: string) => {
    const choices = ideaChoices.data;
    const row = historyRow.data ?? (await historyRow.mutate());
    if (!choices) return;
    if (!row) {
      toast.error(tEdit("panel.notInProjectHistory"));
      return;
    }
    const person = choices.people?.find((p) => p.token === token);
    const extra = person?.idea
      ? withAdvisor(row.add_ideas_extra ?? [], person.idea, !person.hired)
      : withChosenIdea(
          row.add_ideas_extra ?? [],
          choices.ideas.map((idea) => idea.token),
          token,
        );
    try {
      await historyFilesApi.update(projectId, row.id, {
        add_ideas_extra: extra,
      });
      await historyRow.mutate();
      await globalMutate(
        (key) =>
          Array.isArray(key) &&
          [
            "game-view-idea-choices",
            "game-view-characters",
            "game-view-stats",
          ].includes(String(key[0])),
      );
      const name =
        person?.name ??
        choices.ideas.find((idea) => idea.token === token)?.label ??
        token;
      toast.success(
        tEdit(
          person?.idea && person.hired
            ? "politics.startUnhired"
            : "politics.startChosen",
          { name },
        ),
      );
    } catch (err) {
      toast.error(err instanceof Error ? err.message : String(err));
    }
  };

  const doctrineHistory = useHistoryRow(
    projectId,
    tag ?? null,
    editMode && win.name === DOCTRINE_VIEW,
  );
  const chooseDoctrine = async (choice: DoctrineChoice) => {
    const row = doctrineHistory.data ?? (await doctrineHistory.mutate());
    if (!row) {
      toast.error(tEdit("panel.notInProjectHistory"));
      return;
    }
    const patch = doctrinePatch(row, choice);
    if (!patch) {
      if (choice.from !== choice.to)
        toast.error(tEdit("page.doctrineNotInHistory"));
      return;
    }
    try {
      await historyFilesApi.update(projectId, row.id, patch);
      await doctrineHistory.mutate();
      await globalMutate(isGameDataKey);
      const picked =
        choice.kind === "grand"
          ? doctrineData.data?.grand_choices?.find((c) => c.token === choice.to)
          : doctrineData.data?.tracks
              ?.flatMap((t) => t.choices)
              .find((c) => c.token === choice.to);
      toast.success(
        tEdit(
          choice.kind === "grand"
            ? "page.doctrineStartGrand"
            : "page.doctrineStartSub",
          { name: picked?.label ?? choice.to },
        ),
      );
    } catch (err) {
      toast.error(err instanceof Error ? err.message : String(err));
    }
  };
  const runtimeOverrides = useMemo(
    () => ({
      ...runtimeTextOverrides(win, runtimeTexts.data),
      ...officerCorpsTextOverrides(win),
      ...designerTextOverrides(
        win,
        designCost,
        designer.data?.army_experience ?? 0,
        {
          data: previewData ?? designer.data,
          current: designer.data,
          texts: designBar.data,
        },
      ),
      ...topbar?.textOverrides,
      ...politics?.textOverrides,
      ...ideaWindowTexts(win, ideaChoices.data),
      ...detail?.texts,
      ...roster?.texts,
      ...leader?.texts,
      ...doctrine?.texts,
      ...tankDesigner?.texts,
      ...diplomacyTargetView?.texts,
      ...warGoalView?.texts,
      ...intelView?.texts,
      ...insigniaView?.texts,
      ...programView?.texts,
      ...scientistsView?.texts,
      ...intelUpgradesView?.texts,
      ...armyView?.texts,
      ...superEventView?.texts,
      ...factionView?.texts,
      ...constructionView?.texts,
      ...productionView?.texts,
    }),
    [
      win,
      runtimeTexts.data,
      designCost,
      designer.data,
      previewData,
      designBar.data,
      topbar,
      politics,
      ideaChoices.data,
      detail,
      roster,
      leader,
      doctrine,
      tankDesigner,
      diplomacyTargetView,
      warGoalView,
      intelView,
      insigniaView,
      programView,
      scientistsView,
      intelUpgradesView,
      armyView,
      superEventView,
      factionView,
      constructionView,
      productionView,
    ],
  );
  const textOverrides = useMemo(
    () => ({
      ...locEditOverrides(shown, locEdits),
      ...guiTextOverrides(shown, guiTexts),

      ...Object.fromEntries(printedOverIds.map((id) => [id, ""])),
      ...runtimeOverrides,
    }),
    [shown, locEdits, guiTexts, printedOverIds, runtimeOverrides],
  );

  const politicsPeople = useCharacters(
    projectId,
    win.name === POLITICS && tag ? tag : null,
    date ?? null,
  );
  const leaderPrinted = useMemo(
    () =>
      win.name === POLITICS
        ? leaderNamePrinted(shown, politicsPeople.data?.leader)
        : {},
    [win.name, shown, politicsPeople.data],
  );
  const runtimeSources = useMemo(
    () => ({
      ...fixedTextSources(win),
      ...designerTextSources(win),
      ...politics?.sources,
      ...roster?.sources,
      ...leader?.sources,
      ...detail?.sources,
      ...doctrine?.sources,
      ...warGoalView?.sources,
      ...factionView?.sources,
    }),
    [win, politics, roster, leader, detail, doctrine, warGoalView, factionView],
  );

  const swapped = useMemo(
    () => applySwaps(shown.sprites, versions ?? NO_SWAPS),
    [shown.sprites, versions],
  );
  const { x, y, fromX, fromY, ms } = place;

  const animateIn = useCallback(
    (el: HTMLDivElement | null) => {
      if (!el || !ms) return;
      el.animate(
        [
          { transform: `translate(${fromX - x}px, ${fromY - y}px)` },
          { transform: "translate(0, 0)" },
        ],
        { duration: ms, easing: "cubic-bezier(0, 0, 0.2, 1)" },
      );
    },
    [x, y, fromX, fromY, ms],
  );
  return (
    <div
      ref={slideIn ? animateIn : undefined}

      data-screen-layer={owner}
      style={{ position: "absolute", left: x, top: y, pointerEvents: "none" }}
    >
      {modal && (
        <div
          aria-hidden
          style={{
            position: "absolute",
            left: 0,
            top: 0,
            width: Number(win.node.size?.width) || 0,
            height: Number(win.node.size?.height) || 0,
            pointerEvents: "auto",
          }}
        />
      )}
      {underlay}
      <GuiCanvas
        node={shown.node}
        sprites={swapped.sprites}
        texts={win.texts}
        fonts={win.fonts}
        versions={swapped.versions}
        hidden={canvasHidden}
        textOverrides={textOverrides}
        isolateId={null}
        selectedId={selectedId}
        selectionOwner={owner}
        onSelect={(id, node, stack) =>
          onSelectElement?.({
            owner,
            win: shown,
            id,
            node,
            ...pickedText(id, leaderPrinted, runtimeOverrides, runtimeSources),
            stack: stack?.map((entry) => ({
              ...entry,
              ...pickedText(
                entry.id,
                leaderPrinted,
                runtimeOverrides,
                runtimeSources,
              ),
            })),
          })
        }

        editMode={uiEdit}
        transparent
        baseW={SCREEN_W}
        baseH={SCREEN_H}

        layoutOverrides={
          layoutOverrides ?? (layoutEditable ? gameMoving : undefined)
        }
        onLayoutPreview={
          onLayoutPreview ??
          (layoutEditable
            ? (id, to) =>
                setGameMoving((current) => ({
                  ...current,
                  [id]: { ...current[id], ...to },
                }))
            : undefined)
        }
        onLayoutCommit={
          onLayoutCommit ??
          (layoutEditable
            ? (id, to) => {
                setGameMoving((current) => ({
                  ...current,
                  [id]: { ...current[id], ...to },
                }));
                void onMoveGuiNode!(shown, id, to).finally(() =>
                  dropGameMove(id),
                );
              }
            : undefined)
        }
        onLayoutCancel={
          onLayoutCancel ?? (layoutEditable ? dropGameMove : undefined)
        }
        dragThreshold={layoutEditable ? 4 : 0}
        zoom={zoom}
        resizeAxes={
          resizeAxes ?? (layoutEditable ? panelResizeAxes : undefined)
        }
        containerHits={containerHits}
        onEditNode={onEditNode}

        nativeButtonSprites={nativeButtonSprites || !!shown.ui_panel}
        onActivate={(node) => {
          const factionTabHit =
            win.name === FACTION_VIEW ? factionTabOf(node.name) : null;
          if (factionTabHit) {
            setFactionTab(factionTabHit);
            return;
          }

          if (
            win.name === DIVISION_DESIGNER &&
            node.name === "division_equipment_btn"
          ) {
            setEquipmentOpen(designKey);
            setDesignPicker(null);
            return;
          }
          if (
            win.name === DIVISION_DESIGNER &&
            equipmentView &&
            matchNamePath(
              win.node,
              "division_equipment_view/close_button",
            ).some((hit) => idOf(hit) === idOf(node))
          ) {
            setEquipmentOpen(null);
            return;
          }
          if (win.name === DIVISION_DESIGNER && node.name === "save_button") {
            void saveDesign();
            return;
          }

          if (win.name === DIVISION_DESIGNER && node.name === "reset_button") {
            setDesign(null);
            setDesignPicker(null);
            return;
          }
          onActivate(owner, node);
        }}
        tooltipFor={
          win.name === DIVISION_DESIGNER
            ? (node) => designerTooltipFor(node, designBar.data)
            : win.name === OFFICER_CORPS
              ? (node) => officerCorpsTooltipFor(node, officerTips.data)
              : topbar?.tooltipFor
        }
        disabled={designOff}
      />
      {overlay}

      <TopbarClock
        model={topbar}
        texts={win.texts}
        fonts={win.fonts}
        onActivate={(node) => onActivate(owner, node)}
      />
      {onPickCountry && win.name === COUNTRY_PICKER && (
        <CountrySelectLayer
          win={win}
          projectId={projectId}
          initialTag={tag ?? null}
          bookmarkIndex={bookmarkIndex}
          onConfirm={onPickCountry}
        />
      )}
      {onPickScenario && win.name === SCENARIO_PICKER && (
        <ScenarioSelectLayer
          win={win}
          projectId={projectId}
          onConfirm={onPickScenario}
          onPicked={onScenarioTile}
        />
      )}
      <TopbarLayer
        model={topbar}
        stats={stats.data}
        onEditCountry={onEditCountry}
        onPressFlag={(flag) => onActivate(owner, flag)}
      />
      {topbar && (
        <AlertsLayer
          projectId={projectId}
          topbar={shown.node}
          stats={stats.data}
          onOpen={onOpenWindow}
          onUnavailable={onUnsupportedControl}
        />
      )}
      {tag && (
        <DeploymentLayer
          win={win}
          projectId={projectId}
          tag={tag}
          date={date}
          onEditTemplate={onEditTemplate}
          onUnavailable={onUnsupportedControl}
        />
      )}
      {tag && (
        <DesignerLayer
          win={shown}
          projectId={projectId}
          tag={tag}
          date={date}
          template={template}
          cells={designCells}
          draft={draft}
          original={originalDraft}
          equipmentView={equipmentView}
          onSave={
            editMode && designDirty && !designSaving
              ? () => void saveDesign()
              : undefined
          }
          onReset={
            editMode && designDirty && !designSaving
              ? () => {
                  setDesign(null);
                  setDesignPicker(null);
                }
              : undefined
          }

          onDraft={
            designSaving
              ? undefined
              : (next) => setDesign({ key: designKey, draft: next })
          }
          picker={picker}
          onPicker={
            designSaving
              ? undefined
              : (state) =>
                  setDesignPicker(state ? { key: designKey, state } : null)
          }
          onTemplate={onEditTemplate}
          preview={previewDraft}
          onPreview={(next) =>
            setDesignPreview(next ? { key: designKey, draft: next } : null)
          }
          onEditSubUnit={onEditSubUnit}
        />
      )}
      {tag && (
        <EquipmentDesignerLayer
          win={shown}
          projectId={projectId}
          tag={tag}
          date={date}
          name={equipmentDesign}
          draft={designDraft}
          editMode={editMode}
          onEditEquipment={onEditEquipment}
          onEditDesign={onEditDesign}
        />
      )}
      {tag && (
        <DoctrineLayer
          win={shown}
          projectId={projectId}
          tag={tag}
          date={date}
          folder={doctrineFolder}
          onPickFolder={(folder) => onPickDoctrineFolder?.(folder)}
          editMode={editMode}
          onChoose={
            editMode ? (choice) => void chooseDoctrine(choice) : undefined
          }
        />
      )}
      {tag && (
        <TechTreeLayer
          win={shown}
          projectId={projectId}
          tag={tag}
          date={date}
          folder={techFolder ?? ""}
          onPickFolder={onPickFolder}
          saved={techSaved}
          onPick={onPickTech}
          editable={editMode}
          scroll={techOffset ? { x: techScrollX, y: techScrollY } : undefined}
          onScroll={onTechScroll}
          dragScroll={!uiEdit}
        />
      )}
      {tag && (
        <ConstructionLayer
          win={win}
          projectId={projectId}
          tag={tag}
          date={date}
          onUnavailable={onUnsupportedControl}
        />
      )}
      {tag && (
        <DecisionsLayer
          win={win}
          projectId={projectId}
          tag={tag}
          date={date}
          saved={decisionSaved}
          onPick={onPickDecision}
        />
      )}
      {tag && (
        <TradeLayer
          win={win}
          projectId={projectId}
          tag={tag}
          date={date}
          stats={stats.data}
          onOpenCountry={onDiplomacyTarget}
        />
      )}
      {tag && (
        <LogisticsLayer
          win={win}
          projectId={projectId}
          tag={tag}
          date={date}
          stats={stats.data}
        />
      )}
      {tag && (
        <OfficerCorpsLayer
          win={shown}
          projectId={projectId}
          tag={tag}
          date={date}
          onOpenSeat={onOpenSeat}
          scroll={officerY}
          onScroll={onOfficerScroll}
        />
      )}
      {tag && (
        <ArmyHQTemplatesLayer
          win={shown}
          projectId={projectId}
          tag={tag}
          date={date}
          onEdit={onEditHQTemplate}
        />
      )}
      {tag && (
        <DiplomacyLayer
          win={win}
          projectId={projectId}
          tag={tag}
          date={date}
          stats={stats.data}
          target={diplomacyTarget}
          onTarget={onDiplomacyTarget}
          onAction={onDiplomacyAction}
        />
      )}
      {tag && onWarGoalPick && onCloseWarGoal && (
        <WarGoalLayer
          win={win}
          projectId={projectId}
          tag={tag}
          target={diplomacyTarget}
          date={date}
          pick={warGoalPick}
          onPick={onWarGoalPick}
          onClose={onCloseWarGoal}
          editMode={editMode}
          onEditWarGoal={onEditWarGoal}
        />
      )}
      {tag &&
        onOpenInsignia &&
        onIntelPick &&
        onIntelCreate &&
        onIntelCancel &&
        onCloseInsignia && (
          <IntelAgencyLayer
            win={win}
            projectId={projectId}
            tag={tag}
            date={date}
            pick={intelPick}
            forming={intelForming}
            onOpenInsignia={onOpenInsignia}
            onPick={onIntelPick}
            onCreate={onIntelCreate}
            onCancel={onIntelCancel}
            onClose={onCloseInsignia}
            editMode={editMode}
            onEditInsignia={onEditInsignia}
            onOpenUpgrades={onOpenIntelUpgrades}
          />
        )}
      {tag && onCloseInsignia && (
        <IntelUpgradesLayer
          win={win}
          projectId={projectId}
          tag={tag}
          date={date}
          onClose={onCloseInsignia}
          editMode={editMode}
          onEditUpgrade={onEditIntelUpgrade}
        />
      )}
      {tag && (
        <FactionLayer
          win={win}
          projectId={projectId}
          tag={tag}
          date={date}
          tab={factionTab}
          onOpenCountry={onDiplomacyTarget}
        />
      )}
      {tag && (
        <SubjectsLayer
          win={win}
          projectId={projectId}
          tag={tag}
          date={date}
          editMode={editMode}
          onEditLevel={onEditAutonomy}
          onOpenCountry={onDiplomacyTarget}
        />
      )}
      <ResearchLayer
        win={win}
        projectId={projectId}
        stats={stats.data}
        away={researchTab === "facilities"}
        onOpenTree={onOpenTechTree}
      />
      {tag && (
        <ArmyOverviewLayer
          win={win}
          projectId={projectId}
          tag={tag}
          date={date}
          mode={overview}
          editMode={editMode}
          onEditDivision={onEditDivision}
          onEditUnit={onEditUnit}
        />
      )}
      {tag && onResearchTab && (
        <FacilitiesLayer
          win={win}
          projectId={projectId}
          tag={tag}
          date={date}
          tab={researchTab}
          onTab={onResearchTab}
          onOpenFacility={onOpenFacility}
          onOpenScientists={onOpenScientists}
        />
      )}
      {tag && onCloseProgram && (
        <ScientistsLayer
          win={win}
          projectId={projectId}
          tag={tag}
          date={date}
          onClose={onCloseProgram}
          editMode={editMode}
          onEditScientist={onEditScientist}
        />
      )}
      {tag && onCloseProgram && (
        <ProgramWindowLayer
          win={win}
          projectId={projectId}
          tag={tag}
          date={date}
          facility={facilityPick}
          onClose={onCloseProgram}
          editMode={editMode}
          onEditProject={onEditProject}
          page={projectPage}
          onOpenPage={onOpenProjectPage}
        />
      )}
      <PoliticsLayer
        win={win}
        stats={stats.data}
        projectId={projectId}
        onEditParties={onEditParties}
      />
      <IdeaPickerLayer
        win={shown}
        projectId={projectId}
        choices={ideaChoices.data}
        onChoose={editMode ? (token) => void chooseStart(token) : undefined}
        onEditPerson={onEditPerson}
      />
      {tag && (
        <IndustrialOrgLayer
          win={shown}
          projectId={projectId}
          tag={tag}
          date={date}
          filters={mioFilters}
          onDetails={onMioDetails}
        />
      )}
      {tag && (
        <IndustrialOrgDetailLayer
          win={shown}
          projectId={projectId}
          tag={tag}
          date={date}
          token={mioDetail}
          tab={mioTab}
          onEditTrait={onEditMioTrait}
        />
      )}
      {tag && (
        <CommanderRosterLayer
          win={shown}
          projectId={projectId}
          tag={tag}
          date={date}
          onEditPerson={onEditPerson}
          onDetails={onLeaderDetails}
        />
      )}
      {tag && (
        <LeaderDetailsLayer
          win={shown}
          projectId={projectId}
          tag={tag}
          date={date}
          token={leaderDetail}
          onEditPerson={onEditPerson}
        />
      )}
      {tag && (
        <StateLayer
          win={win}
          projectId={projectId}
          tag={tag}
          date={date}
          stateId={stateId}
        />
      )}
      {tag && (
        <ProductionLayer
          win={win}
          projectId={projectId}
          tag={tag}
          date={date}
          onEditEquipment={onEditEquipment}
          onOpenDesigner={onOpenEquipmentDesigner}
          stats={stats.data}
        />
      )}
      <EventLayer
        win={shown}
        event={event?.event ?? null}
        projectId={projectId}
        onPickField={onPickEventField}
      />
      {tag && (
        <CharacterLayer
          win={win}
          projectId={projectId}
          tag={tag}
          date={date}
          hidden={hidden}
          versions={versions ?? NO_SWAPS}
          onOpenIdeas={onOpenIdeas}
          onEditPerson={onEditPerson}
          onEditSpirit={onEditSpirit}
          onAddSpirit={editMode ? onAddSpirit : undefined}
        />
      )}
    </div>
  );
}

let readsLanguage: string | null = null;

export default function GameViewPage() {
  const t = useTranslations("gameViewControls");
  const tEdit = useTranslations("gameViewEdit");
  const tContent = useTranslations("gameViewContent");
  const vanillaName = useVanillaTagName();

  const locale = useLocale();
  setGameLanguage(locale);
  useEffect(() => {
    if (readsLanguage !== null && readsLanguage !== locale)
      void globalMutate(() => true);
    readsLanguage = locale;
  }, [locale]);
  const params = useSearchParams();
  const projectId = Number(params.get("project_id")) || 1;
  useEffect(() => {
    const announce = (event: Event) => {
      const detail = (event as CustomEvent<GuiUnhandledActivateDetail>).detail;
      toast.info(
        t("unsupportedControl", {
          control: detail?.control || detail?.owner || "control",
        }),
      );
    };
    window.addEventListener(GUI_UNHANDLED_ACTIVATE_EVENT, announce);
    return () =>
      window.removeEventListener(GUI_UNHANDLED_ACTIVATE_EVENT, announce);
  }, [t]);
  const requestedPanel = params.get("panel") || "countrypoliticsview";
  const requestedTag = params.get("tag")?.toUpperCase() || null;

  const requestedTemplate = params.get("template");
  const requestedEvent = params.get("event");
  const requestedSuperEventId = Number(params.get("super_event")) || null;
  const requestedLeader = params.get("leader");
  const requestedStateId = Number(params.get("state")) || null;
  const requestedTechFolder = params.get("folder");
  const requestedDoctrineFolder = params.get("doctrine");
  const requestedFocusTree = params.get("tree");
  const [panel, setPanel] = useState<string | null>(() => requestedPanel);
  const [template, setTemplate] = useState<string | null>(
    () => requestedTemplate,
  );
  const [panelRequest, setPanelRequest] = useState(() => ({
    projectId,
    requestedPanel,
  }));
  const [templateRequest, setTemplateRequest] = useState(() => ({
    projectId,
    requestedTag,
    requestedTemplate,
  }));

  if (
    panelRequest.projectId !== projectId ||
    panelRequest.requestedPanel !== requestedPanel
  ) {
    setPanelRequest({ projectId, requestedPanel });
    setPanel(requestedPanel);
  }
  if (
    templateRequest.projectId !== projectId ||
    templateRequest.requestedTag !== requestedTag ||
    templateRequest.requestedTemplate !== requestedTemplate
  ) {
    setTemplateRequest({ projectId, requestedTag, requestedTemplate });
    setTemplate(requestedTemplate);
  }

  const [chosenEvent, setChosenEvent] = useState<string | null>(null);
  const eventId = chosenEvent || requestedEvent;

  const [eventSaved, setEventSaved] = useState<
    Record<string, Partial<GuiEvent>>
  >({});
  const [eventPanelOpen, setEventPanelOpen] = useState(true);

  const [superEventId, setSuperEventId] = useState<number | null>(
    () => requestedSuperEventId,
  );
  const superEventRow = useSuperEvent(projectId, superEventId);
  const projectTag = useProjectTag(projectId, true);
  const [eventFocus, setEventFocus] = useState<{
    field: string;
    at: number;
  } | null>(null);

  const [decisionPick, setDecisionPick] = useState<DecisionPick | null>(null);

  const [personPick, setPersonPick] = useState<{
    token: string;
    name: string;
  } | null>(null);

  const [spiritPick, setSpiritPick] = useState<{
    token: string;
    label?: string;
  } | null>(null);

  const [spiritStartOpen, setSpiritStartOpen] = useState(false);

  const [countryPanelOpen, setCountryPanelOpen] = useState(false);

  const [equipmentPick, setEquipmentPick] = useState<string | null>(null);

  const [equipmentDesign, setEquipmentDesign] = useState<string | null>(null);

  const [designPanel, setDesignPanel] = useState<{
    slot: string | null;
  } | null>(null);
  const [designDraft, setDesignDraft] = useState<DesignDraft | null>(null);

  const [diplomacyTarget, setDiplomacyTarget] = useState<string | null>(null);

  const [focusViewTag, setFocusViewTag] = useState<string | null>(null);

  const [warGoalPick, setWarGoalPick] = useState<WarGoalPick>(NO_WARGOAL);
  const [warGoalEdit, setWarGoalEdit] = useState<string | null>(null);

  const [intelPick, setIntelPick] = useState<IntelPick>(NO_INTEL_PICK);
  const [intelForming, setIntelForming] = useState(false);

  const [intelEdit, setIntelEdit] = useState<{
    file: string;
    ordinal: number;
  } | null>(null);

  const [autonomyEdit, setAutonomyEdit] = useState<string | null>(null);

  const [researchTab, setResearchTab] = useState<ResearchTab>("slots");

  const [facilityPick, setFacilityPick] = useState<GuiFacility | null>(null);

  const [projectPage, setProjectPage] = useState<string | null>(null);

  const [projectEdit, setProjectEdit] = useState<string | null>(null);

  const [upgradeEdit, setUpgradeEdit] = useState<string | null>(null);

  const [divisionEdit, setDivisionEdit] = useState<{
    oob: string;
    ordinal: number;
  } | null>(null);

  const [subUnitEdit, setSubUnitEdit] = useState<string | null>(null);

  const [unitEdit, setUnitEdit] = useState<{
    kind: OOBUnit["kind"];
    oob: string;
    ordinal: number;
  } | null>(null);

  const [scientistEdit, setScientistEdit] = useState<string | null>(null);
  const [partiesPick, setPartiesPick] = useState<{
    focus: string | null;
    colors: Record<string, [number, number, number]>;
  } | null>(null);

  const [mioTraitPick, setMioTraitPick] = useState<string | null>(null);
  const [decisionSaved, setDecisionSaved] = useState<DecisionEdits>({});

  const [techPick, setTechPick] = useState<{
    tech: GuiTechnology;
    baseTechCost?: number;
  } | null>(null);
  const [techSaved, setTechSaved] = useState<Record<string, TechChange>>({});

  const [ideaGroup, setIdeaGroup] = useState<string | null>(null);

  const [mioFilters, setMioFilters] =
    useState<ReadonlySet<MioCategory>>(NO_MIO_FILTERS);

  const [mioDetail, setMioDetail] = useState<string | null>(null);
  const [mioTab, setMioTab] = useState<MioTab>("traits");
  const openMioDetails = useCallback((token: string) => {
    setMioDetail(token);
    setMioTab("traits");
    setPanel(MIO_DETAIL_WINDOW);
  }, []);

  const [leaderDetail, setLeaderDetail] = useState<string | null>(
    () => requestedLeader,
  );
  const openLeaderDetails = useCallback((token: string) => {
    setLeaderDetail(token);
    setPanel(LEADER_DETAILS);
  }, []);

  const [pickerOver, setPickerOver] = useState<string | null>(null);

  const [designerOver, setDesignerOver] = useState<string | null>(null);
  const opensOver = useMemo(
    () => opensOverWith(pickerOver, designerOver),
    [pickerOver, designerOver],
  );
  const openIdeas = useCallback((group: string) => {
    const roster = MIO_SLOTS[group];
    if (roster) {
      setMioFilters(new Set([roster]));
      setPanel(MIO_LIST_WINDOW);
      return;
    }
    setIdeaGroup(group);
    setPickerOver(null);
    setPanel(IDEA_WINDOW);
  }, []);
  const openSeat = useCallback((slot: string) => {
    setIdeaGroup(slot);
    setPickerOver(OFFICER_CORPS);
    setPanel(IDEA_WINDOW);
  }, []);

  const [chosenState, setChosenState] = useState<number | null>(null);
  const stateId = chosenState || requestedStateId;
  const [statePanelOpen, setStatePanelOpen] = useState(true);

  const [techFolder, setTechFolder] = useState<string | null>(
    () => requestedTechFolder,
  );

  const [doctrineFolder, setDoctrineFolder] = useState<string | null>(
    () => requestedDoctrineFolder,
  );
  const currentRouteTargets = {
    projectId,
    panel: requestedPanel,
    tag: requestedTag,
    event: requestedEvent,
    superEventId: requestedSuperEventId,
    leader: requestedLeader,
    stateId: requestedStateId,
    techFolder: requestedTechFolder,
    doctrineFolder: requestedDoctrineFolder,
    template: requestedTemplate,
    focusTree: requestedFocusTree,
  };
  const [routeTargets, setRouteTargets] = useState(() => currentRouteTargets);
  const routeSurfaceChanged =
    routeTargets.projectId !== projectId ||
    routeTargets.panel !== requestedPanel ||
    routeTargets.tag !== requestedTag;
  if (
    routeSurfaceChanged ||
    routeTargets.event !== requestedEvent ||
    routeTargets.superEventId !== requestedSuperEventId ||
    routeTargets.leader !== requestedLeader ||
    routeTargets.stateId !== requestedStateId ||
    routeTargets.techFolder !== requestedTechFolder ||
    routeTargets.doctrineFolder !== requestedDoctrineFolder ||
    routeTargets.template !== requestedTemplate ||
    routeTargets.focusTree !== requestedFocusTree
  ) {
    setRouteTargets(currentRouteTargets);

    setPanel(requestedPanel);
    if (routeSurfaceChanged || routeTargets.event !== requestedEvent) {
      setChosenEvent(null);
      setEventPanelOpen(true);
    }
    if (
      routeSurfaceChanged ||
      routeTargets.superEventId !== requestedSuperEventId
    ) {
      setSuperEventId(requestedSuperEventId);
    }
    if (routeSurfaceChanged || routeTargets.leader !== requestedLeader) {
      setLeaderDetail(requestedLeader);
    }
    if (routeSurfaceChanged || routeTargets.stateId !== requestedStateId) {
      setChosenState(null);
      setStatePanelOpen(true);
    }
    if (
      routeSurfaceChanged ||
      routeTargets.techFolder !== requestedTechFolder
    ) {
      setTechFolder(requestedTechFolder);
    }
    if (
      routeSurfaceChanged ||
      routeTargets.doctrineFolder !== requestedDoctrineFolder
    ) {
      setDoctrineFolder(requestedDoctrineFolder);
    }
    if (routeSurfaceChanged) {
      setDiplomacyTarget(null);
      setFocusViewTag(null);
      setCountryPanelOpen(false);
    }
  }
  const editTemplate = useCallback((name: string) => {
    setTemplate(name);
    setDesignerOver(null);
    setPanel("countrydivisiondesignerview");
  }, []);

  const editHQTemplate = useCallback((name: string) => {
    setTemplate(name);
    setDesignerOver(ARMY_HQ_TEMPLATES);
    setPanel("countrydivisiondesignerview");
  }, []);

  const [selectedFocus, setSelectedFocus] = useState<FocusNode | null>(null);
  const clearInspectedContent = useCallback(() => {
    setSelectedFocus(null);
    setTechPick(null);
  }, []);

  const [editMode, setEditMode] = useState(() => params.get("edit") === "1");
  const [uiEdit, setUiEdit] = useState(() => params.get("edit") === "ui");
  const [selection, setSelection] = useState<EditSelection | null>(null);

  const [panelPick, setPanelPick] = useState<{
    panelId: number;
    path?: string | null;
    names?: string | null;
  } | null>(null);

  const [mapStudio, setMapStudio] = useState(
    () => params.get("mapstudio") === "1",
  );

  const mapShownWaiter = useRef<{ after: number; finish: () => void } | null>(
    null,
  );

  const [locEdits, setLocEdits] = useState<Record<string, string>>({});

  const [guiTexts, setGuiTexts] = useState<Record<string, string>>({});

  const [printedOver] = useState(createPrintedRegistry);

  const [mapProvince, setMapProvince] = useState(0);
  const [worldMapVersion, setWorldMapVersion] = useState(0);

  const [mapScope, setMapScope] = useState<PickScope>("state");
  const mapPick = useRef<((index: number) => boolean) | null>(null);

  const selectElement = useCallback(
    (picked: EditSelection) => {
      setSelectedFocus(null);

      setSelection(withPrintedOver(picked, printedOver));
      const envelope = picked.win.ui_panel;
      if (envelope)
        setPanelPick({
          panelId: envelope.panel_id,
          names: pickedControlNames(envelope, picked.node.name),
        });
    },
    [printedOver],
  );

  const pickInUiEdit = useCallback(
    (e: React.PointerEvent<HTMLDivElement>) => {
      if (e.button !== 0 || passedOn.has(e.nativeEvent)) return;
      const stage = e.currentTarget;
      const pickArt = (art: HTMLElement) => {
        const name = art.dataset.sprite ?? "";
        const frame = Number(art.dataset.frame) || 1;
        const at = art.getBoundingClientRect();
        const id = `${name}#${frame}@${Math.round(at.left)},${Math.round(at.top)}`;
        art.dataset.uiPicked = id;
        void layerSpriteMeta(projectId, name).then((meta) => {
          if (meta)
            selectElement(
              artSelection(
                "layer-art",
                id,
                name,
                meta,
                art.dataset.title || name,
                frame,
              ),
            );
        });
      };
      const target = e.target instanceof HTMLElement ? e.target : null;
      const art = target?.closest("[data-sprite]");
      if (art instanceof HTMLElement && stage.contains(art)) {
        e.stopPropagation();
        pickArt(art);
        return;
      }
      if (!target || target.dataset.hit === undefined || !isBackdropHit(target))
        return;
      const screen = target.closest("[data-screen-layer]");
      const ownCanvas = target.parentElement;
      const under = document
        .elementsFromPoint(e.clientX, e.clientY)
        .find(
          (el): el is HTMLElement =>
            el instanceof HTMLElement &&
            el !== target &&
            !!screen?.contains(el) &&
            !ownCanvas?.contains(el) &&
            ((el.dataset.hit !== undefined && !isBackdropHit(el)) ||
              el.dataset.sprite !== undefined),
        );
      if (!under) return;
      e.stopPropagation();
      if (under.dataset.sprite !== undefined) {
        pickArt(under);
        return;
      }
      const press = new PointerEvent("pointerdown", {
        bubbles: true,
        cancelable: true,
        clientX: e.clientX,
        clientY: e.clientY,
        button: 0,
        pointerId: e.pointerId,
        pointerType: e.pointerType,
        isPrimary: true,
      });
      passedOn.add(press);
      under.dispatchEvent(press);
    },
    [projectId, selectElement],
  );
  const stageRef = useRef<HTMLDivElement | null>(null);
  const openManualEdit = (mode: "ui" | "text") => {
    setMapStudio(false);
    if (mode === "ui") {
      setUiEdit((on) => !on);
      setEditMode(false);
    } else {
      setEditMode((on) => !on);
      setUiEdit(false);
      setMapProvince(0);
    }
    setSelection(null);
  };
  const closeMapStudio = useCallback(() => {
    setMapStudio(false);
    setWorldMapVersion((version) => version + 1);
    void globalMutate((key) => isGameViewRefreshKey(key, true));
  }, []);
  const toggleMapStudio = () => {
    setEditMode(false);
    setUiEdit(false);
    setMapProvince(0);
    setSelection(null);
    if (mapStudio) closeMapStudio();
    else setMapStudio(true);
  };
  useEffect(() => {
    for (const marked of document.querySelectorAll<HTMLElement>(
      "[data-ui-picked]",
    )) {
      if (!uiEdit || marked.dataset.uiPicked !== selection?.id)
        delete marked.dataset.uiPicked;
    }
  }, [uiEdit, selection]);

  const moveGuiNode = async (
    win: GuiWindow,
    id: string,
    to: LayoutOverride,
  ) => {
    const target = buildNamePathMap(win.node)[id];
    const file = win.file;
    if (!file || !addressable(target)) {
      toast.info(tEdit("layout.unnamed"));
      return;
    }
    const reread = () =>
      globalMutate(
        (cacheKey) =>
          Array.isArray(cacheKey) && cacheKey[0] === "game-view-window",
      );
    try {
      const before = await saveGuiLayout(file, target, to);
      recordChange({
        projectId,
        title: tEdit("layout.moved", {
          name: target.split("/").pop() ?? target,
        }),
        kind: `${file} · ${target}`,
        undo: async () => {
          await restoreGuiLayout(file, target, before);
          await reread();
        },
      });
      await reread();
    } catch (err) {
      toast.error(
        err instanceof GuiTextNotExportable
          ? tEdit("layout.notExportable")
          : tEdit("layout.failed"),
      );
    }
  };
  const editProps = (owner: string) => ({
    editMode,
    uiEdit,
    selectedId: selection?.owner === owner ? selection.id : null,
    onSelectElement: selectElement,
    locEdits,
    guiTexts,

    onMoveGuiNode: uiEdit ? moveGuiNode : undefined,
  });
  const savedTexts = useSWR<Record<string, string>>(
    editMode || uiEdit ? ["game-view-text-overrides", projectId] : null,
    () => {
      guiRenderApi.setProjectId(projectId);
      return guiRenderApi.textOverrides();
    },
    { revalidateOnFocus: false },
  );
  const [spriteVersions, setSpriteVersions] = useState<SpriteSwaps>({});

  const spriteChanged = useCallback(
    (
      gfx: string,
      width: number,
      height: number,
      action: "replace" | "restore",
    ) => {
      invalidateLayerSpriteMeta(projectId, gfx);
      setSpriteVersions((current) => ({
        ...current,
        [gfx]: { version: Date.now(), width, height },
      }));
      recordChange({
        projectId,
        title: tEdit(
          action === "restore" ? "page.spriteRestored" : "page.spriteReplaced",
          { gfx },
        ),
        kind: tEdit("page.spriteKind"),
      });

      void globalMutate(["game-view-frontend", projectId]);
    },
    [projectId, tEdit],
  );

  const editContext = useMemo<EditContextValue>(
    () => ({
      editMode: uiEdit,
      contentEditMode: editMode,
      selection: selection
        ? { owner: selection.owner, id: selection.id }
        : null,
      select: selectElement,
      locEdits,
      guiTexts,
      printedOver,
      swaps: spriteVersions,
    }),
    [
      uiEdit,
      editMode,
      selection,
      selectElement,
      locEdits,
      guiTexts,
      printedOver,
      spriteVersions,
    ],
  );

  const routeTag = requestedTag;
  const [chosenCountry, setChosenCountry] = useState<{
    projectId: number;
    routeTag: string | null;
    tag: string;
  } | null>(null);
  const chosenTag =
    chosenCountry?.projectId === projectId &&
    chosenCountry.routeTag === routeTag
      ? chosenCountry.tag
      : null;

  const tag = (chosenTag || routeTag || projectTag.data || "CHI").toUpperCase();

  const madeTag = chosenTag || routeTag || projectTag.data || null;
  useEffect(() => {
    speakFor(projectId, madeTag);
  }, [projectId, madeTag]);
  useEffect(
    () =>
      onCountryTakenIn((takenProject, taken) => {
        if (takenProject !== projectId) return;
        toast.success(
          tEdit("countryTakenIn", { name: vanillaName(taken) ?? taken }),
        );
        void globalMutate(["game-view-project-tag", projectId]);
      }),
    [projectId, tEdit, vanillaName],
  );

  const [chosenTrees, setChosenTrees] = useState<Record<string, string>>({});
  const currentFocusRoute = { projectId, routeTag, tree: requestedFocusTree };
  const [focusRoute, setFocusRoute] = useState(() => currentFocusRoute);
  if (
    focusRoute.projectId !== projectId ||
    focusRoute.routeTag !== routeTag ||
    focusRoute.tree !== requestedFocusTree
  ) {
    setFocusRoute(currentFocusRoute);
    setChosenTrees({});
  }
  const rememberFocusTree = useCallback(
    (countryTag: string, tree: string | null) => {
      const key = countryTag.toUpperCase();
      setChosenTrees((current) => {
        if (!tree && !(key in current)) return current;
        const next = { ...current };
        if (tree) next[key] = tree;
        else delete next[key];
        return next;
      });
    },
    [],
  );

  const [pickedScenario, setScenario] = useState<number | null>(null);
  const [pickedDate, setScenarioDate] = useState<string | null>(null);

  const [scenarioTile, setScenarioTile] = useState(0);
  const [bookmarkPanelOpen, setBookmarkPanelOpen] = useState(true);
  const bookmarks = useBookmarks(projectId, true);
  const startScenario = useMemo(
    () => defaultScenario(bookmarks.data),
    [bookmarks.data],
  );
  const scenario = pickedScenario ?? startScenario?.index ?? 0;
  const scenarioDate = pickedDate ?? startScenario?.date ?? null;

  const pickedMap = useGameWorldMap(
    projectId,
    editMode && mapProvince > 0 ? (scenarioDate ?? "1936.1.1") : null,
  );

  const hqCorps = useOfficerCorps(
    projectId,
    scenarioDate ? tag : null,
    scenarioDate,
  );

  const editingFlag =
    uiEdit &&
    !!selection &&
    selection.win.sprites[selection.node.sprite || ""]?.block_type ===
      "maskedshieldtype";
  const flagStats = useSWR<GuiCountryStats>(
    editingFlag ? ["game-view-stats", projectId, tag, scenarioDate] : null,
    () => {
      guiRenderApi.setProjectId(projectId);
      return guiRenderApi.countryStats(tag, scenarioDate || undefined);
    },
    { revalidateOnFocus: false },
  );

  const dateKnown =
    pickedDate !== null ||
    bookmarks.data !== undefined ||
    bookmarks.error !== undefined;

  const loadActivity = useGuiLoadActivity();
  const [loadBaseline] = useState(() => getGuiLoadSnapshot().finished);
  const [firstScreenReady, setFirstScreenReady] = useState(false);
  useEffect(() => {
    if (firstScreenReady || !dateKnown || loadActivity.pending > 0) return;
    const timer = setTimeout(() => setFirstScreenReady(true), 300);
    return () => clearTimeout(timer);
  }, [firstScreenReady, dateKnown, loadActivity.pending]);

  const [clock, setClock] = useState(START_CLOCK);

  const [openedBy, setOpenedBy] = useState<string | null>(null);

  const [menuOpen, setMenuOpen] = useState(false);
  const menuWin = useGameWindow(projectId, menuOpen ? "ingamemenu" : null);

  const onMainMenu = panel === MAIN_MENU;
  const [singlePlayer, setSinglePlayer] = useState(false);

  const toggleStartScreens = useCallback(() => {
    setMenuOpen(false);
    setPanel(startScreensToggle);
  }, []);
  const frontendPictures = useSWR(
    ["game-view-frontend", projectId],
    () => {
      guiRenderApi.setProjectId(projectId);
      return guiRenderApi.frontend();
    },
    { revalidateOnFocus: false, shouldRetryOnError: false },
  );

  const frontendBackdrop =
    onMainMenu || panel === SCENARIO_PICKER || panel === COUNTRY_PICKER;
  const menuBackground = useGameWindow(
    projectId,
    frontendBackdrop ? MENU_BACKGROUND : null,
  );
  const menuUpperLeft = useGameWindow(
    projectId,
    onMainMenu ? MENU_UPPER_LEFT : null,
  );
  const menuUpperRight = useGameWindow(
    projectId,
    onMainMenu ? MENU_UPPER_RIGHT : null,
  );
  const menuMain = useGameWindow(
    projectId,
    onMainMenu ? MENU_MAIN_PANEL : null,
  );
  const menuSingle = useGameWindow(
    projectId,
    onMainMenu && singlePlayer ? MENU_SINGLE_PLAYER : null,
  );
  const menuSocial = useGameWindow(
    projectId,
    onMainMenu && !singlePlayer ? MENU_SOCIAL : null,
  );
  const menuChangeBackground = useGameWindow(
    projectId,
    onMainMenu ? MENU_CHANGE_BACKGROUND : null,
  );

  const menuDlcColumn = useGameWindow(
    projectId,
    onMainMenu ? MENU_DLC_COLUMN : null,
  );
  const menuDlcItem = useGameWindow(
    projectId,
    onMainMenu ? MENU_DLC_ITEM : null,
  );
  const menuDlcSprites = useSWR(
    onMainMenu ? ["game-view-dlc-sprites", projectId] : null,
    () => {
      guiRenderApi.setProjectId(projectId);
      return guiRenderApi.sprites(dlcColumnSprites());
    },
    { revalidateOnFocus: false },
  );
  const backdropLayer = useMemo(
    () =>
      frontendBackdrop && menuBackground.data
        ? menuBackgroundWindow(
            menuBackground.data,
            frontendPictures.data?.backgrounds ?? [],
          )
        : null,
    [frontendBackdrop, menuBackground.data, frontendPictures.data],
  );
  const menuLayers = useMemo(() => {
    if (!onMainMenu) return [];
    const version = frontendPictures.data?.version ?? "";
    const centre = singlePlayer ? menuSingle.data : menuMain.data;
    return [
      menuDlcColumn.data && menuDlcItem.data && menuDlcSprites.data
        ? dlcColumnWindow(
            menuDlcColumn.data,
            menuDlcItem.data,
            menuDlcSprites.data,
          )
        : null,
      menuUpperLeft.data && mainMenuWindow(menuUpperLeft.data),
      menuUpperRight.data && mainMenuWindow(menuUpperRight.data),
      centre && mainMenuWindow(centre, version),
      !singlePlayer && menuSocial.data && menuMain.data
        ? socialWindow(menuSocial.data, mainPanelSize(menuMain.data), {
            width: SCREEN_W,
            height: SCREEN_H,
          })
        : null,
      menuChangeBackground.data && mainMenuWindow(menuChangeBackground.data),
    ].filter((win): win is GuiWindow => !!win);
  }, [
    onMainMenu,
    singlePlayer,
    frontendPictures.data,
    menuUpperLeft.data,
    menuUpperRight.data,
    menuMain.data,
    menuSingle.data,
    menuSocial.data,
    menuChangeBackground.data,
    menuDlcColumn.data,
    menuDlcItem.data,
    menuDlcSprites.data,
  ]);

  const [loading, setLoading] = useState<{
    next: string;
    seed: number;
    hold: boolean;
  } | null>(() =>
    params.get("panel") === LOAD_SCREEN
      ? { next: "countrypoliticsview", seed: 0, hold: true }
      : null,
  );
  const [loadingProgress, setLoadingProgress] = useState(0);
  const loadWin = useGameWindow(projectId, loading ? LOAD_SCREEN : null);
  const loadingLoc = useSWR(
    loading ? ["game-view-loading-loc", projectId] : null,
    async () => {
      guiRenderApi.setProjectId(projectId);
      return guiRenderApi.loc([...LOADING_STEPS, ...LOADING_TIP_KEYS], {
        markup: true,
      });
    },
    { revalidateOnFocus: false },
  );
  const startLoading = useCallback((next: string) => {
    setLoadingProgress(0);
    setLoading({ next, seed: Math.random(), hold: false });
  }, []);
  const finishLoading = useCallback(() => {
    if (!loading) return;
    setPanel(loading.next);
    setLoading(null);
  }, [loading]);
  useEffect(() => {
    if (!loading || loading.hold || uiEdit) return;
    const started = performance.now();
    const timer = window.setInterval(() => {
      const at = Math.min(1, (performance.now() - started) / LOADING_MS);
      setLoadingProgress(at);
      if (at >= 1) {
        window.clearInterval(timer);
        setPanel(loading.next);
        setLoading(null);
      }
    }, 80);
    return () => window.clearInterval(timer);
  }, [loading, uiEdit]);

  const shownProgress =
    loading && (loading.hold || uiEdit) ? 0.6 : loadingProgress;
  const loadingStep = loadingStepAt(shownProgress);
  const loadScreen = useMemo(() => {
    if (!loading || !loadWin.data) return null;
    const loc = loadingLoc.data ?? {};
    const pictures = frontendPictures.data?.loading ?? [];
    const picture = pictures.length
      ? pictures[
          Math.min(
            pictures.length - 1,
            Math.floor(loading.seed * pictures.length),
          )
        ]
      : null;
    const { win, barId } = loadScreenWindow(
      loadWin.data,
      { key: loadingStep, text: loc[loadingStep] ?? loadingStep },
      pickTip(loc, loading.seed),
      { width: SCREEN_W, height: SCREEN_H },
    );
    const bar = barId
      ? measureWindow(win.node, win.sprites, SCREEN_W, SCREEN_H)[barId]
      : undefined;
    return { win, bar, picture };
  }, [
    loading,
    loadWin.data,
    loadingLoc.data,
    frontendPictures.data,
    loadingStep,
  ]);

  const [music, setMusic] = useState(0);
  const musicSlide = useRef({ raf: 0, open: false, value: 0 });
  const toggleMusic = useCallback(() => {
    const slide = musicSlide.current;
    slide.open = !slide.open;
    cancelAnimationFrame(slide.raf);
    const from = slide.value;
    const to = slide.open ? 1 : 0;
    const ease = slide.open ? DECELERATED : ACCELERATED;
    const ms = MUSIC_SLIDE_MS * Math.abs(to - from);
    const start = performance.now();
    const step = (now: number) => {
      const t = ms ? Math.min(1, (now - start) / ms) : 1;
      slide.value = from + (to - from) * ease(t);
      setMusic(slide.value);
      if (t < 1) slide.raf = requestAnimationFrame(step);
    };
    slide.raf = requestAnimationFrame(step);
  }, []);
  useEffect(() => {
    const slide = musicSlide.current;
    return () => cancelAnimationFrame(slide.raf);
  }, []);

  const topbar = useGameWindow(projectId, dateKnown ? "topbar" : null);

  const mapmodesPanel = useGameWindow(
    projectId,
    dateKnown ? MAPMODES_WINDOW : null,
  );
  const [mapMode, setMapMode] = useState<MapModeId>("land");

  const base = basePanelOf(panel, opensOver);

  const firstEvent = useSWR(
    base === EVENT_WINDOW && !eventId
      ? ["game-view-first-event", projectId, tag ?? ""]
      : null,
    () => {
      guiRenderApi.setProjectId(projectId);

      return guiRenderApi
        .eventChoices("", tag)
        .then((events) => (events[0] ? events : guiRenderApi.eventChoices("")))
        .then((events) => events[0]?.id ?? null);
    },
    { revalidateOnFocus: false },
  );
  const shownEventId = eventId || firstEvent.data || null;
  const currentViewScope: GameViewScope = { projectId, tag, base, panel };
  const [viewScope, setViewScope] = useState(currentViewScope);

  const scopeChange = gameViewScopeChange(viewScope, currentViewScope);
  if (scopeChange !== "none") {
    setViewScope(currentViewScope);
    setSelection(null);
    setSelectedFocus(null);
    setTechPick(null);
    if (scopeChange === "project" || scopeChange === "country") {
      setFocusViewTag(null);

      setEditMode(params.get("edit") === "1");
      setUiEdit(params.get("edit") === "ui");
    }
    if (scopeChange === "project") {
      setEventSaved({});
      setEventFocus(null);
      setDecisionPick(null);
      setDecisionSaved({});
      setPersonPick(null);
      setSpiritPick(null);
      setSpiritStartOpen(false);
      setEquipmentPick(null);
      setEquipmentDesign(null);
      setDesignPanel(null);
      setDesignDraft(null);
      setWarGoalPick(NO_WARGOAL);
      setWarGoalEdit(null);
      setIntelPick(NO_INTEL_PICK);
      setIntelForming(false);
      setIntelEdit(null);
      setAutonomyEdit(null);
      setFacilityPick(null);
      setProjectPage(null);
      setProjectEdit(null);
      setUpgradeEdit(null);
      setDivisionEdit(null);
      setSubUnitEdit(null);
      setUnitEdit(null);
      setScientistEdit(null);
      setPartiesPick(null);
      setMioTraitPick(null);
      setIdeaGroup(null);
      setMioFilters(NO_MIO_FILTERS);
      setMioDetail(null);
      setPickerOver(null);
      setDesignerOver(null);
      setTechSaved({});
      setMapProvince(0);
      setMapScope("state");
      setChosenCountry(null);
      setScenario(null);
      setScenarioDate(null);
      setScenarioTile(0);
      setChosenTrees({});
      setLocEdits({});
      setGuiTexts({});
      setSpriteVersions({});
    }
  }

  const deployedOn = base === DEPLOYMENT_VIEW || panel === DEPLOYMENT_VIEW;
  const shownDivisions = useDivisions(
    projectId,
    tag && deployedOn ? tag : null,
    scenarioDate ?? null,
  );
  const [deploymentPanelOpen, setDeploymentPanelOpen] = useState(true);

  const diplomacyOn = base === DIPLOMACY || panel === DIPLOMACY;
  const shownDiplomacy = useDiplomacy(
    projectId,
    tag && diplomacyOn ? tag : null,
    scenarioDate ?? null,
  );

  const editedWarGoals = useWarGoals(
    projectId,
    panel === WARGOAL_WINDOW && warGoalEdit ? (tag ?? null) : null,
    diplomacyTarget,
    scenarioDate,
  );
  const editedWarGoal =
    guiArray(editedWarGoals.data?.types).find(
      (type) => type.id === warGoalEdit,
    ) ?? null;

  const editedIntel = useIntelAgency(
    projectId,
    panel === INSIGNIA_WINDOW && intelEdit ? (tag ?? null) : null,
    scenarioDate,
  );
  const editedInsignia =
    guiArray(editedIntel.data?.insignia).find(
      (item) =>
        item.file === intelEdit?.file && item.ordinal === intelEdit?.ordinal,
    ) ?? null;

  const editedRoster = useScientists(
    projectId,
    panel === SCIENTISTS_WINDOW && scientistEdit ? (tag ?? null) : null,
    scenarioDate,
  );
  const editedScientist =
    guiArray(editedRoster.data?.scientists).find(
      (item) => item.token === scientistEdit,
    ) ?? null;

  const editedProjects = useSpecialProjects(
    projectId,
    panel === PROGRAM_WINDOW && projectEdit ? (tag ?? null) : null,
    scenarioDate,
  );
  const editedProject =
    guiArray(editedProjects.data?.projects).find(
      (project) => project.id === projectEdit,
    ) ?? null;

  const editedArmy = useArmyOverview(
    projectId,
    panel === ARMY_OVERVIEW && divisionEdit ? (tag ?? null) : null,
    scenarioDate,
  );

  const mapArmy = useArmyOverview(projectId, tag ?? null, scenarioDate);
  const mapUnits = useMemo(
    () => unitCounters(mapArmy.data?.divisions),
    [mapArmy.data],
  );
  const editedDivision =
    guiArray(editedArmy.data?.divisions).find(
      (division) =>
        division.oob === divisionEdit?.oob &&
        division.ordinal === divisionEdit?.ordinal,
    ) ?? null;

  const editedNavy = useNavyOverview(
    projectId,
    panel === ARMY_OVERVIEW && unitEdit?.kind === "task_force"
      ? (tag ?? null)
      : null,
    scenarioDate,
  );
  const editedAir = useAirOverview(
    projectId,
    panel === ARMY_OVERVIEW && unitEdit?.kind === "air_wing"
      ? (tag ?? null)
      : null,
    scenarioDate,
  );
  const editedUnit = ((): OOBUnit | null => {
    const at = (unit: { oob: string; ordinal: number }) =>
      unit.oob === unitEdit?.oob && unit.ordinal === unitEdit?.ordinal;
    const force =
      unitEdit?.kind === "task_force"
        ? guiArray(editedNavy.data?.task_forces).find(at)
        : undefined;
    if (force) return { kind: "task_force", unit: force };
    const wing =
      unitEdit?.kind === "air_wing"
        ? guiArray(editedAir.data?.wings).find(at)
        : undefined;
    return wing ? { kind: "air_wing", unit: wing } : null;
  })();

  const editedAgency = useIntelAgency(
    projectId,
    panel === INTEL_UPGRADES_WINDOW && upgradeEdit ? (tag ?? null) : null,
    scenarioDate,
  );
  const editedUpgrade =
    guiArray(editedAgency.data?.branches)
      .flatMap((branch) => guiArray(branch.upgrades))
      .find((upgrade) => upgrade.id === upgradeEdit) ?? null;

  const editedPage = useProjectPage(
    projectId,
    panel === PROGRAM_WINDOW && projectEdit ? (tag ?? null) : null,
    scenarioDate,
    projectEdit,
  );

  const editedSubjects = useSubjects(
    projectId,
    panel === SUBJECTS_WINDOW && autonomyEdit ? (tag ?? null) : null,
    scenarioDate,
  );
  const editedAutonomy =
    (autonomyEdit && editedSubjects.data?.levels?.[autonomyEdit]) || null;
  const [diplomacyPanelOpen, setDiplomacyPanelOpen] = useState(true);
  const shownStates = useStates(
    projectId,
    tag && (base === STATE_VIEW || panel === STATE_VIEW) ? tag : null,
    scenarioDate ?? null,
    stateId,
  );

  const decisionList = useDecisions(
    projectId,
    editMode && base === DECISIONS ? tag : null,
    scenarioDate,
  );
  const popup = panel && panel !== base ? panel : null;

  const popupUnder =
    popup && opensOver[popup] && opensOver[popup] !== base
      ? opensOver[popup]
      : null;
  const active = useGameWindow(
    projectId,
    dateKnown && panel !== MAIN_MENU ? base : null,
  );

  const screenPanels = useScreenPanels(projectId, base);

  const componentHistory = useComponentHistory(projectId, () => {
    setPanelPick(null);
    setSelection(null);
  });
  const componentPlacer = usePlaceComponent(
    projectId,
    base,
    tag,
    componentHistory.record,
  );
  const moveComponent = useMoveComponent(projectId, componentHistory.record);
  const moveWindow = useMoveWindow(projectId, componentHistory.record);
  const editWords = useEditWords(projectId, componentHistory.record);

  const windowPanels = useMemo(
    () =>
      (screenPanels.list.data ?? []).filter(
        (one) => isWindowPanel(one) && one.context_type === "player_context",
      ),
    [screenPanels.list.data],
  );

  const topbarBand = useMemo(() => {
    if (!topbar.data) return null;
    const place = restPlacement(topbar.data.node, SCREEN_W, SCREEN_H);
    const background = topbar.data.sprites?.GFX_topbar_background_extended;
    return { place, h: Math.round(Number(background?.height) || 100) };
  }, [topbar.data]);
  const topbarPanels = useMemo(
    () =>
      (screenPanels.list.data ?? []).filter(
        (one) => one.parent_window_token === TOPBAR_MOUNT,
      ),
    [screenPanels.list.data],
  );

  const ghostAt = (
    preset: PaletteItem,
    clientX: number,
    clientY: number,
  ): DropGhost | null => {
    const stage = stageRef.current;
    if (!stage) return null;
    const rect = stage.getBoundingClientRect();
    if (
      clientX < rect.left ||
      clientY < rect.top ||
      clientX >= rect.right ||
      clientY >= rect.bottom
    )
      return null;
    const bar = document
      .querySelector("[data-component-palette]")
      ?.getBoundingClientRect();
    if (
      bar &&
      clientX >= bar.left &&
      clientX < bar.right &&
      clientY >= bar.top &&
      clientY < bar.bottom
    )
      return null;
    const scale = rect.width / SCREEN_W || 1;
    const px = (clientX - rect.left) / scale;
    const py = (clientY - rect.top) / scale;
    const { w, h } = itemSize(preset);
    const box = { x: Math.round(px - w / 2), y: Math.round(py - h / 2), w, h };
    if (preset === "window_panel")
      return {
        preset,
        ...box,
        guides: [],
        into: null,
        target: "new",
        origin: { x: 0, y: 0 },
      };
    const over = [...windowPanels].reverse().find((one) => {
      const wp = one.window_props ?? {};
      const x = Number(wp.x) || 0;
      const y = Number(wp.y) || 0;
      return (
        px >= x &&
        py >= y &&
        px < x + (Number(wp.width) || 0) &&
        py < y + (Number(wp.height) || 0)
      );
    });
    const at = active.data
      ? restPlacement(active.data.node, SCREEN_W, SCREEN_H)
      : null;
    const onTopbar = !over && !!topbarBand && py < topbarBand.h;
    if (!over && !onTopbar && (!at || !mountOf(base))) return null;
    const origin = over
      ? { x: 0, y: 0 }
      : onTopbar
        ? { x: topbarBand!.place.x, y: topbarBand!.place.y }
        : { x: at!.x, y: at!.y };
    const own = over ?? (onTopbar ? topbarPanels[0] : screenPanels.panels[0]);
    const wx = origin.x + (Number(own?.window_props?.x) || 0);
    const wy = origin.y + (Number(own?.window_props?.y) || 0);
    const scrollOf = own ? panelScrollOf(own.token) : undefined;
    const others = own
      ? boxesOf(own.controls ?? [], scrollOf).map((b) => ({
          ...b,
          x: b.x + wx,
          y: b.y + wy,
        }))
      : [];
    const snapped = snapBox(box, others, "move");
    const into = own
      ? containerAt(
          own.controls ?? [],
          snapped.x - wx + w / 2,
          snapped.y - wy + h / 2,
          { w, h },
          scrollOf,
        )
      : null;
    return {
      preset,
      x: snapped.x,
      y: snapped.y,
      w,
      h,
      guides: snapped.guides,
      into: into ? (others.find((b) => b.path === into.path) ?? null) : null,
      target: over ? over.id : onTopbar ? "topbar" : "screen",
      origin,
    };
  };
  const paletteDrag: PaletteDrag = {
    onDragMove: (preset, clientX, clientY) =>
      setDropGhost(ghostAt(preset, clientX, clientY)),
    onDragCancel: () => setDropGhost(null),
    onDrop: (preset, clientX, clientY) => {
      const ghost = ghostAt(preset, clientX, clientY);
      setDropGhost(null);
      if (!ghost) return;
      void (async () => {
        if (ghost.preset === "window_panel") {
          const made = await componentPlacer.placeWindow(ghost.x, ghost.y);
          if (!made) return;
          await refreshPanelWindows(projectId, made.token);
          setSelection(null);
          setPanelPick({ panelId: made.id });
          return;
        }
        const into =
          typeof ghost.target === "number"
            ? windowPanels.find((one) => one.id === ghost.target)
            : undefined;
        const origin = ghost.origin ?? { x: 0, y: 0 };
        const placed = await componentPlacer.place(
          ghost.preset,
          ghost.x - origin.x,
          ghost.y - origin.y,
          {
            into,
            mount: ghost.target === "topbar" ? TOPBAR_MOUNT : undefined,
          },
        );
        if (!placed) return;
        await refreshPanelWindows(projectId, placed.panel.token);
        setSelection(null);
        setPanelPick({ panelId: placed.panel.id, path: placed.path });
      })();
    },
  };
  const popupWin = useGameWindow(projectId, dateKnown ? popup : null);
  const popupUnderWin = useGameWindow(projectId, dateKnown ? popupUnder : null);
  const companionName = base ? (COMPANIONS[base]?.[0] ?? null) : null;
  const companion = useGameWindow(projectId, dateKnown ? companionName : null);

  const focusTag = focusViewTag || tag;
  const focusTreeId =
    chosenTrees[focusTag.toUpperCase()] ??
    (focusViewTag ? null : requestedFocusTree) ??
    null;
  const focusTree = useSWR<FocusTreePayload>(
    panel === "nationalfocusview"
      ? ["game-view-focus-tree", projectId, focusTreeId, focusTag, scenarioDate]
      : null,
    async () => {
      guiRenderApi.setProjectId(projectId);
      return (await guiRenderApi.focusTree(
        focusTreeId,
        focusTag,
        scenarioDate,
      )) as FocusTreePayload;
    },
    { revalidateOnFocus: false },
  );

  const [focusEdits, setFocusEdits] = useState<
    Record<
      string,
      {
        name?: string;
        cost?: number;
        x?: number;
        y?: number;
        prereq?: string[][] | null;
        mutex?: string[] | null;
      }
    >
  >({});

  const [picking, setPicking] = useState<{
    focusId: string;
    kind: "prereq" | "mutex";
    into?: number;
  } | null>(null);

  const [focusRemoved, setFocusRemoved] = useState<ReadonlySet<string>>(
    () => new Set(),
  );

  const [continuousAt, setContinuousAt] = useState<{
    tree: string;
    x: number;
    y: number;
  } | null>(null);

  if (scopeChange === "project") {
    setFocusEdits({});
    setFocusRemoved(new Set());
    setPicking(null);
    setContinuousAt(null);
  }
  const shownFocusTree = useMemo(() => {
    const tree = focusTree.data;
    const panelMoved = tree?.continuous && continuousAt?.tree === tree.id;
    if (
      !tree ||
      (!Object.keys(focusEdits).length && !focusRemoved.size && !panelMoved)
    )
      return tree;
    return {
      ...tree,
      focuses: tree.focuses
        .filter((f) => !focusRemoved.has(f.id))
        .map((f) => (focusEdits[f.id] ? { ...f, ...focusEdits[f.id] } : f)),
      continuous:
        panelMoved && tree.continuous && continuousAt
          ? {
              ...tree.continuous,
              position: { x: continuousAt.x, y: continuousAt.y },
            }
          : tree.continuous,
    };
  }, [focusTree.data, focusEdits, focusRemoved, continuousAt]);

  const shownTreeId = shownFocusTree?.id ?? null;

  const focusDb = useSWR(
    (editMode || uiEdit) && panel === "nationalfocusview" && shownTreeId
      ? ["game-view-focus-db", projectId, shownTreeId]
      : null,
    () => focusTreesApi.findByScriptId(projectId, shownTreeId as string),
    { revalidateOnFocus: false },
  );

  const focusWriteQueue = useRef<Promise<unknown>>(Promise.resolve());
  const queueFocusWrite = <T,>(work: () => Promise<T>): Promise<T> => {
    const next = focusWriteQueue.current.then(work);
    focusWriteQueue.current = next.catch(() => undefined);
    return next;
  };
  const NOT_IN_PROJECT = tEdit("page.focusTreeNotInProject");

  const createFocus = (x: number, y: number, parent?: FocusNode) =>
    queueFocusWrite(async () => {
      const db = await focusDb.mutate();
      if (!db) throw new Error(NOT_IN_PROJECT);
      const rows = db.nodes ?? [];

      const suggested = suggestNewFocusId(detectFocusPrefix(rows) || tag, rows);
      const created = await focusTreesApi.createNode(
        projectId,
        db.id,
        {
          ...(suggested
            ? { focus_id: suggested, on_conflict: "suffix" as const }
            : {}),
          x,
          y,
          ...(parent ? { prerequisites: [[parent.id]] } : {}),
        },
        db.edit_revision,
      );
      setFocusRemoved((current) => {
        if (!current.has(created.focus_id)) return current;
        const next = new Set(current);
        next.delete(created.focus_id);
        return next;
      });
      void focusDb.mutate();
      const rebuilt = await focusTree.mutate();
      const shown = rebuilt?.focuses.find((f) => f.id === created.focus_id);
      if (shown) {
        setSelection(null);
        setSelectedFocus(shown);
      }
      toast.success(tEdit("page.focusCreated", { id: created.focus_id }));
    });

  const editFocusesOptimistically = (
    changes: Record<
      string,
      Partial<Pick<FocusNode, "x" | "y" | "prereq" | "mutex">>
    >,
    work: () => Promise<void>,
  ) => {
    const before = Object.fromEntries(
      Object.keys(changes).map((id) => [id, focusEdits[id]]),
    );
    setFocusEdits((current) => {
      const next = { ...current };
      for (const [id, change] of Object.entries(changes))
        next[id] = { ...next[id], ...change };
      return next;
    });
    void queueFocusWrite(async () => {
      try {
        await work();
        void focusDb.mutate();
        void focusTree.mutate();
      } catch (err) {
        setFocusEdits((current) => {
          const next = { ...current };
          for (const [id, was] of Object.entries(before)) {
            if (was) next[id] = was;
            else delete next[id];
          }
          return next;
        });
        toast.error(err instanceof Error ? err.message : String(err));
      }
    });
  };

  const focusRow = async (focusId: string) => {
    const db = await focusDb.mutate();
    const node = db?.nodes?.find((n) => n.focus_id === focusId);
    if (!db || !node)
      throw new Error(
        db ? tEdit("page.focusMissing", { id: focusId }) : NOT_IN_PROJECT,
      );
    return { db, node };
  };

  const dropFocusEdit = (
    focusId: string,
    fields: readonly ("x" | "y" | "prereq" | "mutex")[],
  ) =>
    setFocusEdits((current) => {
      if (!current[focusId]) return current;
      const kept = { ...current[focusId] };
      for (const field of fields) delete kept[field];
      const next = { ...current };
      if (Object.keys(kept).length) next[focusId] = kept;
      else delete next[focusId];
      return next;
    });
  const recordFocusWrite = (
    focus: FocusNode,
    what: "prereq" | "position",
    db: { id: number },
    node: import("@/types").FocusNode,
    saved: import("@/types").FocusNode,
    keys: string[],
    shown: readonly ("x" | "y" | "prereq" | "mutex")[],
  ) => {
    const undoRow = rowUndo<import("@/types").FocusNode>({
      before: pick(node, keys),
      after: pick(saved, keys),
      load: () => focusTreesApi.getNode(projectId, db.id, node.id),
      save: (back, now) =>
        focusTreesApi.updateNode(
          projectId,
          db.id,
          node.id,
          back,
          now.tree_edit_revision,
        ),
    });
    recordChange({
      projectId,
      title: focus.name || focus.id,
      kind: tEdit("page.focusKind", {
        id: focus.id,
        what: tEdit(`page.focusWhat.${what}`),
      }),
      workbenchHref: `/project/${projectId}/focus-tree?tree=${db.id}&focus=${node.id}`,
      undo: async () => {
        await undoRow();
        dropFocusEdit(focus.id, shown);
      },
    });
  };
  const setPrereqs = (focus: FocusNode, groups: string[][]) =>
    editFocusesOptimistically({ [focus.id]: { prereq: groups } }, async () => {
      const { db, node } = await focusRow(focus.id);
      const saved = await focusTreesApi.updateNode(
        projectId,
        db.id,
        node.id,
        { prerequisites: groups },
        node.tree_edit_revision,
      );
      recordFocusWrite(
        focus,
        "prereq",
        db,
        node,
        saved,
        ["prerequisites"],
        ["prereq"],
      );
    });

  const setExclusive = (
    focus: FocusNode,
    other: FocusNode,
    enabled: boolean,
  ) => {
    const toggle = (list: string[] | null, id: string) =>
      enabled
        ? [...(list ?? []).filter((m) => m !== id), id]
        : (list ?? []).filter((m) => m !== id);
    editFocusesOptimistically(
      {
        [focus.id]: { mutex: toggle(focus.mutex, other.id) },
        [other.id]: { mutex: toggle(other.mutex, focus.id) },
      },
      async () => {
        const { db, node } = await focusRow(focus.id);
        const otherNode = db.nodes?.find((n) => n.focus_id === other.id);
        if (!otherNode)
          throw new Error(tEdit("page.focusMissing", { id: other.id }));
        await focusTreesApi.setMutualExclusion(
          projectId,
          db.id,
          node.id,
          otherNode.id,
          enabled,
          node.tree_edit_revision,
        );

        recordChange({
          projectId,
          title: focus.name || focus.id,
          kind: tEdit(
            enabled ? "page.focusExclusive" : "page.focusNotExclusive",
            { id: focus.id, other: other.id },
          ),
          workbenchHref: `/project/${projectId}/focus-tree?tree=${db.id}&focus=${node.id}`,
        });
      },
    );
  };
  const pickFocus = (picked: FocusNode) => {
    const tree = shownFocusTree;
    const job = picking;
    const focus = job && tree?.focuses.find((f) => f.id === job.focusId);
    if (!job || !tree || !focus) {
      setPicking(null);
      return;
    }
    const problem = linkProblem(tree.focuses, focus, picked.id, job.kind);
    if (problem) {
      toast.error(tContent(`focus.link.${problem}`));
      return;
    }
    setPicking(null);
    if (job.kind === "prereq")
      setPrereqs(focus, withPrereq(focus.prereq, picked.id, job.into));
    else setExclusive(focus, picked, true);
  };
  useEffect(() => {
    if (!picking) return;

    const onKey = (e: KeyboardEvent) => {
      if (e.key !== "Escape") return;
      e.stopPropagation();
      setPicking(null);
    };
    window.addEventListener("keydown", onKey, true);
    return () => window.removeEventListener("keydown", onKey, true);
  }, [picking]);
  const deleteFocus = (focus: FocusNode) =>
    queueFocusWrite(async () => {
      const db = await focusDb.mutate();
      const node = db?.nodes?.find((n) => n.focus_id === focus.id);
      if (!db || !node)
        throw new Error(db ? tEdit("page.focusMissingThis") : NOT_IN_PROJECT);
      await focusTreesApi.deleteNode(
        projectId,
        db.id,
        node.id,
        node.tree_edit_revision,
      );
      setFocusRemoved((current) => new Set(current).add(focus.id));
      setSelectedFocus(null);
      void focusDb.mutate();
      void focusTree.mutate();
      toast.success(tEdit("page.focusDeleted", { id: focus.id }));
    });
  const moveFocus = (focus: FocusNode, x: number, y: number) => {
    const tree = shownFocusTree;
    if (!tree) return;

    const followers = relativeDescendants(tree.focuses, focus.id);
    const dx = x - focus.x;
    const dy = y - focus.y;
    const moved: Record<string, { x: number; y: number }> = {};
    for (const f of tree.focuses) {
      if (f.id === focus.id || followers.has(f.id))
        moved[f.id] = { x: f.x + dx, y: f.y + dy };
    }
    editFocusesOptimistically(moved, async () => {
      const { db, node } = await focusRow(focus.id);
      const stored = storedPosition(
        tree.focuses,
        { relative_to: node.relative_position_id || undefined },
        x,
        y,
      );
      const saved = await focusTreesApi.updateNode(
        projectId,
        db.id,
        node.id,
        stored,
        node.tree_edit_revision,
      );
      recordFocusWrite(
        focus,
        "position",
        db,
        node,
        saved,
        Object.keys(stored),
        ["x", "y"],
      );
    });
  };

  const moveContinuous = (position: { x: number; y: number }) => {
    const tree = shownFocusTree;
    if (!tree?.continuous) return;
    const before = continuousAt;
    setContinuousAt({ tree: tree.id, ...position });
    void queueFocusWrite(async () => {
      try {
        const db = await focusDb.mutate();
        if (!db) throw new Error(NOT_IN_PROJECT);
        const keys = [
          "continuous_focus_position_x",
          "continuous_focus_position_y",
        ];
        const saved = await focusTreesApi.update(
          projectId,
          db.id,
          {
            continuous_focus_position_x: position.x,
            continuous_focus_position_y: position.y,
          },
          db.edit_revision,
        );
        const undoRow = rowUndo<import("@/types").FocusTree>({
          before: pick(db, keys),
          after: pick(saved, keys),
          load: () => focusTreesApi.get(projectId, db.id),
          save: (back, now) =>
            focusTreesApi.update(projectId, db.id, back, now.edit_revision),
        });
        recordChange({
          projectId,
          title: tree.id,
          kind: tEdit("page.continuousKind", { id: tree.id }),
          workbenchHref: `/project/${projectId}/focus-tree?tree=${db.id}`,
          undo: async () => {
            await undoRow();
            setContinuousAt(null);
            void focusTree.mutate();
          },
        });
        void focusDb.mutate();
        await focusTree.mutate();
        setContinuousAt((current) =>
          current?.x === position.x && current?.y === position.y
            ? null
            : current,
        );
      } catch (err) {
        setContinuousAt(before);
        toast.error(err instanceof Error ? err.message : String(err));
      }
    });
  };
  const viewport = useSyncExternalStore(
    subscribeResize,
    viewportSnapshot,
    serverViewportSnapshot,
  );
  const [viewportWidth, viewportHeight] = viewport.split(":").map(Number);
  const gameStage = gameStageGeometry(
    viewportWidth,
    viewportHeight,
    SCREEN_W,
    SCREEN_H,
  );
  const control = useCallback(
    (what: TopbarControl) => {
      if (what.kind === "music") toggleMusic();
      else if (what.kind === "menu") setMenuOpen((open) => !open);
      else setClock((at) => nextClock(at, what));
    },
    [toggleMusic],
  );

  const activate = useCallback(
    (owner: string, node: GuiNode) => {
      if (owner === "ingamemenu") {
        if (node.name === "menu_quit_button") {
          setMenuOpen(false);
          setSinglePlayer(false);
          setPanel(MAIN_MENU);
          return;
        }
        const action = resolveActivation(owner, node, panel);
        if (action.kind === "close") setMenuOpen(false);
        else if (action.kind === "unsupported") {
          toast.info(t("unsupportedControl", { control: action.control }));
        }
        return;
      }
      if (
        [
          MENU_UPPER_LEFT,
          MENU_UPPER_RIGHT,
          MENU_MAIN_PANEL,
          MENU_SINGLE_PLAYER,
          MENU_SOCIAL,
          MENU_CHANGE_BACKGROUND,
          MENU_DLC_COLUMN,
        ].includes(owner)
      ) {
        const what = menuActionOf(node);
        if (what.kind === "single-player") setSinglePlayer(true);
        else if (what.kind === "back") setSinglePlayer(false);
        else if (what.kind === "new-game") {
          setSinglePlayer(false);
          setPanel(SCENARIO_PICKER);
        } else if (what.kind === "continue") {
          setSinglePlayer(false);
          startLoading("countrypoliticsview");
        } else if (what.kind === "exit")
          window.open(
            withWorkbenchCountry(
              workbenchHrefForScreen(projectId, null),
              madeTag,
            ),
            workbenchWindowName(projectId),
          );
        else
          toast.info(
            t("unsupportedControl", {
              control: what.named ? tEdit(`menu.${what.button}`) : what.button,
            }),
          );
        return;
      }
      if (owner === "topbar") {
        const what = topbarControlOf(node.name);
        if (what) {
          control(what);
          return;
        }
      }
      if (owner === MAPMODES_WINDOW) {
        const slot = mapModeSlotOf(node.name);
        if (slot) {
          const { mode } = MAP_MODE_SLOTS[slot];
          if (MAP_MODE_LAYER[mode]) setMapMode(mode);
          else
            toast.info(
              tEdit("page.mapModeNotBuilt", { mode: tEdit(`mapmode.${mode}`) }),
            );
        } else {
          toast.info(tEdit("page.buttonNotBuilt"));
        }
        return;
      }
      const filter =
        owner === MIO_LIST_WINDOW
          ? MIO_FILTER_BUTTONS[node.name ?? ""]
          : undefined;
      if (filter) {
        setMioFilters((current) => toggleFilter(current, filter));
        return;
      }
      const mioTabPicked =
        owner === MIO_DETAIL_WINDOW ? MIO_TABS[node.name ?? ""] : undefined;
      if (mioTabPicked) {
        setMioTab(mioTabPicked);
        return;
      }

      if (
        owner === "topbar" &&
        panel === ARMY_OVERVIEW &&
        TOPBAR_TARGETS[node.name ?? ""] === ARMY_OVERVIEW &&
        overviewMode(openedBy) !== overviewMode(node.name)
      ) {
        setOpenedBy(node.name);
        return;
      }
      const action = resolveActivation(owner, node, panel, opensOver);
      if (action.kind === "open") {
        if (action.window === "nationalfocusview") {
          setFocusViewTag(owner === DIPLOMACY ? diplomacyTarget : null);
        }

        if (
          action.window === DIVISION_DESIGNER &&
          owner === ARMY_HQ_TEMPLATES
        ) {
          const first = hqCorps.data?.hq_templates?.[0]?.name;
          if (first) setTemplate(first);
          setDesignerOver(ARMY_HQ_TEMPLATES);
        } else if (action.window === DIVISION_DESIGNER) {
          if (node.name === "open_designer_button") setTemplate(NEW_TEMPLATE);
          setDesignerOver(null);
        } else if (action.window === DOCTRINE_VIEW) {
          setDoctrineFolder(DOCTRINE_RINGS[node.name ?? ""] ?? null);
        }

        if (
          action.window === ARMY_HQ_TEMPLATES &&
          panel === ARMY_HQ_TEMPLATES
        ) {
          setPanel(OFFICER_CORPS);
          setOpenedBy(null);
          return;
        }
        setPanel(action.window);
        setOpenedBy(owner === "topbar" ? node.name : null);
      } else if (action.kind === "close") {
        setPanel(null);
      } else if (action.kind === "unsupported") {
        toast.info(t("unsupportedControl", { control: action.control }));
      }
    },
    [
      panel,
      control,
      opensOver,
      hqCorps.data,
      openedBy,
      startLoading,
      projectId,
      madeTag,
      t,
      tEdit,
      diplomacyTarget,
    ],
  );

  const checkedVersion = useRef<string | null>(null);
  const refreshGameView = useCallback(
    async (quiet = false) => {
      let refreshedVersion: string | null = null;
      if (quiet) {
        guiRenderApi.setProjectId(projectId);
        const version = await guiRenderApi.dataVersion().catch(() => null);
        if (canSkipGameViewRefresh(quiet, version, checkedVersion.current))
          return;
        refreshedVersion = version;
      }
      const working = quiet ? null : toast.loading(t("refreshing"));
      await globalMutate((key) => isGameViewRefreshKey(key, false));
      if (quiet) checkedVersion.current = refreshedVersion;
      if (working !== null) toast.success(t("refreshed"), { id: working });
    },
    [t, projectId],
  );
  useEffect(() => {
    const onVisible = () => {
      if (document.visibilityState === "visible") void refreshGameView(true);
    };
    document.addEventListener("visibilitychange", onVisible);
    return () => document.removeEventListener("visibilitychange", onVisible);
  }, [refreshGameView]);

  useEffect(() => {
    window.name = gameViewWindowName(projectId);
  }, [projectId]);

  const [openFlags, setOpenFlags] = useState<ReadonlySet<string>>(
    () => new Set(),
  );

  const [componentBarRoom, setComponentBarRoom] = useState(0);

  const panelActivate = useCallback(
    (owner: string, node: GuiNode) => {
      const hit = node.name
        ? controlByGuiName(screenPanels.list.data ?? [], node.name)
        : null;
      if (!hit) {
        activate(owner, node);
        return;
      }
      const ops = flagOpsOf(hit.control.click_effect);
      if (!ops.length) return;
      setOpenFlags((current) => {
        const next = new Set(current);
        for (const { flag, op } of ops) {
          if (op === "set" || (op === "toggle" && !next.has(flag)))
            next.add(flag);
          else next.delete(flag);
        }
        return next;
      });
    },
    [activate, screenPanels.list.data],
  );

  const pickPanel = (picked: UiPanel) => {
    const screen = screenOfMount(picked.parent_window_token);
    if (screen && screen !== base) openWindow(screen);
    setSelection(null);
    setPanelPick({ panelId: picked.id });
  };

  const addOpener = async (target: UiPanel) => {
    const name = panelLabel(target, locale);
    const placed = await componentPlacer.place("button", 24, 60, {
      extra: {
        text: `Open ${target.name_en || target.token}`,
        ...(locale !== "en"
          ? {
              text_locs: { [locale]: tEdit("components.openerText", { name }) },
            }
          : {}),
        click_effect: toggleFlagEffect(openFlagOf(target.token)),
      },
    });
    if (!placed) return;
    await Promise.all([
      refreshPanelWindows(projectId, placed.panel.token),
      screenPanels.list.mutate(),
    ]);
    toast.success(tEdit("components.openerPlaced"));
  };
  const openWindow = useCallback((name: string) => {
    if (name === "nationalfocusview") setFocusViewTag(null);
    setPanel(name);
    setOpenedBy(null);
  }, []);

  const litButton = topbarButtonFor(basePanelOf(panel, opensOver), openedBy);
  const topbarUi = useMemo<TopbarUi>(
    () => ({ paused: clock.paused, speed: clock.speed, litButton, music }),
    [clock, litButton, music],
  );
  const cursor = useMemo(
    () => (topbar.data ? gameCursor(topbar.data.sprites) : undefined),
    [topbar.data],
  );

  const shortcuts = useMemo(
    () =>
      topbar.data ? shortcutsOf(topbar.data.node) : new Map<string, GuiNode>(),
    [topbar.data],
  );
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      const target = e.target as HTMLElement | null;
      if (
        target &&
        (target.tagName === "INPUT" ||
          target.tagName === "TEXTAREA" ||
          target.isContentEditable)
      )
        return;
      const key = hotkeyOf(e);

      if (
        key === "ESCAPE" &&
        (selection ||
          (editMode && selectedFocus) ||
          document.querySelector("[data-game-panel]"))
      )
        return;
      if (key === "ESCAPE") {
        if (loading) {
          finishLoading();
          return;
        }

        if (onMainMenu) {
          setSinglePlayer(false);
          return;
        }

        if (menuOpen) {
          setMenuOpen(false);
          return;
        }
        if (!panel) {
          setMenuOpen(true);
          return;
        }

        setPanel((at) => (at && opensOver[at]) || null);
        return;
      }

      if (menuOpen) return;

      if (key === "F2") {
        e.preventDefault();
        toggleStartScreens();
        return;
      }

      const button = shortcuts.get(key);
      if (
        button &&
        (TOPBAR_TARGETS[button.name ?? ""] || topbarControlOf(button.name))
      ) {
        e.preventDefault();
        activate("topbar", button);
      }
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [
    shortcuts,
    activate,
    menuOpen,
    panel,
    selection,
    editMode,
    selectedFocus,
    opensOver,
    loading,
    finishLoading,
    onMainMenu,
    toggleStartScreens,
  ]);

  const baseOverAlerts = !!base && BASES_OVER_ALERTS.has(base);
  const basePlace = active.data
    ? restPlacement(active.data.node, SCREEN_W, SCREEN_H)
    : null;

  const onMoveComponent = async (
    win: GuiWindow,
    nodeName: string | null | undefined,
    to: LayoutOverride,
  ) => {
    if (!win.ui_panel) return;
    const saved = await moveComponent(win.ui_panel, nodeName, to);
    if (!saved) return;
    await refreshPanelWindows(projectId, saved.panel.token);
    await globalMutate(
      ["game-view-ui-panel", projectId, saved.panel.id],
      saved.panel,
      { revalidate: false },
    );
    void screenPanels.list.mutate();

    if (saved.path !== saved.from) {
      setSelection(null);
      setPanelPick({ panelId: saved.panel.id, path: saved.path });
    }
  };

  const onComponentWords = async (
    win: GuiWindow,
    nodeName: string | null | undefined,
    words: string,
  ) => {
    if (!win.ui_panel) return;
    const saved = await editWords(win.ui_panel, nodeName, words);
    if (!saved) return;
    await Promise.all([
      refreshPanelWindows(projectId, saved.token),
      globalMutate(["game-view-ui-panel", projectId, saved.id], saved, {
        revalidate: false,
      }),
      screenPanels.list.mutate(),
    ]);
  };

  const saveWindow = async (
    wp: UiPanel,
    to: { x?: number; y?: number; width?: number; height?: number },
  ) => {
    const saved = await moveWindow(wp.id, to);
    if (!saved) return;
    await Promise.all([
      refreshPanelWindows(projectId, saved.token),
      globalMutate(["game-view-ui-panel", projectId, saved.id], saved, {
        revalidate: false,
      }),
      screenPanels.list.mutate(),
    ]);
  };

  const shownInPlay = (one: UiPanel) =>
    uiEdit || !opensByButton(one) || openFlags.has(openFlagOf(one.token));
  const mountedLayer =
    basePlace &&
    screenPanels.panels
      .filter(shownInPlay)
      .map((mounted) => (
        <MountedPanelLayer
          key={mounted.id}
          windowName={panelWindowName(mounted)}
          parentX={basePlace.x}
          parentY={basePlace.y}
          {...editProps(panelWindowName(mounted))}
          editPropsFor={editProps}
          versions={spriteVersions}
          projectId={projectId}
          onActivate={panelActivate}
          tag={tag}
          zoom={gameStage.scale}
          onMove={uiEdit ? onMoveComponent : undefined}
          onWords={uiEdit ? onComponentWords : undefined}
        />
      ));
  const windowLayer = windowPanels.filter(shownInPlay).map((wp) => {
    const name = panelWindowName(wp);
    return (
      <MountedPanelLayer
        key={`window-${wp.id}`}
        windowName={name}
        parentX={0}
        parentY={0}
        {...editProps(name)}
        editPropsFor={editProps}
        versions={spriteVersions}
        projectId={projectId}
        onActivate={panelActivate}
        tag={tag}
        zoom={gameStage.scale}
        onMove={uiEdit ? onMoveComponent : undefined}
        onWords={uiEdit ? onComponentWords : undefined}
        frame={
          uiEdit
            ? {
                label: panelLabel(wp, locale),
                picked: panelPick?.panelId === wp.id,
                onPick: () => {
                  setSelection(null);
                  setPanelPick({ panelId: wp.id });
                },
                onMove: (x, y) => saveWindow(wp, { x, y }),
                onResize: (width, height) => saveWindow(wp, { width, height }),
              }
            : undefined
        }
      />
    );
  });
  const baseLayer =
    base && active.data ? (
      <ScreenLayer
        key={base}
        win={active.data}
        owner={base}
        {...editProps(base)}
        slideIn
        projectId={projectId}
        onActivate={activate}
        tag={tag}
        template={template}
        onEditTemplate={editTemplate}
        onEditHQTemplate={editHQTemplate}
        techFolder={techFolder}
        onPickFolder={setTechFolder}
        doctrineFolder={doctrineFolder}
        onPickDoctrineFolder={setDoctrineFolder}
        eventId={shownEventId}
        superEvent={
          isSuperEventWindow(base) ? (superEventRow.data ?? null) : null
        }
        eventSaved={shownEventId ? eventSaved[shownEventId] : undefined}
        onPickEventField={(field) => {
          setSelection(null);
          setEventPanelOpen(true);
          setEventFocus({ field, at: Date.now() });
        }}
        decisionSaved={decisionSaved}
        onPickDecision={(pick) => {
          setSelection(null);
          setDecisionPick(pick);
        }}
        techSaved={techSaved}
        onPickTech={(tech, baseTechCost) => {
          setSelection(null);
          setTechPick({ tech, baseTechCost });
        }}
        onOpenIdeas={openIdeas}
        onOpenSeat={openSeat}
        onEditPerson={(token, name) => {
          setSelection(null);
          setSpiritPick(null);
          setPartiesPick(null);
          setPersonPick({ token, name });
        }}
        onEditSpirit={(spirit) => {
          setSelection(null);
          setPersonPick(null);
          setSpiritStartOpen(false);
          setPartiesPick(null);
          setSpiritPick(spirit);
        }}
        onAddSpirit={() => {
          setSelection(null);
          setPersonPick(null);
          setSpiritPick(null);
          setPartiesPick(null);
          setSpiritStartOpen(true);
        }}
        onEditParties={(party, colors) => {
          setSelection(null);
          setPersonPick(null);
          setSpiritPick(null);
          setSpiritStartOpen(false);
          setPartiesPick({ focus: party, colors });
        }}
        onEditMioTrait={(token) => {
          setSelection(null);
          setMioTraitPick(token);
        }}
        onEditEquipment={(name) => {
          setSelection(null);
          setDesignPanel(null);
          setDesignDraft(null);
          setEquipmentPick(name);
        }}
        onUnsupportedControl={(control) =>
          toast.info(t("unsupportedControl", { control }))
        }
        onEditSubUnit={(type) => {
          setSelection(null);
          setSubUnitEdit(type);
        }}
        equipmentDesign={equipmentDesign}
        designDraft={designDraft}
        diplomacyTarget={diplomacyTarget}
        onDiplomacyTarget={(target) => {
          setDiplomacyTarget(target);
          if (target) setPanel(DIPLOMACY);
        }}
        onDiplomacyAction={(id) => {
          setSelection(null);

          if (id !== "generate_wargoal") return;
          setWarGoalPick(NO_WARGOAL);
          setWarGoalEdit(null);
          setPanel(WARGOAL_WINDOW);
        }}
        warGoalPick={warGoalPick}
        onWarGoalPick={setWarGoalPick}
        onCloseWarGoal={() => {
          setWarGoalEdit(null);
          setPanel(DIPLOMACY);
        }}
        onEditWarGoal={(id) => {
          setSelection(null);
          setWarGoalEdit(id);
        }}
        intelPick={intelPick}
        intelForming={intelForming}
        onOpenInsignia={() => {
          setIntelPick(NO_INTEL_PICK);
          setPanel(INSIGNIA_WINDOW);
        }}
        onIntelPick={setIntelPick}
        onIntelCreate={() => {
          setIntelForming(true);
          setPanel(INTEL_VIEW);
        }}
        onIntelCancel={() => setIntelForming(false)}
        onCloseInsignia={() => {
          setIntelEdit(null);
          setPanel(INTEL_VIEW);
        }}
        onOpenIntelUpgrades={() => setPanel(INTEL_UPGRADES_WINDOW)}
        overview={overviewMode(openedBy)}
        onEditDivision={(division) => {
          setSelection(null);
          setDivisionEdit({ oob: division.oob, ordinal: division.ordinal });
        }}
        onEditUnit={(kind, oob, ordinal) => {
          setSelection(null);
          setUnitEdit({ kind, oob, ordinal });
        }}
        onEditIntelUpgrade={(id) => {
          setSelection(null);
          setUpgradeEdit(id);
        }}
        onEditInsignia={(item) => {
          setSelection(null);
          setIntelEdit({ file: item.file, ordinal: item.ordinal });
        }}
        onEditAutonomy={(id) => {
          setSelection(null);
          setAutonomyEdit(id);
        }}
        researchTab={researchTab}
        onResearchTab={setResearchTab}
        onOpenTechTree={() => {
          setTechPick(null);
          setPanel(TECH_TREE);
        }}
        facilityPick={facilityPick}
        onOpenFacility={(facility) => {
          setFacilityPick(facility);
          setProjectPage(null);
          setPanel(PROGRAM_WINDOW);
        }}
        onCloseProgram={() => {
          setProjectEdit(null);
          setProjectPage(null);
          setPanel(TECHNOLOGY);
        }}
        projectPage={projectPage}
        onOpenProjectPage={(id) => {
          setProjectPage(id);
          if (id) setProjectEdit(id);
        }}
        onEditProject={(id) => {
          setSelection(null);
          setProjectEdit(id);
          setProjectPage(id);
        }}
        onOpenScientists={() => setPanel(SCIENTISTS_WINDOW)}
        onEditScientist={(token) => {
          setSelection(null);
          setScientistEdit(token);
        }}
        onEditDesign={(slot) => {
          setSelection(null);
          setEquipmentPick(null);
          setDesignPanel({ slot: slot ?? null });
        }}
        onOpenEquipmentDesigner={(name, kind) => {
          setSelection(null);
          setDesignDraft(null);
          setDesignPanel(null);
          setEquipmentDesign(name);
          setPanel(DESIGNER_VIEWS[kind] ?? TANK_DESIGNER);
        }}
        stateId={stateId}
        bookmarkIndex={scenario}
        date={scenarioDate}
        onPickCountry={(countryTag, tree) => {
          setChosenCountry({
            projectId,
            routeTag,
            tag: countryTag.toUpperCase(),
          });

          if (projectTag.data !== undefined && !projectTag.data)
            void takeInCountry(projectId, countryTag);
          rememberFocusTree(countryTag, tree);

          startLoading("countrypoliticsview");
        }}
        onScenarioTile={setScenarioTile}
        onPickScenario={(at, date) => {
          setScenario(at);
          setScenarioDate(date || null);
          setPanel(COUNTRY_PICKER);
        }}
        versions={spriteVersions}
      />
    ) : null;

  const choosePerson = (token: string, name: string) => {
    setSelection(null);
    setSpiritPick(null);
    setPersonPick({ token, name });
  };

  const eventScreen = base === EVENT_WINDOW || isSuperEventWindow(base);
  const editSwitcher =
    !editMode && !eventScreen ? null : eventScreen ? (
      <EventSwitcher
        projectId={projectId}
        tag={tag}
        browseOnly={!editMode}
        current={shownEventId}
        onChoose={(id) => {
          setSelection(null);
          setEventPanelOpen(true);
          setChosenEvent(id);
          if (base !== EVENT_WINDOW) openWindow(EVENT_WINDOW);
        }}
        currentSuperEvent={
          isSuperEventWindow(base) ? (superEventRow.data ?? null) : null
        }
        onChooseSuperEvent={(row) => {
          setSelection(null);
          setEventPanelOpen(true);
          setSuperEventId(row.id);
          openWindow(superEventWindowName(row, projectTag.data));
        }}
      />
    ) : panel === COUNTRY_PICKER ? (
      <NewCountryControl
        projectId={projectId}
        date={scenarioDate ?? "1936.1.1"}
      />
    ) : (base === OFFICER_CORPS || panel === "countrypoliticsview") && tag ? (
      <PersonSwitcher
        projectId={projectId}
        tag={tag}
        date={scenarioDate}
        current={personPick?.token ?? null}
        onChoose={choosePerson}
      />
    ) : panel === "nationalfocusview" ? (
      <FocusTreeSwitcher
        projectId={projectId}
        tag={focusTag}
        current={shownFocusTree?.id ?? focusTreeId}
        onChoose={(treeId) => {
          setPicking(null);
          setSelectedFocus(null);
          rememberFocusTree(focusTag, treeId);
        }}
      />
    ) : null;

  return (
    <EditContext.Provider value={editContext}>
      <div
        className="fixed inset-0 z-[60] overflow-hidden bg-ingame-panel-deep"
        style={{
          ["--edit-switcher" as string]: editSwitcher
            ? EDIT_SWITCHER_SPACE
            : "0px",

          ["--component-bar" as string]: uiEdit
            ? `${componentBarRoom}px`
            : "0px",
        }}
      >
        {uiEdit && <style>{UI_EDIT_STAGE_CSS}</style>}
        {!firstScreenReady && (
          <GameLoadingOverlay
            done={loadActivity.finished - loadBaseline}
            total={loadActivity.finished - loadBaseline + loadActivity.pending}
          />
        )}
        {firstScreenReady && loadActivity.slow && <GameLoadingPill />}
        <div
          ref={stageRef}
          className="bg-ingame-panel-deep"
          data-ui-edit-stage={uiEdit ? "" : undefined}
          onPointerDownCapture={uiEdit ? pickInUiEdit : undefined}

          onClickCapture={uiEdit ? (e) => e.stopPropagation() : undefined}
          style={{
            position: "absolute",
            left: gameStage.centerX,
            top: "50%",
            width: SCREEN_W,
            height: SCREEN_H,
            overflow: "hidden",
            transform: `translate(-50%, -50%) scale(${gameStage.scale})`,
            cursor,
          }}
        >
          {backdropLayer && (
            <ScreenLayer
              key={MENU_BACKGROUND}
              win={backdropLayer}
              owner={MENU_BACKGROUND}
              {...editProps(MENU_BACKGROUND)}
              versions={spriteVersions}
              slideIn={false}
              projectId={projectId}
              onActivate={activate}
            />
          )}
          {dateKnown && !frontendBackdrop && (
            <GameWorldMap
              projectId={projectId}
              date={scenarioDate ?? "1936.1.1"}
              tag={tag}
              width={SCREEN_W}
              height={SCREEN_H}
              onState={(id) => {
                setSelection(null);
                setChosenState(id);
                setStatePanelOpen(true);
                setPanel(STATE_VIEW);
              }}
              onCountry={(target) => {
                setSelection(null);
                setDiplomacyTarget(target);
                setPanel(DIPLOMACY);
              }}
              onProvince={
                editMode && !uiEdit
                  ? (index) => {
                      if (mapPick.current?.(index)) return;
                      setSelection(null);
                      setMapProvince(index);
                    }
                  : undefined
              }
              picked={mapProvince}
              pickScope={mapScope}
              layer={MAP_MODE_LAYER[mapMode] ?? "political"}
              version={worldMapVersion}
              onPictureShown={(shown) => {
                const waiter = mapShownWaiter.current;
                if (waiter && shown > waiter.after) waiter.finish();
              }}
              units={mapUnits}
            />
          )}

          {mapmodesPanel.data &&
            panel !== COUNTRY_PICKER &&
            panel !== SCENARIO_PICKER &&
            !onMainMenu && (
              <ScreenLayer
                win={mapmodesPanel.data}
                owner={MAPMODES_WINDOW}
                {...editProps(MAPMODES_WINDOW)}
                versions={spriteVersions}
                slideIn={false}
                projectId={projectId}
                onActivate={activate}
                mapMode={mapMode}
              />
            )}

          {!baseOverAlerts && baseLayer}
          {!baseOverAlerts && mountedLayer}

          {menuLayers.map((win) => (
            <ScreenLayer
              key={win.name}
              win={win}
              owner={win.name}
              {...editProps(win.name)}
              versions={spriteVersions}
              slideIn
              projectId={projectId}
              onActivate={activate}
            />
          ))}
          {companionName && companion.data && (
            <ScreenLayer
              key={companionName}
              win={companion.data}
              owner={companionName}
              {...editProps(companionName)}
              slideIn
              projectId={projectId}
              onActivate={activate}

              tag={tag}
              date={scenarioDate}
              template={template}
              onEditTemplate={editTemplate}
              onEditHQTemplate={editHQTemplate}
              onUnsupportedControl={(control) =>
                toast.info(t("unsupportedControl", { control }))
              }
              versions={spriteVersions}
            />
          )}
          {panel === "nationalfocusview" && shownFocusTree && (
            <FocusTreeLayer
              projectId={projectId}
              tree={shownFocusTree}
              left={FOCUS_FRAME.left}
              top={FOCUS_FRAME.top}
              width={FOCUS_FRAME.width}
              height={FOCUS_FRAME.height}
              selectedId={selectedFocus?.id ?? null}
              onSelect={(focus) => {
                if (picking) {
                  pickFocus(focus);
                  return;
                }
                setSelection(null);
                setSelectedFocus(focus);
              }}
              versions={spriteVersions}
              onMove={editMode && !picking ? moveFocus : undefined}

              onMoveContinuous={
                (editMode || uiEdit) && !picking ? moveContinuous : undefined
              }
              continuousHint={
                (editMode || uiEdit) && !picking
                  ? t("continuousPanelPlace")
                  : t("continuousPanelMove")
              }
              onCreateAt={
                editMode
                  ? (x, y) => {
                      createFocus(x, y).catch((err: unknown) =>
                        toast.error(
                          err instanceof Error ? err.message : String(err),
                        ),
                      );
                    }
                  : undefined
              }
            />
          )}
          {topbar.data &&
            panel !== COUNTRY_PICKER &&
            panel !== SCENARIO_PICKER &&
            !onMainMenu && (
              <ScreenLayer
                win={topbar.data}
                owner="topbar"
                {...editProps("topbar")}
                versions={spriteVersions}
                slideIn={false}
                projectId={projectId}
                onActivate={activate}

                tag={tag}
                date={scenarioDate}
                topbarUi={topbarUi}
                onOpenWindow={openWindow}
                onUnsupportedControl={(control) =>
                  toast.info(t("unsupportedControl", { control }))
                }
                onEditCountry={() => {
                  setSelection(null);
                  if (editMode) setCountryPanelOpen(true);
                }}
              />
            )}

          {topbarBand &&
            panel !== COUNTRY_PICKER &&
            panel !== SCENARIO_PICKER &&
            !onMainMenu &&
            topbarPanels
              .filter(shownInPlay)
              .map((mounted) => (
                <MountedPanelLayer
                  key={`topbar-${mounted.id}`}
                  windowName={panelWindowName(mounted)}
                  parentX={topbarBand.place.x}
                  parentY={topbarBand.place.y}
                  {...editProps(panelWindowName(mounted))}
                  editPropsFor={editProps}
                  versions={spriteVersions}
                  projectId={projectId}
                  onActivate={panelActivate}
                  tag={tag}
                  zoom={gameStage.scale}
                  onMove={uiEdit ? onMoveComponent : undefined}
                  onWords={uiEdit ? onComponentWords : undefined}
                />
              ))}

          {baseOverAlerts && baseLayer}
          {baseOverAlerts && mountedLayer}

          {windowLayer}
          {uiEdit && <DropGhostLayer left={0} top={0} />}

          {popupUnder && popupUnderWin.data && (
            <ScreenLayer
              key={popupUnder}
              win={popupUnderWin.data}
              owner={popupUnder}
              {...editProps(popupUnder)}
              slideIn={false}
              modal
              projectId={projectId}
              onActivate={activate}
              tag={tag}
              date={scenarioDate}
              mioFilters={mioFilters}
              onMioDetails={openMioDetails}
              onLeaderDetails={openLeaderDetails}
              versions={spriteVersions}
            />
          )}
          {popup && popupWin.data && (
            <ScreenLayer
              key={popup}
              win={popupWin.data}
              owner={popup}
              {...editProps(popup)}
              slideIn={false}
              modal
              projectId={projectId}
              onActivate={activate}
              tag={tag}
              date={scenarioDate}
              ideaGroup={ideaGroup}
              mioFilters={mioFilters}
              onMioDetails={openMioDetails}
              mioDetail={mioDetail}
              mioTab={mioTab}
              onLeaderDetails={openLeaderDetails}
              leaderDetail={leaderDetail}
              onEditPerson={(token, name) => {
                setSelection(null);
                setSpiritPick(null);
                setPartiesPick(null);
                setPersonPick({ token, name });
              }}
              onEditSpirit={(spirit) => {
                setSelection(null);
                setPersonPick(null);
                setSpiritStartOpen(false);
                setPartiesPick(null);
                setSpiritPick(spirit);
              }}
              onAddSpirit={() => {
                setSelection(null);
                setPersonPick(null);
                setSpiritPick(null);
                setPartiesPick(null);
                setSpiritStartOpen(true);
              }}
              onEditParties={(party, colors) => {
                setSelection(null);
                setPersonPick(null);
                setSpiritPick(null);
                setSpiritStartOpen(false);
                setPartiesPick({ focus: party, colors });
              }}
              onEditMioTrait={(token) => {
                setSelection(null);
                setMioTraitPick(token);
              }}
              onEditEquipment={(name) => {
                setSelection(null);
                setDesignPanel(null);
                setDesignDraft(null);
                setEquipmentPick(name);
              }}
              onEditSubUnit={(type) => {
                setSelection(null);
                setSubUnitEdit(type);
              }}
              equipmentDesign={equipmentDesign}
              designDraft={designDraft}
              diplomacyTarget={diplomacyTarget}
              onDiplomacyTarget={(target) => {
                setDiplomacyTarget(target);
                if (target) setPanel(DIPLOMACY);
              }}
              onDiplomacyAction={(id) => {
                setSelection(null);

                if (id !== "generate_wargoal") return;
                setWarGoalPick(NO_WARGOAL);
                setWarGoalEdit(null);
                setPanel(WARGOAL_WINDOW);
              }}
              warGoalPick={warGoalPick}
              onWarGoalPick={setWarGoalPick}
              onCloseWarGoal={() => {
                setWarGoalEdit(null);
                setPanel(DIPLOMACY);
              }}
              onEditWarGoal={(id) => {
                setSelection(null);
                setWarGoalEdit(id);
              }}
              intelPick={intelPick}
              intelForming={intelForming}
              onOpenInsignia={() => {
                setIntelPick(NO_INTEL_PICK);
                setPanel(INSIGNIA_WINDOW);
              }}
              onIntelPick={setIntelPick}
              onIntelCreate={() => {
                setIntelForming(true);
                setPanel(INTEL_VIEW);
              }}
              onIntelCancel={() => setIntelForming(false)}
              onCloseInsignia={() => {
                setIntelEdit(null);
                setPanel(INTEL_VIEW);
              }}
              onOpenIntelUpgrades={() => setPanel(INTEL_UPGRADES_WINDOW)}
              overview={overviewMode(openedBy)}
              onEditDivision={(division) => {
                setSelection(null);
                setDivisionEdit({
                  oob: division.oob,
                  ordinal: division.ordinal,
                });
              }}
              onEditUnit={(kind, oob, ordinal) => {
                setSelection(null);
                setUnitEdit({ kind, oob, ordinal });
              }}
              onEditIntelUpgrade={(id) => {
                setSelection(null);
                setUpgradeEdit(id);
              }}
              onEditInsignia={(item) => {
                setSelection(null);
                setIntelEdit({ file: item.file, ordinal: item.ordinal });
              }}
              onEditAutonomy={(id) => {
                setSelection(null);
                setAutonomyEdit(id);
              }}
              researchTab={researchTab}
              onResearchTab={setResearchTab}
              facilityPick={facilityPick}
              onOpenFacility={(facility) => {
                setFacilityPick(facility);
                setProjectPage(null);
                setPanel(PROGRAM_WINDOW);
              }}
              onCloseProgram={() => {
                setProjectEdit(null);
                setProjectPage(null);
                setPanel(TECHNOLOGY);
              }}
              projectPage={projectPage}
              onOpenProjectPage={(id) => {
                setProjectPage(id);
                if (id) setProjectEdit(id);
              }}
              onEditProject={(id) => {
                setSelection(null);
                setProjectEdit(id);
                setProjectPage(id);
              }}
              onOpenScientists={() => setPanel(SCIENTISTS_WINDOW)}
              onEditScientist={(token) => {
                setSelection(null);
                setScientistEdit(token);
              }}
              onEditDesign={(slot) => {
                setSelection(null);
                setEquipmentPick(null);
                setDesignPanel({ slot: slot ?? null });
              }}
              onOpenEquipmentDesigner={(name, kind) => {
                setSelection(null);
                setDesignDraft(null);
                setDesignPanel(null);
                setEquipmentDesign(name);
                setPanel(DESIGNER_VIEWS[kind] ?? TANK_DESIGNER);
              }}
              template={template}
              onEditTemplate={editTemplate}
              onEditHQTemplate={editHQTemplate}
              versions={spriteVersions}
            />
          )}

          {menuOpen && menuWin.data && (
            <ScreenLayer
              key="ingamemenu"
              win={menuWin.data}
              owner="ingamemenu"
              {...editProps("ingamemenu")}
              versions={spriteVersions}
              slideIn={false}
              modal
              projectId={projectId}
              onActivate={activate}
            />
          )}

          {loading && loadScreen && (
            <div
              style={{
                position: "absolute",
                left: 0,
                top: 0,
                width: SCREEN_W,
                height: SCREEN_H,
                background: "#000",
                pointerEvents: "auto",
              }}
            >
              {loadScreen.picture &&
                (() => {
                  const picture = loadScreen.picture;
                  const gfx = screenPictureSprite(picture.name);
                  const swap = spriteVersions[gfx];
                  const width = swap?.width || picture.width;
                  const height = swap?.height || picture.height;
                  const shownH = Math.round((height * SCREEN_W) / width);
                  return (
                    <div
                      {...pickable(gfx, 1, picture.name)}
                      style={{
                        position: "absolute",
                        left: 0,

                        top: Math.round((SCREEN_H - shownH) / 2),
                        width: SCREEN_W,
                        height: shownH,
                        backgroundImage: `url("${swap ? guiRenderApi.spriteUrl(gfx, swap.version) : picture.url}")`,
                        backgroundSize: "100% 100%",
                      }}
                    />
                  );
                })()}
              <ScreenLayer
                key={LOAD_SCREEN}
                win={loadScreen.win}
                owner={LOAD_SCREEN}
                {...editProps(LOAD_SCREEN)}
                versions={spriteVersions}
                slideIn={false}
                projectId={projectId}
                onActivate={activate}
              />
              {loadScreen.bar && (
                <LoadingBar
                  sprite={loadScreen.win.sprites["GFX_loadingstatus_progress"]}
                  rect={loadScreen.bar}
                  progress={shownProgress}
                />
              )}
            </div>
          )}

          {mapStudio && (
            <MapStudio
              key={projectId}
              projectId={projectId}
              date={scenarioDate ?? "1936.1.1"}
              width={SCREEN_W}
              height={SCREEN_H}
              mapVersion={worldMapVersion}
              onClose={closeMapStudio}
              onMapChanged={() => setWorldMapVersion((v) => v + 1)}
              onWorldSaved={async (states) => {
                await globalMutate(
                  (key) =>
                    Array.isArray(key) &&
                    typeof key[0] === "string" &&
                    (key[0].startsWith("world-map") ||
                      key[0].startsWith("game-view")),
                );

                if (dateKnown && !frontendBackdrop) {
                  const after = worldMapVersion;
                  await new Promise<void>((resolve) => {
                    const waiter = { after, finish: () => {} };

                    const timer = setTimeout(() => waiter.finish(), 30_000);
                    waiter.finish = () => {
                      clearTimeout(timer);
                      if (mapShownWaiter.current === waiter)
                        mapShownWaiter.current = null;
                      resolve();
                    };
                    mapShownWaiter.current?.finish();
                    mapShownWaiter.current = waiter;
                    setWorldMapVersion((v) => v + 1);
                  });

                  await new Promise((resolve) =>
                    requestAnimationFrame(() => resolve(null)),
                  );
                } else {
                  setWorldMapVersion((v) => v + 1);
                }
                setMapStudio(false);
                toast.success(t("mapStudioSaved"), {
                  description: t("mapStudioSavedBody", { states }),
                });
              }}
            />
          )}

          <GameTooltipHost />
        </div>

        <div
          style={
            onMainMenu
              ? { left: "50%", transform: "translateX(-50%)" }
              : clearOfMapModes(12)
          }
          className="fixed bottom-3 z-[60] flex items-center gap-2"
        >
          {mapStudio && (
            <span className="rounded-md border border-ingame-edge/65 bg-ingame-panel px-2 py-1 text-xs text-ingame-text shadow-lg">
              {t("mode.mapStudio")}
            </span>
          )}

          <DlcBadge projectId={projectId} />
          <button
            type="button"
            onClick={() => void refreshGameView()}
            className="rounded-md border border-ingame-edge/70 bg-ingame-band px-3 py-1.5 text-sm text-ingame-title shadow-lg hover:border-ingame-gold/60 hover:bg-ingame-button/35"
          >
            {t("refresh")}
          </button>
          <HoverTip tip={t("startScreensHint")}>
            {(describedBy) => (
              <button
                type="button"
                aria-pressed={onStartScreens(panel)}
                aria-keyshortcuts="F2"
                aria-describedby={describedBy}
                onClick={toggleStartScreens}
                className={`rounded-md border bg-ingame-band px-3 py-1.5 text-sm shadow-lg hover:bg-ingame-button/35 ${onStartScreens(panel) ? "border-ingame-gold text-ingame-gold" : "border-ingame-edge/70 text-ingame-title"}`}
              >
                {onStartScreens(panel)
                  ? t("startScreensBack")
                  : t("startScreens")}
              </button>
            )}
          </HoverTip>

          <HoverTip tip={uiEdit ? null : t("uiEditHint")}>
            {(describedBy) => (
              <button
                type="button"
                aria-pressed={uiEdit}
                aria-label={t("uiEdit")}
                aria-describedby={describedBy}
                onClick={() => openManualEdit("ui")}
                className={`rounded-md border bg-ingame-band px-3 py-1.5 text-sm shadow-lg hover:bg-ingame-button/35 ${uiEdit ? "border-ingame-gold text-ingame-gold" : "border-ingame-edge/70 text-ingame-title"}`}
              >
                {uiEdit ? t("exitUIEdit") : t("uiEdit")}
              </button>
            )}
          </HoverTip>
          <HoverTip tip={editMode ? null : t("textEditHint")}>
            {(describedBy) => (
              <button
                type="button"
                aria-pressed={editMode}
                aria-label={t("textEdit")}
                aria-describedby={describedBy}
                onClick={() => openManualEdit("text")}
                className={`rounded-md border bg-ingame-band px-3 py-1.5 text-sm shadow-lg hover:bg-ingame-button/35 ${editMode ? "border-ingame-gold text-ingame-gold" : "border-ingame-edge/70 text-ingame-title"}`}
              >
                {editMode ? t("exitTextEdit") : t("textEdit")}
              </button>
            )}
          </HoverTip>
          <HoverTip tip={t("mapStudioHint")}>
            {(describedBy) => (
              <button
                type="button"
                aria-pressed={mapStudio}
                aria-label={t("mapStudio")}
                aria-describedby={describedBy}
                onClick={toggleMapStudio}
                className={`rounded-md border bg-ingame-band px-3 py-1.5 text-sm shadow-lg hover:bg-ingame-button/35 ${mapStudio ? "border-ingame-gold text-ingame-gold" : "border-ingame-edge/70 text-ingame-title"}`}
              >
                {mapStudio ? t("exitMapStudio") : t("mapStudio")}
              </button>
            )}
          </HoverTip>
        </div>
        {editMode &&
          !countryPanelOpen &&
          panel === DIVISION_DESIGNER &&
          subUnitEdit &&
          !selection && (
            <SubUnitEditPanel
              key={subUnitEdit}
              projectId={projectId}
              id={subUnitEdit}
              onSaved={(message) => {
                void globalMutate(isGameDataKey);
                toast.success(message);
              }}
              onClose={() => setSubUnitEdit(null)}
            />
          )}
        {editMode &&
          !countryPanelOpen &&
          base === PRODUCTION &&
          equipmentPick &&
          !selection && (
            <EquipmentEditPanel
              key={equipmentPick}
              projectId={projectId}
              tag={tag ?? null}
              name={equipmentPick}
              onSaved={(message) => {
                void globalMutate(isGameDataKey);
                toast.success(message);
              }}
              onClose={() => setEquipmentPick(null)}
            />
          )}
        {editMode &&
          !countryPanelOpen &&
          tag &&
          isDesignerView(panel ?? "") &&
          equipmentDesign &&
          designPanel &&
          !equipmentPick &&
          !selection && (
            <EquipmentDesignPanel
              key={equipmentDesign}
              projectId={projectId}
              tag={tag}
              date={scenarioDate}
              name={equipmentDesign}
              draft={designDraft}
              focusSlot={designPanel.slot}
              onDraft={setDesignDraft}
              onOpenDesign={(next) => {
                setDesignDraft(null);
                setEquipmentDesign(next);
              }}
              onSaved={(message, next) => {
                void globalMutate(isGameDataKey);
                setDesignDraft(null);
                setEquipmentDesign(next);
                toast.success(message);
              }}
              onClose={() => {
                setDesignPanel(null);
                setDesignDraft(null);
              }}
            />
          )}
        {editMode &&
          !countryPanelOpen &&
          panel === WARGOAL_WINDOW &&
          editedWarGoal &&
          !selection && (
            <WarGoalEditPanel
              key={editedWarGoal.id}
              projectId={projectId}
              type={editedWarGoal}
              onSaved={(message) => {
                void globalMutate(isGameDataKey);
                toast.success(message);
              }}
              onClose={() => setWarGoalEdit(null)}
            />
          )}
        {editMode &&
          !countryPanelOpen &&
          panel === ARMY_OVERVIEW &&
          overviewMode(openedBy) === "army" &&
          editedDivision &&
          !selection && (
            <DivisionEditPanel
              key={`${editedDivision.oob}#${editedDivision.ordinal}`}
              projectId={projectId}
              tag={tag}
              division={editedDivision}
              templates={editedArmy.data?.templates ?? []}
              onSaved={(message) => {
                void globalMutate(isGameDataKey);
                toast.success(message);
              }}
              onRemoved={(message) => {
                setDivisionEdit(null);
                void globalMutate(isGameDataKey);
                toast.success(message);
              }}
              onAddDivision={() => {
                setDivisionEdit(null);
                setDeploymentPanelOpen(true);
                setPanel(DEPLOYMENT_VIEW);
                setOpenedBy("deployment_button");
              }}
              onClose={() => setDivisionEdit(null)}
            />
          )}
        {editMode &&
          !countryPanelOpen &&
          panel === ARMY_OVERVIEW &&
          editedUnit &&
          overviewMode(openedBy) ===
            (editedUnit.kind === "task_force" ? "navy" : "air") &&
          !selection && (
            <OOBUnitEditPanel
              key={`${editedUnit.kind}:${editedUnit.unit.oob}#${editedUnit.unit.ordinal}`}
              projectId={projectId}
              item={editedUnit}
              onSaved={(message) => {
                void globalMutate(isGameDataKey);
                toast.success(message);
              }}
              onClose={() => setUnitEdit(null)}
            />
          )}
        {editMode &&
          !countryPanelOpen &&
          panel === INTEL_UPGRADES_WINDOW &&
          editedUpgrade &&
          !selection && (
            <IntelUpgradeEditPanel
              key={editedUpgrade.id}
              projectId={projectId}
              upgrade={editedUpgrade}
              onSaved={(message) => {
                void globalMutate(isGameDataKey);
                toast.success(message);
              }}
              onClose={() => setUpgradeEdit(null)}
            />
          )}
        {editMode &&
          !countryPanelOpen &&
          panel === SCIENTISTS_WINDOW &&
          editedScientist &&
          !selection && (
            <ScientistEditPanel
              key={editedScientist.token}
              projectId={projectId}
              scientist={editedScientist}
              specializations={editedRoster.data?.specializations ?? []}
              catalog={editedRoster.data?.trait_catalog ?? []}
              onClose={() => setScientistEdit(null)}
            />
          )}
        {editMode &&
          !countryPanelOpen &&
          panel === PROGRAM_WINDOW &&
          editedProject &&
          !selection && (
            <SpecialProjectEditPanel
              key={editedProject.id}
              projectId={projectId}
              project={editedProject}
              specializations={editedProjects.data?.specializations ?? []}
              page={editedPage.data?.found ? editedPage.data : null}
              onSaved={(message) => {
                void globalMutate(isGameDataKey);
                toast.success(message);
              }}
              onClose={() => setProjectEdit(null)}
            />
          )}
        {editMode &&
          !countryPanelOpen &&
          panel === SUBJECTS_WINDOW &&
          editedAutonomy &&
          !selection && (
            <AutonomyEditPanel
              key={editedAutonomy.id}
              projectId={projectId}
              level={editedAutonomy}
              onSaved={(message) => {
                void globalMutate(isGameDataKey);
                toast.success(message);
              }}
              onClose={() => setAutonomyEdit(null)}
            />
          )}
        {editMode &&
          !countryPanelOpen &&
          tag &&
          panel === INSIGNIA_WINDOW &&
          editedInsignia &&
          !selection && (
            <IntelAgencyEditPanel
              key={`${editedInsignia.file}#${editedInsignia.ordinal}`}
              projectId={projectId}
              tag={tag}
              item={editedInsignia}
              typedName={intelPick.name ?? editedIntel.data?.name ?? ""}
              onSaved={(message) => {
                void globalMutate(isGameDataKey);
                toast.success(message);
              }}
              onClose={() => setIntelEdit(null)}
            />
          )}
        {editMode && tag && countryPanelOpen && !selection && (
          <CountryProfilePanel
            projectId={projectId}
            tag={tag}
            date={scenarioDate}
            onSaved={(message) => {
              void globalMutate(isGameDataKey);
              toast.success(message);
            }}
            onClose={() => setCountryPanelOpen(false)}
          />
        )}
        {editMode &&
          !countryPanelOpen &&
          tag &&
          diplomacyOn &&
          diplomacyPanelOpen &&
          panel !== WARGOAL_WINDOW &&
          !selection && (
            <DiplomacyEditPanel
              projectId={projectId}
              tag={tag}
              autonomyLevels={shownDiplomacy.data?.autonomy_states ?? []}
              onSaved={() => {
                void globalMutate(isGameDataKey);
                toast.success(tEdit("page.diplomacySaved", { tag }));
              }}
              onClose={() => setDiplomacyPanelOpen(false)}
            />
          )}
        {editMode &&
          !countryPanelOpen &&
          tag &&
          deployedOn &&
          deploymentPanelOpen &&
          !(panel === DIVISION_DESIGNER && subUnitEdit) &&
          !selection && (
            <DeploymentEditPanel
              projectId={projectId}
              tag={tag}
              date={scenarioDate}
              divisions={shownDivisions.data}
              onSaved={(message) => {
                void globalMutate(isGameDataKey);
                toast.success(message);
              }}
              onClose={() => setDeploymentPanelOpen(false)}
            />
          )}
        {editMode &&
          !countryPanelOpen &&
          panel === SCENARIO_PICKER &&
          bookmarkPanelOpen &&
          !selection && (
            <BookmarkEditPanel
              projectId={projectId}
              shown={bookmarks.data?.bookmarks?.[scenarioTile]}
              onSaved={(label) => {
                void globalMutate(isGameDataKey);
                toast.success(tEdit("page.scenarioSaved", { name: label }));
              }}
              onClose={() => setBookmarkPanelOpen(false)}
            />
          )}

        {editMode &&
          !countryPanelOpen &&
          mapProvince > 0 &&
          pickedMap &&
          !selection && (
            <ProvinceEditPanel
              projectId={projectId}
              map={pickedMap}
              index={mapProvince}
              scope={mapScope}
              onScope={setMapScope}
              pickRef={mapPick}
              onSaved={(message) => {
                void globalMutate(isGameDataKey);
                void globalMutate(
                  (key) =>
                    Array.isArray(key) &&
                    key[0] === "world-map" &&
                    key[1] === projectId,
                );
                toast.success(message, {
                  description: tEdit("page.mapNamesUpdate"),
                });
              }}
              onClose={() => setMapProvince(0)}
            />
          )}
        {editMode &&
          !countryPanelOpen &&
          (base === STATE_VIEW || panel === STATE_VIEW) &&
          statePanelOpen &&
          !selection &&
          !(mapProvince > 0 && pickedMap) && (
            <StateEditPanel
              projectId={projectId}
              states={shownStates.data}
              onPickState={(id) => setChosenState(id)}
              onSaved={(id) => {
                void globalMutate(isGameDataKey);
                toast.success(tEdit("page.stateSaved", { id }));
              }}
              onClose={() => setStatePanelOpen(false)}
            />
          )}

        {editSwitcher && <EditSwitcherDock>{editSwitcher}</EditSwitcherDock>}
        {editMode &&
          !countryPanelOpen &&
          isSuperEventWindow(base) &&
          superEventRow.data &&
          eventPanelOpen &&
          !selection && (
            <SuperEventEditPanel
              key={`${superEventRow.data.id}:${superEventRow.data.updated_at}`}
              projectId={projectId}
              row={superEventRow.data}
              onSaved={(message, saved) => {
                void superEventRow.mutate(saved, { revalidate: false });
                void globalMutate(
                  (key) =>
                    Array.isArray(key) && key[0] === "game-view-super-events",
                );
                toast.success(message);
              }}

              onDeleted={(message) => {
                setSuperEventId(null);
                setEventPanelOpen(false);
                openWindow(EVENT_WINDOW);
                void globalMutate(
                  (key) =>
                    Array.isArray(key) && key[0] === "game-view-super-events",
                );
                toast.success(message);
              }}
              onClose={() => setEventPanelOpen(false)}
            />
          )}
        {editMode &&
          !countryPanelOpen &&
          base === EVENT_WINDOW &&
          shownEventId &&
          eventPanelOpen &&
          !selection && (
            <EventEditPanel
              projectId={projectId}
              eventId={shownEventId}
              tag={tag}
              date={scenarioDate}
              focusField={eventFocus}
              onSaved={(id, change) => {
                setEventSaved((current) => ({
                  ...current,
                  [id]: { ...current[id], ...change },
                }));
                void globalMutate(
                  (key) => Array.isArray(key) && key[0] === "game-view-event",
                );
              }}
              onClose={() => setEventPanelOpen(false)}
            />
          )}
        {editMode &&
          !countryPanelOpen &&
          panel === "countrypoliticsview" &&
          spiritPick &&
          !selection && (
            <SpiritEditPanel
              key={spiritPick.token}
              projectId={projectId}
              spirit={spiritPick}
              onSaved={(label) => {
                void globalMutate(isGameDataKey);
                toast.success(tEdit("politics.spiritSaved", { name: label }));
              }}
              onClose={() => setSpiritPick(null)}
              onManageStart={
                tag
                  ? () => {
                      setSpiritPick(null);
                      setSpiritStartOpen(true);
                    }
                  : undefined
              }
            />
          )}
        {editMode &&
          !countryPanelOpen &&
          panel === MIO_DETAIL_WINDOW &&
          tag &&
          mioDetail &&
          mioTraitPick &&
          !selection && (
            <MioTraitEditPanel
              key={`${mioDetail}/${mioTraitPick}`}
              projectId={projectId}
              tag={tag}
              date={scenarioDate}
              mioId={mioDetail}
              token={mioTraitPick}
              onSaved={(message) => {
                void globalMutate(isGameDataKey);
                toast.success(message);
              }}
              onClose={() => setMioTraitPick(null)}
            />
          )}
        {editMode &&
          !countryPanelOpen &&
          panel === "countrypoliticsview" &&
          tag &&
          partiesPick &&
          !spiritPick &&
          !spiritStartOpen &&
          !personPick &&
          !selection && (
            <IdeologyEditPanel
              key={partiesPick.focus ?? ""}
              projectId={projectId}
              tag={tag}
              focus={partiesPick.focus}
              colors={partiesPick.colors}
              onSaved={(message) => {
                void globalMutate(isGameDataKey);
                toast.success(message);
              }}
              onClose={() => setPartiesPick(null)}
            />
          )}
        {editMode &&
          !countryPanelOpen &&
          panel === "countrypoliticsview" &&
          tag &&
          spiritStartOpen &&
          !spiritPick &&
          !selection && (
            <SpiritStartPanel
              projectId={projectId}
              tag={tag}
              date={scenarioDate}
              onEdit={(spirit) => {
                setSpiritStartOpen(false);
                setSpiritPick(spirit);
              }}
              onChanged={(message) => {
                void globalMutate(isGameDataKey);
                toast.success(message);
              }}
              onClose={() => setSpiritStartOpen(false)}
            />
          )}
        {editMode && !countryPanelOpen && personPick && !selection && (
          <CharacterEditPanel
            key={personPick.token}
            projectId={projectId}
            token={personPick.token}
            name={personPick.name}
            onClose={() => setPersonPick(null)}
          />
        )}
        {!countryPanelOpen &&
          base === TECH_TREE &&
          techPick &&
          !selection &&
          (editMode ? (
            <TechEditPanel
              key={techPick.tech.name}
              projectId={projectId}
              tag={tag}
              tech={{ ...techPick.tech, ...techSaved[techPick.tech.name] }}
              baseTechCost={techPick.baseTechCost}
              onSaved={(name, change) => {
                setTechSaved((current) => ({
                  ...current,
                  [name]: { ...current[name], ...change },
                }));

                if ("researched" in change) void globalMutate(isGameDataKey);
                else
                  void globalMutate(
                    (key) =>
                      Array.isArray(key) && key[0] === "game-view-tech-tree",
                  );
              }}
              onClose={() => setTechPick(null)}
              onCreated={(tech) => {
                void globalMutate(
                  (key) =>
                    Array.isArray(key) && key[0] === "game-view-tech-tree",
                );
                toast.success(
                  tEdit("page.techCreated", { name: tech.label ?? tech.name }),
                );
                setTechPick({ tech, baseTechCost: techPick.baseTechCost });
              }}
              onDeleted={(name) => {
                void globalMutate(isGameDataKey);
                toast.success(tEdit("page.techDeleted", { name }));
                setTechPick(null);
              }}
            />
          ) : (
            <TechInspectPanel
              tech={{ ...techPick.tech, ...techSaved[techPick.tech.name] }}
              baseTechCost={techPick.baseTechCost}
              onRequestEdit={() => openManualEdit("text")}
              onClose={clearInspectedContent}
            />
          ))}
        {editMode &&
          !countryPanelOpen &&
          base === DECISIONS &&
          decisionPick &&
          !selection && (
            <DecisionEditPanel
              projectId={projectId}
              tag={tag}
              pick={decisionPick}
              category={guiArray(decisionList.data?.categories).find(
                (c) => c.name === decisionPick.category,
              )}
              onSaved={(pick, change) => {
                const key = decisionEditKey(pick);
                setDecisionSaved((current) => ({
                  ...current,
                  [key]: { ...current[key], ...change },
                }));
                void decisionList.mutate();
              }}
              onCreated={(pick) => {
                setDecisionPick(pick);
                void decisionList.mutate();
                toast.success(
                  tEdit("page.decisionCreated", { name: pick.name }),
                );
              }}
              onDeleted={(pick) => {
                setDecisionPick(null);
                void decisionList.mutate();
                void globalMutate(isGameDataKey);
                toast.success(
                  tEdit("page.decisionDeleted", {
                    name: pick.kind === "decision" ? pick.name : pick.category,
                  }),
                );
              }}
              onClose={() => setDecisionPick(null)}
            />
          )}
        {editMode && picking && (
          <div className="pointer-events-auto fixed left-1/2 top-4 z-50 flex -translate-x-1/2 items-center gap-3 rounded-md border border-border bg-card px-4 py-2 text-sm text-card-foreground shadow-lg">
            <span>
              {tEdit("page.pickFocusPrompt", {
                name:
                  shownFocusTree?.focuses.find((f) => f.id === picking.focusId)
                    ?.name ?? picking.focusId,
                what: tEdit(
                  picking.kind === "mutex"
                    ? "page.pickWhat.mutex"
                    : picking.into === undefined
                      ? "page.pickWhat.prereqAnd"
                      : "page.pickWhat.prereqOr",
                ),
              })}
            </span>
            <button
              type="button"
              onClick={() => setPicking(null)}
              className="rounded border border-border px-2 py-0.5 text-micro hover:bg-muted"
            >
              {tEdit("page.cancelEsc")}
            </button>
          </div>
        )}
        {editMode &&
          !countryPanelOpen &&
          panel === "nationalfocusview" &&
          selectedFocus &&
          shownFocusTree && (
            <FocusEditPanel
              projectId={projectId}
              treeId={shownFocusTree.id}
              focus={
                shownFocusTree.focuses.find((f) => f.id === selectedFocus.id) ??
                selectedFocus
              }
              onSaved={(focusId, change) => {
                setFocusEdits((current) => ({
                  ...current,
                  [focusId]: { ...current[focusId], ...change },
                }));
                void focusTree.mutate();
              }}
              onAddChild={async () => {
                const parent =
                  shownFocusTree.focuses.find(
                    (f) => f.id === selectedFocus.id,
                  ) ?? selectedFocus;
                const at = freeCell(
                  shownFocusTree.focuses,
                  parent.x,
                  parent.y + 1,
                );
                await createFocus(at.x, at.y, parent);
              }}
              onDelete={() => deleteFocus(selectedFocus)}
              focuses={shownFocusTree.focuses}
              picking={picking?.focusId === selectedFocus.id ? picking : null}
              onPick={(kind, into) =>
                setPicking({ focusId: selectedFocus.id, kind, into })
              }
              onUnlinkPrereq={(group, id) => {
                const focus =
                  shownFocusTree.focuses.find(
                    (f) => f.id === selectedFocus.id,
                  ) ?? selectedFocus;
                setPrereqs(focus, withoutPrereq(focus.prereq, group, id));
              }}
              onUnlinkExclusive={(id) => {
                const focus =
                  shownFocusTree.focuses.find(
                    (f) => f.id === selectedFocus.id,
                  ) ?? selectedFocus;
                const other = shownFocusTree.focuses.find((f) => f.id === id);
                if (other) setExclusive(focus, other, false);
              }}
              onClose={() => {
                setPicking(null);
                setSelectedFocus(null);
              }}
            />
          )}
        {uiEdit && (
          <ComponentPalette
            screen={base}
            busy={componentPlacer.busy}
            drag={paletteDrag}
            history={componentHistory}
            panels={screenPanels.list.data ?? []}
            onPickPanel={pickPanel}
            onRoom={setComponentBarRoom}
            topBar={!!topbarBand && !onMainMenu}
          />
        )}
        {uiEdit && panelPick && (!selection || selection.win.ui_panel) && (
          <PanelComponentEditor
            key={`${panelPick.panelId}|${panelPick.path ?? ""}|${panelPick.names ?? ""}`}
            projectId={projectId}
            panelId={panelPick.panelId}
            path={panelPick.path}
            names={panelPick.names}
            onClose={() => {
              setPanelPick(null);
              setSelection(null);
            }}
            onSaved={async (saved) => {
              await refreshPanelWindows(projectId, saved.token);
              await screenPanels.list.mutate();
            }}
            onRecord={componentHistory.record}
            panels={screenPanels.list.data ?? []}
            canAddOpener={mountOf(base) !== null}
            onAddOpener={addOpener}
            onDeleted={async (gone) => {
              setPanelPick(null);
              setSelection(null);
              await Promise.all([
                screenPanels.list.mutate(),
                refreshPanelWindows(projectId, gone.token),
              ]);
            }}
          />
        )}
        {uiEdit && selection && !selection.win.ui_panel && (
          <EditPanel
            projectId={projectId}
            target={editTargetOf(
              selection,
              selection.shown,
              locEdits,
              savedTexts.data ?? NO_TEXTS,
              guiTexts,
            )}
            spriteVersion={(() => {
              const gfx =
                selection.node.sprite || selection.node.background?.sprite;
              return gfx ? spriteVersions[gfx]?.version : undefined;
            })()}
            onSpriteChanged={spriteChanged}
            onOwnPicture={(gfx, shape) => {
              void globalMutate(
                (cacheKey) =>
                  Array.isArray(cacheKey) && cacheKey[0] === "game-view-window",
              );

              if (gfx === null) setSelection(null);
              else
                setSelection((current) =>
                  current ? withOwnPicture(current, gfx, shape) : current,
                );
            }}
            onElementVisibility={(target, state) => {
              void globalMutate(
                (cacheKey) =>
                  Array.isArray(cacheKey) && cacheKey[0] === "game-view-window",
              );
              setSelection((current) => {
                if (!current || removalOf(current)?.target !== target)
                  return current;
                return state === "removed"
                  ? null
                  : withHiddenMark(current, state === "hidden");
              });
            }}
            onTextSaved={(key, value) => {
              setLocEdits((current) => ({ ...current, [key]: value }));
              void savedTexts.mutate();
            }}
            onTextRestored={(key) => {
              setLocEdits((current) => {
                const next = { ...current };
                delete next[key];
                return next;
              });
              void savedTexts.mutate();

              void globalMutate(
                (cacheKey) =>
                  Array.isArray(cacheKey) && cacheKey[0] === "game-view-window",
              );
            }}
            onGuiText={(file, target, value) => {
              const key = guiTextKey(file, target);
              setGuiTexts((current) => {
                const next = { ...current };
                if (value === null) delete next[key];
                else next[key] = value;
                return next;
              });

              if (value === null) setSelection(null);
            }}
            onTextReload={() => {
              void globalMutate(
                (cacheKey) =>
                  Array.isArray(cacheKey) && cacheKey[0] === "game-view-window",
              );
              void globalMutate(isGameDataKey);
            }}
            onPickStacked={(id) => {
              const entry = selection.stack?.find((at) => at.id === id);
              if (entry)
                setSelection({
                  ...selection,
                  id: entry.id,
                  node: entry.node,
                  shown: entry.shown,
                  source: entry.source,
                });
            }}
            countryFlag={
              editingFlag && flagStats.data?.flag?.url
                ? {
                    tag,
                    url: flagStats.data.flag.url,
                    width: flagStats.data.flag.width || 82,
                    height: flagStats.data.flag.height || 52,
                  }
                : null
            }
            onEditCountryFlag={
              editingFlag
                ? () => {
                    setUiEdit(false);
                    setEditMode(true);
                    setSelection(null);
                    setCountryPanelOpen(true);
                  }
                : undefined
            }
            onClose={() => setSelection(null)}
          />
        )}
      </div>
    </EditContext.Provider>
  );
}
