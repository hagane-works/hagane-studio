import fs from "node:fs";
import path from "node:path";
import { describe, expect, it } from "vitest";

const ROOT = process.cwd();
const read = (rel: string) => fs.readFileSync(path.resolve(ROOT, rel), "utf8");

const wizard = read(
  "src/components/workbench/import-wizard/UnifiedModImport.tsx",
);
const importApi = read("src/lib/api/import.ts");

describe("import · scan-stats 三态预告仍然停用", () => {
  it("向导没有再发 dry-run 请求", () => {
    expect(wizard).not.toContain("twoStageImportApi.scanStats(");
    expect(wizard).not.toContain("importApi.scanStats(");

    expect(wizard).toContain("setScanStatsLoading(false)");
  });

  it("真实导入走的是分块上传 —— 预告若接回来就是重复整包上传", () => {
    expect(wizard).toContain("twoStageImportApi.scanAndParseBestAvailable(");
    expect(importApi).toContain("twoStageImportApi.stageFiles(");
    expect(importApi).toContain("twoStageImportApi.scanAndParseStaged(");
    expect(wizard).not.toContain("twoStageImportApi.stageFiles(");
    expect(wizard).not.toContain("twoStageImportApi.scanAndParseStaged(");
  });

  it("创建后端解析任务时不生成或发送前端幂等身份", () => {
    expect(importApi).not.toContain("idempotency_key");
    expect(importApi).not.toContain("_parse_full");
  });

  it("旧 Python 整包入口不再暴露给前端", () => {
    expect(importApi).not.toContain("scanStats: async");
    expect(importApi).not.toContain("scanAndParse: async");
    expect(importApi).not.toContain("/import/scan-stats");
    expect(importApi).not.toContain("/import/scan-and-parse");
    expect(importApi).not.toContain("/import/mod-folder");
  });

  it("没有伪装成增量的 scan-stats 变体", () => {
    expect(importApi).not.toContain("scanStatsStaged");
    expect(importApi).not.toContain("scan-stats-staged");
  });
});
