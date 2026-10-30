"use client";

import { SWRConfig } from "swr";
import type { ReactNode } from "react";
import { swrConfig } from "@/lib/swr-config";

export function SWRProvider({ children }: { children: ReactNode }) {
  return <SWRConfig value={swrConfig}>{children}</SWRConfig>;
}
