import type { ScanResults } from "./types";
import type { ScanSection, ScanStatusEvent } from "./scan";

export interface ScanWorkerRequest {
  type: "scan";
  dirHandle: FileSystemDirectoryHandle;
  tag: string;
  sections?: ScanSection[];
}

export type ScanWorkerResponse =
  | ({ type: "status" } & ScanStatusEvent)
  | { type: "result"; result: ScanResults }
  | {
      type: "error";
      error: { name: string; message: string; stack?: string };
    };
