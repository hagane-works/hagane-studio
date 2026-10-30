import { describe, it, expect } from "vitest";
import { readFileSync } from "node:fs";
import { join } from "node:path";

const canvas = readFileSync(join(__dirname, "GuiCanvas.tsx"), "utf8");
const parser = readFileSync(
  join(__dirname, "../../../../../internal/hoi4/guirender/parser.go"),
  "utf8",
);

describe("a dropdown's expanded window is parsed but not painted", () => {
  it("is parsed, because that is where a dropdown's real geometry lives", () => {
    expect(parser).toContain('"dropdownboxtype": true, "expandedwindow": true');
    expect(parser).toContain('"show_position": true');
  });

  it("is hidden by the canvas, because the engine only shows it when open", () => {
    expect(canvas).toContain(
      'if (node.type === "expandedwindow") return true;',
    );

    expect(canvas).toMatch(
      /function runtimeHidden[\s\S]{0,1200}expandedwindow/,
    );
  });
});
