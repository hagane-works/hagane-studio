"use client";

import { Fragment, useMemo } from "react";
import {
  parseTagTokens,
  useVanillaCountryTags,
} from "@/lib/vanillaCountryTags";
import { useContentLocaleStore } from "@/lib/store/contentLocaleStore";

interface CountryTagPreviewProps {
  text: string | null | undefined;

  projectTagLookup?: (tag: string) => { display: string; flag: string } | null;

  chipStyle?: boolean;
  className?: string;
}

export function CountryTagPreview({
  text,
  projectTagLookup,
  chipStyle = true,
  className,
}: CountryTagPreviewProps) {
  const locale = useContentLocaleStore((s) => s.locale);
  const { lookup } = useVanillaCountryTags();
  const tokens = useMemo(
    () => parseTagTokens(text ?? "", lookup, projectTagLookup, locale),
    [text, lookup, projectTagLookup, locale],
  );

  if (!text) return null;
  if (tokens.length === 0) {
    return <span className={className}>{text}</span>;
  }

  const parts: React.ReactNode[] = [];
  let cursor = 0;
  let key = 0;
  for (const tok of tokens) {
    if (tok.start > cursor) {
      parts.push(
        <Fragment key={`p${key++}`}>{text.slice(cursor, tok.start)}</Fragment>,
      );
    }
    parts.push(
      <span
        key={`t${key++}`}
        className={
          chipStyle
            ? "inline-flex items-baseline gap-1 px-1.5 py-0.5 rounded bg-amber-50 dark:bg-amber-950/30 text-amber-900 dark:text-amber-200 border border-amber-200 dark:border-amber-800/40 text-[0.9em]"
            : ""
        }
        title={`${tok.tag} · ${tok.raw}`}
      >
        {tok.flag && <span aria-hidden>{tok.flag}</span>}
        <span>{tok.display}</span>
      </span>,
    );
    cursor = tok.end;
  }
  if (cursor < text.length) {
    parts.push(<Fragment key={`p${key++}`}>{text.slice(cursor)}</Fragment>);
  }
  return <span className={className}>{parts}</span>;
}
