import { describe, expect, it } from "vitest";
import {
  guiRenderApi,
  type GuiCharacter,
  type GuiIdeaCategory,
} from "@/lib/guiRenderApi";
import {
  emptySlotSprite,
  fillCategorySlots,
  ideasListBox,
  portraitURL,
  spiritPlace,
} from "./CharacterLayer";

function person(token: string, slot: string): GuiCharacter {
  return { token, name: token, roles: ["advisor"], slot };
}

const CATEGORIES: GuiIdeaCategory[] = [
  {
    name: "government",
    label: "Laws & Government",
    index: 3,
    slots: [
      {
        name: "mobilization_laws",
        label: "Conscription Law",
        character: false,
      },
      { name: "trade_laws", label: "Trade Law", character: false },
      { name: "economy", label: "Economy Law", character: false },
      {
        name: "political_advisor",
        label: "Political Advisor",
        character: true,
      },
      {
        name: "political_advisor",
        label: "Political Advisor",
        character: true,
      },
      {
        name: "political_advisor",
        label: "Political Advisor",
        character: true,
      },
    ],
  },
  {
    name: "research_production",
    label: "Research & Production",
    index: 4,
    slots: [
      {
        name: "industrial_concern",
        label: "Industrial Concern",
        character: false,
      },
      { name: "theorist", label: "Theorist", character: true },
    ],
  },
];

describe("fillCategorySlots", () => {
  it("puts a different advisor in each character slot and leaves law slots empty", () => {
    const rows = fillCategorySlots(CATEGORIES, [
      person("dai_li", "political_advisor"),
      person("soong", "political_advisor"),
      person("chiang_ching_kuo", "political_advisor"),
      person("jiang_baili", "theorist"),
    ]);

    expect(rows[0].slots.slice(0, 3).map((s) => s.person)).toEqual([
      null,
      null,
      null,
    ]);
    expect(rows[0].slots.slice(3).map((s) => s.person?.token)).toEqual([
      "dai_li",
      "soong",
      "chiang_ching_kuo",
    ]);
    expect(rows[1].slots.map((s) => s.person?.token ?? null)).toEqual([
      null,
      "jiang_baili",
    ]);
  });

  it("carries the declared index through as the category icon frame", () => {
    const rows = fillCategorySlots(CATEGORIES, []);
    expect(rows.map((row) => row.frame)).toEqual([3, 4]);
  });

  it("leaves a slot empty when the country has nobody left for it", () => {
    const rows = fillCategorySlots(CATEGORIES, [
      person("dai_li", "political_advisor"),
    ]);
    expect(rows[0].slots.map((s) => s.person?.token ?? null)).toEqual([
      null,
      null,
      null,
      "dai_li",
      null,
      null,
    ]);
    expect(rows[1].slots[1].person).toBeNull();
  });

  it("ignores people whose slot no category declares", () => {
    const rows = fillCategorySlots(CATEGORIES, [
      person("chen_cheng", "high_command"),
    ]);
    expect(
      rows.flatMap((row) => row.slots).every((s) => s.person === null),
    ).toBe(true);
  });
});

describe("fillCategorySlots · who is already hired", () => {
  it("seats the country's hired advisors before the rest of the candidates", () => {
    const rows = fillCategorySlots(CATEGORIES, [
      person("candidate_a", "political_advisor"),
      { ...person("in_office", "political_advisor"), hired: true },
      person("candidate_b", "political_advisor"),
    ]);
    expect(rows[0].slots[3].person?.token).toBe("in_office");
    expect(rows[0].slots.slice(4).map((s) => s.person?.token)).toEqual([
      "candidate_a",
      "candidate_b",
    ]);
  });

  it("keeps the declared order among advisors that are equally hired", () => {
    const rows = fillCategorySlots(CATEGORIES, [
      person("first", "political_advisor"),
      person("second", "political_advisor"),
    ]);
    expect(rows[0].slots.slice(3, 5).map((s) => s.person?.token)).toEqual([
      "first",
      "second",
    ]);
  });
});

describe("fillCategorySlots · laws and manufacturers", () => {
  it("puts the country's law in the slot that holds things, not people", () => {
    const rows = fillCategorySlots(CATEGORIES, [], {
      trade_laws: [{ token: "limited_exports", label: "Limited Exports" }],
      economy: [
        {
          token: "partial_economic_mobilisation",
          label: "Partial Mobilization",
        },
      ],
    });
    expect(rows[0].slots[1].idea?.token).toBe("limited_exports");
    expect(rows[0].slots[2].idea?.token).toBe("partial_economic_mobilisation");

    expect(rows[0].slots[0].idea).toBeNull();

    expect(rows[0].slots.slice(3).every((s) => s.idea === null)).toBe(true);
  });

  it("gives repeated slots of one kind the next thing the country holds", () => {
    const rows = fillCategorySlots(
      [
        {
          name: "research_production",
          label: "Research & Production",
          index: 4,
          slots: [
            {
              name: "industrial_concern",
              label: "Industrial Concern",
              character: false,
            },
            {
              name: "industrial_concern",
              label: "Industrial Concern",
              character: false,
            },
          ],
        },
      ],
      [],
      { industrial_concern: [{ token: "ig_faben" }, { token: "krupp" }] },
    );
    expect(rows[0].slots.map((s) => s.idea?.token)).toEqual([
      "ig_faben",
      "krupp",
    ]);
  });
});

