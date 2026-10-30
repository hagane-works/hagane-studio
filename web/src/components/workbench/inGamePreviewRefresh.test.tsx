import React from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, fireEvent, render } from "@testing-library/react";

vi.mock("use-intl", () => ({ useTranslations: () => (key: string) => key }));
vi.mock("@/lib/api", () => ({
  API_BASE: "",
  decisionsApi: {
    iconUrl: (p: number, id: number) =>
      `/go/projects/${p}/decisions/${id}/icon`,
    categoryIconUrl: (p: number, id: number) =>
      `/go/projects/${p}/decisions/categories/${id}/icon`,
  },
  ideasApi: {
    pictureUrl: (p: number, id: number) =>
      `/go/projects/${p}/ideas/${id}/picture`,
  },
}));
const baseMod = vi.hoisted(() => ({ depKeys: [] as string[] }));
vi.mock("@/lib/api/basemod", () => ({
  useBaseModDepKeys: () => baseMod.depKeys,
  artUrl: () => "",
  gfxUrl: (projectId: number, depKey: string, name: string) =>
    `/go/projects/${projectId}/basemod/${depKey}/gfx/${name}`,
}));
vi.mock("@/components/CountryTagPreview", () => ({
  CountryTagPreview: ({ text }: { text: string }) => <span>{text}</span>,
}));

import {
  bustAssetPreview,
  useAssetPreviewVersionStore,
} from "@/lib/store/assetPreviewVersionStore";
import { IdeaInGamePreview } from "./IdeaInGamePreview";

beforeEach(() => {
  useAssetPreviewVersionStore.setState({ versions: {} });
  baseMod.depKeys = [];
});
afterEach(cleanup);

const idea = {
  id: 7,
  name: "Test Idea",
  picture: "GFX_idea_test",
  idea_type: "country",
} as never;
const pictureUrl = "/go/projects/1/ideas/7/picture";
const img = (root: HTMLElement) => root.querySelector("img");

describe("实时预览 · 换图后刷新", () => {
  it("换图后按版本号重新取图", () => {
    const { container, rerender } = render(
      <IdeaInGamePreview idea={idea} projectId={1} />,
    );
    expect(img(container)!.getAttribute("src")).toBe(pictureUrl);
    bustAssetPreview(pictureUrl);
    rerender(<IdeaInGamePreview idea={idea} projectId={1} />);
    expect(img(container)!.getAttribute("src")).not.toBe(pictureUrl);
  });

  it("这一张加载失败后,换图仍能恢复(不会永久空白)", () => {
    const { container, rerender } = render(
      <IdeaInGamePreview idea={idea} projectId={1} />,
    );
    const before = img(container)!;
    fireEvent.error(before);
    bustAssetPreview(pictureUrl);
    rerender(<IdeaInGamePreview idea={idea} projectId={1} />);
    const after = img(container)!;
    expect(after).not.toBe(before);
    expect(after.style.display).not.toBe("none");
  });

  it("项目里没有这张图时,接着取基础 mod 定义的图(裸名即 GFX_idea_x)", () => {
    baseMod.depKeys = ["the_road_to_56"];
    const bare = {
      id: 7,
      name: "Test Idea",
      picture: "r56_production",
      idea_type: "country",
    } as never;
    const { container } = render(
      <IdeaInGamePreview idea={bare} projectId={1} />,
    );
    fireEvent.error(img(container)!);
    expect(img(container)!.getAttribute("src")).toBe(
      "/go/projects/1/basemod/the_road_to_56/gfx/GFX_idea_r56_production",
    );
    fireEvent.error(img(container)!);
    expect(img(container)).toBeNull();
  });
});
