import { describe, expect, it } from "vitest";
import {
  fieldLocalMarkersForLintTarget,
  previewMarkersForLintTarget,
  type LintReport,
} from "./treeValidationMarkers";

const EMPTY_REPORT: LintReport = {
  focus: [],
  decision: [],
  event: [],
  idea: [],
  totals: {},
};

describe("fieldLocalMarkersForLintTarget", () => {
  it("keeps a legal same-parent inline swap marker-free", () => {
    expect(
      fieldLocalMarkersForLintTarget(EMPTY_REPORT, {
        tab: "focus",
        entityId: "USA_legal_swap",
        field: "completion_reward",
      }),
    ).toEqual([]);
  });

  it("carries each illegal child assignment's exact source span", () => {
    const report: LintReport = {
      ...EMPTY_REPORT,
      focus: [
        {
          level: "warning",
          code: "script.position.swap_ideas_child_outside_parent",
          msg: "add_idea is outside swap_ideas",
          entity_kind: "focus",
          entity_id: "USA_illegal_swap",
          field: "completion_reward",
          source_span: {
            start_offset: 61,
            end_offset: 83,
            start_line: 2,
            start_column: 5,
            end_line: 2,
            end_column: 27,
          },
        },
        {
          level: "warning",
          code: "script.position.swap_ideas_child_outside_parent",
          msg: "remove_idea is outside swap_ideas",
          entity_kind: "focus",
          entity_id: "USA_illegal_swap",
          field: "completion_reward",
          source_span: {
            start_offset: 88,
            end_offset: 113,
            start_line: 3,
            start_column: 5,
            end_line: 3,
            end_column: 30,
          },
        },
      ],
    };

    expect(
      fieldLocalMarkersForLintTarget(report, {
        tab: "focus",
        entityId: "USA_illegal_swap",
        field: "completion_reward",
      }),
    ).toEqual([
      {
        line: 2,
        startColumn: 5,
        endLine: 2,
        endColumn: 27,
        msg: "add_idea is outside swap_ideas",
        severity: "warning",
      },
      {
        line: 3,
        startColumn: 5,
        endLine: 3,
        endColumn: 30,
        msg: "remove_idea is outside swap_ideas",
        severity: "warning",
      },
    ]);
  });

  it("does not leak another focus or an unpositioned lint issue into markers", () => {
    const report: LintReport = {
      ...EMPTY_REPORT,
      focus: [
        {
          level: "warning",
          msg: "another focus",
          entity_kind: "focus",
          entity_id: "USA_other",
          field: "completion_reward",
          source_span: {
            start_offset: 0,
            end_offset: 8,
            start_line: 1,
            start_column: 1,
            end_line: 1,
            end_column: 9,
          },
        },
        {
          level: "warning",
          msg: "no source position",
          entity_kind: "focus",
          entity_id: "USA_target",
          field: "completion_reward",
        },
      ],
    };

    expect(
      fieldLocalMarkersForLintTarget(report, {
        tab: "focus",
        entityId: "USA_target",
        field: "completion_reward",
      }),
    ).toEqual([]);
  });

  it("projects field-local child spans into the normalized whole-focus preview", () => {
    const addAssignment = "add_idea = rogue_idea";
    const removeAssignment = "remove_idea = stale_idea";
    const fieldSource = [
      "swap_ideas = { add_idea = good remove_idea = old }",
      `    ${addAssignment}`,
      `\t${removeAssignment}`,
    ].join("\n");
    const addStart = fieldSource.indexOf(addAssignment);
    const removeStart = fieldSource.indexOf(removeAssignment);
    const report: LintReport = {
      ...EMPTY_REPORT,
      focus: [
        {
          level: "warning",
          msg: "add child outside parent",
          entity_kind: "focus",
          entity_id: "USA_illegal_swap",
          field: "completion_reward",
          source_span: {
            start_offset: addStart,
            end_offset: addStart + addAssignment.length,
            start_line: 2,
            start_column: 5,
            end_line: 2,
            end_column: 5 + addAssignment.length,
          },
        },
        {
          level: "warning",
          msg: "remove child outside parent",
          entity_kind: "focus",
          entity_id: "USA_illegal_swap",
          field: "completion_reward",
          source_span: {
            start_offset: removeStart,
            end_offset: removeStart + removeAssignment.length,
            start_line: 3,
            start_column: 2,
            end_line: 3,
            end_column: 2 + removeAssignment.length,
          },
        },
      ],
    };
    const scriptPreview = [
      "focus = {",
      "    id = USA_illegal_swap",
      "    completion_reward = {",
      "        swap_ideas = { add_idea = good remove_idea = old }",
      `        ${addAssignment}`,
      `            ${removeAssignment}`,
      "    }",
      "}",
    ].join("\n");

    expect(
      previewMarkersForLintTarget(
        report,
        {
          tab: "focus",
          entityId: "USA_illegal_swap",
          field: "completion_reward",
        },
        { fieldSource, scriptPreview },
      ),
    ).toEqual([
      {
        line: 5,
        startColumn: 9,
        endLine: 5,
        endColumn: 9 + addAssignment.length,
        msg: "add child outside parent",
        severity: "warning",
      },
      {
        line: 6,
        startColumn: 13,
        endLine: 6,
        endColumn: 13 + removeAssignment.length,
        msg: "remove child outside parent",
        severity: "warning",
      },
    ]);
  });

  it("fails closed when the whole preview has no unique field wrapper", () => {
    const report: LintReport = {
      ...EMPTY_REPORT,
      focus: [
        {
          level: "warning",
          msg: "positioned",
          entity_kind: "focus",
          entity_id: "USA_target",
          field: "completion_reward",
          source_span: {
            start_offset: 0,
            end_offset: 8,
            start_line: 1,
            start_column: 1,
            end_line: 1,
            end_column: 9,
          },
        },
      ],
    };
    const target = {
      tab: "focus" as const,
      entityId: "USA_target",
      field: "completion_reward",
    };

    expect(
      previewMarkersForLintTarget(report, target, {
        fieldSource: "add_idea = rogue",
        scriptPreview: "focus = { id = USA_target }",
      }),
    ).toEqual([]);
  });
});