describe("portraitURL", () => {
  it("serves a resolved project portrait through the GUI service", () => {
    guiRenderApi.setProjectId(7);
    const portrait = "GFX_portrait_TST_ada";
    expect(
      portraitURL(
        {
          token: "TST_ada",
          name: "Ada",
          roles: ["country_leader"],
          portrait,
          portrait_sprite: {
            resolved: true,
            frames: 1,
            width: 156,
            height: 210,
            effect: null,
            effect_kind: null,
            cornered: false,
            bordersize: null,
          },
        },
        {},
      ),
    ).toBe("/api/go/projects/7/gui-render/sprite?name=GFX_portrait_TST_ada");
    guiRenderApi.setProjectId(null);
  });
});

describe("fillCategorySlots · the game's view and the editor's", () => {
  const people = [
    person("candidate_a", "political_advisor"),
    { ...person("in_office", "political_advisor"), hired: true },
  ];

  it("leaves slots nobody holds empty outside edit mode, naming each slot's kind", () => {
    const rows = fillCategorySlots(CATEGORIES, people, {}, false);
    expect(rows[0].slots.map((s) => s.person?.token ?? null)).toEqual([
      null,
      null,
      null,
      "in_office",
      null,
      null,
    ]);
    expect(rows[0].slots.map((s) => s.slot)).toEqual(
      CATEGORIES[0].slots.map((s) => s.name),
    );
  });

  it("fills them with candidates in edit mode, marked as such", () => {
    const rows = fillCategorySlots(CATEGORIES, people, {}, true);
    expect(
      rows[0].slots.slice(3, 5).map((s) => [s.person?.token, s.candidate]),
    ).toEqual([
      ["in_office", false],
      ["candidate_a", true],
    ]);
  });

  it("names the empty art after the slot's kind", () => {
    expect(emptySlotSprite("political_advisor")).toBe(
      "GFX_idea_slot_political_advisor",
    );

    expect(emptySlotSprite("tank_manufacturer")).toBe("GFX_tank_mio_roster");
    expect(emptySlotSprite("industrial_concern")).toBe(
      "GFX_idea_slot_industrial_concern",
    );
  });
});

describe("spiritPlace · the national spirit row", () => {
  const rows = { row: { x: 180, y: 169 }, small: { x: 190, y: 170 } };
  const size = { w: 64, h: 64 };

  it("keeps six spirits in the 59px row at their own size", () => {
    expect(spiritPlace(0, 6, rows, size)).toEqual({
      x: 180,
      y: 169,
      w: 64,
      h: 64,
    });
    expect(spiritPlace(5, 6, rows, size)).toEqual({
      x: 180 + 5 * 59,
      y: 169,
      w: 64,
      h: 64,
    });
  });

  it("switches to the small grid past six: half-size, ten to a row, in the scrolling box", () => {
    const origin = { x: 180, y: 168 };
    expect(spiritPlace(0, 8, rows, size, origin)).toEqual({
      x: 10,
      y: 2,
      w: 32,
      h: 32,
    });
    expect(spiritPlace(7, 8, rows, size, origin)).toEqual({
      x: 10 + 7 * 33,
      y: 2,
      w: 32,
      h: 32,
    });

    expect(spiritPlace(10, 12, rows, size, origin)).toEqual({
      x: 10,
      y: 2 + 34,
      w: 32,
      h: 32,
    });
  });
});

describe("ideasListBox · the politics rows scroll as the game's list does", () => {
  const box = { x: 0, y: 425, w: 550, h: 577 };
  const margin = { top: 14, bottom: 13 };

  it("keeps the three base-game rows still: they fit", () => {
    expect(ideasListBox(box, margin, 3)).toEqual({
      left: 0,
      top: 439,
      width: 550,
      height: 550,
      total: 318,
      scrollable: false,
    });
  });

  it("scrolls a mod's rows once they run past the box", () => {
    const list = ideasListBox(box, margin, 9);
    expect(list.scrollable).toBe(true);
    expect(list.total).toBe(9 * 106);
    expect(list.top + list.height).toBe(425 + 577 - 13);
  });
});
