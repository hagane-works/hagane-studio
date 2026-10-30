import { describe, expect, it } from "vitest";
import type { GuiNode, GuiText, GuiWindow } from "@/lib/guiRenderApi";
import { assignIds } from "../gui-renderer/GuiCanvas";
import {
  editTargetOf,
  guiTextKey,
  guiTextOverrides,
  locEditOverrides,
  ownPictureName,
  ownPictureOf,
  removalOf,
  withHiddenMark,
  withOwnPicture,
  type EditSelection,
} from "./editModel";
import { matchNamePath } from "./gameNavigation";
import { leaderNamePrinted } from "./textSources";

function node(partial: Partial<GuiNode>): GuiNode {
  return {
    type: "icontype",
    name: null,
    position: { x: 0, y: 0 },
    size: null,
    orientation: null,
    origo: null,
    scale: null,
    sprite: null,
    frame: null,
    text: null,
    text_source: null,
    font: null,
    format: null,
    maxwidth: null,
    maxheight: null,
    text_color: null,
    centerposition: false,
    alwaystransparent: false,
    background: null,
    children: [],
    ...partial,
  };
}

function screen(texts: Record<string, GuiText>): GuiWindow {
  const root = node({
    type: "containerwindowtype",
    name: "countrypoliticsview",
    children: [
      node({
        type: "buttontype",
        name: "new_faction_button",
        sprite: "GFX_factions_button",
        text: "CREATE_FACTION",
      }),
      node({ type: "instanttextboxtype", name: "ideology", text: "ideology" }),
      node({ type: "instanttextboxtype", name: "score", text: "999" }),
      node({
        type: "instanttextboxtype",
        name: "second",
        text: "CREATE_FACTION",
      }),
    ],
  });
  assignIds(root);
  return {
    name: "countrypoliticsview",
    file: "x.gui",
    node: root,
    sprites: {},
    texts,
    fonts: {},
    overlap_groups: [],
  };
}

const TEXTS: Record<string, GuiText> = {
  CREATE_FACTION: { display: "Create Faction", kind: "loc" },
  ideology: { display: "ideology", kind: "literal" },
  "999": { display: "999", kind: "literal" },
};

const pick = (win: GuiWindow, name: string, shown?: string): EditSelection => {
  const target = matchNamePath(win.node, name)[0];
  return {
    owner: "countrypoliticsview",
    win,
    id: (target as GuiNode & { _id: string })._id,
    node: target,
    shown,
  };
};

describe("editTargetOf", () => {
  it("offers both the art and the text of a localised button", () => {
    const win = screen(TEXTS);
    const target = editTargetOf(
      pick(win, "new_faction_button"),
      undefined,
      {},
      {},
    );
    expect(target.sprite).toBe("GFX_factions_button");
    expect(target.spriteSize).toBeNull();
    expect(target.text).toEqual({
      editable: true,
      via: "key",
      key: "CREATE_FACTION",
      current: "Create Faction",
      rewritten: false,
      scope: "everywhere",
      runtime: false,
    });
  });

  it("marks a flag slot, whose sprite is only the gloss over the country's flag", () => {
    const win = screen(TEXTS);
    const flag = node({
      type: "buttontype",
      name: "player_flag",
      sprite: "GFX_shield_medium",
    });
    win.node.children = [...(win.node.children ?? []), flag];
    assignIds(win.node);
    win.sprites = {
      GFX_shield_medium: {
        block_type: "maskedshieldtype",
        width: 82,
        height: 52,
        frames: 1,
        resolved: true,
      } as GuiWindow["sprites"][string],
      GFX_factions_button: {
        block_type: "spritetype",
        width: 40,
        height: 40,
        frames: 1,
        resolved: true,
      } as GuiWindow["sprites"][string],
    };
    expect(editTargetOf(pick(win, "player_flag"), undefined, {}, {}).flag).toBe(
      true,
    );
    expect(
      editTargetOf(pick(win, "new_faction_button"), undefined, {}, {}).flag,
    ).toBe(false);
  });

  it("starts from the player's rewrite and knows it can be put back", () => {
    const win = screen(TEXTS);
    const target = editTargetOf(
      pick(win, "new_faction_button"),
      undefined,
      { CREATE_FACTION: "Found an alliance" },
      {},
    );
    expect(target.text).toMatchObject({
      editable: true,
      current: "Found an alliance",
      rewritten: true,
    });
    const saved = editTargetOf(
      pick(win, "new_faction_button"),
      undefined,
      {},
      { CREATE_FACTION: "x" },
    );
    expect(saved.text).toMatchObject({ rewritten: true });
  });

  it("explains a text the engine writes from the country's data instead of offering an edit that would never show", () => {
    const win = screen(TEXTS);
    const target = editTargetOf(
      pick(win, "ideology", "Fascist"),
      "Fascist",
      {},
      {},
    );
    expect(target.text).toMatchObject({ editable: false, current: "Fascist" });
  });

  it("explains a number the .gui only holds a stand-in for", () => {
    const win = screen(TEXTS);
    const target = editTargetOf(pick(win, "score"), undefined, {}, {});
    expect(target.text).toEqual({
      editable: false,
      current: "999",
      reason: "placeholder",
    });
    expect(target.sprite).toBeNull();
  });
});

