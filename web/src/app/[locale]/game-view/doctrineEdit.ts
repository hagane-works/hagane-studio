import type { HistoryFileRow } from "@/lib/api/scenario";

const token = (name: string) => name.replace(/[^A-Za-z0-9_.-]/g, "");

export function swapCommand(
  script: string,
  command: string,
  from: string,
  to: string,
): string {
  const pattern = new RegExp(
    `(^|[^A-Za-z0-9_])(${command}\\s*=\\s*)${token(from)}(?![A-Za-z0-9_])`,
    "g",
  );
  return script.replace(
    pattern,
    (_, before: string, lead: string) => `${before}${lead}${token(to)}`,
  );
}

export function addCommand(
  script: string,
  command: string,
  value: string,
  after: readonly string[] = [],
): string {
  const lines = script.split("\n");
  const last: Record<string, number> = {};
  let depth = 0;
  lines.forEach((line, index) => {
    const code = line.replace(/#.*/, "").replace(/"[^"]*"/g, "");
    const name = code.match(/^\s*([A-Za-z0-9_]+)\s*=/)?.[1];
    if (depth === 0 && name) last[name] = index;
    for (const ch of code)
      depth = Math.max(0, depth + (ch === "{" ? 1 : ch === "}" ? -1 : 0));
  });
  const at =
    [command, ...after]
      .map((name) => last[name])
      .find((index) => index !== undefined) ?? -1;
  const line = `${command} = ${token(value)}`;
  if (at < 0) {
    const body = script.replace(/\s+$/, "");
    return body ? `${body}\n${line}\n` : `${line}\n`;
  }
  const indent = lines[at].match(/^\s*/)?.[0] ?? "";
  lines.splice(at + 1, 0, indent + line);
  return lines.join("\n");
}

export type DoctrineChoice =
  | { kind: "grand"; from?: string; to: string }
  | { kind: "sub"; from?: string; to: string };

export function doctrinePatch(
  row: Pick<HistoryFileRow, "raw_script" | "date_blocks">,
  choice: DoctrineChoice,
): Partial<HistoryFileRow> | null {
  if (choice.from === choice.to) return null;
  const raw = row.raw_script ?? "";
  const blocks = row.date_blocks ?? [];
  if (!choice.from) {
    return {
      raw_script:
        choice.kind === "grand"
          ? addCommand(raw, "set_grand_doctrine", choice.to)
          : addCommand(raw, "set_sub_doctrine", choice.to, [
              "set_grand_doctrine",
            ]),
    };
  }
  const swap = (script: string) =>
    choice.kind === "grand"
      ? swapCommand(
          script,
          "set_grand_doctrine",
          choice.from as string,
          choice.to,
        )
      : swapCommand(
          swapCommand(
            script,
            "set_sub_doctrine",
            choice.from as string,
            choice.to,
          ),
          "sub_doctrine",
          choice.from as string,
          choice.to,
        );
  const patch: Partial<HistoryFileRow> = {};
  const nextRaw = swap(raw);
  if (nextRaw !== raw) patch.raw_script = nextRaw;
  const nextBlocks = blocks.map((block) => ({
    ...block,
    script: swap(block.script),
  }));
  if (nextBlocks.some((block, i) => block.script !== blocks[i].script))
    patch.date_blocks = nextBlocks;
  return Object.keys(patch).length ? patch : null;
}
