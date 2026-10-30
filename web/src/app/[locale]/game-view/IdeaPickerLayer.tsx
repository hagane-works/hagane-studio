"use client";

import { useMemo, useState } from "react";
import { useTranslations } from "use-intl";
import useSWR from "swr";
import { guiArray, readGuiResponse } from "./guiResponse";
import type {
  GuiCharacter,
  GuiNode,
  GuiSprite,
  GuiWindow,
} from "@/lib/guiRenderApi";
import { withGameLanguage } from "@/lib/guiRenderApi";
import { GuiCanvas, assignIds, measureWindow } from "../gui-renderer/GuiCanvas";
import { BitmapText } from "../gui-renderer/BitmapText";
import { RichText } from "../gui-renderer/RichText";
import { useTemplateWindow } from "./CountrySelectLayer";
import { useEditCanvas } from "./editContext";
import { namedPrinted, personPrinted } from "./textSources";
import { FieldHit } from "./FieldHit";
import { gameTooltipProps } from "../gui-renderer/GameTooltip";
import { matchNamePath } from "./gameNavigation";

export const IDEA_WINDOW = "political_ideas_window";
const ENTRY = "political_selectable_idea_entry_list";
const GRID_ENTRY = "political_selectable_idea_entry_grid";

const ROW_H = 74;
const GRID_W = 231;

const WHEEL_STEP = 37;
const NO_HIDDEN = new Set<string>();

export const LAW_GROUPS = new Set([
  "mobilization_laws",
  "trade_laws",
  "economy",
]);

export const CARD_GROUPS = new Set(["industrial_concern"]);

export const CHARACTER_SLOTS = new Set([
  "political_advisor",
  "theorist",
  "army_chief",
  "navy_chief",
  "air_chief",
  "high_command",
]);

export interface GuiIdeaChoice {
  token: string;
  label?: string;
  icon?: string;
  icon_sprite?: GuiSprite;
  cost?: number;

  stats?: string;

  trait?: string;
  current?: boolean;
}

export interface GuiPickerPerson {
  token: string;
  name: string;
  portrait?: string;
  portrait_sprite?: GuiSprite;

  trait?: string;
  cost?: number;

  dim?: boolean;

  idea?: string;

  hired?: boolean;

  tooltip?: string;
}

export interface GuiIdeaChoices {
  group: string;
  title: string;

  subtitle?: string;
  current?: string;
  current_label?: string;
  ideas: GuiIdeaChoice[];

  people?: GuiPickerPerson[];

  layout?: string;
}

function ideaCards(ideas: readonly GuiIdeaChoice[]): GuiPickerPerson[] {
  return ideas.map((idea) => ({
    token: idea.token,
    name: idea.label || idea.token,
    portrait: idea.icon,
    portrait_sprite: idea.icon_sprite,
    trait: idea.trait,
    cost: idea.cost,
  }));
}

export function advisorChoices(
  people: readonly GuiCharacter[] | undefined,
  slot: string,
  label: string,
  showHidden = false,
  selectText?: string,
): GuiIdeaChoices {
  const rank = (person: GuiCharacter) =>
    person.shown === false ? 2 : person.available ? 0 : 1;
  const out = (people ?? [])
    .filter(
      (person) =>
        person.slot === slot && (showHidden || person.shown !== false),
    )
    .sort((a, b) => rank(a) - rank(b) || a.name.localeCompare(b.name))
    .map((person) => ({
      token: person.token,
      name: person.name,
      portrait: person.portrait,
      portrait_sprite: person.portrait_sprite,
      trait: person.traits?.[0]?.label,
      cost: person.cost,
      dim: person.shown === false,
      idea: person.idea_token || person.token,
      hired: !!person.hired,
      tooltip: person.tooltip,
    }));
  return {
    group: slot,
    title: label,
    subtitle: selectText || `§Y${label}§!`,
    ideas: [],
    people: out,
  };
}

export function useIdeaChoices(
  projectId: number,
  tag: string | null,
  date: string | null | undefined,
  group: string | null,
) {
  return useSWR<GuiIdeaChoices>(
    tag && group
      ? ["game-view-idea-choices", projectId, tag, date ?? "", group]
      : null,
    async () => {
      const query = new URLSearchParams({
        tag: tag as string,
        group: group as string,
      });
      if (date) query.set("date", date);
      const response = await fetch(
        withGameLanguage(
          `/api/go/projects/${projectId}/gui-render/idea-choices?${query}`,
        ),
      );
      return readGuiResponse<GuiIdeaChoices>(response);
    },
    { revalidateOnFocus: false },
  );
}

