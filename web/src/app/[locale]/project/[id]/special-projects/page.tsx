"use client";

import { useCallback, useMemo, useState } from "react";
import { OpenInGameLink } from "@/components/workbench/OpenInGameLink";
import { gameViewHref } from "@/lib/gameViewLinks";
import { useActiveCountry } from "@/lib/store/projectStore";
import { useParams } from "@/platform/navigation";
import { useTranslations } from "use-intl";
import useSWR from "swr";
import {
  ReactFlow,
  ReactFlowProvider,
  Background,
  BackgroundVariant,
  MarkerType,
  type Node,
  type Edge,
  type OnNodesChange,
  type Connection,
  type NodeMouseHandler,
} from "@xyflow/react";
import "@xyflow/react/dist/style.css";
import { AlertTriangle, FlaskConical, Info, Trash2 } from "lucide-react";

import { Button } from "@/components/ui/button";
import { EditorEmptyState } from "@/components/workbench/EditorEmptyState";
import { WorkbenchLoadError } from "@/components/workbench/WorkbenchLoadError";
import { useConfirm } from "@/components/shared/ConfirmDialogProvider";
import { useListboxNav } from "@/lib/hooks/useListboxNav";
import { specialProjectsApi } from "@/lib/api";
import { useContentLocaleStore } from "@/lib/store/contentLocaleStore";
import { resolveDisplayName } from "@/lib/i18n/localizedName";
import type {
  Specialization,
  SpecialProject,
  SpecialProjectValidation,
  SpecialProjectValidationItem,
} from "@/types";
import {
  SpecialProjectGraphNode,
  type SpecialProjectNodeData,
} from "@/components/workbench/special-projects/SpecialProjectGraphNode";
import { SpecialProjectEditor } from "@/components/workbench/special-projects/SpecialProjectEditor";

const nodeTypes = { specialProjectNode: SpecialProjectGraphNode };

const CELL_W = 160;
const CELL_H = 130;

