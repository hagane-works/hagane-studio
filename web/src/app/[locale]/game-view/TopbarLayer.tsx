"use client";

import { useContext, useMemo } from "react";
import { useTranslations } from "use-intl";
import useSWR from "swr";
import {
  guiRenderApi,
  type GuiCountryStats,
  type GuiFontMeta,
  type GuiNode,
  type GuiSprite,
  type GuiText,
} from "@/lib/guiRenderApi";
import { fontTextColors } from "@/lib/referenceTables";
import {
  GUI_UNHANDLED_ACTIVATE_EVENT,
  GuiCanvas,
  measureWindow,
  textAlignOf,
  type GuiUnhandledActivateDetail,
  type MeasuredRect,
} from "../gui-renderer/GuiCanvas";
import { BitmapText } from "../gui-renderer/BitmapText";
import { gameTooltipProps } from "../gui-renderer/GameTooltip";
import { normalizeTooltip } from "../gui-renderer/tooltipText";
import { BLEND_FRAMES_ANIMATION, BlendFramesKeyframes } from "./blendFrames";
import { frameStyle, spriteUrlOf } from "./spriteFrame";
import {
  buildTopbarModel,
  topbarExtraSprites,
  type TopbarModel,
  type TopbarUi,
} from "./topbarModel";
import { TOPBAR_LOC_KEYS } from "./topbarTooltips";
import { hiddenExceptNamed, nodesNamed } from "./topbarLayout";
import { FieldHit } from "./FieldHit";
import { EditContext, useEditCanvas } from "./editContext";

const DISABLED_BUTTONS = ["dismissed_alerts_button"] as const;

const SCREEN_W = 1920;
const SCREEN_H = 1080;
const NO_LOC: Record<string, string> = {};
const NO_SPRITES: Record<string, GuiSprite> = {};
const NO_HIDDEN = new Set<string>();

const LOC_CHUNK = 64;

export function useTopbarLoc(projectId: number, enabled: boolean) {
  return useSWR<Record<string, string>>(
    enabled ? ["game-view-topbar-loc", projectId] : null,
    async () => {
      guiRenderApi.setProjectId(projectId);
      const keys = [...TOPBAR_LOC_KEYS];
      const parts: Promise<Record<string, string>>[] = [];
      for (let at = 0; at < keys.length; at += LOC_CHUNK)
        parts.push(
          guiRenderApi.loc(keys.slice(at, at + LOC_CHUNK), { markup: true }),
        );
      return Object.assign({}, ...(await Promise.all(parts))) as Record<
        string,
        string
      >;
    },
    { revalidateOnFocus: false },
  );
}

export function useTopbar({
  win,
  projectId,
  stats,
  date,
  ui,
}: {
  win: TopbarModel["win"];
  projectId: number;
  stats: GuiCountryStats | undefined;
  date: string | null | undefined;
  ui: TopbarUi;
}): TopbarModel | null {
  const active = win.name === "topbar";
  const loc = useTopbarLoc(projectId, active);
  const names = useMemo(
    () => (active ? topbarExtraSprites(stats) : []),
    [active, stats],
  );
  const extra = useSWR<Record<string, GuiSprite>>(
    names.length
      ? ["game-view-sprites", projectId, "topbar", names.join(",")]
      : null,
    () => {
      guiRenderApi.setProjectId(projectId);
      return guiRenderApi.sprites(names);
    },
    { revalidateOnFocus: false },
  );
  const { paused, speed, litButton, music } = ui;
  return useMemo(
    () =>
      active
        ? buildTopbarModel({
            win,
            stats,
            date,
            ui: { paused, speed, litButton, music },
            loc: loc.data ?? NO_LOC,
            extraSprites: extra.data ?? NO_SPRITES,
          })
        : null,
    [
      active,
      win,
      stats,
      date,
      paused,
      speed,
      litButton,
      music,
      loc.data,
      extra.data,
    ],
  );
}

type ProgressSprite = GuiSprite & {
  size?: { x: number; y: number };
  texture_urls?: string[];
};

const BAR_NUDGE_Y = 1;

