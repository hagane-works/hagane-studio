const DB_NAME = "hoi4hagane";
const DB_VERSION = 1;
const STORE = "modFolderHandles";

export interface StoredModFolder {
  projectId: number;
  dirHandle: FileSystemDirectoryHandle;

  name: string;

  savedAt: number;
}

export function isPersistableDirHandle(handle: unknown): boolean {
  return (
    typeof FileSystemDirectoryHandle !== "undefined" &&
    handle instanceof FileSystemDirectoryHandle
  );
}

function openDb(): Promise<IDBDatabase> {
  return new Promise((resolve, reject) => {
    if (typeof indexedDB === "undefined") {
      reject(new Error("IndexedDB unavailable"));
      return;
    }
    const req = indexedDB.open(DB_NAME, DB_VERSION);
    req.onupgradeneeded = () => {
      const db = req.result;
      if (!db.objectStoreNames.contains(STORE)) {
        db.createObjectStore(STORE, { keyPath: "projectId" });
      }
    };
    req.onsuccess = () => resolve(req.result);
    req.onerror = () => reject(req.error ?? new Error("IndexedDB open failed"));
  });
}

function runTx<T>(
  mode: IDBTransactionMode,
  body: (store: IDBObjectStore) => IDBRequest<T>,
): Promise<T> {
  return openDb().then(
    (db) =>
      new Promise<T>((resolve, reject) => {
        const tx = db.transaction(STORE, mode);
        const req = body(tx.objectStore(STORE));
        req.onsuccess = () => resolve(req.result);
        req.onerror = () =>
          reject(req.error ?? new Error("IndexedDB request failed"));
        tx.oncomplete = () => db.close();
      }),
  );
}

export async function saveModFolderHandle(
  projectId: number,
  dirHandle: FileSystemDirectoryHandle,
): Promise<void> {
  if (!isPersistableDirHandle(dirHandle)) return;
  try {
    const record: StoredModFolder = {
      projectId,
      dirHandle,
      name: dirHandle.name,
      savedAt: Date.now(),
    };
    await runTx("readwrite", (store) => store.put(record));
  } catch {}
}

export async function loadModFolderHandle(
  projectId: number,
): Promise<StoredModFolder | null> {
  try {
    const rec = await runTx<StoredModFolder | undefined>("readonly", (store) =>
      store.get(projectId),
    );
    if (!rec?.dirHandle) return null;
    return rec;
  } catch {
    return null;
  }
}

export async function clearModFolderHandle(projectId: number): Promise<void> {
  try {
    await runTx("readwrite", (store) => store.delete(projectId));
  } catch {}
}

export async function probeDirHandleAlive(
  dirHandle: FileSystemDirectoryHandle,
): Promise<boolean> {
  try {
    const it = dirHandle.values()[Symbol.asyncIterator]();
    await it.next();
    return true;
  } catch {
    return false;
  }
}

export async function ensureReadPermission(
  dirHandle: FileSystemDirectoryHandle,
): Promise<boolean> {
  type Permissioned = {
    queryPermission?: (d: { mode: "read" }) => Promise<PermissionState>;
    requestPermission?: (d: { mode: "read" }) => Promise<PermissionState>;
  };
  const h = dirHandle as unknown as Permissioned;
  try {
    if (typeof h.queryPermission === "function") {
      if ((await h.queryPermission({ mode: "read" })) === "granted")
        return true;
    }
    if (typeof h.requestPermission === "function") {
      return (await h.requestPermission({ mode: "read" })) === "granted";
    }

    return true;
  } catch {
    return false;
  }
}
