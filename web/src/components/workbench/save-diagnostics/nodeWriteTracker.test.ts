import { describe, expect, it } from "vitest";
import { createNodeWriteTracker } from "./nodeWriteTracker";

type NodeSnapshot = Readonly<{ name: string; x: number }>;
const base: NodeSnapshot = Object.freeze({ name: "BASE", x: 0 });
const first: NodeSnapshot = Object.freeze({ name: "FIRST", x: 1 });
const second: NodeSnapshot = Object.freeze({ name: "SECOND", x: 2 });
const key = "1:10:100";

describe("canonical node write tracking", () => {
  for (const order of ["older-first", "newer-first"] as const) {
    it(`both writes fail (${order}): restore BASE, never a failed optimistic snapshot`, () => {
      const tracker = createNodeWriteTracker<typeof base>();
      const one = tracker.begin(key, base);
      const two = tracker.begin(key, first);
      const [early, late] = order === "older-first" ? [one, two] : [two, one];
      expect(tracker.settle(early, { ok: false })).toMatchObject({
        apply: false,
        pending: 1,
      });
      expect(tracker.settle(late, { ok: false })).toMatchObject({
        apply: true,
        canonical: base,
        pending: 0,
      });
    });
  }

  it("latest success applies immediately; an older failure neither rolls it back nor reapplies it", () => {
    const tracker = createNodeWriteTracker<typeof base>();
    const one = tracker.begin(key, base);
    const two = tracker.begin(key, first);
    expect(tracker.settle(two, { ok: true, node: second })).toEqual({
      apply: true,
      canonical: second,
      latest: true,
      pending: 1,
    });
    expect(tracker.settle(one, { ok: false })).toEqual({
      apply: false,
      latest: false,
      pending: 0,
    });
  });

  for (const order of ["success-first", "failure-first"] as const) {
    it(`older success plus latest failure (${order}) restores the confirmed older response`, () => {
      const tracker = createNodeWriteTracker<typeof base>();
      const one = tracker.begin(key, base);
      const two = tracker.begin(key, first);
      if (order === "success-first") {
        expect(tracker.settle(one, { ok: true, node: first }).apply).toBe(
          false,
        );
        expect(tracker.settle(two, { ok: false })).toMatchObject({
          apply: true,
          canonical: first,
        });
      } else {
        expect(tracker.settle(two, { ok: false }).apply).toBe(false);
        expect(tracker.settle(one, { ok: true, node: first })).toMatchObject({
          apply: true,
          canonical: first,
        });
      }
    });
  }

  it("a late earlier success cannot replace a later confirmed response", () => {
    const tracker = createNodeWriteTracker<typeof base>();
    const one = tracker.begin(key, base);
    const two = tracker.begin(key, first);
    const three = tracker.begin(key, second);
    expect(tracker.settle(two, { ok: true, node: second }).apply).toBe(false);
    expect(tracker.settle(three, { ok: false }).apply).toBe(false);
    expect(tracker.settle(one, { ok: true, node: first })).toMatchObject({
      apply: true,
      canonical: second,
    });
  });

  it("a new optimistic edit after an applied success still rolls back to that success", () => {
    const tracker = createNodeWriteTracker<typeof base>();
    const one = tracker.begin(key, base);
    const two = tracker.begin(key, first);
    expect(tracker.settle(two, { ok: true, node: second }).apply).toBe(true);
    const three = tracker.begin(key, second);
    expect(tracker.settle(three, { ok: false }).apply).toBe(false);
    expect(tracker.settle(one, { ok: false })).toMatchObject({
      apply: true,
      canonical: second,
    });
  });

  it("a completed group allows a fresh authoritative baseline; stale duplicate tickets are ignored", () => {
    const tracker = createNodeWriteTracker<typeof base>();
    const one = tracker.begin(key, base);
    tracker.settle(one, { ok: true, node: first });
    const two = tracker.begin(key, second);
    expect(tracker.settle(one, { ok: true, node: base }).apply).toBe(false);
    expect(tracker.settle(two, { ok: false })).toMatchObject({
      apply: true,
      canonical: second,
    });
  });

  it("project/tree/node keys cannot share canonical state or pending ownership", () => {
    const tracker = createNodeWriteTracker<typeof base>();
    const one = tracker.begin("1:10:100", base);
    const two = tracker.begin("2:10:100", second);
    expect(tracker.settle(one, { ok: true, node: first })).toMatchObject({
      apply: true,
      canonical: first,
      pending: 0,
    });
    expect(tracker.settle(two, { ok: false })).toMatchObject({
      apply: true,
      canonical: second,
      pending: 0,
    });
    expect(base).toEqual({ name: "BASE", x: 0 });
  });
});
