"use client";

import { useVanillaTagName } from "@/lib/vanillaCountryTags";
import { CountryFlag } from "@/components/workbench/CountryFlag";

interface CountryTagChipProps {
  tag: string | null | undefined;

  title?: string;
  className?: string;
}

export function CountryTagChip({ tag, title, className }: CountryTagChipProps) {
  const vanillaName = useVanillaTagName();
  if (!tag) return null;
  const vanilla = vanillaName(tag);

  const tip = title ?? (vanilla ? `${tag} · ${vanilla}` : tag);
  return (
    <span
      title={tip}
      className={
        "inline-flex items-center gap-1 px-1.5 py-0.5 rounded border border-border " +
        "text-micro font-mono font-semibold text-muted-foreground bg-muted/40 " +
        "leading-none whitespace-nowrap" +
        (className ? ` ${className}` : "")
      }
    >
      {vanilla && <CountryFlag className="text-xs w-3 h-3" />}
      <span>{tag}</span>
    </span>
  );
}