describe("editTargetOf · how a text is rewritten", () => {
  const politics = () => {
    const root = node({
      type: "containerwindowtype",
      name: "countrypoliticsview",
      children: [
        node({
          type: "instanttextboxtype",
          name: "leader_name",
          text: "leader_name",
        }),
        node({
          type: "instanttextboxtype",
          name: "ideology",
          text: "ideology",
        }),
        node({ type: "instanttextboxtype", name: "title", text: "Politics" }),
        node({
          type: "instanttextboxtype",
          name: "rewritten",
          text: "GUISUB_GUI_X_0000",
        }),
      ],
    });
    assignIds(root);
    const win: GuiWindow = {
      name: "countrypoliticsview",
      file: "interface/countrypoliticsview.gui",
      node: root,
      sprites: {},
      texts: {
        leader_name: { display: "Leader name", kind: "loc" },
        ideology: { display: "ideology", kind: "literal" },
        Politics: { display: "Politics", kind: "literal" },
        GUISUB_GUI_X_0000: { display: "政治2222", kind: "loc" },
      },
      fonts: {},
      overlap_groups: [],
    };
    return win;
  };

  it("rewrites the leader's name, not the key of the box CharacterLayer draws it over", () => {
    const win = politics();
    const printed = leaderNamePrinted(win, {
      token: "ABK_boris",
      name: "波罗斯",
    });
    const id = Object.keys(printed)[0];
    const selection = {
      ...pick(win, "leader_name"),
      shown: printed[id].text,
      source: printed[id].source,
    };
    expect(editTargetOf(selection, selection.shown, {}, {}).text).toEqual({
      editable: true,
      via: "person",
      token: "ABK_boris",
      current: "波罗斯",
    });
  });

  it("rewrites the key a run-time text is printed from, and says who else shows it", () => {
    const win = politics();
    const source = {
      kind: "key",
      key: "neutrality",
      scope: "allCountries",
    } as const;
    const selection = { ...pick(win, "ideology", "中立主义"), source };
    expect(editTargetOf(selection, "中立主义", {}, {}).text).toEqual({
      editable: true,
      via: "key",
      key: "neutrality",
      current: "中立主义",
      rewritten: false,
      scope: "allCountries",
      runtime: true,
    });
    expect(
      editTargetOf(selection, "中立主义", { neutrality: "不结盟" }, {}).text,
    ).toMatchObject({
      current: "不结盟",
      rewritten: true,
    });
  });

  it("explains a run-time text with nothing behind it to rewrite", () => {
    const win = politics();
    expect(
      editTargetOf(pick(win, "ideology", "315.97K"), "315.97K", {}, {}).text,
    ).toEqual({
      editable: false,
      current: "315.97K",
      reason: "computed",
    });
    expect(
      editTargetOf(pick(win, "ideology", "Somewhere"), "Somewhere", {}, {})
        .text,
    ).toMatchObject({
      editable: false,
      reason: "runtimeCountry",
    });

    expect(
      editTargetOf(pick(win, "ideology", ""), "", {}, {}).text,
    ).toMatchObject({ reason: "runtimeCountry" });
  });

  it("gives words written into the .gui a key of their own, in the screen's file", () => {
    const win = politics();
    expect(editTargetOf(pick(win, "title"), undefined, {}, {}).text).toEqual({
      editable: true,
      via: "gui",
      target: "countrypoliticsview/title",
      file: "interface/countrypoliticsview.gui",
      originalText: "Politics",
      originalKind: "literal",
      current: "Politics",
      rewritten: false,
    });
  });

  it("keeps rewriting a .gui's words in its overlay once they have a key there", () => {
    const win = politics();
    expect(
      editTargetOf(pick(win, "rewritten"), undefined, {}, {}).text,
    ).toMatchObject({
      via: "gui",
      target: "countrypoliticsview/rewritten",
      current: "政治2222",
      rewritten: true,
    });
  });

  it("starts from this session's rewrite of a .gui's words", () => {
    const win = politics();
    const guiTexts = {
      [guiTextKey(win.file, "countrypoliticsview/title")]: "政治2222",
    };
    expect(
      editTargetOf(pick(win, "title"), undefined, {}, {}, guiTexts).text,
    ).toMatchObject({
      via: "gui",
      current: "政治2222",
      rewritten: true,
    });
    const id = (
      matchNamePath(win.node, "title")[0] as GuiNode & { _id: string }
    )._id;
    expect(guiTextOverrides(win, guiTexts)).toEqual({ [id]: "政治2222" });
    expect(
      guiTextOverrides({ ...win, file: "interface/other.gui" }, guiTexts),
    ).toEqual({});
  });
});

