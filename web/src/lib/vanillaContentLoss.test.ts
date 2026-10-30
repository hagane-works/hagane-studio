import { describe, expect, it } from "vitest";
import {
  hasVanillaContentLoss,
  summarizeVanillaContentLoss,
} from "@/lib/vanillaContentLoss";
import { STATE_INDEX_FIXTURE as INDEX } from "@/lib/__fixtures__/vanillaStateIndex";
import type { VanillaStateContent } from "@/lib/vanillaStateIndex";

function pickState(pred: (c: VanillaStateContent) => boolean): number {
  const hit = Object.keys(INDEX.content)
    .map(Number)
    .find((id) => pred(INDEX.content[id]));
  if (hit === undefined) throw new Error("索引里找不到符合条件的州");
  return hit;
}

const withNaval = pickState((c) => (c.navalBases ?? 0) > 0);
const withLandmark = pickState((c) => (c.landmarks ?? []).length > 0);

describe("summarizeVanillaContentLoss", () => {
  it("种子来的州(raw_extras 空)按索引累加", () => {
    const loss = summarizeVanillaContentLoss(
      [{ state_id: withNaval, raw_extras: {} }],
      INDEX,
    );
    expect(loss.states).toBe(1);
    expect(loss.navalBases).toBe(INDEX.content[withNaval].navalBases);
  });

  it("🔴 raw_extras 非空的州跳过 —— 宁可少报不可多报", () => {
    const loss = summarizeVanillaContentLoss(
      [{ state_id: withNaval, raw_extras: { buildings: "…" } }],
      INDEX,
    );
    expect(loss.states).toBe(0);
    expect(hasVanillaContentLoss(loss)).toBe(false);
  });

  it("不在原版索引里的州不算(作者自建的州没有「原版内容」可丢)", () => {
    const loss = summarizeVanillaContentLoss(
      [{ state_id: 999999, raw_extras: {} }],
      INDEX,
    );
    expect(loss.states).toBe(0);
  });

  it("🔴 重复 state_id 只算一次 —— 否则同一个州的海军基地会被数两遍", () => {
    const one = summarizeVanillaContentLoss(
      [{ state_id: withNaval, raw_extras: {} }],
      INDEX,
    );
    const twice = summarizeVanillaContentLoss(
      [
        { state_id: withNaval, raw_extras: {} },
        { state_id: withNaval, raw_extras: {} },
      ],
      INDEX,
    );
    expect(twice.states).toBe(1);
    expect(twice.navalBases).toBe(one.navalBases);
  });

  it("地标按名字去重,并给出可读的 key 列表", () => {
    const loss = summarizeVanillaContentLoss(
      [{ state_id: withLandmark, raw_extras: {} }],
      INDEX,
    );
    expect(loss.landmarks).toBeGreaterThan(0);

    expect(loss.landmarkKeys.length).toBeLessThanOrEqual(loss.landmarks);
    expect(loss.landmarkKeys.every((k) => k.startsWith("landmark_"))).toBe(
      true,
    );

    expect([...loss.landmarkKeys].sort()).toEqual(loss.landmarkKeys);
  });

  it("空 / null / 脏输入给全零,不抛", () => {
    for (const input of [null, undefined, [], [null as never]]) {
      const loss = summarizeVanillaContentLoss(input, INDEX);
      expect(loss.states).toBe(0);
      expect(hasVanillaContentLoss(loss)).toBe(false);
    }
  });

  it("🔴 全项目规模:整个原版都是种子来的时候,数字要对得上索引总量", () => {
    const all = Object.keys(INDEX.content).map((id) => ({
      state_id: Number(id),
      raw_extras: {},
    }));
    const loss = summarizeVanillaContentLoss(all, INDEX);
    const expectNaval = Object.values(INDEX.content).reduce(
      (n, c) => n + (c.navalBases ?? 0),
      0,
    );
    expect(loss.states).toBe(Object.keys(INDEX.content).length);
    expect(loss.navalBases).toBe(expectNaval);

    expect(loss.landmarks).toBe(3);
    expect(loss.landmarkKeys.length).toBe(3);
    expect(loss.resistance).toBe(5);
    expect(loss.otherContent).toBe(1);
  });

  it("🔴 379 / 987 / 1057 逐个在索引里,且只靠兜底/新类别进来", () => {
    const nevada = INDEX.content[379];
    expect(nevada).toBeDefined();
    expect(nevada.otherContent).toBe(1);
    expect(nevada.dams ?? 0).toBe(0);

    for (const id of [987, 1057]) {
      const content = INDEX.content[id];
      expect(content).toBeDefined();
      expect(content.resistance ?? 0).toBeGreaterThan(0);
    }

    for (const id of [379, 987, 1057]) {
      const c = INDEX.content[id];
      const namedOnly =
        (c.navalBases ?? 0) +
        (c.bunkers ?? 0) +
        (c.coastalBunkers ?? 0) +
        (c.dams ?? 0) +
        (c.landmarks ?? []).length +
        (c.datedBlocks ?? 0) +
        (c.hasDlc ?? 0) +
        (c.forceLink ? 1 : 0);
      expect(namedOnly).toBe(0);

      const loss = summarizeVanillaContentLoss(
        [{ state_id: id, raw_extras: {} }],
        INDEX,
      );
      expect(loss.states).toBe(1);
      expect(hasVanillaContentLoss(loss)).toBe(true);
    }
  });

  it("otherContent 只是个计数,索引里不带任何名字", () => {
    const loss = summarizeVanillaContentLoss(
      [{ state_id: 379, raw_extras: {} }],
      INDEX,
    );
    expect(loss.otherContent).toBe(1);

    expect(loss.landmarkKeys).toEqual([]);
    expect(Object.keys(INDEX.content[379])).toEqual(["otherContent"]);
  });
});
