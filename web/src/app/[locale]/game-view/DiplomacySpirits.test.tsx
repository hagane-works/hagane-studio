import React from "react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, fireEvent, render } from "@testing-library/react";
import type { GuiPreviewIcon } from "@/lib/guiRenderApi";

const shown: string[] = [];
vi.mock("../gui-renderer/GameTooltip", () => ({
  gameTooltipProps: (tooltip: string) => ({
    onPointerEnter: () => shown.push(tooltip),
  }),
}));

import { DiplomacySpirits } from "./DiplomacyLayer";

const icon = (token: string, tooltip?: string): GuiPreviewIcon => ({
  token,
  label: `${token} label`,
  tooltip,
  icon_sprite: {
    resolved: true,
    frames: 1,
    width: 64,
    height: 64,
    effect: null,
    effect_kind: null,
    cornered: false,
    bordersize: null,
    vanilla_url: `https://example.test/${token}.png`,
  },
});

afterEach(() => {
  cleanup();
  shown.length = 0;
});

describe("the diplomacy screen's national spirits", () => {
  it("show the game's hover text — the spirit's numbers — under the pointer", () => {
    const { container } = render(
      <DiplomacySpirits
        grid={{ x: 10, y: 20 }}
        spirits={[icon("GER_hitler", "§HHitler§!\nPolitical Power: §G+10%§!")]}
      />,
    );
    const spirit = container.querySelector<HTMLElement>(
      '[data-spirit="GER_hitler"]',
    )!;

    expect(spirit.style.pointerEvents).toBe("auto");
    expect(spirit.getAttribute("title")).toBeNull();
    fireEvent.pointerEnter(spirit);
    expect(shown).toEqual(["§HHitler§!\nPolitical Power: §G+10%§!"]);
  });

  it("falls back to the name when there is no hover text, and keeps the game's six places", () => {
    const spirits = [
      icon("plain"),
      ...Array.from({ length: 6 }, (_, k) => icon(`s${k}`, "x")),
    ];
    const { container } = render(
      <DiplomacySpirits grid={{ x: 0, y: 0 }} spirits={spirits} />,
    );
    const plain = container.querySelector<HTMLElement>(
      '[data-spirit="plain"]',
    )!;
    expect(plain.getAttribute("title")).toBe("plain label");
    expect(plain.style.pointerEvents).toBe("auto");
    expect(container.querySelectorAll("[data-spirit]")).toHaveLength(6);
  });
});