function RatioBar({
  rect,
  sprite,
  ratio,
}: {
  rect: MeasuredRect;
  sprite: ProgressSprite;
  ratio: number;
}) {
  const w =
    sprite.size?.x ||
    Math.round(sprite.width / Math.max(1, sprite.frames)) ||
    45;
  const h = sprite.size?.y || Math.round(sprite.height / 3) || 8;
  const fill = sprite.texture_urls?.[0] || sprite.vanilla_url;
  const empty = sprite.texture_urls?.[1];
  const filled = Math.round(w * Math.min(1, Math.max(0, ratio)));
  return (
    <div
      style={{
        position: "absolute",
        left: rect.x,
        top: rect.y + BAR_NUDGE_Y,
        width: w,
        height: h,
        pointerEvents: "none",
      }}
    >
      {empty && filled < w && (
        <div
          style={{
            position: "absolute",
            left: filled,
            top: 0,
            width: w - filled,
            height: h,
            backgroundImage: `url("${empty}")`,
            backgroundPosition: `${-filled}px 0`,
            backgroundSize: `${w}px ${h}px`,
            backgroundRepeat: "no-repeat",
          }}
        />
      )}
      {fill && filled > 0 && (
        <div
          style={{
            position: "absolute",
            left: 0,
            top: 0,
            width: filled,
            height: h,

            backgroundImage: `url("${fill}"), url("${fill}")`,
            backgroundPosition: `${filled - w}px ${-2 * h}px, 0 ${-h}px`,
            backgroundSize: `${w}px ${3 * h}px, ${w}px ${3 * h}px`,
            backgroundRepeat: "no-repeat",
          }}
        />
      )}
    </div>
  );
}

function DisabledButton({
  node,
  rect,
  sprite,
  tooltip,
}: {
  node: GuiNode;
  rect: MeasuredRect;
  sprite: GuiSprite;
  tooltip: ReturnType<TopbarModel["tooltipFor"]>;
}) {
  const content = normalizeTooltip(tooltip ?? null);
  return (
    <div
      role="button"
      aria-label={node.name ?? undefined}
      aria-disabled
      onClick={() =>
        window.dispatchEvent(
          new CustomEvent<GuiUnhandledActivateDetail>(
            GUI_UNHANDLED_ACTIVATE_EVENT,
            {
              detail: { control: node.name || node.type, owner: "topbar" },
            },
          ),
        )
      }
      style={{
        ...frameStyle(sprite, rect.x, rect.y, {
          frame: node.frame ?? 1,
          name: node.sprite,
        }),
        filter: "grayscale(1)",
        pointerEvents: "auto",
      }}
      {...(content
        ? gameTooltipProps(content, { key: `topbar:${node.name}` })
        : {})}
    />
  );
}

function DisabledButtonsEdit({ win }: { win: TopbarModel["win"] }) {
  const edit = useEditCanvas();
  const hidden = useMemo(
    () => hiddenExceptNamed(win.node, DISABLED_BUTTONS),
    [win.node],
  );
  return (
    <div
      style={{ position: "absolute", left: 0, top: 0, pointerEvents: "none" }}
    >
      <GuiCanvas
        node={win.node}
        {...edit("topbar:disabled", win, win.node, win.sprites)}
        texts={win.texts}
        fonts={win.fonts}
        hidden={hidden}
        isolateId={null}
        transparent
        baseW={SCREEN_W}
        baseH={SCREEN_H}
      />
    </div>
  );
}

