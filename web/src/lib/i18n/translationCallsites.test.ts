import { readdirSync, readFileSync, statSync } from "node:fs";
import path from "node:path";
import { describe, expect, it } from "vitest";

const ROOT = process.cwd();
const MESSAGES_EN = path.join(ROOT, "messages", "en");

function walkSources(dir: string, out: string[] = []): string[] {
  for (const entry of readdirSync(dir)) {
    const p = path.join(dir, entry);
    if (statSync(p).isDirectory()) {
      if (entry !== "node_modules") walkSources(p, out);
    } else if (/\.tsx?$/.test(entry) && !/\.test\.tsx?$/.test(entry)) {
      out.push(p);
    }
  }
  return out;
}

function flattenKeys(obj: unknown, prefix: string, out: Set<string>): void {
  if (obj && typeof obj === "object" && !Array.isArray(obj)) {
    for (const [k, v] of Object.entries(obj)) {
      flattenKeys(v, prefix ? `${prefix}.${k}` : k, out);
    }
  } else {
    out.add(prefix);
  }
}

function loadBundles(): Record<string, Set<string>> {
  const bundles: Record<string, Set<string>> = {};
  for (const f of readdirSync(MESSAGES_EN).filter((f) => f.endsWith(".json"))) {
    const set = new Set<string>();
    flattenKeys(
      JSON.parse(readFileSync(path.join(MESSAGES_EN, f), "utf8")),
      "",
      set,
    );
    bundles[f.replace(/\.json$/, "")] = set;
  }
  return bundles;
}

function stripComments(src: string): string {
  return src.replace(/^\s*\/\/.*$/gm, "").replace(/\/\*[\s\S]*?\*\//g, "");
}

interface Violation {
  file: string;
  namespaces: string[];
  key: string;
}

function findViolations(bundles: Record<string, Set<string>>): Violation[] {
  const violations: Violation[] = [];
  for (const file of walkSources(path.join(ROOT, "src"))) {
    const src = stripComments(readFileSync(file, "utf8"));

    const varNs = new Map<string, Set<string>>();
    for (const m of src.matchAll(
      /(?:const|let)\s+(\w+)\s*=\s*useTranslations\(\s*["'`]([\w.]+)["'`]\s*\)/g,
    )) {
      if (!varNs.has(m[1])) varNs.set(m[1], new Set());
      varNs.get(m[1])!.add(m[2]);
    }
    if (varNs.size === 0) continue;

    for (const [variable, namespaces] of varNs) {
      const callRe = new RegExp(
        String.raw`\b${variable}(?:\.rich|\.raw|\.markup)?\(\s*["']([\w.\-]+)["']`,
        "g",
      );
      for (const m of src.matchAll(callRe)) {
        const key = m[1];
        const hit = [...namespaces].some((ns) => {
          const [bundleName, ...nested] = ns.split(".");
          const bundle = bundles[bundleName];
          if (!bundle) return false;
          return bundle.has([...nested, key].join("."));
        });
        if (!hit) {
          violations.push({
            file: path.relative(ROOT, file),
            namespaces: [...namespaces],
            key,
          });
        }
      }
    }
  }
  return violations;
}

describe("i18n · 调用点与 message key 的存在性", () => {
  it('每个静态 t("…") 调用点都能在 en bundle 里解析到', () => {
    const violations = findViolations(loadBundles());
    const report = violations
      .map((v) => `  ${v.file}  ${v.namespaces.join("|")}.${v.key}`)
      .join("\n");
    expect(
      violations,
      `代码里 t() 用到但 messages 里不存在的 key:\n${report}`,
    ).toEqual([]);
  });

  it("守卫本身没有失效 —— 确实扫到了大量调用点", () => {
    const bundles = loadBundles();
    let seen = 0;
    for (const file of walkSources(path.join(ROOT, "src"))) {
      const src = stripComments(readFileSync(file, "utf8"));
      for (const m of src.matchAll(
        /(?:const|let)\s+(\w+)\s*=\s*useTranslations\(\s*["'`]([\w.]+)["'`]\s*\)/g,
      )) {
        const callRe = new RegExp(
          String.raw`\b${m[1]}(?:\.rich|\.raw)?\(\s*["']`,
          "g",
        );
        seen += [...src.matchAll(callRe)].length;
      }
    }
    expect(Object.keys(bundles).length).toBeGreaterThan(50);
    expect(seen).toBeGreaterThan(500);
  });
});
