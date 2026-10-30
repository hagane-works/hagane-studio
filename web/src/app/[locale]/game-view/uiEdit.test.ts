import { readFileSync } from "node:fs";
import { join } from "node:path";
import { describe, expect, it } from "vitest";
import type { GuiNode, GuiSprite, GuiWindow } from "@/lib/guiRenderApi";
import { pickInStack } from "../gui-renderer/GuiCanvas";
import { editTargetOf, spritePreviewOf, type StackEntry } from "./editModel";
import { exportNote, framePreviewStyle } from "./SpriteReplace";
import { EDIT_SWITCHER_SPACE, PANEL_PLACE } from "./GamePanel";

const strip = (over: Partial<GuiSprite> = {}): GuiSprite =>
  ({
    vanilla_url:
      "https://obs.example.com/vanilla/gfx/interface/mapmode/mapmode_buttons_deselected_small.png",
    width: 360,
    height: 18,
    frames: 18,
    resolved: true,
    ...over,
  }) as GuiSprite;
const node = (frame: number | null): GuiNode =>
  ({ frame }) as unknown as GuiNode;

describe("the art an element draws", () => {
  it("is its frame of the strip, from the object store", () => {
    const preview = spritePreviewOf(
      "GFX_mapmode_buttons_deselected_small",
      strip(),
      node(6),
    );
    expect(preview).toEqual({
      url: "https://obs.example.com/vanilla/gfx/interface/mapmode/mapmode_buttons_deselected_small.png",
      frame: 6,
      frames: 18,
      frameWidth: 20,
      frameHeight: 18,
    });
  });

  it("starts at the first frame and stays inside the strip", () => {
    expect(spritePreviewOf("x", strip(), node(null))?.frame).toBe(1);
    expect(spritePreviewOf("x", strip(), node(40))?.frame).toBe(18);
  });

  it("comes from the service when the project replaced it, and is unknown without a size", () => {
    const url =
      "/api/go/projects/69/gui-render/sprite?name=GFX_x&v=content-hash";
    expect(
      spritePreviewOf(
        "GFX_x",
        strip({ overridden: true, vanilla_url: url }),
        node(1),
      )?.url,
    ).toBe(url);
    expect(spritePreviewOf("x", strip({ width: 0 }), node(1))).toBeNull();
  });

  it("slides the strip so only that frame shows", () => {
    const style = framePreviewStyle({
      url: "u",
      frame: 3,
      frames: 18,
      frameWidth: 20,
      frameHeight: 18,
    });

    expect(style.width).toBe(71);
    expect(style.height).toBe(64);
    expect(style.backgroundSize).toBe(`${71 * 18}px 64px`);
    expect(style.backgroundPosition).toBe(`${-2 * 71}px 0`);
  });
});

