"use client";

import { useCallback, useMemo } from "react";
import { useLocale, useTranslations } from "use-intl";
import useSWR from "swr";
import { countriesApi } from "@/lib/api/scenario";
import {
  useVanillaCountryTags,
  vanillaTagName,
} from "@/lib/vanillaCountryTags";
import type { RGB } from "./studioSketch";

export const TAG = /^[A-Z][A-Z0-9]{2}$/;

export function tagColor(tag: string): RGB {
  let n = 0;
  for (const ch of tag) n = n * 31 + ch.charCodeAt(0);
  const hue = (n * 137.508) % 360;
  const c = 0.5;
  const x = c * (1 - Math.abs(((hue / 60) % 2) - 1));
  const [r, g, b] =
    hue < 60
      ? [c, x, 0]
      : hue < 120
        ? [x, c, 0]
        : hue < 180
          ? [0, c, x]
          : hue < 240
            ? [0, x, c]
            : hue < 300
              ? [x, 0, c]
              : [c, 0, x];
  const m = 0.32;
  return [
    Math.round((r + m) * 255),
    Math.round((g + m) * 255),
    Math.round((b + m) * 255),
  ];
}

export function colourFor(
  rowOf: ReadonlyMap<string, { color?: number[] }>,
  mapColours: Record<string, [number, number, number]> | undefined,
  tag: string,
): RGB {
  const c = rowOf.get(tag)?.color;
  if (Array.isArray(c) && c.length === 3) return [c[0], c[1], c[2]];
  return mapColours?.[tag] ?? tagColor(tag);
}

export const swatch = (rgb: RGB) => ({
  backgroundColor: `rgb(${rgb[0]}, ${rgb[1]}, ${rgb[2]})`,
});

export function useStudioCountries(
  projectId: number,
  mapColours: Record<string, [number, number, number]> | undefined,
) {
  const locale = useLocale();
  const { tags: gameTags, lookup } = useVanillaCountryTags();
  const rows = useSWR(
    ["map-studio-countries", projectId],
    () => countriesApi.list(projectId),
    { revalidateOnFocus: false },
  );
  const rowOf = useMemo(
    () => new Map((rows.data?.countries ?? []).map((row) => [row.tag, row])),
    [rows.data],
  );

  const nameOf = useCallback(
    (tag: string) => {
      const row = rowOf.get(tag);
      const game = lookup(tag);
      const name =
        row?.name && row.name !== tag
          ? row.name
          : game
            ? vanillaTagName(game, locale)
            : "";
      return name ? `${name} (${tag})` : tag;
    },
    [rowOf, lookup, locale],
  );
  const colourOf = useCallback(
    (tag: string) => colourFor(rowOf, mapColours, tag),
    [rowOf, mapColours],
  );
  return { rows, rowOf, nameOf, colourOf, locale, gameTags };
}

export function AddCountrySelect({
  countries,
  chosen,
  onMap = [],
  disabled,
  onAdd,
}: {
  countries: ReturnType<typeof useStudioCountries>;
  chosen: readonly string[];

  onMap?: readonly string[];
  disabled?: boolean;
  onAdd: (tag: string) => void;
}) {
  const t = useTranslations("mapStudio");
  const { rows, nameOf, locale, gameTags } = countries;
  const taken = new Set(chosen);
  const project = (rows.data?.countries ?? [])
    .filter((row) => TAG.test(row.tag) && !taken.has(row.tag))
    .map((row) => row.tag);
  for (const tag of project) taken.add(tag);
  const mapTags = onMap.filter((tag) => TAG.test(tag) && !taken.has(tag));
  for (const tag of mapTags) taken.add(tag);
  const byName = (a: string, b: string) =>
    nameOf(a).localeCompare(nameOf(b), locale);
  mapTags.sort(byName);
  const game = gameTags
    .map((c) => c.tag)
    .filter((tag) => !taken.has(tag))
    .sort(byName);
  const groups: [string, string[]][] = [
    [t("groupProject"), project],
    [t("groupMap"), mapTags],
    [t("groupGame"), game],
  ];
  const any = project.length + mapTags.length + game.length > 0;
  return (
    <select
      aria-label={t("addCountry")}
      className="h-7 min-w-0 flex-1 rounded border border-input bg-background px-1 text-xs"
      value=""
      disabled={disabled || !any}
      onChange={(e) => {
        if (e.target.value) onAdd(e.target.value);
      }}
    >
      <option value="">{any ? t("addCountry") : t("addCountryNone")}</option>
      {groups
        .filter(([, tags]) => tags.length > 0)
        .map(([label, tags]) => (
          <optgroup key={label} label={label}>
            {tags.map((tag) => (
              <option key={tag} value={tag}>
                {nameOf(tag)}
              </option>
            ))}
          </optgroup>
        ))}
    </select>
  );
}

export function longTagCount(
  countries: ReturnType<typeof useStudioCountries>,
): number {
  return (countries.rows.data?.countries ?? []).filter(
    (row) => !TAG.test(row.tag),
  ).length;
}
