import { request } from "@/lib/api/_client";
import {
  editRevisionHeaders,
  editRevisionQueue,
  readEditRevisionHeaders,
  validEditRevisions,
  type EditRevisions,
} from "./editRevisions";
import {
  editRevisionErrorCode,
  localizedEditRevisionError,
} from "./editRevisionError";

type RevisionListener = (projectId: number, revisions: EditRevisions) => void;
const listeners = new Set<RevisionListener>();
export function publishEditRevisionWrites(
  projectId: number,
  revisions: EditRevisions,
) {
  for (const listener of listeners) listener(projectId, revisions);
}
export async function readProjectEditRevisions(
  projectId: number,
): Promise<EditRevisions> {
  const response = await request<{ edit_revisions: EditRevisions }>(
    `/go/projects/${projectId}/edit-revisions`,
  );
  return validEditRevisions(response.edit_revisions);
}
export function subscribeEditRevisionWrites(listener: RevisionListener) {
  listeners.add(listener);
  return () => {
    listeners.delete(listener);
  };
}

export function revisionRequest<T>(
  projectId: number,
  path: string,
  options: RequestInit,
  snapshot: EditRevisions,
  followPending = false,
): Promise<T> {
  return editRevisionQueue.run(
    projectId,
    snapshot,
    async (submitted) => {
      let revisions: EditRevisions = {};
      const value = await request<T>(path, {
        ...options,
        headers: { ...options.headers, ...editRevisionHeaders(submitted) },
        onResponse: (response) => {
          revisions = readEditRevisionHeaders(response.headers);
        },
      }).catch((error: unknown) => {
        throw localizedEditRevisionError(error);
      });
      for (const listener of listeners) listener(projectId, revisions);
      return { value, revisions };
    },
    followPending,
  );
}

export function revisionFetch(
  projectId: number,
  url: string,
  options: RequestInit,
  snapshot: EditRevisions,
  followPending = false,
): Promise<Response> {
  return editRevisionQueue.run(
    projectId,
    snapshot,
    async (submitted) => {
      const response = await fetch(url, {
        ...options,
        headers: { ...options.headers, ...editRevisionHeaders(submitted) },
      });
      if (!response.ok) {
        if (
          response.status === 409 ||
          response.status === 428 ||
          response.status === 500
        ) {
          const payload: unknown = await response
            .clone()
            .json()
            .catch(() => null);
          if (editRevisionErrorCode(payload))
            throw localizedEditRevisionError(payload);
        }
        return { value: response, revisions: {} };
      }
      const revisions = readEditRevisionHeaders(response.headers);
      for (const listener of listeners) listener(projectId, revisions);
      return { value: response, revisions };
    },
    followPending,
  );
}
