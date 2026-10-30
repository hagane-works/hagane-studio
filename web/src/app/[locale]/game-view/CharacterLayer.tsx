"use client";

import { useCallback, useMemo } from "react";
import useSWR from "swr";
import { useTranslations } from "use-intl";
import {
  guiRenderApi,
  type GuiCharacter,
  type GuiCharacters,
  type GuiIdeaCategory,
  type GuiPreviewIcon,
  type GuiSprite,
  type GuiWindow,
} from "@/lib/guiRenderApi";
import {
  GuiCanvas,
  assignIds,
  measureWindow,
  type MeasuredRect,
} from "../gui-renderer/GuiCanvas";
import { BitmapText } from "../gui-renderer/BitmapText";
import { gameTooltipProps } from "../gui-renderer/GameTooltip";
import { drawSize, type SpriteSwaps } from "./spriteSwaps";
import type { GuiNode } from "@/lib/guiRenderApi";
import { useEditCanvas } from "./editContext";
import { namedPrinted } from "./textSources";
import { CARD_GROUPS, CHARACTER_SLOTS, LAW_GROUPS } from "./IdeaPickerLayer";
import { MIO_SLOTS } from "./IndustrialOrgLayer";
import { FieldHit } from "./FieldHit";
import { guiArray, guiRecord } from "./guiResponse";
import { UiEditArt, artUrl } from "./UiEditArt";
import { GAME_SCROLLBAR } from "./PageScrollbar";

const PICKED = (slot: string) =>
  LAW_GROUPS.has(slot) ||
  CARD_GROUPS.has(slot) ||
  CHARACTER_SLOTS.has(slot) ||
  slot in MIO_SLOTS;

const CATEGORY_ENTRY = "country_politics_idea_category_entry";

const SPIRIT_PITCH = 59;
const SPIRIT_SLOTS = 6;

const SMALL_SPIRIT = { w: 33, h: 34, columns: 10, scale: 0.5 };

export function spiritPlace(
  at: number,
  count: number,
  rows: { row: { x: number; y: number }; small?: { x: number; y: number } },
  size: { w: number; h: number },
  origin: { x: number; y: number } = { x: 0, y: 0 },
): { x: number; y: number; w: number; h: number } {
  if (count > SPIRIT_SLOTS && rows.small) {
    return {
      x: rows.small.x - origin.x + (at % SMALL_SPIRIT.columns) * SMALL_SPIRIT.w,
      y:
        rows.small.y -
        origin.y +
        Math.floor(at / SMALL_SPIRIT.columns) * SMALL_SPIRIT.h,
      w: Math.round(size.w * SMALL_SPIRIT.scale),
      h: Math.round(size.h * SMALL_SPIRIT.scale),
    };
  }
  return {
    x: rows.row.x - origin.x + at * SPIRIT_PITCH,
    y: rows.row.y - origin.y,
    w: size.w,
    h: size.h,
  };
}

const ROW_PITCH = 106;

const IDEAS_WHEEL = 40;

export function ideasListBox(
  box: { x: number; y: number; w: number; h: number },
  margin: { top?: number | string; bottom?: number | string },
  rowCount: number,
): {
  left: number;
  top: number;
  width: number;
  height: number;
  total: number;
  scrollable: boolean;
} {
  const top = box.y + (Number(margin.top) || 0);
  const height = Math.max(
    0,
    box.y + box.h - (Number(margin.bottom) || 0) - top,
  );
  const total = rowCount * ROW_PITCH;
  return {
    left: box.x,
    top,
    width: box.w,
    height,
    total,
    scrollable: total > height,
  };
}

const SLOT_X = 17;
const SLOT_Y = 46;
const SLOT_W = 80;
const NO_ART: Record<string, GuiSprite> = {};

export interface CategorySlotFill {
  slot: string;
  person: GuiCharacter | null;

  candidate?: boolean;
  idea: GuiPreviewIcon | null;
}

const ROSTER_SPRITES: Readonly<Record<string, string>> = {
  tank_manufacturer: "GFX_tank_mio_roster",
  naval_manufacturer: "GFX_ship_mio_roster",
  aircraft_manufacturer: "GFX_aircraft_mio_roster",
  materiel_manufacturer: "GFX_material_mio_roster",
};

export const emptySlotSprite = (slot: string) =>
  ROSTER_SPRITES[slot] ?? `GFX_idea_slot_${slot}`;

