import { describe, expect, it, vi } from "vitest";
import {
  TECHNOLOGY_SOURCE_GUI_PATH,
  TECHNOLOGY_SOURCE_IMPORT_OPTIONS,
  TECHNOLOGY_SOURCE_TAGS_PATH,
  collectLocalTechnologySource,
} from "./techSource";

interface FakeDirectory {
  [name: string]: File | FakeDirectory;
}

type FakeNode = File | FakeDirectory;

function directory(
  name: string,
  children: Record<string, FakeNode>,
): FileSystemDirectoryHandle {
  const handle = {
    kind: "directory",
    name,
    async getDirectoryHandle(child: string) {
      const node = children[child];
      if (!node || node instanceof File)
        throw new DOMException("missing", "NotFoundError");
      return directory(child, node);
    },
    async getFileHandle(child: string) {
      const node = children[child];
      if (!(node instanceof File))
        throw new DOMException("missing", "NotFoundError");
      return { kind: "file", name: child, getFile: async () => node };
    },
    async *values() {
      for (const [child, node] of Object.entries(children)) {
        yield node instanceof File
          ? { kind: "file", name: child, getFile: async () => node }
          : directory(child, node);
      }
    },
  };
  return handle as unknown as FileSystemDirectoryHandle;
}

function validRoot(): FileSystemDirectoryHandle {
  return directory("Hearts of Iron IV", {
    interface: {
      "countrytechtreeview.gui": new File(["gui"], "countrytechtreeview.gui"),
      "unrelated.gui": new File(["ignored"], "unrelated.gui"),
    },
    common: {
      technology_tags: {
        "00_technology.txt": new File(["base"], "00_technology.txt"),
        "01_extra.txt": new File(["extra"], "01_extra.txt"),
        "README.md": new File(["ignored"], "README.md"),
      },
    },
  });
}

describe("local HOI4 technology source selection", () => {
  it("stages the exact GUI plus every technology_tags txt without unrelated files", async () => {
    const selection = await collectLocalTechnologySource(validRoot());
    expect(selection.label).toBe("Hearts of Iron IV");
    expect(selection.paths).toEqual([
      TECHNOLOGY_SOURCE_GUI_PATH,
      TECHNOLOGY_SOURCE_TAGS_PATH,
      "common/technology_tags/01_extra.txt",
    ]);
    expect(selection.files.map((file) => file.name)).toEqual([
      "countrytechtreeview.gui",
      "00_technology.txt",
      "01_extra.txt",
    ]);
  });

  it("fails before upload when the required GUI is absent", async () => {
    const root = directory("HOI4", {
      common: {
        technology_tags: {
          "00_technology.txt": new File(["x"], "00_technology.txt"),
        },
      },
    });
    await expect(collectLocalTechnologySource(root)).rejects.toMatchObject({
      code: "gui_missing",
    });
  });

  it("fails before upload when the canonical tags file is absent", async () => {
    const root = directory("HOI4", {
      interface: {
        "countrytechtreeview.gui": new File(["x"], "countrytechtreeview.gui"),
      },
      common: {
        technology_tags: { "01_extra.txt": new File(["x"], "01_extra.txt") },
      },
    });
    await expect(collectLocalTechnologySource(root)).rejects.toMatchObject({
      code: "tags_missing",
    });
  });

  it("keeps the source import entity-free", () => {
    expect(
      Object.values(TECHNOLOGY_SOURCE_IMPORT_OPTIONS).every(
        (value) => value === false,
      ),
    ).toBe(true);
    expect(vi.isMockFunction(collectLocalTechnologySource)).toBe(false);
  });
});
