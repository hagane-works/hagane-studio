import { describe, expect, it, vi } from "vitest";
import {
  ensureReadPermission,
  isPersistableDirHandle,
  loadModFolderHandle,
  saveModFolderHandle,
  clearModFolderHandle,
} from "./modFolderHandleStore";

const shimHandle = {
  name: "my-mod",
  kind: "directory",
  values: () => [],
  getDirectoryHandle: async () => {},
  getFileHandle: async () => {},
} as unknown as FileSystemDirectoryHandle;

describe("isPersistableDirHandle", () => {
  it("FileList shim 不可持久化(它进不了 IndexedDB)", () => {
    expect(isPersistableDirHandle(shimHandle)).toBe(false);
  });

  it("null / 普通值不可持久化", () => {
    expect(isPersistableDirHandle(null)).toBe(false);
    expect(isPersistableDirHandle("/some/path")).toBe(false);
  });
});

describe("没有 IndexedDB 时必须静默降级", () => {
  it("save 不抛(shim 句柄本来就被挡掉)", async () => {
    await expect(saveModFolderHandle(1, shimHandle)).resolves.toBeUndefined();
  });

  it("load 返回 null 而不是抛", async () => {
    await expect(loadModFolderHandle(1)).resolves.toBeNull();
  });

  it("clear 不抛", async () => {
    await expect(clearModFolderHandle(1)).resolves.toBeUndefined();
  });
});

describe("ensureReadPermission", () => {
  function handleWith(perm: {
    query?: PermissionState | Error;
    request?: PermissionState | Error;
  }) {
    return {
      name: "my-mod",
      queryPermission:
        perm.query === undefined
          ? undefined
          : vi.fn(async () => {
              if (perm.query instanceof Error) throw perm.query;
              return perm.query;
            }),
      requestPermission:
        perm.request === undefined
          ? undefined
          : vi.fn(async () => {
              if (perm.request instanceof Error) throw perm.request;
              return perm.request;
            }),
    } as unknown as FileSystemDirectoryHandle;
  }

  it("已授权 → true,且不再骚扰用户申请", async () => {
    const h = handleWith({ query: "granted", request: "denied" });
    await expect(ensureReadPermission(h)).resolves.toBe(true);
    expect(
      (h as unknown as { requestPermission: ReturnType<typeof vi.fn> })
        .requestPermission,
    ).not.toHaveBeenCalled();
  });

  it("prompt → 补申请 → 同意则 true", async () => {
    await expect(
      ensureReadPermission(handleWith({ query: "prompt", request: "granted" })),
    ).resolves.toBe(true);
  });

  it("prompt → 补申请 → 拒绝则 false(调用方据此清记录并回退手动选择)", async () => {
    await expect(
      ensureReadPermission(handleWith({ query: "prompt", request: "denied" })),
    ).resolves.toBe(false);
  });

  it("没有权限 API(shim 句柄)→ true,它本来就在内存里", async () => {
    await expect(ensureReadPermission(shimHandle)).resolves.toBe(true);
  });

  it("权限 API 抛异常 → false 而不是把异常冒到导入流程", async () => {
    await expect(
      ensureReadPermission(handleWith({ query: new Error("boom") })),
    ).resolves.toBe(false);
  });
});
