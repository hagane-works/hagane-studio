import type { GuiNode } from "@/lib/guiRenderApi";

const SCROLL_KEY = "hagane_scroll";

export type ScrollOffset = { x: number; y: number };

export function containerScroll(node: GuiNode): ScrollOffset | null {
  const value = node.raw?.[SCROLL_KEY] as Partial<ScrollOffset> | undefined;
  if (!value || typeof value !== "object") return null;
  return { x: Number(value.x) || 0, y: Number(value.y) || 0 };
}

export function withContainerScroll(
  root: GuiNode,
  scrolls: (node: GuiNode) => boolean,
  offset: ScrollOffset,
): GuiNode {
  const visit = (node: GuiNode): GuiNode => {
    let children = node.children;
    if (children?.length) {
      const next = children.map(visit);
      if (next.some((child, index) => child !== children?.[index]))
        children = next;
    }
    const marked = scrolls(node);
    if (!marked && children === node.children) return node;
    return {
      ...node,
      children,
      raw: marked
        ? { ...(node.raw ?? {}), [SCROLL_KEY]: { x: offset.x, y: offset.y } }
        : node.raw,
    };
  };
  return visit(root);
}

type Rect = { x: number; y: number; w: number; h: number };

export function clipRects<T extends Rect>(rects: T[], box: Rect): T[] {
  const out: T[] = [];
  for (const rect of rects) {
    const x = Math.max(rect.x, box.x);
    const y = Math.max(rect.y, box.y);
    const right = Math.min(rect.x + rect.w, box.x + box.w);
    const bottom = Math.min(rect.y + rect.h, box.y + box.h);
    if (right <= x || bottom <= y) continue;
    out.push(
      x === rect.x &&
        y === rect.y &&
        right === rect.x + rect.w &&
        bottom === rect.y + rect.h
        ? rect
        : { ...rect, x, y, w: right - x, h: bottom - y },
    );
  }
  return out;
}
