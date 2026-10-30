"use client";

import { useState } from "react";
import { useTranslations } from "use-intl";
import { ChevronLeft, ChevronRight } from "lucide-react";
import { Button } from "@/components/ui/button";
import { FocusNodeEditor } from "../FocusNodeEditor";
import { ScriptEditor } from "@/shared/editors/ScriptEditor";
import { RelationsPanel } from "@/components/workbench/RelationsPanel";
import type { EditorWiring } from "@/shared/editors/useEditorWiring";
import type { FocusNode } from "@/types";
import type { ImageUploadResult } from "@/shared/upload/ImageUpload";

interface FocusTreeSidebarProps {
  selectedNode: FocusNode | null;
  allNodes: FocusNode[];
  projectId: number;
  countryTag: string;
  onSaveNode: (data: Partial<FocusNode>) => Promise<void>;
  onDraftChange?: (draft: FocusNode) => void;

  editorWiring: EditorWiring;

  selectedTreeId?: number;

  onIconUploaded?: (result: ImageUploadResult) => void;

  onNodeForked?: (forked: FocusNode) => void;
  onPreviewActiveChange?: (active: boolean) => void;
}

export function FocusTreeSidebar({
  selectedNode,
  allNodes,
  projectId,
  countryTag,
  onSaveNode,
  onDraftChange,
  editorWiring,
  selectedTreeId,
  onIconUploaded,
  onNodeForked,
  onPreviewActiveChange,
}: FocusTreeSidebarProps) {
  const [activeTab, setActiveTab] = useState<
    "editor" | "preview" | "relations"
  >("editor");
  const [collapsed, setCollapsed] = useState(false);
  const tRel = useTranslations("relations");
  const t = useTranslations("focusTreeSidebar");

  return (
    <div
      className={`${collapsed ? "w-10" : "w-96"} ui-ide-panel-left flex flex-col shrink-0`}
    >
      {collapsed ? (
        <div className="flex flex-col items-center pt-2">
          <Button
            size="sm"
            variant="ghost"
            className="w-8 p-0"
            onClick={() => setCollapsed(false)}
            title={t("expandSidebar")}
          >
            <ChevronLeft className="h-4 w-4" />
          </Button>
        </div>
      ) : (
        <div className="flex items-center border-b border-border shrink-0">
          <button
            className={`flex-1 px-3 py-2 text-xs font-medium transition-colors ${
              activeTab === "editor"
                ? "text-primary border-b-2 border-primary"
                : "text-muted-foreground hover:text-foreground"
            }`}
            onClick={() => {
              setActiveTab("editor");
              onPreviewActiveChange?.(false);
            }}
          >
            Editor
          </button>
          <button
            className={`flex-1 px-3 py-2 text-xs font-medium transition-colors ${
              activeTab === "preview"
                ? "text-primary border-b-2 border-primary"
                : "text-muted-foreground hover:text-foreground"
            }`}
            onClick={() => {
              setActiveTab("preview");
              onPreviewActiveChange?.(true);
            }}
          >
            Preview
          </button>
          <button
            className={`flex-1 px-3 py-2 text-xs font-medium transition-colors ${
              activeTab === "relations"
                ? "text-primary border-b-2 border-primary"
                : "text-muted-foreground hover:text-foreground"
            }`}
            onClick={() => {
              setActiveTab("relations");
              onPreviewActiveChange?.(false);
            }}
          >
            {tRel("tab")}
          </button>
          <Button
            size="sm"
            variant="ghost"
            className="w-8 p-0 mr-1 shrink-0"
            onClick={() => setCollapsed(true)}
            title={t("collapseSidebar")}
          >
            <ChevronRight className="h-4 w-4" />
          </Button>
        </div>
      )}

      <div className={collapsed ? "hidden" : "flex-1 min-h-0 flex flex-col"}>
        <div
          className={
            activeTab === "editor" ? "flex-1 min-h-0 overflow-y-auto" : "hidden"
          }
        >
          {selectedNode ? (
            <FocusNodeEditor
              node={selectedNode}
              allNodes={allNodes}
              onSave={onSaveNode}
              projectId={projectId}
              countryTag={countryTag}
              onDraftChange={onDraftChange}
              compact
              treeId={selectedTreeId}
              onIconUploaded={onIconUploaded}
              onForked={onNodeForked}
            />
          ) : (
            <div className="p-6 text-center text-muted-foreground">
              <p className="text-xs">Click a node in the graph to edit it.</p>
              <p className="text-xs text-muted-foreground mt-1">
                Double-click empty space to create a new node.
              </p>
            </div>
          )}
        </div>
        {activeTab === "relations" ? (
          <div className="flex-1 min-h-0">
            <RelationsPanel
              kind="focus"
              sourceId={selectedNode?.id ?? null}
              projectId={projectId}
            />
          </div>
        ) : activeTab === "preview" ? (
          <div className="flex-1 min-h-0 flex flex-col">
            {selectedNode && (
              <div className="px-3 py-1.5 bg-muted/50 border-b border-border shrink-0">
                <p className="text-micro text-muted-foreground">
                  Previewing:{" "}
                  <span className="font-mono text-foreground">
                    {selectedNode.focus_id}
                  </span>
                  <span className="ml-1 text-muted-foreground/60">
                    · click empty space for full tree
                  </span>
                </p>
              </div>
            )}
            <div className="flex-1 min-h-0">
              <ScriptEditor
                value={editorWiring.scriptDraft}
                onChange={editorWiring.setScriptDraft}
                onBlur={editorWiring.triggerParse}
                markers={editorWiring.markers}
                indicator={editorWiring.indicator}
                onResetToForm={editorWiring.handleResetToForm}
                onApply={editorWiring.handleApplyEditor}
                canApply={editorWiring.canApply}
                applyLabel="Apply"
              />
            </div>
          </div>
        ) : null}
      </div>
    </div>
  );
}
