export async function uploadInWindows<B>(
  batches: B[],
  inflight: number,
  run: (batch: B) => Promise<void>,
  isCancelled: () => boolean,
): Promise<{ failed: B[]; firstError: unknown }> {
  const failed: B[] = [];
  let firstError: unknown;
  for (let i = 0; i < batches.length; i += inflight) {
    await Promise.all(
      batches.slice(i, i + inflight).map(async (batch) => {
        try {
          await run(batch);
        } catch (err) {
          if (isCancelled()) throw err;
          if (failed.length === 0) firstError = err;
          failed.push(batch);
        }
      }),
    );
  }
  return { failed, firstError };
}
