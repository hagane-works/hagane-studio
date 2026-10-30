"use client";

import { useState } from "react";
import {
  mergeSaveDiagnostics,
  readSaveDiagnostics,
  type SaveDiagnostic,
} from "@/lib/saveDiagnostics";

export function useSaveDiagnostics(identity: string, payload: unknown) {
  const incoming = readSaveDiagnostics(payload);
  const [snapshot, setSnapshot] = useState(() => ({
    identity,
    payload,
    byIdentity: {
      [identity]: mergeSaveDiagnostics(undefined, incoming),
    } as Record<string, readonly SaveDiagnostic[] | undefined>,
  }));
  const diagnostics = mergeSaveDiagnostics(
    snapshot.byIdentity[identity],
    incoming,
  );
  if (snapshot.identity !== identity || snapshot.payload !== payload) {
    setSnapshot({
      identity,
      payload,
      byIdentity: { ...snapshot.byIdentity, [identity]: diagnostics },
    });
  }
  return { diagnostics };
}