export function fillCategorySlots(
  categories: GuiIdeaCategory[],
  advisors: GuiCharacter[],
  slotIdeas: Record<string, GuiPreviewIcon[]> = {},
  candidates = true,
): { name: string; label: string; frame: number; slots: CategorySlotFill[] }[] {
  const queues = new Map<string, GuiCharacter[]>();

  for (const advisor of [...advisors].sort(
    (a, b) => Number(b.hired ?? false) - Number(a.hired ?? false),
  )) {
    if (!candidates && !advisor.hired) continue;
    const slot = advisor.slot || "";
    if (!queues.has(slot)) queues.set(slot, []);
    queues.get(slot)?.push(advisor);
  }
  const taken = new Map<string, number>();
  return guiArray(categories).map((category) => ({
    name: category.name,
    label: category.label,
    frame: category.index,
    slots: guiArray(category.slots).map((slot) => {
      const at = taken.get(slot.name) ?? 0;
      taken.set(slot.name, at + 1);
      if (!slot.character) {
        return {
          slot: slot.name,
          person: null,
          idea: slotIdeas[slot.name]?.[at] ?? null,
        };
      }
      const person = queues.get(slot.name)?.[at] ?? null;
      return {
        slot: slot.name,
        person,
        candidate: !!person && !person.hired,
        idea: null,
      };
    }),
  }));
}

export function useCharacters(
  projectId: number,
  tag: string | null,
  date: string | null,
) {
  return useSWR<GuiCharacters>(
    tag ? ["game-view-characters", projectId, tag, date] : null,
    () => {
      guiRenderApi.setProjectId(projectId);
      return guiRenderApi.characters(tag as string, date || undefined);
    },
    { revalidateOnFocus: false },
  );
}

function useTemplateWindow(projectId: number, name: string | null) {
  return useSWR<GuiWindow>(
    name ? ["game-view-window", projectId, name] : null,
    async () => {
      guiRenderApi.setProjectId(projectId);
      const win = await guiRenderApi.getWindow(name as string);
      assignIds(win.node);
      return win;
    },
    { revalidateOnFocus: false },
  );
}

export function portraitURL(
  person: GuiCharacter,
  versions: SpriteSwaps,
): string | undefined {
  if (!person.portrait) return undefined;
  return artUrl(person.portrait, person.portrait_sprite, versions);
}

function Portrait({
  person,
  left,
  top,
  width,
  height,
  versions,
  onEdit,
}: {
  person: GuiCharacter;
  left: number;
  top: number;
  width: number;
  height: number;
  versions: SpriteSwaps;

  onEdit?: (token: string, name: string) => void;
}) {
  const t = useTranslations("gameViewEdit.politics");
  const url = portraitURL(person, versions);
  if (!url || !person.portrait) return null;

  const drawn = drawSize(versions, person.portrait, width, height);

  const tooltip = !onEdit && person.tooltip ? person.tooltip : undefined;
  return (
    <button
      type="button"
      aria-label={person.token}
      title={
        onEdit
          ? t("editPerson", { name: person.name })
          : tooltip
            ? undefined
            : person.name
      }
      {...(tooltip
        ? gameTooltipProps(tooltip, { key: `advisor:${person.token}` })
        : {})}
      onClick={() => onEdit?.(person.token, person.name)}
      style={{
        position: "absolute",
        left,
        top,
        width: drawn.width,
        height: drawn.height,
        padding: 0,
        border: "none",
        backgroundImage: `url("${url}")`,
        backgroundRepeat: "no-repeat",
        backgroundSize: `${drawn.width}px ${drawn.height}px`,
        cursor: onEdit ? "pointer" : "default",
        pointerEvents: onEdit || tooltip ? "auto" : "none",
      }}
    />
  );
}

