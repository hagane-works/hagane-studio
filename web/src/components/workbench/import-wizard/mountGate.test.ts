import { describe, expect, it } from "vitest";
import { shouldMountModImport } from "./mountGate";

describe("shouldMountModImport", () => {
  it("载入模式 · 0 国家 · latch 已丢失 → 仍然挂载(死锁修复的核心断言)", () => {
    expect(
      shouldMountModImport({
        countryCount: 0,
        hadPendingFolder: false,
        isLoadExistingMod: true,
      }),
    ).toBe(true);
  });

  it("载入模式 · 首次 mount 带 pending handle → 挂载(原 FOLDERFIRST 行为不回退)", () => {
    expect(
      shouldMountModImport({
        countryCount: 0,
        hadPendingFolder: true,
        isLoadExistingMod: true,
      }),
    ).toBe(true);
  });

  it("有国家 → 挂载,与项目来源无关", () => {
    for (const isLoadExistingMod of [true, false]) {
      expect(
        shouldMountModImport({
          countryCount: 3,
          hadPendingFolder: false,
          isLoadExistingMod,
        }),
      ).toBe(true);
    }
  });

  it("非载入模式 · 0 国家 · 无 pending handle → 不挂载(N=0 走「加国家」入口)", () => {
    expect(
      shouldMountModImport({
        countryCount: 0,
        hadPendingFolder: false,
        isLoadExistingMod: false,
      }),
    ).toBe(false);
  });

  it("非载入模式 · 0 国家 · 带 pending handle → 挂载(carry-in 自动消费需要它在场)", () => {
    expect(
      shouldMountModImport({
        countryCount: 0,
        hadPendingFolder: true,
        isLoadExistingMod: false,
      }),
    ).toBe(true);
  });
});