export function TopbarLayer({
  model,
  stats,
  onEditCountry,
  onPressFlag,
}: {
  model: TopbarModel | null;
  stats: GuiCountryStats | undefined;

  onEditCountry?: () => void;

  onPressFlag?: (flag: GuiNode) => void;
}) {
  const tEdit = useTranslations("gameViewEdit");
  const win = model?.win;
  const rects = useMemo(
    () => (win ? measureWindow(win.node, win.sprites, SCREEN_W, SCREEN_H) : {}),
    [win],
  );
  const { editMode: uiEdit } = useContext(EditContext);
  if (!model || !win) return null;
  const nodeOf = (name: string) => nodesNamed(win.node, name)[0];

  const flagRect = rects["name:player_flag"];
  const shield = win.sprites["GFX_shield_medium"];
  const flagW = shield?.width || 82;
  const flagH = shield?.height || 52;

  const fuelNode = nodeOf("fuel_value");
  const fuelRect = rects["name:fuel_value"];
  const fuelFont = fuelNode?.font
    ? win.fonts[fuelNode.font]?.bitmap
    : undefined;
  const threatNode = nodeOf("threat_button");
  const threatRect = rects["name:threat_button"];
  const threatTooltip = normalizeTooltip(
    threatNode ? (model.tooltipFor(threatNode) ?? null) : null,
  );

  return (
    <>
      {stats?.flag && flagRect && (
        <div
          style={{
            position: "absolute",
            left: flagRect.x,
            top: flagRect.y,
            width: flagW,
            height: flagH,
            pointerEvents: "none",
          }}
        >
          <div
            style={{
              position: "absolute",
              inset: 0,
              backgroundImage: `url("${stats.flag.url}")`,
              backgroundSize: "cover",
              backgroundPosition: "center",
            }}
          />
          {shield?.vanilla_url && (
            <div
              style={{
                position: "absolute",
                inset: 0,
                backgroundImage: `url("${shield.vanilla_url}")`,
                backgroundSize: `${flagW}px ${flagH}px`,
              }}
            />
          )}
        </div>
      )}
      {onEditCountry && flagRect && (
        <FieldHit
          x={flagRect.x}
          y={flagRect.y}
          w={flagW}
          h={flagH}
          label={tEdit("country.profile")}
          onClick={onEditCountry}
          onPress={
            onPressFlag && nodeOf("player_flag")
              ? () => onPressFlag(nodeOf("player_flag"))
              : undefined
          }
        />
      )}
      {Object.entries(model.bars).map(([name, ratio]) => {
        const node = nodeOf(name);
        const rect = rects[`name:${name}`];
        const sprite = node?.sprite
          ? (win.sprites[node.sprite] as ProgressSprite | undefined)
          : undefined;
        if (!rect || !sprite?.resolved) return null;
        return (
          <RatioBar key={name} rect={rect} sprite={sprite} ratio={ratio} />
        );
      })}
      {model.fuel?.text && fuelNode && fuelRect && fuelFont && (
        <BitmapText
          font={fuelFont}
          text={model.fuel.text}
          left={fuelRect.x}
          top={fuelRect.y}
          width={fuelNode.maxwidth || fuelRect.w}
          align={textAlignOf(fuelNode.format, "left")}
          vcenter={false}
          maxLines={1}
          color={
            model.fuel.color
              ? fontTextColors("hoi_18mbs")[model.fuel.color]
              : undefined
          }
        />
      )}
      {uiEdit ? (
        <DisabledButtonsEdit win={win} />
      ) : (
        DISABLED_BUTTONS.map((name) => {
          const node = nodeOf(name);
          const rect = rects[`name:${name}`];
          const sprite = node?.sprite ? win.sprites[node.sprite] : undefined;
          if (
            !node ||
            !rect ||
            !sprite?.resolved ||
            !spriteUrlOf(sprite, node.sprite)
          )
            return null;
          return (
            <DisabledButton
              key={name}
              node={node}
              rect={rect}
              sprite={sprite}
              tooltip={model.tooltipFor(node)}
            />
          );
        })
      )}
      {!uiEdit && threatNode && threatRect && (
        <div
          role="button"
          aria-label="threat_button"
          onClick={() =>
            window.dispatchEvent(
              new CustomEvent<GuiUnhandledActivateDetail>(
                GUI_UNHANDLED_ACTIVATE_EVENT,
                {
                  detail: { control: "threat_button", owner: "topbar" },
                },
              ),
            )
          }
          style={{
            position: "absolute",
            left: threatRect.x,
            top: threatRect.y,
            width: threatRect.w || 48,
            height: threatRect.h || 48,
            pointerEvents: "auto",
          }}
          {...(threatTooltip
            ? gameTooltipProps(threatTooltip, { key: "topbar:threat_button" })
            : {})}
        />
      )}
    </>
  );
}

export function TopbarClock({
  model,
  texts,
  fonts,
  onActivate,
}: {
  model: TopbarModel | null;
  texts: Record<string, GuiText>;
  fonts: Record<string, GuiFontMeta>;
  onActivate: (node: GuiNode) => void;
}) {
  const win = model?.win;
  const rects = useMemo(
    () => (win ? measureWindow(win.node, win.sprites, SCREEN_W, SCREEN_H) : {}),
    [win],
  );
  const edit = useEditCanvas();
  const clock = model?.clock;
  if (!model || !win || !clock) return null;
  const box = rects[clock.pausedBoxId];
  const sprite = clock.sprite ? win.sprites[clock.sprite] : undefined;

  const editProps = edit(
    "topbar:clock",
    win,
    win.node,
    win.sprites,
    model.textOverrides,
  );
  return (
    <div
      style={{ position: "absolute", left: 0, top: 0, pointerEvents: "none" }}
    >
      <BlendFramesKeyframes />
      {box && sprite?.resolved && (sprite.frames ?? 1) > 1 && (
        <div
          style={{
            ...frameStyle(sprite, box.x, box.y, {
              frame: 2,
              name: clock.sprite,
            }),
            animation: BLEND_FRAMES_ANIMATION,
          }}
        />
      )}
      <GuiCanvas
        node={win.node}
        {...editProps}
        texts={texts}
        fonts={fonts}
        hidden={clock.hidden.size ? clock.hidden : NO_HIDDEN}
        isolateId={null}
        transparent
        baseW={SCREEN_W}
        baseH={SCREEN_H}
        onActivate={onActivate}
        tooltipFor={model.tooltipFor}
      />
    </div>
  );
}
