export interface EditorMarker {
  line: number;
  msg: string;
  severity: "error" | "warning" | "info";
  startColumn?: number;
  endLine?: number;
  endColumn?: number;
}

export interface MarkerTextModel {
  getLineCount: () => number;
  getLineLength: (lineNumber: number) => number;
}

export interface EditorMarkerRange {
  startLineNumber: number;
  startColumn: number;
  endLineNumber: number;
  endColumn: number;
}

function clamp(value: number, lower: number, upper: number): number {
  return Math.max(lower, Math.min(value, upper));
}

export function editorMarkerRange(
  marker: EditorMarker,
  model: MarkerTextModel,
): EditorMarkerRange {
  const lineCount = Math.max(1, model.getLineCount());
  const startLineNumber = clamp(marker.line, 1, lineCount);
  const requestedEndLine = marker.endLine ?? startLineNumber;
  const endLineNumber = clamp(
    Math.max(startLineNumber, requestedEndLine),
    startLineNumber,
    lineCount,
  );
  const startMax = model.getLineLength(startLineNumber) + 1;
  const endMax = model.getLineLength(endLineNumber) + 1;
  const startColumn = clamp(marker.startColumn ?? 1, 1, startMax);

  const requestedEndColumn = marker.endColumn ?? endMax;
  const endColumn = clamp(
    endLineNumber === startLineNumber
      ? Math.max(startColumn, requestedEndColumn)
      : requestedEndColumn,
    1,
    endMax,
  );

  return {
    startLineNumber,
    startColumn,
    endLineNumber,
    endColumn,
  };
}
