"use client";

import { useEffect, useMemo, useRef, useState } from "react";
import { useTranslations } from "use-intl";
import { useParams } from "@/platform/navigation";
import {
  AlertCircle,
  Download,
  FileImage,
  Loader2,
  Package,
  PackageOpen,
  RotateCcw,
  Sparkles,
  Upload,
  X,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { Badge } from "@/components/ui/badge";
import { Select } from "@/components/ui/select";

const TABS = [
  { value: "convert", labelKey: "tabConvert", icon: FileImage },
  { value: "vanilla", labelKey: "tabVanilla", icon: Sparkles },
  { value: "placeholder", labelKey: "tabPlaceholder", icon: Upload },
] as const;

type TabValue = (typeof TABS)[number]["value"];

const ASSET_TYPES = [
  { value: "idea_pictures", labelKey: "assetIdeaPictures", dims: "235×104" },
  { value: "event_pictures", labelKey: "assetEventPictures", dims: "165×165" },
  {
    value: "decision_pictures",
    labelKey: "assetDecisionPictures",
    dims: "152×80",
  },
  { value: "focus_icons", labelKey: "assetFocusIcons", dims: "95×95" },
] as const;

interface ConvertFileState {
  name: string;
  status: "queued" | "ok" | "error";
  progress?: number;
  width?: number;
  height?: number;
  output_size_bytes?: number;
  error?: string;
  b64_output?: string;
}

function DDSConvertTab() {
  const t = useTranslations("imageBatch");
  const [files, setFiles] = useState<File[]>([]);
  const [targetWidth, setTargetWidth] = useState<string>("");
  const [targetHeight, setTargetHeight] = useState<string>("");
  const [results, setResults] = useState<ConvertFileState[]>([]);
  const [streaming, setStreaming] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleFilesPicked = (picked: FileList | null) => {
    if (!picked) return;
    setFiles((prev) => [...prev, ...Array.from(picked)]);
  };

  const removeFile = (idx: number) => {
    setFiles((prev) => prev.filter((_, i) => i !== idx));
  };

  const handleConvert = async () => {
    if (files.length === 0) return;
    setStreaming(true);
    setError(null);
    setResults(files.map((f) => ({ name: f.name, status: "queued" })));

    const form = new FormData();
    for (const f of files) {
      form.append("files", f);
    }
    if (targetWidth.trim() || targetHeight.trim()) {
      form.append("target_width", targetWidth.trim());
      form.append("target_height", targetHeight.trim());
    }

    let res: Response;
    try {
      res = await fetch("/api/go/image-batch/convert-dds", {
        method: "POST",
        body: form,
      });
    } catch (e) {
      setError(`Network: ${(e as Error).message}`);
      setStreaming(false);
      return;
    }

    if (!res.ok) {
      let detail = `${res.status}`;
      try {
        const body = await res.json();
        detail = body.detail || JSON.stringify(body);
      } catch {}
      setError(`HTTP ${res.status}: ${detail}`);
      setStreaming(false);
      return;
    }

    const body = res.body;
    if (!body) {
      setError(t("errNoResponseBody"));
      setStreaming(false);
      return;
    }
    const reader = body.getReader();
    const decoder = new TextDecoder();
    let buffer = "";
    try {
      while (true) {
        const { done, value } = await reader.read();
        if (done) break;
        buffer += decoder.decode(value, { stream: true });
        let sep: number;
        while ((sep = buffer.indexOf("\n\n")) >= 0) {
          const frame = buffer.slice(0, sep);
          buffer = buffer.slice(sep + 2);
          if (!frame.startsWith("data:")) continue;
          const payload = frame.slice(5).trim();
          if (!payload) continue;
          try {
            const ev = JSON.parse(payload) as Record<string, unknown>;
            if (ev.event === "file") {
              const idx = ev.index as number;
              setResults((prev) =>
                prev.map((r, i) =>
                  i === idx
                    ? {
                        ...r,
                        status: ev.status as ConvertFileState["status"],
                        progress: ev.progress as number,
                        width: ev.width as number | undefined,
                        height: ev.height as number | undefined,
                        output_size_bytes: ev.output_size_bytes as
                          number | undefined,
                        error: ev.error as string | undefined,
                        b64_output: ev.b64_output as string | undefined,
                      }
                    : r,
                ),
              );
            }
          } catch {}
        }
      }
    } finally {
      reader.releaseLock();
      setStreaming(false);
    }
  };

  const downloadFile = (r: ConvertFileState) => {
    if (!r.b64_output) return;
    const bin = atob(r.b64_output);
    const bytes = new Uint8Array(bin.length);
    for (let i = 0; i < bin.length; i++) bytes[i] = bin.charCodeAt(i);
    const blob = new Blob([bytes as BlobPart], { type: "image/vnd.ms-dds" });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = r.name.replace(/\.[^.]+$/, "") + ".dds";
    a.click();
    URL.revokeObjectURL(url);
  };

  const downloadAllAsZip = async () => {
    for (const r of results) {
      if (r.status === "ok" && r.b64_output) {
        downloadFile(r);

        await new Promise((r) => setTimeout(r, 80));
      }
    }
  };

  const okCount = results.filter((r) => r.status === "ok").length;
  const errCount = results.filter((r) => r.status === "error").length;

  return (
    <div className="space-y-4">
      <div className="space-y-2">
        <Label>{t("filesLabel")}</Label>
        <div className="rounded-lg border border-dashed border-border p-4">
          <label className="inline-flex items-center gap-2 text-xs cursor-pointer">
            <span className="px-3 py-1.5 rounded bg-accent text-accent-foreground">
              {t("chooseFiles")}
            </span>
            <span className="text-muted-foreground">
              {files.length > 0
                ? t("filesSelected", { count: files.length })
                : t("noFileChosen")}
            </span>
            <input
              type="file"
              multiple
              accept="image/png,image/x-tga,application/octet-stream,.dds,.tga"
              onChange={(e) => handleFilesPicked(e.target.files)}
              className="sr-only"
            />
          </label>
          {files.length > 0 && (
            <ul className="mt-3 space-y-1 max-h-40 overflow-y-auto">
              {files.map((f, i) => (
                <li
                  key={i}
                  className="flex items-center justify-between text-xs px-2 py-1 rounded bg-muted/40"
                >
                  <span className="truncate flex-1 font-mono">{f.name}</span>
                  <span className="text-muted-foreground ml-2 font-mono">
                    {(f.size / 1024).toFixed(0)} KB
                  </span>
                  <button
                    onClick={() => removeFile(i)}
                    className="ml-2 text-muted-foreground hover:text-destructive"
                    title={t("remove")}
                  >
                    <X className="h-3 w-3" />
                  </button>
                </li>
              ))}
            </ul>
          )}
        </div>
      </div>

      <div className="grid grid-cols-2 gap-3">
        <div className="space-y-1">
          <Label>{t("targetWidthLabel")}</Label>
          <Input
            type="number"
            value={targetWidth}
            onChange={(e) => setTargetWidth(e.target.value)}
            placeholder="235"
            min={1}
            className="text-xs font-mono"
          />
        </div>
        <div className="space-y-1">
          <Label>{t("targetHeightLabel")}</Label>
          <Input
            type="number"
            value={targetHeight}
            onChange={(e) => setTargetHeight(e.target.value)}
            placeholder="104"
            min={1}
            className="text-xs font-mono"
          />
        </div>
      </div>

      <div className="flex items-center gap-2">
        <Button
          onClick={handleConvert}
          disabled={files.length === 0 || streaming}
          size="sm"
        >
          {streaming ? (
            <>
              <Loader2 className="h-3 w-3 mr-1 animate-spin" />
              {t("converting")}
            </>
          ) : (
            <>{t("convertButton", { count: files.length })}</>
          )}
        </Button>
        {okCount > 0 && !streaming && (
          <Button size="sm" variant="outline" onClick={downloadAllAsZip}>
            <Download className="h-3 w-3 mr-1" />
            {t("downloadAllConverted", { count: okCount })}
          </Button>
        )}
        {files.length > 0 && !streaming && (
          <Button
            size="sm"
            variant="ghost"
            onClick={() => {
              setFiles([]);
              setResults([]);
              setError(null);
            }}
          >
            <RotateCcw className="h-3 w-3 mr-1" />
            {t("reset")}
          </Button>
        )}
      </div>

      {error && (
        <div className="rounded border border-destructive/40 bg-destructive/5 px-3 py-2 text-xs text-destructive">
          {error}
        </div>
      )}

      {results.length > 0 && (
        <div className="space-y-2">
          <div className="flex items-center gap-2 text-xs">
            <Badge variant="outline">
              {t("badgeTotal", { count: results.length })}
            </Badge>
            {okCount > 0 && (
              <Badge className="bg-success text-success-foreground hover:bg-success/90">
                {t("badgeOk", { count: okCount })}
              </Badge>
            )}
            {errCount > 0 && (
              <Badge variant="destructive">
                {t("badgeError", { count: errCount })}
              </Badge>
            )}
          </div>
          <ul className="space-y-1 max-h-72 overflow-y-auto rounded border border-border">
            {results.map((r, i) => (
              <li
                key={i}
                className={
                  "flex items-center gap-2 px-2 py-1.5 text-xs border-b border-border last:border-0 " +
                  (r.status === "error" ? "bg-destructive/5" : "")
                }
              >
                <span className="font-mono truncate flex-1" title={r.name}>
                  {r.name}
                </span>
                {r.status === "queued" && (
                  <span className="text-muted-foreground">
                    {t("statusQueued")}
                  </span>
                )}
                {r.status === "ok" && (
                  <>
                    <span className="text-success font-mono">
                      {r.width}×{r.height}
                    </span>
                    <span className="text-muted-foreground font-mono">
                      {((r.output_size_bytes ?? 0) / 1024).toFixed(0)} KB
                    </span>
                    <Button
                      size="icon-sm"
                      variant="ghost"
                      onClick={() => downloadFile(r)}
                      title={t("downloadDds")}
                    >
                      <Download className="h-3 w-3" />
                    </Button>
                  </>
                )}
                {r.status === "error" && (
                  <span
                    className="text-destructive truncate max-w-md"
                    title={r.error}
                  >
                    {r.error}
                  </span>
                )}
              </li>
            ))}
          </ul>
        </div>
      )}
    </div>
  );
}

interface VanillaAsset {
  name: string;
  filename: string;
  relative_path: string;
}

interface VanillaRootStatus {
  resolved_path: string;
  root_exists: boolean;
  asset_dirs: Array<{
    asset_type: string;
    subdir: string;
    exists: boolean;
    dds_count: number;
  }>;
}

function VanillaExtractTab() {
  const t = useTranslations("imageBatch");
  const [assetType, setAssetType] = useState<string>("idea_pictures");
  const [assets, setAssets] = useState<VanillaAsset[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [filter, setFilter] = useState("");
  const [rootStatus, setRootStatus] = useState<VanillaRootStatus | null>(null);
  const assetRequestGeneration = useRef(0);

  useEffect(() => {
    fetch("/api/go/image-batch/vanilla-root-status")
      .then((r) => (r.ok ? r.json() : null))
      .then((body) => setRootStatus(body))
      .catch(() => setRootStatus(null));
  }, []);

  const loadAssets = async (type: string) => {
    const request = ++assetRequestGeneration.current;
    setLoading(true);
    setError(null);
    try {
      const res = await fetch(`/api/go/image-batch/vanilla-extract/${type}`);
      if (request !== assetRequestGeneration.current || type !== assetType)
        return;
      if (!res.ok) {
        setError(`HTTP ${res.status}: ${await res.text()}`);
        setAssets([]);
        return;
      }
      const body = await res.json();
      if (request !== assetRequestGeneration.current || type !== assetType)
        return;
      setAssets(body.assets || []);
    } catch (e) {
      if (request !== assetRequestGeneration.current || type !== assetType)
        return;
      setError(`Network: ${(e as Error).message}`);
      setAssets([]);
    } finally {
      if (request === assetRequestGeneration.current) setLoading(false);
    }
  };

  const filtered = useMemo(() => {
    const q = filter.trim().toLowerCase();
    if (!q) return assets;
    return assets.filter((a) => a.name.toLowerCase().includes(q));
  }, [assets, filter]);

  const activeAssetDirStatus = rootStatus?.asset_dirs?.find(
    (d) => d.asset_type === assetType,
  );

  const assetLabel = (a: (typeof ASSET_TYPES)[number]) =>
    `${t(a.labelKey)} (${a.dims})`;

  return (
    <div className="space-y-4">
      {rootStatus && !rootStatus.root_exists && (
        <div className="rounded-md border border-warning/40 bg-warning/5 px-3 py-2 text-xs">
          <div className="flex items-start gap-1.5 text-warning font-medium">
            <AlertCircle className="h-3.5 w-3.5 mt-0.5 shrink-0" />
            <span>{t("vanillaRootMissing")}</span>
          </div>
        </div>
      )}
      {rootStatus?.root_exists &&
        activeAssetDirStatus &&
        !activeAssetDirStatus.exists && (
          <div className="rounded-md border border-warning/40 bg-warning/5 px-3 py-2 text-xs">
            <div className="flex items-start gap-1.5 text-warning">
              <AlertCircle className="h-3.5 w-3.5 mt-0.5 shrink-0" />
              <span>
                {t.rich("assetSubdirMissing", {
                  subdir: activeAssetDirStatus.subdir,
                  code: (chunks) => <code className="font-mono">{chunks}</code>,
                })}
              </span>
            </div>
          </div>
        )}
      {rootStatus?.root_exists && (
        <div className="rounded-md border border-success/30 bg-success/5 px-3 py-2 text-xs">
          <p className="text-success">
            {t("vanillaRootReady")}{" "}
            <code className="font-mono">{rootStatus.resolved_path}</code>
          </p>
          {rootStatus.asset_dirs.length > 0 && (
            <p className="text-muted-foreground mt-1">
              {rootStatus.asset_dirs
                .map((d) => `${d.asset_type}: ${d.dds_count}`)
                .join(" · ")}
            </p>
          )}
        </div>
      )}

      <div className="grid grid-cols-[1fr_auto_auto] gap-2 items-end">
        <div className="space-y-1">
          <Label>{t("assetTypeLabel")}</Label>
          <Select
            value={assetType}
            onChange={(e) => {
              assetRequestGeneration.current += 1;
              setAssetType(e.target.value);
              setAssets([]);
              setError(null);
            }}
            className="text-xs font-mono px-2"
          >
            {ASSET_TYPES.map((a) => (
              <option key={a.value} value={a.value}>
                {assetLabel(a)}
              </option>
            ))}
          </Select>
        </div>
        <Button
          size="sm"
          onClick={() => loadAssets(assetType)}
          disabled={loading}
        >
          {loading ? (
            <Loader2 className="h-3 w-3 mr-1 animate-spin" />
          ) : (
            <Sparkles className="h-3 w-3 mr-1" />
          )}
          {t("load")}
        </Button>
        {assets.length > 0 && (
          <Input
            value={filter}
            onChange={(e) => setFilter(e.target.value)}
            placeholder={t("filterPlaceholder")}
            className="w-40"
          />
        )}
      </div>

      {error && (
        <div className="rounded border border-destructive/40 bg-destructive/5 px-3 py-2 text-xs text-destructive">
          {error}
        </div>
      )}

      {assets.length === 0 && !loading && !error && (
        <div className="ui-card flex flex-col items-center justify-center gap-3 py-10 px-6 text-center">
          <PackageOpen className="h-8 w-8 text-muted-foreground" />
          <div className="space-y-1">
            <p className="text-sm font-medium text-foreground">
              {(() => {
                const a = ASSET_TYPES.find((x) => x.value === assetType);
                return a ? assetLabel(a) : t("assetsFallback");
              })()}
            </p>
            <p className="text-xs text-muted-foreground">
              {t("vanillaHint", {
                label: (() => {
                  const a = ASSET_TYPES.find((x) => x.value === assetType);
                  return a ? assetLabel(a) : "";
                })(),
              })}
            </p>
          </div>
        </div>
      )}

      {filtered.length > 0 && (
        <>
          <div className="flex items-center gap-2 text-xs">
            <Badge variant="outline">
              {t("assetsCount", {
                shown: filtered.length,
                total: assets.length,
              })}
            </Badge>
          </div>
          <ul className="space-y-0.5 max-h-96 overflow-y-auto rounded border border-border">
            {filtered.map((a) => (
              <li
                key={a.relative_path}
                className="px-2 py-1.5 text-xs border-b border-border last:border-0 hover:bg-muted/40"
              >
                <div className="flex items-center gap-2">
                  <span className="font-mono truncate flex-1" title={a.name}>
                    {a.name}
                  </span>
                  <span
                    className="text-muted-foreground font-mono text-xs truncate"
                    title={a.relative_path}
                  >
                    {a.relative_path}
                  </span>
                </div>
              </li>
            ))}
          </ul>
        </>
      )}
    </div>
  );
}

function GeneratePlaceholderTab() {
  const t = useTranslations("imageBatch");
  const [entityList, setEntityList] = useState<string>("");
  const [width, setWidth] = useState<string>("235");
  const [height, setHeight] = useState<string>("104");
  const [bgColor, setBgColor] = useState<string>("60,60,80");
  const [generating, setGenerating] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [lastCount, setLastCount] = useState<number | null>(null);

  const parseEntities = (): string[] =>
    entityList
      .split(/[\n,]/)
      .map((s) => s.trim())
      .filter(Boolean);

  const parseBgColor = (): [number, number, number] => {
    const parts = bgColor.split(",").map((s) => parseInt(s.trim(), 10));
    if (
      parts.length === 3 &&
      parts.every((n) => !isNaN(n) && n >= 0 && n <= 255)
    ) {
      return [parts[0], parts[1], parts[2]];
    }
    return [60, 60, 80];
  };

  const handleGenerate = async () => {
    const entities = parseEntities();
    if (entities.length === 0) {
      setError(t("errNeedEntityId"));
      return;
    }
    const w = parseInt(width, 10);
    const h = parseInt(height, 10);
    if (!w || !h || w <= 0 || h <= 0) {
      setError(t("errPositiveDimensions"));
      return;
    }
    setGenerating(true);
    setError(null);
    setLastCount(null);

    const bg = parseBgColor();
    try {
      const res = await fetch("/api/go/image-batch/generate-placeholder", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          entities: entities.map((eid) => ({ entity_id: eid, bg_color: bg })),
          width: w,
          height: h,
          output: "zip",
        }),
      });
      if (!res.ok) {
        let detail = `${res.status}`;
        try {
          const body = await res.json();
          detail = body.detail || JSON.stringify(body);
        } catch {}
        setError(`HTTP ${res.status}: ${detail}`);
        return;
      }
      const blob = await res.blob();
      const url = URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = url;
      a.download = "placeholders.zip";
      a.click();
      URL.revokeObjectURL(url);
      setLastCount(entities.length);
    } catch (e) {
      setError(`Network: ${(e as Error).message}`);
    } finally {
      setGenerating(false);
    }
  };

  return (
    <div className="space-y-4">
      <div className="space-y-1">
        <Label>{t("entityIdsLabel")}</Label>
        <Textarea
          value={entityList}
          onChange={(e) => setEntityList(e.target.value)}
          placeholder={
            "modern_usa_fiscal_state\nwar_economy_strain\nindustrial_giant"
          }
          className="font-mono text-xs min-h-32"
          disabled={generating}
        />
        <p className="text-xs text-muted-foreground">
          {t("placeholderSizeHint")}
        </p>
      </div>

      <div className="grid grid-cols-3 gap-3">
        <div className="space-y-1">
          <Label>{t("widthLabel")}</Label>
          <Input
            type="number"
            value={width}
            onChange={(e) => setWidth(e.target.value)}
            min={1}
            className="text-xs font-mono"
            disabled={generating}
          />
        </div>
        <div className="space-y-1">
          <Label>{t("heightLabel")}</Label>
          <Input
            type="number"
            value={height}
            onChange={(e) => setHeight(e.target.value)}
            min={1}
            className="text-xs font-mono"
            disabled={generating}
          />
        </div>
        <div className="space-y-1">
          <Label>{t("bgColorLabel")}</Label>
          <Input
            type="text"
            value={bgColor}
            onChange={(e) => setBgColor(e.target.value)}
            placeholder="60,60,80"
            className="text-xs font-mono"
            disabled={generating}
          />
        </div>
      </div>

      <div className="flex items-center gap-2">
        <Button onClick={handleGenerate} disabled={generating} size="sm">
          {generating ? (
            <>
              <Loader2 className="h-3 w-3 mr-1 animate-spin" />
              {t("generating")}
            </>
          ) : (
            <>
              <Download className="h-3 w-3 mr-1" />
              {t("generateZip", { count: parseEntities().length })}
            </>
          )}
        </Button>
        {lastCount !== null && (
          <span className="text-xs text-success">
            {t("downloadedPlaceholders", { count: lastCount })}
          </span>
        )}
      </div>

      {error && (
        <div className="rounded border border-destructive/40 bg-destructive/5 px-3 py-2 text-xs text-destructive">
          {error}
        </div>
      )}
    </div>
  );
}

