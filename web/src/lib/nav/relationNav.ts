"use client";

import type { AppRouter as AppRouterInstance } from "@/platform/navigation";

import { useCharacterStore } from "@/lib/store/characterStore";
import { useIdeaStore } from "@/lib/store/ideaStore";
import { useEventStore } from "@/lib/store/eventStore";
import { useDecisionStore } from "@/lib/store/decisionStore";
import { useFocusTreeStore } from "@/lib/store/focusTreeStore";
import { useTechnologyStore } from "@/lib/store/technologyStore";
import { confirmDiscardIfDirty } from "@/lib/hooks/useUnsavedGuard";

interface TargetTypeNav {
  route: string;

  selectByHoi4Id: (hoi4Id: string) => boolean;

  selectByPk: (pk: number) => boolean;

  subscribe: (listener: () => void) => () => void;
}

const SELECT_RETRY_WINDOW_MS = 6000;

function selectWhenReady(nav: TargetTypeNav, trySelect: () => boolean): void {
  if (trySelect()) return;

  let settled = false;

  const unsubscribe = nav.subscribe(() => {
    if (settled) return;

    settled = true;
    const hit = trySelect();
    if (hit) {
      unsubscribe();
      clearTimeout(timer);
    } else {
      settled = false;
    }
  });

  const timer = setTimeout(() => {
    if (settled) return;
    settled = true;
    unsubscribe();
  }, SELECT_RETRY_WINDOW_MS);
}

export const TARGET_TYPE_ROUTE: Record<string, TargetTypeNav> = {
  character: {
    route: "characters",

    selectByHoi4Id: (hoi4Id) => {
      const row = useCharacterStore
        .getState()
        .characters.find((c) => c.tag === hoi4Id);
      if (!row) return false;
      useCharacterStore.getState().selectCharacter(row.id);
      return true;
    },
    selectByPk: (pk) => {
      const row = useCharacterStore
        .getState()
        .characters.find((c) => c.id === pk);
      if (!row) return false;
      useCharacterStore.getState().selectCharacter(pk);
      return true;
    },
    subscribe: (listener) => useCharacterStore.subscribe(listener),
  },
  idea: {
    route: "ideas",
    selectByHoi4Id: (hoi4Id) => {
      const row = useIdeaStore
        .getState()
        .ideas.find((i) => i.idea_id === hoi4Id);
      if (!row) return false;
      useIdeaStore.getState().selectIdea(row.id);
      return true;
    },
    selectByPk: (pk) => {
      const row = useIdeaStore.getState().ideas.find((i) => i.id === pk);
      if (!row) return false;
      useIdeaStore.getState().selectIdea(pk);
      return true;
    },
    subscribe: (listener) => useIdeaStore.subscribe(listener),
  },
  event: {
    route: "events",
    selectByHoi4Id: (hoi4Id) => {
      const row = useEventStore
        .getState()
        .events.find((e) => e.event_id === hoi4Id);
      if (!row) return false;
      useEventStore.getState().selectEvent(row.id);
      return true;
    },
    selectByPk: (pk) => {
      const row = useEventStore.getState().events.find((e) => e.id === pk);
      if (!row) return false;
      useEventStore.getState().selectEvent(pk);
      return true;
    },
    subscribe: (listener) => useEventStore.subscribe(listener),
  },
  decision: {
    route: "decisions",
    selectByHoi4Id: (hoi4Id) => {
      const row = useDecisionStore
        .getState()
        .decisions.find((d) => d.decision_id === hoi4Id);
      if (!row) return false;
      useDecisionStore.getState().selectDecision(row.id);
      return true;
    },
    selectByPk: (pk) => {
      const row = useDecisionStore
        .getState()
        .decisions.find((d) => d.id === pk);
      if (!row) return false;
      useDecisionStore.getState().selectDecision(pk);
      return true;
    },
    subscribe: (listener) => useDecisionStore.subscribe(listener),
  },
  focus: {
    route: "focus-tree",

    selectByHoi4Id: (hoi4Id) => {
      const st = useFocusTreeStore.getState();
      for (const tree of st.trees) {
        const node = tree.nodes.find((n) => n.focus_id === hoi4Id);
        if (node) {
          st.selectTree(tree.id);
          st.selectNode(node.id);
          return true;
        }
      }
      return false;
    },

    selectByPk: (pk) => {
      const st = useFocusTreeStore.getState();
      for (const tree of st.trees) {
        const node = tree.nodes.find((n) => n.id === pk);
        if (node) {
          st.selectTree(tree.id);
          st.selectNode(node.id);
          return true;
        }
      }
      return false;
    },
    subscribe: (listener) => useFocusTreeStore.subscribe(listener),
  },
  technology: {
    route: "technology",
    selectByHoi4Id: (hoi4Id) => {
      const row = useTechnologyStore
        .getState()
        .technologies.find((t) => t.tech_id === hoi4Id);
      if (!row) return false;
      const st = useTechnologyStore.getState();

      if (row.tech_line) st.setTechLine(row.tech_line);
      st.selectTech(row.id);
      return true;
    },
    selectByPk: (pk) => {
      const st = useTechnologyStore.getState();
      const row = st.technologies.find((t) => t.id === pk);
      if (!row) return false;
      if (row.tech_line) st.setTechLine(row.tech_line);
      st.selectTech(pk);
      return true;
    },
    subscribe: (listener) => useTechnologyStore.subscribe(listener),
  },
};

export function canJump(kindOrType: string): boolean {
  return kindOrType in TARGET_TYPE_ROUTE;
}

function listRoute(projectId: number, nav: TargetTypeNav): string {
  return `/project/${projectId}/${nav.route}`;
}

export function jumpToSource(
  router: AppRouterInstance,
  projectId: number,
  sourceKind: string,
  sourceId: number,
): boolean {
  const nav = TARGET_TYPE_ROUTE[sourceKind];
  if (!nav) return false;

  if (!confirmDiscardIfDirty()) return false;
  router.push(listRoute(projectId, nav));

  selectWhenReady(nav, () => nav.selectByPk(sourceId));
  return true;
}

export function jumpToTarget(
  router: AppRouterInstance,
  projectId: number,
  targetType: string,
  targetId: string,
): boolean {
  const nav = TARGET_TYPE_ROUTE[targetType];
  if (!nav) return false;

  if (!confirmDiscardIfDirty()) return false;
  router.push(listRoute(projectId, nav));

  selectWhenReady(nav, () => nav.selectByHoi4Id(targetId));
  return true;
}
