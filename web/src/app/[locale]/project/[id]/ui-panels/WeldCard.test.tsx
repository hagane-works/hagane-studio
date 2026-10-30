import { render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import type { UiPanel } from "@/types";
import { WeldCard } from "./WeldCard";

const mocks = vi.hoisted(() => ({
  scan: vi.fn(),
}));

vi.mock("use-intl", () => ({
  useTranslations: () => (key: string) => key,
}));

vi.mock("@/lib/api", () => ({
  panelContractApi: { scan: mocks.scan },
  uiPanelsApi: { files: vi.fn().mockResolvedValue(null) },
}));

const panel: UiPanel = {
  id: 1,
  project_id: 7,
  token: "tst_panel",
  country_tag: "TST",
  name_en: "Panel",
  name_zh: "",
  display_name_locs: {},
  context_type: "",
  parent_window_token: "",
  visible_trigger: "",
  window_props: {},
  controls: [],
  raw_extras: {},
  source: "created",
  created_at: "",
  updated_at: "1",
};

describe("WeldCard", () => {
  it("shows the core check and says what is not checked", async () => {
    mocks.scan.mockResolvedValue({
      findings: [],
      totals: { error: 0, warning: 0, info: 0 },
      count: 0,
    });

    render(<WeldCard projectId={7} panel={panel} controls={[]} />);

    await waitFor(() => expect(screen.getByText("catWindow")).toBeTruthy());
    expect(mocks.scan).toHaveBeenCalledWith(7);
    expect(screen.getByText("notCheckedNote")).toBeTruthy();
    expect(screen.queryByText("scanFailed")).toBeNull();
  });
});
