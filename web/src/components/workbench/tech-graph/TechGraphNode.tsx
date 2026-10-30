"use client";

import { memo, useEffect, useState } from "react";
import Image from "@/platform/image";
import { useTranslations } from "use-intl";
import {
  Handle,
  Position,
  useReactFlow,
  type NodeProps,
  type Node,
} from "@xyflow/react";
import type { TechGraphNodeData } from "@/lib/techGraph";
import { SOURCE_COLORS } from "@/lib/techGraph";
import { technologiesApi } from "@/lib/api";
import { AlertTriangle, ImageOff, Lock } from "lucide-react";
import { cn } from "@/lib/utils";
import { useConfirm } from "@/components/shared/ConfirmDialogProvider";

type TechGraphNodeType = Node<TechGraphNodeData, "techNode">;

interface UnlockPath {
  source_kind: "focus" | "event" | "decision";
  source_id: number;
  source_name: string;
  field_name: string;
}

const UNLOCK_KIND_COLOR: Record<UnlockPath["source_kind"], string> = {
  focus:
    "text-emerald-700 dark:text-emerald-300 border-emerald-600/40 bg-emerald-500/10",
  event: "text-sky-700 dark:text-sky-300 border-sky-600/40 bg-sky-500/10",
  decision:
    "text-violet-700 dark:text-violet-300 border-violet-600/40 bg-violet-500/10",
};

