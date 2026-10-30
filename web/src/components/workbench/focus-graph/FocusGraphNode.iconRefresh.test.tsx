import React from "react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, fireEvent, render } from "@testing-library/react";

vi.mock("use-intl", () => ({ useTranslations: () => (key: string) => key }));
vi.mock("@/platform/image", () => ({
  default: ({
    src,
    alt,
    onError,
  }: {
    src: string;
    alt: string;
    onError?: () => void;
  }) => <img src={src} alt={alt} onError={onError} />,
}));
vi.mock("@xyflow/react", () => ({
  Handle: () => null,
  Position: { Left: "left", Right: "right", Top: "top", Bottom: "bottom" },
}));
vi.mock("./inGamePreviewContext", () => ({
  useFocusInGamePreview: () => false,
}));

import { FocusGraphNode } from "./FocusGraphNode";

afterEach(cleanup);

type NodeProps = React.ComponentProps<typeof FocusGraphNode>;
const nodeProps = (iconUrl: string) =>
  ({
    id: "1",
    selected: false,
    data: {
      focusId: "_New_Focus_5",
      label: "New Focus",
      icon: "GFX_focus_new_focus_5",
      iconUrl,
      baseModIconUrls: [],
      vanillaIconUrl: null,
      cost: 10,
    },
  }) as unknown as NodeProps;

const icon = (root: HTMLElement) =>
  root.querySelector('img[alt="GFX_focus_new_focus_5"]');

describe("FocusGraphNode · 换图标后画布刷新", () => {
  it("换 URL 时换元素,上一张被中止的请求不会把新图标标记成坏图", () => {
    const { container, rerender } = render(
      <FocusGraphNode {...nodeProps("/icons/a?t=1")} />,
    );
    const before = icon(container);
    expect(before).toBeTruthy();
    rerender(<FocusGraphNode {...nodeProps("/icons/a?t=2")} />);
    const after = icon(container);
    expect(after).toBeTruthy();
    expect(after).not.toBe(before);
    fireEvent.error(before!);
    expect(icon(container)).toBeTruthy();
  });

  it("当前这张真的加载失败时,仍回落到文字占位", () => {
    const { container } = render(
      <FocusGraphNode {...nodeProps("/icons/broken?t=1")} />,
    );
    fireEvent.error(icon(container)!);
    expect(icon(container)).toBeNull();
  });
});
