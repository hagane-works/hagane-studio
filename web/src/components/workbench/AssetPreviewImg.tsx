"use client";

import { useState } from "react";
import { useAssetPreviewUrl } from "@/lib/store/assetPreviewVersionStore";

export function AssetPreviewImg({
  url,
  fallbackUrls = [],
  alt = "",
  className,
}: {
  url: string | null;
  fallbackUrls?: string[];
  alt?: string;
  className?: string;
}) {
  const src = useAssetPreviewUrl(url);
  if (!src) return null;
  const candidates = [src, ...fallbackUrls];
  return (
    <FirstLoadingImg
      key={candidates.join(" ")}
      candidates={candidates}
      alt={alt}
      className={className}
    />
  );
}

function FirstLoadingImg({
  candidates,
  alt,
  className,
}: {
  candidates: string[];
  alt: string;
  className?: string;
}) {
  const [index, setIndex] = useState(0);
  if (index >= candidates.length) return null;
  return (
    <img
      key={index}
      src={candidates[index]}
      alt={alt}
      className={className}
      onError={() => setIndex(index + 1)}
    />
  );
}
