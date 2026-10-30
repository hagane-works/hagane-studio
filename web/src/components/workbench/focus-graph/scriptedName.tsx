import { Fragment, type ReactElement } from "react";
import type { ContentLocale } from "@/lib/store/contentLocaleStore";

type Word = { en: string; zh: string };

const SCOPE_WORDS: Record<string, Word> = {
  ROOT: { en: "country", zh: "国家" },
  FROM: { en: "other country", zh: "他国" },
  OWNER: { en: "owner", zh: "拥有者" },
  CONTROLLER: { en: "controller", zh: "控制者" },
  CAPITAL: { en: "capital", zh: "首都" },
  THIS: { en: "current", zh: "当前" },
  PREV: { en: "previous", zh: "上一个" },
};

const METHOD_WORDS: Record<string, Word> = {
  Adjective: { en: "adjective", zh: "形容词" },
  Name: { en: "name", zh: "名称" },
  NameDefCap: { en: "name", zh: "名称" },
  NameWithFlag: { en: "name", zh: "名称" },
  ShortName: { en: "name", zh: "名称" },
  OldName: { en: "old name", zh: "旧称" },
  Leader: { en: "leader", zh: "领导人" },
  LeaderName: { en: "leader", zh: "领导人" },
  Ruler: { en: "leader", zh: "领导人" },
  Tag: { en: "TAG", zh: "TAG" },
  Flag: { en: "flag", zh: "旗帜" },
  PartyName: { en: "party name", zh: "政党名" },
  RulingParty: { en: "ruling party", zh: "执政党" },
  Capital: { en: "capital", zh: "首都" },
  CapitalName: { en: "capital", zh: "首都" },
  Manpower: { en: "manpower", zh: "人力" },
  Stability: { en: "stability", zh: "稳定度" },
  WarSupport: { en: "war support", zh: "战争支持度" },
  Ideology: { en: "ideology", zh: "意识形态" },
  IdeologyName: { en: "ideology", zh: "意识形态" },
  AllianceName: { en: "alliance name", zh: "同盟名" },
};

const SCRIPTED_HINT: Word = {
  en: "Scripted name — the game fills this in automatically",
  zh: "脚本生成名 —— 游戏内会自动填入实际文字",
};

const SCRIPTED_TOKEN_RE =
  /\[(([A-Z][A-Z0-9_]*)\.([A-Za-z0-9_]+)(?::[^\]]+)?)\]/g;

function camelToWords(method: string): string {
  return method.replace(/([a-z0-9])([A-Z])/g, "$1 $2").toLowerCase();
}

export function readableScriptedToken(
  inner: string,
  locale: ContentLocale,
): string {
  const m = inner.match(/^([A-Z][A-Z0-9_]*)\.([A-Za-z0-9_]+)/);
  if (!m) return `[${inner}]`;
  const [, scope, rawMethod] = m;
  const method = rawMethod.replace(/^Get/, "");
  const bilingualLocale = locale === "zh" ? "zh" : "en";
  const scopeWord = SCOPE_WORDS[scope]?.[bilingualLocale];
  const methodWord =
    METHOD_WORDS[method]?.[bilingualLocale] ??
    (locale === "zh" ? "" : camelToWords(method));
  if (!scopeWord || !methodWord) return `[${inner}]`;
  return locale === "zh"
    ? `[${scopeWord}${methodWord}]`
    : `[${scopeWord} ${methodWord}]`;
}

export interface ScriptedNameSegment {
  text: string;

  scripted: boolean;

  raw?: string;
}

export function splitScriptedName(
  name: string,
  locale: ContentLocale,
): ScriptedNameSegment[] | null {
  SCRIPTED_TOKEN_RE.lastIndex = 0;
  if (!SCRIPTED_TOKEN_RE.test(name)) return null;
  SCRIPTED_TOKEN_RE.lastIndex = 0;
  const segments: ScriptedNameSegment[] = [];
  let cursor = 0;
  let match: RegExpExecArray | null;
  while ((match = SCRIPTED_TOKEN_RE.exec(name)) !== null) {
    if (match.index > cursor) {
      segments.push({ text: name.slice(cursor, match.index), scripted: false });
    }
    segments.push({
      text: readableScriptedToken(match[1], locale),
      scripted: true,
      raw: match[0],
    });
    cursor = match.index + match[0].length;
  }
  if (cursor < name.length) {
    segments.push({ text: name.slice(cursor), scripted: false });
  }
  return segments;
}

export function ReadableScriptedName({
  name,
  locale,
}: {
  name: string;
  locale: ContentLocale;
}): ReactElement {
  const segments = splitScriptedName(name, locale);
  if (!segments) return <>{name}</>;
  return (
    <>
      {segments.map((seg, i) =>
        seg.scripted ? (
          <span
            key={i}
            className="text-muted-foreground"
            title={`${SCRIPTED_HINT[locale === "zh" ? "zh" : "en"]} · ${seg.raw}`}
          >
            {seg.text}
          </span>
        ) : (
          <Fragment key={i}>{seg.text}</Fragment>
        ),
      )}
    </>
  );
}
