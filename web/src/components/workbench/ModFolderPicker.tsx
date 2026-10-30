"use client";

import {
  forwardRef,
  useCallback,
  useEffect,
  useImperativeHandle,
  useRef,
  type ChangeEvent,
} from "react";
import { useTranslations } from "use-intl";
import { useConfirm } from "@/components/shared/ConfirmDialogProvider";
import { hasDirectoryPicker } from "@/lib/fsAccess";
import {
  createDirHandleFromPreparedFileList,
  prepareFileListImport,
} from "@/lib/fileListDirHandle";

export interface ModFolderPickerHandle {
  pick: () => void;
}

interface ModFolderPickerProps {
  onPicked: (dirHandle: FileSystemDirectoryHandle) => void | Promise<void>;

  onError?: (message: string) => void;

  onSettled?: () => void;

  mode?: "read" | "readwrite";

  largeFolderThreshold?: number;
}

const DEFAULT_LARGE_FOLDER_THRESHOLD = 8000;

const PICKER_ID = "hoi4-mod-folder";

export const ModFolderPicker = forwardRef<
  ModFolderPickerHandle,
  ModFolderPickerProps
>(function ModFolderPicker(
  {
    onPicked,
    onError,
    onSettled,
    mode = "read",
    largeFolderThreshold = DEFAULT_LARGE_FOLDER_THRESHOLD,
  },
  ref,
) {
  const t = useTranslations("browserCompat");
  const confirmDialog = useConfirm();
  const inputRef = useRef<HTMLInputElement | null>(null);

  const pick = useCallback(() => {
    if (hasDirectoryPicker()) {
      window
        .showDirectoryPicker({ mode, id: PICKER_ID })
        .then(async (handle) => {
          await onPicked(handle);
        })
        .catch((e: unknown) => {
          if ((e as { name?: string })?.name === "AbortError") return;
          onError?.(e instanceof Error ? e.message : String(e));
        })
        .finally(() => {
          onSettled?.();
        });
      return;
    }

    inputRef.current?.click();
  }, [mode, onPicked, onError, onSettled]);

  const handleInputChange = useCallback(
    async (e: ChangeEvent<HTMLInputElement>) => {
      const inputEl = e.target;
      const fileList = inputEl.files;

      if (!fileList || fileList.length === 0) {
        inputEl.value = "";
        onSettled?.();
        return;
      }
      try {
        const prepared = await prepareFileListImport(fileList);
        const relevant = prepared.relevantCount;
        if (relevant === 0) {
          onError?.(t("noRelevantFiles"));
          return;
        }
        if (relevant > largeFolderThreshold) {
          const ok = await confirmDialog({
            body: t("largeFolderConfirm", { count: relevant }),
          });
          if (!ok) return;
        }

        const dirHandle = await createDirHandleFromPreparedFileList(prepared);
        await onPicked(dirHandle);
      } catch (err) {
        onError?.(err instanceof Error ? err.message : String(err));
      } finally {
        inputEl.value = "";
        onSettled?.();
      }
    },
    [onPicked, onError, onSettled, largeFolderThreshold, t, confirmDialog],
  );

  useImperativeHandle(ref, () => ({ pick }), [pick]);

  useEffect(() => {
    const el = inputRef.current;
    if (!el || !onSettled) return;
    const handler = () => onSettled();
    el.addEventListener("cancel", handler);
    return () => el.removeEventListener("cancel", handler);
  }, [onSettled]);

  return (
    <input
      ref={inputRef}
      type="file"
      multiple
      webkitdirectory=""
      hidden
      tabIndex={-1}
      aria-hidden
      onChange={handleInputChange}
    />
  );
});
