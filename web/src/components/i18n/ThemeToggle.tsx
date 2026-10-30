"use client";

import { useEffect, useState } from "react";
import { useTranslations } from "use-intl";
import { Moon, Sun } from "lucide-react";

export function ThemeToggle({ className = "" }: { className?: string }) {
  const t = useTranslations("common");

  const [isDark, setIsDark] = useState(false);

  useEffect(() => {
    let active = true;
    queueMicrotask(() => {
      if (active) {
        setIsDark(document.documentElement.classList.contains("dark"));
      }
    });
    return () => {
      active = false;
    };
  }, []);

  const toggle = () => {
    const root = document.documentElement;

    const next = !root.classList.contains("dark");
    setIsDark(next);
    root.classList.toggle("dark", next);
    try {
      localStorage.setItem("theme", next ? "dark" : "light");
    } catch {}
  };

  const label = isDark ? t("theme.switchToLight") : t("theme.switchToDark");

  return (
    <button
      type="button"
      onClick={toggle}
      title={label}
      aria-label={label}
      aria-pressed={isDark}
      className={`inline-flex h-8 w-8 shrink-0 items-center justify-center rounded-md text-muted-foreground hover:text-accent hover:bg-muted ui-interactive ui-focus-ring ${className}`}
    >
      {isDark ? (
        <Sun className="h-4 w-4" aria-hidden />
      ) : (
        <Moon className="h-4 w-4" aria-hidden />
      )}
    </button>
  );
}
