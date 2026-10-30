"use client";

import { useMemo, useState } from "react";
import useSWR from "swr";
import {
  guiRenderApi,
  type GuiBitmapFont,
  type GuiBookmarks,
  type GuiCountryCard,
  type GuiNode,
  type GuiSprite,
  type GuiWindow,
} from "@/lib/guiRenderApi";
import {
  GuiCanvas,
  assignIds,
  measureWindow,
  textAlignOf,
  type MeasuredRect,
} from "../gui-renderer/GuiCanvas";
import { BitmapText } from "../gui-renderer/BitmapText";
import { gameTooltipProps } from "../gui-renderer/GameTooltip";
import { useEditCanvas, usePrintedOver } from "./editContext";
import { spriteUrlOf } from "./spriteFrame";
import { countryCardPrinted, countryPickPrinted } from "./textSources";

export const COUNTRY_PICKER = "gamesetup_interesting_countries_window";

const COUNTRY_ENTRY = "country_entry";

const MINI_ENTRY = "country_entry_mini";

const FILTER_ENTRY = "country_filter";

const FILTER_W = 149;
const FILTER_H = 34;
const FILTER_GAP = 14;

const CARD_PITCH = 150;

const MINI_W = 52;
const MINI_H = 38;
const MINI_COLS = 23;
const MINI_ROWS = 2;

const SPIRIT_PITCH = 60;
const FOCUS_PITCH = 105;
const PREVIEW_ICON = 63;

export function useBookmarks(projectId: number, enabled: boolean) {
  return useSWR<GuiBookmarks>(
    enabled ? ["game-view-bookmarks", projectId] : null,
    () => {
      guiRenderApi.setProjectId(projectId);
      return guiRenderApi.bookmarks();
    },
    { revalidateOnFocus: false },
  );
}

