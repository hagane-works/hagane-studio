import { useCallback, useEffect } from "react";
import { useTranslations } from "use-intl";
import { useConfirm } from "@/components/shared/ConfirmDialogProvider";
import { useEditorDirtyStore } from "@/lib/store/editorDirtyStore";

interface ProgramBridge {
  open(url: string): void;
  close(answer: "close" | "asking" | "stay"): void;
}

declare global {
  interface Window {
    haganeWorks?: ProgramBridge;
    haganeWorksCloseRequested?: () => void;
    haganeWorksHasUnsaved?: () => boolean;
  }
}

function programBridge(): ProgramBridge | null {
  let current: Window | null = window;
  for (let depth = 0; current && depth < 8; depth++) {
    try {
      if (current.haganeWorks) return current.haganeWorks;
      current = current.opener as Window | null;
    } catch {
      return null;
    }
  }
  return null;
}

export function inProgramWindow(): boolean {
  return programBridge() !== null;
}

const openedWindows = new Set<Window>();

export function hasUnsavedChanges(): boolean {
  if (useEditorDirtyStore.getState().hasAnyDirty()) return true;
  for (const opened of openedWindows) {
    try {
      if (!opened.closed && opened.haganeWorksHasUnsaved?.()) return true;
    } catch {}
  }
  return false;
}

function outsideAddress(href: string): string {
  try {
    const url = new URL(href, location.href);
    return (url.protocol === "http:" || url.protocol === "https:") &&
      url.origin !== location.origin
      ? url.href
      : "";
  } catch {
    return "";
  }
}

export function installProgramWindow(): void {
  window.haganeWorksHasUnsaved = hasUnsavedChanges;
  if (!inProgramWindow()) return;
  const open = window.open.bind(window);
  window.open = (url?: string | URL, target?: string, features?: string) => {
    const outside = url ? outsideAddress(String(url)) : "";
    if (outside) {
      programBridge()?.open(outside);
      return null;
    }
    const opened = open(url, target, features);
    try {
      if (opened && opened.opener === window) openedWindows.add(opened);
    } catch {}
    return opened;
  };
  const follow = (event: MouseEvent) => {
    if (event.button > 1) return;
    const link =
      event.target instanceof Element ? event.target.closest("a[href]") : null;
    if (!(link instanceof HTMLAnchorElement) || link.hasAttribute("download"))
      return;
    const outside = outsideAddress(link.href);
    if (outside) {
      event.preventDefault();
      programBridge()?.open(outside);
    } else if (
      link.target &&
      !["_self", "_parent", "_top"].includes(link.target)
    ) {
      event.preventDefault();
      window.open(link.href, link.target);
    }
  };
  document.addEventListener("click", follow, true);
  document.addEventListener("auxclick", follow, true);
  document.addEventListener("contextmenu", (event) => {
    const target = event.target instanceof Element ? event.target : null;
    const editable = target?.closest(
      "input, textarea, [contenteditable]:not([contenteditable='false'])",
    );
    if (!editable && !window.getSelection()?.toString()) event.preventDefault();
  });
}

export function useConfirmClose(): () => Promise<boolean> {
  const confirmDialog = useConfirm();
  const t = useTranslations("desktop.unsaved");
  return useCallback(
    async () =>
      !hasUnsavedChanges() ||
      confirmDialog({
        title: t("title"),
        body: t("body"),
        confirmLabel: t("close"),
        danger: true,
      }),
    [confirmDialog, t],
  );
}

export function ProgramWindowClose(): null {
  const confirmClose = useConfirmClose();
  useEffect(() => {
    const bridge = window.haganeWorks;
    if (!bridge) return;
    window.haganeWorksCloseRequested = () => {
      if (!hasUnsavedChanges()) return bridge.close("close");
      bridge.close("asking");
      void confirmClose().then((close) =>
        bridge.close(close ? "close" : "stay"),
      );
    };
    return () => {
      delete window.haganeWorksCloseRequested;
    };
  }, [confirmClose]);
  return null;
}
