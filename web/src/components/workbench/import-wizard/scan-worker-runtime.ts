import { performScan } from "./scan";
import type {
  ScanWorkerRequest,
  ScanWorkerResponse,
} from "./scan-worker-protocol";

export async function handleScanWorkerRequest(
  request: ScanWorkerRequest,
  emit: (response: ScanWorkerResponse) => void,
): Promise<void> {
  if (request.type !== "scan") {
    emit({
      type: "error",
      error: {
        name: "ProtocolError",
        message: "Unknown mod scan worker request.",
      },
    });
    return;
  }

  try {
    const result = await performScan(
      request.dirHandle,
      request.tag,

      () => {},
      undefined,
      {
        sections: request.sections,
        onStatusEvent: (event) => emit({ type: "status", ...event }),
      },
    );
    emit({ type: "result", result });
  } catch (reason) {
    const error = reason instanceof Error ? reason : new Error(String(reason));
    emit({
      type: "error",
      error: {
        name: error.name,
        message: error.message,
        ...(error.stack ? { stack: error.stack } : {}),
      },
    });
  }
}
