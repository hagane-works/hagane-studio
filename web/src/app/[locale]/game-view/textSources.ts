import type {
  GuiBookmark,
  GuiCharacter,
  GuiCountryCard,
  GuiNode,
  GuiPopularity,
  GuiWindow,
} from "@/lib/guiRenderApi";
import { matchNamePath, RUNTIME_TEXT } from "./gameNavigation";
import type { PrintedText, TextScope, TextSource } from "./editModel";

function idOf(node: GuiNode): string {
  return (node as GuiNode & { _id?: string })._id ?? "";
}

export function sourceAt(
  out: Record<string, TextSource>,
  win: GuiWindow,
  namePath: string,
  source: TextSource | null | undefined,
): void {
  if (!source) return;
  for (const node of matchNamePath(win.node, namePath)) {
    const id = idOf(node);
    if (id) out[id] = source;
  }
}

const LOC_KEY = /^[A-Za-z0-9_.-]{1,512}$/;

export function keySource(
  key: string | null | undefined,
  scope: TextScope,
): TextSource | null {
  const trimmed = (key ?? "").trim();
  return LOC_KEY.test(trimmed) ? { kind: "key", key: trimmed, scope } : null;
}

export function fixedTextSources(win: GuiWindow): Record<string, TextSource> {
  const out: Record<string, TextSource> = {};
  for (const [namePath, key] of Object.entries(RUNTIME_TEXT[win.name] ?? {})) {
    sourceAt(out, win, namePath, keySource(key, "everywhere"));
  }
  return out;
}

export function partyNamePrinted(
  row: GuiNode,
  party: Pick<GuiPopularity, "party" | "label" | "label_key">,
): Record<string, PrintedText> {
  const out: Record<string, PrintedText> = {};
  const source = keySource(party.label_key, "country");
  for (const child of row.children ?? []) {
    const id = idOf(child);
    if (id && child.name === "name")
      out[id] = {
        text: party.label || party.party,
        ...(source ? { source } : {}),
      };
  }
  return out;
}

function printedAt(
  out: Record<string, PrintedText>,
  root: GuiNode,
  name: string,
  text: string | null | undefined,
  source: TextSource | null | undefined,
): void {
  if (!text) return;
  const walk = (node: GuiNode) => {
    const id = idOf(node);
    if (id && node.name === name)
      out[id] = { text, ...(source ? { source } : {}) };
    for (const child of node.children ?? []) walk(child);
  };
  walk(root);
}

export function namedPrinted(
  row: GuiNode,
  name: string,
  text: string | null | undefined,
  key: string | null | undefined,
  scope: TextScope = "everywhere",
): Record<string, PrintedText> {
  const out: Record<string, PrintedText> = {};
  printedAt(out, row, name, text, keySource(key, scope));
  return out;
}

export function personPrinted(
  row: GuiNode,
  name: string,
  person: { token?: string; name?: string } | null | undefined,
): Record<string, PrintedText> {
  const out: Record<string, PrintedText> = {};
  printedAt(
    out,
    row,
    name,
    person?.name,
    person?.token ? { kind: "person", token: person.token } : null,
  );
  return out;
}

export function countryCardPrinted(
  card: GuiNode,
  country: Pick<GuiCountryCard, "name" | "name_key">,
): Record<string, PrintedText> {
  const out: Record<string, PrintedText> = {};
  printedAt(
    out,
    card,
    "country_name",
    country.name,
    keySource(country.name_key, "country"),
  );
  return out;
}

export function countryPickPrinted(
  win: GuiWindow,
  country: GuiCountryCard | undefined,
  party: string | undefined,
): Record<string, PrintedText> {
  const out: Record<string, PrintedText> = {};
  if (!country) return out;
  printedAt(
    out,
    win.node,
    "nation_title",
    country.name,
    keySource(country.name_key, "country"),
  );
  printedAt(
    out,
    win.node,
    "leader",
    country.leader?.name,
    country.leader?.token
      ? { kind: "person", token: country.leader.token }
      : null,
  );

  const ideology = keySource(country.ruling_party, "allCountries");
  printedAt(
    out,
    win.node,
    "ideology",
    country.ideology_label,
    keySource(country.ideology, "allCountries"),
  );
  printedAt(out, win.node, "government", country.ruling_party_label, ideology);
  printedAt(out, win.node, "party", party, ideology);
  printedAt(
    out,
    win.node,
    "history",
    country.history,
    keySource(country.history_key, "country"),
  );
  return out;
}

