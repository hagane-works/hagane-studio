package exportcheck

type Country struct {
	ID                  int64                        `json:"id"`
	ProjectID           int64                        `json:"project_id"`
	Tag                 string                       `json:"tag"`
	Name                string                       `json:"name"`
	Color               []int64                      `json:"color"`
	ColorUI             []int64                      `json:"color_ui"`
	Ideology            string                       `json:"ideology"`
	GraphicalCulture    string                       `json:"graphical_culture"`
	GraphicalCulture2D  string                       `json:"graphical_culture_2d"`
	UseLegacyAIPPSpend  *bool                        `json:"use_legacy_ai_pp_spend"`
	NameEn              string                       `json:"name_en"`
	NameZh              string                       `json:"name_zh"`
	AdjectiveEn         string                       `json:"adjective_en"`
	AdjectiveZh         string                       `json:"adjective_zh"`
	FlagImagePath       string                       `json:"flag_image_path"`
	FlagPathsByIdeology map[string]string            `json:"flag_paths_by_ideology"`
	NameDemocraticEn    string                       `json:"name_democratic_en"`
	NameDemocraticZh    string                       `json:"name_democratic_zh"`
	NameFascismEn       string                       `json:"name_fascism_en"`
	NameFascismZh       string                       `json:"name_fascism_zh"`
	NameCommunismEn     string                       `json:"name_communism_en"`
	NameCommunismZh     string                       `json:"name_communism_zh"`
	NameNeutralityEn    string                       `json:"name_neutrality_en"`
	NameNeutralityZh    string                       `json:"name_neutrality_zh"`
	DisplayNameLocs     map[string]string            `json:"display_name_locs"`
	AdjectiveLocs       map[string]string            `json:"adjective_locs"`
	IdeologyNameLocs    map[string]map[string]string `json:"ideology_name_locs"`
	AutoAdded           bool                         `json:"auto_added"`
	Warning             *string                      `json:"warning"`
}
