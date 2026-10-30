import type { GuiCountryStats, GuiNode } from "@/lib/guiRenderApi";
import { textColors } from "@/lib/referenceTables";
import {
  shortcutMarkup,
  type GameTooltipContent,
} from "../gui-renderer/tooltipText";
import { HAS_INTERNATIONAL_MARKET } from "./gameNavigation";
import { SHORT_MONTHS, decisionsTakeable } from "./topbarStats";

export const TOPBAR_BUTTON_KEYS: Readonly<
  Record<string, readonly [string, string]>
> = {
  player_flag: ["TOPBAR_POLITICS", "TOPBAR_POLITICS_DESC"],
  decisionview_button: ["TOPBAR_DECISIONVIEW", "TOPBAR_DECISIONVIEW_DESC"],
  intel_agency_button: ["TOPBAR_INTELLIGENCE", "TOPBAR_INTELLIGENCE_DESC"],
  technology_button: ["TOPBAR_TECHNOLOGY", "TOPBAR_TECHNOLOGY_DESC"],
  diplomacy_button: HAS_INTERNATIONAL_MARKET
    ? ["TOPBAR_INTERNATIONAL_MARKET", "TOPBAR_INTERNATIONAL_MARKET_DESC"]
    : ["TOPBAR_DIPLOMACY", "TOPBAR_DIPLOMACY_DESC"],
  trade_button: ["TOPBAR_TRADE", "TOPBAR_TRADE_DESC"],
  construction_button: ["TOPBAR_CONSTRUCTION", "TOPBAR_CONSTRUCTION_DESC"],
  production_button: ["TOPBAR_PRODUCTION", "TOPBAR_PRODUCTION_DESC"],
  deployment_button: ["TOPBAR_DEPLOYMENT", "TOPBAR_DEPLOYMENT_DESC"],
  logistics_button: ["TOPBAR_LOGISTICS", "TOPBAR_LOGISTICS_DESC"],
  officer_corp_button: ["TOPBAR_OFFICER_CORP", "TOPBAR_OFFICER_CORP_DESC"],
  army_button: ["TOPBAR_ARMY", "TOPBAR_ARMY_DESC"],
  navy_button: ["TOPBAR_NAVY", "TOPBAR_NAVY_DESC"],
  air_button: ["TOPBAR_AIR", "TOPBAR_AIR_DESC"],
};

const RESOURCE_BOXES: readonly {
  nodes: readonly string[];
  keys: readonly [string, string];
  id: string;
}[] = [
  {
    id: "pp",
    nodes: ["pol_power_value_bg", "pol_power_icon", "pol_power"],
    keys: ["POLITICAL_POWER", "POLITICAL_POWER_DESC"],
  },

  {
    id: "stability",
    nodes: ["sability_value_bg", "stability_icon", "stability"],
    keys: ["STABILITY", "STABILITY_DESC"],
  },
  {
    id: "war_support",
    nodes: ["war_support_value_bg", "war_support_icon", "war_support"],
    keys: ["WAR_SUPPORT", "WAR_SUPPORT_DESC"],
  },
  {
    id: "manpower",
    nodes: ["manpower_value_bg", "manpower_icon", "manpower"],
    keys: ["MANPOWER", "MANPOWER_DESC"],
  },
  {
    id: "factories",
    nodes: [
      "industrial_capacity_bg",
      "industrial_capacity_icon",
      "industrial_capacity",
    ],
    keys: ["INDUSTRIAL_CAPACITY", "INDUSTRIAL_CAPACITY_DESC"],
  },
  {
    id: "fuel",
    nodes: ["fuel_value_bg", "fuel_icon", "fuel_value"],
    keys: ["FUEL_TITLE", "FUEL_DESC"],
  },

  {
    id: "supply",
    nodes: ["supply_value_bg", "supply_icon", "supply_value"],
    keys: ["EQUIPMENT_IN_USE", "EQUIPMENT_IN_USE_DESC"],
  },
  {
    id: "convoys",
    nodes: ["convoy_text_bg", "convoys_icon", "convoys_count"],
    keys: ["CONVOYS", "CONVOYS_USAGE_DELAYED"],
  },

  { id: "cp", nodes: ["cp"], keys: ["COMMAND_POWER", "COMMAND_POWER_DESC"] },
  {
    id: "army_xp",
    nodes: ["land"],
    keys: ["ARMY_EXPERIENCE", "ARMY_EXPERIENCE_DESC"],
  },
  {
    id: "navy_xp",
    nodes: ["navy"],
    keys: ["NAVY_EXPERIENCE", "NAVY_EXPERIENCE_DESC"],
  },
  {
    id: "air_xp",
    nodes: ["air"],
    keys: ["AIR_EXPERIENCE", "AIR_EXPERIENCE_DESC"],
  },
];