describe("a click where a label lies over a button", () => {
  const entry = (
    id: string,
    type: string,
    extra: Partial<GuiNode> = {},
  ): StackEntry => ({
    id,
    node: { type, name: id, ...extra } as GuiNode,
  });
  const label = entry("manage_occupied_button_text", "instanttextboxtype", {
    text: "POLITICS_OCCUPIED_TERRITORIES",
  });
  const button = entry("manage_occupied_button", "buttontype", {
    sprite: "GFX_button_148x34",
  });
  const panel = entry("politics_bg", "containerwindowtype", {
    background: { sprite: "GFX_tiled_window" } as GuiNode,
  });

  it("picks the button, which draws the art", () => {
    expect(pickInStack([label, button, panel])?.id).toBe(
      "manage_occupied_button",
    );
    expect(
      pickInStack([
        entry("view_faction_icon", "icontype", { sprite: "GFX_x" }),
        button,
      ])?.id,
    ).toBe("manage_occupied_button");
  });

  it("picks a badge drawn on a button, but not an overlay as big as it", () => {
    const cap = {
      ...entry("advisor_type_icon", "icontype", {
        sprite: "GFX_advisor_type_army",
      }),
      w: 28,
      h: 28,
    };
    const seat = {
      ...entry("button", "buttontype", {
        sprite: "GFX_idea_slot_high_command",
      }),
      w: 63,
      h: 66,
    };
    expect(pickInStack([cap, seat])?.id).toBe("advisor_type_icon");

    const glow = {
      ...entry("yellow_alert_glow", "icontype", { sprite: "GFX_alert_glow" }),
      w: 47,
      h: 42,
    };
    const alert = {
      ...entry("icon", "buttontype", { sprite: "GFX_global_alert_icons" }),
      w: 47,
      h: 42,
    };
    expect(pickInStack([glow, alert])?.id).toBe("icon");

    expect(pickInStack([label, cap, seat])?.id).toBe("advisor_type_icon");
  });

  it("looks through a see-through veil, not an opaque window", () => {
    const veil = entry("Background", "background", {
      sprite: "GFX_opacity_layer",
    });
    const operatives = entry("total_operatives", "icontype", {
      sprite: "GFX_total_operatives",
    });
    expect(pickInStack([veil, operatives])?.id).toBe("total_operatives");
    expect(pickInStack([veil])?.id).toBe("Background");
    const popup = entry("Background", "background", {
      sprite: "GFX_division_designer_popup_bg",
    });
    expect(pickInStack([popup, operatives])?.id).toBe("Background");
  });

  it("keeps the topmost element otherwise", () => {
    const army = entry("army_button", "buttontype", { sprite: "GFX_army" });
    expect(pickInStack([army, button])?.id).toBe("army_button");

    expect(pickInStack([label, entry("click_area", "buttontype")])?.id).toBe(
      "manage_occupied_button_text",
    );

    expect(pickInStack([panel, button])?.id).toBe("politics_bg");
    expect(pickInStack([label, panel, button])?.id).toBe(
      "manage_occupied_button_text",
    );
  });

  it("shows the button's art with its label's text, and every element at the spot", () => {
    const win = {
      name: "countrypoliticsview",
      sprites: {
        GFX_button_148x34: {
          width: 148,
          height: 34,
          frames: 1,
          resolved: true,
          vanilla_url: "u",
        },
      },
      texts: {
        POLITICS_OCCUPIED_TERRITORIES: {
          kind: "loc",
          display: "Occupied Territories",
        },
      },
    } as unknown as GuiWindow;
    const target = editTargetOf(
      {
        owner: "countrypoliticsview",
        win,
        id: button.id,
        node: button.node,
        stack: [label, button, panel],
      },
      undefined,
      {},
      {},
    );
    expect(target.title).toBe("manage_occupied_button");
    expect(target.sprite).toBe("GFX_button_148x34");
    expect(target.text).toEqual({
      editable: true,
      via: "key",
      key: "POLITICS_OCCUPIED_TERRITORIES",
      current: "Occupied Territories",
      rewritten: false,
      scope: "everywhere",
      runtime: false,
    });
    expect(
      target.stack.map((s) => `${s.kind} · ${s.name}${s.picked ? " *" : ""}`),
    ).toEqual([
      "text · manage_occupied_button_text",
      "button · manage_occupied_button *",
      "panel · politics_bg",
    ]);
    const backdrop = editTargetOf(
      {
        owner: "x",
        win,
        id: button.id,
        node: button.node,
        stack: [button, entry("bg", "background", { name: "Background" })],
      },
      undefined,
      {},
      {},
    );
    expect(backdrop.stack[1]).toMatchObject({
      kind: "background",
      backdrop: true,
    });
  });

  it("is wired through the canvas, the layers and the panel", () => {
    const canvas = readFileSync(
      join(__dirname, "../gui-renderer/GuiCanvas.tsx"),
      "utf8",
    );
    expect(canvas).toContain("h = pickInStack(stack) ?? clicked;");

    expect(readFileSync(join(__dirname, "editContext.ts"), "utf8")).toContain(
      "shown: printed[entry.id]?.text ?? overrides[entry.id],",
    );
    const page = readFileSync(join(__dirname, "page.tsx"), "utf8");
    expect(page).toContain(
      "...pickedText(entry.id, leaderPrinted, runtimeOverrides, runtimeSources),",
    );

    expect(page).toContain(
      "setSelection(withPrintedOver(picked, printedOver));",
    );
    expect(page).toContain("onPickStacked={(id) => {");
  });
});

describe("what a replacement says about the export", () => {
  it("names the new picture and the .gfx copy", () => {
    const note = exportNote({
      export: {
        gfx: "interface/core.gfx",
        texture: "gfx/interface/game_view/gfx_government_button.dds",
      },
    });
    expect(note).toEqual({
      key: "exportGfx",
      values: {
        texture: "gfx/interface/game_view/gfx_government_button.dds",
        gfx: "interface/core.gfx",
      },
      ok: true,
    });
  });
  it("names only the picture for a loading picture, which no .gfx declares", () => {
    const note = exportNote({
      export: { texture: "gfx/loadingscreens/load_1.dds" },
    });
    expect(note).toEqual({
      key: "exportPath",
      values: { texture: "gfx/loadingscreens/load_1.dds" },
      ok: true,
    });
  });
  it("says so when the export will not carry it", () => {
    const note = exportNote({
      export_error: "no .gfx of the game or the mod declares GFX_x",
    });
    expect(note).toEqual({
      key: "exportNoneWhy",
      values: { error: "no .gfx of the game or the mod declares GFX_x" },
      ok: false,
    });
    expect(exportNote({})).toEqual({ key: "exportNone", ok: false });
  });
});

