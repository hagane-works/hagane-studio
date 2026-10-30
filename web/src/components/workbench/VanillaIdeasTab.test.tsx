import { cleanup, render, screen } from "@testing-library/react";
import { IntlProvider } from "use-intl";
import { afterEach, describe, expect, it, vi } from "vitest";
import vanillaBrowser from "../../../messages/en/vanillaBrowser.json";

const { list } = vi.hoisted(() => ({ list: vi.fn() }));

vi.mock("@/lib/api", () => ({
  API_BASE: "/api",
  vanillaIdeasApi: { list },
}));

import { VanillaIdeasTab } from "./VanillaIdeasTab";

afterEach(cleanup);

describe("VanillaIdeasTab thumbnails", () => {
  it("asks for the sprite an idea's picture names (GFX_idea_<picture>)", async () => {
    list.mockResolvedValue({
      total_matching: 2,
      limit: 50,
      offset: 0,
      items: [
        {
          category: "country",
          idea_id: "JAP_defense",
          picture: "JAP_national_defense_state",
          file: "japan.txt",
          line: 3,
        },
        {
          category: "country",
          idea_id: "already_a_sprite",
          picture: "GFX_idea_isolation",
          file: "generic.txt",
          line: 9,
        },
      ],
      meta: { total: 2, categories: 1 },
    });
    render(
      <IntlProvider locale="en" messages={{ vanillaBrowser }} timeZone="UTC">
        <VanillaIdeasTab />
      </IntlProvider>,
    );
    const first = await screen.findByAltText("JAP_national_defense_state");
    expect(first.getAttribute("src")).toBe(
      "/api/go/vanilla/focus-icon/GFX_idea_JAP_national_defense_state",
    );

    expect(screen.getByAltText("GFX_idea_isolation").getAttribute("src")).toBe(
      "/api/go/vanilla/focus-icon/GFX_idea_isolation",
    );
  });
});