export const TOPBAR_LOC_KEYS: readonly string[] = [
  ...new Set([
    ...Object.values(TOPBAR_BUTTON_KEYS).flat(),
    ...RESOURCE_BOXES.flatMap((box) => box.keys),
    "SEPARATION_LINE",
    "SHORTCUT",
    "TOPBAR_DECISIONS_CAN_TAKE_DESC",
    "TOPBAR_TECHNOLOGY_LEFT_CLICK",
    "TOPBAR_TECHNOLOGY_RIGHT_CLICK",
    "FACTION_BUTTON_NAME",
    "FACTION_BUTTON_FACTION_NAME",
    "FACTION_BUTTON_DESC",
    "FACTION_BUTTON_INITIATIVE",
    "FACTION_BUTTON_NO_FACTION",
    "FACTION_BUTTON_JOIN_FACTION",
    "CURRENT_COMMAND_POWER_TOPBAR",
    "CURRENT_FUEL",
    "FUEL_CAPACITY",
    "FUEL_CAPACITY_FILL_INFO",
    "DURATION_DAYS",
    "THREAT_TOOLTIP",
    "DISMISSED_ALERTS_MENU",
    "DISMISSED_ALERTS_NONE",
    "ACHIEVEMENTS",
    "ACHIEVEMENTS_VIEW_LOCKED",
    "DAY_UNIT_SHORT",
    "YEAR_UNIT_SHORT",
    "FUEL_STOCKPILE_MAX_SHORT",
    "FUEL_STOCKPILE_EMPTY_SHORT",
    ...SHORT_MONTHS,
  ]),
];

const FMT_CHARS = "[A-Za-z0-9%=+\\-.^*]";

const CLEANED_SLOT = new RegExp(
  `(?<![$\\w])([A-Z][A-Z0-9_]*)\\|(${FMT_CHARS}+)(?![$\\w])`,
  "g",
);

const CLEANED_NAME = /(?<![$\w|])([A-Z][A-Z0-9]*(?:_[A-Z0-9]+)+)(?![\w|$])/g;
const SLOT = /\$([A-Za-z0-9_]+)(?:\|([^$\n]*))?\$/g;

const CLEANED_SCRIPTED = /\{([A-Za-z_][\w.]*)\}/g;

export function formatLocValue(value: number | string, fmt = ""): string {
  let decimals: number | null = null;
  let percent = false;
  let sign = false;
  let color: string | null = null;
  for (const ch of fmt) {
    if (/[0-9]/.test(ch)) decimals = Number(ch);
    else if (ch === "%") percent = true;
    else if (ch === "+") sign = true;
    else if (/[A-Za-z]/.test(ch) && textColors()[ch]) color = ch;
  }
  let text: string;
  if (typeof value === "number") {
    const n = percent ? value * 100 : value;
    if (decimals !== null) text = n.toFixed(decimals);
    else
      text = Number.isInteger(n)
        ? String(n)
        : String(Math.round(n * 100) / 100);
    if (percent) text += "%";
    if (sign && n > 0) text = `+${text}`;
  } else {
    text = value;
  }
  return color ? `§${color}${text}§!` : text;
}

export function fillLoc(
  text: string,
  vars: Readonly<Record<string, number | string>> = {},
): string {
  let out = text;
  if (!out.includes("$")) {
    out = out.replace(
      CLEANED_SLOT,
      (_m, name: string, fmt: string) => `$${name}|${fmt}$`,
    );
    out = out.replace(CLEANED_NAME, (_m, name: string) => `$${name}$`);
    out = out.replace(CLEANED_SCRIPTED, (_m, call: string) => `[${call}]`);
    for (const name of Object.keys(vars)) {
      out = out.replace(
        new RegExp(`(?<![$\\w|])${name}(?![\\w|$])`, "g"),
        `$${name}$`,
      );
    }
  }
  return out.replace(SLOT, (match, name: string, fmt: string | undefined) => {
    const value = vars[name];
    return value === undefined ? match : formatLocValue(value, fmt ?? "");
  });
}

