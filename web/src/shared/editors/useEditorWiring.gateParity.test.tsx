import { act, cleanup, renderHook } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { useScriptDraftStore } from "./scriptDraftStore";
import { useEditorWiring } from "./useEditorWiring";

vi.mock("use-intl", () => ({ useTranslations: () => (key: string) => key }));
vi.mock("sonner", () => ({ toast: { error: vi.fn() } }));
vi.mock("./cwtoolStore", () => ({ useCwtoolRules: () => ({}) }));
vi.mock("./cwtoolValidate", () => ({ cwtoolValidate: () => [] }));
vi.mock("./useScriptParse", () => ({
  useScriptParse: () => ({ triggerParse: vi.fn() }),
}));
vi.mock("@/lib/ideology/useIdeologyOptions", () => ({
  useIdeologyOptions: () => ({ knownSubIds: new Set() }),
}));

function seed() {
  useScriptDraftStore.getState().setDraft("focus:1", "submitted", "base");
  useScriptDraftStore.getState().setParseResult("focus:1", {
    status: "ok",
    errors: [],
    entity: { focus_id: "FIRST" },
    extras: null,
    warnings: [],
  });
}
function deferred() {
  let resolve!: () => void;
  let reject!: (error: Error) => void;
  const promise = new Promise<void>((yes, no) => {
    resolve = yes;
    reject = no;
  });
  return { promise, resolve, reject };
}
beforeEach(() => {
  useScriptDraftStore.getState().clearAll();
  seed();
});
afterEach(cleanup);

describe("source Apply owns only its submitted draft", () => {
  it("keeps input typed while persistence is pending and does not start duplicate writes", async () => {
    const pending = deferred();
    const applyEntity = vi.fn(() => pending.promise);
    const { result } = renderHook(() =>
      useEditorWiring({
        workbench: "focus",
        entityId: 1,
        scriptPreview: "base",
        parsePreview: vi.fn(),
        applyEntity,
      }),
    );
    let saving!: Promise<void>;
    act(() => {
      saving = result.current.handleApplyEditor!();
    });
    act(() => {
      void result.current.handleApplyEditor!();
      result.current.setScriptDraft("newer input");
    });
    await act(async () => {
      pending.resolve();
      await saving;
    });
    expect(applyEntity).toHaveBeenCalledTimes(1);
    expect(useScriptDraftStore.getState().getDraft("focus:1")?.draft).toBe(
      "newer input",
    );
  });

  it("retains the submitted buffer on failure, but clears it after confirmed successful apply", async () => {
    const applyEntity = vi
      .fn()
      .mockRejectedValueOnce(new Error("HTTP 500"))
      .mockResolvedValueOnce({});
    const { result } = renderHook(() =>
      useEditorWiring({
        workbench: "focus",
        entityId: 1,
        scriptPreview: "base",
        parsePreview: vi.fn(),
        applyEntity,
      }),
    );
    await act(async () => {
      await result.current.handleApplyEditor!();
    });
    expect(useScriptDraftStore.getState().getDraft("focus:1")?.draft).toBe(
      "submitted",
    );
    await act(async () => {
      await result.current.handleApplyEditor!();
    });
    expect(useScriptDraftStore.getState().getDraft("focus:1")).toBeNull();
  });
});
