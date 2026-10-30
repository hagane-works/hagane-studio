import type { Locale } from "@/lib/store/localeStore";
import type { UnitNames } from "@/lib/vanillaUnitNames";

export const equipmentStatGlossZh: Record<string, string> = {
  soft_attack: "软攻",
  hard_attack: "硬攻",
  ap_attack: "穿甲",
  air_attack: "对空攻击",
  breakthrough: "突破",
  defense: "防御",
  armor_value: "装甲",
  hardness: "装甲率",
  max_strength: "最大强度",
  build_cost_ic: "生产花费",
  manpower: "人力",
  reliability: "可靠性",
  maximum_speed: "最大速度",
  naval_speed: "航速",
  fire_range: "射程",
  lg_armor_piercing: "轻型火炮穿甲",
  lg_attack: "轻型火炮攻击",
  hg_armor_piercing: "重型火炮穿甲",
  hg_attack: "重型火炮攻击",
  torpedo_attack: "鱼雷攻击",
  anti_air_attack: "防空",
  shore_bombardment: "对岸炮击",
  evasion: "规避",
  surface_detection: "对海探测",
  sub_attack: "反潜攻击",
  sub_detection: "对潜探测",
  surface_visibility: "水面可见度",
  sub_visibility: "潜航可见度",
  naval_range: "最大航程",
  port_capacity_usage: "港口容量占用",
  search_and_destroy_coordination: "搜索歼灭协同",
  convoy_raiding_coordination: "破交协同",
  air_defence: "空中防御",
  air_range: "航程",
  air_agility: "机动",
  air_ground_attack: "对地攻击",
  air_bombing: "战略轰炸",
  air_superiority: "空优",
  naval_strike_attack: "对海攻击",
  naval_strike_targetting: "对海瞄准",
  carrier_size: "航母占位",
  default_carrier_composition_weight: "航母编成权重",
};

export const equipmentTypeGlossZh: Record<string, string> = {
  infantry: "步兵",
  motorized: "摩托化",
  mechanized: "机械化",
  armor: "装甲",
  artillery: "火炮",
  anti_air: "防空",
  anti_tank: "反坦克",
  rocket: "火箭炮",
  support_equipment: "支援装备",
  light_tank_chassis: "轻型坦克",
  medium_tank_chassis: "中型坦克",
  heavy_tank_chassis: "重型坦克",
  ship: "舰船",
  submarine: "潜艇",
  convoy: "运输船",
  fighter: "战斗机",
  cas: "近距支援机",
  naval_bomber: "海军轰炸机",
  tactical_bomber: "战术轰炸机",
  strategic_bomber: "战略轰炸机",
  scout_plane: "侦察机",
};

export function equipmentTokenLabel(
  token: string,
  locale: Locale,
  names: UnitNames,
): string {
  if (locale === "en") return token;
  const gloss =
    locale === "zh"
      ? (equipmentStatGlossZh[token] ??
        names[token]?.zh ??
        equipmentTypeGlossZh[token])
      : names[token]?.[locale];
  return gloss ? `${token} · ${gloss}` : token;
}