export function useTemplateWindow(projectId: number, name: string | null) {
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

function findNode(root: GuiNode, name: string): GuiNode | undefined {
  if (root.name === name) return root;
  for (const child of root.children ?? []) {
    const found = findNode(child, name);
    if (found) return found;
  }
  return undefined;
}

export function countryLeaderPortraitUrl(
  leader: GuiCountryCard["leader"],
): string | undefined {
  const sprite = leader?.portrait_sprite;
  return sprite ? spriteUrlOf(sprite, leader.portrait) : undefined;
}

function CountryCard({
  template,
  country,
  left,
  top,
  selected,
  onSelect,
}: {
  template: GuiWindow;
  country: GuiCountryCard;
  left: number;
  top: number;
  selected: boolean;
  onSelect: () => void;
}) {
  const edit = useEditCanvas();
  const portrait = country.leader?.portrait_sprite;
  const portraitUrl = countryLeaderPortraitUrl(country.leader);

  const node = useMemo(() => {
    const clone = structuredClone(template.node) as GuiNode;
    for (const child of clone.children ?? []) {
      if (child.name === "country") child.frame = selected ? 2 : 1;

      if (child.name === "new_content" || child.name === "country_flag")
        child.sprite = null;

      if (child.name === "country_leader" && portraitUrl) {
        child.sprite = null;
      }
    }
    assignIds(clone);
    return clone;
  }, [template.node, selected, portraitUrl]);
  const empty = useMemo(() => new Set<string>(), []);

  const printed = useMemo(
    () => countryCardPrinted(node, country),
    [node, country],
  );
  const rects = useMemo(
    () => measureWindow(node, template.sprites, 1920, 1080),
    [node, template.sprites],
  );
  const nameNode = findNode(template.node, "country_name");
  const font = nameNode?.font
    ? template.fonts[nameNode.font]?.bitmap
    : undefined;
  const shield = template.sprites["GFX_shield_medium"];
  const portraitRect = rects["name:country_leader"];
  const flagRect = rects["name:country_flag"];
  const nameRect = rects["name:country_name"];

  return (
    <div style={{ position: "absolute", left, top }}>
      <GuiCanvas
        node={node}
        {...edit(
          `${template.name}:${left},${top}`,
          template,
          node,
          template.sprites,
          undefined,
          printed,
        )}
        texts={template.texts}
        fonts={template.fonts}
        hidden={empty}
        isolateId={null}
        transparent
        baseW={CARD_PITCH}
        baseH={274}
      />
      {portrait && portraitUrl && portraitRect && (
        <div
          style={{
            position: "absolute",
            left: portraitRect.x,
            top: portraitRect.y,
            width: portrait.width,
            height: portrait.height,
            backgroundImage: `url("${portraitUrl}")`,
            backgroundSize: `${portrait.width}px ${portrait.height}px`,
            backgroundRepeat: "no-repeat",
            pointerEvents: "none",
          }}
        />
      )}
      {country.flag?.url && flagRect && (
        <>
          <div
            style={{
              position: "absolute",
              left: flagRect.x,
              top: flagRect.y,
              width: shield?.width || 82,
              height: shield?.height || 52,
              backgroundImage: `url("${country.flag.url}")`,
              backgroundSize: "cover",
              backgroundPosition: "center",
              pointerEvents: "none",
            }}
          />
          {shield?.vanilla_url && (
            <div
              style={{
                position: "absolute",
                left: flagRect.x,
                top: flagRect.y,
                width: shield.width,
                height: shield.height,
                backgroundImage: `url("${shield.vanilla_url}")`,
                backgroundSize: `${shield.width}px ${shield.height}px`,
                pointerEvents: "none",
              }}
            />
          )}
        </>
      )}
      {font && nameRect && (
        <BitmapText
          font={font}
          text={country.name}
          left={nameRect.x}
          top={nameRect.y}
          width={nameNode?.maxwidth || 131}
          height={24}
          align="center"
          vcenter={false}
          maxLines={1}
        />
      )}
      <button
        type="button"
        aria-label={country.tag}
        title={country.name}
        onClick={onSelect}
        style={{
          position: "absolute",
          left: 0,
          top: 0,
          width: CARD_PITCH,
          height: 274,
          background: "transparent",
          border: "none",
          cursor: "inherit",
        }}
      />
    </div>
  );
}

function FilterButton({
  template,
  label,
  left,
  active,
  onToggle,
}: {
  template: GuiWindow;
  label: string;
  left: number;
  active: boolean;
  onToggle: () => void;
}) {
  const edit = useEditCanvas();
  const node = useMemo(() => {
    const clone = structuredClone(template.node) as GuiNode;
    for (const child of clone.children ?? []) {
      if (child.name === "button") child.frame = active ? 2 : 1;
    }
    assignIds(clone);
    return clone;
  }, [template.node, active]);
  const empty = useMemo(() => new Set<string>(), []);
  const labelId = useMemo(() => {
    const button = (node.children ?? []).find(
      (child) => child.name === "button",
    ) as (GuiNode & { _id?: string }) | undefined;
    return button?._id ?? "";
  }, [node]);
  const textOverride = useMemo(
    () => (labelId ? { [labelId]: label } : {}),
    [labelId, label],
  );

  return (
    <div style={{ position: "absolute", left, top: 0 }}>
      <GuiCanvas
        node={node}
        {...edit(
          `${template.name}:${left}`,
          template,
          node,
          template.sprites,
          textOverride,
        )}
        texts={template.texts}
        fonts={template.fonts}
        hidden={empty}
        isolateId={null}
        transparent
        baseW={FILTER_W}
        baseH={FILTER_H}
      />
      <button
        type="button"
        aria-label={`filter_${label}`}
        onClick={onToggle}
        style={{
          position: "absolute",
          left: 0,
          top: 0,
          width: FILTER_W,
          height: FILTER_H,
          background: "transparent",
          border: "none",
          cursor: "inherit",
        }}
      />
    </div>
  );
}

function MiniCountry({
  template,
  country,
  left,
  top,
  selected,
  onSelect,
}: {
  template: GuiWindow;
  country: GuiCountryCard;
  left: number;
  top: number;
  selected: boolean;
  onSelect: () => void;
}) {
  const edit = useEditCanvas();
  const node = useMemo(() => {
    const clone = structuredClone(template.node) as GuiNode;
    for (const child of clone.children ?? []) {
      if (child.name === "country") child.frame = selected ? 2 : 1;
      if (child.name === "country_flag" || child.name === "new_content")
        child.sprite = null;
    }
    assignIds(clone);
    return clone;
  }, [template.node, selected]);
  const empty = useMemo(() => new Set<string>(), []);
  const rects = useMemo(
    () => measureWindow(node, template.sprites, 1920, 1080),
    [node, template.sprites],
  );
  const flagRect = rects["name:country_flag"];
  const flagArt = template.sprites["GFX_flag_small2"];

  return (
    <div style={{ position: "absolute", left, top }}>
      <GuiCanvas
        node={node}
        {...edit(
          `${template.name}:${left},${top}`,
          template,
          node,
          template.sprites,
        )}
        texts={template.texts}
        fonts={template.fonts}
        hidden={empty}
        isolateId={null}
        transparent
        baseW={MINI_W}
        baseH={MINI_H}
      />
      {country.flag?.url && flagRect && (
        <div
          style={{
            position: "absolute",
            left: flagRect.x,
            top: flagRect.y,
            width: flagArt?.width || 41,
            height: flagArt?.height || 26,
            backgroundImage: `url("${country.flag.url}")`,
            backgroundSize: "cover",
            backgroundPosition: "center",
            pointerEvents: "none",
          }}
        />
      )}
      <button
        type="button"
        aria-label={country.tag}
        title={country.name}
        onClick={onSelect}
        style={{
          position: "absolute",
          left: 0,
          top: 0,
          width: MINI_W,
          height: MINI_H,
          background: "transparent",
          border: "none",
          cursor: "inherit",
        }}
      />
    </div>
  );
}

function PreviewIcons({
  items,
  rect,
  pitch,
  max,
}: {
  items: {
    token: string;
    label?: string;
    icon?: string;
    icon_sprite?: GuiSprite;
    tooltip?: string;
  }[];
  rect: MeasuredRect | undefined;
  pitch: number;
  max: number;
}) {
  if (!rect) return null;
  return (
    <>
      {items.slice(0, max).map((item, at) => {
        const url = item.icon_sprite
          ? spriteUrlOf(item.icon_sprite, item.icon)
          : undefined;
        if (!url) return null;
        const width = item.icon_sprite?.width
          ? Math.round(
              item.icon_sprite.width /
                Math.max(1, item.icon_sprite.frames || 1),
            )
          : PREVIEW_ICON;
        return (
          <div
            key={item.token}
            title={item.tooltip ? undefined : item.label || item.token}
            {...(item.tooltip
              ? gameTooltipProps(item.tooltip, { key: `preview:${item.token}` })
              : {})}
            style={{
              position: "absolute",
              left: rect.x + at * pitch,
              top: rect.y,
              width,
              height: item.icon_sprite?.height || PREVIEW_ICON,
              backgroundImage: `url("${url}")`,
              backgroundSize: `${width}px ${item.icon_sprite?.height || PREVIEW_ICON}px`,
              backgroundRepeat: "no-repeat",

              pointerEvents: item.tooltip ? "auto" : "none",
            }}
          />
        );
      })}
    </>
  );
}

function StripCount({
  font,
  left,
  top,
  shown,
  total,
}: {
  font: GuiBitmapFont | undefined;
  left: number;
  top: number;
  shown: number;
  total: number;
}) {
  if (!font || total <= shown) return null;
  return (
    <BitmapText
      font={font}
      text={`${shown} / ${total}`}
      left={left}
      top={top}
      width={90}
      height={font.line + 4}
      align="right"
      vcenter={false}
      maxLines={1}
    />
  );
}

function ValueText({
  win,
  rects,
  name,
  value,
  width,
  maxLines = 1,
}: {
  win: GuiWindow;
  rects: Record<string, MeasuredRect>;
  name: string;
  value: string | undefined;
  width?: number;
  maxLines?: number;
}) {
  const node = findNode(win.node, name);
  const rect = rects[`name:${name}`];
  const font = node?.font ? win.fonts[node.font]?.bitmap : undefined;
  if (!value || !rect || !font) return null;
  return (
    <BitmapText
      font={font}
      text={value}
      left={rect.x}
      top={rect.y}
      width={width || node?.maxwidth || 300}
      height={maxLines * (font.line + 2)}

      align={textAlignOf(node?.format)}
      vcenter={false}
      maxLines={maxLines}
    />
  );
}

export function CountrySelectLayer({
  win,
  projectId,
  initialTag,
  bookmarkIndex = 0,
  onConfirm,
}: {
  win: GuiWindow;
  projectId: number;

  initialTag: string | null;

  bookmarkIndex?: number;

  onConfirm: (tag: string, focusTree: string | null) => void;
}) {
  const bookmarks = useBookmarks(projectId, win.name === COUNTRY_PICKER);
  const template = useTemplateWindow(
    projectId,
    bookmarks.data ? COUNTRY_ENTRY : null,
  );
  const miniTemplate = useTemplateWindow(
    projectId,
    bookmarks.data ? MINI_ENTRY : null,
  );
  const filterTemplate = useTemplateWindow(
    projectId,
    bookmarks.data ? FILTER_ENTRY : null,
  );
  const rects = useMemo(
    () => measureWindow(win.node, win.sprites, 1920, 1080),
    [win],
  );
  const [scroll, setScroll] = useState(0);
  const [minorPage, setMinorPage] = useState(0);
  const [picked, setPicked] = useState<string | null>(initialTag);
  const [continent, setContinent] = useState<string | null>(null);
  const mark =
    bookmarks.data?.bookmarks?.[bookmarkIndex] ??
    bookmarks.data?.bookmarks?.[0];
  const countries = useMemo(
    () => (mark?.countries ?? []).filter((c) => !c.placeholder),
    [mark],
  );

  const minors = useMemo(() => {
    const all = bookmarks.data?.minors ?? [];
    return continent
      ? all.filter((country) => country.continent === continent)
      : all;
  }, [bookmarks.data, continent]);
  const minorsPerPage = MINI_COLS * MINI_ROWS;

  const selected = useMemo(() => {
    const byTag =
      countries.find((c) => c.tag === picked) ??
      minors.find((c) => c.tag === picked);
    if (byTag) return byTag;
    return (
      countries.find((c) => c.tag === mark?.default_country) ?? countries[0]
    );
  }, [countries, minors, picked, mark]);
  const party =
    selected?.ruling_party_share !== undefined
      ? `${selected.ruling_party_label ?? selected.ruling_party} ${selected.ruling_party_share}%`
      : (selected?.ruling_party_label ?? selected?.ruling_party);

  const printed = useMemo(
    () => countryPickPrinted(win, selected, party),
    [win, selected, party],
  );
  usePrintedOver(win, "country-select", printed);

  if (win.name !== COUNTRY_PICKER || !bookmarks.data) return null;
  const gridRect = rects["name:countries_grid"];
  const stripRect = rects["name:countries"];
  const bigPortrait = rects["name:minor_country_leader"];
  const minorRect = rects["name:countries_mini"];
  const spiritRect = rects["name:ideas_grid"];
  const selectRect = rects["name:select_button"];
  const filterRect = rects["name:filters"];

  const countFont = filterTemplate.data
    ? filterTemplate.data.fonts[
        (filterTemplate.data.node.children ?? []).find(
          (child) => child.name === "button",
        )?.font ?? ""
      ]?.bitmap
    : undefined;
  const majorsShown = Math.min(
    countries.length,
    Math.floor((1225 - (stripRect?.x ?? 0) - 20) / CARD_PITCH),
  );
  const continents = mark?.continents ?? [];
  const focusRect = rects["name:focus_grid"];
  const leader = selected?.leader;
  const leaderPortraitUrl = countryLeaderPortraitUrl(leader);
  const elections =
    selected?.elections_allowed === undefined
      ? undefined
      : selected.elections_allowed
        ? "Yes"
        : "No";

  return (
    <div
      style={{ position: "absolute", left: 0, top: 0, pointerEvents: "none" }}
    >
      {template.data && gridRect && stripRect && (
        <div
          style={{
            position: "absolute",
            left: stripRect.x,

            top: gridRect.y,
            width: 1225 - stripRect.x - 20,
            height: 290,
            overflow: "hidden",
            pointerEvents: "auto",
          }}
          onWheel={(e) => {
            const span = Math.max(
              0,
              countries.length * CARD_PITCH - (1225 - stripRect.x - 20),
            );
            setScroll((at) =>
              Math.min(
                span,
                Math.max(0, at + (e.deltaY > 0 ? CARD_PITCH : -CARD_PITCH)),
              ),
            );
          }}
        >
          {countries.map((country, at) => (
            <CountryCard
              key={country.tag}
              template={template.data as GuiWindow}
              country={country}
              left={at * CARD_PITCH - scroll}
              top={0}
              selected={selected?.tag === country.tag}
              onSelect={() => setPicked(country.tag)}
            />
          ))}
        </div>
      )}

      {filterTemplate.data && filterRect && continents.length > 0 && (
        <div
          style={{
            position: "absolute",
            left: filterRect.x,
            top: filterRect.y,
            pointerEvents: "auto",
          }}
        >
          {continents.map((item, at) => (
            <FilterButton
              key={item.name}
              template={filterTemplate.data as GuiWindow}
              label={item.label || item.name}
              left={at * (FILTER_W + FILTER_GAP)}
              active={continent === item.name}
              onToggle={() => {
                setContinent((now) => (now === item.name ? null : item.name));
                setMinorPage(0);
              }}
            />
          ))}
        </div>
      )}

      {miniTemplate.data && minorRect && (
        <div
          style={{
            position: "absolute",
            left: minorRect.x,
            top: minorRect.y,
            width: MINI_COLS * MINI_W,
            height: MINI_ROWS * MINI_H,
            overflow: "hidden",
            pointerEvents: "auto",
          }}
          onWheel={(e) => {
            const pages = Math.ceil(minors.length / minorsPerPage) - 1;
            setMinorPage((at) =>
              Math.min(pages, Math.max(0, at + (e.deltaY > 0 ? 1 : -1))),
            );
          }}
        >
          {minors
            .slice(minorPage * minorsPerPage, (minorPage + 1) * minorsPerPage)
            .map((country, at) => (
              <MiniCountry
                key={country.tag}
                template={miniTemplate.data as GuiWindow}
                country={country}
                left={(at % MINI_COLS) * MINI_W}
                top={Math.floor(at / MINI_COLS) * MINI_H}
                selected={selected?.tag === country.tag}
                onSelect={() => setPicked(country.tag)}
              />
            ))}
        </div>
      )}

      {stripRect && gridRect && (
        <StripCount
          font={countFont}
          left={1225 - 150}

          top={gridRect.y - 20}
          shown={majorsShown}
          total={countries.length}
        />
      )}
      {minorRect && (
        <StripCount
          font={countFont}
          left={minorRect.x + MINI_COLS * MINI_W - 92}
          top={minorRect.y + MINI_ROWS * MINI_H + 2}
          shown={Math.min(minors.length, minorsPerPage)}
          total={minors.length}
        />
      )}

      {leader?.portrait_sprite && leaderPortraitUrl && bigPortrait && (
        <div
          style={{
            position: "absolute",
            left: bigPortrait.x,
            top: bigPortrait.y,
            width: leader.portrait_sprite.width,
            height: leader.portrait_sprite.height,
            backgroundImage: `url("${leaderPortraitUrl}")`,
            backgroundSize: `${leader.portrait_sprite.width}px ${leader.portrait_sprite.height}px`,
            backgroundRepeat: "no-repeat",

            transform: `rotate(${Number(findNode(win.node, "minor_country_leader")?.raw?.rotation ?? 0)}rad)`,
          }}
        />
      )}

      <PreviewIcons
        items={selected?.ideas ?? []}
        rect={spiritRect}
        pitch={SPIRIT_PITCH}
        max={3}
      />
      <PreviewIcons
        items={selected?.focuses ?? []}
        rect={focusRect}
        pitch={FOCUS_PITCH}
        max={4}
      />

      <ValueText
        win={win}
        rects={rects}
        name="nation_title"
        value={selected?.name}
        width={340}
      />
      <ValueText
        win={win}
        rects={rects}
        name="leader"
        value={leader?.name}
        width={240}
      />
      <ValueText
        win={win}
        rects={rects}
        name="ideology"
        value={selected?.ideology_label}
        width={240}
      />
      <ValueText
        win={win}
        rects={rects}
        name="government"
        value={selected?.ruling_party_label}
        width={240}
      />
      <ValueText
        win={win}
        rects={rects}
        name="elections"
        value={elections}
        width={240}
      />
      <ValueText
        win={win}
        rects={rects}
        name="party"
        value={party}
        width={240}
      />
      <ValueText
        win={win}
        rects={rects}
        name="history"
        value={selected?.history}
        width={330}
        maxLines={9}
      />

      {selectRect && selected && (
        <button
          type="button"
          aria-label="select_country"
          onClick={() => onConfirm(selected.tag, selected.focus_tree ?? null)}
          style={{
            position: "absolute",
            left: selectRect.x,
            top: selectRect.y,
            width: selectRect.w || 148,
            height: selectRect.h || 34,
            background: "transparent",
            border: "none",
            cursor: "inherit",
            pointerEvents: "auto",
          }}
        />
      )}
    </div>
  );
}
