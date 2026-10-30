import { describe, expect, it } from "vitest";
import {
  applyPatch,
  blankSketch,
  cloneSketch,
  countryShares,
  decodeDraft,
  decodeRuns,
  encodeDraft,
  encodeRuns,
  fill,
  legendPixels,
  ownerPixels,
  patchOf,
  randomWorld,
  removeCountry,
  sampleWorld,
  SEA,
  SKETCH_H,
  SKETCH_W,
  stamp,
  strokeTo,
  type Brush,
  type Sketch,
} from "./studioSketch";

const brush = (over: Partial<Brush>): Brush => ({
  tool: "paint",
  terrain: 1,
  country: 1,
  radius: 4,
  natural: false,
  ...over,
});

function landShare(s: Sketch): number {
  let land = 0;
  for (const t of s.terrain) if (t !== SEA) land++;
  return land / s.terrain.length;
}

describe("randomWorld", () => {
  it("is the same world for the same options, another for another seed", () => {
    const a = randomWorld(
      { seed: 5, land: 0.3, continents: 3, mountains: 0.5 },
      352,
      128,
    );
    const b = randomWorld(
      { seed: 5, land: 0.3, continents: 3, mountains: 0.5 },
      352,
      128,
    );
    const c = randomWorld(
      { seed: 6, land: 0.3, continents: 3, mountains: 0.5 },
      352,
      128,
    );
    expect(a.terrain).toEqual(b.terrain);
    expect(a.terrain).not.toEqual(c.terrain);
  });

  it("leaves about the asked share of land, with seas at the poles and nothing painted with a country", () => {
    for (const land of [0.15, 0.3, 0.5]) {
      const s = randomWorld(
        { seed: 11, land, continents: 3, mountains: 0.5 },
        352,
        128,
      );
      expect(Math.abs(landShare(s) - land)).toBeLessThan(0.03);
      for (let x = 0; x < s.w; x++) {
        expect(s.terrain[x]).toBe(SEA);
        expect(s.terrain[(s.h - 1) * s.w + x]).toBe(SEA);
      }
      expect(s.owner.every((o) => o === 0)).toBe(true);
    }
  });

  it("gives more mountains when asked for more", () => {
    const count = (m: number) =>
      randomWorld(
        { seed: 3, land: 0.35, continents: 2, mountains: m },
        352,
        128,
      ).terrain.filter((t) => t === 4).length;
    expect(count(1)).toBeGreaterThan(count(0) * 2);
  });

  it("joins its east and west edges, as the game's map wraps", () => {
    const s = randomWorld(
      { seed: 9, land: 0.4, continents: 3, mountains: 0.5 },
      352,
      128,
    );
    let same = 0;
    for (let y = 0; y < s.h; y++)
      if (
        (s.terrain[y * s.w] === SEA) ===
        (s.terrain[y * s.w + s.w - 1] === SEA)
      )
        same++;
    expect(same / s.h).toBeGreaterThan(0.9);
  });
});

describe("brushes", () => {
  it("paints land of the chosen class inside the radius, and the sea clears the country", () => {
    const s = blankSketch(64, 32);
    stamp(s, 20, 16, brush({ terrain: 2, radius: 5 }));
    expect(s.terrain[16 * 64 + 20]).toBe(2);
    expect(s.terrain[16 * 64 + 30]).toBe(SEA);
    s.owner[16 * 64 + 20] = 3;
    stamp(s, 20, 16, brush({ terrain: SEA, radius: 2 }));
    expect(s.terrain[16 * 64 + 20]).toBe(SEA);
    expect(s.owner[16 * 64 + 20]).toBe(0);
  });

  it("changes the terrain on land only with the terrain brush, and paints countries on land only", () => {
    const s = blankSketch(64, 32);
    stamp(s, 10, 16, brush({ terrain: 1, radius: 4 }));
    stamp(s, 12, 16, brush({ tool: "terrain", terrain: 4, radius: 8 }));
    expect(s.terrain[16 * 64 + 10]).toBe(4);
    expect(s.terrain[16 * 64 + 19]).toBe(SEA);
    stamp(s, 12, 16, brush({ tool: "country", country: 2, radius: 8 }));
    expect(s.owner[16 * 64 + 10]).toBe(2);
    expect(s.owner[16 * 64 + 19]).toBe(0);
  });

  it("leaves no gaps along a fast stroke, and a ragged edge stays near the radius", () => {
    const s = blankSketch(128, 32);
    strokeTo(s, [5, 16], [120, 16], brush({ radius: 3, natural: true }));
    for (let x = 6; x < 119; x++) expect(s.terrain[16 * 128 + x]).toBe(1);
    for (let x = 0; x < 128; x++) {
      expect(s.terrain[2 * 128 + x]).toBe(SEA);
      expect(s.terrain[30 * 128 + x]).toBe(SEA);
    }
  });

  it("fills the connected cells of the class clicked, and a country over a whole island", () => {
    const s = blankSketch(40, 20);
    stamp(s, 8, 10, brush({ radius: 4 }));
    stamp(s, 30, 10, brush({ radius: 4 }));
    fill(s, 8, 10, brush({ tool: "fill", terrain: 5 }));
    expect(s.terrain[10 * 40 + 8]).toBe(5);
    expect(s.terrain[10 * 40 + 30]).toBe(1);
    fill(s, 30, 10, brush({ tool: "countryFill", country: 1 }));
    const shares = countryShares(s, 1);
    expect(shares.owned[0]).toBeGreaterThan(0);
    expect(shares.owned[0] + shares.unowned).toBe(shares.land);
    expect(s.owner[10 * 40 + 8]).toBe(0);
    expect(
      fill(s, 0, 0, brush({ tool: "countryFill", country: 1 })),
    ).toBeNull();
  });
});

