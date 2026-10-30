"use client";

import { BaseModFlagPreview } from "@/components/workbench/BaseModFlagPreview";
import { countriesApi, type CountryRow } from "@/lib/api";

interface CountryFlagPreviewProps {
  projectId: number;
  country: CountryRow;
  root?: string;
  roots: readonly string[];
  baseModDepKey: string | null;
  bust?: number;
  className?: string;
}
export function CountryFlagPreview({
  projectId,
  country,
  root,
  roots,
  baseModDepKey,
  bust = 0,
  className,
}: CountryFlagPreviewProps) {
  const flagClassName = `aspect-[82/52] object-cover ${className ?? ""}`;
  const hasVariant = Boolean(root && country.flag_paths_by_ideology?.[root]);
  const hasBase = Boolean(country.flag_image_path);
  const directIdeology = hasVariant ? root : undefined;
  if (hasVariant || hasBase) {
    const previewUrl = countriesApi.flagPreviewUrl(
      projectId,
      country.id,
      directIdeology,
    );
    return (
      <img
        key={`${country.id}-${root ?? "base"}-${bust}`}
        src={`${previewUrl}${previewUrl.includes("?") ? "&" : "?"}v=${bust}`}
        alt={`${country.tag} ${root ?? "default"} flag`}
        width={82}
        height={52}
        className={flagClassName}
      />
    );
  }
  if (baseModDepKey) {
    return (
      <BaseModFlagPreview
        key={`${country.tag}-${root ?? "base"}-${bust}`}
        projectId={projectId}
        depKey={baseModDepKey}
        tag={country.tag}
        ideology={root}
        effectiveRootIds={roots}
        width={82}
        height={52}
        className={flagClassName}
        placeholder={
          <FlagPlaceholder tag={country.tag} className={flagClassName} />
        }
      />
    );
  }
  return <FlagPlaceholder tag={country.tag} className={flagClassName} />;
}

function FlagPlaceholder({
  tag,
  className,
}: {
  tag: string;
  className?: string;
}) {
  return (
    <div
      role="img"
      aria-label={`${tag} flag placeholder`}
      className={`flex aspect-[82/52] items-center justify-center rounded-md border border-border bg-muted font-mono text-sm text-muted-foreground ${className ?? ""}`}
    >
      {tag}
    </div>
  );
}
