type PositionedEntity = {
  id: number;
  x: number;
  y: number;
};

export function findPersistedSelectedEntity<T extends PositionedEntity>(
  persistedEntities: readonly T[],
  selectedEntityId: number | null,
): T | null {
  if (selectedEntityId == null) return null;
  return (
    persistedEntities.find((entity) => entity.id === selectedEntityId) ?? null
  );
}

export function buildDraftPreviewEntities<T extends PositionedEntity>(
  persistedEntities: T[],
  selectedEntityId: number | null,
  draft: T | null,
): T[] {
  if (!draft || draft.id !== selectedEntityId) return persistedEntities;

  return persistedEntities.map((entity) => {
    if (entity.id !== draft.id) return entity;
    if (entity.x === draft.x && entity.y === draft.y) return draft;
    return { ...draft, x: entity.x, y: entity.y };
  });
}
