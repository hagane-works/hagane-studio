import { describe, expect, it } from "vitest";

import {
  beginDraftRequest,
  beginDraftSave,
  confirmDraftSelection,
  editDraftSession,
  emptyDraftSession,
  openDraftSession,
  requestDraftSelection,
  resolveDraftRefetch,
  resolveDraftSave,
  stableDraftFingerprint,
} from "../../lib/workbench-state/draftSession";

type Row = { id: string; title: string };

const row = (id: string, title: string): Row => ({ id, title });
const fingerprint = (value: Row) => stableDraftFingerprint(value);

function deferred<T>() {
  let resolve!: (value: T) => void;
  const promise = new Promise<T>((done) => {
    resolve = done;
  });
  return { promise, resolve };
}

describe("HistoryFilesContent async draft ownership", () => {
  it("does not replace dirty A with B until the deferred discard decision agrees", async () => {
    let session = openDraftSession(
      emptyDraftSession<Row>(),
      "A",
      fingerprint(row("A", "saved")),
      row("A", "saved"),
    );
    session = editDraftSession(
      session,
      row("A", "typed"),
      fingerprint(row("A", "typed")),
    );
    const requested = requestDraftSelection(
      session,
      "B",
      fingerprint(row("B", "saved")),
      row("B", "saved"),
    );
    expect(requested.kind).toBe("confirm-discard");

    const decision = deferred<boolean>();
    const selection = decision.promise.then((discard) =>
      discard
        ? confirmDraftSelection(
            session,
            "B",
            fingerprint(row("B", "saved")),
            row("B", "saved"),
          )
        : session,
    );
    decision.resolve(false);
    session = await selection;

    expect(session).toMatchObject({
      identity: "A",
      draft: row("A", "typed"),
      dirty: true,
    });
  });

  it("keeps edits typed after Save A began when its deferred response arrives", async () => {
    let session = openDraftSession(
      emptyDraftSession<Row>(),
      "A",
      fingerprint(row("A", "saved")),
      row("A", "saved"),
    );
    session = editDraftSession(
      session,
      row("A", "first edit"),
      fingerprint(row("A", "first edit")),
    );
    const save = beginDraftSave(session)!;
    const response = deferred<Row>();
    const completion = response.promise.then((saved) =>
      resolveDraftSave(session, save, fingerprint(saved), saved),
    );

    session = editDraftSession(
      session,
      row("A", "continue editing"),
      fingerprint(row("A", "continue editing")),
    );
    response.resolve(row("A", "first edit"));
    session = await completion;

    expect(session).toMatchObject({
      draft: row("A", "continue editing"),
      dirty: true,
    });
  });

  it("ignores a deferred A refetch after the creator moved to B", async () => {
    let session = openDraftSession(
      emptyDraftSession<Row>(),
      "A",
      fingerprint(row("A", "saved")),
      row("A", "saved"),
    );
    const request = beginDraftRequest(session)!;
    const response = deferred<Row>();
    const completion = response.promise.then((saved) =>
      resolveDraftRefetch(session, request, "A", fingerprint(saved), saved),
    );

    session = confirmDraftSelection(
      session,
      "B",
      fingerprint(row("B", "saved")),
      row("B", "saved"),
    );
    response.resolve(row("A", "late"));
    const result = await completion;

    expect(result.kind).toBe("ignored");
    expect(result.session.identity).toBe("B");
  });

  it("only treats a same-id Apply/refetch as an explicit draft replacement", async () => {
    let session = openDraftSession(
      emptyDraftSession<Row>(),
      "A",
      fingerprint(row("A", "saved")),
      row("A", "saved"),
    );
    session = editDraftSession(
      session,
      row("A", "typed"),
      fingerprint(row("A", "typed")),
    );
    const request = beginDraftRequest(session)!;
    const response = deferred<Row>();
    const completion = response.promise.then((applied) =>
      resolveDraftRefetch(session, request, "A", fingerprint(applied), applied),
    );

    response.resolve(row("A", "applied by AI"));
    const result = await completion;

    expect(result.kind).toBe("external-change-replaced-draft");
    expect(result.session).toMatchObject({
      draft: row("A", "applied by AI"),
      dirty: false,
    });
  });
});