describe("undo", () => {
  it("takes a stroke back and puts it again", () => {
    const s = sampleWorld(352, 128);
    const before = cloneSketch(s);
    const changed = strokeTo(
      s,
      [10, 10],
      [200, 100],
      brush({ tool: "paint", terrain: 4, radius: 6 }),
    );
    const after = cloneSketch(s);
    const patch = patchOf(before, s, changed);
    expect(patch).not.toBeNull();
    applyPatch(s, patch!, "before");
    expect(s.terrain).toEqual(before.terrain);
    applyPatch(s, patch!, "after");
    expect(s.terrain).toEqual(after.terrain);
  });

  it("records nothing for a stroke that changed nothing", () => {
    const s = blankSketch(32, 16);
    const before = cloneSketch(s);
    const changed = stamp(s, 8, 8, brush({ tool: "country", radius: 4 }));
    expect(patchOf(before, s, changed)).toBeNull();
  });
});

describe("countries", () => {
  it("unpaints a country taken out and moves the ones after it down a slot", () => {
    const s = blankSketch(4, 1);
    s.terrain.fill(1);
    s.owner.set([1, 2, 3, 0]);
    removeCountry(s, 2);
    expect([...s.owner]).toEqual([1, 0, 2, 0]);
  });

  it("sends the legend's colours and the countries' numbers, none on the sea", () => {
    const s = blankSketch(2, 1);
    s.terrain.set([0, 4]);
    s.owner.set([3, 2]);
    expect([...legendPixels(s, { mountain: [1, 2, 3] })]).toEqual([
      40, 80, 170, 255, 1, 2, 3, 255,
    ]);
    expect([...ownerPixels(s)]).toEqual([0, 0, 0, 255, 2, 0, 0, 255]);
  });
});

describe("the draft", () => {
  it("keeps cells as runs, and refuses runs that do not add up", () => {
    const cells = new Uint8Array(70000);
    cells.fill(3, 10, 66000);
    const text = encodeRuns(cells);
    expect(decodeRuns(text, cells.length)).toEqual(cells);
    expect(decodeRuns(text, cells.length + 1)).toBeNull();
    expect(decodeRuns("@@", 4)).toBeNull();
  });

  it("comes back whole, and a damaged or foreign one is dropped", () => {
    const sketch = sampleWorld();
    sketch.owner[100 * SKETCH_W + 300] =
      sketch.terrain[100 * SKETCH_W + 300] === SEA ? 0 : 1;
    const text = encodeDraft({
      sketch,
      countries: ["GER"],
      size: "fine",
      seed: 4,
    });
    const back = decodeDraft(text);
    expect(back?.sketch.terrain).toEqual(sketch.terrain);
    expect(back?.sketch.owner).toEqual(sketch.owner);
    expect(back?.countries).toEqual(["GER"]);
    expect(back?.size).toBe("fine");
    expect(decodeDraft(null)).toBeNull();
    expect(decodeDraft("{")).toBeNull();
    expect(
      decodeDraft(JSON.stringify({ v: 1, w: SKETCH_W, h: SKETCH_H - 1 })),
    ).toBeNull();
  });

  it("drops countries the draft no longer lists", () => {
    const sketch = blankSketch();
    sketch.terrain[0] = 1;
    sketch.owner[0] = 2;
    const back = decodeDraft(
      encodeDraft({ sketch, countries: ["GER"], size: "medium", seed: 1 }),
    );
    expect(back?.sketch.owner[0]).toBe(0);
  });
});