export default function ImageBatchPage() {
  useParams();
  const t = useTranslations("imageBatch");
  const [tab, setTab] = useState<TabValue>("convert");

  return (
    <div className="flex flex-col h-full min-w-0">
      <div className="border-b border-border px-4 pt-4 pb-2 shrink-0">
        <h1 className="flex items-center gap-2 text-lg font-semibold tracking-tight">
          <Package
            className="h-4 w-4 shrink-0 text-muted-foreground"
            aria-hidden
          />
          {t("pageTitle")}
        </h1>
        <p className="text-xs text-muted-foreground mt-1">
          {t("pageSubtitle")}
        </p>
      </div>

      <div className="border-b border-border px-4 shrink-0">
        <nav className="flex gap-1">
          {TABS.map((tabDef) => {
            const Icon = tabDef.icon;
            const active = tab === tabDef.value;
            return (
              <button
                key={tabDef.value}
                onClick={() => setTab(tabDef.value)}
                className={
                  "flex items-center gap-1.5 px-3 py-2 text-xs font-medium ui-interactive border-b-2 -mb-px " +
                  (active
                    ? "border-accent text-foreground"
                    : "border-transparent text-muted-foreground hover:text-foreground")
                }
              >
                <Icon className="h-3.5 w-3.5" />
                {t(tabDef.labelKey)}
              </button>
            );
          })}
        </nav>
      </div>

      <div className="flex-1 min-h-0 overflow-y-auto p-4">
        <div className="max-w-3xl">
          <div hidden={tab !== "convert"}>
            <DDSConvertTab />
          </div>
          <div hidden={tab !== "vanilla"}>
            <VanillaExtractTab />
          </div>
          <div hidden={tab !== "placeholder"}>
            <GeneratePlaceholderTab />
          </div>
        </div>
      </div>
    </div>
  );
}
