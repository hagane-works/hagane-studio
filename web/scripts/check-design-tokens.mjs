









import { readdirSync, readFileSync, statSync } from "node:fs";
import { join, sep } from "node:path";

const ROOT = "src";

const EXCLUDED_DIR = join("src", "components", "ui");



const EXCLUDED_FILES = new Set([
  join("src", "components", "workbench", "DecisionInGamePreview.tsx"),
  join("src", "components", "workbench", "EventInGamePreview.tsx"),
]);
const SCAN_EXT = new Set([".ts", ".tsx", ".js", ".jsx", ".mjs", ".cjs"]);

const RULES = [
  { name: "text-[Npx] arbitrary font size", re: /text-\[\d+px\]/ },
  { name: "heavy font weight (font-bold|extrabold|black)", re: /\bfont-(bold|extrabold|black)\b/ },
];


function walk(dir, out) {
  for (const entry of readdirSync(dir)) {
    const full = join(dir, entry);

    if (full === EXCLUDED_DIR || full.startsWith(EXCLUDED_DIR + sep)) continue;
    const st = statSync(full);
    if (st.isDirectory()) {
      walk(full, out);
    } else if (SCAN_EXT.has(full.slice(full.lastIndexOf("."))) && !EXCLUDED_FILES.has(full)) {
      out.push(full);
    }
  }
}

const files = [];
walk(ROOT, files);

const violations = [];
for (const file of files) {
  const lines = readFileSync(file, "utf8").split(/\r?\n/);
  lines.forEach((line, i) => {




    const trimmed = line.trimStart();
    if (trimmed.startsWith("*") || trimmed.startsWith("//") || trimmed.startsWith("/*")) return;
    for (const rule of RULES) {
      const m = line.match(rule.re);
      if (m) {
        violations.push(`${file}:${i + 1}: ${rule.name} -> "${m[0]}"`);
      }
    }
  });
}

if (violations.length > 0) {
  console.error("[FAIL] design-token check found " + violations.length + " violation(s):");
  for (const v of violations) console.error("  " + v);
  console.error(
    "\nFix: text-[Npx] -> type scale class (text-micro/text-xs/text-sm/...), " +
      "font-bold -> font-semibold. See DESIGN.md sections 3.2 / 3.3."
  );
  process.exit(1);
}

console.log("[OK] design-token check passed (" + files.length + " files scanned)");
process.exit(0);
