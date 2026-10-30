import type { GuiNode } from "@/lib/guiRenderApi";

type Margin = {
  top?: unknown;
  left?: unknown;
  bottom?: unknown;
  right?: unknown;
};

export function insetByMargin(node: GuiNode): GuiNode {
  const margin = node.raw?.margin as Margin | undefined;
  if (!margin) return node;
  const left = Number(margin.left) || 0;
  const top = Number(margin.top) || 0;
  const right = Number(margin.right) || 0;
  const bottom = Number(margin.bottom) || 0;
  if (!left && !top && !right && !bottom) return node;
  node.children = [
    {
      type: "containerwindowtype",
      name: `${node.name}_content`,
      position: { x: left, y: top },
      size: { width: -(left + right), height: -(top + bottom) },
      orientation: null,
      origo: null,
      scale: null,
      sprite: null,
      frame: null,
      text: null,
      text_source: null,
      font: null,
      format: null,
      maxwidth: null,
      maxheight: null,
      text_color: null,
      centerposition: false,
      alwaystransparent: false,
      background: null,
      children: node.children ?? [],
      raw: { clipping: "no" },
    },
  ];
  return node;
}
