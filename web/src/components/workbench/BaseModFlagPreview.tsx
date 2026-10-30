"use client";

import { useMemo, useState, type ReactNode } from "react";
import { slugify } from "@/lib/wiki";

export type FlagSize = "large" | "medium" | "small";

const VANILLA_FLAG_IDEOLOGY_GROUPS = new Set([
  "democratic",
  "fascism",
  "communism",
  "neutrality",
]);

export function baseModFlagUrl(
  projectId: number,
  depKey: string,
  tag: string,
  ideology?: string,
  size: FlagSize = "large",
  effectiveRootIds?: readonly string[],
): string {
  const q = new URLSearchParams();
  const allowedRoots = effectiveRootIds
    ? new Set(effectiveRootIds.filter(Boolean))
    : VANILLA_FLAG_IDEOLOGY_GROUPS;
  if (ideology && allowedRoots.has(ideology)) q.set("ideology", ideology);
  if (size && size !== "large") q.set("size", size);
  const qs = q.toString();
  return `/api/go/projects/${projectId}/basemod/${encodeURIComponent(depKey)}/flag/${encodeURIComponent(
    tag,
  )}${qs ? `?${qs}` : ""}`;
}

export function firstBaseModDepKey(
  dependencies?: string[] | null,
): string | null {
  const first = (dependencies || []).map((d) => (d || "").trim()).find(Boolean);
  return first ? slugify(first) : null;
}

interface BaseModFlagPreviewProps {
  projectId: number;
  depKey: string;
  tag: string;

  ideology?: string;
  size?: FlagSize;
  width?: number;
  height?: number;
  className?: string;
  alt?: string;
  title?: string;

  effectiveRootIds?: readonly string[];

  placeholder: ReactNode;
}

export function BaseModFlagPreview({
  projectId,
  depKey,
  tag,
  ideology,
  size = "large",
  width = 82,
  height = 52,
  className,
  alt,
  title,
  effectiveRootIds,
  placeholder,
}: BaseModFlagPreviewProps) {
  const candidates = useMemo(() => {
    const urls: string[] = [];
    const effectiveRoots = effectiveRootIds
      ? new Set(effectiveRootIds.filter(Boolean))
      : VANILLA_FLAG_IDEOLOGY_GROUPS;
    if (ideology && effectiveRoots.has(ideology)) {
      urls.push(
        baseModFlagUrl(
          projectId,
          depKey,
          tag,
          ideology,
          size,
          effectiveRootIds,
        ),
      );
    }
    urls.push(
      baseModFlagUrl(projectId, depKey, tag, undefined, size, effectiveRootIds),
    );
    return urls;
  }, [projectId, depKey, tag, ideology, size, effectiveRootIds]);

  const [idx, setIdx] = useState(0);

  if (idx >= candidates.length) return <>{placeholder}</>;

  return (
    <img
      src={candidates[idx]}
      alt={alt ?? `${tag} flag`}
      title={title}
      width={width}
      height={height}
      className={className}
      onError={() => setIdx((i) => i + 1)}
    />
  );
}
