import { readFileSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";
import { afterEach, describe, expect, it, vi } from "vitest";
import { rawFilesApi, type RawFileRow } from "@/lib/api/scenario";

const here = dirname(fileURLToPath(import.meta.url));
const frontendRoot = join(here, "..", "..");
const read = (path: string) => readFileSync(join(frontendRoot, path), "utf8");

const row: RawFileRow = {
  id: 1,
  project_id: 7,
  dest_path: "common/on_actions/exact_source.txt",
  content: "",
  is_binary: false,
  generate_replace_path: false,
  raw_kind: "empty_shadow",
  target_layer: "vanilla",
  target_sha256: "a".repeat(64),
  note: "verified source",
  created_at: "2026-08-03T00:00:00Z",
  updated_at: "2026-08-03T00:00:00Z",
};

afterEach(() => {
  vi.unstubAllGlobals();
});

describe("A5 empty-file shadow contract", () => {
  it("sends an explicit zero-byte single-file intent without replace_path", async () => {
    const fetchMock = vi.fn(
      async (input: RequestInfo | URL, init?: RequestInit) => {
        void input;
        void init;
        return new Response(JSON.stringify(row), {
          status: 201,
          headers: { "Content-Type": "application/json" },
        });
      },
    );
    vi.stubGlobal("fetch", fetchMock);

    await rawFilesApi.createEmptyShadow(7, {
      dest_path: row.dest_path,
      target_layer: row.target_layer!,
      target_sha256: row.target_sha256!,
      note: row.note,
    });

    const [, request] = fetchMock.mock.calls[0];
    expect(JSON.parse(String(request?.body))).toEqual({
      dest_path: row.dest_path,
      target_layer: "vanilla",
      target_sha256: "a".repeat(64),
      note: "verified source",
      content: "",
      generate_replace_path: false,
      raw_kind: "empty_shadow",
    });
  });

  it("keeps the advanced action separate and renders policy rejection inline", () => {
    const manager = read(
      "src/components/workbench/RawOverrideFilesManager.tsx",
    );
    expect(manager).toContain("rawFilesApi.createEmptyShadow");
    expect(manager).toContain('file.raw_kind === "empty_shadow"');
    expect(manager).toContain('role="alert"');
    expect(manager).toContain('t("rawFileEmptyShadowEvidenceWarning")');
    expect(manager).toContain('t("rawFileEmptyShadowPolicyCode"');
  });

  it("ships the same A5 message keys in all eight app locales", () => {
    const locales = ["en", "zh", "ru", "de", "es", "fr", "ja", "ko"];
    const keys = [
      "rawFileEmptyShadowAdd",
      "rawFileEmptyShadowTitle",
      "rawFileEmptyShadowDesc",
      "rawFileEmptyShadowEvidenceWarning",
      "rawFileEmptyShadowLayerLabel",
      "rawFileEmptyShadowHashLabel",
      "rawFileEmptyShadowEvidenceRequired",
      "rawFileEmptyShadowSubmit",
      "rawFileEmptyShadowRejected",
      "rawFileEmptyShadowPolicyCode",
      "rawFileEmptyShadowBadge",
    ];
    for (const locale of locales) {
      const catalog = JSON.parse(
        read(`messages/${locale}/export.json`),
      ) as Record<string, string>;
      for (const key of keys) {
        expect(catalog[key], `${locale}.${key}`).toBeTruthy();
      }
      expect(catalog.rawFileEmptyShadowRejected).toContain("{message}");
      expect(catalog.rawFileEmptyShadowPolicyCode).toContain("{code}");
    }
  });
});
