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
} from "./draftSession";

type Row = { id: string; title: string };
const fingerprint = (row: Row) => stableDraftFingerprint(row);
const row = (id: string, title: string): Row => ({ id, title });

describe("DraftSession async ownership", () => {
  it("requires an explicit discard before dirty A can select B", async () => {
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
    session = confirmDraftSelection(
      session,
      "B",
      fingerprint(row("B", "saved")),
      row("B", "saved"),
    );
    expect(session).toMatchObject({
      identity: "B",
      dirty: false,
      draft: row("B", "saved"),
    });
  });

  it("keeps edits made after Save A started", async () => {
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
    session = editDraftSession(
      session,
      row("A", "continue editing"),
      fingerprint(row("A", "continue editing")),
    );
    session = resolveDraftSave(
      session,
      save,
      fingerprint(row("A", "first edit")),
      row("A", "first edit"),
    );
    expect(session).toMatchObject({
      draft: row("A", "continue editing"),
      dirty: true,
    });
  });

  it("ignores A's late request after selection moved to B", async () => {
    let session = openDraftSession(
      emptyDraftSession<Row>(),
      "A",
      fingerprint(row("A", "saved")),
      row("A", "saved"),
    );
    const request = beginDraftRequest(session)!;
    session = confirmDraftSelection(
      session,
      "B",
      fingerprint(row("B", "saved")),
      row("B", "saved"),
    );
    const completed = resolveDraftRefetch(
      session,
      request,
      "A",
      fingerprint(row("A", "late")),
      row("A", "late"),
    );
    expect(completed.kind).toBe("ignored");
    expect(completed.session.identity).toBe("B");
  });

  it("replaces a same-id dirty draft only as an explicit Apply/refetch conflict", async () => {
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
    const completed = resolveDraftRefetch(
      session,
      request,
      "A",
      fingerprint(row("A", "applied")),
      row("A", "applied"),
    );
    expect(completed.kind).toBe("external-change-replaced-draft");
    expect(completed.session).toMatchObject({
      draft: row("A", "applied"),
      dirty: false,
    });
  });
});
