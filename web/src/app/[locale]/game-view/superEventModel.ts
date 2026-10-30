import useSWR from "swr";
import type { GuiNode, GuiSprite, GuiWindow } from "@/lib/guiRenderApi";
import { projectsApi } from "@/lib/api/project";
import { superEventsApi, type SuperEventRow } from "@/lib/api/scenario";
import { assignIds } from "../gui-renderer/GuiCanvas";
import { matchNamePath } from "./gameNavigation";

export const SUPER_EVENT_SUFFIX = "_SE_main_container";

const PICTURE = { w: 460, h: 252 };

export function isSuperEventWindow(name: string | null | undefined): boolean {
  return Boolean(name?.endsWith(SUPER_EVENT_SUFFIX));
}

export function superEventTag(
  row: Pick<SuperEventRow, "country_tag">,
  projectTag: string | null | undefined,
): string {
  return (row.country_tag.trim() || projectTag?.trim() || "UNK").toUpperCase();
}

export function superEventWindowName(
  row: Pick<SuperEventRow, "country_tag">,
  projectTag: string | null | undefined,
): string {
  return superEventTag(row, projectTag) + SUPER_EVENT_SUFFIX;
}

const pick = (
  locs: Record<string, string> | null | undefined,
  legacy: string,
) => (locs?.en ?? "").trim() || (legacy ?? "").trim();

export function superEventTexts(row: SuperEventRow): {
  title: string;
  desc: string;
  quote: string;
  button: string;
} {
  return {
    title:
      pick(row.display_name_locs, row.title_en) ||
      `(superevent ${String(row.number).padStart(4, "0")})`,
    desc: pick(row.description_locs, row.desc_en),
    quote: pick(row.quote_locs, row.quote_en),
    button: pick(row.button_locs, row.btn_en) || "(continue)",
  };
}

function idOf(node: GuiNode | undefined): string {
  return (node as (GuiNode & { _id?: string }) | undefined)?._id ?? "";
}

export function superEventWindow(
  win: GuiWindow,
  row: SuperEventRow,
  pictureUrl: string | null,
): { win: GuiWindow; texts: Record<string, string>; hidden: string[] } {
  const tag = win.name.slice(0, -SUPER_EVENT_SUFFIX.length);
  const node = structuredClone(win.node) as GuiNode;
  assignIds(node);
  const at = (suffix: string) => matchNamePath(node, `${tag}_SE_${suffix}`)[0];
  const words = superEventTexts(row);
  const texts: Record<string, string> = {};
  for (const [suffix, text] of [
    ["title", words.title],
    ["desc_left", words.desc],
    ["quote", words.quote],
    ["btn", words.button],
  ] as const) {
    const id = idOf(at(suffix));
    if (id) texts[id] = text;
  }
  const hidden = ["desc_center", "quote_extra"]
    .map((suffix) => idOf(at(suffix)))
    .filter(Boolean);
  let sprites = win.sprites;
  const picture = at("picture_positioning_entry");
  if (picture && pictureUrl) {
    const name = `GFX_${tag}_SE_picture_row_${row.id}`;
    picture.sprite = name;
    sprites = {
      ...sprites,
      [name]: {
        resolved: true,
        frames: 1,
        width: PICTURE.w,
        height: PICTURE.h,
        effect: null,
        effect_kind: null,
        cornered: false,
        bordersize: null,
        vanilla: true,
        vanilla_url: pictureUrl,
      } satisfies GuiSprite,
    };
  }
  return { win: { ...win, node, sprites }, texts, hidden };
}

export function superEventPictureUrl(
  projectId: number,
  row: SuperEventRow,
): string | null {
  if (!row.picture_path?.trim()) return null;
  return `${superEventsApi.picturePreviewUrl(projectId, row.id)}?v=${encodeURIComponent(row.updated_at)}`;
}

export function useSuperEvents(projectId: number, enabled: boolean) {
  return useSWR(
    enabled ? ["game-view-super-events", projectId] : null,
    () => superEventsApi.list(projectId),
    {
      revalidateOnFocus: false,
    },
  );
}

export function useSuperEvent(projectId: number, id: number | null) {
  return useSWR(
    id ? ["game-view-super-event", projectId, id] : null,
    () => superEventsApi.getSuperEvent(projectId, id as number),
    {
      revalidateOnFocus: false,
    },
  );
}

export function useProjectTag(projectId: number, enabled: boolean) {
  return useSWR(
    enabled ? ["game-view-project-tag", projectId] : null,
    async () => (await projectsApi.get(projectId)).country_tag ?? null,
    {
      revalidateOnFocus: false,
    },
  );
}
