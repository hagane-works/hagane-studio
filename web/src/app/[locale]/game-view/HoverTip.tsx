"use client";

import { useId, type ReactNode } from "react";

export function HoverTip({
  tip,
  side = "top",
  align = "right",
  children,
}: {
  tip: ReactNode | null;
  side?: "top" | "bottom" | "left";

  align?: "left" | "right";

  children: (describedBy: string | undefined) => ReactNode;
}) {
  const id = useId();
  if (tip === null) return <>{children(undefined)}</>;
  return (
    <span className="group/tip relative inline-flex">
      {children(id)}
      <span
        id={id}
        role="tooltip"
        data-hover-tip=""
        className={`pointer-events-none absolute z-10 w-max max-w-xs rounded-md border border-ingame-edge/70 bg-ingame-panel px-2 py-1 text-xs text-ingame-text opacity-0 shadow-lg transition-opacity duration-150 group-focus-within/tip:opacity-100 group-hover/tip:opacity-100 group-hover/tip:delay-300 ${
          side === "left"
            ? "right-full top-1/2 mr-2 -translate-y-1/2"
            : `${side === "top" ? "bottom-full mb-2" : "top-full mt-2"} ${align === "right" ? "right-0" : "left-0"}`
        }`}
      >
        {tip}
      </span>
    </span>
  );
}
