import { createContext, useContext, useEffect } from "react";
import type { GuiNode, GuiSprite, GuiWindow } from "@/lib/guiRenderApi";
import { applySwaps, type SpriteSwaps } from "./spriteSwaps";
import {
  guiTextOverrides,
  locEditOverridesFor,
  type EditSelection,
  type PrintedText,
} from "./editModel";

export interface PrintedRegistry {
  publish: (
    window: string,
    layer: string,
    printed: Readonly<Record<string, PrintedText>> | null,
  ) => void;
  lookup: (window: string, id: string) => PrintedText | undefined;
}

export function createPrintedRegistry(): PrintedRegistry {
  const byWindow = new Map<
    string,
    Map<string, Readonly<Record<string, PrintedText>>>
  >();
  return {
    publish: (window, layer, printed) => {
      let layers = byWindow.get(window);
      if (!printed) {
        layers?.delete(layer);
        return;
      }
      if (!layers) {
        layers = new Map();
        byWindow.set(window, layers);
      }
      layers.set(layer, printed);
    },
    lookup: (window, id) => {
      for (const printed of byWindow.get(window)?.values() ?? []) {
        if (printed[id]) return printed[id];
      }
      return undefined;
    },
  };
}

export function withPrintedOver(
  selection: EditSelection,
  registry: PrintedRegistry,
): EditSelection {
  const over = (id: string, shown: string | undefined, has: boolean) =>
    shown === undefined && !has
      ? registry.lookup(selection.win.name, id)
      : undefined;
  const own = over(selection.id, selection.shown, Boolean(selection.source));
  return {
    ...selection,
    ...(own ? { shown: own.text, source: own.source } : {}),
    stack: selection.stack?.map((entry) => {
      const printed = over(entry.id, entry.shown, Boolean(entry.source));
      return printed
        ? { ...entry, shown: printed.text, source: printed.source }
        : entry;
    }),
  };
}

export function usePrintedOver(
  win: GuiWindow,
  layer: string,
  printed: Readonly<Record<string, PrintedText>>,
): void {
  const registry = useContext(EditContext).printedOver;
  const windowName = win.name;
  useEffect(() => {
    registry.publish(windowName, layer, printed);
    return () => registry.publish(windowName, layer, null);
  }, [registry, windowName, layer, printed]);
}

export interface EditContextValue {
  editMode: boolean;

  contentEditMode: boolean;

  selection: { owner: string; id: string } | null;
  select: (selection: EditSelection) => void;
  locEdits: Readonly<Record<string, string>>;

  guiTexts: Readonly<Record<string, string>>;

  printedOver: PrintedRegistry;
  swaps: SpriteSwaps;
}

const NONE: Readonly<Record<string, string>> = {};
const NO_PRINTED: Readonly<Record<string, PrintedText>> = {};

export const EditContext = createContext<EditContextValue>({
  editMode: false,
  contentEditMode: false,
  selection: null,
  select: () => {},
  locEdits: NONE,
  guiTexts: NONE,
  printedOver: createPrintedRegistry(),
  swaps: {},
});

export interface EditCanvasProps {
  editMode: boolean;
  selectedId: string | null;
  selectionOwner: string;
  onSelect: (
    id: string,
    node: GuiNode,
    stack?: { id: string; node: GuiNode }[],
  ) => void;
  sprites: Record<string, GuiSprite>;
  versions: Record<string, number>;
  textOverrides: Record<string, string>;
}

export function useEditCanvas() {
  const ctx = useContext(EditContext);
  return (
    owner: string,
    win: GuiWindow,
    node: GuiNode,
    sprites: Record<string, GuiSprite>,
    overrides: Readonly<Record<string, string>> = NONE,
    printed: Readonly<Record<string, PrintedText>> = NO_PRINTED,
  ): EditCanvasProps => {
    const swapped = applySwaps(sprites, ctx.swaps);
    const rewritten =
      Object.keys(ctx.locEdits).length || Object.keys(ctx.guiTexts).length
        ? {
            ...locEditOverridesFor(node, ctx.locEdits),
            ...guiTextOverrides({ ...win, node }, ctx.guiTexts),
          }
        : NONE;
    return {
      editMode: ctx.editMode,
      selectedId: ctx.selection?.owner === owner ? ctx.selection.id : null,
      selectionOwner: owner,
      onSelect: (id, picked, stack) =>
        ctx.select({
          owner,
          win: { ...win, sprites: swapped.sprites },
          id,
          node: picked,
          shown: printed[id]?.text ?? overrides[id],
          source: printed[id]?.source,
          stack: stack?.map((entry) => ({
            ...entry,
            shown: printed[entry.id]?.text ?? overrides[entry.id],
            source: printed[entry.id]?.source,
          })),
        }),
      sprites: swapped.sprites,
      versions: swapped.versions,
      textOverrides:
        rewritten === NONE ? { ...overrides } : { ...rewritten, ...overrides },
    };
  };
}
