import { afterEach, describe, expect, it, vi } from "vitest";
import { exportApi, existingModName, isNotAModFolderError } from "./export";
import { useLocaleStore } from "@/lib/store/localeStore";

describe("advanced exports take a zip or a mod folder", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
    useLocaleStore.setState({ locale: "en" });
  });

  function stubFetch(response: Response) {
    const fetchMock = vi.fn().mockResolvedValue(response);
    vi.stubGlobal("fetch", fetchMock);
    return fetchMock;
  }

  function zipResponse(): Response {
    return new Response(new Blob(["zip"]), {
      status: 200,
      headers: {
        "Content-Type": "application/zip",
        "X-Merge-Conflict-Count": "2",
      },
    });
  }

  it("sends a chosen folder as base_mod_folder, with the interface language", async () => {
    useLocaleStore.setState({ locale: "ja" });
    const fetchMock = stubFetch(zipResponse());
    const folder =
      "D:\\SteamLibrary\\steamapps\\workshop\\content\\394360\\2318957791";
    const result = await exportApi.appendToExisting(7, { folder });
    const form = fetchMock.mock.calls[0][1].body as FormData;
    expect(form.get("base_mod_folder")).toBe(folder);
    expect(form.get("base_mod_file")).toBeNull();
    expect(form.get("base_mod_filename")).toBe("2318957791");
    expect(form.get("locale")).toBe("ja");
    expect(result).toMatchObject({
      conflictCount: 2,
      filename: "appended_2318957791",
    });
  });

  it("still uploads a picked zip as base_mod_file", async () => {
    const fetchMock = stubFetch(zipResponse());
    const zip = new File(["zip"], "base.zip", { type: "application/zip" });
    await exportApi.diffExportZip(7, zip, {}, {}, true);
    const form = fetchMock.mock.calls[0][1].body as FormData;
    expect(form.get("base_mod_file")).toBeInstanceOf(File);
    expect(form.get("base_mod_folder")).toBeNull();
    expect(form.get("locale")).toBe("en");
  });

  it("names the download as the program does, quoted or not", async () => {
    for (const [disposition, filename] of [
      [
        "attachment; filename=merged_Downsize_AirForce.zip",
        "merged_Downsize_AirForce.zip",
      ],
      [
        'attachment; filename="Live Append_appended.zip"',
        "Live Append_appended.zip",
      ],
      [
        "attachment; filename*=utf-8''%E8%BF%BD%E5%8A%A0_appended.zip",
        "追加_appended.zip",
      ],
    ]) {
      stubFetch(
        new Response(new Blob(["zip"]), {
          status: 200,
          headers: { "Content-Disposition": disposition },
        }),
      );
      const result = await exportApi.diffExportZip(
        7,
        { folder: "D:\\mods\\2318957791" },
        {},
        {},
        true,
      );
      expect(result.filename).toBe(filename);
    }
  });

  it("recognises a folder the program refused as not a mod", async () => {
    stubFetch(
      new Response(
        JSON.stringify({
          detail: { error: "not_a_mod", message: "no descriptor.mod" },
        }),
        { status: 422, headers: { "content-type": "application/json" } },
      ),
    );
    const error = await exportApi
      .diffPreview(7, { folder: "C:\\Pictures" })
      .catch((e: unknown) => e);
    expect(isNotAModFolderError(error)).toBe(true);
    expect(isNotAModFolderError(new Error("no descriptor.mod"))).toBe(false);
  });

  it("names a folder by its last path segment", () => {
    expect(existingModName({ folder: "C:\\mods\\Base Mod\\" })).toBe(
      "Base Mod",
    );
    expect(existingModName({ folder: "/home/me/mods/base" })).toBe("base");
  });
});
