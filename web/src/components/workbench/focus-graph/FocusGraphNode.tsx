"use client";

import { memo, useRef, useState } from "react";
import Image from "@/platform/image";
import { useTranslations } from "use-intl";
import { Handle, Position, type NodeProps, type Node } from "@xyflow/react";
import type { FocusGraphNodeData } from "@/lib/focusTreeGraph";
import { useContentLocaleStore } from "@/lib/store/contentLocaleStore";
import { ReadableScriptedName } from "./scriptedName";
import { useFocusInGamePreview } from "./inGamePreviewContext";

type FocusGraphNodeType = Node<FocusGraphNodeData, "focusNode">;

const failedIconUrls = new Set<string>();

function FocusGraphNodeComponent({
  data,
  selected,
}: NodeProps<FocusGraphNodeType>) {
  const t = useTranslations("focusGraphNode");
  const locale = useContentLocaleStore((s) => s.locale);

  const inGame = useFocusInGamePreview();

  const [, setIconFailureVersion] = useState(0);

  const clickTrackRef = useRef<{
    startTime: number;
    startX: number;
    startY: number;
    callback?: () => void;
  } | null>(null);

  const createHandleClickDetector =
    (callback: () => void) => (e: React.MouseEvent) => {
      const startTime = Date.now();
      const startX = e.clientX;
      const startY = e.clientY;

      clickTrackRef.current = { startTime, startX, startY, callback };

      const handleMouseUp = (upEvent: MouseEvent) => {
        if (!clickTrackRef.current) {
          document.removeEventListener("mouseup", handleMouseUp);
          return;
        }

        const elapsed = Date.now() - clickTrackRef.current.startTime;
        const distance = Math.hypot(
          upEvent.clientX - clickTrackRef.current.startX,
          upEvent.clientY - clickTrackRef.current.startY,
        );

        if (elapsed < 200 && distance < 5) {
          clickTrackRef.current.callback?.();
        }

        clickTrackRef.current = null;
        document.removeEventListener("mouseup", handleMouseUp);
      };

      document.addEventListener("mouseup", handleMouseUp);
    };

  const iconCandidates: string[] = [];
  if (data.iconUrl) iconCandidates.push(data.iconUrl);
  if (data.baseModIconUrls?.length)
    iconCandidates.push(...data.baseModIconUrls);
  if (data.vanillaIconUrl) iconCandidates.push(data.vanillaIconUrl);
  const currentIconUrl =
    iconCandidates.find((url) => !failedIconUrls.has(url)) ?? null;
  const hasIcon = !!currentIconUrl;

  return (
    <div
      className="relative"
      style={{
        paddingLeft: "40px",
        paddingRight: "40px",
        paddingBottom: "40px",
        marginLeft: "-40px",
        marginRight: "-40px",
        marginBottom: "-40px",
      }}
    >
      <div
        className={
          inGame
            ? `
          relative rounded-[4px] border shadow-sm
          w-[110px] text-center overflow-visible
          border-ingame-edge/70 bg-gradient-to-b from-ingame-panel/95 to-ingame-panel-deep
          text-ingame-title
          transition-[border-color,box-shadow] duration-150
          ${
            selected
              ? "border-accent ring-2 ring-accent/60"
              : "hover:border-ingame-gold/60"
          }
        `
            : `
          relative rounded-[11px] border-[1.5px] shadow-sm
          w-[110px] text-center overflow-visible
          bg-card text-card-foreground
          transition-[border-color,box-shadow,background-color] duration-150
          ${
            selected
              ? "border-primary ring-2 ring-primary/40 bg-primary/5"
              : "border-border/80 hover:border-muted-foreground/60"
          }
        `
        }
      >
        <Handle
          type="target"
          position={Position.Top}
          className="!w-6 !h-2.5 !rounded-sm !bg-muted-foreground/50 !border-0 !min-w-0 !min-h-0 hover:!bg-primary/70 hover:!shadow-md transition-all"
          title="Click to add prerequisite child node"
        />

        <div className="px-2 pt-2 pb-1">
          <p
            className={`text-xs font-semibold leading-tight truncate ${inGame ? "text-ingame-gold" : ""}`}
            title={data.label}
          >
            <ReadableScriptedName name={data.label} locale={locale} />
          </p>
        </div>

        <div className="flex items-center justify-center px-2 py-1">
          {hasIcon ? (
            <Image
              key={currentIconUrl}
              src={currentIconUrl!}
              alt={data.icon}
              width={60}
              height={60}
              unoptimized
              className="w-[60px] h-[60px] object-contain rounded"
              loading="lazy"
              onError={() => {
                failedIconUrls.add(currentIconUrl!);
                setIconFailureVersion((version) => version + 1);
              }}
            />
          ) : (
            <div
              className={`w-[60px] h-[60px] rounded flex items-center justify-center ${inGame ? "bg-ingame-band/60" : "bg-muted/50"}`}
              title={data.icon}
            >
              <span
                className={`text-micro font-mono leading-tight text-center px-0.5 break-all ${inGame ? "text-ingame-text/80" : "text-muted-foreground"}`}
              >
                {data.icon
                  ?.replace("GFX_", "")
                  .replace("goal_", "")
                  .replace("focus_", "")
                  .substring(0, 20) || "?"}
              </span>
            </div>
          )}
        </div>

        <div className="pb-2">
          <span
            className={`inline-block text-micro px-2 py-0.5 rounded-full font-semibold ${inGame ? "bg-ingame-band/80 text-ingame-gold" : "bg-primary/15 text-primary"}`}
          >
            {data.cost}w
          </span>
        </div>

        <Handle
          type="source"
          position={Position.Bottom}
          onMouseDown={
            data.onAddChild
              ? createHandleClickDetector(data.onAddChild)
              : undefined
          }
          className="!w-6 !h-2.5 !rounded-sm !bg-muted-foreground/50 !border-0 !min-w-0 !min-h-0 hover:!bg-primary/70 hover:!scale-125 hover:!shadow-md transition-all cursor-pointer"
          title="Click to add prerequisite child node"
        />

        <Handle
          type="source"
          position={Position.Left}
          id="mutex-left"
          isConnectable={true}
          onMouseDown={
            data.onAddMutexChild
              ? createHandleClickDetector(() => data.onAddMutexChild?.("left"))
              : undefined
          }
          className="!w-3 !h-6 !rounded-sm !bg-destructive !border-0 !min-w-0 !min-h-0 opacity-60 hover:opacity-100 hover:!scale-125 hover:!shadow-md transition-all cursor-pointer"
          title="Click to add left mutual exclusion node"
        />

        <Handle
          type="target"
          position={Position.Right}
          id="mutex-right"
          isConnectable={true}
          onMouseDown={
            data.onAddMutexChild
              ? createHandleClickDetector(() => data.onAddMutexChild?.("right"))
              : undefined
          }
          className="!w-3 !h-6 !rounded-sm !bg-destructive !border-0 !min-w-0 !min-h-0 opacity-60 hover:opacity-100 hover:!scale-125 hover:!shadow-md transition-all cursor-pointer"
          title="Click to add right mutual exclusion node"
        />

        {data.isRelative && (
          <div
            className="absolute -top-1 -right-1 w-2.5 h-2.5 rounded-full bg-accent border border-background"
            title={t("relativePositionTitle")}
          />
        )}

        {!inGame && (data.hasAllowBranch || data.hasLoadFocusTreeRules) && (
          <div className="absolute top-0.5 left-0.5 flex flex-col gap-0.5 items-start pointer-events-auto">
            {data.hasAllowBranch && (
              <span
                className="text-micro leading-none px-1 py-0.5 rounded bg-accent/15 text-accent font-semibold shadow-sm"
                title={t("allowBranchTitle")}
              >
                {t("allowBranchBadge")}
              </span>
            )}
            {data.hasLoadFocusTreeRules && (
              <span
                className="text-micro leading-none px-1 py-0.5 rounded bg-muted text-muted-foreground font-semibold shadow-sm"
                title={t("loadTreeTitle")}
              >
                {t("loadTreeBadge")}
              </span>
            )}
          </div>
        )}
      </div>
    </div>
  );
}

export const FocusGraphNode = memo(FocusGraphNodeComponent);
