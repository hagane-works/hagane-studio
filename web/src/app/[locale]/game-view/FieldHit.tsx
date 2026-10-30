"use client";

import { useContext } from "react";
import { useTranslations } from "use-intl";
import { gameTooltipProps } from "../gui-renderer/GameTooltip";
import { EditContext } from "./editContext";

export function FieldHit({
  x,
  y,
  w,
  h,
  label,
  onClick,
  onPress,
  appearance,
  tooltip,
}: {
  x: number;
  y: number;
  w: number;
  h: number;
  label: string;
  onClick: () => void;
  onPress?: () => void;

  appearance?: "edit" | "inspect";

  tooltip?: string;
}) {
  const t = useTranslations("gameViewEdit.panel");
  const { contentEditMode } = useContext(EditContext);
  const shownAppearance = appearance ?? (contentEditMode ? "edit" : "inspect");
  return (
    <div
      role="button"
      aria-label={label}
      data-field-hit-appearance={shownAppearance}
      title={onPress && !tooltip ? t("rightClickEdit", { label }) : undefined}
      {...(tooltip ? gameTooltipProps(tooltip, { key: `hit:${label}` }) : {})}
      onClick={(e) => {
        e.stopPropagation();
        (onPress ?? onClick)();
      }}
      onContextMenu={
        onPress
          ? (e) => {
              e.preventDefault();
              e.stopPropagation();
              onClick();
            }
          : undefined
      }
      style={{
        position: "absolute",
        left: x,
        top: y,
        width: w,
        height: h,
        pointerEvents: "auto",
        cursor: shownAppearance === "inspect" || onPress ? "pointer" : "text",
        outline:
          shownAppearance === "edit"
            ? "1px dashed rgba(244, 178, 62, 0.7)"
            : "none",
      }}
    />
  );
}
