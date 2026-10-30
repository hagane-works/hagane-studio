"use client";

import dynamic from "@/platform/dynamic";
import { useCallback, useEffect, useRef, useState } from "react";
import { useTranslations } from "use-intl";
import { loader } from "@monaco-editor/react";
import type { ComponentType } from "react";
import type { EditorProps, Monaco, OnMount } from "@monaco-editor/react";
import type { editor as MonacoEditorNs } from "monaco-editor";

loader.config({ paths: { vs: "/monaco-editor/min/vs" } });
import { Button } from "@/components/ui/button";
import { Copy, Check, RotateCcw } from "lucide-react";
import {
  HOI4_LANGUAGE_ID,
  hoi4LanguageConfig,
  hoi4Monarch,
} from "./hoi4Monarch";
import {
  findUnknownScriptTokens,
  useVanillaScriptNames,
} from "@/lib/vanillaScriptNames";
import { editorMarkerRange, type EditorMarker } from "./editorMarkers";

export type { EditorMarker } from "./editorMarkers";

export interface DirtyIndicator {
  dirty: boolean;

  label?: string;

  status?: "idle" | "parsing" | "ok" | "error" | "info" | "warning";
}

function EditorLoadingFallback() {
  const t = useTranslations("scriptEditor");
  return (
    <div className="flex items-center justify-center h-full text-xs text-muted-foreground">
      {t("loadingEditor")}
    </div>
  );
}

const MonacoEditor = dynamic(
  () => import("@monaco-editor/react").then((m) => m.default),
  {
    ssr: false,
    loading: () => <EditorLoadingFallback />,
  },
) as ComponentType<EditorProps>;

interface ScriptEditorProps {
  value: string;
  onChange?: (next: string) => void;

  onBlur?: () => void;
  readOnly?: boolean;
  title?: string;

  height?: string;

  placeholder?: string;

  markers?: EditorMarker[];
  markerOwner?: string;

  indicator?: DirtyIndicator;

  onResetToForm?: () => void;

  onApply?: () => void;
  canApply?: boolean;
  applyLabel?: string;

  enableVanillaTokenHint?: boolean;

  customScriptNames?: ReadonlySet<string>;
}

const _STATUS_DOT_COLORS: Record<
  NonNullable<DirtyIndicator["status"]>,
  string
> = {
  idle: "bg-muted-foreground/40",
  parsing: "bg-blue-400 animate-pulse",
  ok: "bg-emerald-500",
  error: "bg-red-500",
  warning: "bg-amber-500",
  info: "bg-blue-400",
};

