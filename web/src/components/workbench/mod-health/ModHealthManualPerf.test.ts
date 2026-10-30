import { describe, expect, it } from "vitest";
import fs from "node:fs";
import path from "node:path";

const source = fs.readFileSync(
  path.join(
    process.cwd(),
    "src/components/workbench/mod-health/ModHealthSection.tsx",
  ),
  "utf8",
);

describe("Mod Health performance scan activation", () => {
  it("keeps performance profiling manual while the cheap timeline lint may auto-run", () => {
    const perf = source.slice(
      source.indexOf("function PerfPanel"),
      source.indexOf("function ErrorLogPanel"),
    );
    expect(perf).not.toContain("void run();");
    expect(perf).toContain("onClick={run}");

    const timeline = source.slice(
      source.indexOf("function TimelinePanel"),
      source.indexOf("function PerfPanel"),
    );
    expect(timeline).toContain("void run();");
  });
});