function CategoryRow({
  template,
  label,
  labelKey,
  frame,
  slots,
  left,
  top,
  place,
  versions,
  emptyArt,
  onEditPerson,
  onOpenIdeas,
}: {
  template: GuiWindow;
  label: string;

  labelKey: string;
  frame: number;
  slots: CategorySlotFill[];
  left: number;
  top: number;

  place: string;
  versions: SpriteSwaps;

  emptyArt: Record<string, GuiSprite>;

  onEditPerson?: (token: string, name: string) => void;

  onOpenIdeas?: (group: string) => void;
}) {
  const t = useTranslations("gameViewEdit.politics");
  const edit = useEditCanvas();

  const node = useMemo(() => {
    const clone = structuredClone(template.node) as GuiNode;
    for (const child of clone.children ?? []) {
      if (child.name === "category_icon") child.frame = frame;
    }
    assignIds(clone);
    return clone;
  }, [template.node, frame]);
  const empty = useMemo(() => new Set<string>(), []);
  const nameId = useMemo(() => {
    const match = (node.children ?? []).find(
      (child) => child.name === "name",
    ) as (GuiNode & { _id?: string }) | undefined;
    return match?._id ?? "";
  }, [node]);
  const titleOverride = useMemo(
    () => (nameId ? { [nameId]: label } : {}),
    [nameId, label],
  );

  const printed = useMemo(
    () => namedPrinted(node, "name", label, labelKey),
    [node, label, labelKey],
  );

  return (
    <div style={{ position: "absolute", left, top }}>
      <GuiCanvas
        node={node}
        {...edit(
          `${template.name}:${place}`,
          template,
          node,
          template.sprites,
          titleOverride,
          printed,
        )}
        texts={template.texts}
        fonts={template.fonts}
        hidden={empty}
        isolateId={null}
        transparent
        baseW={550}
        baseH={100}
      />
      {slots.map((fill, at) =>
        fill.person ? (
          <div
            key={`${fill.person.token}-${at}`}
            style={fill.candidate ? { opacity: 0.45 } : undefined}
          >
            <Portrait
              person={fill.person}
              left={SLOT_X + at * SLOT_W}
              top={SLOT_Y}
              width={fill.person.portrait_sprite?.width || 65}
              height={fill.person.portrait_sprite?.height || 67}
              versions={versions}
              onEdit={onEditPerson}
            />
          </div>
        ) : fill.idea?.icon_sprite &&
          artUrl(fill.idea.icon, fill.idea.icon_sprite, versions) ? (
          <div
            key={`${fill.idea.token}-${at}`}
            title={
              fill.idea.tooltip ? undefined : fill.idea.label || fill.idea.token
            }
            {...(fill.idea.tooltip
              ? gameTooltipProps(fill.idea.tooltip, {
                  key: `law:${fill.idea.token}`,
                })
              : {})}
            role={onOpenIdeas && PICKED(fill.slot) ? "button" : undefined}
            aria-label={
              onOpenIdeas && PICKED(fill.slot)
                ? `${fill.idea.label || fill.idea.token}`
                : undefined
            }
            onClick={
              onOpenIdeas && PICKED(fill.slot)
                ? (e) => {
                    e.stopPropagation();
                    onOpenIdeas(fill.slot);
                  }
                : undefined
            }
            style={{
              position: "absolute",
              left: SLOT_X + at * SLOT_W,
              top: SLOT_Y,
              width: fill.idea.icon_sprite.width || 64,
              height: fill.idea.icon_sprite.height || 64,
              backgroundImage: `url("${artUrl(fill.idea.icon, fill.idea.icon_sprite, versions)}")`,
              backgroundSize: `${fill.idea.icon_sprite.width || 64}px ${fill.idea.icon_sprite.height || 64}px`,
              backgroundRepeat: "no-repeat",
              pointerEvents: onOpenIdeas && PICKED(fill.slot) ? "auto" : "none",
              cursor: onOpenIdeas && PICKED(fill.slot) ? "pointer" : undefined,
            }}
          />
        ) : artUrl(
            emptySlotSprite(fill.slot),
            emptyArt[emptySlotSprite(fill.slot)],
            versions,
          ) ? (
          <div
            key={`empty-${fill.slot}-${at}`}
            role={onOpenIdeas && PICKED(fill.slot) ? "button" : undefined}
            aria-label={
              onOpenIdeas && PICKED(fill.slot) ? `${fill.slot} slot` : undefined
            }
            onClick={
              onOpenIdeas && PICKED(fill.slot)
                ? (e) => {
                    e.stopPropagation();
                    onOpenIdeas(fill.slot);
                  }
                : undefined
            }
            style={{
              position: "absolute",
              left: SLOT_X + at * SLOT_W,
              top: SLOT_Y,
              width: emptyArt[emptySlotSprite(fill.slot)].width || 63,
              height: emptyArt[emptySlotSprite(fill.slot)].height || 63,
              backgroundImage: `url("${artUrl(emptySlotSprite(fill.slot), emptyArt[emptySlotSprite(fill.slot)], versions)}")`,
              backgroundSize: "100% 100%",
              backgroundRepeat: "no-repeat",
              pointerEvents: onOpenIdeas && PICKED(fill.slot) ? "auto" : "none",
              cursor: onOpenIdeas && PICKED(fill.slot) ? "pointer" : undefined,
            }}
          />
        ) : null,
      )}

      {slots.map((fill, at) => {
        const art = slotArt(fill, emptyArt);
        return (
          <UiEditArt
            key={`ui-${fill.slot}-${at}`}
            owner={`${template.name}:${place}:${at}`}
            gfx={art.gfx}
            sprite={art.sprite}
            title={
              fill.person || fill.idea
                ? art.title
                : t("emptySeat", { slot: art.title })
            }
            x={SLOT_X + at * SLOT_W}
            y={SLOT_Y}
            w={art.sprite?.width || 63}
            h={art.sprite?.height || 63}
          />
        );
      })}
    </div>
  );
}

