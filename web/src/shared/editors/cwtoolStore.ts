"use client";

import { useEffect } from "react";
import { create } from "zustand";
import { cwtoolApi, type CwtoolRules } from "@/lib/api";

interface CwtoolStoreState {
  rules: CwtoolRules | null;
  loading: boolean;
  error: string | null;
  fetchRules: () => Promise<void>;
}

export const useCwtoolStore = create<CwtoolStoreState>((set, get) => ({
  rules: null,
  loading: false,
  error: null,
  fetchRules: async () => {
    if (get().rules || get().loading) return;
    set({ loading: true, error: null });
    try {
      const data = await cwtoolApi.whitelist();
      set({ rules: data, loading: false });
    } catch (err) {
      set({
        error: err instanceof Error ? err.message : "cwtool fetch failed",
        loading: false,
      });
    }
  },
}));

export function useCwtoolRules(): CwtoolRules | null {
  const rules = useCwtoolStore((s) => s.rules);
  const fetchRules = useCwtoolStore((s) => s.fetchRules);
  useEffect(() => {
    fetchRules();
  }, [fetchRules]);
  return rules;
}
