export const HOI4_LANGUAGE_ID = "hoi4";

export const hoi4Monarch: Record<string, unknown> = {
  defaultToken: "",
  tokenPostfix: ".hoi4",

  brackets: [{ open: "{", close: "}", token: "delimiter.curly" }],

  tokenizer: {
    root: [
      [/#.*$/, "comment"],

      [/"/, { token: "string.quote", next: "@string" }],

      [/@[a-zA-Z_]\w*/, "variable.predefined"],

      [/-?\d+\.\d+\.\d+/, "number"],

      [/-?\d+\.\d+/, "number.float"],
      [/-?\d+/, "number"],

      [/[{}]/, "@brackets"],

      [/[=<>]/, "operator"],

      [
        /\b(if|else|else_if|limit|OR|AND|NOT|any_of|all_of|hidden_effect|hidden_trigger|count_triggers|scripted_trigger|scripted_effect|original_tag|tag|has_dlc|has_country_flag|set_country_flag|clr_country_flag|yes|no)\b/,
        "keyword",
      ],

      [/[a-zA-Z_]\w*/, "identifier"],

      [/\s+/, "white"],
    ],

    string: [
      [/[^\\"]+/, "string"],
      [/\\./, "string.escape"],
      [/"/, { token: "string.quote", next: "@pop" }],
    ],
  },
};

export const hoi4LanguageConfig: Record<string, unknown> = {
  comments: {
    lineComment: "#",
  },
  brackets: [["{", "}"]],
  autoClosingPairs: [
    { open: "{", close: "}" },
    { open: '"', close: '"' },
  ],
  surroundingPairs: [
    { open: "{", close: "}" },
    { open: '"', close: '"' },
  ],
};
