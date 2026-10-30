"use client";

import { useMemo } from "react";
import { ArrowDown, ArrowUp, Plus, Trash2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";

export interface HistoryDateBlock {
  date: string;
  script: string;
}

export type HistoryDateValidationError = "format" | "range" | "hour";

export interface HistoryDateBlocksEditorLabels {
  title: string;
  description: string;
  add: string;
  empty: string;
  date: string;
  script: string;
  moveUp: string;
  moveDown: string;
  delete: string;
  validation: Record<HistoryDateValidationError, string>;
}

export interface HistoryDateBlocksEditorProps {
  blocks: HistoryDateBlock[];
  onChange: (blocks: HistoryDateBlock[]) => void;
  labels: HistoryDateBlocksEditorLabels;
  disabled?: boolean;
  className?: string;
}

const HOI4_DATE_RE =
  /^([0-9]{1,4})\.([0-9]{1,2})\.([0-9]{1,2})(?:\.([0-9]{1,2}))?$/;

export function validateHistoryDate(
  value: string,
): HistoryDateValidationError | null {
  if (typeof value !== "string") return "format";
  const raw = value.trim();
  const match = HOI4_DATE_RE.exec(raw);
  if (!match) return "format";
  const year = Number(match[1]);
  const month = Number(match[2]);
  const day = Number(match[3]);
  const hour = match[4] == null ? 12 : Number(match[4]);
  if (year < 1 || month < 1 || month > 12 || day < 1 || day > 31) {
    return "range";
  }
  if (hour < 0 || hour > 23) return "hour";
  return null;
}

export function addHistoryDateBlock(
  blocks: HistoryDateBlock[],
  block: HistoryDateBlock = { date: "1936.1.1", script: "" },
): HistoryDateBlock[] {
  return [...blocks, { ...block }];
}

export function moveHistoryDateBlock(
  blocks: HistoryDateBlock[],
  index: number,
  delta: number,
): HistoryDateBlock[] {
  if (index < 0 || index >= blocks.length) return [...blocks];
  const target = Math.min(Math.max(index + delta, 0), blocks.length - 1);
  if (target === index) return [...blocks];
  const next = [...blocks];
  const [row] = next.splice(index, 1);
  next.splice(target, 0, row);
  return next;
}

export function deleteHistoryDateBlock(
  blocks: HistoryDateBlock[],
  index: number,
): HistoryDateBlock[] {
  if (index < 0 || index >= blocks.length) return [...blocks];
  return blocks.filter((_, rowIndex) => rowIndex !== index);
}

export function HistoryDateBlocksEditor({
  blocks,
  onChange,
  labels,
  disabled = false,
  className,
}: HistoryDateBlocksEditorProps) {
  const errors = useMemo(
    () => blocks.map((block) => validateHistoryDate(block.date)),
    [blocks],
  );

  const updateRow = (index: number, patch: Partial<HistoryDateBlock>) => {
    onChange(
      blocks.map((block, rowIndex) =>
        rowIndex === index ? { ...block, ...patch } : block,
      ),
    );
  };

  return (
    <section className={className} aria-label={labels.title}>
      <div className="flex items-center justify-between gap-3 mb-2">
        <div>
          <h3 className="text-sm font-semibold">{labels.title}</h3>
          <p className="text-xs text-muted-foreground">{labels.description}</p>
        </div>
        <Button
          type="button"
          variant="outline"
          size="sm"
          disabled={disabled}
          onClick={() => onChange(addHistoryDateBlock(blocks))}
        >
          <Plus aria-hidden="true" />
          {labels.add}
        </Button>
      </div>

      <div className="space-y-3">
        {blocks.length === 0 && (
          <p className="text-xs text-muted-foreground border border-dashed rounded-md p-3">
            {labels.empty}
          </p>
        )}
        {blocks.map((block, index) => (
          <div
            key={index}
            className="grid gap-2 border rounded-md p-3 md:grid-cols-[minmax(10rem,15rem)_1fr_auto]"
          >
            <div className="space-y-1">
              <label
                className="text-xs font-medium"
                htmlFor={`history-date-${index}`}
              >
                {labels.date}
              </label>
              <Input
                id={`history-date-${index}`}
                value={block.date}
                disabled={disabled}
                aria-invalid={errors[index] != null}
                aria-describedby={
                  errors[index] ? `history-date-error-${index}` : undefined
                }
                onChange={(event) =>
                  updateRow(index, { date: event.target.value })
                }
                placeholder="1939.1.1.12"
                className="font-mono text-xs"
              />
              {errors[index] && (
                <p
                  id={`history-date-error-${index}`}
                  className="text-xs text-destructive"
                >
                  {labels.validation[errors[index]]}
                </p>
              )}
            </div>
            <div className="space-y-1">
              <label
                className="text-xs font-medium"
                htmlFor={`history-script-${index}`}
              >
                {labels.script}
              </label>
              <Textarea
                id={`history-script-${index}`}
                value={block.script}
                disabled={disabled}
                onChange={(event) =>
                  updateRow(index, { script: event.target.value })
                }
                placeholder={"set_technology = {\n  atomic_research = 1\n}"}
                className="min-h-20 font-mono text-xs"
              />
            </div>
            <div className="flex items-end justify-end gap-1 md:flex-col md:items-center md:justify-end">
              <Button
                type="button"
                variant="ghost"
                size="icon-sm"
                disabled={disabled || index === 0}
                aria-label={labels.moveUp}
                title={labels.moveUp}
                onClick={() =>
                  onChange(moveHistoryDateBlock(blocks, index, -1))
                }
              >
                <ArrowUp aria-hidden="true" />
              </Button>
              <Button
                type="button"
                variant="ghost"
                size="icon-sm"
                disabled={disabled || index === blocks.length - 1}
                aria-label={labels.moveDown}
                title={labels.moveDown}
                onClick={() => onChange(moveHistoryDateBlock(blocks, index, 1))}
              >
                <ArrowDown aria-hidden="true" />
              </Button>
              <Button
                type="button"
                variant="ghost"
                size="icon-sm"
                disabled={disabled}
                aria-label={labels.delete}
                title={labels.delete}
                onClick={() => onChange(deleteHistoryDateBlock(blocks, index))}
              >
                <Trash2 aria-hidden="true" />
              </Button>
            </div>
          </div>
        ))}
      </div>
    </section>
  );
}
