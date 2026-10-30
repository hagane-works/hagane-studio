package exportcheck

type FocusNode struct {
	TreeEditRevision       string            `json:"tree_edit_revision"`
	Diagnostics            *[]SaveDiagnostic `json:"diagnostics,omitempty"`
	TreeDiagnostics        *[]SaveDiagnostic `json:"tree_diagnostics,omitempty"`
	ID                     int64             `json:"id"`
	FocusTreeID            int64             `json:"focus_tree_id"`
	FocusID                string            `json:"focus_id"`
	Name                   string            `json:"name"`
	Icon                   string            `json:"icon"`
	Cost                   float64           `json:"cost"`
	X                      int               `json:"x"`
	Y                      int               `json:"y"`
	RelativePositionID     string            `json:"relative_position_id"`
	Prerequisites          [][]string        `json:"prerequisites"`
	MutuallyExclusive      []string          `json:"mutually_exclusive"`
	CompletionReward       string            `json:"completion_reward"`
	Available              string            `json:"available"`
	Bypass                 string            `json:"bypass"`
	Cancel                 string            `json:"cancel"`
	SelectEffect           string            `json:"select_effect"`
	AiWillDo               string            `json:"ai_will_do"`
	SearchFilters          []string          `json:"search_filters"`
	CancelIfInvalid        bool              `json:"cancel_if_invalid"`
	ContinueIfInvalid      bool              `json:"continue_if_invalid"`
	AvailableIfCapitulated bool              `json:"available_if_capitulated"`
	WillLeadToWarWith      []string          `json:"will_lead_to_war_with"`
	AllowBranchAst         map[string]any    `json:"allow_branch_ast"`
	LoadFocusTreeRules     []map[string]any  `json:"load_focus_tree_rules"`
	NameEn                 string            `json:"name_en"`
	NameZh                 string            `json:"name_zh"`
	DescEn                 string            `json:"desc_en"`
	DescZh                 string            `json:"desc_zh"`
	DisplayNameLocs        map[string]string `json:"display_name_locs"`
	DescriptionLocs        map[string]string `json:"description_locs"`
	EffectTooltipLocs      map[string]string `json:"effect_tooltip_locs"`
	RawExtras              map[string]any    `json:"raw_extras"`
	Source                 string            `json:"source"`
	Offset                 []map[string]any  `json:"offset"`
	DynamicIcon            []map[string]any  `json:"dynamic_icon"`
	TextIcon               string            `json:"text_icon"`
	Warnings               []string          `json:"warnings"`

	sourceSnapshot map[string]any
}
