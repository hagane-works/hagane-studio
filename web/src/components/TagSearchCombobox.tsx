"use client";

import { useState, useEffect, useRef } from "react";
import { useTranslations } from "use-intl";
import {
  Command,
  CommandInput,
  CommandList,
  CommandEmpty,
  CommandItem,
} from "@/components/ui/command";
import { Button } from "@/components/ui/button";
import { tagsApi } from "@/lib/api";
import type { CountryTag } from "@/types";

interface TagSearchComboboxProps {
  value: CountryTag | null;
  onChange: (tag: CountryTag) => void;
}

export function TagSearchCombobox({ value, onChange }: TagSearchComboboxProps) {
  const t = useTranslations("tagSearchCombobox");
  const tCommon = useTranslations("common");
  const [open, setOpen] = useState(false);
  const [query, setQuery] = useState("");

  const [tags, setTags] = useState<CountryTag[] | null>(null);
  const ref = useRef<HTMLDivElement>(null);

  useEffect(() => {
    let cancelled = false;
    const t = setTimeout(() => {
      tagsApi.search(query).then((res) => {
        if (!cancelled) setTags(res.tags);
      });
    }, 250);
    return () => {
      cancelled = true;
      clearTimeout(t);
    };
  }, [query]);

  useEffect(() => {
    function handleClickOutside(e: MouseEvent) {
      if (ref.current && !ref.current.contains(e.target as Node)) {
        setOpen(false);
      }
    }
    if (open) document.addEventListener("mousedown", handleClickOutside);
    return () => document.removeEventListener("mousedown", handleClickOutside);
  }, [open]);

  return (
    <div ref={ref} className="relative">
      <Button
        variant="outline"
        aria-haspopup="listbox"
        aria-expanded={open}
        className={
          "w-full h-9 justify-start rounded-lg border-input bg-card dark:bg-card aria-expanded:bg-card text-left font-normal " +
          (value ? "" : "text-muted-foreground")
        }
        onClick={() => setOpen(!open)}
      >
        {value ? `${value.tag} - ${value.name}` : t("trigger")}
      </Button>
      {open && (
        <div className="absolute top-full left-0 z-50 mt-1 w-full rounded-lg border border-border bg-popover shadow-md">
          <Command shouldFilter={false}>
            <CommandInput
              placeholder={t("searchPlaceholder")}
              value={query}
              onValueChange={setQuery}
            />
            <CommandList className="mt-2">
              <CommandEmpty>
                {tags ? t("empty") : tCommon("loading")}
              </CommandEmpty>
              {(tags ?? []).slice(0, 50).map((tag) => (
                <CommandItem
                  key={tag.tag}
                  value={tag.tag}
                  onSelect={() => {
                    onChange(tag);
                    setOpen(false);
                  }}
                >
                  <span className="font-mono font-semibold mr-2">
                    {tag.tag}
                  </span>
                  <span className="text-muted-foreground">{tag.name}</span>
                </CommandItem>
              ))}
            </CommandList>
          </Command>
        </div>
      )}
    </div>
  );
}