describe("ownPictureOf · a picture of the picked element's own", () => {
  const sprite = (
    width: number,
    height: number,
    frames = 1,
    cornered = false,
  ) => ({
    resolved: true,
    frames,
    width,
    height,
    effect: null,
    effect_kind: null,
    cornered,
    bordersize: cornered ? { x: 64, y: 64 } : null,
  });
  function panel(size: GuiNode["size"]): GuiWindow {
    const root = node({
      type: "containerwindowtype",
      name: "politics",
      size,
      background: node({
        type: "background",
        name: "Background",
        sprite: "GFX_tiled_window",
      }),
      children: [
        node({
          type: "buttontype",
          name: "faction",
          sprite: "GFX_button_strip",
        }),
        node({
          type: "icontype",
          name: "tile",
          sprite: "GFX_tiled_window",
          size: { width: 300, height: 40 },
        }),
        node({ type: "icontype", sprite: "GFX_button_strip" }),
      ],
    });
    assignIds(root);
    return {
      name: "politics",
      file: "interface/politics.gui",
      scaffold_hash: "abc",
      node: root,
      sprites: {
        GFX_tiled_window: sprite(190, 190, 1, true),
        GFX_button_strip: sprite(90, 30, 3),
      },
      texts: {},
      fonts: {},
      overlap_groups: [],
    };
  }
  const backdrop = (
    win: GuiWindow,
    drawn?: { width: number; height: number },
  ): EditSelection => {
    const bg = win.node.background as GuiNode & { _id: string };
    return {
      owner: "politics",
      win,
      id: bg._id,
      node: bg,
      stack: drawn ? [{ id: bg._id, node: bg, size: drawn }] : undefined,
    };
  };

  it("fills a window: its backdrop becomes one picture the window's size, which the window is pointed at", () => {
    const win = panel({ width: 620, height: 480 });
    const own = editTargetOf(backdrop(win), undefined, {}, {}).ownPicture;
    expect(own).toEqual({
      file: "interface/politics.gui",
      target: "politics",
      gfx: ownPictureName("interface/politics.gui", "politics"),
      width: 620,
      height: 480,
      frames: 1,
      replaced: false,
    });
  });

  it("takes the size the window is drawn at when the .gui gives none in numbers", () => {
    const win = panel({ width: "100%%", height: "100%%" });
    expect(
      ownPictureOf(
        backdrop(win, { width: 800, height: 600 }),
        "GFX_tiled_window",
      ),
    ).toMatchObject({ width: 800, height: 600 });
  });

  it("fits the part of the window the screen shows, not what it declares past the screen's edge", () => {
    const win = panel({ width: 550, height: 1080 });
    expect(
      ownPictureOf(
        backdrop(win, { width: 550, height: 655 }),
        "GFX_tiled_window",
      ),
    ).toMatchObject({ width: 550, height: 655 });
  });

  it("keeps a button's sprite size and frame count, so the frame it draws is one it has", () => {
    const win = panel(null);
    const own = editTargetOf(
      pick(win, "faction"),
      undefined,
      {},
      {},
    ).ownPicture;
    expect(own).toMatchObject({
      target: "politics/faction",
      width: 90,
      height: 30,
      frames: 3,
      replaced: false,
    });
  });

  it("makes a stretched nine-slice the element's own size", () => {
    const win = panel(null);
    expect(ownPictureOf(pick(win, "tile"), "GFX_tiled_window")).toMatchObject({
      width: 300,
      height: 40,
      frames: 1,
    });
  });

  it("has none for an element the export cannot find by name, or a screen whose file it cannot carry", () => {
    const win = panel(null);
    const unnamed = win.node.children[2] as GuiNode & { _id: string };
    expect(
      ownPictureOf(
        { owner: "politics", win, id: unnamed._id, node: unnamed },
        "GFX_button_strip",
      ),
    ).toBeNull();
    const loose = { ...win, scaffold_hash: undefined };
    expect(ownPictureOf(pick(loose, "faction"), "GFX_button_strip")).toBeNull();
  });

  it("knows the element already draws its own picture, and draws it at once after a replacement", () => {
    const win = panel(null);
    const picked = pick(win, "faction");
    const own = ownPictureOf(picked, "GFX_button_strip")!;
    const after = withOwnPicture(picked, own.gfx, {
      width: own.width,
      height: own.height,
      frames: own.frames,
    });
    expect(after.node.sprite).toBe(own.gfx);
    expect(after.win.sprites[own.gfx]).toMatchObject({
      resolved: true,
      width: 90,
      height: 30,
      frames: 3,
    });
    expect(editTargetOf(after, undefined, {}, {}).ownPicture).toMatchObject({
      gfx: own.gfx,
      replaced: true,
      width: 90,
      frames: 3,
    });

    const filled = withOwnPicture(backdrop(win), "GFX_gv_politics_x", {
      width: 620,
      height: 480,
      frames: 1,
    });
    expect(filled.node.sprite).toBe("GFX_gv_politics_x");
  });

  it("keeps a backdrop's frames: the game shows a bookmark's second one while it is picked", () => {
    const root = node({
      type: "containerwindowtype",
      name: "bookmark_entry",
      size: { width: 232, height: 211 },
      background: node({
        type: "background",
        name: "background",
        sprite: "GFX_bookmark_entry_bg",
      }),
    });
    assignIds(root);
    const win: GuiWindow = {
      name: "bookmark_entry",
      file: "interface/frontendgamesetupview.gui",
      scaffold_hash: "abc",
      node: root,
      sprites: { GFX_bookmark_entry_bg: sprite(464, 211, 2) },
      texts: {},
      fonts: {},
      overlap_groups: [],
    };
    expect(ownPictureOf(backdrop(win), "GFX_bookmark_entry_bg")).toMatchObject({
      width: 464,
      height: 211,
      frames: 2,
    });
  });

  it("has none for the main menu's background or the loading screen: the game swaps their picture itself", () => {
    for (const name of ["frontend_background", "load_screen"]) {
      const root = node({
        type: "containerwindowtype",
        name,
        size: { width: 1920, height: 1440 },
        background: node({
          type: "background",
          name: "Background",
          sprite: "GFX_load_taog_ast",
        }),
      });
      assignIds(root);
      const win: GuiWindow = {
        name,
        file: "interface/frontendmainview.gui",
        scaffold_hash: "abc",
        node: root,
        sprites: { GFX_load_taog_ast: sprite(1920, 1440) },
        texts: {},
        fonts: {},
        overlap_groups: [],
      };
      expect(ownPictureOf(backdrop(win), "GFX_load_taog_ast")).toBeNull();
    }
  });

  it("names the picture by its file and element, in a name the game takes", () => {
    const name = ownPictureName(
      "interface/politics.gui",
      "politics/faction button",
    );
    expect(name).toMatch(/^GFX_[A-Za-z0-9_]{0,75}[A-Za-z0-9]$/);
    expect(name.startsWith("GFX_gv_faction_button_")).toBe(true);
    expect(
      ownPictureName("interface/politics.gui", "politics/faction button"),
    ).toBe(name);
    expect(
      ownPictureName("interface/other.gui", "politics/faction button"),
    ).not.toBe(name);
    expect(
      ownPictureName("a.gui", `w/${"x".repeat(200)}`).length,
    ).toBeLessThanOrEqual(80);
  });
});

