package exportcheck

type Idea struct {
	ID                int64             `json:"id"`
	ProjectID         int64             `json:"project_id"`
	IdeaID            string            `json:"idea_id"`
	CountryTag        string            `json:"country_tag"`
	ScopeKind         string            `json:"scope_kind"`
	SharedCountryTags []string          `json:"shared_country_tags"`
	IdeaType          string            `json:"idea_type"`
	Name              string            `json:"name"`
	Picture           string            `json:"picture"`
	Modifier          map[string]any    `json:"modifier"`
	AIWillDo          string            `json:"ai_will_do"`
	RemovalCost       int               `json:"removal_cost"`
	Allowed           string            `json:"allowed"`
	AllowedCivilWar   string            `json:"allowed_civil_war"`
	IsDesigner        bool              `json:"is_designer"`
	ResearchBonus     map[string]any    `json:"research_bonus"`
	DesignerTraits    []string          `json:"designer_traits"`
	RawExtras         map[string]any    `json:"raw_extras"`
	NameEn            string            `json:"name_en"`
	NameZh            string            `json:"name_zh"`
	DescEn            string            `json:"desc_en"`
	DescZh            string            `json:"desc_zh"`
	DisplayNameLocs   map[string]string `json:"display_name_locs"`
	DescriptionLocs   map[string]string `json:"description_locs"`
	Available         string            `json:"available"`
	Visible           string            `json:"visible"`
	AllowedToRemove   string            `json:"allowed_to_remove"`
	Cancel            string            `json:"cancel"`
	DoEffect          string            `json:"do_effect"`
	OnAdd             string            `json:"on_add"`
	OnRemove          string            `json:"on_remove"`
	CancelIfInvalid   bool              `json:"cancel_if_invalid"`
	TargetedModifier  []map[string]any  `json:"targeted_modifier"`
	EquipmentBonus    map[string]any    `json:"equipment_bonus"`
	Rule              map[string]any    `json:"rule"`
	Level             int               `json:"level"`
	Ledger            string            `json:"ledger"`
	Cost              int               `json:"cost"`
	AutoAttachOnStart bool              `json:"auto_attach_on_start"`
	Source            string            `json:"source"`
	Warnings          []string          `json:"warnings"`
}