export function slotArt(
  fill: CategorySlotFill,
  emptyArt: Readonly<Record<string, GuiSprite>>,
): { gfx: string | undefined; sprite: GuiSprite | undefined; title: string } {
  if (fill.person)
    return {
      gfx: fill.person.portrait,
      sprite: fill.person.portrait_sprite,
      title: fill.person.name,
    };
  if (fill.idea)
    return {
      gfx: fill.idea.icon,
      sprite: fill.idea.icon_sprite,
      title: fill.idea.label || fill.idea.token,
    };
  const empty = emptySlotSprite(fill.slot);
  return { gfx: empty, sprite: emptyArt[empty], title: fill.slot };
}

export function CharacterLayer({
  win,
  projectId,
  tag,
  date,
  hidden,
  versions,
  onOpenIdeas,
  onEditPerson,
  onEditSpirit,
  onAddSpirit,
}: {
  win: GuiWindow;
  projectId: number;

  tag: string;

  date?: string | null;
  hidden: Set<string>;
  versions: SpriteSwaps;

  onOpenIdeas?: (group: string) => void;

  onEditPerson?: (token: string, name: string) => void;

  onEditSpirit?: (spirit: { token: string; label?: string }) => void;

  onAddSpirit?: () => void;
}) {
  const t = useTranslations("gameViewEdit.politics");
  const isPolitics = win.name === "countrypoliticsview";
  const wanted = isPolitics;
  const people = useCharacters(projectId, wanted ? tag : null, date ?? null);
  const template = useTemplateWindow(
    projectId,
    isPolitics && people.data ? CATEGORY_ENTRY : null,
  );
  const rects = useMemo(
    () =>
      wanted ? measureWindow(win.node, win.sprites, 1920, 1080, hidden) : {},
    [wanted, win, hidden],
  );

  const rows = useMemo(
    () =>
      fillCategorySlots(
        guiArray(people.data?.categories),
        guiArray(people.data?.advisors),
        guiRecord(people.data?.slot_ideas),
        false,
      ),
    [people.data],
  );
  const emptyNames = useMemo(() => {
    const names = new Set<string>();
    for (const row of rows)
      for (const fill of row.slots) names.add(emptySlotSprite(fill.slot));
    return [...names].sort();
  }, [rows]);
  const emptyArt = useSWR<Record<string, GuiSprite>>(
    isPolitics && emptyNames.length
      ? ["game-view-sprites", projectId, "empty-slots", emptyNames.join(",")]
      : null,
    () => {
      guiRenderApi.setProjectId(projectId);
      return guiRenderApi.sprites(emptyNames);
    },
    { revalidateOnFocus: false },
  );

  const ideasWheel = useCallback((list: HTMLDivElement | null) => {
    if (!list) return;
    const onWheel = (e: WheelEvent) => {
      if (e.ctrlKey || list.scrollHeight <= list.clientHeight) return;
      const step =
        e.deltaMode === 1
          ? e.deltaY * IDEAS_WHEEL
          : Math.abs(e.deltaY) >= 50
            ? Math.sign(e.deltaY) * IDEAS_WHEEL
            : e.deltaY;
      if (!step) return;
      e.preventDefault();
      list.scrollTop += step;
    };
    list.addEventListener("wheel", onWheel, { passive: false });
    return () => list.removeEventListener("wheel", onWheel);
  }, []);

  if (!wanted || !people.data) return null;
  const leader = people.data.leader;
  const spirits = guiArray(people.data.spirits);
  const leaderRect: MeasuredRect | undefined = rects["name:leader"];
  const nameRect: MeasuredRect | undefined = rects["name:leader_name"];
  const gridRect: MeasuredRect | undefined = rects["name:idea_categories_grid"];
  const spiritRect: MeasuredRect | undefined = rects["name:spirit_grid"];

  const spiritBox: MeasuredRect | undefined =
    rects["name:national_spirit_container"];
  const smallGrid: MeasuredRect | undefined =
    rects["name:spirit_grid_over_defined"];
  const smallSpirits =
    spirits.length > SPIRIT_SLOTS && !!spiritBox && !!smallGrid;
  const shownSpirits = smallSpirits ? spirits : spirits.slice(0, SPIRIT_SLOTS);
  const spiritOrigin =
    smallSpirits && spiritBox
      ? { x: spiritBox.x, y: spiritBox.y }
      : { x: 0, y: 0 };
  const spiritSize = (sprite: GuiSprite | undefined) => ({
    w: sprite?.width
      ? Math.round(sprite.width / Math.max(1, sprite.frames || 1))
      : SPIRIT_PITCH,
    h: sprite?.height || 64,
  });
  const placeSpirit = (at: number, sprite: GuiSprite | undefined) =>
    spiritPlace(
      at,
      spirits.length,
      {
        row: spiritRect ?? { x: 0, y: 0 },
        small: smallSpirits ? smallGrid : undefined,
      },
      spiritSize(sprite),
      spiritOrigin,
    );
  const nameNode = matchName(win.node, "leader_name");

  const ideasMargin = (matchName(win.node, "ideas")?.raw?.margin ?? {}) as {
    top?: number | string;
    bottom?: number | string;
  };
  const rowsTop = (gridRect?.y ?? 0) + (Number(ideasMargin.top) || 0);
  const ideasRect: MeasuredRect | undefined = rects["name:ideas"];
  const list = ideasRect
    ? ideasListBox(ideasRect, ideasMargin, rows.length)
    : null;
  const nameFont = nameNode?.font
    ? win.fonts[nameNode.font]?.bitmap
    : undefined;

  return (
    <div
      style={{ position: "absolute", left: 0, top: 0, pointerEvents: "none" }}
    >
      {leader && leaderRect && !leaderRect.hidden && (
        <Portrait
          person={leader}
          left={leaderRect.x}
          top={leaderRect.y}
          width={leader.portrait_sprite?.width || leaderRect.w}
          height={leader.portrait_sprite?.height || leaderRect.h}
          versions={versions}
          onEdit={onEditPerson}
        />
      )}
      {spiritRect && (
        <div
          data-spirit-row={smallSpirits ? "small" : "row"}
          style={
            smallSpirits && spiritBox
              ? {
                  position: "absolute",
                  left: spiritBox.x,
                  top: spiritBox.y,
                  width: spiritBox.w,
                  height: spiritBox.h,
                  overflowX: "hidden",
                  overflowY: "auto",
                  ...GAME_SCROLLBAR,
                  pointerEvents: "auto",
                }
              : { position: "absolute", left: 0, top: 0 }
          }
        >
          {smallSpirits && (
            <div
              style={{
                height:
                  Math.ceil(spirits.length / SMALL_SPIRIT.columns) *
                    SMALL_SPIRIT.h +
                  (smallGrid?.y ?? 0) -
                  spiritOrigin.y,
              }}
            />
          )}
          {shownSpirits.map((spirit, at) => {
            const url = artUrl(spirit.icon, spirit.icon_sprite, versions);
            if (!url) return null;
            const place = placeSpirit(at, spirit.icon_sprite);
            return (
              <div
                key={spirit.token}
                title={
                  spirit.tooltip ? undefined : spirit.label || spirit.token
                }
                {...(spirit.tooltip && !onEditSpirit
                  ? gameTooltipProps(spirit.tooltip, {
                      key: `spirit:${spirit.token}`,
                    })
                  : {})}
                style={{
                  position: "absolute",
                  left: place.x,
                  top: place.y,
                  width: place.w,
                  height: place.h,
                  backgroundImage: `url("${url}")`,
                  backgroundSize: `${place.w}px ${place.h}px`,
                  backgroundRepeat: "no-repeat",

                  pointerEvents:
                    spirit.tooltip && !onEditSpirit ? "auto" : "none",
                }}
              />
            );
          })}
          {onEditSpirit &&
            shownSpirits.map((spirit, at) => {
              const place = placeSpirit(at, spirit.icon_sprite);
              return (
                <FieldHit
                  key={`edit-${spirit.token}`}
                  x={place.x}
                  y={place.y}
                  w={smallSpirits ? SMALL_SPIRIT.w - 1 : SPIRIT_PITCH - 2}
                  h={place.h}
                  label={t("editSpirit", {
                    name: spirit.label || spirit.token,
                  })}
                  tooltip={spirit.tooltip}
                  onClick={() =>
                    onEditSpirit({ token: spirit.token, label: spirit.label })
                  }
                />
              );
            })}

          {shownSpirits.map((spirit, at) => {
            const place = placeSpirit(at, spirit.icon_sprite);
            return (
              <UiEditArt
                key={`ui-${spirit.token}`}
                owner={`spirit:${at}`}
                gfx={spirit.icon}
                sprite={spirit.icon_sprite}
                title={spirit.label || spirit.token}
                x={place.x}
                y={place.y}
                w={place.w}
                h={place.h}
              />
            );
          })}
          {onAddSpirit && (smallSpirits || spirits.length < SPIRIT_SLOTS) && (
            <FieldHit
              x={placeSpirit(spirits.length, undefined).x}
              y={placeSpirit(spirits.length, undefined).y}
              w={smallSpirits ? SMALL_SPIRIT.w - 1 : SPIRIT_PITCH - 2}
              h={smallSpirits ? SMALL_SPIRIT.h - 2 : 64}
              label={t("addStartSpirit")}
              onClick={onAddSpirit}
            />
          )}
        </div>
      )}
      {leader && leaderRect && !leaderRect.hidden && (
        <UiEditArt
          owner="leader"
          gfx={leader.portrait}
          sprite={leader.portrait_sprite}
          title={leader.name}
          x={leaderRect.x}
          y={leaderRect.y}
          w={
            drawSize(
              versions,
              leader.portrait,
              leader.portrait_sprite?.width || leaderRect.w,
              leader.portrait_sprite?.height || leaderRect.h,
            ).width
          }
          h={
            drawSize(
              versions,
              leader.portrait,
              leader.portrait_sprite?.width || leaderRect.w,
              leader.portrait_sprite?.height || leaderRect.h,
            ).height
          }
        />
      )}
      {leader && nameRect && nameFont && (
        <BitmapText
          font={nameFont}
          text={leader.name}
          left={nameRect.x}
          top={nameRect.y}
          width={leaderRect?.w || 156}
          height={24}
          align="center"
          vcenter={false}
          maxLines={1}
        />
      )}
      {leader && nameRect && onEditPerson && (
        <FieldHit
          x={nameRect.x}
          y={nameRect.y}
          w={leaderRect?.w || 156}
          h={24}
          label={t("editPerson", { name: leader.name })}
          onClick={() => onEditPerson(leader.token, leader.name)}
        />
      )}
      {template.data && gridRect && list && (
        <div
          ref={ideasWheel}
          data-politics-rows
          style={{
            position: "absolute",
            left: list.left,
            top: list.top,
            width: list.width,
            height: list.height,
            overflowX: "hidden",
            overflowY: list.scrollable ? "auto" : "hidden",
            ...GAME_SCROLLBAR,
            pointerEvents: list.scrollable ? "auto" : "none",
          }}
        >
          <div style={{ position: "relative", height: list.total }}>
            {rows.map((row, index) => (
              <CategoryRow
                key={row.name}
                template={template.data as GuiWindow}
                label={row.label}
                labelKey={row.name}
                frame={row.frame}
                slots={row.slots}
                left={gridRect.x - list.left}
                top={index * ROW_PITCH}
                place={`${gridRect.x},${rowsTop + index * ROW_PITCH}`}
                versions={versions}
                emptyArt={emptyArt.data ?? NO_ART}
                onEditPerson={onEditPerson}
                onOpenIdeas={onOpenIdeas}
              />
            ))}
          </div>
        </div>
      )}
    </div>
  );
}

function matchName(root: GuiNode, name: string): GuiNode | undefined {
  if (root.name === name) return root;
  for (const child of root.children ?? []) {
    const found = matchName(child, name);
    if (found) return found;
  }
  return undefined;
}
