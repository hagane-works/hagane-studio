"use client";

import { useState } from "react";
import useSWR from "swr";
import { useTranslations } from "use-intl";
import { eventsApi } from "@/lib/api/content";
import { EventEditor } from "@/components/workbench/EventEditor";
import type { EventPreviewTarget } from "@/components/workbench/EventInGamePreview";
import type { GameEvent } from "@/types";
import { eventDraft } from "./eventEdit";
import { useEvent, type GuiEvent } from "./eventModel";
import { GamePanel } from "./GamePanel";
import { pick, recordChange, rowUndo } from "./changeLog";
import { BaseTextEdit } from "./BaseTextEdit";
import { changedKeys, useWorkbenchProject } from "./workbenchProject";

export type EventField = string;

export function eventFocusTarget(
  field: EventField,
): { target: EventPreviewTarget; option: number | null } | null {
  if (field === "title") return { target: "title", option: null };
  if (field === "desc") return { target: "description", option: null };
  const option = /^option:(\d+)$/.exec(field);
  if (option) return { target: "options", option: Number(option[1]) };
  if (field === "picture") return { target: "picture", option: null };
  return null;
}

export function EventEditPanel({
  projectId,
  eventId,
  tag,
  date,
  focusField,
  onSaved,
  onClose,
}: {
  projectId: number;
  eventId: string;
  tag: string | null;
  date: string | null;

  focusField?: { field: EventField; at: number } | null;

  onSaved: (eventId: string, shown: Partial<GuiEvent>) => void;
  onClose: () => void;
}) {
  const t = useTranslations("gameViewContent.event");
  const tEdit = useTranslations("gameViewEdit");
  const shown = useEvent(projectId, eventId, tag, date);
  const row = useSWR(
    ["game-view-event-db", projectId, eventId],
    () => eventsApi.findByScriptId(projectId, eventId),
    {
      revalidateOnFocus: false,
    },
  );
  const projectReady = useWorkbenchProject(projectId);

  const [consumedAt, setConsumedAt] = useState<number | null>(null);
  const focus =
    focusField && focusField.at !== consumedAt
      ? eventFocusTarget(focusField.field)
      : null;

  const save = async (data: Partial<GameEvent>) => {
    const current = row.data;
    if (!current) return;
    const id = current.id;
    const keys = changedKeys(current, data);
    const before = pick(current, keys);
    const saved = await eventsApi.update(projectId, id, data);
    const texts = shown.data ? eventDraft(saved, shown.data) : null;
    recordChange({
      projectId,
      title: texts?.title || shown.data?.title || eventId,
      kind: t("kind", { id: eventId }),
      workbenchHref: `/project/${projectId}/events?namespace=${current.namespace_id}&event=${id}`,
      undo: keys.length
        ? rowUndo({
            before,
            after: pick(saved, keys),
            load: () => eventsApi.get(projectId, id),
            save: (back) => eventsApi.update(projectId, id, back),
          })
        : undefined,
    });
    onSaved(
      eventId,
      texts
        ? {
            title: texts.title,
            desc: texts.desc,
            picture: texts.picture,
            options: texts.options.map((o) => ({ name: o.key, label: o.text })),
          }
        : {},
    );
    await row.mutate();
  };

  return (
    <GamePanel
      title={shown.data?.title || eventId}
      kind={t("kind", { id: eventId })}
      source={row.isLoading ? undefined : row.data ? "项目" : "原版"}
      workbenchHref={
        row.data
          ? `/project/${projectId}/events?namespace=${row.data.namespace_id}&event=${row.data.id}`
          : null
      }

      wide={row.data ? "workbench" : true}
      onClose={onClose}
    >
      {row.isLoading && (
        <p className="mt-4 text-micro text-muted-foreground">{t("loading")}</p>
      )}
      {!row.isLoading && !row.data && shown.data && (
        <BaseTextEdit
          projectId={projectId}
          origin={shown.data.origin}
          title={shown.data.title || eventId}
          fields={[
            ...(shown.data.title_key
              ? [{ field: "title" as const, key: shown.data.title_key }]
              : []),
            ...(shown.data.desc_key
              ? [{ field: "body" as const, key: shown.data.desc_key }]
              : []),
            ...shown.data.options.map((option, i) => ({
              field: "option" as const,
              key: option.name,
              n: i + 1,
            })),
          ]}
          onSaved={(key, value) => {
            const data = shown.data;
            const text = value ?? undefined;
            onSaved(
              eventId,
              key === data?.title_key
                ? { title: text }
                : key === data?.desc_key
                  ? { desc: text }
                  : {},
            );
          }}
        />
      )}
      {!row.isLoading && !row.data && !shown.data && (
        <p className="mt-4 text-micro text-muted-foreground">
          {t("unreadable")}
        </p>
      )}

      {row.data &&
        (projectReady ? (
          <div className="-mx-4 mt-2">
            <EventEditor
              key={row.data.id}
              event={row.data}
              projectId={projectId}
              countryTag={tag ?? ""}
              onSave={save}
              onPictureUploaded={() => {
                void row.mutate();
                onSaved(eventId, {});
              }}
              focusTarget={focus?.target ?? null}
              focusOptionIndex={focus?.option ?? null}
              onFocusTargetConsumed={() =>
                setConsumedAt(focusField?.at ?? null)
              }
            />
          </div>
        ) : (
          <p className="mt-4 text-micro text-muted-foreground">
            {tEdit("panel.loadingProject")}
          </p>
        ))}
    </GamePanel>
  );
}
