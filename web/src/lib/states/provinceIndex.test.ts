import { describe, expect, it } from "vitest";

import {
  buildProvinceIndex,
  detectCollisions,
  normalizeProvinces,
  planProvinceMove,
  type IndexableState,
} from "./provinceIndex";

function st(
  id: number,
  stateId: number,
  provinces: number[],
  name = `STATE_${stateId}`,
): IndexableState {
  return { id, state_id: stateId, name, provinces };
}

describe("缺口 C · province 反查索引", () => {
  it("同一个 state 里重复列同一个 province 不算冲突(与后端 set() 同判)", () => {
    const index = buildProvinceIndex([st(1, 42, [100, 100, 101])]);
    expect(index.get(100)).toHaveLength(1);
    expect(detectCollisions(index)).toEqual([]);
  });

  it("只有跨 state 才算冲突,owners 按 state_id 升序", () => {
    const collisions = detectCollisions(
      buildProvinceIndex([
        st(1, 43, [100]),
        st(2, 42, [100]),
        st(3, 44, [200]),
      ]),
    );
    expect(collisions).toHaveLength(1);
    expect(collisions[0].provinceId).toBe(100);
    expect(collisions[0].owners.map((o) => o.stateId)).toEqual([42, 43]);
  });

  it("normalizeProvinces 去重但保留作者写的顺序", () => {
    expect(normalizeProvinces([300, 100, 300, 200])).toEqual([300, 100, 200]);
  });
});

describe("缺口 C · MOVE 计划(guides/state.md:153 的 nudger 语义)", () => {
  it("把 province 交给另一个 state = 自动从原 state 摘掉,不是报错", () => {
    const states = [st(1, 42, [100, 101]), st(2, 43, [200])];
    const plan = planProvinceMove(states, 2, [200, 100]);

    expect(plan).not.toBeNull();
    expect(plan!.movedProvinceIds).toEqual([100]);
    expect(plan!.donors).toHaveLength(1);
    expect(plan!.donors[0]).toMatchObject({
      rowId: 1,
      stateId: 42,
      prevProvinces: [100, 101],
      nextProvinces: [101],
      taken: [100],
    });

    expect(plan!.targetPrevProvinces).toEqual([200]);
    expect(plan!.targetNextProvinces).toEqual([200, 100]);
  });

  it("只改自己已有的 province 时不产生任何 donor(不触发 MOVE)", () => {
    const states = [st(1, 42, [100, 101]), st(2, 43, [200])];
    const plan = planProvinceMove(states, 1, [100]);
    expect(plan!.donors).toEqual([]);
    expect(plan!.movedProvinceIds).toEqual([]);
  });

  it("导入自带的双重归属:一次保存把它从全部原 state 摘掉(能自动消解就不报错)", () => {
    const states = [st(1, 42, [100]), st(2, 43, [100]), st(3, 44, [])];
    const plan = planProvinceMove(states, 3, [100]);

    expect(plan!.donors.map((d) => d.stateId)).toEqual([42, 43]);
    expect(plan!.donors.every((d) => d.nextProvinces.length === 0)).toBe(true);

    const after: IndexableState[] = [
      st(1, 42, plan!.donors[0].nextProvinces),
      st(2, 43, plan!.donors[1].nextProvinces),
      st(3, 44, plan!.targetNextProvinces),
    ];
    expect(detectCollisions(buildProvinceIndex(after))).toEqual([]);
  });

  it("target 不在 store 里时返回 null", () => {
    expect(planProvinceMove([st(1, 42, [100])], 999, [100])).toBeNull();
  });
});
