import React from "react";
import { cleanup, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";

import type { CountryRow } from "@/lib/api";

vi.mock("@/lib/api", () => ({
  countriesApi: {
    flagPreviewUrl: (projectId: number, countryId: number) =>
      `/flag/${projectId}/${countryId}`,
  },
}));

import { CountryFlagPreview } from "./CountryFlagPreview";

afterEach(cleanup);

describe("CountryFlagPreview", () => {
  it("keeps project flags at HOI4's native 82x52 aspect", () => {
    render(
      <CountryFlagPreview
        projectId={76}
        country={
          {
            id: 9,
            project_id: 76,
            tag: "KOR",
            name: "Korea",
            ideology: "neutrality",
            flag_image_path: "flags/KOR.tga",
          } as CountryRow
        }
        roots={[]}
        baseModDepKey={null}
        className="w-full rounded-md"
      />,
    );

    const flag = screen.getByRole("img", { name: "KOR default flag" });
    expect(flag.getAttribute("width")).toBe("82");
    expect(flag.getAttribute("height")).toBe("52");
    expect(flag.className).toContain("aspect-[82/52]");
  });
});