export type Loc = Readonly<Record<string, string>>;

function say(
  loc: Loc,
  key: string,
  vars: Readonly<Record<string, number | string>> = {},
  fallback = "",
): string {
  const raw = loc[key] ?? fallback;
  return raw ? fillLoc(raw, vars) : "";
}

function trimNewlines(text: string): string {
  return text.replace(/\n+$/, "");
}

function header(text: string): string {
  return text.startsWith("§") ? text : `§H${text}§!`;
}

function withHeaderLine(text: string): string {
  if (text.includes("§")) return text;
  const at = text.indexOf("\n");
  return at < 0
    ? header(text)
    : `${header(text.slice(0, at))}${text.slice(at)}`;
}

export function composeButtonTooltip(
  title: string,
  desc: string,
  sections: readonly string[],
  separator: string,
  shortcutLine: string | null,
): string {
  let out = `${header(title)}\n`;

  if (desc) out += `${desc}\n`;
  const shown = sections.map(trimNewlines).filter(Boolean);

  if (shown.length)
    out += `${out.endsWith("\n\n") ? "" : "\n"}${shown.map((section) => `${section}\n`).join("")}`;
  if (shortcutLine)
    out += `${trimNewlines(separator) || "--------------"}\n${shortcutLine}`;
  return trimNewlines(out);
}

interface ShortcutLine {
  line: string;
  icons: Record<string, string>;
}

function shortcutOf(node: GuiNode, loc: Loc): ShortcutLine | null {
  const raw = node.raw?.shortcut;
  if (typeof raw !== "string" || !raw.trim()) return null;
  const keys = shortcutMarkup(raw.trim());
  return {
    line: say(loc, "SHORTCUT", { KEY: keys.text }, "Shortcut: $KEY$"),
    icons: keys.icons,
  };
}

export function decisionsSection(
  stats: GuiCountryStats | undefined,
  loc: Loc,
): string | null {
  const count = decisionsTakeable(stats);
  if (!count) return null;
  const names = (stats?.decisions_takeable ?? []).map(
    (decision) => decision.name || decision.id,
  );
  return say(
    loc,
    "TOPBAR_DECISIONS_CAN_TAKE_DESC",
    { AMOUNT: count, LIST: names.join("\n") },
    "§YDecisions possible to take§! ($AMOUNT|G$)\n$LIST$\n",
  );
}

export function buttonTooltip(
  node: GuiNode,
  stats: GuiCountryStats | undefined,
  loc: Loc,
): GameTooltipContent | null {
  const name = node.name ?? "";
  const shortcut = shortcutOf(node, loc);
  if (name === "faction_button") return factionTooltip(stats, loc, shortcut);
  const keys = TOPBAR_BUTTON_KEYS[name];
  if (!keys) return null;
  const title = say(loc, keys[0]);
  if (!title) return null;
  const sections: string[] = [];
  if (name === "decisionview_button") {
    const decisions = decisionsSection(stats, loc);
    if (decisions) sections.push(decisions);
  }
  if (name === "technology_button") {
    const clicks = [
      say(loc, "TOPBAR_TECHNOLOGY_LEFT_CLICK"),
      say(loc, "TOPBAR_TECHNOLOGY_RIGHT_CLICK"),
    ].filter(Boolean);
    if (clicks.length) sections.push(clicks.join("\n"));
  }
  const text = composeButtonTooltip(
    title,
    say(loc, keys[1]),
    sections,
    say(loc, "SEPARATION_LINE"),
    shortcut?.line ?? null,
  );
  return { text, icons: shortcut?.icons };
}

