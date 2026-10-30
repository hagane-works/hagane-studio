import { IMAGE_EXTS, isModMapFile } from "./fs-scanner";

const IMPORT_RELEVANT_EXTS: ReadonlySet<string> = new Set<string>([
  "txt",
  "gfx",
  "gui",
  "yml",
  "mod",
  ...IMAGE_EXTS,
]);

function extensionOf(name: string): string {
  const dot = name.lastIndexOf(".");
  return dot >= 0 ? name.slice(dot + 1).toLowerCase() : "";
}

export function isImportRelevant(file: File): boolean {
  if (IMPORT_RELEVANT_EXTS.has(extensionOf(file.name))) return true;

  const parts = (file.webkitRelativePath || file.name)
    .split("/")
    .filter(Boolean);
  return parts.length > 1 && isModMapFile(parts.slice(1).join("/"));
}

export function countImportRelevant(
  fileList: FileList | File[],
  filter: (file: File) => boolean = isImportRelevant,
): number {
  let n = 0;

  for (let i = 0; i < fileList.length; i++) {
    const file = fileList[i];
    if (file && filter(file)) n += 1;
  }
  return n;
}

const DEFAULT_COOPERATIVE_CHUNK_SIZE = 512;

function yieldToEventLoop(): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, 0));
}

function normalizedChunkSize(value: number | undefined): number {
  if (value === undefined || !Number.isFinite(value) || value <= 0) {
    return DEFAULT_COOPERATIVE_CHUNK_SIZE;
  }
  return Math.max(1, Math.floor(value));
}

function rootNameForFile(file: File): string | null {
  const rel = file.webkitRelativePath || file.name;
  return rel.split("/")[0] || null;
}

export interface PreparedFileListImport {
  rootName: string;
  files: File[];
  relevantCount: number;
}

export async function prepareFileListImport(
  fileList: FileList | File[],
  opts: {
    filter?: (file: File) => boolean;

    chunkSize?: number;
  } = {},
): Promise<PreparedFileListImport> {
  const filter = opts.filter ?? isImportRelevant;
  const chunkSize = normalizedChunkSize(opts.chunkSize);
  const files: File[] = [];
  let rootName = "mod";
  let foundRootName = false;

  await yieldToEventLoop();

  for (let i = 0; i < fileList.length; i++) {
    const file = fileList[i];
    if (file) {
      if (!foundRootName) {
        const candidate = rootNameForFile(file);
        if (candidate) {
          rootName = candidate;
          foundRootName = true;
        }
      }
      if (filter(file)) files.push(file);
    }

    if ((i + 1) % chunkSize === 0 && i + 1 < fileList.length) {
      await yieldToEventLoop();
    }
  }

  return { rootName, files, relevantCount: files.length };
}

interface FileNode {
  kind: "file";
  name: string;
  file: File;
}
interface DirNode {
  kind: "directory";
  name: string;
  children: Map<string, DirNode | FileNode>;
}

function newRootNode(name: string): DirNode {
  return { kind: "directory", name, children: new Map() };
}

function insertFile(root: DirNode, file: File): void {
  const rel = file.webkitRelativePath || file.name;
  const parts = rel.split("/").filter(Boolean);

  const segments = parts.length > 1 ? parts.slice(1) : parts;
  if (segments.length === 0) return;

  let cursor = root;
  for (let i = 0; i < segments.length - 1; i++) {
    const seg = segments[i];
    const existing = cursor.children.get(seg);
    if (existing && existing.kind === "directory") {
      cursor = existing;
    } else {
      const next: DirNode = {
        kind: "directory",
        name: seg,
        children: new Map(),
      };
      cursor.children.set(seg, next);
      cursor = next;
    }
  }
  const fileName = segments[segments.length - 1];
  cursor.children.set(fileName, { kind: "file", name: fileName, file });
}

class FileListFileHandle {
  readonly kind = "file" as const;
  constructor(
    readonly name: string,
    private readonly _file: File,
  ) {}
  async getFile(): Promise<File> {
    return this._file;
  }
}

class FileListDirectoryHandle {
  readonly kind = "directory" as const;
  constructor(
    readonly name: string,
    private readonly _node: DirNode,
  ) {}

  async getDirectoryHandle(name: string): Promise<FileListDirectoryHandle> {
    const child = this._node.children.get(name);
    if (!child || child.kind !== "directory") {
      throw new DOMException(
        `No subdirectory named "${name}"`,
        "NotFoundError",
      );
    }
    return new FileListDirectoryHandle(name, child);
  }

  async getFileHandle(name: string): Promise<FileListFileHandle> {
    const child = this._node.children.get(name);
    if (!child || child.kind !== "file") {
      throw new DOMException(`No file named "${name}"`, "NotFoundError");
    }
    return new FileListFileHandle(name, child.file);
  }

  async *values(): AsyncIterableIterator<
    FileListFileHandle | FileListDirectoryHandle
  > {
    for (const child of this._node.children.values()) {
      yield child.kind === "file"
        ? new FileListFileHandle(child.name, child.file)
        : new FileListDirectoryHandle(child.name, child);
    }
  }

  async *entries(): AsyncIterableIterator<
    [string, FileListFileHandle | FileListDirectoryHandle]
  > {
    for await (const handle of this.values()) yield [handle.name, handle];
  }
  async *keys(): AsyncIterableIterator<string> {
    for (const name of this._node.children.keys()) yield name;
  }
  [Symbol.asyncIterator](): AsyncIterableIterator<
    [string, FileListFileHandle | FileListDirectoryHandle]
  > {
    return this.entries();
  }
}

export function isFileListDirectoryHandle(
  handle: FileSystemDirectoryHandle,
): boolean {
  return handle instanceof FileListDirectoryHandle;
}

export function createDirHandleFromFileList(
  fileList: FileList | File[],
  opts: { filter?: (file: File) => boolean } = {},
): FileSystemDirectoryHandle {
  const filter = opts.filter ?? isImportRelevant;

  let rootName = "mod";
  for (let i = 0; i < fileList.length; i++) {
    const file = fileList[i];
    if (!file) continue;
    const candidate = rootNameForFile(file);
    if (candidate) {
      rootName = candidate;
      break;
    }
  }

  const root = newRootNode(rootName);
  for (let i = 0; i < fileList.length; i++) {
    const file = fileList[i];
    if (file && filter(file)) insertFile(root, file);
  }

  return new FileListDirectoryHandle(
    rootName,
    root,
  ) as unknown as FileSystemDirectoryHandle;
}

export async function createDirHandleFromPreparedFileList(
  prepared: PreparedFileListImport,
  opts: {
    chunkSize?: number;
  } = {},
): Promise<FileSystemDirectoryHandle> {
  const chunkSize = normalizedChunkSize(opts.chunkSize);
  const root = newRootNode(prepared.rootName);

  await yieldToEventLoop();

  for (let i = 0; i < prepared.files.length; i++) {
    insertFile(root, prepared.files[i]);
    if ((i + 1) % chunkSize === 0 && i + 1 < prepared.files.length) {
      await yieldToEventLoop();
    }
  }

  return new FileListDirectoryHandle(
    prepared.rootName,
    root,
  ) as unknown as FileSystemDirectoryHandle;
}
