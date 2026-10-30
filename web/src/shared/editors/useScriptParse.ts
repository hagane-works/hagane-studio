"use client";

import { useCallback, useRef } from "react";
import {
  useScriptDraftStore,
  draftKey,
  type ParseStatus,
} from "./scriptDraftStore";

export interface ParsePreviewApiResponse {
  entity: Record<string, unknown> | null;
  extras: Record<string, unknown>;
  warnings: string[];
  parse_errors: {
    line: number;
    msg: string;
    severity: "error" | "warning" | "info";
  }[];
}

interface UseScriptParseOptions {
  workbench: string;
  entityId: number | null | undefined;

  parsePreview: (script: string) => Promise<ParsePreviewApiResponse>;
}

export function useScriptParse({
  workbench,
  entityId,
  parsePreview,
}: UseScriptParseOptions) {
  const key = draftKey(workbench, entityId);
  const markParsing = useScriptDraftStore((s) => s.markParsing);
  const setParseResult = useScriptDraftStore((s) => s.setParseResult);
  const getDraft = useScriptDraftStore((s) => s.getDraft);

  const latestRequest = useRef(0);

  const triggerParse = useCallback(async () => {
    if (!key) return;
    const entry = getDraft(key);
    if (!entry) return;
    if (entry.draft === entry.baselineScript) return;

    markParsing(key);
    const reqId = ++latestRequest.current;

    let response: ParsePreviewApiResponse;
    try {
      response = await parsePreview(entry.draft);
    } catch (err) {
      if (reqId !== latestRequest.current) return;
      setParseResult(key, {
        status: "error",
        errors: [
          {
            line: 1,
            msg: err instanceof Error ? err.message : "Parse request failed",
            severity: "error",
          },
        ],
        entity: null,
        extras: null,
        warnings: [],
      });
      return;
    }
    if (reqId !== latestRequest.current) return;

    const errs = response.parse_errors ?? [];
    let status: ParseStatus = "ok";
    if (errs.length > 0) {
      status = errs.some((e) => e.severity === "error") ? "error" : "info";
    }

    setParseResult(key, {
      status,
      errors: errs,
      entity: response.entity,
      extras: response.extras,
      warnings: response.warnings ?? [],
    });
  }, [key, getDraft, markParsing, setParseResult, parsePreview]);

  return { triggerParse };
}
