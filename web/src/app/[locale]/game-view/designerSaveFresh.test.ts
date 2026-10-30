import { readFileSync } from "node:fs";
import { resolve } from "node:path";
import { describe, expect, it } from "vitest";
import { NEW_TEMPLATE } from "./designerEdit";
import { designerKey } from "./DesignerLayer";

const compact = (value: string) => value.replace(/[\s,]+/g, "");
const page = compact(readFileSync(resolve(__dirname, "page.tsx"), "utf8"));
const layer = compact(
  readFileSync(resolve(__dirname, "DesignerLayer.tsx"), "utf8"),
);

function at(text: string, needle: string, from = 0): number {
  const i = text.indexOf(compact(needle), from);
  expect(i, `missing: ${needle}`).toBeGreaterThanOrEqual(0);
  return i;
}

describe("division designer Save", () => {
  it("puts the template as saved where the designer reads it before the draft goes", () => {
    const start = at(page, "const saveDesign = async () => {");
    const end = at(page, "const baseHidden = useMemo(", start);
    const body = page.slice(start, end);
    const seeded = at(
      body,
      "await globalMutate(designerKey(projectId, tag, date ?? null, savedName), fresh, { revalidate: false })",
    );
    expect(seeded).toBeGreaterThan(
      at(
        body,
        "await guiRenderApi.designer(tag, date || undefined, savedName)",
      ),
    );
    expect(at(body, "setDesign(null)")).toBeGreaterThan(seeded);
  });

  it("takes no change while a Save runs", () => {
    expect(page).toContain(
      "onDraft={designSaving ? undefined : (next) => setDesign({ key: designKey, draft: next })}",
    );
    expect(page).toContain("onPicker={designSaving ? undefined : (state) =>");
  });

  it("caches a designer request under the one key the save seeds", () => {
    expect(layer).toContain(
      "tag ? designerKey(projectId, tag, date, template, cells) : null",
    );
  });
});

describe("designerKey", () => {
  it("names a saved template by its name and a new one by the service's '-'", () => {
    expect(designerKey(7, "GER", "1936.1.1", "Infanterie-Division")).toEqual([
      "game-view-designer",
      7,
      "GER",
      "1936.1.1",
      "Infanterie-Division",
      null,
    ]);
    expect(designerKey(7, "GER", null, NEW_TEMPLATE)).toEqual([
      "game-view-designer",
      7,
      "GER",
      null,
      "-",
      "-",
    ]);
    expect(designerKey(7, "GER", null, "X", "r:0:0:infantry")[5]).toBe(
      "r:0:0:infantry",
    );
  });
});
