"use client";

import { useTranslations } from "use-intl";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";

export const CUSTOM_TAG_REGEX = /^[A-Z][A-Z0-9]{2,4}$/;

interface CustomTagInputProps {
  value: string;
  onChange: (value: string) => void;

  label?: string;

  id?: string;
  disabled?: boolean;
}

export function CustomTagInput({
  value,
  onChange,
  label = "TAG",
  id = "custom-tag-input",
  disabled = false,
}: CustomTagInputProps) {
  const t = useTranslations("customTagInput");
  const sanitize = (raw: string) =>
    raw
      .replace(/[^A-Za-z0-9]/g, "")
      .toUpperCase()
      .replace(/^[0-9]+/, "")
      .slice(0, 5);

  const trimmed = value.trim();
  const isEmpty = trimmed.length === 0;
  const isShort = !isEmpty && trimmed.length < 2;
  const isValid = CUSTOM_TAG_REGEX.test(trimmed);

  let hint: { tone: "muted" | "destructive" | "success"; text: string };
  if (isEmpty) {
    hint = { tone: "muted", text: t("hintEmpty") };
  } else if (isShort) {
    hint = { tone: "destructive", text: t("hintShort") };
  } else if (isValid) {
    hint = { tone: "success", text: t("hintValid", { tag: trimmed }) };
  } else {
    hint = { tone: "destructive", text: t("hintInvalid") };
  }

  return (
    <div className="space-y-1.5">
      <Label htmlFor={id}>{label}</Label>
      <Input
        id={id}
        value={value}
        onChange={(e) => onChange(sanitize(e.target.value))}
        placeholder={t("placeholder")}
        maxLength={5}
        disabled={disabled}
        autoComplete="off"
        autoCapitalize="characters"
        spellCheck={false}
        className="font-mono uppercase tracking-wider"
      />
      <p
        className={
          "text-micro " +
          (hint.tone === "destructive"
            ? "text-destructive"
            : hint.tone === "success"
              ? "text-success"
              : "text-muted-foreground")
        }
      >
        {hint.text}
      </p>
    </div>
  );
}
