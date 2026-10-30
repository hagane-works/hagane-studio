"use client";

import { useState, type ReactNode } from "react";
import { Menu } from "lucide-react";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import styles from "./SiteHeader.module.css";

export function SiteHeaderNavigation({
  label,
  children,
}: {
  label: string;
  children: ReactNode;
}) {
  const [open, setOpen] = useState(false);
  return (
    <>
      <nav aria-label={label} className={styles.nav}>
        {children}
      </nav>
      <div className={styles.compactNav}>
        <Popover open={open} onOpenChange={setOpen}>
          <PopoverTrigger
            aria-label={label}
            title={label}
            className="ui-interactive inline-flex size-11 items-center justify-center rounded-lg text-muted-foreground hover:bg-muted hover:text-foreground"
          >
            <Menu className="size-5" aria-hidden />
          </PopoverTrigger>
          <PopoverContent
            align="end"
            className="public-site max-h-(--available-height) max-w-[calc(100vw-2rem)] overflow-y-auto p-3"
          >
            <nav
              aria-label={label}
              className={styles.compactLinks}
              onClick={(event) => {
                if (
                  event.target instanceof Element &&
                  event.target.closest("a")
                )
                  setOpen(false);
              }}
            >
              {children}
            </nav>
          </PopoverContent>
        </Popover>
      </div>
    </>
  );
}
