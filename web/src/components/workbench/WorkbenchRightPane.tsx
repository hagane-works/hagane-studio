"use client";

import { useState, type KeyboardEvent, type ReactNode } from "react";

export interface RightPaneTab {
  id: string;

  label: string;

  icon?: ReactNode;

  hint?: string;
  content: ReactNode;
}

interface WorkbenchRightPaneProps {
  tabs: RightPaneTab[];
  defaultTabId?: string;

  activeTabId?: string;
  onTabChange?: (id: string) => void;
}

export function WorkbenchRightPane({
  tabs,
  defaultTabId,
  activeTabId,
  onTabChange,
}: WorkbenchRightPaneProps) {
  const [internalId, setInternalId] = useState<string>(
    defaultTabId ?? tabs[0]?.id ?? "",
  );

  const activeId = activeTabId ?? internalId;
  const active = tabs.find((t) => t.id === activeId) ?? tabs[0];

  const selectTab = (id: string) => {
    setInternalId(id);
    onTabChange?.(id);
  };

  const tabButtonId = (id: string) => `wrp-tab-${id}`;
  const tabPanelId = (id: string) => `wrp-panel-${id}`;

  const onTabKeyDown = (e: KeyboardEvent<HTMLButtonElement>) => {
    if (e.key !== "ArrowLeft" && e.key !== "ArrowRight") return;
    const currentIndex = tabs.findIndex((t) => t.id === active?.id);
    if (currentIndex < 0) return;
    e.preventDefault();
    const dir = e.key === "ArrowRight" ? 1 : -1;
    const nextIndex = (currentIndex + dir + tabs.length) % tabs.length;
    const next = tabs[nextIndex];
    if (!next) return;
    selectTab(next.id);

    const el = document.getElementById(tabButtonId(next.id));
    if (el instanceof HTMLElement) el.focus();
  };

  return (
    <div className="flex h-full min-h-0 flex-col">
      <div
        role="tablist"
        className="shrink-0 flex items-center overflow-x-auto border-b border-border text-micro uppercase tracking-widest font-semibold"
      >
        {tabs.map((tabDef) => {
          const isActive = active?.id === tabDef.id;
          return (
            <button
              key={tabDef.id}
              type="button"
              role="tab"
              id={tabButtonId(tabDef.id)}
              aria-selected={isActive}
              aria-controls={tabPanelId(tabDef.id)}
              tabIndex={isActive ? 0 : -1}
              onClick={() => selectTab(tabDef.id)}
              onKeyDown={onTabKeyDown}

              className={`flex shrink-0 items-center gap-1.5 whitespace-nowrap px-3 py-2 ui-interactive ${
                isActive
                  ? "text-accent shadow-[inset_0_-2px_0_0] shadow-accent"
                  : "text-muted-foreground hover:text-foreground"
              }`}
            >
              {tabDef.icon}
              {tabDef.label}
            </button>
          );
        })}
      </div>
      <div
        role="tabpanel"
        id={active ? tabPanelId(active.id) : undefined}
        aria-labelledby={active ? tabButtonId(active.id) : undefined}
        className="flex min-h-0 flex-1 flex-col"
      >
        {active?.hint && (
          <p className="shrink-0 border-b border-border/40 bg-muted/20 px-3 py-1.5 text-micro text-muted-foreground">
            {active.hint}
          </p>
        )}
        <div className="min-h-0 flex-1">{active?.content}</div>
      </div>
    </div>
  );
}
