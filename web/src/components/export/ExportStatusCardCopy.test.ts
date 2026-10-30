import { readFileSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";
import { describe, expect, it } from "vitest";

const here = dirname(fileURLToPath(import.meta.url));
const frontendRoot = join(here, "..", "..", "..");
const read = (path: string) => readFileSync(join(frontendRoot, path), "utf8");
const statusCard = read("src/components/export/ExportStatusCard.tsx");

interface ExportUi {
  ui: {
    status: Record<string, string>;
    attention: Record<string, string>;
  };
}
const zh = (JSON.parse(read("messages/zh/export.json")) as ExportUi).ui;
const en = (JSON.parse(read("messages/en/export.json")) as ExportUi).ui;

describe("A9 · no unconditional 'impact + fix entry' promise", () => {
  it("the attention verdict no longer claims every item has an entry", () => {
    expect(zh.status.copyAttn).not.toContain("修复入口");
    expect(zh.status.copyAttn).not.toContain("已附");
    expect(en.status.copyAttn).not.toMatch(/fix entry/i);
    expect(en.status.copyAttn).not.toMatch(/each comes with/i);
  });

  it("states the repairable count only when the page measured one", () => {
    expect(statusCard).toContain("repairableCount");
    expect(statusCard).toContain("repairableCount !== null");
    expect(statusCard).toContain('t("ui.status.copyAttnRepairable"');
    expect(zh.status.copyAttnRepairable).toContain("{count}");
    expect(en.status.copyAttnRepairable).toContain("count");
  });
});

describe("A9 · the clean verdict does not claim completeness", () => {
  it("drops 'all checks passed' / 'no problems found'", () => {
    expect(zh.status.titleClean).not.toContain("全部通过");
    expect(zh.status.copyClean).not.toContain("未发现问题");
    expect(zh.attention.emptyBody).not.toContain("全部检查通过");
    expect(en.status.titleClean).not.toMatch(/all checks passed/i);
    expect(en.status.copyClean).not.toMatch(/no problems found/i);
    expect(en.attention.emptyBody).not.toMatch(/all checks passed/i);
  });

  it("points at in-game verification for the answer the check cannot give", () => {
    expect(zh.status.copyClean).toContain("游戏内验证");
    expect(zh.status.copyCleanAuto).toContain("游戏内验证");
    expect(en.status.copyClean).toMatch(/verify in-game/i);
    expect(en.status.copyCleanAuto).toMatch(/verify in-game/i);
  });
});

describe("A9 · no phantom export gate", () => {
  it("stops advertising that nothing blocked the export", () => {
    for (const key of ["copyOk", "copyOkAuto"] as const) {
      expect(zh.status[key]).not.toContain("阻止导出");
      expect(en.status[key]).not.toMatch(/nothing blocks/i);
    }
  });

  it("keeps the fact the author can act on: download is always available", () => {
    expect(zh.status.copyOk).toContain("随时可以下载");
    expect(zh.status.copyAttn).toContain("随时可以下载");
    expect(en.status.copyOk).toMatch(/download at any time/i);
    expect(en.status.copyAttn).toMatch(/download at any time/i);
  });
});
