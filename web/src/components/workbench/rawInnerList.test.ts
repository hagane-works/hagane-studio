import { readFileSync } from "node:fs";
import { fileURLToPath } from "node:url";
import { dirname, join } from "node:path";
import { describe, expect, it } from "vitest";

import { normalizeInnerList } from "./rawInnerList";

const here = dirname(fileURLToPath(import.meta.url));
const doctrinesSource = readFileSync(
  join(here, "DoctrinesContent.tsx"),
  "utf8",
);

describe("normalizeInnerList · whitespace and newlines are equivalent separators", () => {
  it("joins a newline-separated import into one space-separated line", () => {
    expect(normalizeInnerList("sub_a\nsub_b")).toBe("sub_a sub_b");
    expect(normalizeInnerList("1024\n1025")).toBe("1024 1025");
  });

  it("collapses tabs / CRLF / repeated spaces and trims the ends", () => {
    expect(normalizeInnerList("  sub_a\r\n\tsub_b   sub_c \n")).toBe(
      "sub_a sub_b sub_c",
    );
  });

  it("never merges tokens (the defect) and never alters token text", () => {
    expect(normalizeInnerList("sub_a\nsub_b")).not.toBe("sub_asub_b");

    expect(normalizeInnerList("  mass_assault_deep_battle  ")).toBe(
      "mass_assault_deep_battle",
    );
  });

  it("whitespace-only → '' so the caller can tell 'cleared' from 'has tokens'", () => {
    expect(normalizeInnerList("")).toBe("");
    expect(normalizeInnerList("  \n\t ")).toBe("");
  });

  it("is idempotent (re-saving a normalized value is not a change)", () => {
    const once = normalizeInnerList("sub_a\nsub_b");
    expect(normalizeInnerList(once)).toBe(once);
  });
});

describe("DoctrinesContent · xor binding (UI guard)", () => {
  it("binds xor to a Textarea, never a single-line Input", () => {
    const field = doctrinesSource.slice(
      doctrinesSource.indexOf('t("xorLabel")'),
    );
    const control = field.slice(0, field.indexOf('t("xorHint")'));
    expect(control).toMatch(/<Textarea/);
    expect(control).not.toMatch(/<Input/);
  });

  it("sends xor through normalizeInnerList (not a bare .trim())", () => {
    expect(doctrinesSource).toMatch(
      /const xorList = normalizeInnerList\(draft\.xor\)/,
    );
    expect(doctrinesSource).not.toMatch(/xor: draft\.xor\.trim\(\)/);
  });
});
