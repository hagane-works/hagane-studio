"use client";

import { useEffect } from "react";
import { Trash2, type LucideIcon } from "lucide-react";

export interface FocusContextMenuItem {
  label: string;
  icon?: LucideIcon;

  destructive?: boolean;
  onSelect: () => void;
}

interface FocusContextMenuProps {
  x: number;
  y: number;
  items: FocusContextMenuItem[];
  onClose: () => void;
}

export function FocusContextMenu({
  x,
  y,
  items,
  onClose,
}: FocusContextMenuProps) {
  useEffect(() => {
    window.addEventListener("click", onClose);
    return () => window.removeEventListener("click", onClose);
  }, [onClose]);

  return (
    <div
      className="fixed z-50 rounded-md py-1 min-w-[140px] bg-popover border border-border shadow-lg"
      style={{ left: x, top: y }}
      onClick={(e) => e.stopPropagation()}
    >
      {items.map((item, i) => {
        const Icon = item.icon ?? Trash2;
        return (
          <button
            key={i}
            className={`flex w-full items-center gap-2 px-3 py-1.5 text-xs hover:bg-muted ui-interactive ${
              item.destructive ? "text-destructive" : ""
            }`}
            onClick={() => {
              item.onSelect();
              onClose();
            }}
          >
            <Icon className="h-3.5 w-3.5" />
            {item.label}
          </button>
        );
      })}
    </div>
  );
}
