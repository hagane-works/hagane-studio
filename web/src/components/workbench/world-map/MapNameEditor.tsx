"use client";

import { useState } from "react";
import { useTranslations } from "use-intl";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { nameChanges } from "./mapNames";

export function MapNameEditor({
  label,
  shown,
  languages,
  saved,
  where,
  onSave,
}: {
  label: string;

  shown: string;

  languages: string[];

  saved: Record<string, string | undefined>;

  where: string;

  onSave: (changes: Record<string, string | null>) => Promise<void>;
}) {
  const t = useTranslations("worldMap");
  const [draft, setDraft] = useState<Record<string, string> | null>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const own = languages.some((lang) => saved[lang]);

  const run = async (changes: Record<string, string | null>) => {
    if (!Object.keys(changes).length) {
      setDraft(null);
      return;
    }
    setBusy(true);
    setError(null);
    try {
      await onSave(changes);
      setDraft(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setBusy(false);
    }
  };

  return (
    <div>
      <div className="flex items-center justify-between gap-2">
        <div className="min-w-0">
          <div className="text-xs text-muted-foreground">{label}</div>
          <div className="truncate font-medium">
            {shown || "—"}
            {own && (
              <span className="ml-1.5 rounded border border-accent/50 px-1 text-micro text-accent">
                {t("ownName")}
              </span>
            )}
          </div>
        </div>
        {!draft && (
          <Button
            size="sm"
            variant="outline"
            onClick={() =>
              setDraft(
                Object.fromEntries(
                  languages.map((lang) => [lang, saved[lang] ?? ""]),
                ),
              )
            }
          >
            {t("rename")}
          </Button>
        )}
      </div>
      {draft && (
        <div className="mt-2 space-y-2">
          {languages.map((lang) => (
            <label key={lang} className="block text-xs">
              <span className="font-mono text-muted-foreground">
                {lang.toUpperCase()}
              </span>
              <Input
                className="mt-1"
                value={draft[lang] ?? ""}
                placeholder={lang === "en" ? shown : t("otherPlaceholder")}
                disabled={busy}
                onChange={(e) => setDraft({ ...draft, [lang]: e.target.value })}
              />
            </label>
          ))}
          <div className="flex flex-wrap items-center gap-2">
            <Button
              size="sm"
              disabled={busy}
              onClick={() => void run(nameChanges(draft, saved))}
            >
              {busy ? t("saving") : t("save")}
            </Button>
            <Button
              size="sm"
              variant="ghost"
              disabled={busy}
              onClick={() => setDraft(null)}
            >
              {t("cancel")}
            </Button>
            {own && (
              <Button
                size="sm"
                variant="ghost"
                disabled={busy}
                onClick={() =>
                  void run(
                    Object.fromEntries(
                      languages
                        .filter((lang) => saved[lang])
                        .map((lang) => [lang, null]),
                    ),
                  )
                }
              >
                {t("restore")}
              </Button>
            )}
          </div>
          <p className="text-xs text-muted-foreground">{where}</p>
        </div>
      )}
      {error && <p className="mt-1 text-xs text-destructive">{error}</p>}
    </div>
  );
}
