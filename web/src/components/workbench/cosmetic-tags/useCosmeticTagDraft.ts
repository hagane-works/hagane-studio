import { useEffect, useReducer, useRef, useState } from "react";
import { ApiError } from "@/lib/api/_client";
import {
  cosmeticTagsApi,
  type CosmeticDraft,
  type CosmeticResolution,
  type CosmeticSuffix,
  type CosmeticTagView,
} from "@/lib/api/cosmeticTags";
import { draftFrom, hasChanges, patchFor, retainEdits } from "./draft";

type Status =
  | "loading"
  | "ready"
  | "missing"
  | "saving"
  | "saved"
  | "failed"
  | "conflict"
  | "review"
  | "readbackFailed";
interface State {
  view: CosmeticTagView | null;
  draft: CosmeticDraft | null;
  status: Status;
  preview: CosmeticResolution | null;
  previewLocale: string;
  previewState: "pending" | "ready" | "failed";
}
type Action =
  | { type: "loaded"; view: CosmeticTagView }
  | { type: "status"; status: Status }
  | { type: "edit"; update: (draft: CosmeticDraft) => CosmeticDraft }
  | {
      type: "reconcile";
      view: CosmeticTagView;
      submitted: CosmeticDraft;
      status: Status;
    }
  | {
      type: "preview";
      preview: CosmeticResolution | null;
      state: State["previewState"];
      locale: string;
    };
const initial: State = {
  view: null,
  draft: null,
  status: "loading",
  preview: null,
  previewLocale: "",
  previewState: "pending",
};

function reducer(state: State, action: Action): State {
  switch (action.type) {
    case "loaded":
      return {
        ...initial,
        view: action.view,
        draft: draftFrom(action.view),
        status: "ready",
      };
    case "status":
      return { ...state, status: action.status };
    case "edit":
      return state.draft
        ? {
            ...state,
            draft: action.update(state.draft),
            status: ["saving", "conflict", "readbackFailed", "review"].includes(
              state.status,
            )
              ? state.status
              : "ready",
            preview: null,
            previewState: "pending",
          }
        : state;
    case "reconcile":
      return {
        ...state,
        view: action.view,
        draft: retainEdits(
          action.view,
          action.submitted,
          state.draft ?? action.submitted,
        ),
        status: action.status,
        preview: null,
        previewState: "pending",
      };
    case "preview":
      return {
        ...state,
        preview: action.preview,
        previewState: action.state,
        previewLocale: action.locale,
      };
  }
}

export function useCosmeticTagDraft(
  projectId: number,
  tag: string,
  locale: string,
) {
  const [state, dispatch] = useReducer(reducer, initial);
  const [loadAttempt, setLoadAttempt] = useState(0);
  const [previewAttempt, setPreviewAttempt] = useState(0);
  const alive = useRef(false);
  const mutationPending = useRef(false);

  useEffect(() => {
    alive.current = true;
    return () => {
      alive.current = false;
    };
  }, []);

  useEffect(() => {
    const controller = new AbortController();
    void cosmeticTagsApi
      .get(projectId, tag, controller.signal)
      .then((view) => {
        if (!controller.signal.aborted) dispatch({ type: "loaded", view });
      })
      .catch((error: unknown) => {
        if (!controller.signal.aborted)
          dispatch({
            type: "status",
            status:
              error instanceof ApiError && error.status === 404
                ? "missing"
                : "failed",
          });
      });
    return () => controller.abort();
  }, [projectId, tag, loadAttempt]);

  useEffect(() => {
    if (!state.draft) return;
    const controller = new AbortController();
    const draft = state.draft;
    const timer = setTimeout(() => {
      dispatch({ type: "preview", preview: null, state: "pending", locale });
      void cosmeticTagsApi
        .resolve(projectId, tag, locale, draft, controller.signal)
        .then((preview) => {
          if (!controller.signal.aborted)
            dispatch({ type: "preview", preview, state: "ready", locale });
        })
        .catch(() => {
          if (!controller.signal.aborted)
            dispatch({
              type: "preview",
              preview: null,
              state: "failed",
              locale,
            });
        });
    }, 180);
    return () => {
      clearTimeout(timer);
      controller.abort();
    };
  }, [projectId, tag, locale, state.draft, previewAttempt]);

  async function create() {
    if (mutationPending.current) return;
    mutationPending.current = true;
    dispatch({ type: "status", status: "saving" });
    try {
      const view = await cosmeticTagsApi.create(projectId, tag);
      if (alive.current) dispatch({ type: "loaded", view });
    } catch {
      if (alive.current) dispatch({ type: "status", status: "failed" });
    } finally {
      mutationPending.current = false;
    }
  }

  async function save() {
    if (!state.view || !state.draft || mutationPending.current) return;
    mutationPending.current = true;
    const submitted = state.draft;
    const baseline = draftFrom(state.view);
    const patch = patchFor(state.view, submitted);
    dispatch({ type: "status", status: "saving" });
    let persisted: CosmeticTagView | null = null;
    try {
      persisted = await cosmeticTagsApi.update(projectId, tag, patch);
      const readback = await cosmeticTagsApi.get(projectId, tag);
      if (alive.current)
        dispatch({
          type: "reconcile",
          view: readback,
          submitted:
            readback.revision === persisted.revision ? submitted : baseline,
          status: readback.revision === persisted.revision ? "saved" : "review",
        });
    } catch (error) {
      if (!alive.current) return;
      if (persisted)
        dispatch({
          type: "reconcile",
          view: persisted,
          submitted,
          status: "readbackFailed",
        });
      else
        dispatch({
          type: "status",
          status:
            error instanceof ApiError && error.status === 409
              ? "conflict"
              : "failed",
        });
    } finally {
      mutationPending.current = false;
    }
  }

  async function reviewLatest() {
    if (!state.view || mutationPending.current) return;
    mutationPending.current = true;
    const baseline = draftFrom(state.view);
    dispatch({ type: "status", status: "saving" });
    try {
      const view = await cosmeticTagsApi.get(projectId, tag);
      if (alive.current)
        dispatch({
          type: "reconcile",
          view,
          submitted: baseline,
          status: "review",
        });
    } catch {
      if (alive.current) dispatch({ type: "status", status: "readbackFailed" });
    } finally {
      mutationPending.current = false;
    }
  }

  function changeName(header: string, suffix: CosmeticSuffix, value: string) {
    dispatch({
      type: "edit",
      update: (draft) => {
        const names = {
          ...draft.name_locs,
          [header]: { ...draft.name_locs[header] },
        };
        if (value.trim() === "") delete names[header][suffix];
        else names[header][suffix] = value;
        return { ...draft, name_locs: names };
      },
    });
  }

  return {
    ...state,
    previewState:
      state.previewLocale === locale ? state.previewState : "pending",
    dirty: !!state.view && !!state.draft && hasChanges(state.view, state.draft),
    create,
    save,
    reviewLatest,
    changeName,
    edit: (update: (draft: CosmeticDraft) => CosmeticDraft) =>
      dispatch({ type: "edit", update }),
    retryLoad: () => {
      dispatch({ type: "status", status: "loading" });
      setLoadAttempt((n) => n + 1);
    },
    retryPreview: () => setPreviewAttempt((n) => n + 1),
    discard: () => {
      if (state.view) dispatch({ type: "loaded", view: state.view });
    },
  };
}
