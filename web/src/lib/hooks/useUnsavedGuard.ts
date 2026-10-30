import { useEffect } from "react";

import { useEditorDirtyStore } from "@/lib/store/editorDirtyStore";

export function useRegisterDirty(workbenchKey: string, isDirty: boolean): void {
  const setDirty = useEditorDirtyStore((s) => s.setDirty);
  useEffect(() => {
    setDirty(workbenchKey, isDirty);
  }, [workbenchKey, isDirty, setDirty]);

  useEffect(() => {
    return () => {
      setDirty(workbenchKey, false);
    };
  }, [workbenchKey, setDirty]);
}

export function useBeforeUnloadGuard(): void {
  useEffect(() => {
    const handler = (e: BeforeUnloadEvent) => {
      if (useEditorDirtyStore.getState().hasAnyDirty()) {
        e.preventDefault();

        e.returnValue = "";
      }
    };
    window.addEventListener("beforeunload", handler);
    return () => window.removeEventListener("beforeunload", handler);
  }, []);
}

export function confirmDiscardIfDirty(): boolean {
  const store = useEditorDirtyStore.getState();
  if (!store.hasAnyDirty()) return true;

  if (typeof window === "undefined") return true;
  return window.confirm(store.confirmMessage);
}
