import { createMainThreadCheckpoint } from "@/lib/fs-scanner";
import type { ModFile } from "./modMapFolder";

export interface ModMapScanProgress {
  files: number;
  directories: number;
  path: string;

  total?: number;
}

const FILE_CONCURRENCY = 8;

function abortError(): DOMException {
  return new DOMException("The folder scan was cancelled", "AbortError");
}

function checkAbort(signal?: AbortSignal): void {
  if (signal?.aborted) throw abortError();
}

function abortable<T>(promise: Promise<T>, signal?: AbortSignal): Promise<T> {
  if (!signal) return promise;
  if (signal.aborted) {
    void promise.catch(() => undefined);
    return Promise.reject(abortError());
  }
  return new Promise<T>((resolve, reject) => {
    const abort = () => reject(abortError());
    signal.addEventListener("abort", abort, { once: true });
    promise.then(
      (value) => {
        signal.removeEventListener("abort", abort);
        if (signal.aborted) reject(abortError());
        else resolve(value);
      },
      (error: unknown) => {
        signal.removeEventListener("abort", abort);
        reject(signal.aborted ? abortError() : error);
      },
    );
  });
}

function needed(path: string): boolean {
  const lower = path.toLowerCase();
  return (
    /(?:^|\/)map\/(?:provinces\.bmp|heightmap\.bmp|terrain\.bmp|rivers\.bmp|definition\.csv)$/.test(
      lower,
    ) ||
    /(?:^|\/)map\/strategicregions\/.+\.txt$/.test(lower) ||
    /(?:^|\/)history\/states\/.+\.txt$/.test(lower) ||
    /(?:^|\/)localisation\/.+\.yml$/.test(lower)
  );
}

function reporter(
  onProgress: (progress: ModMapScanProgress) => void,
  signal?: AbortSignal,
  total?: number,
) {
  const state: ModMapScanProgress = {
    files: 0,
    directories: 0,
    path: "",
    ...(total === undefined ? {} : { total }),
  };
  const checkpoint = createMainThreadCheckpoint();
  let lastTime = 0;
  let lastItems = 0;
  const report = (force = false) => {
    checkAbort(signal);
    const items = state.files + state.directories;
    const now = Date.now();
    if (force || items - lastItems >= 128 || now - lastTime >= 50) {
      onProgress({ ...state });
      lastItems = items;
      lastTime = now;
      checkAbort(signal);
    }
  };
  const step = async () => {
    report();
    const pause = checkpoint();
    if (pause) await abortable(pause, signal);
    checkAbort(signal);
  };
  return { state, report, step };
}

export async function readModMapDirectory(
  root: FileSystemDirectoryHandle,
  onProgress: (progress: ModMapScanProgress) => void,
  signal?: AbortSignal,
): Promise<{ files: ModFile[]; count: number }> {
  checkAbort(signal);
  const progress = reporter(onProgress, signal);
  const files: ModFile[] = [];
  let batch: { handle: FileSystemFileHandle; path: string }[] = [];
  const flush = async () => {
    if (!batch.length) return;
    checkAbort(signal);
    const current = batch;
    batch = [];
    const read = await abortable(
      Promise.all(
        current.map(async ({ handle, path }) => {
          checkAbort(signal);
          const file = await handle.getFile();
          checkAbort(signal);
          return { path, file };
        }),
      ),
      signal,
    );
    files.push(...read);
    await progress.step();
  };
  const walk = async (
    directory: FileSystemDirectoryHandle,
    path: string,
  ): Promise<void> => {
    checkAbort(signal);
    progress.state.directories += 1;
    progress.state.path = path;
    if (progress.state.directories === 1) progress.report(true);
    await progress.step();
    const iterator = directory.values()[Symbol.asyncIterator]();
    let completed = false;
    try {
      while (true) {
        checkAbort(signal);
        const next = await abortable(iterator.next(), signal);
        if (next.done) {
          completed = true;
          break;
        }
        const entry = next.value;
        const entryPath = path ? `${path}/${entry.name}` : entry.name;
        if (entry.kind === "directory") {
          await walk(entry, entryPath);
          progress.state.path = path;
        } else {
          progress.state.files += 1;
          if (needed(entryPath)) {
            batch.push({
              handle: entry as FileSystemFileHandle,
              path: entryPath,
            });
            if (batch.length === FILE_CONCURRENCY) await flush();
          }
        }
        await progress.step();
      }
    } finally {
      if (!completed && iterator.return) {
        try {
          void Promise.resolve(iterator.return()).catch(() => undefined);
        } catch {}
      }
    }
  };
  await walk(root, root.name);
  await flush();
  progress.state.path = root.name;
  progress.report(true);
  return { files, count: progress.state.files };
}

export async function readModMapFileList(
  list: ArrayLike<File>,
  onProgress: (progress: ModMapScanProgress) => void,
  signal?: AbortSignal,
): Promise<{ files: ModFile[]; count: number }> {
  checkAbort(signal);
  const progress = reporter(onProgress, signal, list.length);
  const directories = new Set<string>();
  const files: ModFile[] = [];
  progress.report(true);
  for (let i = 0; i < list.length; i += 1) {
    checkAbort(signal);
    const file = list[i];
    const path = (file.webkitRelativePath || file.name).replace(/\\/g, "/");
    const parts = path.split("/");
    let parent = "";
    for (let j = 0; j < parts.length - 1; j += 1) {
      parent = parent ? `${parent}/${parts[j]}` : parts[j];
      directories.add(parent);
    }
    progress.state.files += 1;
    progress.state.directories = directories.size;
    progress.state.path = parent;
    if (needed(path)) files.push({ path, file });
    await progress.step();
  }
  progress.report(true);
  return { files, count: progress.state.files };
}
