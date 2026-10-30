"use client";

import { useEffect, useRef, useState } from "react";
import { DEFAULT_LOCALE, type AppLocale } from "@/i18n/routing";

export type ImportStage = 1 | 2 | 3;

export interface ImportBucketCount {
  done: number;

  total: number;
}

export type ImportBucketKey =
  | "countries"
  | "characters"
  | "events"
  | "focus"
  | "decisions"
  | "ideas"
  | "dynamic_mod"
  | "superevents";

export interface ImportCounts {
  countries: ImportBucketCount | null;
  characters: ImportBucketCount | null;
  events: ImportBucketCount | null;
  focus: ImportBucketCount | null;
  decisions: ImportBucketCount | null;
  ideas: ImportBucketCount | null;
  dynamic_mod: ImportBucketCount | null;
  superevents: ImportBucketCount | null;
}

export interface ImportProgress {
  stage: ImportStage;
  stage_label?: string;

  phase?: string;
  percent: number;
  current_file: string | null;
  processed: number;
  total: number;
  throughput_per_sec: number | null;
  counts: ImportCounts;

  received_at_ms: number;
}

export type SSETransport = "sse" | "poll" | "idle" | "error";

const EMPTY_COUNTS: ImportCounts = {
  countries: null,
  characters: null,
  events: null,
  focus: null,
  decisions: null,
  ideas: null,
  dynamic_mod: null,
  superevents: null,
};

const EMPTY_PROGRESS: ImportProgress = {
  stage: 2,
  percent: 0,
  current_file: null,
  processed: 0,
  total: 0,
  throughput_per_sec: null,
  counts: EMPTY_COUNTS,
  received_at_ms: 0,
};

function stageStrToNum(stage: unknown): ImportStage | null {
  if (stage === "scan" || stage === "read" || stage === 1) return 1;
  if (stage === "parse" || stage === "write" || stage === 2) return 2;
  if (stage === "upload" || stage === "image" || stage === 3) return 3;
  return null;
}

function parseRate(rate: unknown): number | null {
  if (typeof rate === "number") return rate;
  if (typeof rate === "string") {
    const m = rate.match(/([\d.]+)/);
    if (m) {
      const n = Number(m[1]);
      if (!Number.isNaN(n)) return n;
    }
  }
  return null;
}

function reduceEvent(
  prev: ImportProgress,
  raw: unknown,
): {
  next: ImportProgress;
  terminal: "complete" | "error" | null;
  errorMessage?: string;
} {
  if (!raw || typeof raw !== "object") return { next: prev, terminal: null };
  const r = raw as Record<string, unknown>;
  const type = r.type;
  const next: ImportProgress = { ...prev, received_at_ms: Date.now() };

  switch (type) {
    case "stage": {
      const stageNum = stageStrToNum(r.stage);
      if (stageNum) next.stage = stageNum;
      if (typeof r.stage === "string") next.stage_label = r.stage;
      if (typeof r.phase === "string") next.phase = r.phase;

      if (typeof r.total === "number") next.total = r.total;
      if (typeof r.current === "number") next.processed = r.current;

      next.percent = 0;
      return { next, terminal: null };
    }
    case "file": {
      if (typeof r.name === "string") next.current_file = r.name;

      next.processed = prev.processed + 1;
      return { next, terminal: null };
    }
    case "progress": {
      if (typeof r.percent === "number") {
        next.percent = clamp(r.percent, 0, 100);
      }
      const rate = parseRate(r.rate);
      if (rate !== null) next.throughput_per_sec = rate;
      return { next, terminal: null };
    }
    case "counts": {
      next.counts = mergeFlatCounts(prev.counts, r);
      return { next, terminal: null };
    }
    case "complete": {
      next.percent = 100;
      return { next, terminal: "complete" };
    }
    case "error": {
      return {
        next,
        terminal: "error",
        errorMessage:
          typeof r.message === "string" ? r.message : "Import error",
      };
    }
    default:
      return { next: prev, terminal: null };
  }
}

function mergeFlatCounts(
  base: ImportCounts,
  raw: Record<string, unknown>,
): ImportCounts {
  const out: ImportCounts = { ...base };
  for (const key of Object.keys(out) as ImportBucketKey[]) {
    const v = raw[key];
    if (typeof v === "number") {
      out[key] = { done: v, total: 0 };
    } else if (v && typeof v === "object") {
      const o = v as Record<string, unknown>;
      if (typeof o.done === "number") {
        out[key] = {
          done: o.done,
          total: typeof o.total === "number" ? o.total : 0,
        };
      }
    }
  }
  return out;
}

function clamp(n: number, lo: number, hi: number): number {
  return Math.max(lo, Math.min(hi, n));
}

export interface UseImportProgressSSEResult {
  progress: ImportProgress | null;
  error: string | null;
  transport: SSETransport;
}

