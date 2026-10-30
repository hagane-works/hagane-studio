import { readFileSync } from "node:fs";
import { join } from "node:path";
import { describe, expect, it, vi } from "vitest";
import type { GuiCharacter, GuiSprite } from "@/lib/guiRenderApi";
import { guiRenderApi } from "@/lib/guiRenderApi";
import { slotArt } from "./CharacterLayer";
import { editTargetOf } from "./editModel";
import { pickable } from "./spriteFrame";
import {
  artSelection,
  artUrl,
  invalidateLayerSpriteMeta,
  layerSpriteMeta,
} from "./UiEditArt";

const sprite = (over: Partial<GuiSprite> = {}): GuiSprite =>
  ({
    vanilla_url:
      "https://example.com/vanilla/gfx/interface/ideas/idea_sour_loser.png",
    width: 60,
    height: 68,
    frames: 1,
    resolved: true,
    ...over,
  }) as GuiSprite;

describe("a layer's picture in UI 编辑", () => {
  it("is picked as one element with its own art", () => {
    const picked = artSelection(
      "spirit:0",
      "spirit:0:GFX_idea_sour_loser",
      "GFX_idea_sour_loser",
      sprite(),
      "Sour Loser",
    );
    const target = editTargetOf(picked, undefined, {}, {});
    expect(target.title).toBe("Sour Loser");
    expect(target.sprite).toBe("GFX_idea_sour_loser");
    expect(target.spriteSize).toEqual({ width: 60, height: 68, frames: 1 });
    expect(target.preview?.url).toContain("idea_sour_loser.png");
    expect(target.text).toBeNull();
  });

  it("draws a picture replaced this session from the service", () => {
    expect(artUrl("GFX_idea_sour_loser", sprite(), {})).toContain(
      "idea_sour_loser.png",
    );
    expect(
      artUrl("GFX_idea_sour_loser", sprite(), {
        GFX_idea_sour_loser: { version: 7, width: 60, height: 68 },
      }),
    ).toContain("sprite?name=GFX_idea_sour_loser&v=7");
    expect(artUrl("GFX_x", undefined, {})).toBeUndefined();
  });

  it("forgets cached layer metadata after the picture changes", async () => {
    const name = "GFX_test_layer_cache";
    const projectId = 98765;
    let width = 60;
    const fetchMock = vi.fn(async () => ({
      ok: true,
      json: async () => ({ sprites: { [name]: sprite({ width }) } }),
    }));
    vi.stubGlobal("fetch", fetchMock);
    try {
      expect((await layerSpriteMeta(projectId, name))?.width).toBe(60);
      width = 100;
      expect((await layerSpriteMeta(projectId, name))?.width).toBe(60);
      expect(fetchMock).toHaveBeenCalledTimes(1);

      invalidateLayerSpriteMeta(projectId, name);
      expect((await layerSpriteMeta(projectId, name))?.width).toBe(100);
      expect(fetchMock).toHaveBeenCalledTimes(2);
    } finally {
      invalidateLayerSpriteMeta(projectId, name);
      guiRenderApi.setProjectId(null);
      vi.unstubAllGlobals();
    }
  });

  it("names what each box of a politics row draws", () => {
    const face = {
      token: "GER_x",
      name: "Hjalmar Schacht",
      roles: [],
      portrait: "GFX_portrait_schacht",
      portrait_sprite: sprite(),
    } as GuiCharacter;
    expect(
      slotArt({ slot: "political_advisor", person: face, idea: null }, {}),
    ).toMatchObject({ gfx: "GFX_portrait_schacht", title: "Hjalmar Schacht" });
    expect(
      slotArt(
        {
          slot: "economy",
          person: null,
          idea: {
            token: "partial_economic_mobilisation",
            label: "Partial Mobilization",
            icon: "GFX_idea_partial_economic_mobilisation",
            icon_sprite: sprite(),
          },
        },
        {},
      ),
    ).toMatchObject({
      gfx: "GFX_idea_partial_economic_mobilisation",
      title: "Partial Mobilization",
    });
    const seat = sprite({ width: 63, height: 66 });
    expect(
      slotArt(
        { slot: "army_chief", person: null, idea: null },
        { GFX_idea_slot_army_chief: seat },
      ),
    ).toEqual({
      gfx: "GFX_idea_slot_army_chief",
      sprite: seat,
      title: "army_chief",
    });
  });

  it("covers the spirits, the leader, every row box and the pie", () => {
    const layer = readFileSync(join(__dirname, "CharacterLayer.tsx"), "utf8");
    expect(layer).toContain("owner={`spirit:${at}`}");
    expect(layer).toContain('owner="leader"');
    expect(layer).toContain("const art = slotArt(fill, emptyArt);");
    expect(
      readFileSync(join(__dirname, "PoliticsLayer.tsx"), "utf8"),
    ).toContain('<UiEditArt owner="pie" gfx={OVERLAY}');
  });

  it("tags a layer's picture with its sprite and frame", () => {
    expect(pickable("GFX_buildings_strip", 3, "Infrastructure")).toEqual({
      "data-sprite": "GFX_buildings_strip",
      "data-frame": 3,
      "data-title": "Infrastructure",
    });
    expect(pickable(undefined)).toEqual({});
    const tagged: Record<string, string> = {
      "ConstructionLayer.tsx":
        "{...pickable(STRIP, building.icon_frame || 1, building.label || building.name)}",
      "LogisticsLayer.tsx":
        '{...pickable("GFX_resources_strip", frames[resource] ?? 1, resource)}',
      "DesignerLayer.tsx": "{...artTag(name, 1, label || name)}",
    };
    for (const [file, tag] of Object.entries(tagged))
      expect(readFileSync(join(__dirname, file), "utf8")).toContain(tag);
    expect(
      readFileSync(join(__dirname, "../gui-renderer/RichText.tsx"), "utf8"),
    ).toContain("data-sprite={iconSpriteName(item.name)}");
  });

  it("is picked by the page, where nothing on the screen presses in UI 编辑", () => {
    const page = readFileSync(join(__dirname, "page.tsx"), "utf8");
    expect(page).toContain(
      "onPointerDownCapture={uiEdit ? pickInUiEdit : undefined}",
    );
    expect(page).toContain(
      "onClickCapture={uiEdit ? (e) => e.stopPropagation() : undefined}",
    );
    expect(page).toContain(
      '[data-ui-edit-stage] [role="button"], [data-ui-edit-stage] button { pointer-events: none !important; }',
    );
    expect(page).toContain("data-screen-layer={owner}");
  });

  it("covers each division template's own counter", () => {
    const layer = readFileSync(join(__dirname, "DeploymentLayer.tsx"), "utf8");
    expect(layer).toContain("owner={`${entry.name}:${left},${top}:counter`}");
    expect(layer).toContain("gfx={template.icon}");
    expect(layer).toContain(
      "const counterUrl = artUrl(template.icon, counter, swaps);",
    );
  });
});
