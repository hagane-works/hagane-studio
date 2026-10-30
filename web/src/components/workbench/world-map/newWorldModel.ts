export type WorldSource = "sketch" | "game" | "upload" | "mod";

export const UPLOAD_FIELDS = [
  "provinces",
  "heightmap",
  "terrain",
  "rivers",
  "definition",
] as const;
export type UploadField = (typeof UPLOAD_FIELDS)[number];

export const UPLOAD_REQUIRED: readonly UploadField[] = [
  "provinces",
  "heightmap",
  "terrain",
];
export type UploadFiles = Partial<Record<UploadField, File | null>>;

export const WORLD_SIZES = { fine: 400, medium: 2000, coarse: 10000 } as const;
export type WorldSize = keyof typeof WORLD_SIZES;

export const WORLD_MAX_COUNTRIES = 8;

export interface WorldSummary {
  source: WorldSource;
  seed: number;
  seconds: number;
  provinces: { land: number; sea: number; lake: number };
  states: number;
  regions: number;
  countries: { tag: string; states: number; own_start: boolean }[];

  unloaded: string[];

  replaced_states: number;

  made?: string[];

  fixed?: Record<string, number>;
}

export interface WorldFinding {
  code: string;
  severity: "error" | "warning";
  message: string;
  count: number;
}

export interface WorldPreview {
  summary: WorldSummary;
  findings: WorldFinding[];
  previews: { political: string; terrain?: string };
}

export function cleanTag(text: string): string {
  return text
    .toUpperCase()
    .replace(/[^A-Z0-9]/g, "")
    .slice(0, 3);
}

export function addTag(tags: string[], tag: string): string[] {
  const clean = cleanTag(tag);
  if (
    clean.length !== 3 ||
    tags.includes(clean) ||
    tags.length >= WORLD_MAX_COUNTRIES
  )
    return tags;
  return [...tags, clean];
}

export interface WorldRequest {
  source: WorldSource;
  sketch: File | null;
  upload: UploadFiles;
  size: WorldSize;
  seed: string;
  tags: string[];
  languages: string[];
}

export function worldForm(req: WorldRequest): FormData {
  const body = new FormData();
  body.append("source", req.source);
  if (req.source === "sketch" && req.sketch) {
    body.append("sketch", req.sketch);
    body.append("land", String(WORLD_SIZES[req.size]));
  }
  if (req.source === "upload") {
    for (const field of UPLOAD_FIELDS) {
      const file = req.upload[field];
      if (file) body.append(field, file);
    }
  }
  body.append("seed", req.seed || "1");
  body.append("tags", req.tags.join(","));
  body.append("languages", req.languages.join(","));
  return body;
}

export function uploadReady(files: UploadFiles): boolean {
  return UPLOAD_REQUIRED.every((field) => files[field]);
}

const fileKey = (file: File | null | undefined) =>
  file ? `${file.name}:${file.size}:${file.lastModified}` : "";

export function worldKey(req: WorldRequest): string {
  const map =
    req.source === "sketch" && req.sketch
      ? `${fileKey(req.sketch)}|${req.size}`
      : req.source === "upload"
        ? UPLOAD_FIELDS.map((field) => fileKey(req.upload[field])).join(",")
        : "";
  return [req.source, map, req.seed || "1", req.tags.join(",")].join("|");
}

export function unloadedLabel(folder: string): string {
  return `worldUnload_${folder.replace(/^(map|history|common)\//, "").replace(/\//g, "_")}`;
}
