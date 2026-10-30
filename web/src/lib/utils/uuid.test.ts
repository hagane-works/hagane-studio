import { describe, expect, it, vi } from "vitest";

import { createUUID } from "./uuid";

const UUID_V4_RE =
  /^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/;

describe("createUUID", () => {
  it("uses native randomUUID when the secure-context API is available", () => {
    const randomUUID = vi.fn(() => "11111111-1111-4111-8111-111111111111");

    expect(createUUID({ randomUUID })).toBe(
      "11111111-1111-4111-8111-111111111111",
    );
    expect(randomUUID).toHaveBeenCalledOnce();
  });

  it("creates an RFC 4122 v4 UUID from getRandomValues on an HTTP origin", () => {
    const getRandomValues = vi.fn((values: Uint8Array) => {
      values.forEach((_, index) => {
        values[index] = index;
      });
      return values;
    });

    const result = createUUID({ getRandomValues });

    expect(result).toMatch(UUID_V4_RE);
    expect(result).toBe("00010203-0405-4607-8809-0a0b0c0d0e0f");
    expect(getRandomValues).toHaveBeenCalledOnce();
  });

  it("falls back when an exposed randomUUID method rejects the HTTP context", () => {
    const getRandomValues = vi.fn((values: Uint8Array) => values.fill(0x2a));

    expect(
      createUUID({
        randomUUID: () => {
          throw new TypeError("randomUUID is unavailable");
        },
        getRandomValues,
      }),
    ).toMatch(UUID_V4_RE);
    expect(getRandomValues).toHaveBeenCalledOnce();
  });
});