export function ScriptEditor({
  value,
  onChange,
  onBlur,
  readOnly = false,
  title = "Script Preview",
  height = "100%",
  placeholder = "# Select an item to preview script",
  markers,
  markerOwner = "hoi4-parse",
  indicator,
  onResetToForm,
  onApply,
  canApply = false,
  applyLabel = "Apply",
  enableVanillaTokenHint = true,
  customScriptNames,
}: ScriptEditorProps) {
  const t = useTranslations("scriptEditor");

  const vanillaScriptNames = useVanillaScriptNames();

  const [copiedValue, setCopiedValue] = useState<string | null>(null);
  const copied = copiedValue !== null && copiedValue === value;

  const editorRef = useRef<MonacoEditorNs.IStandaloneCodeEditor | null>(null);
  const monacoRef = useRef<Monaco | null>(null);

  const onBlurRef = useRef(onBlur);
  useEffect(() => {
    onBlurRef.current = onBlur;
  }, [onBlur]);

  const effectivelyReadOnly = readOnly || !onChange;
  const displayValue = value || placeholder;

  const handleBeforeMount = useCallback((monaco: Monaco) => {
    const alreadyRegistered = monaco.languages
      .getLanguages()
      .some((l: { id: string }) => l.id === HOI4_LANGUAGE_ID);
    if (alreadyRegistered) return;

    monaco.languages.register({ id: HOI4_LANGUAGE_ID });
    monaco.languages.setMonarchTokensProvider(
      HOI4_LANGUAGE_ID,

      hoi4Monarch as any,
    );
    monaco.languages.setLanguageConfiguration(
      HOI4_LANGUAGE_ID,

      hoi4LanguageConfig as any,
    );
  }, []);

  const handleCopy = useCallback(async () => {
    try {
      await navigator.clipboard.writeText(value || "");
      setCopiedValue(value);
    } catch {}
  }, [value]);

  const handleMount: OnMount = useCallback((editor, monaco) => {
    editorRef.current = editor;
    monacoRef.current = monaco;
    editor.onDidBlurEditorText(() => {
      onBlurRef.current?.();
    });
  }, []);

  useEffect(() => {
    const editor = editorRef.current;
    const monaco = monacoRef.current;
    if (!editor || !monaco) return;
    const model = editor.getModel();
    if (!model) return;
    const sevMap = {
      error: monaco.MarkerSeverity.Error,
      warning: monaco.MarkerSeverity.Warning,
      info: monaco.MarkerSeverity.Info,
    } as const;
    const monacoMarkers = (markers ?? []).map((m) => {
      return {
        severity: sevMap[m.severity] ?? monaco.MarkerSeverity.Info,
        ...editorMarkerRange(m, model),
        message: m.msg,
      };
    });
    monaco.editor.setModelMarkers(model, markerOwner, monacoMarkers);
  }, [markers, markerOwner, value]);

  useEffect(() => {
    return () => {
      const editor = editorRef.current;
      const monaco = monacoRef.current;
      if (!editor || !monaco) return;
      const model = editor.getModel();
      if (!model) return;
      monaco.editor.setModelMarkers(model, markerOwner, []);
    };
  }, [markerOwner]);

  const VANILLA_HINT_OWNER = "hoi4-vanilla-hint";
  useEffect(() => {
    const editor = editorRef.current;
    const monaco = monacoRef.current;
    if (!editor || !monaco) return;
    const model = editor.getModel();
    if (!model) return;
    if (!enableVanillaTokenHint || !vanillaScriptNames.loaded) {
      monaco.editor.setModelMarkers(model, VANILLA_HINT_OWNER, []);
      return;
    }
    const issues = findUnknownScriptTokens(
      value || "",
      vanillaScriptNames,
      customScriptNames ?? new Set(),
    );
    const hintMarkers = issues.map((iss) => ({
      severity: monaco.MarkerSeverity.Info,
      startLineNumber: iss.line,
      startColumn: iss.column,
      endLineNumber: iss.line,
      endColumn: iss.column + iss.token.length,
      message: iss.suggestion
        ? t("unknownTokenWithSuggestion", {
            token: iss.token,
            suggestion: iss.suggestion,
          })
        : t("unknownToken", { token: iss.token }),
      source: "vanilla-token-hint",
    }));
    monaco.editor.setModelMarkers(model, VANILLA_HINT_OWNER, hintMarkers);
  }, [value, vanillaScriptNames, enableVanillaTokenHint, customScriptNames, t]);

  useEffect(() => {
    return () => {
      const editor = editorRef.current;
      const monaco = monacoRef.current;
      if (!editor || !monaco) return;
      const model = editor.getModel();
      if (!model) return;
      monaco.editor.setModelMarkers(model, "hoi4-vanilla-hint", []);
    };
  }, []);

  const dotColor = indicator?.status
    ? _STATUS_DOT_COLORS[indicator.status]
    : _STATUS_DOT_COLORS.idle;

  return (
    <div className="flex h-full flex-col">
      <div className="px-4 pt-3 pb-2 flex flex-col gap-1.5">
        <div className="flex items-center gap-2 min-w-0">
          <h3 className="text-sm font-semibold shrink-0">{title}</h3>
          {indicator?.dirty && (
            <div className="flex items-center gap-1.5 min-w-0 flex-1">
              <span
                className={`inline-block w-2 h-2 rounded-full shrink-0 ${dotColor}`}
                title={indicator.label || "Edited"}
              />
              {indicator.label && (
                <span className="text-xs text-muted-foreground truncate">
                  {indicator.label}
                </span>
              )}
            </div>
          )}
          {!indicator?.dirty && (
            <div className="flex-1 flex justify-end">
              <Button
                variant="ghost"
                size="sm"
                className="h-7 px-2 text-xs"
                onClick={handleCopy}
                disabled={!value}
                title={t("copyTooltip")}
              >
                {copied ? (
                  <>
                    <Check className="h-3.5 w-3.5 mr-1" />
                    {t("copiedLabel")}
                  </>
                ) : (
                  <>
                    <Copy className="h-3.5 w-3.5 mr-1" />
                    {t("copyLabel")}
                  </>
                )}
              </Button>
            </div>
          )}
        </div>
        {indicator?.dirty && (
          <div className="flex items-center gap-1 flex-wrap">
            {onApply && (
              <Button
                variant="default"
                size="sm"
                className="h-7 px-2 text-xs"
                onClick={onApply}
                disabled={!canApply}
                title={canApply ? t("applyTooltip") : t("applyErrorTooltip")}
              >
                {applyLabel}
              </Button>
            )}
            {onResetToForm && (
              <Button
                variant="ghost"
                size="sm"
                className="h-7 px-2 text-xs"
                onClick={onResetToForm}
                title={t("resetTooltip")}
              >
                <RotateCcw className="h-3.5 w-3.5 mr-1" />
                {t("resetLabel")}
              </Button>
            )}
            <Button
              variant="ghost"
              size="sm"
              className="h-7 px-2 text-xs ml-auto"
              onClick={handleCopy}
              disabled={!value}
              title={t("copyTooltip")}
            >
              {copied ? (
                <>
                  <Check className="h-3.5 w-3.5 mr-1" />
                  {t("copiedLabel")}
                </>
              ) : (
                <>
                  <Copy className="h-3.5 w-3.5 mr-1" />
                  {t("copyLabel")}
                </>
              )}
            </Button>
          </div>
        )}
      </div>
      <div className="flex-1 min-h-0 px-4 pb-4">
        <div className="h-full rounded border border-border overflow-hidden">
          <MonacoEditor
            height={height}
            defaultLanguage={HOI4_LANGUAGE_ID}
            language={HOI4_LANGUAGE_ID}
            value={displayValue}
            beforeMount={handleBeforeMount}
            onMount={handleMount}
            onChange={(v) => onChange?.(v ?? "")}
            theme="vs-dark"
            options={{
              readOnly: effectivelyReadOnly,
              minimap: { enabled: false },
              lineNumbers: "on",
              fontSize: 12,
              scrollBeyondLastLine: false,
              wordWrap: "on",
              tabSize: 4,
              insertSpaces: false,
              renderWhitespace: "selection",
              folding: true,
              bracketPairColorization: { enabled: true },
              automaticLayout: true,

              quickSuggestions: false,
              suggestOnTriggerCharacters: false,
            }}
          />
        </div>
      </div>
    </div>
  );
}
