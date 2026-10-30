"use client";

import { useLinkStatus } from "@/platform/link";

export function SidebarNavPending() {
  const { pending } = useLinkStatus();
  return (
    <span
      aria-hidden
      className={`ml-auto h-1.5 w-1.5 shrink-0 rounded-full bg-accent transition-opacity ${
        pending ? "opacity-100 animate-pulse" : "opacity-0"
      }`}
    />
  );
}
