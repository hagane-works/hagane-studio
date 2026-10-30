import { describe, expect, it } from "vitest";
import { createDirHandleFromFileList } from "@/lib/fileListDirHandle";
import { performScan } from "./scan";

function modFile(path: string, body = "x = {}") {
  const file = new File([body], path.split("/").pop()!);
  Object.defineProperty(file, "webkitRelativePath", {
    value: path,
    configurable: true,
  });
  return file;
}

describe("performScan technology profile source", () => {
  it("always stages the GUI and technology_tags fingerprints together", async () => {
    const scan = await performScan(
      createDirHandleFromFileList([
        modFile("Vanilla/interface/countrytechtreeview.gui"),
        modFile("Vanilla/common/technology_tags/00_technology.txt"),
        modFile("Vanilla/common/technologies/infantry.txt"),
      ]),
      "GER",
      () => {},
    );

    expect(scan.gui.files.map((entry) => entry.path)).toEqual([
      "interface/countrytechtreeview.gui",
      "common/technology_tags/00_technology.txt",
    ]);
  });
});
