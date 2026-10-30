import { create } from "zustand";

interface PendingModFolderStore {
  projectId: number | null;

  dirHandle: FileSystemDirectoryHandle | null;

  setPendingModFolder: (payload: {
    projectId: number;
    dirHandle: FileSystemDirectoryHandle;
  }) => void;

  clearPendingModFolder: () => void;
}

export const usePendingModFolderStore = create<PendingModFolderStore>(
  (set) => ({
    projectId: null,
    dirHandle: null,

    setPendingModFolder: ({ projectId, dirHandle }) => {
      set({ projectId, dirHandle });
    },

    clearPendingModFolder: () => {
      set({ projectId: null, dirHandle: null });
    },
  }),
);

export const usePendingModFolderProjectId = () =>
  usePendingModFolderStore((s) => s.projectId);

export const usePendingModFolderHandle = () =>
  usePendingModFolderStore((s) => s.dirHandle);

export const useSetPendingModFolder = () =>
  usePendingModFolderStore((s) => s.setPendingModFolder);

export const useClearPendingModFolder = () =>
  usePendingModFolderStore((s) => s.clearPendingModFolder);
