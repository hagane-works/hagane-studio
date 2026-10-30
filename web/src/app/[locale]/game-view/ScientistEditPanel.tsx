"use client";

import { useState } from "react";
import useSWR, { mutate as globalMutate } from "swr";
import { toast } from "sonner";
import { useTranslations } from "use-intl";
import { charactersApi } from "@/lib/api/characters";
import {
  SCIENTIST_GENERIC_TRAITS,
  SCIENTIST_SPEC_TRAITS,
  type Character,
  type CustomTrait,
  type ScientistSpecialization,
} from "@/types";
import { isGameDataKey } from "./gameStage";
import { BUTTON, FIELD, GamePanel, NUMBER, PanelSave } from "./GamePanel";
import { pick, recordChange, rowUndo } from "./changeLog";
import { BasePersonText } from "./BaseTextEdit";
import { ProjectCharacterEditor } from "./CharacterEditPanel";
import type { GuiScientist } from "./ScientistsLayer";

export interface ScientistDraft {
  levels: Record<string, number>;
  traits: string[];
}

export function scientistDraft(
  row: Pick<Character, "scientist_data" | "traits">,
): ScientistDraft {
  const levels: Record<string, number> = {};
  for (const item of row.scientist_data?.specializations ?? [])
    levels[item.spec] = item.level;
  return { levels, traits: [...(row.traits ?? [])] };
}

export function scientistPatch(
  row: Pick<Character, "scientist_data" | "traits">,
  draft: ScientistDraft,
  order: readonly string[],
): Partial<Character> {
  const start = scientistDraft(row);
  const patch: Partial<Character> = {};
  const specs = [
    ...new Set([
      ...Object.keys(start.levels),
      ...order,
      ...Object.keys(draft.levels),
    ]),
  ].filter((spec) => (draft.levels[spec] ?? 0) > 0);
  const same =
    specs.length ===
      Object.keys(start.levels).filter((spec) => start.levels[spec] > 0)
        .length &&
    specs.every((spec) => start.levels[spec] === draft.levels[spec]);
  if (!same) {
    patch.scientist_data = {
      ...(row.scientist_data ?? { allow_multiple: false, specializations: [] }),
      specializations: specs.map((spec) => ({
        spec,
        level: draft.levels[spec],
      })),
    };
  }
  if (draft.traits.join(" ") !== start.traits.join(" "))
    patch.traits = draft.traits;
  return patch;
}

export function traitsInEditor(
  form: Pick<Character, "scientist_data">,
  custom: readonly string[],
): Set<string> {
  const spec = (form.scientist_data?.specializations?.[0]?.spec ||
    "specialization_land") as ScientistSpecialization;
  return new Set(
    [
      ...SCIENTIST_GENERIC_TRAITS,
      ...(SCIENTIST_SPEC_TRAITS[spec] ?? []),
      ...custom,
    ].map((trait) => trait.toLowerCase()),
  );
}

export function ScientistEditPanel({
  projectId,
  scientist,
  specializations,
  catalog,
  onClose,
}: {
  projectId: number;
  scientist: GuiScientist;
  specializations: { id: string; label?: string }[];
  catalog: { id: string; label: string }[];
  onClose: () => void;
}) {
  const t = useTranslations("gameViewResearch.scientist");
  const tPerson = useTranslations("gameViewEdit.person");
  const rows = useSWR(
    ["game-view-characters-db", projectId],
    () => charactersApi.list(projectId),
    { revalidateOnFocus: false },
  );
  const row =
    rows.data?.find((character) => character.tag === scientist.token) ?? null;

  const [form, setForm] = useState<Character | null>(null);
  const current = row && form?.id === row.id ? form : row;
  const editorUnsaved =
    !!row &&
    !!form &&
    form.id === row.id &&
    JSON.stringify(form) !== JSON.stringify(row);

  const custom = useSWR(
    row ? ["game-view-custom-traits", projectId] : null,
    async () => {
      const response = await fetch(
        `/api/go/projects/${projectId}/custom-traits`,
      );
      return response.ok ? ((await response.json()) as CustomTrait[]) : [];
    },
    { revalidateOnFocus: false },
  );
  const customScientist = (custom.data ?? [])
    .filter((trait) => trait.trait_type === "scientist")
    .map((trait) => trait.trait_id);
  const reload = async () => {
    await rows.mutate();
    await globalMutate(isGameDataKey);
  };
  return (
    <GamePanel
      title={scientist.name}
      kind={t("kind", { token: scientist.token })}
      source={rows.isLoading ? undefined : row ? "项目" : "原版"}
      workbenchHref={
        row ? `/project/${projectId}/characters?c=${row.id}` : null
      }

      wide={row ? "workbench" : true}
      onClose={onClose}
    >
      {rows.isLoading ? (
        <p className="mt-4 text-micro text-muted-foreground">
          {tPerson("loading")}
        </p>
      ) : !row ? (
        <BasePersonText
          projectId={projectId}
          token={scientist.token}
          name={scientist.name}
          onSaved={() =>
            void globalMutate(
              (key) => Array.isArray(key) && key[0] === "game-view-scientists",
            )
          }
        />
      ) : (
        <ProjectCharacterEditor
          projectId={projectId}
          row={row}
          kind={t("kind", { token: row.tag })}
          name={scientist.name}
          reload={reload}
          onDraftChange={setForm}
        >
          {current?.role_type === "scientist" && (
            <ScientistExtras
              key={`${row.id}/${JSON.stringify(scientistDraft(row))}`}
              projectId={projectId}
              row={row}
              specializations={specializations}
              catalog={catalog}
              inEditor={traitsInEditor(current, customScientist)}
              locked={editorUnsaved}
              reload={reload}
            />
          )}
        </ProjectCharacterEditor>
      )}
    </GamePanel>
  );
}

