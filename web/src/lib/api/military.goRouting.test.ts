import { describe, expect, it, vi } from "vitest";

vi.mock("./_client", () => ({
  API_BASE: "/api",
  request: vi.fn(async (path: string) => ({ path })),
}));

import {
  doctrinesApi,
  subUnitsApi,
  equipmentApi,
  divisionTemplatesApi,
  divisionDeploymentsApi,
  miosApi,
  intelligenceAgenciesApi,
} from "./military";
import { request } from "./_client";

describe("military Go routing", () => {
  it("routes doctrine CRUD to the Go service", async () => {
    vi.mocked(request).mockClear();
    await doctrinesApi.list(182);
    await doctrinesApi.list(182, "track");
    await doctrinesApi.create(182, {
      entity_key: "x",
      entity_type: "grand_doctrine",
    });
    await doctrinesApi.update(182, 4, {});
    await doctrinesApi.remove(182, 4);
    expect(vi.mocked(request).mock.calls.map(([path]) => path)).toEqual([
      "/go/projects/182/doctrines",
      "/go/projects/182/doctrines?entity_type=track",
      "/go/projects/182/doctrines",
      "/go/projects/182/doctrines/4",
      "/go/projects/182/doctrines/4",
    ]);
  });
});

describe("sub-units Go routing", () => {
  it("routes sub-unit CRUD, preview, and text import to the Go service", async () => {
    vi.mocked(request).mockClear();
    await subUnitsApi.list(182);
    await subUnitsApi.get(182, 9);
    await subUnitsApi.create(182, {});
    await subUnitsApi.update(182, 9, {});
    await subUnitsApi.delete(182, 9);
    await subUnitsApi.preview(182, 9);
    await subUnitsApi.importText(
      182,
      "sub_units = {}",
      "common/units/demo.txt",
      "skip",
    );
    expect(vi.mocked(request).mock.calls.map(([path]) => path)).toEqual([
      "/go/projects/182/sub-units",
      "/go/projects/182/sub-units/9",
      "/go/projects/182/sub-units",
      "/go/projects/182/sub-units/9",
      "/go/projects/182/sub-units/9",
      "/go/projects/182/sub-units/preview/9",
      "/go/projects/182/sub-units/import",
    ]);
  });
});

describe("equipment Go routing", () => {
  it("routes equipment CRUD, preview, enabled-by, and text import to the Go service", async () => {
    vi.mocked(request).mockClear();
    await equipmentApi.list(182);
    await equipmentApi.get(182, 12);
    await equipmentApi.create(182, {});
    await equipmentApi.update(182, 12, {});
    await equipmentApi.delete(182, 12);
    await equipmentApi.preview(182, 12);
    await equipmentApi.enabledBy(182, "infantry equipment/1");
    await equipmentApi.importText(
      182,
      "equipments = {}",
      "common/units/equipment/demo.txt",
      "skip",
    );
    expect(vi.mocked(request).mock.calls.map(([path]) => path)).toEqual([
      "/go/projects/182/equipment",
      "/go/projects/182/equipment/12",
      "/go/projects/182/equipment",
      "/go/projects/182/equipment/12",
      "/go/projects/182/equipment/12",
      "/go/projects/182/equipment/preview/12",
      "/go/projects/182/equipment/enabled-by/infantry%20equipment%2F1",
      "/go/projects/182/equipment/import",
    ]);
  });
});

describe("division Go routing", () => {
  it("routes division template CRUD/import + deployment CRUD to the Go service", async () => {
    vi.mocked(request).mockClear();
    await divisionTemplatesApi.list(182);
    await divisionTemplatesApi.update(182, 3, {});
    await divisionTemplatesApi.preview(182, 3);
    await divisionTemplatesApi.importText(
      182,
      "GER_1936",
      "division_template = {}",
      "overwrite",
    );
    await divisionDeploymentsApi.list(182, "GER_1936");
    await divisionDeploymentsApi.update(182, 4, {});
    expect(vi.mocked(request).mock.calls.map(([path]) => path)).toEqual([
      "/go/projects/182/division-templates",
      "/go/projects/182/division-templates/3",
      "/go/projects/182/division-templates/preview/3",
      "/go/projects/182/division-templates/import",
      "/go/projects/182/division-deployments?oob_file=GER_1936",
      "/go/projects/182/division-deployments/4",
    ]);
  });
});

describe("mio + agency Go routing", () => {
  it("routes MIO and intelligence-agency CRUD to the Go service", async () => {
    vi.mocked(request).mockClear();
    await miosApi.list(182);
    await miosApi.update(182, 5, {});
    await intelligenceAgenciesApi.list(182);
    await intelligenceAgenciesApi.update(182, 6, {});
    expect(vi.mocked(request).mock.calls.map(([path]) => path)).toEqual([
      "/go/projects/182/mios",
      "/go/projects/182/mios/5",
      "/go/projects/182/intelligence-agencies",
      "/go/projects/182/intelligence-agencies/6",
    ]);
    expect(miosApi.iconUrl(182, 5)).toBe("/api/go/projects/182/mios/5/icon");
    expect(intelligenceAgenciesApi.logoUrl(182, 6)).toBe(
      "/api/go/projects/182/intelligence-agencies/6/logo",
    );
  });
});
