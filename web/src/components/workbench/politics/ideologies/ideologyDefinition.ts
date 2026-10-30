import type {
  PoliticsDefinitionPatch,
  PoliticsIdeology,
  PoliticsIdeologyKind,
  PoliticsReference,
  PoliticsRootCreateInput,
  PoliticsSubtypeCreateInput,
  PoliticsTarget,
} from "@/lib/api/politics";
import {
  groupIdeologyDisplayName,
  type IdeologyLocalisationMap,
  type IdeologyRegistry,
} from "@/lib/ideology/registry";
import {
  defaultRootBody,
  rootBodyColor,
  rootBodyDraftError,
  rootBodyWithColor,
} from "./rootBody";

export type CreateIntent = "subtype" | "root";
export type RootRuleMode = "safe" | "custom";

export const STANDARD_POLITICS_COLORS: Record<string, string> = {
  democratic: "#285fa8",
  communism: "#a62c35",
  fascism: "#8a5d2a",
  neutrality: "#6f7468",
};

export function localizedValue(
  values: Record<string, string> | null | undefined,
  locale: string,
  fallback: string,
): string {
  return (
    values?.[locale]?.trim() ||
    values?.en?.trim() ||
    values?.zh?.trim() ||
    fallback
  );
}

export function suggestedPoliticsScriptId(displayName: string): string {
  const candidate = displayName
    .normalize("NFKD")
    .replace(/[̀-ͯ]/g, "")
    .toLowerCase()
    .replace(/[^a-z0-9_]+/g, "_")
    .replace(/^_+|_+$/g, "")
    .replace(/_+/g, "_");
  return candidate || "new_subtype";
}

export function targetKey(target: PoliticsTarget): string {
  return `${target.kind}:${target.script_id}`;
}

export function referenceKey(reference: PoliticsReference): string {
  return `${reference.entity.kind}:${reference.entity.id}:${reference.entity.path}:${reference.context ?? ""}`;
}

export function normalizedLocaleMap(
  source: IdeologyLocalisationMap,
): Record<string, string> {
  return Object.fromEntries(
    Object.entries(source)
      .map(([locale, value]) => [locale, value?.trim() ?? ""] as const)
      .filter(([, value]) => Boolean(value)),
  );
}

export interface ProjectDefinition {
  name: IdeologyLocalisationMap;
  desc: IdeologyLocalisationMap;
  randomDisabled: boolean;
  body: string;
}

export function projectDefinition(
  registry: IdeologyRegistry,
  ideology: PoliticsIdeology,
): ProjectDefinition {
  if (ideology.kind === "root") {
    const entry = registry.root_definitions?.find(
      (item) => item.id === ideology.script_id,
    );
    return {
      name: entry?.name ?? {},
      desc: entry?.desc ?? {},
      randomDisabled: false,
      body: entry?.body ?? "",
    };
  }
  const entry = registry.sub_ideologies.find(
    (item) => item.id === ideology.script_id,
  );
  return {
    name: entry?.name ?? {},
    desc: entry?.desc ?? {},
    randomDisabled: Boolean(entry?.random_disabled),
    body: "",
  };
}

export function ideologyDisplayName(
  registry: IdeologyRegistry,
  ideology: PoliticsIdeology,
  locale: Parameters<typeof groupIdeologyDisplayName>[1],
): string {
  const definition = projectDefinition(registry, ideology);
  if (ideology.kind === "root") {
    return groupIdeologyDisplayName(
      ideology.script_id,
      locale,
      definition.name,
    );
  }
  return localizedValue(
    definition.name,
    locale,
    ideology.display_name || ideology.script_id,
  );
}

export function rootSwatchColor(
  registry: IdeologyRegistry,
  rootId: string,
  index: number,
): string {
  const body =
    registry.root_definitions?.find((entry) => entry.id === rootId)?.body ?? "";
  return body
    ? rootBodyColor(body)
    : (STANDARD_POLITICS_COLORS[rootId] ?? `var(--chart-${(index % 5) + 1})`);
}

export interface IdeologyDefinitionDraft {
  name: IdeologyLocalisationMap;
  desc: IdeologyLocalisationMap;
  randomDisabled: boolean;
  rootBody: string;
  rootColor: string;
}

export function definitionDraft(
  definition: ProjectDefinition,
): IdeologyDefinitionDraft {
  return {
    name: definition.name,
    desc: definition.desc,
    randomDisabled: definition.randomDisabled,
    rootBody: definition.body,
    rootColor: rootBodyColor(definition.body),
  };
}

export function definitionDraftDirty(
  ideology: PoliticsIdeology | null,
  definition: ProjectDefinition | null,
  draft: Omit<IdeologyDefinitionDraft, "rootColor">,
): boolean {
  return Boolean(
    ideology?.source === "project_managed" &&
    definition &&
    (JSON.stringify(normalizedLocaleMap(draft.name)) !==
      JSON.stringify(normalizedLocaleMap(definition.name)) ||
      JSON.stringify(normalizedLocaleMap(draft.desc)) !==
        JSON.stringify(normalizedLocaleMap(definition.desc)) ||
      draft.randomDisabled !== definition.randomDisabled ||
      (ideology.kind === "root" &&
        draft.rootBody.trim() !== definition.body.trim())),
  );
}

