"use client";

import { useEffect, useRef, type ComponentProps } from "react";
import { Input } from "@/components/ui/input";

type BookmarkLabelInputProps = Omit<
  ComponentProps<typeof Input>,
  "defaultValue" | "onChange"
> & {
  value: string;
  onTokensChange: (tokens: string[]) => void;
};

export function BookmarkLabelInput({
  value,
  onTokensChange,
  ...props
}: BookmarkLabelInputProps) {
  const inputRef = useRef<HTMLInputElement>(null);
  const localNormalizedRef = useRef(value);

  useEffect(() => {
    if (value === localNormalizedRef.current) return;

    if (inputRef.current) inputRef.current.value = value;
    localNormalizedRef.current = value;
  }, [value]);

  return (
    <Input
      {...props}
      ref={inputRef}
      defaultValue={value}
      onChange={(event) => {
        const tokens = event.target.value.split(/[\s,]+/).filter(Boolean);
        localNormalizedRef.current = tokens.join(" ");
        onTokensChange(tokens);
      }}
    />
  );
}
