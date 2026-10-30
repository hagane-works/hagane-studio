export function detectFocusPrefix(nodes: { focus_id: string }[]): string {
  if (nodes.length === 0) return "";
  const counts = new Map<string, number>();
  for (const n of nodes) {
    const m = n.focus_id.match(/^([A-Z]{2,3})_/);
    if (!m) continue;
    counts.set(m[1], (counts.get(m[1]) || 0) + 1);
  }
  if (counts.size === 0) return "";

  return [...counts.entries()].sort((a, b) => {
    if (b[1] !== a[1]) return b[1] - a[1];
    return a[0].localeCompare(b[0]);
  })[0][0];
}

export function suggestNewFocusId(
  prefix: string,
  existing: { focus_id: string }[],
): string {
  if (!prefix) return "";
  const taken = new Set(existing.map((n) => n.focus_id));
  const base = `${prefix}_new_focus`;
  if (!taken.has(base)) return base;
  for (let i = 2; i < 100; i++) {
    const candidate = `${base}_${i}`;
    if (!taken.has(candidate)) return candidate;
  }

  return "";
}
