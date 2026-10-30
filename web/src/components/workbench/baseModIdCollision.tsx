"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { useTranslations } from "use-intl";
import { useProjectStore } from "@/lib/store/projectStore";

export type BaseModIdEntityType = "character" | "focus" | "idea" | "decision";

export interface BaseModIdCollision {
  id: string;
  dependency: string;
}

export const BASE_MOD_ID_DEBOUNCE_MS = 300;

interface IdTakenResponse {
  indexed?: unknown;
  taken?: unknown;
  dependency?: unknown;
}

export async function fetchBaseModIdCollision(
  projectId: number,
  entityType: BaseModIdEntityType,
  id: string,
  signal?: AbortSignal,
): Promise<BaseModIdCollision | null> {
  const trimmed = id.trim();
  if (!projectId || !trimmed) return null;
  const query = new URLSearchParams({
    project_id: String(projectId),
    type: entityType,
    id: trimmed,
  });
  try {
    const response = await fetch(
      `/api/go/basemod/id-taken?${query.toString()}`,
      {
        method: "GET",
        credentials: "include",
        signal,
      },
    );
    if (!response.ok) return null;
    const body = (await response.json()) as IdTakenResponse;
    if (body.indexed !== true || body.taken !== true) return null;
    const dependency =
      typeof body.dependency === "string" ? body.dependency.trim() : "";
    return { id: trimmed, dependency };
  } catch {
    return null;
  }
}

export interface UseBaseModIdCollisionResult {
  collision: BaseModIdCollision | null;

  verify: (candidateId: string) => Promise<BaseModIdCollision | null>;
}

export function useBaseModIdCollision(
  entityType: BaseModIdEntityType,
  id: string,
): UseBaseModIdCollisionResult {
  const projectId = useProjectStore((s) => s.currentProject?.id);
  const dependencyCount = useProjectStore(
    (s) => s.currentProject?.dependencies?.length ?? 0,
  );

  const [verdict, setVerdict] = useState<BaseModIdCollision | null>(null);
  const trimmedId = id.trim();

  const collision =
    verdict !== null && verdict.id === trimmedId ? verdict : null;

  const seqRef = useRef(0);
  const abortRef = useRef<AbortController | null>(null);

  const mountIdRef = useRef(trimmedId);
  const dirtyRef = useRef(false);
  const mountedRef = useRef(true);

  useEffect(() => {
    mountedRef.current = true;
    return () => {
      mountedRef.current = false;
      abortRef.current?.abort();
    };
  }, []);

  const run = useCallback(
    async (candidate: string): Promise<BaseModIdCollision | null> => {
      const seq = ++seqRef.current;
      abortRef.current?.abort();
      const controller = new AbortController();
      abortRef.current = controller;
      const hit = await fetchBaseModIdCollision(
        projectId ?? 0,
        entityType,
        candidate,
        controller.signal,
      );

      if (seq !== seqRef.current) return hit;
      if (mountedRef.current) setVerdict(hit);
      return hit;
    },
    [entityType, projectId],
  );

  useEffect(() => {
    if (!dirtyRef.current) {
      if (trimmedId === mountIdRef.current) return;
      dirtyRef.current = true;
    }

    if (!projectId || dependencyCount === 0 || !trimmedId) {
      seqRef.current++;
      abortRef.current?.abort();
      return;
    }
    const timer = setTimeout(() => {
      void run(trimmedId);
    }, BASE_MOD_ID_DEBOUNCE_MS);
    return () => clearTimeout(timer);
  }, [trimmedId, projectId, dependencyCount, run]);

  const verify = useCallback(
    async (candidateId: string): Promise<BaseModIdCollision | null> => {
      const candidate = candidateId.trim();
      if (!projectId || dependencyCount === 0 || !candidate) return null;
      dirtyRef.current = true;
      return run(candidate);
    },
    [projectId, dependencyCount, run],
  );

  return { collision, verify };
}

const ENTITY_LABEL_KEY: Record<BaseModIdEntityType, string> = {
  character: "entityCharacter",
  focus: "entityFocus",
  idea: "entityIdea",
  decision: "entityDecision",
};

export function BaseModIdCollisionNotice({
  entityType,
  collision,
}: {
  entityType: BaseModIdEntityType;
  collision: BaseModIdCollision | null;
}) {
  const t = useTranslations("baseModIdCollision");
  if (!collision) return null;
  const dependency = collision.dependency || t("unknownDependency");
  const entityLabel = t(ENTITY_LABEL_KEY[entityType]);
  return (
    <div
      role="alert"
      data-testid="basemod-id-collision"
      className="mt-1 space-y-0.5"
    >
      <p className="text-xs text-destructive">
        {t("message", { id: collision.id, dependency, entityLabel })}
      </p>
      <p className="text-xs text-muted-foreground">
        {t("fixHint", { dependency })}
      </p>
    </div>
  );
}