function idOf(node: GuiNode | undefined): string {
  return (node as (GuiNode & { _id?: string }) | undefined)?._id ?? "";
}

export function ideaWindowTexts(
  win: GuiWindow,
  choices: GuiIdeaChoices | undefined,
): Record<string, string> {
  if (win.name !== IDEA_WINDOW || !choices) return {};
  const out: Record<string, string> = {};
  const title = idOf(matchNamePath(win.node, "title")[0]);
  const replace = idOf(matchNamePath(win.node, "switch")[0]);
  if (title) out[title] = choices.title;
  if (replace) {
    out[replace] = choices.subtitle ?? "";
  }
  return out;
}

function IdeaRow({
  entry,
  idea,
  left,
  top,
  onChoose,
}: {
  entry: GuiWindow;
  idea: GuiIdeaChoice;
  left: number;
  top: number;

  onChoose?: () => void;
}) {
  const t = useTranslations("gameViewEdit.politics");
  const edit = useEditCanvas();
  const held = !!idea.current;
  const iconName = idea.icon && idea.icon_sprite ? idea.icon : null;
  const node = useMemo(() => {
    const clone = structuredClone(entry.node) as GuiNode;
    const walk = (at: GuiNode) => {
      for (const child of at.children ?? []) {
        if (child.name === "idea_entry_bg") child.frame = held ? 2 : 1;
        if (child.name === "icon") child.sprite = iconName;

        if (child.name === "idea_traits") child.sprite = null;
        walk(child);
      }
    };
    walk(clone);
    assignIds(clone);
    return clone;
  }, [entry.node, held, iconName]);
  const sprites = useMemo(
    () =>
      idea.icon && idea.icon_sprite
        ? { ...entry.sprites, [idea.icon]: idea.icon_sprite }
        : entry.sprites,
    [entry.sprites, idea.icon, idea.icon_sprite],
  );

  const blanks = useMemo(() => {
    const out: Record<string, string> = {};
    const walk = (at: GuiNode) => {
      for (const child of at.children ?? []) {
        const id = idOf(child);
        if (
          id &&
          (child.name === "name" ||
            child.name === "traits" ||
            child.name === "cost" ||
            child.name === "stats")
        ) {
          out[id] = "";
        }
        walk(child);
      }
    };
    walk(node);
    return out;
  }, [node]);

  const printed = useMemo(
    () => namedPrinted(node, "name", idea.label || idea.token, idea.token),
    [node, idea.label, idea.token],
  );
  const rects = useMemo(
    () => measureWindow(node, sprites, 1920, 1080),
    [node, sprites],
  );
  const nameNode = matchNamePath(entry.node, "traits_container/name")[0];
  const costNode = matchNamePath(entry.node, "cost")[0];
  const statsNode = matchNamePath(entry.node, "stats")[0];
  const font = (at: GuiNode | undefined) =>
    at?.font ? entry.fonts[at.font]?.bitmap : undefined;
  const nameRect = rects["name:name"];
  const costRect = rects["name:cost"];
  const statsRect = rects["name:stats"];
  const nameFont = font(nameNode);
  const costFont = font(costNode);
  const statsFont = font(statsNode);
  return (
    <div style={{ position: "absolute", left, top }}>
      <GuiCanvas
        node={node}
        {...edit(
          `${entry.name}:${idea.token}`,
          { ...entry, sprites },
          node,
          sprites,
          blanks,
          printed,
        )}
        texts={entry.texts}
        fonts={entry.fonts}
        hidden={NO_HIDDEN}
        isolateId={null}
        transparent
        baseW={462}
        baseH={ROW_H}
      />
      {nameFont && nameRect && (
        <BitmapText
          font={nameFont}
          text={idea.label || idea.token}
          left={nameRect.x}
          top={0}
          width={nameNode?.maxwidth || 160}
          height={ROW_H - 13}
          align="left"
          vcenter
          maxLines={2}
        />
      )}
      {costFont && costRect && idea.cost !== undefined && (
        <RichText
          font={costFont}
          markup={`£pol_power ${Math.round(idea.cost)}`}
          left={costRect.x}
          top={costRect.y - 4}
          width={costNode?.maxwidth || 142}
          align="center"
          vcenter={false}
          maxLines={1}
        />
      )}
      {statsFont && statsRect && idea.stats && (
        <RichText
          font={statsFont}
          markup={idea.stats}
          left={statsRect.x}
          top={statsRect.y - 4}
          width={statsNode?.maxwidth || 142}
          align="center"
          vcenter={false}
          maxLines={1}
        />
      )}
      {onChoose && nameRect && (
        <FieldHit
          x={nameRect.x}
          y={6}
          w={nameNode?.maxwidth || 160}
          h={ROW_H - 12}
          label={t("setAsStart", { name: idea.label || idea.token })}
          onClick={onChoose}
        />
      )}
    </div>
  );
}

