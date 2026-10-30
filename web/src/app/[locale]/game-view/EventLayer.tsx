"use client";

import { useMemo } from "react";
import { useTranslations } from "use-intl";
import type { GuiNode, GuiWindow } from "@/lib/guiRenderApi";
import {
  GuiCanvas,
  measureWindow,
  textAlignOf,
} from "../gui-renderer/GuiCanvas";

import { matchNamePath } from "./gameNavigation";
import { BitmapText } from "../gui-renderer/BitmapText";
import { useTemplateWindow } from "./CountrySelectLayer";
import { useEditCanvas, usePrintedOver } from "./editContext";
import type { PrintedText } from "./editModel";
import {
  EVENT_WINDOW,
  OPTION_ENTRY,
  nodeFont,
  type GuiEvent,
} from "./eventModel";
import { FieldHit } from "./FieldHit";
import { eventOptionPrinted, eventPrinted } from "./textSources";

const OPTION_H = 47;
const OPTION_W = 300;
const NO_HIDDEN = new Set<string>();
const NO_PRINTED: Readonly<Record<string, PrintedText>> = {};

function findNode(root: GuiNode, name: string): GuiNode | undefined {
  if (root.name === name) return root;
  for (const child of root.children ?? []) {
    const found = findNode(child, name);
    if (found) return found;
  }
  return undefined;
}

export function EventLayer({
  win,
  event,
  projectId,
  onPickField,
}: {
  win: GuiWindow;
  event: GuiEvent | null;
  projectId: number;

  onPickField?: (field: string) => void;
}) {
  const t = useTranslations("gameViewContent.event");
  const edit = useEditCanvas();
  const active = win.name === EVENT_WINDOW && !!event;
  const entry = useTemplateWindow(projectId, active ? OPTION_ENTRY : null);
  const rects = useMemo(
    () => (active ? measureWindow(win.node, win.sprites, 1920, 1080) : {}),
    [active, win],
  );
  const titleFont = nodeFont(win, "Title");
  const bodyFont = nodeFont(win, "Description");

  const titleNode = useMemo(
    () =>
      active
        ? (matchNamePath(win.node, "Title")[0] ??
          matchNamePath(win.node, "title")[0])
        : undefined,
    [active, win.node],
  );
  const entryRects = useMemo(
    () =>
      entry.data
        ? measureWindow(entry.data.node, entry.data.sprites, OPTION_W, OPTION_H)
        : {},
    [entry.data],
  );

  const printed = useMemo(
    () => (active ? eventPrinted(win, event) : NO_PRINTED),
    [active, win, event],
  );
  usePrintedOver(win, "event", printed);

  if (!active || !event) return null;

  const titleRect = rects["name:title"];
  const descRect = rects["name:description"];
  const optionsRect = rects["name:options_grid"];
  const nameNode = entry.data ? findNode(entry.data.node, "Name") : undefined;
  const nameRect = entryRects["name:name"];
  const optionFont =
    entry.data && nameNode?.font
      ? entry.data.fonts[nameNode.font]?.bitmap
      : undefined;

  return (
    <div
      style={{ position: "absolute", left: 0, top: 0, pointerEvents: "none" }}
    >
      {titleFont && titleRect && event.title && (
        <BitmapText
          font={titleFont}
          text={event.title}
          left={titleRect.x}
          top={titleRect.y}
          width={titleNode?.maxwidth || 551}
          height={titleFont.line + 6}
          align={textAlignOf(titleNode?.format)}
          vcenter={false}
          maxLines={1}
        />
      )}
      {bodyFont && descRect && event.desc && (
        <BitmapText
          font={bodyFont}
          text={event.desc}
          left={descRect.x}
          top={descRect.y}
          width={512}
          height={230}
          align="left"
          vcenter={false}
          maxLines={Math.floor(230 / bodyFont.line)}
        />
      )}
      {optionsRect &&
        entry.data &&
        event.options.map((option, at) => (
          <div
            key={option.name || `option-${at}`}
            style={{
              position: "absolute",
              left: optionsRect.x,
              top: optionsRect.y + at * OPTION_H,
            }}
          >
            <GuiCanvas
              node={entry.data!.node}
              {...edit(
                `${OPTION_ENTRY}:${at}`,
                entry.data!,
                entry.data!.node,
                entry.data!.sprites,
                undefined,
                eventOptionPrinted(entry.data!.node, option),
              )}
              texts={entry.data!.texts}
              fonts={entry.data!.fonts}
              hidden={NO_HIDDEN}
              isolateId={null}
              transparent
              baseW={OPTION_W}
              baseH={OPTION_H}
            />
            {optionFont && nameRect && (
              <BitmapText
                font={optionFont}

                text={option.label || option.name || ""}
                left={nameRect.x}
                top={nameRect.y}
                width={nameNode?.maxwidth || 340}
                height={nameNode?.maxheight || 40}
                align={textAlignOf(nameNode?.format)}
                vcenter
                maxLines={2}
              />
            )}
            {onPickField && nameRect && (
              <FieldHit
                x={nameRect.x}
                y={nameRect.y}
                w={nameNode?.maxwidth || 340}
                h={nameNode?.maxheight || 40}
                label={t("hitOption", { name: option.name || at + 1 })}
                onClick={() => onPickField(`option:${at}`)}
              />
            )}
          </div>
        ))}
      {onPickField && titleRect && titleFont && (
        <FieldHit
          x={titleRect.x}
          y={titleRect.y}
          w={titleNode?.maxwidth || 551}
          h={titleFont.line + 6}
          label={t("hitTitle")}
          onClick={() => onPickField("title")}
        />
      )}
      {onPickField && descRect && (
        <FieldHit
          x={descRect.x}
          y={descRect.y}
          w={512}
          h={Math.max(40, (rects["name:midsection"]?.h ?? 100) - 10)}
          label={t("hitDesc")}
          onClick={() => onPickField("desc")}
        />
      )}
    </div>
  );
}
