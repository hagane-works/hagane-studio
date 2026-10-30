import useSWR from "swr";
import { historyFilesApi, type HistoryFileRow } from "@/lib/api/scenario";

export function withChosenIdea(
  extra: readonly string[],
  group: readonly string[],
  chosen: string,
): string[] {
  const others = new Set(group);
  const out = extra.filter((token) => !others.has(token) && token !== chosen);

  const at = extra.findIndex((token) => others.has(token) || token === chosen);
  out.splice(at < 0 ? out.length : Math.min(at, out.length), 0, chosen);
  return out;
}

export function withAdvisor(
  extra: readonly string[],
  idea: string,
  hire: boolean,
): string[] {
  const out = extra.filter((token) => token !== idea);
  if (hire) out.push(idea);
  return out;
}

export function useHistoryRow(
  projectId: number,
  tag: string | null,
  enabled: boolean,
) {
  return useSWR<HistoryFileRow | null>(
    enabled && tag ? ["game-view-history-row", projectId, tag] : null,
    async () => {
      const rows = await historyFilesApi.list(projectId);
      return (
        rows.find(
          (row) =>
            row.country_tag.toUpperCase() === (tag as string).toUpperCase(),
        ) ?? null
      );
    },
    { revalidateOnFocus: false },
  );
}
