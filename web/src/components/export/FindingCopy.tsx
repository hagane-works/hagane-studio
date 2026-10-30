"use client";

import type { ReactNode } from "react";
import { useTranslations } from "use-intl";
import type { CtdIssue } from "@/lib/api/export";
import { gateParityRepairRoute } from "@/lib/gateParityFindingMetadata";
import {
  RAW_ICU_FINDING_KEYS,
  formatRawFindingTemplate,
  type PresentedFinding,
} from "@/lib/exportCheckPresentation";
import {
  TYPE_TO_ROUTE,
  issueInterp,
} from "@/components/workbench/mod-check/validationIssuePresentation";

export function renderBoldMarkup(text: string): ReactNode {
  const parts = text.split(/<b>(.*?)<\/b>/g);
  if (parts.length === 1) return text;
  return parts.map((part, i) =>
    i % 2 === 1 ? (
      <b key={i} className="font-semibold text-foreground">
        {part}
      </b>
    ) : (
      part
    ),
  );
}

export interface FindingCopyResult {
  what: ReactNode;

  impact: ReactNode | null;

  action: string | null;

  jump: string | null;

  verbatim: boolean;
}

const MACHINE_TOKEN = /^[\p{L}\p{N}]+(?:[._][\p{L}\p{N}]+)+$/u;

function usableMessage(issue: CtdIssue): string | null {
  const message = (issue.message ?? "").trim();
  if (!message) return null;
  if (message === (issue.code ?? "").trim()) return null;
  if (MACHINE_TOKEN.test(message)) return null;
  return message;
}

const ACTION_MAX_CHARS = 68;

const SENTENCE_PUNCTUATION = /[.!?。！？；;]/;

function fixHintAsAction(issue: CtdIssue): string | null {
  const hint = (issue.fix_hint ?? "").trim();
  if (!hint) return null;
  if (!/\s/.test(hint)) return null;
  if (SENTENCE_PUNCTUATION.test(hint)) return null;
  if (hint.length > ACTION_MAX_CHARS) return null;
  return hint;
}

export function useFindingCopy(): (
  finding: PresentedFinding,
) => FindingCopyResult {
  const t = useTranslations("exportFindings");
  const tUi = useTranslations("export");

  const synthesizedHeadline = (): string => tUi("ui.attention.noDescription");

  return (finding: PresentedFinding): FindingCopyResult => {
    const { issue, meta } = finding;
    const message = usableMessage(issue);
    const fallback: FindingCopyResult = {
      what: message ?? synthesizedHeadline(),
      impact: null,
      action: fixHintAsAction(issue),
      jump: null,
      verbatim: message !== null,
    };
    if (!meta) return fallback;
    const key = meta.i18nKey;
    if (!t.has(`${key}.what`)) return fallback;

    const values = issueInterp(issue);
    const rich = (raw: string): ReactNode => renderBoldMarkup(raw);
    const resolve = (field: "what" | "impact"): ReactNode | null => {
      const fullKey = `${key}.${field}`;
      if (!t.has(fullKey)) return null;
      try {
        if (RAW_ICU_FINDING_KEYS.has(key)) {
          return rich(
            formatRawFindingTemplate(t.raw(fullKey) as string, values),
          );
        }

        return rich(formatRawFindingTemplate(t.raw(fullKey) as string, values));
      } catch {
        return null;
      }
    };
    const plain = (field: "action" | "jump"): string | null => {
      const fullKey = `${key}.${field}`;
      if (!t.has(fullKey)) return null;
      try {
        return formatRawFindingTemplate(t.raw(fullKey) as string, values);
      } catch {
        return null;
      }
    };

    const what = resolve("what");
    if (what === null) return fallback;
    return {
      what,
      impact: resolve("impact"),
      action: plain("action") ?? fixHintAsAction(issue),
      jump: plain("jump"),
      verbatim: false,
    };
  };
}

export type FindingRepairTarget =
  { kind: "href"; href: string } | { kind: "project-settings" };

export function findingRepairTarget(
  projectId: number,
  issue: CtdIssue,
): FindingRepairTarget | null {
  const ref = issue.entity_ref;
  if (!ref) return null;
  const collectionRoute = gateParityRepairRoute(issue.code, String(ref.type));
  if (collectionRoute) {
    return { kind: "href", href: `/project/${projectId}/${collectionRoute}` };
  }
  if (String(ref.type) === "project") return { kind: "project-settings" };
  const route = TYPE_TO_ROUTE[String(ref.type)] ?? null;
  if (!route) return null;
  const base = `/project/${projectId}/${route}`;
  const id = encodeURIComponent(String(ref.id));
  if (ref.type === "character")
    return { kind: "href", href: `${base}?c=${id}` };
  if (ref.type === "event") {
    const namespaceId = ref.namespace_id;
    return {
      kind: "href",
      href:
        namespaceId == null
          ? `${base}?event=${id}`
          : `${base}?namespace=${encodeURIComponent(String(namespaceId))}&event=${id}`,
    };
  }
  if (ref.type === "focus" || ref.type === "focus_tree") {
    const treeId = ref.tree_id;
    return {
      kind: "href",
      href:
        treeId == null
          ? `${base}?focus=${id}`
          : `${base}?tree=${encodeURIComponent(String(treeId))}&focus=${id}`,
    };
  }
  return { kind: "href", href: base };
}
