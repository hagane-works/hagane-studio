





import { readFileSync, readdirSync, writeFileSync } from "node:fs";
import { join } from "node:path";
import { build } from "vite";

const NOTICES = "../THIRD_PARTY_NOTICES.md";
const START = "<!-- npm packages: written by web/scripts/third-party-notices.mjs -->";
const END = "<!-- end of npm packages -->";


const UNSEEN = ["tailwindcss", "tw-animate-css", "shadcn", "monaco-editor"];

const dirs = new Set(UNSEEN.map((name) => join("node_modules", name)));
await build({
  logLevel: "error",
  build: { write: false },
  plugins: [
    {
      name: "list-packages",
      generateBundle() {
        for (const id of this.getModuleIds()) {
          const path = id.replace(/^\0/, "").split("?")[0].replaceAll("\\", "/");
          const at = path.lastIndexOf("/node_modules/");
          if (at < 0) continue;
          const [first, second] = path.slice(at + "/node_modules/".length).split("/");
          dirs.add(path.slice(0, at) + "/node_modules/" + (first.startsWith("@") ? `${first}/${second}` : first));
        }
      },
    },
  ],
});

const packages = new Map();
for (const dir of dirs) {
  const { name, version, license } = JSON.parse(readFileSync(join(dir, "package.json"), "utf8"));
  const texts = readdirSync(dir)
    .filter((file) => /^(licen[cs]e|copying|notice|thirdpartynotices)/i.test(file))
    .sort()
    .map((file) => readFileSync(join(dir, file), "utf8").replace(/\r\n?/g, "\n").trim());
  packages.set(`${name}@${version}`, { name, version, license: license?.type ?? license ?? "see the package", texts });
}
const sorted = [...packages.values()].sort((a, b) => a.name.localeCompare(b.name) || a.version.localeCompare(b.version));

const lines = [START, "", "| Package | Version | Licence |", "| --- | --- | --- |"];
for (const p of sorted) lines.push(`| ${p.name} | ${p.version} | ${p.license} |`);
for (const p of sorted) {
  lines.push("", `<details><summary>${p.name} ${p.version} (${p.license})</summary>`, "");
  if (p.texts.length === 0) lines.push(`The package has no licence file; its package.json gives ${p.license}.`);
  for (const text of p.texts) lines.push("````text", text, "````");
  lines.push("", "</details>");
}
lines.push("", END);

const notices = readFileSync(NOTICES, "utf8");
const start = notices.indexOf(START);
const end = notices.indexOf(END);
if (start < 0 || end < start) throw new Error(`${NOTICES} has no npm packages section`);
writeFileSync(NOTICES, notices.slice(0, start) + lines.join("\n") + notices.slice(end + END.length));
console.log(`${sorted.length} packages written to ${NOTICES}`);
