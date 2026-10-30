"use client";

import { useCallback, useState } from "react";

import type { SourceEditorChoice } from "./SourceEditorDialog";

interface UseImportedSourceGateOptions<Payload> {
  entitySource: string | undefined;

  performSave: (payload: Payload) => Promise<void>;

  performFork?: (newId: string) => Promise<void>;

  resetForm?: () => void;
}

export interface ImportedSourceGate<Payload> {
  showDialog: boolean;

  interceptSave: (payload: Payload) => boolean;
  handleChoice: (choice: SourceEditorChoice, newId?: string) => Promise<void>;
  handleCancel: () => void;
}

export function useImportedSourceGate<Payload>({
  entitySource,
  performSave,
  performFork,
  resetForm,
}: UseImportedSourceGateOptions<Payload>): ImportedSourceGate<Payload> {
  const [showDialog, setShowDialog] = useState(false);
  const [pendingPayload, setPendingPayload] = useState<Payload | null>(null);

  const interceptSave = useCallback(
    (payload: Payload): boolean => {
      if (entitySource === "imported") {
        setPendingPayload(payload);
        setShowDialog(true);
        return true;
      }
      return false;
    },
    [entitySource],
  );

  const handleChoice = useCallback(
    async (choice: SourceEditorChoice, newId?: string) => {
      setShowDialog(false);
      if (choice === "keep_imported") {
        resetForm?.();
        setPendingPayload(null);
        return;
      }
      if (choice === "override_as_creator" && pendingPayload) {
        try {
          await performSave(pendingPayload);
        } finally {
          setPendingPayload(null);
        }
        return;
      }
      if (choice === "create_new_copy" && newId && performFork) {
        try {
          await performFork(newId);
        } finally {
          setPendingPayload(null);
        }
      }
    },
    [pendingPayload, performSave, performFork, resetForm],
  );

  const handleCancel = useCallback(() => {
    setShowDialog(false);
    setPendingPayload(null);
  }, []);

  return { showDialog, interceptSave, handleChoice, handleCancel };
}