describe("removalOf · taking the picked element off its screen", () => {
  function panel(): GuiWindow {
    const root = node({
      type: "containerwindowtype",
      name: "politics",
      background: node({
        type: "background",
        name: "Background",
        sprite: "GFX_bg",
      }),
      children: [
        node({ type: "buttontype", name: "close", sprite: "GFX_close" }),
        node({ type: "icontype", sprite: "GFX_x" }),
      ],
    });
    assignIds(root);
    return {
      name: "politics",
      file: "interface/politics.gui",
      scaffold_hash: "abc",
      node: root,
      sprites: {},
      texts: {},
      fonts: {},
      overlap_groups: [],
    };
  }

  it("names the element's file and NAME-PATH, and whether it is hidden", () => {
    const win = panel();
    const picked = pick(win, "close");
    expect(removalOf(picked)).toEqual({
      file: "interface/politics.gui",
      target: "politics/close",
      hidden: false,
    });
    expect(
      editTargetOf(withHiddenMark(picked, true), undefined, {}, {}).removal
        ?.hidden,
    ).toBe(true);
    expect(
      withHiddenMark(withHiddenMark(picked, true), false).node.raw
        ?.gameview_hidden,
    ).toBeUndefined();
  });

  it("has none for the window itself, its backdrop, an unnamed element or a file the export cannot carry", () => {
    const win = panel();
    const at = (n: GuiNode): EditSelection => ({
      owner: "politics",
      win,
      id: (n as GuiNode & { _id: string })._id,
      node: n,
    });
    expect(removalOf(at(win.node))).toBeNull();
    expect(removalOf(at(win.node.background as GuiNode))).toBeNull();
    expect(removalOf(at(win.node.children[1]))).toBeNull();
    expect(
      removalOf(pick({ ...win, scaffold_hash: undefined }, "close")),
    ).toBeNull();
  });
});

describe("locEditOverrides", () => {
  it("rewrites every node on the screen that prints the key, at once", () => {
    const win = screen(TEXTS);
    const out = locEditOverrides(win, { CREATE_FACTION: "Found an alliance" });
    const ids = ["new_faction_button", "second"].map(
      (name) =>
        (matchNamePath(win.node, name)[0] as GuiNode & { _id: string })._id,
    );
    expect(Object.keys(out).sort()).toEqual(ids.sort());
    expect(Object.values(out)).toEqual([
      "Found an alliance",
      "Found an alliance",
    ]);
  });

  it("changes nothing without rewrites", () => {
    expect(locEditOverrides(screen(TEXTS), {})).toEqual({});
  });
});