function PersonCard({
  entry,
  person,
  left,
  top,
  onChoose,
  onEdit,
}: {
  entry: GuiWindow;
  person: GuiPickerPerson;
  left: number;
  top: number;

  onChoose?: () => void;

  onEdit?: () => void;
}) {
  const t = useTranslations("gameViewEdit.politics");
  const edit = useEditCanvas();
  const portrait =
    person.portrait && person.portrait_sprite ? person.portrait : null;
  const node = useMemo(() => {
    const clone = structuredClone(entry.node) as GuiNode;
    const walk = (at: GuiNode) => {
      for (const child of at.children ?? []) {
        if (child.name === "icon") child.sprite = portrait;

        if (child.name === "idea_traits") child.sprite = null;
        walk(child);
      }
    };
    walk(clone);
    assignIds(clone);
    return clone;
  }, [entry.node, portrait]);
  const sprites = useMemo(
    () =>
      portrait && person.portrait_sprite
        ? { ...entry.sprites, [portrait]: person.portrait_sprite }
        : entry.sprites,
    [entry.sprites, portrait, person.portrait_sprite],
  );
  const blanks = useMemo(() => {
    const out: Record<string, string> = {};
    const walk = (at: GuiNode) => {
      for (const child of at.children ?? []) {
        const id = idOf(child);
        if (
          id &&
          (child.name === "name" ||
            child.name === "traits" ||
            child.name === "cost" ||
            child.name === "stats")
        ) {
          out[id] = "";
        }
        walk(child);
      }
    };
    walk(node);
    return out;
  }, [node]);

  const printed = useMemo(
    () => personPrinted(node, "name", person),
    [node, person],
  );
  const rects = useMemo(
    () => measureWindow(node, sprites, 1920, 1080),
    [node, sprites],
  );
  const nameNode = matchNamePath(entry.node, "traits_container/name")[0];
  const traitNode = matchNamePath(entry.node, "traits_container/traits")[0];
  const costNode = matchNamePath(entry.node, "cost")[0];
  const font = (at: GuiNode | undefined) =>
    at?.font ? entry.fonts[at.font]?.bitmap : undefined;
  const nameRect = rects["name:name"];
  const traitRect = rects["name:traits"];
  const costRect = rects["name:cost"];
  const portraitRect = rects["name:icon"];
  const nameFont = font(nameNode);
  const traitFont = font(traitNode);
  const costFont = font(costNode);
  return (
    <div
      style={{ position: "absolute", left, top, width: GRID_W, height: ROW_H }}
      {...(person.tooltip
        ? gameTooltipProps(person.tooltip, { key: `advisor:${person.token}` })
        : {})}
    >
      <GuiCanvas
        node={node}
        {...edit(
          `${entry.name}:${person.token}`,
          { ...entry, sprites },
          node,
          sprites,
          blanks,
          printed,
        )}
        texts={entry.texts}
        fonts={entry.fonts}
        hidden={NO_HIDDEN}
        isolateId={null}
        transparent
        baseW={GRID_W}
        baseH={ROW_H}
      />
      {nameFont && nameRect && (
        <BitmapText
          font={nameFont}
          text={person.name}
          left={nameRect.x}
          top={nameRect.y}
          width={nameNode?.maxwidth || 150}
          height={nameNode?.maxheight || 13}
          align="left"
          vcenter={false}
          maxLines={1}
        />
      )}
      {traitFont && traitRect && person.trait && (
        <BitmapText
          font={traitFont}
          text={person.trait}
          left={traitRect.x}
          top={traitRect.y}
          width={traitNode?.maxwidth || 140}
          height={ROW_H - traitRect.y}
          align="left"
          vcenter={false}
          maxLines={2}
        />
      )}
      {costFont && costRect && person.cost !== undefined && (
        <RichText
          font={costFont}
          markup={`£pol_power ${Math.round(person.cost)}`}
          left={costRect.x}
          top={costRect.y - 4}
          width={costNode?.maxwidth || 142}
          align="right"
          vcenter={false}
          maxLines={1}
        />
      )}
      {onChoose && nameRect && (
        <FieldHit
          x={nameRect.x}
          y={nameRect.y}
          w={nameNode?.maxwidth || 150}
          h={40}
          label={t(
            person.idea && person.hired ? "unhireAtStart" : "setAsStart",
            { name: person.name },
          )}
          onClick={onChoose}
        />
      )}
      {onEdit && portraitRect && (
        <FieldHit
          x={portraitRect.x}
          y={portraitRect.y}
          w={portraitRect.w || 60}
          h={portraitRect.h || 64}
          label={t("editPerson", { name: person.name })}
          onClick={onEdit}
        />
      )}
    </div>
  );
}

