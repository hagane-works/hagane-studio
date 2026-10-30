"use client";

import { useDeferredValue, useMemo, useState } from "react";
import { useLocale, useTranslations } from "use-intl";
import useSWR, { mutate as globalMutate } from "swr";
import { toast } from "sonner";
import { charactersApi } from "@/lib/api/characters";
import { projectsApi } from "@/lib/api/project";
import type { GuiCharacter, GuiCharacters } from "@/lib/guiRenderApi";
import { subIdeologyOptions } from "@/lib/ideology/registry";
import { isContentLocale } from "@/lib/store/contentLocaleStore";
import { ADVISOR_SLOT_OPTIONS, type RoleType } from "@/types";
import { isGameDataKey } from "./gameStage";
import { FIELD } from "./GamePanel";
import { useCharacters } from "./CharacterLayer";
import {
  PERSON_KINDS,
  newPersonProblem,
  newPersonRow,
  personToken,
} from "./personCreate";

export type PersonGroup =
  "leader" | "fieldMarshal" | "general" | "admiral" | "advisor";

const GROUPS: readonly { role: string; group: PersonGroup }[] = [
  { role: "field_marshal", group: "fieldMarshal" },
  { role: "corps_commander", group: "general" },
  { role: "navy_leader", group: "admiral" },
];

export function commanderGroups(
  commanders: readonly GuiCharacter[],
  query: string,
): { group: PersonGroup; people: GuiCharacter[] }[] {
  const want = query.trim().toLowerCase();
  const placed = new Set<string>();
  return GROUPS.map(({ role, group }) => {
    const people = commanders
      .filter(
        (person) =>
          (person.roles ?? []).includes(role) && !placed.has(person.token),
      )
      .filter(
        (person) =>
          !want ||
          person.name.toLowerCase().includes(want) ||
          person.token.toLowerCase().includes(want),
      )
      .sort((a, b) => a.name.localeCompare(b.name));
    people.forEach((person) => placed.add(person.token));
    return { group, people };
  }).filter((group) => group.people.length > 0);
}

export function peopleGroups(
  data: Pick<GuiCharacters, "leader" | "advisors" | "commanders"> | undefined,
  query: string,
): { group: PersonGroup; people: GuiCharacter[] }[] {
  const want = query.trim().toLowerCase();
  const matches = (person: GuiCharacter) =>
    !want ||
    person.name.toLowerCase().includes(want) ||
    person.token.toLowerCase().includes(want);
  const leader =
    data?.leader && matches(data.leader)
      ? [{ group: "leader" as const, people: [data.leader] }]
      : [];
  const commanders = commanderGroups(data?.commanders ?? [], query);
  const placed = new Set(
    [...leader, ...commanders].flatMap((group) =>
      group.people.map((person) => person.token),
    ),
  );
  const advisors: GuiCharacter[] = [];
  for (const person of data?.advisors ?? []) {
    if (!matches(person) || placed.has(person.token)) continue;
    placed.add(person.token);
    advisors.push(person);
  }
  advisors.sort((a, b) => a.name.localeCompare(b.name));
  return [
    ...leader,
    ...commanders,
    ...(advisors.length
      ? [{ group: "advisor" as const, people: advisors }]
      : []),
  ];
}

