import { describe, expect, it } from "vitest";
import { uploadInWindows } from "./uploadInWindows";

describe("uploadInWindows", () => {
  it("keeps uploading after a batch fails and reports the failed ones", async () => {
    const sent: number[] = [];
    const { failed, firstError } = await uploadInWindows(
      [1, 2, 3, 4, 5, 6, 7, 8],
      2,
      async (batch) => {
        sent.push(batch);
        if (batch === 6)
          throw new Error(
            "ON CONFLICT DO UPDATE command cannot affect row a second time",
          );
      },
      () => false,
    );
    expect(sent.sort((a, b) => a - b)).toEqual([1, 2, 3, 4, 5, 6, 7, 8]);
    expect(failed).toEqual([6]);
    expect((firstError as Error).message).toContain("ON CONFLICT");
  });

  it("never has more than the allowed number of batches in flight", async () => {
    let open = 0;
    let most = 0;
    await uploadInWindows(
      [1, 2, 3, 4, 5],
      2,
      async () => {
        open++;
        most = Math.max(most, open);
        await new Promise((resolve) => setTimeout(resolve, 1));
        open--;
      },
      () => false,
    );
    expect(most).toBe(2);
  });

  it("stops with the error once the user has cancelled", async () => {
    let cancelled = false;
    const sent: number[] = [];
    await expect(
      uploadInWindows(
        [1, 2, 3, 4],
        1,
        async (batch) => {
          sent.push(batch);
          if (batch === 2) {
            cancelled = true;
            throw new DOMException("cancelled", "AbortError");
          }
        },
        () => cancelled,
      ),
    ).rejects.toThrow("cancelled");
    expect(sent).toEqual([1, 2]);
  });
});
