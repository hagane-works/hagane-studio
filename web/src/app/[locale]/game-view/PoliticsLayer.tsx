"use client";

import { useContext, useMemo } from "react";
import { useTranslations } from "use-intl";
import {
  guiRenderApi,
  type GuiCountryStats,
  type GuiNode,
  type GuiPopularity,
  type GuiWindow,
} from "@/lib/guiRenderApi";
import { GuiCanvas, assignIds, measureWindow } from "../gui-renderer/GuiCanvas";
import { BitmapText } from "../gui-renderer/BitmapText";
import { spriteUrl } from "../gui-renderer/GameTooltip";
import { useTemplateWindow } from "./CountrySelectLayer";
import { EditContext, useEditCanvas } from "./editContext";
import { UiEditArt } from "./UiEditArt";
import { FieldHit } from "./FieldHit";
import { matchNamePath } from "./gameNavigation";
import { PIE_RADIUS, POLITICS, pieSlicePaths } from "./politicsModel";
import { partyNamePrinted } from "./textSources";

const OVERLAY = "GFX_pol_piechart_overlay_63x63";
const PARTY_ENTRY = "political_party_info_entry";

const PARTY_ROW = 16;
const PARTY_W = 230;
const NO_HIDDEN = new Set<string>();

function PartyRow({
  entry,
  party,
  ruling,
  left,
  top,
}: {
  entry: GuiWindow;
  party: GuiPopularity;
  ruling: boolean;
  left: number;
  top: number;
}) {
  const edit = useEditCanvas();
  const node = useMemo(() => {
    const clone = structuredClone(entry.node) as GuiNode;
    for (const child of clone.children ?? []) {
      if (child.name === "leading_pol_party_bg" && !ruling) child.sprite = null;

      if (child.name === "color_block") child.sprite = null;
    }
    assignIds(clone);
    return clone;
  }, [entry.node, ruling]);
  const blanks = useMemo(() => {
    const out: Record<string, string> = {};
    for (const child of (node.children ?? []) as (GuiNode & {
      _id?: string;
    })[]) {
      if (child._id && child.name === "name") out[child._id] = "";
    }
    return out;
  }, [node]);

  const printed = useMemo(() => partyNamePrinted(node, party), [node, party]);
  const rects = useMemo(
    () => measureWindow(node, entry.sprites, 1920, 1080),
    [node, entry.sprites],
  );
  const nameNode = (entry.node.children ?? []).find(
    (child) => child.name === "name",
  );
  const font = nameNode?.font ? entry.fonts[nameNode.font]?.bitmap : undefined;
  const nameRect = rects["name:name"];
  const block = (entry.node.children ?? []).find(
    (child) => child.name === "color_block",
  );
  const size = block?.sprite ? entry.sprites[block.sprite] : undefined;
  const [r, g, b] = party.color;
  return (
    <div style={{ position: "absolute", left, top }}>
      <GuiCanvas
        node={node}
        {...edit(
          `${entry.name}:${party.party}`,
          entry,
          node,
          entry.sprites,
          blanks,
          printed,
        )}
        texts={entry.texts}
        fonts={entry.fonts}
        hidden={NO_HIDDEN}
        isolateId={null}
        transparent
        baseW={330}
        baseH={18}
      />
      <div
        style={{
          position: "absolute",
          left: Number(block?.position?.x) || 64,
          top: Number(block?.position?.y) || 6,
          width: size?.width || 13,
          height: size?.height || 13,
          background: `rgb(${r}, ${g}, ${b})`,
        }}
      />
      {font && nameRect && (
        <BitmapText
          font={font}
          text={`${party.label || party.party} (${party.value})`}
          left={nameRect.x}
          top={nameRect.y}
          width={nameNode?.maxwidth || 175}
          height={nameNode?.maxheight || 18}
          align="left"
          vcenter={false}
          maxLines={1}
        />
      )}
    </div>
  );
}

function rectOf(
  win: GuiWindow,
  rects: ReturnType<typeof measureWindow>,
  namePath: string,
) {
  const node = matchNamePath(win.node, namePath)[0] as
    (GuiNode & { _id?: string }) | undefined;
  return node?._id ? rects[node._id] : undefined;
}

export function pieColors(
  pie: readonly GuiPopularity[],
): Record<string, [number, number, number]> {
  const out: Record<string, [number, number, number]> = {};
  for (const slice of pie)
    out[slice.party] = [slice.color[0], slice.color[1], slice.color[2]];
  return out;
}

