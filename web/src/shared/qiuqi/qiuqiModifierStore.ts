"use client";

import { useEffect } from "react";
import { create } from "zustand";
import { qiuqiApi, type QiuqiModifierEntry } from "@/lib/api";

interface QiuqiStoreState {
  translations: Record<string, QiuqiModifierEntry> | null;

  count: number | null;
  loading: boolean;
  error: string | null;
  fetchTranslations: () => Promise<void>;
}

export const useQiuqiModifierStore = create<QiuqiStoreState>((set, get) => ({
  translations: null,
  count: null,
  loading: false,
  error: null,
  fetchTranslations: async () => {
    if (get().translations || get().loading) return;
    set({ loading: true, error: null });
    try {
      const data = await qiuqiApi.modifiers();

      if (!data.count) {
        console.warn(
          "[qiuqi] modifier translation table is empty (count=0) — " +
            "VSMOD_LIBRARY_ROOT is likely unset/unreadable on the backend; " +
            "中文 modifier hovers will be unavailable.",
        );
      }
      set({
        translations: data.translations,
        count: data.count,
        loading: false,
      });
    } catch (err) {
      set({
        error: err instanceof Error ? err.message : "qiuqi fetch failed",
        loading: false,
      });
    }
  },
}));

export function useQiuqiModifierTranslations(): Record<
  string,
  QiuqiModifierEntry
> | null {
  const translations = useQiuqiModifierStore((s) => s.translations);
  const fetchTranslations = useQiuqiModifierStore((s) => s.fetchTranslations);
  useEffect(() => {
    fetchTranslations();
  }, [fetchTranslations]);
  return translations;
}

export function useQiuqiLibraryStatus(): { loaded: boolean; empty: boolean } {
  const count = useQiuqiModifierStore((s) => s.count);
  return { loaded: count != null && count > 0, empty: count === 0 };
}
