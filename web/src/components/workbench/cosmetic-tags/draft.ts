import {
  COSMETIC_SUFFIXES,
  type CosmeticDraft,
  type CosmeticNameOperation,
  type CosmeticRGB,
  type CosmeticTagPatch,
  type CosmeticTagView,
} from "@/lib/api/cosmeticTags";

export function draftFrom(view: CosmeticTagView): CosmeticDraft {
  return {
    name_locs: structuredClone(view.name_locs ?? {}),
    color: view.color,
    color_ui: view.color_ui,
  };
}

export function sameColor(
  a: CosmeticRGB | null,
  b: CosmeticRGB | null,
): boolean {
  return (
    a === b ||
    (a !== null &&
      b !== null &&
      a.every((channel, index) => channel === b[index]))
  );
}

export function nameOperations(
  before: CosmeticDraft,
  after: CosmeticDraft,
): CosmeticNameOperation[] {
  const operations: CosmeticNameOperation[] = [];
  for (const locale of new Set([
    ...Object.keys(before.name_locs),
    ...Object.keys(after.name_locs),
  ])) {
    for (const suffix of COSMETIC_SUFFIXES) {
      const oldValue = before.name_locs[locale]?.[suffix];
      const newValue = after.name_locs[locale]?.[suffix];
      if (oldValue !== newValue)
        operations.push({ locale, suffix, value: newValue ?? null });
    }
  }
  return operations;
}

export function patchFor(
  view: CosmeticTagView,
  draft: CosmeticDraft,
): CosmeticTagPatch {
  const before = draftFrom(view);
  return {
    expected_revision: view.revision,
    name_ops: nameOperations(before, draft),
    ...(!sameColor(before.color, draft.color) ? { color: draft.color } : {}),
    ...(!sameColor(before.color_ui, draft.color_ui)
      ? { color_ui: draft.color_ui }
      : {}),
  };
}

export function hasChanges(
  view: CosmeticTagView,
  draft: CosmeticDraft,
): boolean {
  const patch = patchFor(view, draft);
  return patch.name_ops.length > 0 || "color" in patch || "color_ui" in patch;
}

export function retainEdits(
  view: CosmeticTagView,
  submitted: CosmeticDraft,
  current: CosmeticDraft,
): CosmeticDraft {
  const next = draftFrom(view);
  for (const operation of nameOperations(submitted, current)) {
    next.name_locs[operation.locale] = { ...next.name_locs[operation.locale] };
    if (operation.value === null)
      delete next.name_locs[operation.locale][operation.suffix];
    else next.name_locs[operation.locale][operation.suffix] = operation.value;
  }
  if (!sameColor(submitted.color, current.color)) next.color = current.color;
  if (!sameColor(submitted.color_ui, current.color_ui))
    next.color_ui = current.color_ui;
  return next;
}
