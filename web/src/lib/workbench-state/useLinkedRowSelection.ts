"use client";

import { useEffect, useRef } from "react";
import { useSearchParams } from "@/platform/navigation";

export function useLinkedRowSelection<R extends { id: number }>(
  param: string,
  rows: readonly R[],
  select: (row: R) => unknown,
): void {
  const searchParams = useSearchParams();
  const pending = useRef<number | null>(
    Number(searchParams?.get(param)) || null,
  );
  useEffect(() => {
    const wanted = pending.current;
    if (!wanted) return;
    const row = rows.find((r) => r.id === wanted);
    if (!row) return;
    pending.current = null;
    queueMicrotask(() => void select(row));
  }, [rows, select]);
}
