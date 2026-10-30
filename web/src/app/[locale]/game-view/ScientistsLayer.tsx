"use client";

import { useMemo, useState } from "react";
import useSWR from "swr";
import { useTranslations } from "use-intl";
import { guiArray, guiRecord, readGuiResponse } from "./guiResponse";
import type { GuiNode, GuiSprite, GuiWindow } from "@/lib/guiRenderApi";
import { withGameLanguage } from "@/lib/guiRenderApi";
import { assignIds, measureWindow } from "../gui-renderer/GuiCanvas";
import { useTemplateWindow } from "./CountrySelectLayer";
import { FieldHit } from "./FieldHit";
import { Filled } from "./FilledWindow";
import { matchNamePath } from "./gameNavigation";
import { GAME_SCROLLBAR } from "./PageScrollbar";

export const SCIENTISTS_WINDOW = "scientist_roster_window";
const ROW = "scientist_item";
const SKILL = "skill_level_entry";
const TRAIT = "scientist_trait_entry";
const SORT = "specialization_sort_entry";

const ROW_SLOT = { w: 515, h: 100 };
const ROW_SIZE = { w: 515, h: 105 };

const SKILL_SLOT_W = 35;
const TRAIT_SLOT_W = 24;
const SORT_SLOT_W = 30;

export interface GuiScientist {
  token: string;
  name: string;
  portrait?: string;
  traits: { id: string; label: string; icon: string }[];
  skills: {
    specialization: string;
    label: string;
    level: number;
    icon: string;
  }[];
}

export interface GuiScientists {
  tag: string;
  scientists: GuiScientist[];
  specializations: { id: string; icon: string; label: string }[];

  trait_catalog: { id: string; label: string }[];
  title: string;
  sprites: Record<string, GuiSprite>;
}

export function useScientists(
  projectId: number,
  tag: string | null,
  date: string | null | undefined,
) {
  return useSWR<GuiScientists>(
    tag ? ["game-view-scientists", projectId, tag, date ?? ""] : null,
    async () => {
      const query = new URLSearchParams({ tag: tag as string });
      if (date) query.set("date", date);
      const response = await fetch(
        withGameLanguage(
          `/api/go/projects/${projectId}/gui-render/scientists?${query}`,
        ),
      );
      return readGuiResponse<GuiScientists>(response);
    },
    { revalidateOnFocus: false },
  );
}

function idOf(node: GuiNode | undefined): string {
  return (node as (GuiNode & { _id?: string }) | undefined)?._id ?? "";
}

const LIST_FRAME = "scientists";

export function scientistsWindow(
  win: GuiWindow,
  data: GuiScientists | undefined,
): { win: GuiWindow; texts: Record<string, string> } {
  const node = structuredClone(win.node) as GuiNode;
  const windowH = Number(node.size?.height ?? 0);
  for (const frame of matchNamePath(node, LIST_FRAME)) {
    const height = Number(frame.size?.height);
    const top = Number(frame.position?.y ?? 0);
    if (windowH > 0 && height < 0)
      frame.size = { ...frame.size, height: windowH - top + height };
  }
  assignIds(node);
  const texts: Record<string, string> = {};
  for (const hit of matchNamePath(node, "title_text"))
    if (idOf(hit)) texts[idOf(hit)] = data?.title ?? "Scientists";
  for (const hit of matchNamePath(node, "instructions_text"))
    if (idOf(hit)) texts[idOf(hit)] = "";
  return { win: { ...win, node }, texts };
}

type Rect = { x: number; y: number; w: number; h: number };

function rectsOf(
  node: GuiNode,
  sprites: GuiWindow["sprites"],
  w: number,
  h: number,
  paths: readonly string[],
) {
  const measured = measureWindow(node, sprites, w, h);
  const out: Record<string, Rect | undefined> = {};
  for (const path of paths)
    out[path] = measured[idOf(matchNamePath(node, path)[0])] as
      Rect | undefined;
  return out;
}

const WINDOW_PARTS = [
  "scientist_list",
  "scientist_list_container",
  "sort_name",
  "sort_specialization_box",
  "close_button",
];
const ROW_PARTS = ["skill_levels", "trait_icons"];

export function sortScientists(
  scientists: GuiScientist[],
  key: string,
  descending: boolean,
): GuiScientist[] {
  const level = (scientist: GuiScientist) =>
    guiArray(scientist.skills).find((skill) => skill.specialization === key)
      ?.level ?? 0;
  return [...scientists].sort((left, right) => {
    const compared =
      key === "name"
        ? left.name.localeCompare(right.name)
        : level(left) - level(right);
    const ordered = descending ? -compared : compared;
    return ordered || left.name.localeCompare(right.name);
  });
}

