import { createElement, useState } from "react";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import {
  createDirHandleFromPreparedFileList,
  prepareFileListImport,
} from "@/lib/fileListDirHandle";
import { performScan } from "./scan";

function modFile(path: string, content = "x"): File {
  const name = path.split("/").at(-1) ?? path;
  const file = new File([content], name);
  Object.defineProperty(file, "webkitRelativePath", {
    configurable: true,
    value: path,
  });
  return file;
}

function ScanFeedbackHarness({ root }: { root: FileSystemDirectoryHandle }) {
  const [stage, setStage] = useState("idle");
  const [completed, setCompleted] = useState(false);

  const scan = async () => {
    await performScan(root, "GER", setStage, (key) => key);
    setCompleted(true);
  };

  return createElement(
    "div",
    null,
    createElement(
      "button",
      { type: "button", onClick: () => void scan() },
      "start",
    ),
    createElement("output", { "data-testid": "stage" }, stage),
    createElement(
      "output",
      { "data-testid": "completion" },
      completed ? "complete" : "pending",
    ),
  );
}

describe("local scan responsiveness contract", () => {
  it("paints a real stage and runs a macrotask before completing, while preserving results", async () => {
    const characterPaths = [
      "BigMod/common/characters/GER_z.txt",
      "BigMod/common/characters/nested/GER_nested.txt",
      "BigMod/common/characters/GER_a.txt",
    ];
    const eventPaths = [
      "BigMod/events/GER_second.txt",
      "BigMod/events/nested/GER_nested.txt",
      "BigMod/events/GER_first.txt",
    ];
    const localisationPaths = Array.from({ length: 700 }, (_, i) => {
      const id = (i * 37) % 700;
      return `BigMod/localisation/english/pack_${String(id).padStart(4, "0")}_l_english.yml`;
    });
    const statePaths = [
      "BigMod/history/states/9-Zeta.txt",
      "BigMod/history/states/1-Alpha.txt",
    ];
    const interfacePaths = [
      "BigMod/interface/z.gui",
      "BigMod/interface/a.gfx",
      "BigMod/interface/nested/b.gfx",
      "BigMod/interface/nested/deeper/d.gui",
    ];
    const relevantPaths = [
      ...characterPaths,
      ...eventPaths,
      ...localisationPaths,
      ...statePaths,
      ...interfacePaths,
      "BigMod/descriptor.mod",
    ];
    const pickedFiles = [
      ...relevantPaths.map((path) => modFile(path)),

      modFile("BigMod/music/ignored.ogg"),
    ];

    const prepared = await prepareFileListImport(pickedFiles);
    expect(prepared.relevantCount).toBe(relevantPaths.length);
    const root = await createDirHandleFromPreparedFileList(prepared);

    const lifecycle: string[] = [];
    let completed = false;
    let heartbeatSawCompleted = true;
    const heartbeat = new Promise<void>((resolve) => {
      setTimeout(() => {
        heartbeatSawCompleted = completed;
        lifecycle.push("heartbeat");
        resolve();
      }, 0);
    });

    const result = await performScan(
      root,
      "GER",
      (message) => lifecycle.push(`status:${message}`),
      (key) => key,
    ).then((scanResult) => {
      completed = true;
      lifecycle.push("complete");
      return scanResult;
    });
    await heartbeat;

    const firstStatus = lifecycle.indexOf("status:scanCharacters");
    const heartbeatIndex = lifecycle.indexOf("heartbeat");
    const completionIndex = lifecycle.indexOf("complete");
    expect(firstStatus).toBeGreaterThanOrEqual(0);
    expect(firstStatus).toBeLessThan(heartbeatIndex);
    expect(heartbeatIndex).toBeLessThan(completionIndex);
    expect(heartbeatSawCompleted).toBe(false);
    expect(lifecycle).toContain("status:scanLocalisation");
    expect(lifecycle).toContain("status:scanGfx");

    expect(result.characters.fileCount).toBe(characterPaths.length);
    expect(result.characters.files.map((entry) => entry.path)).toEqual([
      "common/characters/GER_z.txt",
      "common/characters/nested/GER_nested.txt",
      "common/characters/GER_a.txt",
    ]);
    expect(result.events.fileCount).toBe(eventPaths.length);
    expect(result.events.files.map((entry) => entry.path)).toEqual([
      "events/GER_second.txt",
      "events/nested/GER_nested.txt",
      "events/GER_first.txt",
    ]);
    expect(result.loc.fileCount).toBe(localisationPaths.length);
    expect(result.loc.files.map((entry) => entry.path)).toEqual(
      localisationPaths.map((path) => path.slice("BigMod/".length)),
    );
    expect(result.states.fileCount).toBe(statePaths.length);
    expect(result.states.files.map((entry) => entry.path)).toEqual([
      "history/states/9-Zeta.txt",
      "history/states/1-Alpha.txt",
    ]);
    expect(result.gfx.files.map((entry) => entry.path)).toEqual([
      "interface/a.gfx",
      "interface/nested/b.gfx",
    ]);
    expect(result.gui.files.map((entry) => entry.path)).toEqual([
      "interface/z.gui",
      "interface/nested/deeper/d.gui",
    ]);
    expect(result.descriptor?.path).toBe("descriptor.mod");
  });

  it("commits stage feedback to React before the scan completes", async () => {
    const prepared = await prepareFileListImport([
      modFile("FeedbackMod/common/characters/GER.txt"),
    ]);
    const root = await createDirHandleFromPreparedFileList(prepared);

    render(createElement(ScanFeedbackHarness, { root }));
    fireEvent.click(screen.getByRole("button", { name: "start" }));

    expect(screen.getByTestId("stage").textContent).toBe("scanCharacters");
    expect(screen.getByTestId("completion").textContent).toBe("pending");

    await waitFor(() => {
      expect(screen.getByTestId("completion").textContent).toBe("complete");
    });
  });
});
