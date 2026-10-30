import fs from "node:fs";
import path from "node:path";
import { describe, expect, it } from "vitest";

const MESSAGES = path.resolve(
  path.dirname(
    new URL(import.meta.url).pathname.replace(/^\/([A-Za-z]:)/, "$1"),
  ),
  "../../../messages",
);

const PATTERNS: { rx: RegExp; why: string }[] = [
  { rx: /借鉴|竞品/, why: "cites our competitor-analysis notes" },
  { rx: /Borrowed from|\bPer #/i, why: "cites where we copied the idea from" },
  { rx: /ContentMaker|Content Maker/i, why: "names a competitor product" },
  { rx: /\bWave\s*\d/i, why: "internal wave / sprint number" },
  { rx: /(?<![\w#])#\d+/, why: "internal backlog item number" },
  { rx: /single source of tru(th|st)|单 source/i, why: "architecture jargon" },
  { rx: /export pipeline|import pipeline/i, why: "internal pipeline name" },
  { rx: /跨工作台共用/, why: "architecture jargon" },
  { rx: /\bTODO\b|\bFIXME\b/, why: "leftover work marker" },
];

const NON_RENDERED_KEYS = new Set(["__placeholder"]);

interface Finding {
  file: string;
  key: string;
  why: string;
  text: string;
}

function walk(
  node: unknown,
  keyPath: string,
  file: string,
  out: Finding[],
): void {
  if (typeof node === "string") {
    const leaf = keyPath.split(".").pop() ?? "";
    if (NON_RENDERED_KEYS.has(leaf)) return;
    for (const { rx, why } of PATTERNS) {
      if (rx.test(node)) {
        out.push({ file, key: keyPath, why, text: node.slice(0, 120) });
        return;
      }
    }
    return;
  }
  if (Array.isArray(node)) {
    node.forEach((v, i) => walk(v, `${keyPath}[${i}]`, file, out));
    return;
  }
  if (node && typeof node === "object") {
    for (const [k, v] of Object.entries(node)) {
      walk(v, keyPath ? `${keyPath}.${k}` : k, file, out);
    }
  }
}

function scanAll(): Finding[] {
  const out: Finding[] = [];
  for (const locale of fs.readdirSync(MESSAGES)) {
    const dir = path.join(MESSAGES, locale);
    if (!fs.statSync(dir).isDirectory()) continue;
    for (const f of fs.readdirSync(dir)) {
      if (!f.endsWith(".json")) continue;
      const rel = `${locale}/${f}`;
      const raw = fs.readFileSync(path.join(dir, f), "utf8");
      walk(JSON.parse(raw), "", rel, out);
    }
  }
  return out;
}

describe("shipped copy carries no development process", () => {
  it("no message string names a competitor, a ticket number, or our internals", () => {
    const findings = scanAll();
    const report = findings
      .map((f) => `  ${f.file} :: ${f.key}\n    ${f.why}\n    ${f.text}`)
      .join("\n");
    expect(
      findings,
      findings.length
        ? `Copy is leaking development process — write what the feature does ` +
            `for the author instead:\n${report}`
        : "",
    ).toEqual([]);
  });

  it("the check itself still fires (negative control)", () => {
    const out: Finding[] = [];
    walk(
      {
        description:
          "借鉴 #7 HOI4 ContentMaker · 跨工作台共用 · 从 export pipeline 单 source of truth",
        fine: "查看整个 mod 导出后的真实文件树和文件内容。",
      },
      "",
      "synthetic",
      out,
    );
    expect(out).toHaveLength(1);
    expect(out[0].key).toBe("description");
  });

  it("does not fire on legitimate domain copy", () => {
    const out: Finding[] = [];
    walk(
      {
        a: "写入 common/dynamic_modifiers/ —— enable / remove_trigger",
        b: "Primary TAG <tag>{primaryTag}</tag> · {totalFiles} files",
        c: "Served via /api/vanilla/focus-icon/<gfx> (DDS→PNG conversion)",
        d: "英文始终输出(HOI4 引擎 fallback)",
        e: "#RRGGBB",
      },
      "",
      "synthetic",
      out,
    );
    expect(out).toEqual([]);
  });
});
