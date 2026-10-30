import { beforeEach, describe, expect, it, vi } from "vitest";

import { charactersApi } from "@/lib/api";
import { useCharacterStore } from "./characterStore";

vi.mock("@/lib/api", () => ({
  charactersApi: { list: vi.fn() },
}));

describe("characterStore list refresh", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    useCharacterStore.setState({ characters: [], loading: false, error: null });
  });

  it("does not let a pre-import request overwrite the completed import refresh", async () => {
    let finishOld!: (value: unknown[]) => void;
    let finishNew!: (value: unknown[]) => void;
    const oldRequest = new Promise<unknown[]>((resolve) => {
      finishOld = resolve;
    });
    const newRequest = new Promise<unknown[]>((resolve) => {
      finishNew = resolve;
    });
    vi.mocked(charactersApi.list)
      .mockReturnValueOnce(oldRequest as ReturnType<typeof charactersApi.list>)
      .mockReturnValueOnce(newRequest as ReturnType<typeof charactersApi.list>);

    const first = useCharacterStore.getState().fetchCharacters(182);
    const refreshAfterImport = useCharacterStore
      .getState()
      .fetchCharacters(182);
    const imported = [{ id: 1, country_tag: "ENG", name: "Churchill" }];

    finishNew(imported);
    await refreshAfterImport;
    finishOld([]);
    await first;

    expect(useCharacterStore.getState().characters).toEqual(imported);
    expect(useCharacterStore.getState().loading).toBe(false);
  });
});
