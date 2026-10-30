import { readFileSync } from "node:fs";
import { join } from "node:path";
import { describe, expect, it } from "vitest";
import type { GuiNode, GuiWindow } from "@/lib/guiRenderApi";
import type { SuperEventRow } from "@/lib/api/scenario";
import { matchNamePath } from "./gameNavigation";
import { superEventLabel } from "./EventSwitcher";
import {
  isSuperEventWindow,
  superEventTexts,
  superEventWindow,
  superEventWindowName,
} from "./superEventModel";

const row = (extra: Partial<SuperEventRow> = {}): SuperEventRow => ({
  id: 7,
  project_id: 2,
  number: 3,
  country_tag: "",
  title_en: "",
  title_zh: "",
  desc_en: "",
  desc_zh: "",
  quote_en: "",
  quote_zh: "",
  btn_en: "",
  btn_zh: "",
  display_name_locs: {},
  description_locs: {},
  quote_locs: {},
  button_locs: {},
  picture_path: "",
  song_token: "",
  auto_fire_on_startup: false,
  auto_fire_after_days: 0,
  options: [],
  source: "created",
  created_at: "",
  updated_at: "2026-09-30T20:00:00",
  ...extra,
});

const node = (
  name: string,
  type: string,
  children: GuiNode[] = [],
  extra: Partial<GuiNode> = {},
): GuiNode =>
  ({ name, type, children, position: { x: 0, y: 0 }, ...extra }) as GuiNode;
const idOf = (n: GuiNode | undefined) =>
  (n as GuiNode & { _id?: string })?._id ?? "";

const window = (): GuiWindow =>
  ({
    name: "GER_SE_main_container",
    sprites: {},
    texts: {},
    fonts: {},
    node: node("GER_SE_main_container", "containerwindowtype", [
      node("GER_SE_bg", "icontype"),
      node("GER_SE_picture_positioning_entry", "icontype", [], {
        sprite: "GFX_GER_SE_pic_entry",
      }),
      node("GER_SE_frame_overlay", "icontype"),
      node("GER_SE_exit_btn", "buttontype"),
      node("GER_SE_btn", "instanttextboxtype"),
      node("GER_SE_title", "instanttextboxtype"),
      node("GER_SE_desc_left", "instanttextboxtype"),
      node("GER_SE_desc_center", "instanttextboxtype"),
      node("GER_SE_quote", "instanttextboxtype"),
      node("GER_SE_quote_extra", "instanttextboxtype"),
    ]),
  }) as unknown as GuiWindow;

describe("a super event's window", () => {
  it("is the window of its country, or of the project's", () => {
    expect(superEventWindowName(row({ country_tag: "eng" }), "GER")).toBe(
      "ENG_SE_main_container",
    );
    expect(superEventWindowName(row(), "ger")).toBe("GER_SE_main_container");
    expect(superEventWindowName(row(), null)).toBe("UNK_SE_main_container");
    expect(isSuperEventWindow("GER_SE_main_container")).toBe(true);
    expect(isSuperEventWindow("EventWindow")).toBe(false);
  });

  it("words its texts as the export's English: locale entry, English column, then the placeholders", () => {
    expect(superEventTexts(row())).toEqual({
      title: "(superevent 0003)",
      desc: "",
      quote: "",
      button: "(continue)",
    });
    expect(
      superEventTexts(
        row({
          title_en: "Old",
          display_name_locs: { en: " New " },
          desc_en: "Desc",
          btn_en: "Go",
        }),
      ),
    ).toEqual({
      title: "New",
      desc: "Desc",
      quote: "",
      button: "Go",
    });
    expect(superEventLabel(row({ title_en: "Anschluss" }))).toBe(
      "#0003 Anschluss",
    );
  });

  it("fills the four texts, hides the centred copies and puts the upload in the picture box", () => {
    const out = superEventWindow(
      window(),
      row({
        title_en: "Anschluss",
        desc_en: "Austria joins.",
        quote_en: "q",
        btn_en: "Heim ins Reich",
      }),
      "/pic.png",
    );
    const text = (name: string) =>
      out.texts[idOf(matchNamePath(out.win.node, name)[0])];
    expect([
      text("GER_SE_title"),
      text("GER_SE_desc_left"),
      text("GER_SE_quote"),
      text("GER_SE_btn"),
    ]).toEqual(["Anschluss", "Austria joins.", "q", "Heim ins Reich"]);
    expect(out.hidden).toEqual([
      idOf(matchNamePath(out.win.node, "GER_SE_desc_center")[0]),
      idOf(matchNamePath(out.win.node, "GER_SE_quote_extra")[0]),
    ]);
    const picture = matchNamePath(
      out.win.node,
      "GER_SE_picture_positioning_entry",
    )[0];
    expect(picture.sprite).toBe("GFX_GER_SE_picture_row_7");
    expect(out.win.sprites["GFX_GER_SE_picture_row_7"]).toMatchObject({
      vanilla_url: "/pic.png",
      width: 460,
      height: 252,
    });

    const plain = superEventWindow(window(), row(), null);
    expect(
      matchNamePath(plain.win.node, "GER_SE_picture_positioning_entry")[0]
        .sprite,
    ).toBe("GFX_GER_SE_pic_entry");
  });
});

describe("a super event's edit panel", () => {
  it("is picked from the event switcher in edit mode and shown in its own window", () => {
    const page = readFileSync(join(__dirname, "page.tsx"), "utf8");

    expect(page).toContain(
      "const eventScreen = base === EVENT_WINDOW || isSuperEventWindow(base);",
    );
    expect(page).toContain(
      "const editSwitcher = !editMode && !eventScreen ? null : eventScreen ? (",
    );
    expect(page).toContain("browseOnly={!editMode}");
    expect(page).toContain(
      "openWindow(superEventWindowName(row, projectTag.data));",
    );
    expect(page).toContain("superEventView?.win ?? win");
    expect(page).toContain(
      "void superEventRow.mutate(saved, { revalidate: false });",
    );
    const switcher = readFileSync(join(__dirname, "EventSwitcher.tsx"), "utf8");
    expect(switcher).toContain('{t("newSuperEvent")}');
    expect(switcher).toContain("{!browseOnly && (");
    expect(switcher).toContain("superEventsApi.create(projectId, {");
  });
});
