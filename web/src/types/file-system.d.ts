interface FileSystemHandlePermissionDescriptor {
  mode?: "read" | "readwrite";
}

type PermissionState = "granted" | "denied" | "prompt";

interface FileSystemHandle {
  kind: "file" | "directory";
  name: string;

  queryPermission?(
    descriptor?: FileSystemHandlePermissionDescriptor,
  ): Promise<PermissionState>;
  requestPermission?(
    descriptor?: FileSystemHandlePermissionDescriptor,
  ): Promise<PermissionState>;
}

interface FileSystemFileHandle extends FileSystemHandle {
  kind: "file";
  getFile(): Promise<File>;
}

interface FileSystemDirectoryHandle extends FileSystemHandle {
  kind: "directory";
  getDirectoryHandle(
    name: string,
    options?: { create?: boolean },
  ): Promise<FileSystemDirectoryHandle>;
  getFileHandle(
    name: string,
    options?: { create?: boolean },
  ): Promise<FileSystemFileHandle>;
  values(): AsyncIterableIterator<
    FileSystemFileHandle | FileSystemDirectoryHandle
  >;
}

interface ShowDirectoryPickerOptions {
  mode?: "read" | "readwrite";
  startIn?:
    "desktop" | "documents" | "downloads" | "music" | "pictures" | "videos";

  id?: string;
}

interface Window {
  showDirectoryPicker(
    options?: ShowDirectoryPickerOptions,
  ): Promise<FileSystemDirectoryHandle>;
}