export function IdeaPickerLayer({
  win,
  projectId,
  choices,
  onChoose,
  onEditPerson,
}: {
  win: GuiWindow;
  projectId: number;
  choices: GuiIdeaChoices | undefined;

  onChoose?: (token: string) => void;

  onEditPerson?: (token: string, name: string) => void;
}) {
  const active = win.name === IDEA_WINDOW;
  const ideas = guiArray(choices?.ideas);
  const hasPeople = Array.isArray(choices?.people);
  const people = useMemo(
    () =>
      hasPeople
        ? guiArray(choices?.people)
        : choices?.layout === "grid"
          ? ideaCards(ideas)
          : undefined,
    [choices, hasPeople, ideas],
  );

  const editPerson = hasPeople ? onEditPerson : undefined;
  const entry = useTemplateWindow(
    projectId,
    active && choices && !people ? ENTRY : null,
  );
  const card = useTemplateWindow(
    projectId,
    active && people ? GRID_ENTRY : null,
  );
  const rects = useMemo(
    () => (active ? measureWindow(win.node, win.sprites, 1920, 1080) : {}),
    [active, win],
  );
  const [scroll, setScroll] = useState(0);
  const list = rects[people ? "name:box_grid" : "name:box_list"];
  const container = rects["name:grid"];

  const grid = matchNamePath(win.node, "grid")[0];
  const marginTop =
    Number((grid?.raw?.margin as { top?: number } | undefined)?.top) || 0;
  if (!active || !choices || !list) return null;
  const rows = people ? Math.ceil(people.length / 2) : ideas.length;

  const viewTop = list.y + marginTop;
  const winH = Number(win.node.size?.height) || 590;
  const viewH = Math.max(
    0,
    Math.min(winH, container ? container.y + container.h : winH) - viewTop,
  );
  const maxScroll = Math.max(0, rows * ROW_H - viewH);
  const at = Math.min(scroll, maxScroll);
  return (
    <div
      style={{ position: "absolute", left: 0, top: 0, pointerEvents: "none" }}
    >
      <div
        style={{
          position: "absolute",
          left: list.x,
          top: viewTop,
          width: 2 * GRID_W + 10,
          height: viewH,
          overflow: "hidden",
          pointerEvents: "auto",
        }}
        onWheel={(e) => {
          const step = e.deltaY > 0 ? WHEEL_STEP : -WHEEL_STEP;
          setScroll((now) =>
            Math.min(maxScroll, Math.max(0, Math.min(now, maxScroll) + step)),
          );
        }}
      >
        {people && card.data
          ? people.map((person, index) => (
              <div
                key={person.token}
                style={person.dim ? { opacity: 0.45 } : undefined}
              >
                <PersonCard
                  entry={card.data as GuiWindow}
                  person={person}
                  left={(index % 2) * GRID_W}
                  top={Math.floor(index / 2) * ROW_H - at}
                  onChoose={onChoose ? () => onChoose(person.token) : undefined}
                  onEdit={
                    editPerson
                      ? () => editPerson(person.token, person.name)
                      : undefined
                  }
                />
              </div>
            ))
          : entry.data
            ? ideas.map((idea, index) => (
                <IdeaRow
                  key={idea.token}
                  entry={entry.data as GuiWindow}
                  idea={idea}
                  left={0}
                  top={index * ROW_H - at}
                  onChoose={onChoose ? () => onChoose(idea.token) : undefined}
                />
              ))
            : null}
      </div>
    </div>
  );
}
