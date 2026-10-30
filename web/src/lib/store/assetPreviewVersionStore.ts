import { create } from "zustand";

interface AssetPreviewVersionState {
  versions: Record<string, number>;
  bump: (url: string) => void;
}

export const useAssetPreviewVersionStore = create<AssetPreviewVersionState>(
  (set) => ({
    versions: {},
    bump: (url) =>
      set((state) =>
        url
          ? {
              versions: {
                ...state.versions,
                [url]: (state.versions[url] ?? 0) + 1,
              },
            }
          : state,
      ),
  }),
);

export function bustAssetPreview(url: string | null | undefined): void {
  if (url) useAssetPreviewVersionStore.getState().bump(url);
}

export function useAssetPreviewUrl(
  url: string | null | undefined,
): string | null {
  const version = useAssetPreviewVersionStore((state) =>
    url ? (state.versions[url] ?? 0) : 0,
  );
  if (!url) return null;
  return version ? `${url}${url.includes("?") ? "&" : "?"}t=${version}` : url;
}
