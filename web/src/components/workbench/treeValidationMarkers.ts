import type { EditorMarker } from "@/shared/editors/editorMarkers";

export type LintLevel = "error" | "warning" | "info";
export type LintTab = "focus" | "decision" | "event" | "idea";

export interface LintSourceSpan {
  start_offset: number;
  end_offset: number;
  start_line: number;
  start_column: number;
  end_line: number;
  end_column: number;
}

export interface LintIssue {
  level: LintLevel;
  code?: string;
  msg: string;
  suggested_fix?: string;
  entity_kind: string;
  entity_id: string;
  field?: string;
  source_span?: LintSourceSpan;
  parent_path?: string[];
  repair_route?: string;

  entity_api_id?: number;
  target_id?: string;
  boolean_ancestry?: string[];
  replacement_suggestion?: string | null;
}

export type CrossRefRepairLintIssue = LintIssue & {
  repair_route: "cross_ref.explicit_repair";
  entity_kind: "focus" | "event" | "decision" | "idea";
  entity_api_id: number;
  field: string;
  target_id: string;
  source_span: LintSourceSpan;
};

const CROSS_REF_ENTITY_KINDS = new Set(["focus", "event", "decision", "idea"]);

export function asCrossRefRepairIssue(
  issue: LintIssue,
): CrossRefRepairLintIssue | null {
  const span = issue.source_span;
  if (
    issue.repair_route !== "cross_ref.explicit_repair" ||
    !CROSS_REF_ENTITY_KINDS.has(issue.entity_kind) ||
    !Number.isInteger(issue.entity_api_id) ||
    (issue.entity_api_id ?? 0) <= 0 ||
    !issue.field ||
    !issue.target_id?.trim() ||
    !span ||
    !Number.isInteger(span.start_offset) ||
    !Number.isInteger(span.end_offset) ||
    span.start_offset < 0 ||
    span.end_offset <= span.start_offset
  ) {
    return null;
  }
  return issue as CrossRefRepairLintIssue;
}

export interface LintReport {
  focus: LintIssue[];
  decision: LintIssue[];
  event: LintIssue[];
  idea: LintIssue[];
  totals: { error?: number; warning?: number; info?: number };
}

export interface LintMarkerTarget {
  tab: LintTab;
  entityId: string;
  field: string;
}

export interface LintPreviewProjection {
  fieldSource: string;

  scriptPreview: string;
}

function matchingIssues(
  report: LintReport | null | undefined,
  target: LintMarkerTarget,
): LintIssue[] {
  if (!report) return [];
  return (report[target.tab] ?? []).filter(
    (issue) =>
      issue.entity_kind === target.tab &&
      issue.entity_id === target.entityId &&
      issue.field === target.field &&
      issue.source_span !== undefined,
  );
}

function markerForIssue(issue: LintIssue): EditorMarker {
  const span = issue.source_span!;
  return {
    line: span.start_line,
    startColumn: span.start_column,
    endLine: span.end_line,
    endColumn: span.end_column,
    msg: issue.msg,
    severity: issue.level,
  };
}

export function fieldLocalMarkersForLintTarget(
  report: LintReport | null | undefined,
  target: LintMarkerTarget,
): EditorMarker[] {
  return matchingIssues(report, target).map(markerForIssue);
}

function expandedWidth(text: string): number {
  return text.replaceAll("\t", "    ").length;
}

function projectPoint(
  sourceLines: string[],
  sourceLine: number,
  sourceColumn: number,
  wrapperLineIndex: number,
  baseIndent: number,
): { line: number; column: number } | null {
  const rawLine = sourceLines[sourceLine - 1];
  if (rawLine === undefined || rawLine.trim().length === 0) return null;
  const contentStart = rawLine.search(/\S/);
  const leadingTabs = rawLine.length - rawLine.replace(/^\t+/, "").length;
  const sourceIndex = Math.max(
    contentStart,
    Math.min(sourceColumn - 1, rawLine.length),
  );
  const priorEmittedLines = sourceLines
    .slice(0, sourceLine - 1)
    .filter((line) => line.trim().length > 0).length;
  const contentPrefix = rawLine.slice(contentStart, sourceIndex);
  return {
    line: wrapperLineIndex + priorEmittedLines + 2,
    column: baseIndent + leadingTabs * 4 + expandedWidth(contentPrefix) + 1,
  };
}

export function previewMarkersForLintTarget(
  report: LintReport | null | undefined,
  target: LintMarkerTarget,
  projection: LintPreviewProjection,
): EditorMarker[] {
  const issues = matchingIssues(report, target);
  if (issues.length === 0) return [];

  const previewLines = projection.scriptPreview.split("\n");
  const escapedField = target.field.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
  const wrapperPattern = new RegExp(`^(\\s*)${escapedField}\\s*=\\s*\\{\\s*$`);
  const wrappers = previewLines
    .map((line, index) => ({ match: wrapperPattern.exec(line), index }))
    .filter((entry) => entry.match !== null);
  if (wrappers.length !== 1) return [];

  const wrapper = wrappers[0];
  const baseIndent = expandedWidth(wrapper.match![1]) + 4;
  const sourceLines = projection.fieldSource.split("\n");
  return issues.flatMap((issue) => {
    const span = issue.source_span!;
    const start = projectPoint(
      sourceLines,
      span.start_line,
      span.start_column,
      wrapper.index,
      baseIndent,
    );
    const end = projectPoint(
      sourceLines,
      span.end_line,
      span.end_column,
      wrapper.index,
      baseIndent,
    );
    if (!start || !end) return [];
    return [
      {
        line: start.line,
        startColumn: start.column,
        endLine: end.line,
        endColumn: end.column,
        msg: issue.msg,
        severity: issue.level,
      },
    ];
  });
}
