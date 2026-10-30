import { readdirSync, readFileSync, statSync } from "node:fs";
import { join } from "node:path";
import { describe, expect, it } from "vitest";

const SRC = join(__dirname, "..");

function files(dir: string, out: string[] = []): string[] {
  for (const name of readdirSync(dir)) {
    if (name === "node_modules" || name === "generated") continue;
    const path = join(dir, name);
    if (statSync(path).isDirectory()) files(path, out);
    else if (/\.(css|tsx?)$/.test(name) && !name.includes(".test."))
      out.push(path);
  }
  return out;
}

describe("透明混色与模糊在生产构建下的写法", () => {
  const all = files(SRC).map((path) => ({
    path,
    text: readFileSync(path, "utf8"),
  }));

  it("和 transparent 混色不用 oklch", () => {
    const bad = all
      .filter(({ text }) =>
        /color-mix\(in oklch,[^;"]*transparent\)/.test(text),
      )
      .map(({ path }) => path);
    expect(bad).toEqual([]);
  });

  it("CSS 里不手写 -webkit-backdrop-filter", () => {
    const bad = all
      .filter(
        ({ path, text }) =>
          path.endsWith(".css") && text.includes("-webkit-backdrop-filter"),
      )
      .map(({ path }) => path);
    expect(bad).toEqual([]);
  });
});
