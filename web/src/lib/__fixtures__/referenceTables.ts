import type { ReferenceTables } from "@/lib/referenceTables";

export const REFERENCE_TABLES: ReferenceTables = {
  ideologies: {
    groups: ["democratic", "communism", "fascism", "neutrality"],
    sub_ideologies: [
      {
        id: "conservatism",
        group: "democratic",
        name: { en: "Conservatism", zh: "保守主义" },
      },
      {
        id: "liberalism",
        group: "democratic",
        name: { en: "Liberalism", zh: "自由主义" },
      },
      {
        id: "socialism",
        group: "democratic",
        name: { en: "Socialism", zh: "社会主义" },
      },
      {
        id: "populism",
        group: "democratic",
        name: { en: "Populism", zh: "民粹主义" },
      },
      {
        id: "marxism",
        group: "communism",
        name: { en: "Marxism", zh: "马克思主义" },
      },
      {
        id: "leninism",
        group: "communism",
        name: { en: "Leninism", zh: "列宁主义" },
      },
      {
        id: "stalinism",
        group: "communism",
        name: { en: "Stalinism", zh: "斯大林主义" },
      },
      {
        id: "anti_revisionism",
        group: "communism",
        name: { en: "Anti-Revisionism", zh: "反修正主义" },
      },
      {
        id: "anarchist_communism",
        group: "communism",
        name: { en: "Anarchist Communism", zh: "安那其共产主义" },
      },
      {
        id: "buddhist_socialism",
        group: "communism",
        name: { en: "Buddhist Socialism", zh: "佛教社会主义" },
        random_disabled: true,
      },
      {
        id: "nazism",
        group: "fascism",
        name: { en: "Nazism", zh: "纳粹主义" },
      },
      {
        id: "gen_nazism",
        group: "fascism",
        name: { en: "Nazism", zh: "纳粹主义" },
      },
      {
        id: "fascism_ideology",
        group: "fascism",
        name: { en: "Fascism", zh: "法西斯主义" },
      },
      {
        id: "falangism",
        group: "fascism",
        name: { en: "Falangism", zh: "长枪党主义" },
      },
      {
        id: "rexism",
        group: "fascism",
        name: { en: "Rexism", zh: "雷克斯主义" },
      },
      {
        id: "emperor_fascism",
        group: "fascism",
        name: { en: "Emperor Fascism", zh: "皇道法西斯主义" },
        random_disabled: true,
      },
      {
        id: "despotism",
        group: "neutrality",
        name: { en: "Despotic", zh: "专制主义" },
      },
      {
        id: "oligarchism",
        group: "neutrality",
        name: { en: "Oligarchic", zh: "寡头政治" },
      },
      {
        id: "anarchism",
        group: "neutrality",
        name: { en: "Anarchism", zh: "安那其主义" },
      },
      {
        id: "moderatism",
        group: "neutrality",
        name: { en: "Moderatism", zh: "温和主义" },
      },
      {
        id: "centrism",
        group: "neutrality",
        name: { en: "Centrism", zh: "中间主义" },
      },
      {
        id: "japan_militarism_ideology",
        group: "neutrality",
        name: { en: "Militarism", zh: "军国主义" },
        random_disabled: true,
      },
    ],
  },
  text_colors: {
    global: {
      C: "rgb(35, 206, 255)",
      L: "rgb(195, 176, 145)",
      W: "rgb(255, 255, 255)",
      B: "rgb(0, 0, 255)",
      G: "rgb(0, 159, 3)",
      R: "rgb(255, 50, 50)",
      b: "rgb(0, 0, 0)",
      g: "rgb(176, 176, 176)",
      Y: "rgb(255, 189, 0)",
      H: "rgb(255, 189, 0)",
      T: "rgb(255, 255, 255)",
      O: "rgb(255, 112, 25)",
      "0": "rgb(203, 0, 203)",
      "1": "rgb(128, 120, 211)",
      "2": "rgb(81, 112, 243)",
      "3": "rgb(81, 143, 220)",
      "4": "rgb(90, 190, 231)",
      "5": "rgb(63, 181, 194)",
      "6": "rgb(119, 204, 186)",
      "7": "rgb(153, 209, 153)",
      "8": "rgb(204, 163, 51)",
      "9": "rgb(252, 169, 125)",
      t: "rgb(255, 76, 77)",
    },
    fonts: {
      hoi_18mbs: {
        G: "rgb(86, 172, 91)",
        R: "rgb(222, 86, 70)",
        Y: "rgb(238, 201, 35)",
        H: "rgb(238, 201, 35)",
        T: "rgb(255, 255, 255)",
      },
    },
  },
};
