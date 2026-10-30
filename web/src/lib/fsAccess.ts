type MaybePickerWindow = Window &
  typeof globalThis & {
    showDirectoryPicker?: unknown;
  };

export function hasDirectoryPicker(): boolean {
  return (
    typeof window !== "undefined" &&
    typeof (window as MaybePickerWindow).showDirectoryPicker === "function"
  );
}

export function canWriteToFolder(): boolean {
  return hasDirectoryPicker();
}
