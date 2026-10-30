const COLOR_BLOCK_RE =
  /\bcolor\s*=\s*\{\s*(\d{1,3})\s+(\d{1,3})\s+(\d{1,3})\s*\}/;

function channelToHex(value: number): string {
  return Math.max(0, Math.min(255, value)).toString(16).padStart(2, "0");
}

export function rootBodyColor(body: string): string {
  const match = body.match(COLOR_BLOCK_RE);
  if (!match) return "#808080";
  const channels = match.slice(1, 4).map((value) => Number(value));
  if (
    channels.some(
      (value) => !Number.isInteger(value) || value < 0 || value > 255,
    )
  ) {
    return "#808080";
  }
  return `#${channels.map(channelToHex).join("")}`;
}

export function rootBodyWithColor(body: string, color: string): string {
  const normalized = /^#[0-9a-f]{6}$/i.test(color) ? color : "#808080";
  const channels = [1, 3, 5].map((offset) =>
    Number.parseInt(normalized.slice(offset, offset + 2), 16),
  );
  const colorBlock = `color = { ${channels.join(" ")} }`;
  if (COLOR_BLOCK_RE.test(body))
    return body.replace(COLOR_BLOCK_RE, colorBlock);
  return `${body.trim()}\n\n${colorBlock}`.trim();
}

export function defaultRootBody(color = "#808080"): string {
  return rootBodyWithColor("types = {\n}", color);
}

export function rootBodyDraftError(
  body: string,
): "missing_types" | "missing_color" | "unbalanced" | null {
  if (!/\btypes\s*=\s*\{/.test(body)) return "missing_types";
  if (!COLOR_BLOCK_RE.test(body)) return "missing_color";
  let depth = 0;
  for (const char of body) {
    if (char === "{") depth += 1;
    if (char === "}") depth -= 1;
    if (depth < 0) return "unbalanced";
  }
  return depth === 0 ? null : "unbalanced";
}
