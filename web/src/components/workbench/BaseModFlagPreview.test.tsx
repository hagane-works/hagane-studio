import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";

import { BaseModFlagPreview, baseModFlagUrl } from "./BaseModFlagPreview";

afterEach(cleanup);

describe("BaseModFlagPreview dynamic ROOT contract", () => {
  it("keeps a fifth project ROOT in the resolver URL", () => {
    expect(
      baseModFlagUrl(4, "tno", "GER", "military_government", "medium", [
        "democratic",
        "military_government",
      ]),
    ).toBe(
      "/api/go/projects/4/basemod/tno/flag/GER?ideology=military_government&size=medium",
    );
  });

  it("does not mistake a subtype for a ROOT", () => {
    expect(
      baseModFlagUrl(4, "tno", "GER", "national_military_government", "large", [
        "military_government",
      ]),
    ).toBe("/api/go/projects/4/basemod/tno/flag/GER");
  });

  it("falls back from a fifth-ROOT candidate to the generic flag, then placeholder", () => {
    render(
      <BaseModFlagPreview
        projectId={4}
        depKey="tno"
        tag="GER"
        ideology="military_government"
        effectiveRootIds={["democratic", "military_government"]}
        placeholder={<span>generic missing</span>}
      />,
    );

    const first = screen.getByRole("img") as HTMLImageElement;
    expect(first.src).toContain("ideology=military_government");
    fireEvent.error(first);
    const generic = screen.getByRole("img") as HTMLImageElement;
    expect(generic.src).not.toContain("ideology=");
    fireEvent.error(generic);
    expect(screen.getByText("generic missing")).toBeTruthy();
  });
});