export function scenarioPrinted(
  win: GuiWindow,
  mark: GuiBookmark | undefined,
): Record<string, PrintedText> {
  const out: Record<string, PrintedText> = {};
  if (!mark) return out;
  printedAt(
    out,
    win.node,
    "bookmark_title",
    mark.label || mark.name,
    keySource(mark.name, "everywhere"),
  );
  printedAt(
    out,
    win.node,
    "bookmark_desc",
    mark.desc,
    keySource(mark.desc_key, "everywhere"),
  );
  return out;
}

export function eventPrinted(
  win: GuiWindow,
  event:
    | { title?: string; desc?: string; title_key?: string; desc_key?: string }
    | null
    | undefined,
): Record<string, PrintedText> {
  const out: Record<string, PrintedText> = {};
  if (!event) return out;
  printedAt(
    out,
    win.node,
    "Title",
    event.title,
    keySource(event.title_key, "everywhere"),
  );
  printedAt(
    out,
    win.node,
    "Description",
    event.desc,
    keySource(event.desc_key, "everywhere"),
  );
  return out;
}

export function eventOptionPrinted(
  row: GuiNode,
  option: { name: string; label?: string },
): Record<string, PrintedText> {
  const out: Record<string, PrintedText> = {};
  printedAt(
    out,
    row,
    "Name",
    option.label || option.name,
    keySource(option.name, "everywhere"),
  );
  return out;
}

export function statePrinted(
  win: GuiWindow,
  state: { id: number; name?: string } | null | undefined,
): Record<string, PrintedText> {
  const out: Record<string, PrintedText> = {};
  if (!state) return out;
  printedAt(
    out,
    win.node,
    "state_name",
    state.name || `#${state.id}`,
    keySource(`STATE_${state.id}`, "everywhere"),
  );
  return out;
}

export function diplomacyHeaderPrinted(
  win: GuiWindow,
  country: { tag: string; name?: string; name_key?: string } | null | undefined,
  leader: { name?: string; token?: string } | null | undefined,
): Record<string, PrintedText> {
  const out: Record<string, PrintedText> = {};
  if (!country) return out;
  printedAt(
    out,
    win.node,
    "country_name",
    country.name || country.tag,
    keySource(country.name_key, "country"),
  );
  printedAt(
    out,
    win.node,
    "leader_name",
    leader?.name,
    leader?.token ? { kind: "person", token: leader.token } : null,
  );
  return out;
}

export function countryRowPrinted(
  row: GuiNode,
  country: { tag: string; name?: string; name_key?: string },
): Record<string, PrintedText> {
  const out: Record<string, PrintedText> = {};
  printedAt(
    out,
    row,
    "name",
    country.name || country.tag,
    keySource(country.name_key, "country"),
  );
  return out;
}

export function leaderNamePrinted(
  win: GuiWindow,
  leader: Pick<GuiCharacter, "token" | "name"> | null | undefined,
): Record<string, PrintedText> {
  const out: Record<string, PrintedText> = {};
  if (!leader?.token) return out;
  for (const node of matchNamePath(win.node, "leader_name")) {
    const id = idOf(node);
    if (id)
      out[id] = {
        text: leader.name,
        source: { kind: "person", token: leader.token },
      };
  }
  return out;
}

export function pickedText(
  id: string,
  printed: Readonly<Record<string, PrintedText>>,
  overrides: Readonly<Record<string, string>>,
  sources: Readonly<Record<string, TextSource>>,
): { shown: string | undefined; source: TextSource | undefined } {
  const own = printed[id];
  return own
    ? { shown: own.text, source: own.source }
    : { shown: overrides[id], source: sources[id] };
}
