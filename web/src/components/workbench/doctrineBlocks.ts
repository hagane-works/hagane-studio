export interface NamedBlockEntry {
  name: string;
  body: string;
}

export function tokenBlockToText(raw: unknown): string {
  if (Array.isArray(raw)) return raw.map((x) => String(x)).join(" ");
  if (raw == null) return "";
  return String(raw)
    .trim()
    .split(/[\s,]+/)
    .filter(Boolean)
    .join(" ");
}

export function textToTokenBlock(text: string): string {
  return text
    .split(/[\s,]+/)
    .map((s) => s.trim())
    .filter(Boolean)
    .join("\n");
}

export function parseNamedBlocks(blockText: string): NamedBlockEntry[] {
  const text = (blockText ?? "").trim();
  if (!text) return [];
  const entries: NamedBlockEntry[] = [];

  let i = 0;
  const n = text.length;
  let pending = "";

  const flushPendingToPrev = () => {
    if (!pending.trim()) {
      pending = "";
      return;
    }
    if (entries.length > 0) {
      entries[entries.length - 1].body +=
        (entries[entries.length - 1].body ? "\n" : "") + pending.trim();
    } else {
      entries.push({ name: "__prologue__", body: pending.trim() });
    }
    pending = "";
  };

  while (i < n) {
    const head = /^\s*([A-Za-z0-9_.]+)\s*=\s*\{/.exec(text.slice(i));
    if (head) {
      flushPendingToPrev();
      const name = head[1];
      let j = i + head[0].length;
      let depth = 1;
      const bodyStart = j;
      while (j < n && depth > 0) {
        const ch = text[j];
        if (ch === "{") depth++;
        else if (ch === "}") depth--;
        if (depth === 0) break;
        j++;
      }
      const body = text.slice(bodyStart, j);
      entries.push({ name, body: body.trim() });
      i = j + 1;
      continue;
    }

    pending += text[i];
    i++;
  }
  flushPendingToPrev();
  return entries;
}

export function serializeNamedBlocks(entries: NamedBlockEntry[]): string {
  const parts: string[] = [];
  for (const e of entries) {
    const name = (e.name ?? "").trim();
    const body = (e.body ?? "").trim();
    if (name === "__prologue__") {
      if (body) parts.push(body);
      continue;
    }
    if (!name && !body) continue;
    if (!name) {
      if (body) parts.push(body);
      continue;
    }
    const indented = body
      ? body
          .split("\n")
          .map((l) => (l.trim() ? `\t${l.trim()}` : ""))
          .join("\n")
      : "";
    parts.push(indented ? `${name} = {\n${indented}\n}` : `${name} = {}`);
  }
  return parts.join("\n");
}

export function sectionToText(raw: unknown): string {
  if (raw == null) return "";
  if (Array.isArray(raw)) return raw.map((x) => String(x)).join("\n");
  return String(raw);
}