export function definitionPatch(
  kind: PoliticsIdeologyKind,
  draft: Omit<IdeologyDefinitionDraft, "rootColor">,
): PoliticsDefinitionPatch {
  return {
    name: normalizedLocaleMap(draft.name),
    desc: normalizedLocaleMap(draft.desc),
    ...(kind === "root" ? { body: draft.rootBody.trim() } : {}),
    ...(kind === "subtype" ? { random_disabled: draft.randomDisabled } : {}),
  };
}

export interface IdeologyCreateDraft {
  intent: CreateIntent | null;
  name: IdeologyLocalisationMap;
  desc: IdeologyLocalisationMap;
  scriptId: string;

  scriptIdTouched: boolean;

  parent: string;
  randomDisabled: boolean;
  rootBody: string;
  rootColor: string;
  ruleMode: RootRuleMode;
}

export function newCreateDraft(): IdeologyCreateDraft {
  return {
    intent: null,
    name: {},
    desc: {},
    scriptId: "new_subtype",
    scriptIdTouched: false,
    parent: "",
    randomDisabled: false,
    rootBody: defaultRootBody(),
    rootColor: "#808080",
    ruleMode: "safe",
  };
}

export function reopenedCreateDraft(
  draft: IdeologyCreateDraft,
): IdeologyCreateDraft {
  return {
    ...draft,
    intent: null,
    rootColor: "#808080",
    ruleMode: "safe",
    rootBody: defaultRootBody(),
  };
}

export function withCreateIntent(
  draft: IdeologyCreateDraft,
  intent: CreateIntent | null,
): IdeologyCreateDraft {
  return {
    ...draft,
    intent,
    scriptId:
      intent === "root" && !draft.scriptIdTouched
        ? "new_political_system"
        : draft.scriptId,
  };
}

export function withCreateName(
  draft: IdeologyCreateDraft,
  next: IdeologyLocalisationMap,
): IdeologyCreateDraft {
  const changedValue =
    Object.entries(next).find(
      ([locale, value]) =>
        draft.name[locale as keyof IdeologyLocalisationMap] !== value,
    )?.[1] ??
    Object.values(next).find((value) => value.trim()) ??
    "";
  return {
    ...draft,
    name: next,
    scriptId: draft.scriptIdTouched
      ? draft.scriptId
      : suggestedPoliticsScriptId(changedValue),
  };
}

export function withCreateRootColor(
  draft: IdeologyCreateDraft,
  color: string,
): IdeologyCreateDraft {
  return {
    ...draft,
    rootColor: color,
    rootBody:
      draft.ruleMode === "safe"
        ? defaultRootBody(color)
        : rootBodyWithColor(draft.rootBody, color),
  };
}

export function withCreateRuleMode(
  draft: IdeologyCreateDraft,
  mode: RootRuleMode,
): IdeologyCreateDraft {
  return {
    ...draft,
    ruleMode: mode,
    rootBody:
      mode === "safe" ? defaultRootBody(draft.rootColor) : draft.rootBody,
  };
}

export function withCreateRootBody(
  draft: IdeologyCreateDraft,
  body: string,
): IdeologyCreateDraft {
  return { ...draft, rootBody: body, rootColor: rootBodyColor(body) };
}

export function createParent(
  draft: IdeologyCreateDraft,
  roots: readonly PoliticsIdeology[],
): string {
  return draft.parent || roots[0]?.script_id || "";
}

export function createRootBodyError(draft: IdeologyCreateDraft) {
  return draft.intent === "root" ? rootBodyDraftError(draft.rootBody) : null;
}

export function createDraftReady(
  draft: IdeologyCreateDraft,
  parent: string,
): boolean {
  return Boolean(
    draft.intent &&
    Object.keys(normalizedLocaleMap(draft.name)).length > 0 &&
    draft.scriptId.trim() &&
    !(draft.intent === "subtype" && !parent) &&
    !(
      draft.intent === "root" &&
      (!draft.rootBody.trim() || Boolean(createRootBodyError(draft)))
    ),
  );
}

export function createRequest(
  draft: IdeologyCreateDraft & { intent: CreateIntent },
  parent: string,
):
  | { kind: "subtype"; input: PoliticsSubtypeCreateInput }
  | { kind: "root"; input: PoliticsRootCreateInput } {
  const name = normalizedLocaleMap(draft.name);
  const desc = normalizedLocaleMap(draft.desc);
  const scriptId = draft.scriptId.trim();
  return draft.intent === "subtype"
    ? {
        kind: "subtype",
        input: {
          script_id: scriptId,
          root_id: parent,
          name,
          desc,
          random_disabled: draft.randomDisabled,
        },
      }
    : {
        kind: "root",
        input: { script_id: scriptId, body: draft.rootBody.trim(), name, desc },
      };
}