function ScientistExtras({
  projectId,
  row,
  specializations,
  catalog,
  inEditor,
  locked,
  reload,
}: {
  projectId: number;
  row: Character;
  specializations: { id: string; label?: string }[];
  catalog: { id: string; label: string }[];

  inEditor: ReadonlySet<string>;

  locked: boolean;
  reload: () => Promise<unknown>;
}) {
  const t = useTranslations("gameViewResearch.scientist");
  const tCommon = useTranslations("gameViewResearch.common");
  const [draft, setDraft] = useState<ScientistDraft>(() => scientistDraft(row));
  const [busy, setBusy] = useState(false);
  const order = specializations.map((spec) => spec.id);
  const patch = scientistPatch(row, draft, order);
  const changed = Object.keys(patch).length > 0;
  const labelOf = (id: string) =>
    catalog.find((trait) => trait.id.toLowerCase() === id.toLowerCase())
      ?.label ?? id;
  const held = new Set(draft.traits.map((trait) => trait.toLowerCase()));
  const others = draft.traits.filter(
    (trait) => !inEditor.has(trait.toLowerCase()),
  );
  const offered = catalog.filter(
    (trait) =>
      !held.has(trait.id.toLowerCase()) &&
      !inEditor.has(trait.id.toLowerCase()),
  );
  const save = async () => {
    setBusy(true);
    try {
      const id = row.id;
      const keys = Object.keys(patch);
      const before = pick(row, keys);
      const saved = await charactersApi.update(projectId, row.id, patch);
      recordChange({
        projectId,
        title: row.name || row.tag,
        kind: t("kind", { token: row.tag }),
        workbenchHref: `/project/${projectId}/characters?c=${id}`,
        undo: rowUndo({
          before,
          after: pick(saved, keys),
          load: () => charactersApi.get(projectId, id),
          save: (back) => charactersApi.update(projectId, id, back),
        }),
      });
      await reload();
      toast.success(t("saved", { name: row.name || row.tag }));
    } catch (err) {
      toast.error(
        t("saveFailed", {
          error: err instanceof Error ? err.message : String(err),
        }),
      );
    } finally {
      setBusy(false);
    }
  };
  return (
    <section className="mt-4 border-t border-border pt-3">
      <p className="text-micro text-muted-foreground">{t("extrasNote")}</p>
      <fieldset disabled={locked} className="mt-3 space-y-2">
        <div className="text-xs">
          {t("levels")}
          <div className="mt-1 grid grid-cols-2 gap-2">
            {specializations.map((spec) => (
              <label
                key={spec.id}
                className="block text-micro text-muted-foreground"
              >
                {spec.label ?? spec.id}
                <input
                  aria-label={`level ${spec.id}`}
                  className={NUMBER}
                  inputMode="numeric"
                  value={String(draft.levels[spec.id] ?? 0)}
                  onChange={(e) => {
                    const level = Math.max(
                      0,
                      Math.min(10, Math.round(Number(e.target.value) || 0)),
                    );
                    setDraft({
                      ...draft,
                      levels: { ...draft.levels, [spec.id]: level },
                    });
                  }}
                />
              </label>
            ))}
          </div>
        </div>
        <div className="text-xs">
          {t("otherTraits")}
          <ul className="mt-1 space-y-1">
            {others.map((trait) => (
              <li
                key={trait}
                className="flex items-center justify-between gap-2 text-micro"
              >
                <span className="truncate">{labelOf(trait)}</span>
                <button
                  type="button"
                  className={BUTTON}
                  aria-label={`remove ${trait}`}
                  onClick={() =>
                    setDraft({
                      ...draft,
                      traits: draft.traits.filter((token) => token !== trait),
                    })
                  }
                >
                  {tCommon("remove")}
                </button>
              </li>
            ))}
          </ul>
          <select
            aria-label="add trait"
            className={FIELD}
            value=""
            onChange={(e) => {
              if (e.target.value)
                setDraft({
                  ...draft,
                  traits: [...draft.traits, e.target.value],
                });
            }}
          >
            <option value="">{t("addTrait")}</option>
            {offered.map((trait) => (
              <option key={trait.id} value={trait.id}>
                {trait.label}
              </option>
            ))}
          </select>
        </div>
      </fieldset>
      <PanelSave
        label={t("saveLabel")}
        onSave={() => void save()}
        canSave={changed && !locked}
        busy={busy}
        onRevert={changed ? () => setDraft(scientistDraft(row)) : undefined}
        problem={locked ? t("locked") : undefined}
        note={t("note")}
      />
    </section>
  );
}