function TechHoverPopup({ data }: { data: TechGraphNodeData }) {
  const t = useTranslations("techGraphNode");
  const sourceColor = SOURCE_COLORS[data.source];
  const cats = data.categories ?? [];
  const prereqs = data.prerequisites ?? [];

  const [unlockPaths, setUnlockPaths] = useState<UnlockPath[] | null>(null);
  const [unlockLoading, setUnlockLoading] = useState(false);

  useEffect(() => {
    if (!data.isReachable) return;
    if (!data.reachabilityProjectId || !data.reachabilityCountryTag) return;
    if (unlockPaths !== null || unlockLoading) return;
    let cancelled = false;

    Promise.resolve().then(() => {
      if (!cancelled) setUnlockLoading(true);
    });
    technologiesApi
      .reachabilityPaths(
        data.reachabilityProjectId,
        data.reachabilityCountryTag,
        data.techId,
      )
      .then((res) => {
        if (cancelled) return;
        setUnlockPaths(res.paths);
      })
      .catch(() => {
        if (cancelled) return;
        setUnlockPaths([]);
      })
      .finally(() => {
        if (!cancelled) setUnlockLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [
    data.isReachable,
    data.reachabilityProjectId,
    data.reachabilityCountryTag,
    data.techId,
    unlockPaths,
    unlockLoading,
  ]);
  return (
    <div
      className="
        absolute z-[60] left-full top-0 ml-2
        w-[280px] pointer-events-none
        rounded-md border border-accent/50 bg-popover/95 text-popover-foreground
        shadow-md
        px-3 py-2 text-xs leading-tight
        animate-in fade-in zoom-in-95 duration-150
      "
      style={{
        backdropFilter: "blur(4px)",
      }}
    >
      <div className="text-xs font-semibold text-accent truncate">
        {data.label || data.techId}
      </div>
      <div className="font-mono text-micro text-muted-foreground truncate mb-2">
        {data.techId}
      </div>
      <div className="grid grid-cols-2 gap-x-2 gap-y-1 text-micro">
        <div className="text-muted-foreground">{t("startYear")}</div>
        <div className="font-mono text-foreground">{data.startYear}</div>
        <div className="text-muted-foreground">{t("cost")}</div>
        <div className="font-mono text-foreground">
          {t("costValue", {
            cost: data.cost ?? 0,
            days: Math.round((data.cost ?? 0) * 7),
          })}
        </div>
        <div className="text-muted-foreground">{t("source")}</div>
        <div
          className="font-mono text-micro"
          style={{ color: sourceColor.border }}
        >
          {sourceColor.label}
        </div>
        {data.levelBadge && (
          <>
            <div className="text-muted-foreground">{t("level")}</div>
            <div className="font-mono text-foreground">{data.levelBadge}</div>
          </>
        )}
      </div>
      {cats.length > 0 && (
        <div className="mt-2 pt-2 border-t border-border">
          <div className="text-micro uppercase tracking-wide text-muted-foreground mb-1">
            categories ({cats.length})
          </div>
          <div className="flex flex-wrap gap-1">
            {cats.slice(0, 6).map((c) => (
              <span
                key={c}
                className="font-mono text-micro px-1.5 py-0.5 rounded bg-muted text-foreground border border-border"
              >
                {c}
              </span>
            ))}
            {cats.length > 6 && (
              <span className="font-mono text-micro text-muted-foreground">
                +{cats.length - 6}
              </span>
            )}
          </div>
        </div>
      )}
      {prereqs.length > 0 && (
        <div className="mt-2 pt-2 border-t border-border">
          <div className="text-micro uppercase tracking-wide text-muted-foreground mb-1">
            prereq ({prereqs.length})
          </div>
          <div className="flex flex-wrap gap-1">
            {prereqs.slice(0, 5).map((p) => (
              <span
                key={p}
                className="font-mono text-micro px-1.5 py-0.5 rounded bg-accent/10 text-accent border border-accent/30"
              >
                {p}
              </span>
            ))}
            {prereqs.length > 5 && (
              <span className="font-mono text-micro text-muted-foreground">
                +{prereqs.length - 5}
              </span>
            )}
          </div>
        </div>
      )}
      {data.isReachable && (
        <div className="mt-2 pt-2 border-t border-border">
          <div className="text-micro uppercase tracking-wide text-accent mb-1 flex items-center gap-1">
            <Lock className="w-2.5 h-2.5" />
            {t("unlockPath")}
            {unlockPaths ? ` (${unlockPaths.length})` : ""}
          </div>
          {unlockLoading && unlockPaths === null && (
            <div className="text-micro text-muted-foreground italic">
              {t("loading")}
            </div>
          )}
          {unlockPaths && unlockPaths.length === 0 && (
            <div className="text-micro text-muted-foreground italic">
              {t("noPath")}
            </div>
          )}
          {unlockPaths && unlockPaths.length > 0 && (
            <div className="flex flex-col gap-1">
              {unlockPaths.slice(0, 5).map((p) => (
                <div
                  key={`${p.source_kind}-${p.source_id}-${p.field_name}`}
                  className={`font-mono text-micro px-1.5 py-0.5 rounded border ${UNLOCK_KIND_COLOR[p.source_kind]}`}
                  title={`${p.source_kind} ${p.source_name} · ${p.field_name}`}
                >
                  <span className="text-muted-foreground mr-1">
                    {t(`unlockKind.${p.source_kind}`)}
                  </span>
                  <span className="font-semibold">{p.source_name}</span>
                  <span className="ml-1 text-muted-foreground">
                    .{p.field_name}
                  </span>
                </div>
              ))}
              {unlockPaths.length > 5 && (
                <span className="font-mono text-micro text-muted-foreground">
                  {t("moreCount", { n: unlockPaths.length - 5 })}
                </span>
              )}
            </div>
          )}
        </div>
      )}
    </div>
  );
}

function IconFallbackBox({
  variant,
  techId,
  projectIconExpected,
}: {
  variant: "banner" | "compact";
  techId: string;
  projectIconExpected: boolean;
}) {
  const t = useTranslations("techGraphNode");
  const title = projectIconExpected
    ? t("iconLoadFailedTitle")
    : t("noIconTitle");
  return (
    <div
      className={cn(
        "relative flex items-center justify-center rounded border border-dashed border-border bg-background/50 text-muted-foreground",
        variant === "banner" ? "w-[48px] h-[48px]" : "w-[52px] h-[52px]",
      )}
      title={title}
    >
      {variant === "banner" ? (
        <ImageOff className="w-4 h-4 text-muted-foreground/70" />
      ) : (
        <span className="text-micro font-mono leading-tight text-center px-0.5 break-all">
          {techId.substring(0, 16) || "?"}
        </span>
      )}
      {projectIconExpected && (
        <span
          data-testid="tech-icon-warning"
          className="absolute bottom-0 left-0 flex items-center justify-center p-0.5 rounded-tr
                     bg-background/85 border-t border-r border-border/60 text-warning pointer-events-none"
        >
          <AlertTriangle className="w-2.5 h-2.5" strokeWidth={2.5} />
        </span>
      )}
    </div>
  );
}

function TechGraphNodeComponent({
  data,
  id,
  selected,
}: NodeProps<TechGraphNodeType>) {
  const t = useTranslations("techGraphNode");
  const confirmDialog = useConfirm();

  const [imgStage, setImgStage] = useState(0);

  const [imgAspect, setImgAspect] = useState<number | null>(null);
  const [hover, setHover] = useState(false);

  useEffect(() => {
    let cancelled = false;
    Promise.resolve().then(() => {
      if (!cancelled) {
        setImgStage(0);
        setImgAspect(null);
      }
    });
    return () => {
      cancelled = true;
    };
  }, [data.techId]);

  const imgCandidates: string[] = [];
  if (data.iconUrl) imgCandidates.push(data.iconUrl);
  if (data.baseModIconUrls?.length) imgCandidates.push(...data.baseModIconUrls);
  if (data.vanillaIconUrl) imgCandidates.push(data.vanillaIconUrl);
  const currentImgSrc =
    imgStage < imgCandidates.length ? imgCandidates[imgStage] : null;
  const hasIcon = !!currentImgSrc;

  const handleImgError = () => {
    setImgAspect(null);
    setImgStage((s) => s + 1);
  };
  const handleImgLoad = (e: React.SyntheticEvent<HTMLImageElement>) => {
    const el = e.currentTarget;
    if (el.naturalHeight > 0) setImgAspect(el.naturalWidth / el.naturalHeight);
  };
  const isWideArt = data.isBanner && imgAspect !== null && imgAspect >= 2;
  const sourceColor = SOURCE_COLORS[data.source];
  const reactFlow = useReactFlow();
  const isHorizontal = data.layout === "horizontal";

  const handleDelete = async (e: React.MouseEvent) => {
    e.stopPropagation();
    if (
      await confirmDialog({
        body: `Delete tech "${data.label}" (${data.techId})?`,
        danger: true,
      })
    ) {
      reactFlow.deleteElements({ nodes: [{ id }] });
    }
  };

  const handleClassNames = isHorizontal
    ? "!w-1.5 !h-4 !rounded-sm !bg-muted-foreground/50 !border-0 !min-w-0 !min-h-0"
    : "!w-4 !h-1.5 !rounded-sm !bg-muted-foreground/50 !border-0 !min-w-0 !min-h-0";

  const tooltip = `${data.techId}\nstart_year: ${data.startYear} · cost: ${data.cost}\n[${sourceColor.label}]`;

  if (data.isBanner) {
    return (
      <div
        className={`
          relative rounded-md border-2 shadow-md
          w-[200px] h-[60px] flex items-stretch overflow-hidden
          bg-gradient-to-r from-card to-card/80 text-card-foreground
          tech-spring
          animate-in fade-in zoom-in-95
          hover:scale-[1.03] hover:shadow-xl hover:z-20
          ${selected ? "ring-2 ring-primary shadow-primary/40 animate-pulse-ring" : ""}
        `}
        style={{
          borderColor: selected ? undefined : sourceColor.border,
          backgroundImage: !selected
            ? `linear-gradient(to right, ${sourceColor.bg}, transparent)`
            : undefined,
        }}
        title={tooltip}
        onMouseEnter={() => setHover(true)}
        onMouseLeave={() => setHover(false)}
      >
        {hover && <TechHoverPopup data={data} />}
        {hover && data.layoutMode && (
          <button
            onClick={handleDelete}
            className="absolute top-0.5 right-0.5 z-10 w-5 h-5 rounded-full bg-destructive/85 hover:bg-destructive text-destructive-foreground text-micro leading-none font-semibold shadow-md flex items-center justify-center"
            title={t("deleteTitle")}
          >
            ×
          </button>
        )}

        <Handle
          type="target"
          position={isHorizontal ? Position.Left : Position.Top}
          className={handleClassNames}
        />

        <div
          className={cn(
            "flex items-center justify-center shrink-0 border-r border-border/40 bg-muted/20",
            isWideArt ? "w-[96px]" : "w-[56px]",
          )}
        >
          {hasIcon ? (
            <Image
              src={currentImgSrc!}
              alt={data.techId}
              width={isWideArt ? 88 : 48}
              height={48}
              unoptimized
              className={cn(
                "object-contain rounded",
                isWideArt ? "w-[88px] h-[48px]" : "w-[48px] h-[48px]",
              )}
              loading="lazy"

              key={currentImgSrc}
              onLoad={handleImgLoad}
              onError={handleImgError}
            />
          ) : (
            <IconFallbackBox
              variant="banner"
              techId={data.techId}
              projectIconExpected={!!data.iconUrl}
            />
          )}
        </div>

        <div className="flex-1 min-w-0 px-2 py-1 flex flex-col justify-center">
          <p
            className="text-xs font-semibold leading-tight truncate"
            title={data.label}
          >
            {data.label}
          </p>
          <p className="text-micro text-muted-foreground font-mono leading-tight truncate">
            {data.startYear} · cost {data.cost}
            {data.levelBadge && (
              <span className="ml-1.5 px-1 rounded bg-background/70 border border-border/60 text-micro font-semibold text-foreground">
                {data.levelBadge}
              </span>
            )}
          </p>
        </div>

        {data.isReachable && (
          <div
            className="absolute top-0.5 left-0.5 z-10 w-4 h-4 rounded-full
                       bg-accent text-accent-foreground shadow-md
                       flex items-center justify-center pointer-events-none
                       ring-1 ring-accent/40"
            title={t("unlockPathTitle")}
          >
            <Lock className="w-2.5 h-2.5" strokeWidth={2.5} />
          </div>
        )}

        <Handle
          type="source"
          position={isHorizontal ? Position.Right : Position.Bottom}
          className={handleClassNames}
        />
      </div>
    );
  }

  return (
    <div
      className={`
        relative rounded border-2 shadow-md
        w-[64px] h-[64px] overflow-hidden
        bg-card text-card-foreground
        tech-spring
        animate-in fade-in zoom-in-95
        hover:scale-[1.08] hover:shadow-xl hover:z-20
        ${selected ? "ring-2 ring-primary shadow-primary/40 animate-pulse-ring" : ""}
      `}
      style={{
        borderColor: selected ? undefined : sourceColor.border,
        backgroundColor: sourceColor.bg,
      }}
      title={tooltip + `\n${data.label}`}
      onMouseEnter={() => setHover(true)}
      onMouseLeave={() => setHover(false)}
    >
      {hover && <TechHoverPopup data={data} />}
      {hover && data.layoutMode && (
        <button
          onClick={handleDelete}
          className="absolute top-0 right-0 z-10 w-4 h-4 rounded-full bg-destructive/85 hover:bg-destructive text-destructive-foreground text-micro leading-none font-semibold shadow-md flex items-center justify-center"
          title={t("deleteTitle")}
        >
          ×
        </button>
      )}

      <Handle
        type="target"
        position={isHorizontal ? Position.Left : Position.Top}
        className={handleClassNames}
      />

      <div className="w-full h-full flex items-center justify-center relative">
        {hasIcon ? (
          <Image
            src={currentImgSrc!}
            alt={data.techId}
            width={52}
            height={52}
            unoptimized
            className="w-[52px] h-[52px] object-contain"
            loading="lazy"
            onError={handleImgError}
          />
        ) : (
          <IconFallbackBox
            variant="compact"
            techId={data.techId}
            projectIconExpected={!!data.iconUrl}
          />
        )}

        {data.levelBadge && (
          <span
            className="absolute bottom-0 right-0 text-micro font-semibold px-1 rounded-tl
                       bg-background/85 text-foreground border-l border-t border-border/60 leading-tight"
            title={`Level ${data.levelBadge}`}
          >
            {data.levelBadge}
          </span>
        )}

        {data.isReachable && (
          <div
            className="absolute top-0 left-0 z-10 w-3.5 h-3.5 rounded-br
                       bg-accent text-accent-foreground shadow-md
                       flex items-center justify-center pointer-events-none
                       ring-1 ring-accent/40"
            title={t("unlockPathTitle")}
          >
            <Lock className="w-2 h-2" strokeWidth={2.5} />
          </div>
        )}
      </div>

      <Handle
        type="source"
        position={isHorizontal ? Position.Right : Position.Bottom}
        className={handleClassNames}
      />
    </div>
  );
}

export const TechGraphNode = memo(TechGraphNodeComponent);
