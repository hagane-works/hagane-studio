import { handleScanWorkerRequest } from "./scan-worker-runtime";
import type {
  ScanWorkerRequest,
  ScanWorkerResponse,
} from "./scan-worker-protocol";

type ModuleWorkerScope = {
  addEventListener(
    type: "message",
    listener: (event: MessageEvent<ScanWorkerRequest>) => void,
  ): void;
  addEventListener(
    type: "messageerror",
    listener: (event: MessageEvent<unknown>) => void,
  ): void;
  postMessage(message: ScanWorkerResponse): void;
};

const workerScope = self as unknown as ModuleWorkerScope;

workerScope.addEventListener("message", (event) => {
  void handleScanWorkerRequest(event.data, (response) => {
    workerScope.postMessage(response);
  });
});

workerScope.addEventListener("messageerror", () => {
  workerScope.postMessage({
    type: "error",
    error: {
      name: "DataCloneError",
      message: "The directory handle could not be decoded by the scan worker.",
    },
  });
});

export {};
