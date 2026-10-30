import { create } from "zustand";
import type { GameEvent, EventNamespace } from "@/types";
import { eventsApi } from "@/lib/api";

let latestNamespaceRequest = 0;
let latestEventRequest = 0;
let latestPreviewRequest = 0;
let activeProjectId: number | null = null;

interface EventStore {
  namespaces: EventNamespace[];
  events: GameEvent[];
  selectedNsId: number | null;
  selectedEventId: number | null;
  scriptPreview: string;
  loading: boolean;
  error: string | null;

  fetchNamespaces: (projectId: number) => Promise<void>;
  selectNamespace: (nsId: number | null) => void;
  createNamespace: (
    projectId: number,
    namespace: string,
    countryTag?: string,
  ) => Promise<EventNamespace>;
  deleteNamespace: (projectId: number, nsId: number) => Promise<void>;

  fetchEvents: (projectId: number, namespaceId?: number) => Promise<void>;
  selectEvent: (id: number | null) => void;
  createEvent: (
    projectId: number,
    data: Partial<GameEvent>,
  ) => Promise<GameEvent>;
  updateEvent: (
    projectId: number,
    id: number,
    data: Partial<GameEvent>,
  ) => Promise<void>;
  deleteEvent: (projectId: number, id: number) => Promise<void>;
  fetchPreview: (projectId: number, eventId: number) => Promise<void>;
}

export const useEventStore = create<EventStore>((set, get) => ({
  namespaces: [],
  events: [],
  selectedNsId: null,
  selectedEventId: null,
  scriptPreview: "",
  loading: false,
  error: null,

  fetchNamespaces: async (projectId) => {
    const requestId = ++latestNamespaceRequest;
    const changedProject = activeProjectId !== projectId;
    activeProjectId = projectId;
    if (changedProject) {
      latestEventRequest += 1;
      latestPreviewRequest += 1;
      set({
        namespaces: [],
        events: [],
        selectedNsId: null,
        selectedEventId: null,
        scriptPreview: "",
      });
    }
    set({ loading: true, error: null });
    try {
      const namespaces = await eventsApi.listNamespaces(projectId);
      if (requestId !== latestNamespaceRequest || activeProjectId !== projectId)
        return;
      set({ namespaces });

      if (!get().selectedNsId && namespaces.length > 0) {
        set({ selectedNsId: namespaces[0].id });
        await get().fetchEvents(projectId, namespaces[0].id);
      }
    } catch (err) {
      if (
        requestId === latestNamespaceRequest &&
        activeProjectId === projectId
      ) {
        set({
          error: (err as Error).message || "Failed to load event namespaces",
        });
      }
    } finally {
      if (requestId === latestNamespaceRequest && activeProjectId === projectId)
        set({ loading: false });
    }
  },

  selectNamespace: (nsId) => {
    latestEventRequest += 1;
    latestPreviewRequest += 1;
    set({
      selectedNsId: nsId,
      selectedEventId: null,
      events: [],
      scriptPreview: "",
    });
  },

  createNamespace: async (projectId, namespace, countryTag) => {
    const ns = await eventsApi.createNamespace(projectId, {
      namespace,
      country_tag: countryTag,
    });
    set((s) => ({ namespaces: [...s.namespaces, ns], selectedNsId: ns.id }));
    return ns;
  },

  deleteNamespace: async (projectId, nsId) => {
    await eventsApi.deleteNamespace(projectId, nsId);
    set((s) => ({
      namespaces: s.namespaces.filter((n) => n.id !== nsId),
      selectedNsId: s.selectedNsId === nsId ? null : s.selectedNsId,
      events: s.selectedNsId === nsId ? [] : s.events,
      selectedEventId: s.selectedNsId === nsId ? null : s.selectedEventId,
    }));
  },

  fetchEvents: async (projectId, namespaceId) => {
    const nsId = namespaceId ?? get().selectedNsId ?? undefined;
    const selectedNamespaceId = nsId ?? null;
    const requestId = ++latestEventRequest;
    set({ loading: true, error: null });
    try {
      const events = await eventsApi.list(projectId, nsId);
      if (
        requestId !== latestEventRequest ||
        activeProjectId !== projectId ||
        get().selectedNsId !== selectedNamespaceId
      )
        return;
      set({ events });

      if (!get().selectedEventId && events.length > 0) {
        set({ selectedEventId: events[0].id });
        get().fetchPreview(projectId, events[0].id);
      }
    } catch (err) {
      if (
        requestId === latestEventRequest &&
        activeProjectId === projectId &&
        get().selectedNsId === selectedNamespaceId
      ) {
        set({ error: (err as Error).message || "Failed to load events" });
      }
    } finally {
      if (
        requestId === latestEventRequest &&
        activeProjectId === projectId &&
        get().selectedNsId === selectedNamespaceId
      )
        set({ loading: false });
    }
  },

  selectEvent: (id) => {
    latestPreviewRequest += 1;
    set({ selectedEventId: id, scriptPreview: "" });
  },

  createEvent: async (projectId, data) => {
    let ev;
    try {
      ev = await eventsApi.create(projectId, data);
    } catch (err) {
      const msg = (err as Error).message || "";
      if (msg.includes("already exists") || msg.includes("race-collision")) {
        return null as unknown as GameEvent;
      }
      throw err;
    }
    set((s) => ({
      events: [...s.events, ev],
      selectedEventId: ev.id,
    }));

    set((s) => ({
      namespaces: s.namespaces.map((ns) =>
        ns.id === ev.namespace_id
          ? { ...ns, event_count: ns.event_count + 1 }
          : ns,
      ),
    }));
    get().fetchPreview(projectId, ev.id);
    return ev;
  },

  updateEvent: async (projectId, id, data) => {
    const updated = await eventsApi.update(projectId, id, data);
    set((s) => ({
      events: s.events.map((e) => (e.id === id ? updated : e)),
    }));
    get().fetchPreview(projectId, id);
  },

  deleteEvent: async (projectId, id) => {
    const ev = get().events.find((e) => e.id === id);
    await eventsApi.delete(projectId, id);
    set((s) => ({
      events: s.events.filter((e) => e.id !== id),
      selectedEventId: s.selectedEventId === id ? null : s.selectedEventId,
    }));
    if (ev) {
      set((s) => ({
        namespaces: s.namespaces.map((ns) =>
          ns.id === ev.namespace_id
            ? { ...ns, event_count: Math.max(0, ns.event_count - 1) }
            : ns,
        ),
      }));
    }
  },

  fetchPreview: async (projectId, eventId) => {
    const requestId = ++latestPreviewRequest;
    try {
      const res = await eventsApi.preview(projectId, eventId);
      if (
        requestId === latestPreviewRequest &&
        activeProjectId === projectId &&
        get().selectedEventId === eventId
      ) {
        set({ scriptPreview: res.script });
      }
    } catch {
      if (
        requestId === latestPreviewRequest &&
        activeProjectId === projectId &&
        get().selectedEventId === eventId
      ) {
        set({ scriptPreview: "# Preview unavailable" });
      }
    }
  },
}));