export function factionTooltip(
  stats: GuiCountryStats | undefined,
  loc: Loc,
  shortcut: ShortcutLine | null,
): GameTooltipContent | null {
  const title = say(loc, "FACTION_BUTTON_NAME");
  if (!title) return null;
  const lines: string[] = [];
  const faction = stats?.faction;
  if (faction) {
    lines.push(
      say(
        loc,
        "FACTION_BUTTON_FACTION_NAME",
        { NAME: faction.name },
        "$NAME|H$",
      ),
    );
    lines.push(say(loc, "FACTION_BUTTON_DESC"));
    lines.push(
      say(loc, "FACTION_BUTTON_INITIATIVE", {
        VAL: Math.floor(faction.initiative ?? 0),
      }),
    );
  } else if (stats) {
    lines.push(
      say(loc, "FACTION_BUTTON_NO_FACTION"),
      say(loc, "FACTION_BUTTON_JOIN_FACTION"),
    );
  }
  const text = composeButtonTooltip(
    title,
    lines.filter(Boolean).join("\n"),
    [],
    say(loc, "SEPARATION_LINE"),
    shortcut?.line ?? null,
  );
  return { text, icons: shortcut?.icons };
}

function resourceExtras(
  id: string,
  stats: GuiCountryStats | undefined,
  loc: Loc,
): string[] {
  if (!stats) return [];
  if (id === "cp")
    return [
      say(loc, "CURRENT_COMMAND_POWER_TOPBAR", {
        VALUE: stats.command_power ?? 0,
      }),
    ];
  if (id === "fuel" && stats.fuel) {
    const { capacity, ratio, days, gaining } = stats.fuel;
    const out = [
      say(loc, "CURRENT_FUEL", { NUM: Math.round(capacity * ratio) }),
      say(loc, "FUEL_CAPACITY", { NUM: Math.round(capacity) }),
    ];
    if (gaining && days !== null && days !== undefined && ratio < 1) {
      out.push(
        say(loc, "FUEL_CAPACITY_FILL_INFO", {
          DUR: say(loc, "DURATION_DAYS", { NUM: days }, "$NUM|.0$ days"),
        }),
      );
    }
    return out;
  }
  return [];
}

export function topbarTooltips(
  root: GuiNode,
  stats: GuiCountryStats | undefined,
  loc: Loc,
): Map<string, GameTooltipContent | null> {
  const out = new Map<string, GameTooltipContent | null>();
  const walk = (node: GuiNode) => {
    const name = node.name ?? "";
    if (name && (TOPBAR_BUTTON_KEYS[name] || name === "faction_button"))
      out.set(name, buttonTooltip(node, stats, loc));
    for (const child of node.children ?? []) walk(child);
  };
  walk(root);
  for (const box of RESOURCE_BOXES) {
    const [titleKey, descKey] = box.keys;
    const title = say(loc, titleKey);
    if (!title) continue;
    const content: GameTooltipContent =
      box.id === "convoys"
        ? { text: header(title), delayed: say(loc, descKey) }
        : {
            text: [
              header(title),
              say(loc, descKey),
              ...resourceExtras(box.id, stats, loc),
            ]
              .filter(Boolean)
              .join("\n"),
          };
    for (const name of box.nodes) out.set(name, content);
  }

  if (out.has("faction_button"))
    out.set("initiative_text", out.get("faction_button") ?? null);
  if (out.has("decisionview_button"))
    out.set(
      "decisionview_amount_to_take_items",
      out.get("decisionview_button") ?? null,
    );

  if (loc.THREAT_TOOLTIP && stats?.world_tension !== undefined) {
    const threat = {
      text: fillLoc(withHeaderLine(loc.THREAT_TOOLTIP), {
        VAL: stats.world_tension / 100,
      }),
    };
    out.set("threat_button", threat);
    out.set("threat_value_button", threat);
  }

  const dismissed = say(loc, "DISMISSED_ALERTS_MENU");
  if (dismissed) {
    out.set("dismissed_alerts_button", {
      text: [header(dismissed), say(loc, "DISMISSED_ALERTS_NONE")]
        .filter(Boolean)
        .join("\n"),
    });
  }
  const achievements = say(loc, "ACHIEVEMENTS");
  if (achievements) {
    out.set("achievements_button", {
      text: [header(achievements), say(loc, "ACHIEVEMENTS_VIEW_LOCKED")]
        .filter(Boolean)
        .join("\n"),
    });
  }
  return out;
}
