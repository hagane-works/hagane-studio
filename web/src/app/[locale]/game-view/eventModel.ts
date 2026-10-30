import { useMemo } from "react";
import useSWR from "swr";
import {
  guiRenderApi,
  type GuiBitmapFont,
  type GuiNode,
  type GuiSprite,
  type GuiWindow,
  withGameLanguage,
} from "@/lib/guiRenderApi";
import { wrapLines } from "../gui-renderer/BitmapText";

export const EVENT_WINDOW = "EventWindow";

export const OPTION_ENTRY = "event_option_entry";

const TOP_H = 121;
const MID_MIN = 100;

const MID_PAD = 10;

export interface GuiEvent {
  id: string;
  kind: string;
  namespace: string;
  title?: string;
  desc?: string;

  title_key?: string;
  desc_key?: string;

  origin?: string;
  picture?: string;

  options: { name: string; label?: string }[];
  hidden: boolean;
  major: boolean;
  triggered_only: boolean;
}

export function useEvent(
  projectId: number,
  id: string | null,
  tag: string | null,
  date: string | null,
) {
  return useSWR<GuiEvent>(
    id ? ["game-view-event", projectId, id, tag, date] : null,
    async () => {
      const query = new URLSearchParams({ id: id as string });
      if (tag) query.set("tag", tag);
      if (date) query.set("date", date);
      const response = await fetch(
        withGameLanguage(
          `/api/go/projects/${projectId}/gui-render/event?${query}`,
        ),
      );
      return response.ok ? response.json() : null;
    },
    { revalidateOnFocus: false },
  );
}

function childNamed(node: GuiNode, name: string): GuiNode | undefined {
  if (node.name === name) return node;
  for (const child of node.children ?? []) {
    const found = childNamed(child, name);
    if (found) return found;
  }
  return undefined;
}

export function nodeFont(
  win: GuiWindow,
  name: string,
): GuiBitmapFont | undefined {
  const font = childNamed(win.node, name)?.font;
  return font ? win.fonts[font]?.bitmap : undefined;
}

export function midsectionHeight(
  font: GuiBitmapFont | undefined,
  text: string,
  width = 512,
  maxHeight = 230,
): number {
  if (!font || !text) return MID_MIN;
  const lines = wrapLines(font, text, width).length;
  return Math.max(
    MID_MIN,
    Math.ceil(Math.min(maxHeight, lines * font.line)) + MID_PAD,
  );
}

export function stretchEventWindow(
  win: GuiWindow,
  midH: number,
  picture?: { name: string; sprite: GuiSprite } | null,
): GuiWindow {
  const root = win.node;
  let bottomH = 206;
  const withPicture = (node: GuiNode): GuiNode =>
    node.name === "event_picture" && picture
      ? { ...node, sprite: picture.name }
      : node.children?.length
        ? { ...node, children: node.children.map(withPicture) }
        : node;
  const children = (root.children ?? []).map((child) => {
    if (child.name === "midsection")
      return { ...child, size: { ...child.size, height: midH } };
    if (child.name === "bottom_Window") {
      bottomH = Number(child.size?.height) || bottomH;
      return withPicture({
        ...child,
        position: { ...child.position, y: TOP_H + midH },
      });
    }
    return child;
  });
  return {
    ...win,
    node: {
      ...root,
      size: { ...root.size, height: TOP_H + midH + bottomH },
      children,
    },
    sprites: picture
      ? { ...win.sprites, [picture.name]: picture.sprite }
      : win.sprites,
  };
}

export function useEventWindow({
  win,
  projectId,
  eventId,
  tag,
  date,
  saved,
}: {
  win: GuiWindow;
  projectId: number;
  eventId?: string | null;
  tag?: string | null;
  date?: string | null;

  saved?: Partial<GuiEvent>;
}): { win: GuiWindow; event: GuiEvent } | null {
  const active = win.name === EVENT_WINDOW;
  const fetched = useEvent(
    projectId,
    active ? (eventId ?? null) : null,
    tag ?? null,
    date ?? null,
  );
  const data = useMemo(
    () =>
      fetched.data
        ? saved
          ? { ...fetched.data, ...saved }
          : fetched.data
        : undefined,
    [fetched.data, saved],
  );

  const pictureName =
    data?.picture && !data.picture.includes("[") ? data.picture : null;
  const picture = useSWR<Record<string, GuiSprite>>(
    pictureName ? ["game-view-sprites", projectId, "event", pictureName] : null,
    () => {
      guiRenderApi.setProjectId(projectId);
      return guiRenderApi.sprites([pictureName as string]);
    },
    { revalidateOnFocus: false },
  );
  return useMemo(() => {
    if (!active || !data) return null;
    const midH = midsectionHeight(
      nodeFont(win, "Description"),
      data.desc ?? "",
    );
    const sprite = pictureName ? picture.data?.[pictureName] : undefined;
    return {
      win: stretchEventWindow(
        win,
        midH,
        sprite && pictureName ? { name: pictureName, sprite } : null,
      ),
      event: data,
    };
  }, [active, win, data, pictureName, picture.data]);
}
