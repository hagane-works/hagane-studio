import { afterEach, describe, expect, it, vi } from "vitest";

const texts = vi.hoisted(() => ({ now: {} as Record<string, string> }));
vi.mock("@/lib/guiRenderApi", () => ({
  guiRenderApi: {
    textOverrides: async () => ({ ...texts.now }),
    setText: async (key: string, value: string) => {
      texts.now[key] = value;
      return { key, value };
    },
    clearText: async (key: string) => {
      delete texts.now[key];
      return { key, removed: true };
    },
  },
}));

import {
  UndoRefused,
  fileUndo,
  pick,
  recordChange,
  rowUndo,
  textBefore,
  textUndo,
  undoAll,
  useChangeLog,
} from "./changeLog";

afterEach(() => {
  useChangeLog.setState({ entries: [] });
  vi.unstubAllGlobals();
});

describe("改动记录", () => {
  it("lists saves newest first and takes one back once", async () => {
    const undo = vi.fn(async () => {});
    recordChange({
      projectId: 2,
      title: "Conquer",
      kind: "战争目标 · take_state",
      undo,
    });
    recordChange({ projectId: 2, title: "换图 · GFX_x", kind: "图片" });
    const [newest, older] = useChangeLog.getState().entries;
    expect(newest.title).toBe("换图 · GFX_x");
    expect(newest.undo).toBeUndefined();
    expect(older.state).toBe("saved");

    expect(await useChangeLog.getState().undo(older.id)).toBe(true);
    expect(useChangeLog.getState().entries[1].state).toBe("undone");

    expect(await useChangeLog.getState().undo(older.id)).toBe(false);
    expect(undo).toHaveBeenCalledTimes(1);

    expect(await useChangeLog.getState().undo(newest.id)).toBe(false);
  });

  it("keeps a failed undo listed with why", async () => {
    recordChange({
      projectId: 2,
      title: "germany.5",
      undo: async () => Promise.reject(new UndoRefused("changedSince")),
    });
    const [entry] = useChangeLog.getState().entries;
    expect(await useChangeLog.getState().undo(entry.id)).toBe(false);

    expect(useChangeLog.getState().entries[0]).toMatchObject({
      state: "failed",
      refused: { reason: "changedSince", fields: [] },
    });
  });
});

describe("rowUndo", () => {
  type Row = { id: number; title: string; cost: number; note: string };
  const before = { id: 1, title: "Old", cost: 100, note: "kept" };
  const after = { ...before, title: "New", cost: 150 };
  const keys = ["title", "cost"];

  it("writes the saved fields back to what they were", async () => {
    let row: Row = after;
    const save = vi.fn(async (back: Partial<Row>) => {
      row = { ...row, ...back };
    });
    await rowUndo<Row>({
      before: pick(before, keys),
      after: pick(after, keys),
      load: async () => row,
      save,
    })();
    expect(save).toHaveBeenCalledWith({ title: "Old", cost: 100 }, after);
    expect(row).toEqual(before);
  });

  it("says so when the server kept a field it was told to put back", async () => {
    let row: Row & { cost: number | null } = { ...after };
    const save = vi.fn(async (back: Partial<Row>) => {
      row = {
        ...row,
        ...Object.fromEntries(
          Object.entries(back).filter(([, v]) => v !== null),
        ),
      };
    });
    const undo = rowUndo<Row>({
      before: { title: "Old", cost: null as unknown as number },
      after: pick(after, keys),
      load: async () => row,
      save,
    });
    await expect(undo()).rejects.toMatchObject({
      reason: "notAllBack",
      fields: ["cost"],
    });
  });

  it("leaves a row changed since alone", async () => {
    const save = vi.fn(async () => {});
    const undo = rowUndo<Row>({
      before: pick(before, keys),
      after: pick(after, keys),
      load: async () => ({ ...after, cost: 175 }),
      save,
    });
    await expect(undo()).rejects.toMatchObject({ reason: "changedSince" });
    expect(save).not.toHaveBeenCalled();
  });

  it("does not mind a field the save did not touch", async () => {
    let row: Row = { ...after, note: "edited" };
    const save = vi.fn(async (back: Partial<Row>) => {
      row = { ...row, ...back };
    });
    await rowUndo<Row>({
      before: pick(before, keys),
      after: pick(after, keys),
      load: async () => row,
      save,
    })();
    expect(row).toEqual({ ...before, note: "edited" });
  });
});

describe("fileUndo", () => {
  it("is there only when the save wrote files, and posts them back", async () => {
    expect(
      fileUndo(2, { name: "take_state", imported: false }),
    ).toBeUndefined();
    expect(fileUndo(2, { undo: { files: [] } })).toBeUndefined();
    const files = [
      {
        id: 979,
        file: "common/wargoals/00_invasion.txt",
        before: null,
        after_sha256: "ab",
      },
    ];
    const fetchMock = vi.fn(
      async () => new Response(JSON.stringify({ undone: 1 }), { status: 200 }),
    );
    vi.stubGlobal("fetch", fetchMock);
    await fileUndo(2, { undo: { files } })!();
    expect(fetchMock).toHaveBeenCalledWith(
      "/api/go/projects/2/gui-render/undo-files",
      expect.objectContaining({
        method: "POST",
        body: JSON.stringify({ files }),
      }),
    );

    fetchMock.mockResolvedValueOnce(
      new Response(JSON.stringify({ detail: "changed since" }), {
        status: 409,
      }),
    );
    await expect(fileUndo(2, { undo: { files } })!()).rejects.toThrow(
      "changed since",
    );
  });
});

describe("textUndo", () => {
  it("puts a rewritten label back to the project's earlier text, or to the game's", async () => {
    texts.now = { colony_name: "Colony" };
    const before = await textBefore("colony_name");
    texts.now.colony_name = "Dominion";
    await textUndo("colony_name", "Dominion", before)();
    expect(texts.now).toEqual({ colony_name: "Colony" });

    texts.now = {};
    const fresh = await textBefore("colony_name");
    texts.now.colony_name = "Dominion";
    const cleared = vi.fn();
    await textUndo("colony_name", "Dominion", fresh, { cleared })();
    expect(texts.now).toEqual({});
    expect(cleared).toHaveBeenCalledWith("colony_name");
  });

  it("leaves a label rewritten again since alone", async () => {
    texts.now = { colony_name: "Protectorate" };
    await expect(
      textUndo("colony_name", "Dominion", null)(),
    ).rejects.toMatchObject({ reason: "changedSince" });
    expect(texts.now).toEqual({ colony_name: "Protectorate" });
  });
});

describe("undoAll", () => {
  it("takes the writes back last first, and has no way back when a part has none", async () => {
    const order: string[] = [];
    const file = async () => void order.push("file");
    const name = async () => void order.push("name");
    await undoAll(file, name)!();
    expect(order).toEqual(["name", "file"]);
    expect(undoAll(file, undefined)).toBeUndefined();
    expect(undoAll()).toBeUndefined();
  });
});
