export const GRAPHICAL_CULTURES = [
  "western_european_gfx",
  "eastern_european_gfx",
  "middle_eastern_gfx",
  "asian_gfx",
  "asian_european_gfx",
  "african_gfx",
  "southamerican_gfx",
  "commonwealth_gfx",
  "sami_gfx",
] as const;

export type GraphicalCulture = (typeof GRAPHICAL_CULTURES)[number];

const GRAPHICAL_CULTURE_TO_2D: Record<string, string> = {
  western_european_gfx: "western_european_2d",
  eastern_european_gfx: "eastern_european_2d",
  middle_eastern_gfx: "middle_eastern_2d",
  asian_gfx: "asian_2d",
  asian_european_gfx: "asian_2d",
  african_gfx: "african_2d",
  southamerican_gfx: "southamerican_2d",
  commonwealth_gfx: "commonwealth_2d",
  sami_gfx: "sami_2d",
};

export function deriveGraphicalCulture2d(gfx: string): string {
  const g = (gfx || "").trim();
  if (!g) return "";
  return GRAPHICAL_CULTURE_TO_2D[g] ?? g.replace("_gfx", "_2d");
}

export function rgbToHex(rgb?: number[] | null): string {
  if (!rgb || rgb.length !== 3) return "#808080";
  const clamp = (n: number) => Math.max(0, Math.min(255, Math.round(n || 0)));
  return (
    "#" +
    rgb
      .slice(0, 3)
      .map((c) => clamp(c).toString(16).padStart(2, "0"))
      .join("")
  );
}

export function hexToRgb(hex: string): number[] {
  const m = /^#?([0-9a-fA-F]{6})$/.exec((hex || "").trim());
  if (!m) return [128, 128, 128];
  const int = parseInt(m[1], 16);
  return [(int >> 16) & 255, (int >> 8) & 255, int & 255];
}