describe("where the edit panels open", () => {
  it("always in the column at the lower right; folded beside it", () => {
    expect(PANEL_PLACE.open.right).toContain("calc(12px + (100vw");

    expect(PANEL_PLACE.open.right).toContain("var(--component-bar, 0px)");

    expect(PANEL_PLACE.open.top).toBe("calc(64px + var(--edit-switcher, 0px))");

    expect(PANEL_PLACE.open.maxHeight).toBe(
      "calc(100vh - 176px - var(--edit-switcher, 0px))",
    );
    expect(PANEL_PLACE.folded.right).toContain("calc(68px + (100vw");
    expect(PANEL_PLACE.folded.bottom).toBe(56);
    const page = readFileSync(join(__dirname, "page.tsx"), "utf8");
    expect(page).not.toContain("side={");
  });

  it("puts the 新建/选择 cards at the head of the same column, one at a time", () => {
    expect(PANEL_PLACE.switcher.right).toBe(PANEL_PLACE.open.right);
    expect(PANEL_PLACE.switcher.top).toBe(64);
    expect(EDIT_SWITCHER_SPACE).toBe("72px");
    const page = readFileSync(join(__dirname, "page.tsx"), "utf8");
    expect(page).toContain(
      '["--edit-switcher" as string]: editSwitcher ? EDIT_SWITCHER_SPACE : "0px"',
    );

    for (const file of [
      "PersonSwitcher.tsx",
      "EventSwitcher.tsx",
      "FocusTreeSwitcher.tsx",
      "NewCountryControl.tsx",
    ]) {
      expect(readFileSync(join(__dirname, file), "utf8"), file).not.toMatch(
        /fixed (left|right)-4 top-16/,
      );
    }
  });

  it("folds without losing what is typed, and opens again on the next thing picked", () => {
    const panel = readFileSync(
      join(__dirname, "GamePanel.tsx"),
      "utf8",
    ).replace(/\s+/g, " ");
    expect(panel).toContain("hidden={folded}");
    expect(panel).toContain(
      "if (shownName !== name) { setShownName(name); setFolded(false); }",
    );
  });
});

describe("the two edit modes", () => {
  const page = readFileSync(join(__dirname, "page.tsx"), "utf8");

  it("switches both manual edit modes from their buttons", () => {
    expect(page).toContain('const t = useTranslations("gameViewControls")');
    expect(page).toContain('uiEdit ? t("exitUIEdit")');
    expect(page).toContain('editMode ? t("exitTextEdit")');
    expect(page).toContain('onClick={() => openManualEdit("ui")}');
    expect(page).toContain('onClick={() => openManualEdit("text")}');
  });

  it("is view-only by default and removes the old navigation/history controls", () => {
    expect(page).toContain(
      'const [editMode, setEditMode] = useState(() => params.get("edit") === "1");',
    );
    expect(page).toContain(
      'const [uiEdit, setUiEdit] = useState(() => params.get("edit") === "ui");',
    );
    expect(page).not.toContain("<ChangeLogButton");
    expect(page).not.toContain("回工作台");
    expect(page).not.toContain("返回工作台");
  });

  it("keeps technology nodes inspectable without granting edit access", () => {
    const tree = readFileSync(join(__dirname, "TechTreeLayer.tsx"), "utf8");
    const hit = readFileSync(join(__dirname, "FieldHit.tsx"), "utf8");
    expect(page).toContain("onPick={onPickTech}");
    expect(page).toContain("editable={editMode}");
    expect(page).toContain("<TechInspectPanel");
    expect(page).toContain('onRequestEdit={() => openManualEdit("text")}');
    expect(tree).toContain('appearance={editable ? "edit" : "inspect"}');
    expect(hit).toContain('appearance?: "edit" | "inspect"');
    expect(hit).toContain('shownAppearance === "edit" ? "1px dashed');
  });

  it("keeps every content hit wired while write panels remain edit-only", () => {
    expect(page).toContain("contentEditMode: editMode");
    expect(page).not.toContain("<ContentInspectPanel");
    expect(page).toContain("editable={editMode}");
    expect(page).toContain("onPick={onPickDecision}");
    expect(page).toContain("onPickField={onPickEventField}");
    expect(page).toContain("onEditPerson={onEditPerson}");
    expect(page).toContain("onEditSpirit={onEditSpirit}");
    expect(page).toContain("onEditEquipment={onEditEquipment}");
    expect(page).toContain("onEditTrait={onEditMioTrait}");
    expect(page).toContain("onEditUpgrade={onEditIntelUpgrade}");
    expect(page).toContain("onEditScientist={onEditScientist}");
    expect(page).toContain(
      "{editMode && !countryPanelOpen && base === DECISIONS",
    );
    expect(page).toContain("{editMode && !countryPanelOpen && personPick");
  });

  it("picks elements on the canvases only in UI 编辑, with nothing pressed through", () => {
    expect(page).toContain("editMode={uiEdit}");
    expect(page).toContain("editMode: uiEdit,");
    expect(page).not.toContain("playInEdit: navigatesInEdit");

    expect(page).toContain(
      "{uiEdit && selection && !selection.win.ui_panel && (",
    );
    expect(page).toContain(
      "{uiEdit && panelPick && (!selection || selection.win.ui_panel) && (",
    );
  });
});
