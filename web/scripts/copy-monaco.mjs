





import { cpSync, mkdirSync, existsSync } from "node:fs";
import { dirname } from "node:path";

const SRC = "node_modules/monaco-editor/min/vs";
const DST = "public/monaco-editor/min/vs";

if (!existsSync(SRC)) {
  console.error(`[copy-monaco] source not found: ${SRC} · skipping (run after npm install)`);
  process.exit(0);
}

mkdirSync(dirname(DST), { recursive: true });
cpSync(SRC, DST, { recursive: true });
console.log(`[copy-monaco] ${SRC} -> ${DST}`);
