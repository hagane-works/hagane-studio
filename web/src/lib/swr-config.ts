import type { SWRConfiguration } from "swr";

export const swrConfig: SWRConfiguration = {
  fetcher: (url: string) => fetch(url).then((r) => r.json()),
  dedupingInterval: 2000,
  revalidateOnFocus: false,
  errorRetryCount: 2,
};
