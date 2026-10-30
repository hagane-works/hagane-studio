import { describe, expect, it } from "vitest";
import { deriveWizardStep } from "./wizardStep";
import type { WizardStepInputs } from "./wizardStep";

const base: Omit<WizardStepInputs, "phase"> = {
  showPreflight: false,
  reImporting: false,
  countryPreparationInFlight: false,
};

describe("deriveWizardStep", () => {
  it("idle / scanning → ①选文件夹", () => {
    expect(deriveWizardStep({ ...base, phase: "idle" })).toBe(1);
    expect(deriveWizardStep({ ...base, phase: "scanning" })).toBe(1);
  });

  it("scanned 且未开预检 → ②选内容", () => {
    expect(deriveWizardStep({ ...base, phase: "scanned" })).toBe(2);
  });

  it("scanned 且开预检 → ③确认(只由步②「导入所选」触发,没有自动流)", () => {
    expect(
      deriveWizardStep({ ...base, phase: "scanned", showPreflight: true }),
    ).toBe(3);
  });

  it("parsing / uploading / done → ④导入", () => {
    for (const phase of ["parsing", "uploading", "done"] as const) {
      expect(deriveWizardStep({ ...base, phase })).toBe(4);
    }
  });

  it("silent-seed 建国中 → ④,即使 phase 仍是 scanned 且预检已关(防回跳步②)", () => {
    expect(
      deriveWizardStep({
        ...base,
        phase: "scanned",
        countryPreparationInFlight: true,
      }),
    ).toBe(4);
  });

  it("加国重导入 → ④,无论 phase 回落到哪(done 后点「扩展加国」的 30-90s 周期)", () => {
    for (const phase of ["scanned", "parsing", "done"] as const) {
      expect(deriveWizardStep({ ...base, phase, reImporting: true })).toBe(4);
    }
  });

  it("取消导入回落:scanned + showPreflight 已清 → ②(调用方负责清 showPreflight)", () => {
    expect(
      deriveWizardStep({ ...base, phase: "scanned", showPreflight: false }),
    ).toBe(2);
  });
});
