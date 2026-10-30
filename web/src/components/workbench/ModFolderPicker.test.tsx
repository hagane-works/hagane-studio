import { createRef } from "react";
import { fireEvent, render, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { ModFolderPicker, type ModFolderPickerHandle } from "./ModFolderPicker";
import { readFileByPath } from "@/lib/fs-scanner";

const mocks = vi.hoisted(() => ({
  confirm: vi.fn(),
}));

vi.mock("@/lib/fsAccess", () => ({
  hasDirectoryPicker: () => false,
}));

vi.mock("@/components/shared/ConfirmDialogProvider", () => ({
  useConfirm: () => mocks.confirm,
}));

vi.mock("use-intl", () => ({
  useTranslations:
    () => (key: string, values?: Record<string, string | number>) =>
      values?.count === undefined ? key : `${key}:${values.count}`,
}));

function makeFile(relPath: string): File {
  const file = new File([relPath], relPath.split("/").pop()!);
  Object.defineProperty(file, "webkitRelativePath", {
    configurable: true,
    value: relPath,
  });
  return file;
}

function selectFiles(input: HTMLInputElement, files: File[]): void {
  Object.defineProperty(input, "files", {
    configurable: true,
    value: files,
  });
  fireEvent.change(input);
}

describe("ModFolderPicker webkitdirectory fallback", () => {
  beforeEach(() => {
    mocks.confirm.mockReset();
  });

  it("keeps the relevant-file threshold contract and cancels before tree delivery", async () => {
    mocks.confirm.mockResolvedValue(false);
    const onPicked = vi.fn();
    const onSettled = vi.fn();
    const pickerRef = createRef<ModFolderPickerHandle>();
    const { container } = render(
      <ModFolderPicker
        ref={pickerRef}
        onPicked={onPicked}
        onSettled={onSettled}
        largeFolderThreshold={1}
      />,
    );
    const input = container.querySelector(
      "input[type=file]",
    ) as HTMLInputElement;

    selectFiles(input, [
      makeFile("Threshold/common/a.txt"),
      makeFile("Threshold/common/b.txt"),
      makeFile("Threshold/music/ignored.ogg"),
    ]);

    await waitFor(() => {
      expect(mocks.confirm).toHaveBeenCalledWith({
        body: "largeFolderConfirm:2",
      });
      expect(onSettled).toHaveBeenCalledTimes(1);
    });
    expect(onPicked).not.toHaveBeenCalled();
  });

  it("delivers the same ordered, filtered handle after confirmation", async () => {
    mocks.confirm.mockResolvedValue(true);
    const onPicked = vi.fn();
    const { container } = render(
      <ModFolderPicker onPicked={onPicked} largeFolderThreshold={1} />,
    );
    const input = container.querySelector(
      "input[type=file]",
    ) as HTMLInputElement;
    const first = makeFile("Confirmed/common/a.txt");
    const second = makeFile("Confirmed/common/b.txt");

    selectFiles(input, [
      first,
      makeFile("Confirmed/music/ignored.ogg"),
      second,
    ]);

    await waitFor(() => expect(onPicked).toHaveBeenCalledTimes(1));
    const handle = onPicked.mock.calls[0][0] as FileSystemDirectoryHandle;
    expect(handle.name).toBe("Confirmed");
    expect(await readFileByPath(handle, "common/a.txt")).toBe(first);
    expect(await readFileByPath(handle, "common/b.txt")).toBe(second);
    expect(await readFileByPath(handle, "music/ignored.ogg")).toBeNull();
  });
});