export function ScientistsLayer({
  win,
  projectId,
  tag,
  date,
  onClose,
  onEditScientist,
}: {
  win: GuiWindow;
  projectId: number;
  tag: string;
  date?: string | null;
  onClose: () => void;
  editMode?: boolean;

  onEditScientist?: (token: string) => void;
}) {
  const tLayer = useTranslations("gameViewResearch.layer");
  const active = win.name === SCIENTISTS_WINDOW;
  const data = useScientists(projectId, active ? tag : null, date);
  const row = useTemplateWindow(projectId, active ? ROW : null);
  const skill = useTemplateWindow(projectId, active ? SKILL : null);
  const trait = useTemplateWindow(projectId, active ? TRAIT : null);
  const sort = useTemplateWindow(projectId, active ? SORT : null);
  const [sortState, setSortState] = useState({
    key: "name",
    descending: false,
  });
  const at = useMemo(
    () =>
      active
        ? rectsOf(
            structuredClone(win.node) as GuiNode,
            win.sprites,
            1920,
            1080,
            WINDOW_PARTS,
          )
        : {},
    [active, win],
  );
  const inRow = useMemo(() => {
    if (!row.data) return {};
    const node = structuredClone(row.data.node) as GuiNode;
    assignIds(node);
    return rectsOf(node, row.data.sprites, ROW_SIZE.w, ROW_SIZE.h, ROW_PARTS);
  }, [row.data]);
  if (!active || !data.data || !row.data || !skill.data || !trait.data)
    return null;
  const roster = data.data;
  const sprites = guiRecord(roster.sprites);
  const specializations = guiArray(roster.specializations);
  const scientists = sortScientists(
    guiArray(roster.scientists),
    sortState.key,
    sortState.descending,
  );
  const list = at["scientist_list"];
  const box = at["scientist_list_container"];
  const sortName = at["sort_name"];
  const sortBox = at["sort_specialization_box"];
  const close = at["close_button"];
  const skills = inRow["skill_levels"];
  const traits = inRow["trait_icons"];
  const chooseSort = (key: string) =>
    setSortState((current) =>
      current.key === key
        ? { key, descending: !current.descending }
        : { key, descending: key !== "name" },
    );
  return (
    <div
      style={{ position: "absolute", left: 0, top: 0, pointerEvents: "none" }}
    >
      {sortName && (
        <button
          type="button"
          aria-label="sort scientists by name"
          aria-pressed={sortState.key === "name"}
          onClick={() => chooseSort("name")}
          style={{
            position: "absolute",
            left: sortName.x,
            top: sortName.y,
            width: sortName.w || 80,
            height: sortName.h || 35,
            padding: 0,
            border: "none",
            background: "transparent",
            cursor: "inherit",
            pointerEvents: "auto",
          }}
        />
      )}
      {sort.data &&
        sortBox &&
        specializations.map((spec, index) => (
          <div
            key={spec.id}
            style={{
              position: "absolute",
              left: sortBox.x + index * SORT_SLOT_W,
              top: sortBox.y,
              width: SORT_SLOT_W,
              height: 40,
            }}
          >
            <Filled
              entry={sort.data as GuiWindow}
              left={0}
              top={0}
              width={SORT_SLOT_W}
              height={40}
              texts={{}}
              pictures={{ specialization_icon: spec.icon }}
              extraSprites={sprites}
            />
            <button
              type="button"
              aria-label={"sort scientists by " + spec.id}
              aria-pressed={sortState.key === spec.id}
              onClick={() => chooseSort(spec.id)}
              style={{
                position: "absolute",
                inset: 0,
                padding: 0,
                border: "none",
                background: "transparent",
                cursor: "inherit",
                pointerEvents: "auto",
              }}
            />
          </div>
        ))}
      {list && box && (
        <div
          style={{
            position: "absolute",
            left: list.x,
            top: list.y,
            width: ROW_SIZE.w + 4,
            height: Math.max(0, box.y + box.h - list.y),
            overflowY: "auto",
            overflowX: "hidden",
            ...GAME_SCROLLBAR,
            pointerEvents: "auto",
          }}
        >
          <div
            style={{
              position: "relative",
              height:
                Math.max(0, scientists.length - 1) * ROW_SLOT.h + ROW_SIZE.h,
            }}
          >
            {scientists.map((scientist, index) => (
              <div
                key={scientist.token}
                style={{
                  position: "absolute",
                  left: 0,
                  top: index * ROW_SLOT.h,
                  width: ROW_SIZE.w,
                  height: ROW_SIZE.h,
                }}
              >
                <Filled
                  entry={row.data as GuiWindow}
                  left={0}
                  top={0}
                  width={ROW_SIZE.w}
                  height={ROW_SIZE.h}
                  texts={{ name: scientist.name }}
                  pictures={
                    scientist.portrait ? { portrait: scientist.portrait } : {}
                  }
                  frames={{ background_button: 1 }}
                  hide={["recruit_cost", "scientist_injured"]}
                  extraSprites={sprites}
                />
                {skills &&
                  guiArray(scientist.skills).map((item, at) => (
                    <Filled
                      key={item.specialization}
                      entry={skill.data as GuiWindow}
                      left={skills.x + at * SKILL_SLOT_W}
                      top={skills.y}
                      width={50}
                      height={20}
                      texts={{ skill_level: String(item.level) }}
                      pictures={{ specialization_icon: item.icon }}
                      extraSprites={sprites}
                    />
                  ))}
                {traits &&
                  guiArray(scientist.traits).map((item, at) => (
                    <Filled
                      key={item.id}
                      entry={trait.data as GuiWindow}
                      left={traits.x + at * TRAIT_SLOT_W}
                      top={traits.y}
                      width={24}
                      height={32}
                      texts={{}}
                      pictures={{ trait_icon: item.icon }}
                      extraSprites={sprites}
                    />
                  ))}
                {onEditScientist && (
                  <FieldHit
                    x={0}
                    y={0}
                    w={ROW_SLOT.w}
                    h={ROW_SLOT.h}
                    label={tLayer("scientist", { name: scientist.name })}
                    onClick={() => onEditScientist(scientist.token)}
                  />
                )}
              </div>
            ))}
          </div>
        </div>
      )}
      {close && (
        <div
          role="button"
          aria-label="Close scientists"
          onClick={onClose}
          style={{
            position: "absolute",
            left: close.x,
            top: close.y,
            width: close.w || 30,
            height: close.h || 30,
            cursor: "pointer",
            pointerEvents: "auto",
          }}
        />
      )}
    </div>
  );
}