export function useImportProgressSSE(
  projectId: number,
  sessionId: string | null,
): UseImportProgressSSEResult {
  const [progress, setProgress] = useState<ImportProgress | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [transport, setTransport] = useState<SSETransport>("idle");

  const snapshotRef = useRef<ImportProgress>(EMPTY_PROGRESS);

  const activeRef = useRef<string | null>(null);

  const rafPendingRef = useRef<boolean>(false);
  const rafIdRef = useRef<number | null>(null);

  useEffect(() => {
    if (!sessionId) {
      activeRef.current = null;
      snapshotRef.current = EMPTY_PROGRESS;
      if (rafIdRef.current !== null && typeof window !== "undefined") {
        window.cancelAnimationFrame(rafIdRef.current);
        rafIdRef.current = null;
      }
      rafPendingRef.current = false;
      setProgress(null);
      setError(null);
      setTransport("idle");
      return;
    }

    activeRef.current = sessionId;
    snapshotRef.current = { ...EMPTY_PROGRESS };
    rafPendingRef.current = false;
    setProgress(null);
    setError(null);

    const scheduleCommit = () => {
      if (rafPendingRef.current) return;
      if (typeof window === "undefined") {
        setProgress(snapshotRef.current);
        return;
      }
      rafPendingRef.current = true;
      rafIdRef.current = window.requestAnimationFrame(() => {
        rafPendingRef.current = false;
        rafIdRef.current = null;
        if (activeRef.current !== sessionId) return;
        setProgress(snapshotRef.current);
      });
    };

    const flushCommit = () => {
      if (rafIdRef.current !== null && typeof window !== "undefined") {
        window.cancelAnimationFrame(rafIdRef.current);
        rafIdRef.current = null;
      }
      rafPendingRef.current = false;
      setProgress(snapshotRef.current);
    };

    const applyRaw = (raw: unknown) => {
      if (activeRef.current !== sessionId) return;
      const { next, terminal, errorMessage } = reduceEvent(
        snapshotRef.current,
        raw,
      );
      snapshotRef.current = next;
      if (terminal) {
        flushCommit();
        if (terminal === "error" && errorMessage) setError(errorMessage);
      } else {
        scheduleCommit();
      }
    };

    const stopPolling = startPolling(sessionId);
    return () => {
      stopPolling();
      if (rafIdRef.current !== null && typeof window !== "undefined") {
        window.cancelAnimationFrame(rafIdRef.current);
        rafIdRef.current = null;
      }
      rafPendingRef.current = false;
    };

    function startPolling(sid: string): () => void {
      setTransport("poll");
      let cancelled = false;
      let timer: ReturnType<typeof setTimeout> | null = null;

      const tick = async () => {
        if (cancelled || activeRef.current !== sid) return;
        try {
          const res = await fetch(
            `/api/go/import/jobs/${projectId}/${encodeURIComponent(sid)}`,
            { cache: "no-store", credentials: "same-origin" },
          );

          if (res.ok) {
            const data = (await res.json()) as {
              status?: string;
              phase?: string;
              files_done?: number;
              files_total?: number;
              percent?: number;
              error?: string;
            };
            if (!cancelled && activeRef.current === sid) {
              applyRaw({
                type: "stage",
                stage: data.phase === "images" ? "upload" : "parse",
                phase: data.phase,
                current: data.files_done ?? 0,
                total: data.files_total ?? 0,
              });
              const total = data.files_total ?? 0;
              applyRaw({
                type: "progress",
                percent:
                  typeof data.percent === "number"
                    ? data.percent
                    : total > 0
                      ? ((data.files_done ?? 0) * 100) / total
                      : 0,
              });
              if (data.status === "complete") applyRaw({ type: "complete" });
              if (data.status === "failed" || data.status === "cancelled") {
                applyRaw({
                  type: "error",
                  message: data.error ?? `Go import ${data.status}`,
                });
              }
            }
          }
        } catch (err) {
          if (!cancelled && activeRef.current === sid) {
            setError(err instanceof Error ? err.message : "Polling error");
          }
        } finally {
          if (!cancelled && activeRef.current === sid) {
            timer = setTimeout(tick, 2000);
          }
        }
      };

      void tick();

      return () => {
        cancelled = true;
        if (timer) clearTimeout(timer);
      };
    }
  }, [projectId, sessionId]);

  return { progress, error, transport };
}

export function estimateRemainingMs(
  percent: number,
  elapsedMs: number,
): number | null {
  if (percent < 2 || percent >= 100 || elapsedMs <= 0) return null;
  const totalEstimateMs = elapsedMs / (percent / 100);
  const remaining = totalEstimateMs - elapsedMs;
  return remaining > 0 ? remaining : null;
}

export function formatDuration(
  ms: number,
  locale: AppLocale = DEFAULT_LOCALE,
): string {
  const seconds = new Intl.NumberFormat(locale, {
    style: "unit",
    unit: "second",
    unitDisplay: "short",
  });
  const minutes = new Intl.NumberFormat(locale, {
    style: "unit",
    unit: "minute",
    unitDisplay: "short",
  });
  if (ms < 1000) return `<${seconds.format(1)}`;
  const totalSec = Math.round(ms / 1000);
  const min = Math.floor(totalSec / 60);
  const sec = totalSec % 60;
  if (min === 0) return seconds.format(sec);
  if (sec === 0) return minutes.format(min);
  return `${minutes.format(min)} ${seconds.format(sec)}`;
}
