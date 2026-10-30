import { isFileListDirectoryHandle } from "@/lib/fileListDirHandle";
import { performScan, type PerformScanOptions } from "./scan";
import type {
  ScanWorkerRequest,
  ScanWorkerResponse,
} from "./scan-worker-protocol";
import type { ScanResults, ScanStatusTranslator } from "./types";

class WorkerRequestNotSentError extends Error {
  constructor(readonly reason: unknown) {
    super("The directory handle could not be sent to the scan worker.");
    this.name = "WorkerRequestNotSentError";
  }
}

function scanInModuleWorker(
  worker: Worker,
  request: ScanWorkerRequest,
  onStatus: (message: string) => void,
  t?: ScanStatusTranslator,
  signal?: AbortSignal,
): Promise<ScanResults> {
  return new Promise((resolve, reject) => {
    let settled = false;
    const rejectAborted = () => {
      cleanup();
      reject(
        signal?.reason ??
          new DOMException("The mod scan was cancelled.", "AbortError"),
      );
    };
    const cleanup = () => {
      if (settled) return;
      settled = true;
      signal?.removeEventListener("abort", rejectAborted);
      worker.onmessage = null;
      worker.onerror = null;
      worker.onmessageerror = null;
      worker.terminate();
    };

    if (signal) {
      signal.addEventListener("abort", rejectAborted, { once: true });
      if (signal.aborted) {
        rejectAborted();
        return;
      }
    }

    worker.onmessage = (event: MessageEvent<ScanWorkerResponse>) => {
      const response = event.data;
      if (response?.type === "status") {
        try {
          onStatus(t ? t(response.key, response.values) : response.fallback);
        } catch (reason) {
          cleanup();
          reject(reason);
        }
        return;
      }
      if (response?.type === "result") {
        cleanup();
        resolve(response.result);
        return;
      }
      if (response?.type === "error") {
        const error = new Error(response.error.message);
        error.name = response.error.name;
        if (response.error.stack) error.stack = response.error.stack;
        cleanup();
        reject(error);
        return;
      }
      cleanup();
      reject(new Error("The mod scan worker returned an invalid response."));
    };
    worker.onerror = (event: ErrorEvent) => {
      event.preventDefault();
      cleanup();
      reject(new Error(event.message || "The mod scan worker failed to load."));
    };
    worker.onmessageerror = () => {
      cleanup();
      reject(new Error("The mod scan worker result could not be decoded."));
    };

    try {
      worker.postMessage(request);
    } catch (reason) {
      cleanup();
      reject(new WorkerRequestNotSentError(reason));
    }
  });
}

export async function performScanInBrowser(
  dirHandle: FileSystemDirectoryHandle,
  tag: string,
  onStatus: (message: string) => void,
  t?: ScanStatusTranslator,
  opts?: PerformScanOptions,
): Promise<ScanResults> {
  opts?.signal?.throwIfAborted();
  const mainThreadScan = () => performScan(dirHandle, tag, onStatus, t, opts);

  if (isFileListDirectoryHandle(dirHandle) || typeof Worker !== "function") {
    return mainThreadScan();
  }

  let worker: Worker;
  try {
    worker = new Worker(new URL("./scan.worker.ts", import.meta.url), {
      type: "module",
      name: "hoi4-mod-folder-scan",
    });
  } catch {
    return mainThreadScan();
  }

  const request: ScanWorkerRequest = {
    type: "scan",
    dirHandle,
    tag,
    ...(opts?.sections ? { sections: [...opts.sections] } : {}),
  };

  try {
    return await scanInModuleWorker(worker, request, onStatus, t, opts?.signal);
  } catch (reason) {
    if (reason instanceof WorkerRequestNotSentError) {
      return mainThreadScan();
    }
    throw reason;
  }
}