function SpecialProjectsInner() {
  const t = useTranslations("specialProjects");
  const tCommon = useTranslations("common");
  const contentLocale = useContentLocaleStore((s) => s.locale);
  const params = useParams();
  const projectId = Number(params.id);
  const confirmDialog = useConfirm();
  const specListNav = useListboxNav();

  const [specs, setSpecs] = useState<Specialization[]>([]);
  const [projects, setProjects] = useState<SpecialProject[]>([]);
  const [activeSpec, setActiveSpec] = useState<string>("");
  const [selectedId, setSelectedId] = useState<number | null>(null);
  const activeCountry = useActiveCountry();
  const [validation, setValidation] = useState<SpecialProjectValidation | null>(
    null,
  );

  const refresh = useCallback(async () => {
    const [s, p, v] = await Promise.all([
      specialProjectsApi.listSpecializations(projectId),
      specialProjectsApi.listProjects(projectId),
      specialProjectsApi.validate(projectId).catch(() => null),
    ]);
    setSpecs(s);
    setProjects(p);
    setValidation(v);
  }, [projectId]);

  const { error: listLoadError, mutate: retryList } = useSWR(
    projectId ? ["special-projects", projectId] : null,
    () => refresh(),
  );

  const selected =
    selectedId == null
      ? null
      : (projects.find((p) => p.id === selectedId) ?? null);

  const visibleProjects = useMemo(
    () =>
      activeSpec
        ? projects.filter((p) => p.specialization === activeSpec)
        : projects,
    [projects, activeSpec],
  );

  const rfNodes: Node<SpecialProjectNodeData>[] = useMemo(
    () =>
      visibleProjects.map((p, i) => ({
        id: String(p.id),
        type: "specialProjectNode",
        position: {
          x: p.x || (i % 5) * CELL_W,
          y: p.y || Math.floor(i / 5) * CELL_H,
        },
        data: {
          dbId: p.id,
          token: p.project_token,
          label: resolveDisplayName(
            contentLocale,
            p.display_name_locs,
            p.name_en,
            p.name_zh,
            p.project_token,
          ),
          specialization: p.specialization,
          complexity: p.complexity,
        },
        selected: p.id === selectedId,
      })),
    [visibleProjects, selectedId, contentLocale],
  );

  const tokenToId = useMemo(() => {
    const m = new Map<string, number>();
    projects.forEach((p) => m.set(p.project_token, p.id));
    return m;
  }, [projects]);

  const rfEdges: Edge[] = useMemo(() => {
    const edges: Edge[] = [];
    for (const p of visibleProjects) {
      for (const parentToken of p.special_project_parent || []) {
        const parentId = tokenToId.get(parentToken);
        if (parentId == null) continue;
        edges.push({
          id: `sp-${parentId}-${p.id}`,
          source: String(parentId),
          target: String(p.id),
          markerEnd: { type: MarkerType.ArrowClosed },
        });
      }
    }
    return edges;
  }, [visibleProjects, tokenToId]);

  const onNodesChange: OnNodesChange<Node<SpecialProjectNodeData>> =
    useCallback(() => {}, []);

  const onNodeDragStop = useCallback(
    async (_e: React.MouseEvent, node: Node<SpecialProjectNodeData>) => {
      const dbId = node.data.dbId;
      const x = Math.round(node.position.x);
      const y = Math.round(node.position.y);
      try {
        await specialProjectsApi.updateProject(projectId, dbId, { x, y });

        setProjects((current) =>
          current.map((project) =>
            project.id === dbId ? { ...project, x, y } : project,
          ),
        );
      } catch {
        void refresh();
      }
    },
    [projectId, refresh],
  );

  const onConnect = useCallback(
    async (conn: Connection) => {
      if (!conn.source || !conn.target || conn.source === conn.target) return;
      const parent = projects.find((p) => String(p.id) === conn.source);
      const child = projects.find((p) => String(p.id) === conn.target);
      if (!parent || !child) return;
      const nextParents = Array.from(
        new Set([
          ...(child.special_project_parent || []),
          parent.project_token,
        ]),
      );
      await specialProjectsApi.updateProject(projectId, child.id, {
        special_project_parent: nextParents,
      });
      await refresh();
    },
    [projects, projectId, refresh],
  );

  const onNodeClick: NodeMouseHandler<Node<SpecialProjectNodeData>> =
    useCallback((_e, node) => setSelectedId(node.data.dbId), []);

  const createSpec = useCallback(async () => {
    const id = window.prompt(t("newSpecializationPrompt"));
    if (!id || !id.trim()) return;
    await specialProjectsApi.createSpecialization(projectId, {
      specialization_id: id.trim(),
    });
    await refresh();
    setActiveSpec(id.trim());
  }, [projectId, refresh, t]);

  const deleteSpec = useCallback(
    async (spec: Specialization) => {
      if (
        !(await confirmDialog({
          body: t("deleteSpecializationConfirm"),
          danger: true,
        }))
      )
        return;
      await specialProjectsApi.deleteSpecialization(projectId, spec.id);
      if (activeSpec === spec.specialization_id) setActiveSpec("");
      await refresh();
    },
    [projectId, refresh, activeSpec, t, confirmDialog],
  );

  const createProject = useCallback(async () => {
    const token = window.prompt(t("newProjectPrompt"));
    if (!token || !token.trim()) return;
    const created = await specialProjectsApi.createProject(projectId, {
      project_token: token.trim(),
      specialization: activeSpec || "",
    });
    await refresh();
    setSelectedId(created.id);
  }, [projectId, refresh, activeSpec, t]);

  const saveNode = useCallback(
    async (data: Partial<SpecialProject>) => {
      if (selectedId == null) return;
      await specialProjectsApi.updateProject(projectId, selectedId, data);
      await refresh();
    },
    [projectId, selectedId, refresh],
  );

  const deleteNode = useCallback(async () => {
    if (selectedId == null) return;
    if (
      !(await confirmDialog({ body: t("deleteProjectConfirm"), danger: true }))
    )
      return;
    await specialProjectsApi.deleteProject(projectId, selectedId);
    setSelectedId(null);
    await refresh();
  }, [projectId, selectedId, refresh, t, confirmDialog]);

  const hasActionable = (validation?.actionable.length ?? 0) > 0;
  const hasInfo = (validation?.info.length ?? 0) > 0;

  const localizeValidation = (it: SpecialProjectValidationItem): string => {
    switch (it.kind) {
      case "missing_parent":
        return it.parent
          ? t("validatorMissingParent", {
              token: it.project_token ?? "?",
              parent: it.parent,
            })
          : it.detail;
      case "cycle":
        return t("validatorCycle", { chain: (it.nodes ?? []).join(" → ") });
      case "orphan":
        return t("validatorOrphan", { token: it.project_token ?? "?" });
      default:
        return it.detail;
    }
  };

  if (listLoadError && specs.length === 0 && projects.length === 0) {
    return (
      <WorkbenchLoadError
        onRetry={() => retryList(undefined, { throwOnError: false })}
      />
    );
  }

  return (
    <div className="flex flex-col h-full">
      <div className="px-4 py-3 border-b">
        <div className="flex items-center gap-2">
          <h1 className="text-lg font-semibold">{t("title")}</h1>
          <span className="flex-1" />
          <OpenInGameLink
            href={gameViewHref(projectId, {
              panel: "countrytechnologyview",
              tag: activeCountry,
            })}
          />
        </div>
        <p className="text-sm text-muted-foreground mt-1 max-w-3xl">
          {t("intro")}
        </p>
        <p className="text-micro text-muted-foreground mt-1 font-mono">
          {t("landing")}
        </p>
      </div>

      {validation && (hasActionable || hasInfo) && (
        <div className="px-4 py-2 space-y-1 border-b">
          {hasActionable && (
            <div className="rounded-md bg-amber-500/10 border border-amber-500/40 px-3 py-2">
              <div className="flex items-center gap-1.5 text-amber-600 text-sm font-semibold">
                <AlertTriangle className="w-4 h-4" />
                {t("validatorActionableHeading")}
              </div>
              <ul className="mt-1 text-xs text-amber-700 list-disc pl-5 space-y-0.5">
                {validation.actionable.map((it, i) => (
                  <li key={i}>{localizeValidation(it)}</li>
                ))}
              </ul>
            </div>
          )}
          {hasInfo && (
            <div className="rounded-md bg-muted/40 border border-border px-3 py-2">
              <div className="flex items-center gap-1.5 text-muted-foreground text-sm font-medium">
                <Info className="w-4 h-4" />
                {t("validatorInfoHeading")}
              </div>
              <ul className="mt-1 text-xs text-muted-foreground list-disc pl-5 space-y-0.5">
                {validation.info.map((it, i) => (
                  <li key={i}>{localizeValidation(it)}</li>
                ))}
              </ul>
            </div>
          )}
        </div>
      )}

      <div className="flex flex-1 min-h-0">
        <aside className="w-56 border-r flex flex-col min-h-0">
          <div className="px-3 py-2 flex items-center justify-between border-b">
            <span className="text-sm font-semibold">
              {t("specializationsHeading")}
            </span>
            <button
              className="text-xs text-primary hover:underline"
              onClick={createSpec}
            >
              {t("newSpecialization")}
            </button>
          </div>
          <div
            {...specListNav.containerProps}
            aria-label={t("specializationsHeading")}
            className="flex-1 overflow-auto outline-none"
          >
            <button
              {...specListNav.optionProps(activeSpec === "")}
              className={`w-full text-left px-3 py-2 text-sm hover:bg-muted/50 focus-visible:ring-1 focus-visible:ring-ring ${
                activeSpec === "" ? "bg-muted font-medium" : ""
              }`}
              onClick={() => setActiveSpec("")}
            >
              {t("allTracks")}
            </button>
            {specs.length === 0 && (
              <p className="px-3 py-2 text-xs text-muted-foreground">
                {t("noSpecializations")}
              </p>
            )}
            {specs.map((s) => (
              <div
                key={s.id}
                {...specListNav.optionProps(activeSpec === s.specialization_id)}
                className={`group flex items-center justify-between px-3 py-2 text-sm hover:bg-muted/50 cursor-pointer focus-visible:ring-1 focus-visible:ring-ring ${
                  activeSpec === s.specialization_id
                    ? "bg-muted font-medium"
                    : ""
                }`}
                onClick={() => setActiveSpec(s.specialization_id)}
              >
                <span className="truncate font-mono text-xs">
                  {s.specialization_id.replace("specialization_", "")}
                </span>
                <button
                  className="opacity-0 group-hover:opacity-100 text-muted-foreground hover:text-destructive"
                  onClick={(e) => {
                    e.stopPropagation();
                    void deleteSpec(s);
                  }}
                  aria-label={t("delete")}
                >
                  <Trash2 className="w-3.5 h-3.5" />
                </button>
              </div>
            ))}
          </div>
          <div className="border-t px-3 py-2">
            <Button size="sm" className="w-full" onClick={createProject}>
              {t("newProject")}
            </Button>
          </div>
        </aside>

        <div className="flex-1 min-w-0 relative">
          {visibleProjects.length === 0 ? (
            projects.length === 0 ? (
              <div className="absolute inset-0 flex items-center justify-center p-6">
                <EditorEmptyState
                  variant="hero"
                  className="max-w-md"
                  icon={
                    <FlaskConical className="h-10 w-10 text-muted-foreground/60" />
                  }
                  title={t("emptyHeroTitle")}
                  description={t("emptyHeroDesc")}
                  actions={
                    <Button
                      size="sm"
                      onClick={() => {
                        void createProject();
                      }}
                    >
                      {tCommon("emptyCtaCreate")}
                    </Button>
                  }
                />
              </div>
            ) : (
              <div className="absolute inset-0 flex items-center justify-center text-sm text-muted-foreground">
                {t("noProjects")}
              </div>
            )
          ) : (
            <ReactFlow
              nodes={rfNodes}
              edges={rfEdges}
              nodeTypes={nodeTypes}
              onNodesChange={onNodesChange}
              onNodeDragStop={onNodeDragStop}
              onConnect={onConnect}
              onNodeClick={onNodeClick}
              fitView
              proOptions={{ hideAttribution: true }}
            >
              <Background variant={BackgroundVariant.Lines} gap={CELL_W} />
            </ReactFlow>
          )}
        </div>

        <aside className="w-80 border-l overflow-auto">
          {selected ? (
            <SpecialProjectEditor
              key={selected.id}
              project={selected}
              specs={specs}
              onSave={saveNode}
              onDelete={() => void deleteNode()}
            />
          ) : (
            <p className="p-4 text-sm text-muted-foreground">
              {t("selectNodeHint")}
            </p>
          )}
        </aside>
      </div>
    </div>
  );
}

export default function SpecialProjectsPage() {
  return (
    <ReactFlowProvider>
      <SpecialProjectsInner />
    </ReactFlowProvider>
  );
}
