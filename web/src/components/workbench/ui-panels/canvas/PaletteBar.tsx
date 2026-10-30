"use client";

import { useState } from "react";
import { useTranslations } from "use-intl";
import {
  AlignLeft,
  Heading,
  Image as ImageIcon,
  LayoutGrid,
  List,
  Minus,
  MousePointerClick,
  RectangleHorizontal,
  ScrollText,
  Square,
  SquareDashed,
  StretchHorizontal,
  ToggleRight,
  Type,
} from "lucide-react";
import type { LucideIcon } from "lucide-react";

import {
  PRESET_GROUPS,
  type ComponentPreset,
} from "@/app/[locale]/game-view/panelMounts";

const PRESET_ICON: Record<ComponentPreset, LucideIcon> = {
  box: Square,
  scroll_box: ScrollText,
  list: List,
  grid: LayoutGrid,
  group: SquareDashed,
  text: Type,
  title: Heading,
  paragraph: AlignLeft,
  button: MousePointerClick,
  small_button: MousePointerClick,
  wide_button: StretchHorizontal,
  image_button: ImageIcon,
  switch: ToggleRight,
  icon: ImageIcon,
  divider: Minus,
  progressbar: RectangleHorizontal,
};

export function PaletteBar({
  onAdd,
  into,
}: {
  onAdd: (preset: ComponentPreset) => void;

  into?: string | null;
}) {
  const t = useTranslations("uiPanelsCanvas");
  const tc = useTranslations("gameViewEdit.components");
  const [group, setGroup] =
    useState<(typeof PRESET_GROUPS)[number]["group"]>("container");
  const presets =
    PRESET_GROUPS.find((entry) => entry.group === group)?.presets ?? [];
  return (
    <div className="flex flex-wrap items-center gap-2 px-3 py-2 border-b border-border bg-sidebar">
      <span className="text-micro uppercase tracking-widest font-semibold text-muted-foreground mr-1">
        {t("paletteLabel")}
      </span>
      <div
        role="group"
        aria-label={tc("groupsLabel")}
        className="inline-flex items-center gap-0.5 rounded-md border border-border bg-card p-0.5"
      >
        {PRESET_GROUPS.map((entry) => (
          <button
            key={entry.group}
            type="button"
            aria-pressed={entry.group === group}
            onClick={() => setGroup(entry.group)}
            className={`h-6 px-2 text-xs rounded ui-interactive ui-focus-ring ${
              entry.group === group
                ? "bg-accent text-accent-foreground"
                : "text-muted-foreground hover:bg-muted"
            }`}
          >
            {tc(`group.${entry.group}`)}
          </button>
        ))}
      </div>
      {presets.map((preset) => {
        const Icon = PRESET_ICON[preset];
        return (
          <button
            key={preset}
            type="button"
            data-preset={preset}
            title={tc(`hint.${preset}`)}
            onClick={() => onAdd(preset)}
            className="inline-flex items-center gap-1.5 h-7 px-2.5 text-xs rounded-md border border-border bg-card text-foreground hover:bg-muted ui-interactive ui-focus-ring"
          >
            <Icon className="w-3.5 h-3.5 opacity-75" aria-hidden />
            {tc(`preset.${preset}`)}
          </button>
        );
      })}
      <span className="ml-auto text-xs text-muted-foreground">
        {into ? t("addIntoContainer", { name: into }) : t("addIntoPanel")}
      </span>
    </div>
  );
}
