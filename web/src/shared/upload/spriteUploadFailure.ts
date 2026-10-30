import { SpriteUploadError } from "@/lib/spriteUploadError";

export interface SpriteUploadFailure {
  key:
    | "errSourceChanged"
    | "errNotReplaceable"
    | "errServer"
    | "errTooLarge"
    | "errRefused";

  detail: string;

  retryable: boolean;
}

export function sourceChanged(err: unknown): boolean {
  return (
    err instanceof SpriteUploadError &&
    err.status === 409 &&
    /source changed/i.test(err.detail)
  );
}

export function spriteUploadFailure(err: unknown): SpriteUploadFailure {
  const detail = err instanceof Error ? err.message : String(err);
  if (!(err instanceof SpriteUploadError))
    return { key: "errServer", detail, retryable: true };
  if (err.status === 0 || err.status >= 500)
    return { key: "errServer", detail, retryable: true };
  if (sourceChanged(err))
    return { key: "errSourceChanged", detail, retryable: false };
  if (err.status === 409 && /no longer declares/i.test(err.detail))
    return { key: "errNotReplaceable", detail, retryable: false };
  if (err.status === 413)
    return { key: "errTooLarge", detail, retryable: false };
  return { key: "errRefused", detail, retryable: false };
}
