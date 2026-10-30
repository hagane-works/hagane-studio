

import { build } from "esbuild";
import { readdirSync, statSync, readFileSync, rmSync, existsSync } from "node:fs";
import { join, relative, resolve, dirname } from "node:path";

const root = process.cwd();
const src = join(root, "src");
const args = process.argv.slice(2);
const result = await build({
  entryPoints: [join(src, "main.tsx"), join(src, "components/workbench/import-wizard/scan.worker.ts")],
  bundle: true, write: false, metafile: true, logLevel: "silent", outdir: "out",
  platform: "browser", format: "esm", jsx: "automatic",
  packages: "external",
  alias: { "@": src },
  loader: { ".png": "empty", ".svg": "empty", ".css": "empty", ".module.css": "empty", ".woff2": "empty", ".json": "json" },
  plugins: [{ name: "glob", setup(b) { b.onResolve({ filter: /^@fontsource/ }, () => ({ external: true })); } }],
}).catch((e) => { console.error(e.errors?.slice(0, 10)); process.exit(1); });

const inputs = result.metafile.inputs;
const reached = new Set(Object.keys(inputs).map((p) => resolve(root, p)));

const importers = new Map();
for (const [file, info] of Object.entries(inputs)) {
  for (const imp of info.imports) {
    const target = resolve(root, imp.path);
    if (!importers.has(target)) importers.set(target, []);
    importers.get(target).push(resolve(root, file));
  }
}

function* walk(dir) {
  for (const name of readdirSync(dir)) {
    const full = join(dir, name);
    if (statSync(full).isDirectory()) yield* walk(full); else yield full;
  }
}
const all = [...walk(src)].filter((f) => /\.(ts|tsx|mts|css|json|mjs|js)$/.test(f));
const isTest = (f) => /\.test\.(ts|tsx)$/.test(f) || f.includes(`${"\\"}__tests__${"\\"}`) || f.includes("/__tests__/") || f.includes("__fixtures__");
const unreached = all.filter((f) => !reached.has(f) && !isTest(f));


const importRE = /(?:import|from|mock\(|importActual(?:<[^>]*>)?\()\s*\(?\s*["']([^"']+)["']/g;
function resolveLocal(spec, from) {
  let base;
  if (spec.startsWith("@/")) base = join(src, spec.slice(2));
  else if (spec.startsWith(".")) base = resolve(dirname(from), spec);
  else return null;
  for (const ext of ["", ".ts", ".tsx", "/index.ts", "/index.tsx", ".json"]) {
    if (existsSync(base + ext) && statSync(base + ext).isFile()) return base + ext;
  }
  return base;
}
const tests = all.filter(isTest);
const deadTests = tests.filter((t) => {
  const text = readFileSync(t, "utf8");
  for (const m of text.matchAll(importRE)) {
    const target = resolveLocal(m[1], t);
    if (target && !reached.has(target) && !isTest(target)) return true;
  }
  return false;
});

const edgesIndex = args.indexOf("--edges");
if (edgesIndex >= 0) {
  const needle = args[edgesIndex + 1];
  const lines = new Set();
  for (const [file, info] of Object.entries(inputs)) {
    if (file.includes(needle)) continue;
    for (const imp of info.imports) {
      if (imp.path.includes(needle)) lines.add(`${file}  ->  ${imp.path}`);
    }
  }
  console.log([...lines].sort().join("\n"));
  process.exit(0);
}
const whyIndex = args.indexOf("--why");
if (whyIndex >= 0) {
  const needle = args[whyIndex + 1];
  for (const file of reached) {
    if (!file.includes(needle)) continue;
    console.log(relative(src, file), "<-", (importers.get(file) || []).map((f) => relative(src, f)).join(", "));
  }
  process.exit(0);
}
console.log(`reached ${reached.size} files; unreached sources ${unreached.length}; tests ${tests.length}, dead tests ${deadTests.length}`);
if (args.includes("--list")) for (const f of unreached) console.log("  -", relative(src, f));
if (args.includes("--prune")) {
  for (const f of [...unreached, ...deadTests]) rmSync(f);
  console.log("pruned", unreached.length + deadTests.length, "files");
}
