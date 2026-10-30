import { describe, expect, it, vi } from "vitest";

vi.mock("./_client", () => ({
  API_BASE: "/api",
  request: vi.fn(async (path: string) => ({ path })),
}));

import { oobUnitsApi } from "./oobUnits";
import { request } from "./_client";

describe("oob-units Go routing", () => {
  it("routes snapshot, preview, import, and entity CRUD to Go", async () => {
    vi.mocked(request).mockClear();
    await oobUnitsApi.snapshot(182);
    await oobUnitsApi.airWingGroups.create(182, {
      location: "1234",
      oob_file: "",
      source_file: "",
      country_tag: "",
      ordering: 0,
    });
    await oobUnitsApi.airWings.update(182, 2, {});
    await oobUnitsApi.fleets.remove(182, 3);
    await oobUnitsApi.taskForces.create(182, {
      fleet_id: 3,
      name: "",
      location: "",
      ordering: 0,
    });
    await oobUnitsApi.ships.update(182, 5, {});
    await oobUnitsApi.preview(182);
    await oobUnitsApi.importText(182, {
      text: "units = {}",
      oob_file: "GER_1936",
      source_file: "history/units/GER_1936.txt",
      country_tag: "GER",
      policy: "overwrite",
    });
    expect(vi.mocked(request).mock.calls.map(([path]) => path)).toEqual([
      "/go/projects/182/oob-units/snapshot",
      "/go/projects/182/oob-units/air-wing-groups",
      "/go/projects/182/oob-units/air-wings/2",
      "/go/projects/182/oob-units/fleets/3",
      "/go/projects/182/oob-units/task-forces",
      "/go/projects/182/oob-units/ships/5",
      "/go/projects/182/oob-units/preview",
      "/go/projects/182/oob-units/import",
    ]);
  });
});
