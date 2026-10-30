"use client";

import { useCallback, useRef } from "react";
import type { FocusEvent, KeyboardEvent } from "react";

const INTERACTIVE =
  "input, button, select, textarea, a[href], [contenteditable]";

export function useListboxNav<T extends HTMLElement = HTMLDivElement>() {
  const containerRef = useRef<T | null>(null);

  const getOptions = useCallback((): HTMLElement[] => {
    const root = containerRef.current;
    return root
      ? Array.from(root.querySelectorAll<HTMLElement>('[role="option"]'))
      : [];
  }, []);

  const focusOption = useCallback((el: HTMLElement | undefined) => {
    if (!el) return;
    el.focus();
    el.scrollIntoView({ block: "nearest" });
  }, []);

  const onFocus = useCallback(
    (e: FocusEvent<T>) => {
      if (e.target !== containerRef.current) return;
      const opts = getOptions();
      focusOption(
        opts.find((o) => o.getAttribute("aria-selected") === "true") ?? opts[0],
      );
    },
    [getOptions, focusOption],
  );

  const onKeyDown = useCallback(
    (e: KeyboardEvent<T>) => {
      const opts = getOptions();
      if (opts.length === 0) return;
      const target = e.target as HTMLElement;
      const row = target.closest<HTMLElement>('[role="option"]');
      const inNestedControl = target !== row && target.matches(INTERACTIVE);

      switch (e.key) {
        case "ArrowDown":
        case "ArrowUp": {
          if (inNestedControl && target.matches("input, textarea, select"))
            return;
          e.preventDefault();
          const i = row ? opts.indexOf(row) : -1;
          const next =
            e.key === "ArrowDown"
              ? Math.min(i + 1, opts.length - 1)
              : Math.max(i - 1, 0);
          focusOption(opts[next]);
          break;
        }
        case "Home":
        case "End": {
          if (inNestedControl) return;
          e.preventDefault();
          focusOption(e.key === "Home" ? opts[0] : opts[opts.length - 1]);
          break;
        }
        case "Enter":
        case " ": {
          if (inNestedControl) return;
          if (row) {
            e.preventDefault();
            row.click();
          }
          break;
        }
      }
    },
    [getOptions, focusOption],
  );

  return {
    containerProps: {
      ref: containerRef,
      role: "listbox" as const,
      tabIndex: 0,
      onFocus,
      onKeyDown,
    },

    optionProps: (selected: boolean) => ({
      role: "option" as const,
      "aria-selected": selected,
      tabIndex: -1,
    }),
  };
}
