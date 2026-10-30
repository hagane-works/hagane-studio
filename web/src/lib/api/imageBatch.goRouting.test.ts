import { describe, expect, it } from "vitest";
import { readFileSync } from "node:fs";
import { resolve } from "node:path";

const page = readFileSync(
  resolve(process.cwd(), "src/app/[locale]/project/[id]/image-batch/page.tsx"),
  "utf8",
);

describe("image batch capability ownership", () => {
  it("routes original-size and resized DDS conversion directly to Go without an env flag or fallback", () => {
    expect(page).toContain('fetch("/api/go/image-batch/convert-dds"');
    expect(page).not.toMatch(/GO_IMAGE_BATCH|NEXT_PUBLIC.*IMAGE_BATCH/);
    expect(page).not.toMatch(
      /catch[\s\S]{0,300}\/api\/image-batch\/convert-dds/,
    );
    expect(page).not.toContain('"/api/image-batch/convert-dds"');
  });

  it("keeps placeholder and vanilla disk reads on fixed Go routes", () => {
    expect(page).toContain('fetch("/api/go/image-batch/generate-placeholder"');
    expect(page).toContain('fetch("/api/go/image-batch/vanilla-root-status"');
    expect(page).toContain("`/api/go/image-batch/vanilla-extract/${type}`");
  });
});
