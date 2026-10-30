export interface SaveDiagnostic {
  key: string;
  code: string;
  severity: string;
  entity_ref: { type: string; id: string };
  evidence: Record<string, unknown>;
  repair_route: string;
  affected: string[];
  change?:
    | "introduced"
    | "unchanged"
    | "improved"
    | "worsened"
    | "changed"
    | "resolved";
  source?: unknown;
}

function record(value: unknown): value is Record<string, unknown> {
  return value !== null && typeof value === "object" && !Array.isArray(value);
}

const changes = new Set([
  "introduced",
  "unchanged",
  "improved",
  "worsened",
  "changed",
  "resolved",
]);
function diagnostic(value: unknown): value is SaveDiagnostic {
  return (
    record(value) &&
    typeof value.key === "string" &&
    typeof value.code === "string" &&
    typeof value.severity === "string" &&
    record(value.entity_ref) &&
    typeof value.entity_ref.type === "string" &&
    typeof value.entity_ref.id === "string" &&
    record(value.evidence) &&
    typeof value.repair_route === "string" &&
    Array.isArray(value.affected) &&
    value.affected.every((item) => typeof item === "string") &&
    (value.change === undefined ||
      (typeof value.change === "string" && changes.has(value.change)))
  );
}

export function readSaveDiagnostics(
  payload: unknown,
): SaveDiagnostic[] | undefined {
  const seen = new Set<unknown>();
  function read(value: unknown): SaveDiagnostic[] | undefined {
    if (!record(value) || seen.has(value)) return undefined;
    seen.add(value);
    if (Object.hasOwn(value, "diagnostics")) {
      return Array.isArray(value.diagnostics) &&
        value.diagnostics.every(diagnostic)
        ? value.diagnostics
        : undefined;
    }
    return read(value.payload) ?? read(value.detail);
  }
  return read(payload);
}

export function currentSaveDiagnostics(
  diagnostics: readonly SaveDiagnostic[],
): SaveDiagnostic[] {
  return diagnostics.filter((item) => item.change !== "resolved");
}

export function mergeSaveDiagnostics(
  previous: readonly SaveDiagnostic[] | undefined,
  incoming: readonly SaveDiagnostic[] | undefined,
): readonly SaveDiagnostic[] | undefined {
  return incoming === undefined ? previous : currentSaveDiagnostics(incoming);
}

export function withoutSaveDiagnostics<T extends object>(value: T): T {
  const copy = { ...value };
  delete (copy as Record<string, unknown>).diagnostics;
  delete (copy as Record<string, unknown>).tree_diagnostics;
  return copy;
}
