"use client";

import { useId, useRef, type ReactNode } from "react";
import { useTranslations } from "use-intl";
import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import {
  InfoTooltip,
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import type { EffectTemplate } from "./effectTemplates";
import type { ConditionTemplate } from "./conditionTemplates";
import { ScriptLintMessages } from "./ScriptLintMessages";
import { useScriptLint } from "./scriptLint";
import enConditions from "../../../../messages/en/conditions.json";
import enEffects from "../../../../messages/en/effects.json";

type Template = EffectTemplate | ConditionTemplate;

function indexEnglishCatalog(
  catalog: Record<string, string>,
): Map<string, string> {
  return new Map(Object.entries(catalog).map(([key, value]) => [value, key]));
}

const CONDITION_KEY_BY_ENGLISH = indexEnglishCatalog(enConditions);
const EFFECT_KEY_BY_ENGLISH = indexEnglishCatalog(enEffects);

interface ScriptBlockEditorProps {
  label: string;

  labelTooltip?: string;
  value: string;
  onChange: (value: string) => void;
  placeholder?: string;
  templates?: Template[];
  minHeight?: string;
  children?: ((insertCode: (code: string) => void) => ReactNode) | ReactNode;
}

export function ScriptBlockEditor({
  label,
  labelTooltip,
  value,
  onChange,
  placeholder,
  templates,
  minHeight = "min-h-16",
  children,
}: ScriptBlockEditorProps) {
  const textareaRef = useRef<HTMLTextAreaElement>(null);

  const fieldId = useId();
  const tConditions = useTranslations("conditions");
  const tEffects = useTranslations("effects");
  const safeValue = typeof value === "string" ? value : "";
  const lint = useScriptLint(safeValue);

  const translateTemplateCopy = (english: string, chinese: string) => {
    const conditionKey = CONDITION_KEY_BY_ENGLISH.get(english);
    if (conditionKey) return tConditions.raw(conditionKey) as string;
    const effectKey = EFFECT_KEY_BY_ENGLISH.get(english);
    if (effectKey) return tEffects.raw(effectKey) as string;

    return english || chinese;
  };

  const insertCode = (code: string) => {
    const newValue = safeValue ? `${safeValue}\n${code}` : code;
    onChange(newValue);
    setTimeout(() => textareaRef.current?.focus(), 50);
  };

  const hasTemplates = templates && templates.length > 0;

  const renderedChildren =
    typeof children === "function" ? children(insertCode) : children;
  const hasButtons = hasTemplates || renderedChildren;

  return (
    <div className="space-y-2">
      <div className="flex items-center gap-1">
        <Label htmlFor={fieldId}>{label}</Label>
        {labelTooltip && <InfoTooltip title={label} content={labelTooltip} />}
      </div>
      {hasButtons && (
        <div className="flex flex-wrap gap-1 mb-1">
          {renderedChildren}
          {templates?.map((tmpl) => {
            const tLabel = translateTemplateCopy(tmpl.label_en, tmpl.label_zh);
            const desc = tmpl.desc_en
              ? translateTemplateCopy(tmpl.desc_en, tmpl.desc_zh ?? "")
              : undefined;
            if (!desc) {
              return (
                <Button
                  key={tmpl.code}
                  size="sm"
                  variant="outline"
                  className="text-xs h-6 px-2"
                  onClick={() => insertCode(tmpl.code)}
                >
                  {tLabel}
                </Button>
              );
            }
            return (
              <Tooltip key={tmpl.code}>
                <TooltipTrigger
                  render={
                    <Button
                      size="sm"
                      variant="outline"
                      className="text-xs h-6 px-2"
                      onClick={() => insertCode(tmpl.code)}
                    >
                      {tLabel}
                    </Button>
                  }
                />
                <TooltipContent>
                  <div className="font-medium mb-1">{tLabel}</div>
                  <div className="text-muted-foreground whitespace-pre-line">
                    {desc}
                  </div>
                </TooltipContent>
              </Tooltip>
            );
          })}
        </div>
      )}
      <Textarea
        id={fieldId}
        ref={textareaRef}
        value={safeValue}
        onChange={(e) => onChange(e.target.value)}
        className={`font-mono text-xs ${minHeight}`}
        placeholder={placeholder}
      />
      <ScriptLintMessages result={lint} />
    </div>
  );
}
