import { describe, expect, it, vi } from "vitest";
import {
  createEditRevisionQueue,
  editRevisionHeaders,
  validEditRevisions,
} from "./editRevisions";

function deferred<T>() {
  let resolve!: (value: T) => void;
  let reject!: (error: Error) => void;
  const promise = new Promise<T>((yes, no) => {
    resolve = yes;
    reject = no;
  });
  return { promise, resolve, reject };
}

describe("edit revision snapshots", () => {
  it("preserves bigint strings and rejects numbers/unknown scopes", () => {
    expect(
      validEditRevisions({
        "focus_tree:8": "9007199254740993",
        "bookmark:1": 1,
        "project:1": "1",
      }),
    ).toEqual({ "focus_tree:8": "9007199254740993" });
    expect(editRevisionHeaders({ "bookmark:1": "2" })).toEqual({
      "X-Edit-Revisions": '{"bookmark:1":"2"}',
    });
  });
  it("explicit canvas continuation follows its still-pending predecessor", async () => {
    const queue = createEditRevisionQueue();
    const first = deferred<{
      value: string;
      revisions: Record<string, string>;
    }>();
    const write = vi.fn(async (revisions) => ({
      value: revisions["focus_tree:1"],
      revisions: {},
    }));
    const a = queue.run(1, { "focus_tree:1": "4" }, async () => first.promise);
    const b = queue.run(1, { "focus_tree:1": "4" }, write, true);
    await Promise.resolve();
    expect(write).not.toHaveBeenCalled();
    first.resolve({ value: "a", revisions: { "focus_tree:1": "5" } });
    await a;
    expect(await b).toBe("5");
  });
  it("does not advance after a conflict or retry it", async () => {
    const queue = createEditRevisionQueue();
    const fail = vi.fn(async () => {
      throw new Error("save_conflict");
    });
    await expect(queue.run(1, { "focus_tree:1": "4" }, fail)).rejects.toThrow(
      "save_conflict",
    );
    const seen = await queue.run(
      1,
      { "focus_tree:1": "4" },
      async (revisions) => ({ value: revisions, revisions: {} }),
    );
    expect(seen).toEqual({ "focus_tree:1": "4" });
    expect(fail).toHaveBeenCalledTimes(1);
  });
  it("an unrelated refreshed token never rebases an older draft chain", async () => {
    const queue = createEditRevisionQueue();
    await queue.run(1, { "focus_tree:1": "1" }, async () => ({
      value: null,
      revisions: { "focus_tree:1": "2" },
    }));

    await queue.run(1, { "focus_tree:1": "3" }, async () => ({
      value: null,
      revisions: { "focus_tree:1": "4" },
    }));
    expect(
      await queue.run(1, { "focus_tree:1": "1" }, async (revisions) => ({
        value: revisions,
        revisions: {},
      })),
    ).toEqual({ "focus_tree:1": "1" });
  });
  it("different projects and trees are independent; multi-scope work waits for both", async () => {
    const queue = createEditRevisionQueue();
    const held = deferred<{ value: null; revisions: Record<string, string> }>();
    const a = queue.run(1, { "focus_tree:1": "1" }, () => held.promise);
    expect(
      await queue.run(2, { "focus_tree:1": "1" }, async () => ({
        value: "other",
        revisions: {},
      })),
    ).toBe("other");
    const multi = vi.fn(async () => ({ value: null, revisions: {} }));
    const b = queue.run(1, { "focus_tree:1": "1", "bookmark:2": "1" }, multi);
    await Promise.resolve();
    expect(multi).not.toHaveBeenCalled();
    held.resolve({ value: null, revisions: {} });
    await a;
    await b;
    expect(multi).toHaveBeenCalledOnce();
  });
});
