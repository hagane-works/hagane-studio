import { useState } from "react";
import { afterEach, describe, expect, it } from "vitest";
import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import {
  addHistoryDateBlock,
  deleteHistoryDateBlock,
  HistoryDateBlocksEditor,
  moveHistoryDateBlock,
  validateHistoryDate,
  type HistoryDateBlock,
  type HistoryDateBlocksEditorLabels,
} from "./HistoryDateBlocksEditor";

const rows: HistoryDateBlock[] = [
  { date: "1939.1.1", script: "first = yes" },
  { date: "1939.1.1", script: "second = yes" },
  { date: "1940.6.15.12", script: "third = yes" },
];

const labels: HistoryDateBlocksEditorLabels = {
  title: "Start-date changes",
  description: "Ordered history blocks",
  add: "Add date block",
  empty: "No date blocks",
  date: "Date",
  script: "Script",
  moveUp: "Move up",
  moveDown: "Move down",
  delete: "Delete",
  validation: {
    format: "Format error",
    range: "Range error",
    hour: "Hour error",
  },
};

function EditorHarness({ initial = rows }: { initial?: HistoryDateBlock[] }) {
  const [blocks, setBlocks] = useState(initial);
  return (
    <HistoryDateBlocksEditor
      blocks={blocks}
      onChange={setBlocks}
      labels={labels}
    />
  );
}

function ParentDraftHarness() {
  const [draft, setDraft] = useState({
    capital: 121,
    raw_script: "custom_unmapped = { keep = yes }",
    date_blocks: rows,
  });
  return (
    <>
      <output aria-label="capital">{draft.capital}</output>
      <output aria-label="raw script">{draft.raw_script}</output>
      <HistoryDateBlocksEditor
        blocks={draft.date_blocks}
        onChange={(date_blocks) =>
          setDraft((current) => ({ ...current, date_blocks }))
        }
        labels={labels}
      />
    </>
  );
}

afterEach(cleanup);

describe("validateHistoryDate", () => {
  it("accepts date-only and date-hour HOI4 tokens", () => {
    expect(validateHistoryDate("1936.1.1")).toBeNull();
    expect(validateHistoryDate("1939.1.1.12")).toBeNull();
  });

  it("rejects malformed and out-of-range tokens", () => {
    expect(validateHistoryDate("1936.1")).not.toBeNull();
    expect(validateHistoryDate("1936.13.1")).not.toBeNull();
    expect(validateHistoryDate("1936.1.1.24")).not.toBeNull();
    expect(validateHistoryDate("1936.1.0")).toBe("range");
    expect(validateHistoryDate("not-a-date")).toBe("format");
  });
});

describe("date-block positional helpers", () => {
  it("appends without sorting or deduplicating", () => {
    const next = addHistoryDateBlock(rows, {
      date: "1939.1.1",
      script: "fourth = yes",
    });
    expect(next.map((row) => row.script)).toEqual([
      "first = yes",
      "second = yes",
      "third = yes",
      "fourth = yes",
    ]);
    expect(next.filter((row) => row.date === "1939.1.1")).toHaveLength(3);
    expect(rows).toHaveLength(3);
  });

  it("reorders and deletes exactly one duplicate by position", () => {
    const moved = moveHistoryDateBlock(rows, 2, -2);
    expect(moved.map((row) => row.script)).toEqual([
      "third = yes",
      "first = yes",
      "second = yes",
    ]);
    const deleted = deleteHistoryDateBlock(moved, 1);
    expect(deleted.map((row) => row.script)).toEqual([
      "third = yes",
      "second = yes",
    ]);
    expect(rows.map((row) => row.script)).toEqual([
      "first = yes",
      "second = yes",
      "third = yes",
    ]);
  });

  it("returns a new list for boundary moves", () => {
    const next = moveHistoryDateBlock(rows, 0, -1);
    expect(next).toEqual(rows);
    expect(next).not.toBe(rows);
  });
});

describe("HistoryDateBlocksEditor", () => {
  it("drives add, reorder, and positional delete without collapsing duplicates", () => {
    render(<EditorHarness />);

    fireEvent.click(
      screen.getAllByRole("button", { name: labels.moveDown })[0],
    );
    expect(
      screen
        .getAllByLabelText(labels.script)
        .map((node) => (node as HTMLTextAreaElement).value),
    ).toEqual(["second = yes", "first = yes", "third = yes"]);

    fireEvent.click(screen.getAllByRole("button", { name: labels.delete })[1]);
    expect(
      screen
        .getAllByLabelText(labels.script)
        .map((node) => (node as HTMLTextAreaElement).value),
    ).toEqual(["second = yes", "third = yes"]);

    fireEvent.click(screen.getByRole("button", { name: labels.add }));
    expect(screen.getAllByLabelText(labels.date)).toHaveLength(3);
    expect(
      (screen.getAllByLabelText(labels.date)[2] as HTMLInputElement).value,
    ).toBe("1936.1.1");
  });

  it("warns inline while preserving the author's typed token", () => {
    render(<EditorHarness />);
    const firstDate = screen.getAllByLabelText(labels.date)[0];

    fireEvent.change(firstDate, { target: { value: "1936.13.1" } });

    expect((firstDate as HTMLInputElement).value).toBe("1936.13.1");
    expect(screen.getByText(labels.validation.range)).toBeTruthy();
    expect(firstDate.getAttribute("aria-invalid")).toBe("true");
  });

  it("renders an explicit empty state and one-command add path", () => {
    render(<EditorHarness initial={[]} />);
    expect(screen.getByText(labels.empty)).toBeTruthy();
    fireEvent.click(screen.getByRole("button", { name: labels.add }));
    expect(screen.queryByText(labels.empty)).toBeNull();
    expect(screen.getAllByLabelText(labels.date)).toHaveLength(1);
  });

  it("updates only date_blocks in a parent draft", () => {
    render(<ParentDraftHarness />);

    fireEvent.click(
      screen.getAllByRole("button", { name: labels.moveDown })[0],
    );
    fireEvent.change(screen.getAllByLabelText(labels.script)[0], {
      target: { value: "edited duplicate = yes" },
    });

    expect(screen.getByLabelText("capital").textContent).toBe("121");
    expect(screen.getByLabelText("raw script").textContent).toBe(
      "custom_unmapped = { keep = yes }",
    );
    expect(
      screen
        .getAllByLabelText(labels.script)
        .map((node) => (node as HTMLTextAreaElement).value),
    ).toEqual(["edited duplicate = yes", "first = yes", "third = yes"]);
  });
});
