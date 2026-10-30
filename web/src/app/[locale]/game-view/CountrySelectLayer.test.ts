import { describe, expect, it } from "vitest";
import type { GuiCharacter, GuiSprite } from "@/lib/guiRenderApi";
import { countryLeaderPortraitUrl } from "./CountrySelectLayer";

function sprite(partial: Partial<GuiSprite>): GuiSprite {
  return {
    resolved: true,
    frames: 1,
    width: 156,
    height: 210,
    effect: null,
    effect_kind: null,
    cornered: false,
    bordersize: null,
    ...partial,
  };
}

function leader(portraitSprite: GuiSprite): GuiCharacter {
  return {
    token: "KOR_test_leader",
    name: "Test Leader",
    roles: ["country_leader"],
    portrait: "GFX_portrait_KOR_test_leader",
    portrait_sprite: portraitSprite,
  };
}

describe("country-select leader portraits", () => {
  it("serves project portraits and base-game replacements by GFX name", () => {
    expect(countryLeaderPortraitUrl(leader(sprite({})))).toContain(
      "sprite?name=GFX_portrait_KOR_test_leader",
    );
    expect(
      countryLeaderPortraitUrl(
        leader(
          sprite({
            vanilla_url:
              "/api/go/projects/69/gui-render/sprite?name=GFX_portrait_KOR_test_leader&v=content-hash",
            overridden: true,
          }),
        ),
      ),
    ).toBe(
      "/api/go/projects/69/gui-render/sprite?name=GFX_portrait_KOR_test_leader&v=content-hash",
    );
  });

  it("keeps untouched base-game portraits on the public store URL", () => {
    expect(
      countryLeaderPortraitUrl(
        leader(sprite({ vanilla_url: "https://store/leader.png" })),
      ),
    ).toBe("https://store/leader.png");
  });
});