export function PoliticsLayer({
  win,
  stats,
  projectId,
  onEditParties,
}: {
  win: GuiWindow;
  stats?: GuiCountryStats;
  projectId: number;

  onEditParties?: (
    party: string | null,
    colors: Record<string, [number, number, number]>,
  ) => void;
}) {
  const t = useTranslations("gameViewEdit.politics");
  const active = win.name === POLITICS;
  const rects = useMemo(
    () => (active ? measureWindow(win.node, win.sprites, 1920, 1080) : {}),
    [active, win],
  );
  const entry = useTemplateWindow(projectId, active ? PARTY_ENTRY : null);
  const { swaps } = useContext(EditContext);
  const pie = stats?.popularities;
  if (!active || !pie?.length) return null;
  const partiesAt = rectOf(win, rects, "chart_explanation/parties_grid");
  const ruling = stats?.politics?.ruling_party;

  const overlay = rectOf(win, rects, "political_pie_chart/pol_view_bg");
  const chart = rectOf(win, rects, "political_pie_chart/chart");
  const cx = overlay ? overlay.x + overlay.w / 2 : chart ? chart.x : null;
  const cy = overlay
    ? overlay.y + overlay.h / 2
    : chart
      ? chart.y + PIE_RADIUS
      : null;
  if (cx === null || cy === null) return null;
  const slices = pieSlicePaths(pie, cx, cy);
  const sprite = win.sprites[OVERLAY];

  const ideologyLine = rectOf(win, rects, "ruling_party_info/ideology");
  const electionsLine = rectOf(win, rects, "ruling_party_info/elections");
  const rulingLines =
    ideologyLine && electionsLine
      ? {
          x: Math.min(ideologyLine.x, electionsLine.x),
          y: ideologyLine.y,
          w: Math.max(ideologyLine.w, electionsLine.w),
          h: electionsLine.y + electionsLine.h - ideologyLine.y,
        }
      : null;

  return (
    <div
      style={{ position: "absolute", left: 0, top: 0, pointerEvents: "none" }}
    >
      <svg
        width={1}
        height={1}
        style={{ position: "absolute", left: 0, top: 0, overflow: "visible" }}
        aria-hidden
      >
        {slices.map((slice) => (
          <path key={slice.party} d={slice.d} fill={slice.fill} />
        ))}
      </svg>
      {overlay && sprite?.resolved && (
        <div
          style={{
            position: "absolute",
            left: overlay.x,
            top: overlay.y,
            width: overlay.w,
            height: overlay.h,
            backgroundImage: `url("${swaps[OVERLAY] ? guiRenderApi.spriteUrl(OVERLAY, swaps[OVERLAY].version) : spriteUrl(OVERLAY, sprite)}")`,
            backgroundSize: "100% 100%",
            backgroundRepeat: "no-repeat",
          }}
        />
      )}
      {overlay && (
        <UiEditArt
          owner="pie"
          gfx={OVERLAY}
          sprite={sprite}
          title="pol_piechart_overlay"
          x={overlay.x}
          y={overlay.y}
          w={overlay.w}
          h={overlay.h}
        />
      )}
      {entry.data &&
        partiesAt &&
        pie.map((party, at) => (
          <PartyRow
            key={party.party}
            entry={entry.data as GuiWindow}
            party={party}
            ruling={party.party === ruling}
            left={partiesAt.x}
            top={partiesAt.y + at * PARTY_ROW}
          />
        ))}
      {onEditParties && (
        <>
          {partiesAt &&
            pie.map((party, at) => (
              <FieldHit
                key={`edit-${party.party}`}
                x={partiesAt.x}
                y={partiesAt.y + at * PARTY_ROW}
                w={PARTY_W}
                h={PARTY_ROW}
                label={t("party", { name: party.label || party.party })}
                onClick={() => onEditParties(party.party, pieColors(pie))}
              />
            ))}
          {overlay && (
            <FieldHit
              x={overlay.x}
              y={overlay.y}
              w={overlay.w}
              h={overlay.h}
              label={t("startSupport")}
              onClick={() => onEditParties(null, pieColors(pie))}
            />
          )}
          {rulingLines && (
            <FieldHit
              x={rulingLines.x}
              y={rulingLines.y}
              w={rulingLines.w}
              h={rulingLines.h}
              label={t("rulingParty")}
              onClick={() => onEditParties(ruling ?? null, pieColors(pie))}
            />
          )}
        </>
      )}
    </div>
  );
}
