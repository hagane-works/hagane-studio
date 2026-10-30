"use client";

import { useState, useEffect, useMemo, useRef } from "react";
import { useTranslations } from "use-intl";
import useSWR from "swr";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import type { GameEvent, EventNamespace } from "@/types";
import { eventsApi } from "@/lib/api";

type EffectType = "country_event" | "country_event_delayed" | "news_event";

interface EventReferenceDropdownProps {
  projectId: number;
  effectType: EffectType;
  onSelect: (code: string) => void;
  buttonLabel: string;
}

interface DisplayItem {
  event_id: string;
  namespace: string;
  event_type: string;
  title_en: string;
}

export function EventReferenceDropdown({
  projectId,
  effectType,
  onSelect,
  buttonLabel,
}: EventReferenceDropdownProps) {
  const [open, setOpen] = useState(false);
  const [search, setSearch] = useState("");
  const [activeNs, setActiveNs] = useState<number | null>(null);
  const dropdownRef = useRef<HTMLDivElement>(null);
  const t = useTranslations("referenceDropdown");
  const entity = t("entityEvent");

  const { data: namespaces = [] } = useSWR<EventNamespace[]>(
    open && projectId ? ["events/dropdown-ns", projectId] : null,
    () => eventsApi.listNamespaces(projectId),
    { keepPreviousData: true },
  );
  const { data: events = [] } = useSWR<GameEvent[]>(
    open && projectId ? ["events/dropdown-all", projectId] : null,
    () => eventsApi.list(projectId),
    { keepPreviousData: true },
  );

  useEffect(() => {
    if (!open) return;
    const handler = (e: MouseEvent) => {
      if (
        dropdownRef.current &&
        !dropdownRef.current.contains(e.target as Node)
      ) {
        setOpen(false);
      }
    };
    document.addEventListener("mousedown", handler);
    return () => document.removeEventListener("mousedown", handler);
  }, [open]);

  const items = useMemo((): DisplayItem[] => {
    const q = search.toLowerCase().trim();
    let filtered = events;

    if (q) {
      filtered = events.filter(
        (e) =>
          e.event_id.toLowerCase().includes(q) ||
          (e.title_en || "").toLowerCase().includes(q),
      );
    } else if (activeNs !== null) {
      filtered = events.filter((e) => e.namespace_id === activeNs);
    }

    return filtered.map((e) => ({
      event_id: e.event_id,
      namespace:
        namespaces.find((ns) => ns.id === e.namespace_id)?.namespace || "",
      event_type: e.event_type,
      title_en: e.title_en || "",
    }));
  }, [search, events, namespaces, activeNs]);

  const handleSelect = (eventId: string, eventType: string) => {
    let code: string;
    const type =
      effectType === "news_event" ? "news_event" : eventType || "country_event";
    if (effectType === "country_event_delayed") {
      code = `${type} = { id = ${eventId} days = 1 }`;
    } else {
      code = `${type} = { id = ${eventId} }`;
    }
    onSelect(code);
    setOpen(false);
    setSearch("");
  };

  return (
    <div className="relative inline-block" ref={dropdownRef}>
      <Button
        size="sm"
        variant="outline"
        className="text-xs h-6 px-2"
        onClick={() => setOpen(!open)}
      >
        {buttonLabel} ▾
      </Button>

      {open && (
        <div className="absolute z-50 top-7 left-0 w-80 bg-popover border rounded-md shadow-lg">
          <div className="p-2 border-b">
            <Input
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              placeholder={t("searchPlaceholder", { entity })}
              className="text-xs h-7"
              autoFocus
            />
          </div>

          {!search && namespaces.length > 0 && (
            <div className="flex flex-wrap gap-0.5 p-1.5 border-b">
              <button
                className={`px-2 py-0.5 rounded text-xs transition-colors ${
                  activeNs === null
                    ? "bg-primary text-primary-foreground"
                    : "bg-muted hover:bg-muted/80"
                }`}
                onClick={() => setActiveNs(null)}
              >
                {t("all")}
              </button>
              {namespaces.map((ns) => (
                <button
                  key={ns.id}
                  className={`px-2 py-0.5 rounded text-xs transition-colors ${
                    activeNs === ns.id
                      ? "bg-primary text-primary-foreground"
                      : "bg-muted hover:bg-muted/80"
                  }`}
                  onClick={() => setActiveNs(ns.id)}
                >
                  {ns.namespace} ({ns.event_count})
                </button>
              ))}
            </div>
          )}

          <div className="max-h-60 overflow-y-auto">
            {items.length === 0 ? (
              <p className="p-3 text-xs text-muted-foreground">
                {events.length === 0
                  ? t("emptyProject", { entity })
                  : t("noResults")}
              </p>
            ) : (
              items.map((item) => (
                <button
                  key={item.event_id}
                  className="w-full text-left px-3 py-1.5 text-xs hover:bg-muted transition-colors flex items-center justify-between"
                  onClick={() => handleSelect(item.event_id, item.event_type)}
                >
                  <span className="font-mono truncate">{item.event_id}</span>
                  <span className="text-muted-foreground ml-2 shrink-0 text-right">
                    {item.title_en || item.namespace}
                  </span>
                </button>
              ))
            )}
          </div>

          <div className="border-t p-2">
            <button
              className="w-full text-left px-2 py-1 text-xs text-muted-foreground hover:text-foreground"
              onClick={() => {
                const id = prompt(t("promptManual", { entity }));
                if (id) handleSelect(id.trim(), "country_event");
              }}
            >
              {t("enterManually")}
            </button>
          </div>
        </div>
      )}
    </div>
  );
}
