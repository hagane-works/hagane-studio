import { describe, expect, it } from "vitest";
import { performScan, PREVIEW_SECTIONS } from "./scan";

function makeAccountingHandle(tree: Record<string, unknown>) {
  const opened: string[] = [];

  function node(
    obj: Record<string, unknown>,
    name: string,
    path: string,
  ): FileSystemDirectoryHandle {
    return {
      kind: "directory",
      name,
      async getDirectoryHandle(child: string) {
        const v = obj[child];
        if (!v || typeof v === "string") throw new Error("NotFound");
        const childPath = path ? `${path}/${child}` : child;
        opened.push(childPath);
        return node(v as Record<string, unknown>, child, childPath);
      },
      async getFileHandle(child: string) {
        const v = obj[child];
        if (typeof v !== "string") throw new Error("NotFound");
        return fileHandle(child, v);
      },
      async *values() {
        for (const [k, v] of Object.entries(obj)) {
          yield typeof v === "string"
            ? fileHandle(k, v)
            : node(v as Record<string, unknown>, k, path ? `${path}/${k}` : k);
        }
      },
    } as unknown as FileSystemDirectoryHandle;
  }

  function fileHandle(name: string, content: string): FileSystemFileHandle {
    return {
      kind: "file",
      name,
      async getFile() {
        const f = new File([content], name);

        Object.defineProperty(f, "text", { value: async () => content });
        return f;
      },
    } as unknown as FileSystemFileHandle;
  }

  return { root: node(tree, "", ""), opened };
}

const MOD = {
  "descriptor.mod": 'name="Test Mod"',
  common: {
    characters: { "GER_characters.txt": "x" },
    national_focus: { "GER.txt": "x" },
    ideas: { "GER_ideas.txt": "x" },
    decisions: { "GER_decisions.txt": "x" },
    country_tags: {
      "00_countries.txt":
        'GER = "countries/Germany.txt"\nSOV = "countries/Soviet.txt"',
    },

    technologies: { "tech.txt": "x" },
    bookmarks: { "bm.txt": "x" },
    raids: { "raid.txt": "x" },
    units: { "unit.txt": "x" },
    scripted_effects: { "eff.txt": "x" },
    doctrines: { tracks: { "d.txt": "x" } },
  },
  events: { "GER_events.txt": "x" },
  history: {
    countries: { "GER.txt": "x" },
    states: { "1-Anhalt.txt": "x" },
    units: { "GER_1936.txt": "x" },
  },
  localisation: { english: { "GER_l_english.yml": "x" } },
  interface: { "icons.gfx": "x" },
  dlc: { dlc001: { interface: { "d.gfx": "x" } } },
};

describe("performScan · sections", () => {
  it("立即 resolve 的目录句柄也会在首个分段状态后让出任务供 UI 绘制", async () => {
    const { root } = makeAccountingHandle(MOD);
    let latestStatus = "";
    let statusSeenByTimer = "";
    let finished = false;

    const paintOpportunity = new Promise<void>((resolve) => {
      setTimeout(() => {
        statusSeenByTimer = latestStatus;
        resolve();
      }, 0);
    });
    const scanDone = performScan(root, "GER", (status) => {
      latestStatus = status;
    }).then((result) => {
      finished = true;
      return result;
    });

    await paintOpportunity;
    expect(statusSeenByTimer).toBe("Scanning characters for GER...");
    expect(finished).toBe(false);
    await scanDone;
  });

  it("预览段跳过最大的那几棵树(根本不去打开它们)", async () => {
    const { root, opened } = makeAccountingHandle(MOD);
    await performScan(root, "", () => {}, undefined, {
      sections: PREVIEW_SECTIONS,
    });

    for (const skipped of [
      "interface",
      "dlc",
      "localisation",
      "history/countries",
      "history/states",
      "history/units",
      "common/units",
      "common/technologies",
      "common/bookmarks",
      "common/raids",
    ]) {
      expect(opened, `${skipped} 不该被打开`).not.toContain(skipped);
    }
  });

  it("预览段仍然拿到它要显示的两个数字", async () => {
    const { root } = makeAccountingHandle(MOD);
    const r = await performScan(root, "", () => {}, undefined, {
      sections: PREVIEW_SECTIONS,
    });

    expect(r.country_tags.declaredTagCount).toBe(2);

    const entities =
      r.characters.fileCount +
      r.ideas.fileCount +
      r.focus_trees.fileCount +
      r.events.fileCount +
      r.decisions.fileCount;
    expect(entities).toBe(5);
  });

  it("不传 sections = 全量扫描,导入路径不受影响", async () => {
    const { root, opened } = makeAccountingHandle(MOD);
    const r = await performScan(root, "", () => {});

    expect(opened).toContain("interface");
    expect(opened).toContain("localisation");
    expect(opened).toContain("history/states");
    expect(r.technologies.fileCount).toBe(1);
    expect(r.states.fileCount).toBe(1);
    expect(r.loc.fileCount).toBe(1);
    expect(r.units.fileCount).toBe(2);
    expect(r.gfx.fileCount).toBeGreaterThan(0);
    expect(r.common_rules.fileCount).toBeGreaterThan(0);
    expect(r.descriptor).not.toBeNull();
  });

  it("预览段不读 descriptor(弹窗自己在第一步已经读过了)", async () => {
    const { root } = makeAccountingHandle(MOD);
    const r = await performScan(root, "", () => {}, undefined, {
      sections: PREVIEW_SECTIONS,
    });
    expect(r.descriptor).toBeNull();
  });
});