function NewPersonForm({
  projectId,
  tag,
  leaderIdeology,
  onCreated,
}: {
  projectId: number;
  tag: string;

  leaderIdeology?: string;
  onCreated: (token: string, name: string) => void;
}) {
  const t = useTranslations("gameViewEdit.person");
  const tEnums = useTranslations("enums");
  const locale = useLocale();
  const [name, setName] = useState("");
  const [kind, setKind] = useState<RoleType>("advisor");
  const [slot, setSlot] = useState("political_advisor");
  const [ideology, setIdeology] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  const project = useSWR(
    kind === "country_leader" ? ["game-view-project", projectId] : null,
    () => projectsApi.get(projectId),
    {
      revalidateOnFocus: false,
    },
  );
  const ideologies = useMemo(
    () =>
      subIdeologyOptions(
        project.data?.ideology_registry,
        isContentLocale(locale) ? locale : "en",
      ),
    [project.data, locale],
  );
  const person = {
    tag,
    name,
    kind,
    slot,
    ideology: ideology ?? leaderIdeology ?? "",
  };
  const problem = newPersonProblem(person);

  const create = async () => {
    if (problem || busy) return;
    setBusy(true);
    try {
      const created = await charactersApi.create(
        projectId,
        newPersonRow(person),
      );
      await globalMutate(
        (key) =>
          isGameDataKey(key) ||
          (Array.isArray(key) && key[0] === "game-view-characters-db"),
      );
      toast.success(t("created", { name: created.name }));
      setName("");
      onCreated(created.tag, created.name);
    } catch (err) {
      toast.error(
        t("createFailed", {
          error: err instanceof Error ? err.message : String(err),
        }),
      );
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="space-y-2">
      <label className="block">
        <span className="text-micro font-medium text-muted-foreground">
          {t("name")}
        </span>
        <input
          value={name}
          disabled={busy}
          onChange={(e) => setName(e.target.value)}
          placeholder="Otto Weber"
          className={FIELD}
        />
      </label>
      <label className="block">
        <span className="text-micro font-medium text-muted-foreground">
          {t("role")}
        </span>
        <select
          aria-label={t("role")}
          value={kind}
          disabled={busy}
          onChange={(e) => setKind(e.target.value as RoleType)}
          className={FIELD}
        >
          {PERSON_KINDS.map((option) => (
            <option key={option.value} value={option.value}>
              {t(`roleOption.${option.labelKey}`)}
            </option>
          ))}
        </select>
      </label>
      {kind === "advisor" && (
        <label className="block">
          <span className="text-micro font-medium text-muted-foreground">
            {t("advisorSlot")}
          </span>
          <select
            aria-label={t("advisorSlot")}
            value={slot}
            disabled={busy}
            onChange={(e) => setSlot(e.target.value)}
            className={FIELD}
          >
            {ADVISOR_SLOT_OPTIONS.map((option) => (
              <option key={option.value} value={option.value}>
                {tEnums(option.label_i18n_key)}
              </option>
            ))}
          </select>
        </label>
      )}
      {kind === "country_leader" && (
        <label className="block">
          <span className="text-micro font-medium text-muted-foreground">
            {t("ideology")}
          </span>
          <select
            aria-label={t("ideology")}
            value={person.ideology}
            disabled={busy}
            onChange={(e) => setIdeology(e.target.value)}
            className={`${FIELD} font-mono`}
          >
            <option value="">{t("choose")}</option>
            {ideologies.map((option) => (
              <option key={option.value} value={option.value}>
                {option.label}
              </option>
            ))}
          </select>
        </label>
      )}
      {name.trim() && (
        <p className="font-mono text-micro text-muted-foreground">
          {personToken(tag, name)}
        </p>
      )}
      {problem && name.trim() && (
        <p className="text-micro text-destructive">{t(`problem.${problem}`)}</p>
      )}
      <button
        type="button"
        disabled={busy || !!problem}
        onClick={() => void create()}
        className="rounded bg-primary px-3 py-1 text-micro text-primary-foreground hover:opacity-90 disabled:cursor-not-allowed disabled:opacity-50"
      >
        {t("create")}
      </button>
      <p className="text-micro text-muted-foreground">
        {t("createNote", { tag })}
      </p>
    </div>
  );
}

export function PersonSwitcher({
  projectId,
  tag,
  date,
  current,
  onChoose,
}: {
  projectId: number;
  tag: string;
  date?: string | null;

  current: string | null;
  onChoose: (token: string, name: string) => void;
}) {
  const t = useTranslations("gameViewEdit.person");
  const [open, setOpen] = useState(false);
  const [creating, setCreating] = useState(false);
  const [query, setQuery] = useState("");
  const q = useDeferredValue(query);
  const people = useCharacters(projectId, open ? tag : null, date ?? null);
  const groups = useMemo(() => peopleGroups(people.data, q), [people.data, q]);

  const choose = (token: string, name: string) => {
    setOpen(false);
    onChoose(token, name);
  };
  return (
    <div className="w-full rounded-md border border-border bg-card p-3 text-card-foreground shadow-lg">
      <button
        type="button"
        onClick={() => setOpen((v) => !v)}
        aria-expanded={open}
        className="flex w-full items-center justify-between gap-2 text-left"
      >
        <span className="min-w-0">
          <span className="block text-micro text-muted-foreground">
            {t("switcherLabel")}
          </span>
          <span className="block truncate text-sm">{t("switcherHint")}</span>
        </span>
        <span className="text-muted-foreground">{open ? "▴" : "▾"}</span>
      </button>
      {open && (
        <div className="mt-3 space-y-3">
          <input
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder={t("search")}
            aria-label={t("searchLabel")}
            className="w-full rounded border border-input bg-background px-2 py-1 text-sm"
          />
          <div className="max-h-96 space-y-3 overflow-y-auto">
            {groups.map((group) => (
              <section key={group.group}>
                <div className="text-micro font-medium text-muted-foreground">
                  {t(`group.${group.group}`)}
                </div>
                <ul className="mt-1 space-y-1">
                  {group.people.map((person) => (
                    <li key={person.token}>
                      <button
                        type="button"
                        onClick={() => choose(person.token, person.name)}
                        className={`w-full rounded px-2 py-1 text-left text-micro ${
                          person.token === current
                            ? "bg-primary text-primary-foreground"
                            : "hover:bg-muted"
                        }`}
                      >
                        {person.name}
                      </button>
                    </li>
                  ))}
                </ul>
              </section>
            ))}
            {people.data && !groups.length && (
              <p className="text-micro text-muted-foreground">{t("noMatch")}</p>
            )}
            {!people.data && (
              <p className="text-micro text-muted-foreground">
                {t("loadingList")}
              </p>
            )}
          </div>
          <div className="border-t border-border pt-3">
            <button
              type="button"
              aria-expanded={creating}
              onClick={() => setCreating((v) => !v)}
              className="rounded border border-border px-2 py-0.5 text-micro hover:bg-muted"
            >
              {creating ? t("hideCreate") : t("showCreate")}
            </button>
            {creating && (
              <div className="mt-2">
                <NewPersonForm
                  projectId={projectId}
                  tag={tag}
                  leaderIdeology={people.data?.leader?.ideology}
                  onCreated={(token, name) => {
                    setCreating(false);
                    choose(token, name);
                  }}
                />
              </div>
            )}
          </div>
        </div>
      )}
    </div>
  );
}
