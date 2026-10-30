"use client";

import { useEffect, useId, useRef, useState } from "react";
import { useTranslations } from "use-intl";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Select } from "@/components/ui/select";
import {
  cosmeticTagsApi,
  COSMETIC_LOCALES,
  type CosmeticTagView,
} from "@/lib/api/cosmeticTags";
import { useContentLocaleStore } from "@/lib/store/contentLocaleStore";
import { useRegisterDirty } from "@/lib/hooks/useUnsavedGuard";

interface Props {
  projectId: number;
  countryTag: string;
  value: string;
  savedValue: string;
  latestView?: CosmeticTagView | null;
  disabled?: boolean;
  onChange: (tag: string) => Promise<boolean>;
}

export function CountryAppearanceSettings(props: Props) {
  return (
    <AppearanceSession
      key={`${props.projectId}:${props.countryTag}`}
      {...props}
    />
  );
}

function AppearanceSession({
  projectId,
  countryTag,
  value,
  savedValue,
  latestView,
  disabled,
  onChange,
}: Props) {
  const t = useTranslations("historyFilesDialog.appearance");
  const locale = useContentLocaleStore((s) => s.locale);
  const header = COSMETIC_LOCALES[locale];
  const id = useId();
  const [items, setItems] = useState<CosmeticTagView[]>([]);
  const [loading, setLoading] = useState(true);
  const [failed, setFailed] = useState(false);
  const [retry, setRetry] = useState(0);
  const [newName, setNewName] = useState("");
  const [creating, setCreating] = useState(false);
  const [createOpen, setCreateOpen] = useState(false);
  const [createFailed, setCreateFailed] = useState(false);
  const [changing, setChanging] = useState(false);
  const created = useRef<CosmeticTagView | null>(null);
  const alive = useRef(true);
  useRegisterDirty(
    `appearance-setup:${projectId}:${countryTag}`,
    !!newName.trim() || creating,
  );

  useEffect(() => {
    alive.current = true;
    return () => {
      alive.current = false;
    };
  }, []);
  useEffect(() => {
    const controller = new AbortController();
    cosmeticTagsApi
      .list(projectId, controller.signal)
      .then((response) => {
        if (controller.signal.aborted) return;
        setItems(response.cosmetic_tags);
        setLoading(false);
        setFailed(false);
      })
      .catch(() => {
        if (!controller.signal.aborted) {
          setLoading(false);
          setFailed(true);
        }
      });
    return () => controller.abort();
  }, [projectId, retry]);

  async function change(next: string) {
    setChanging(true);
    try {
      return await onChange(next);
    } finally {
      if (alive.current) setChanging(false);
    }
  }

  async function createAppearance() {
    if (!newName.trim() || !countryTag || creating) return;
    setCreating(true);
    setCreateFailed(false);
    try {
      const row =
        created.current ??
        (await cosmeticTagsApi.create(
          projectId,
          `${countryTag}_appearance_${crypto.randomUUID().replaceAll("-", "")}`,
        ));
      created.current = row;
      if (!alive.current) return;
      const named = await cosmeticTagsApi.update(projectId, row.tag, {
        expected_revision: row.revision,
        name_ops: [{ locale: header, suffix: "", value: newName.trim() }],
      });
      created.current = named;
      if (!alive.current) return;
      setItems((current) => [
        ...current.filter((item) => item.tag !== named.tag),
        named,
      ]);
      if (await change(named.tag)) {
        setCreateOpen(false);
        setNewName("");
        created.current = null;
      }
    } catch {
      if (alive.current) setCreateFailed(true);
    } finally {
      if (alive.current) setCreating(false);
    }
  }

  const busy = disabled || creating || changing;
  const listedView = items.find((item) => item.tag === latestView?.tag);
  const visibleItems =
    latestView?.project_id === projectId &&
    (!listedView || latestView.revision >= listedView.revision)
      ? listedView
        ? items.map((item) => (item.tag === latestView.tag ? latestView : item))
        : [...items, latestView]
      : items;
  const titleFor = (item: CosmeticTagView, index: number) =>
    item.name_locs[header]?.[""] ||
    item.imported_name_locs[header]?.[""] ||
    item.name_locs.l_english?.[""] ||
    item.imported_name_locs.l_english?.[""] ||
    t("untitled", { number: index + 1 });
  return (
    <section
      aria-label={t("title")}
      className="space-y-3 rounded-lg border border-border bg-card p-4"
    >
      <h3 className="text-base font-semibold">{t("title")}</h3>
      <label htmlFor={`${id}-select`} className="block text-sm">
        {t("select")}
      </label>
      <div className="flex flex-wrap items-center gap-2">
        <Select
          id={`${id}-select`}
          value={value}
          disabled={busy || loading}
          onChange={(event) => void change(event.target.value)}
          className="min-w-0 flex-1 ui-interactive"
        >
          <option value="">{t("none")}</option>
          {value && !visibleItems.some((item) => item.tag === value) && (
            <option value={value}>{t("existing")}</option>
          )}
          {visibleItems.map((item, index) => (
            <option key={item.tag} value={item.tag}>
              {titleFor(item, index)}
            </option>
          ))}
        </Select>
        {value && (
          <Button
            type="button"
            size="sm"
            variant="outline"
            disabled={busy}
            onClick={() => void change("")}
          >
            {t("stopUsing")}
          </Button>
        )}
        <Button
          type="button"
          size="sm"
          variant="outline"
          disabled={busy || !countryTag}
          onClick={() => setCreateOpen(true)}
        >
          {t("create")}
        </Button>
      </div>
      <p className="text-xs text-muted-foreground">{t("stopUsingHelp")}</p>
      {value !== savedValue && (
        <p role="status" className="text-sm">
          {t("pending")}
        </p>
      )}
      {loading && (
        <p role="status" className="text-xs text-muted-foreground">
          {t("loading")}
        </p>
      )}
      {failed && (
        <div role="alert" className="flex items-center gap-2 text-sm">
          <span>{t("loadFailed")}</span>
          <Button
            type="button"
            size="sm"
            variant="outline"
            onClick={() => {
              setLoading(true);
              setRetry((n) => n + 1);
            }}
          >
            {t("retry")}
          </Button>
        </div>
      )}
      {createOpen && (
        <div className="space-y-2 rounded-md bg-muted p-3">
          <label htmlFor={`${id}-name`} className="text-sm">
            {t("newName")}
          </label>
          <Input
            id={`${id}-name`}
            value={newName}
            disabled={busy}
            placeholder={t("newPlaceholder")}
            onChange={(event) => setNewName(event.target.value)}
          />
          <p className="text-xs text-muted-foreground">{t("createdHint")}</p>
          <div className="flex gap-2">
            <Button
              type="button"
              size="sm"
              disabled={busy || !newName.trim()}
              onClick={() => void createAppearance()}
            >
              {creating ? t("creating") : t("createAndUse")}
            </Button>
            <Button
              type="button"
              size="sm"
              variant="ghost"
              disabled={busy}
              onClick={() => {
                setCreateOpen(false);
                setNewName("");
              }}
            >
              {t("cancel")}
            </Button>
          </div>
          {createFailed && (
            <p role="alert" className="text-sm text-destructive">
              {t("creationFailed")}
            </p>
          )}
        </div>
      )}
      <details className="text-xs text-muted-foreground">
        <summary className="cursor-pointer ui-focus-ring ui-interactive">
          {t("advanced")}
        </summary>
        <div className="space-y-2 pt-2">
          <label htmlFor={`${id}-tag`}>{t("identifier")}</label>
          <Input
            id={`${id}-tag`}
            value={value}
            disabled={busy}
            onChange={(event) => void change(event.target.value)}
            className="font-mono text-xs"
          />
          <p>{t("identifierHelp")}</p>
        </div>
      </details>
    </section>
  );
}
