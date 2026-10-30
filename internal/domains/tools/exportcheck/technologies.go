package exportcheck

import (
	"encoding/json"
)

type Technology struct {
	ID                      int64             `json:"id"`
	ProjectID               int64             `json:"project_id"`
	TechID                  string            `json:"tech_id"`
	CountryTag              string            `json:"country_tag"`
	Name                    string            `json:"name"`
	TechLine                string            `json:"tech_line"`
	ResearchCost            float64           `json:"research_cost"`
	StartYear               int               `json:"start_year"`
	Folder                  string            `json:"folder"`
	PositionX               float64           `json:"position_x"`
	PositionY               float64           `json:"position_y"`
	PositionYIsExplicit     *bool             `json:"position_y_is_explicit"`
	FolderPlacements        []map[string]any  `json:"folder_placements"`
	Prerequisites           []string          `json:"prerequisites"`
	Categories              []string          `json:"categories"`
	Modifiers               map[string]any    `json:"modifiers"`
	AIWillDo                string            `json:"ai_will_do"`
	RawExtras               map[string]any    `json:"raw_extras"`
	Source                  string            `json:"source"`
	IsBanner                bool              `json:"is_banner"`
	ForceUseSmallTechLayout bool              `json:"force_use_small_tech_layout"`
	NameEn                  string            `json:"name_en"`
	NameZh                  string            `json:"name_zh"`
	DescEn                  string            `json:"desc_en"`
	DescZh                  string            `json:"desc_zh"`
	DisplayNameLocs         map[string]string `json:"display_name_locs"`
	DescriptionLocs         map[string]string `json:"description_locs"`
	EnableEquipments        []string          `json:"enable_equipments"`
	EnableSubunits          []string          `json:"enable_subunits"`
	EnableEquipmentModules  []string          `json:"enable_equipment_modules"`
	EnableBuilding          string            `json:"enable_building"`
	EnableTactic            string            `json:"enable_tactic"`
	Xor                     []string          `json:"xor"`
	Dependencies            map[string]any    `json:"dependencies"`
	SubTechnologies         []string          `json:"sub_technologies"`
	Doctrine                bool              `json:"doctrine"`
	Allow                   string            `json:"allow"`
	AllowBranch             string            `json:"allow_branch"`
	OnResearchComplete      string            `json:"on_research_complete"`
	OnResearchCompleteLimit string            `json:"on_research_complete_limit"`
	Path                    []map[string]any  `json:"path"`
	XpResearchType          string            `json:"xp_research_type"`
	XpBoostCost             int               `json:"xp_boost_cost"`
	XpResearchBonus         map[string]any    `json:"xp_research_bonus"`
	Warnings                []string          `json:"warnings"`
}

func folderPlacementFloat(value any) float64 {
	switch v := value.(type) {
	case float64:
		return v
	case float32:
		return float64(v)
	case int:
		return float64(v)
	case int64:
		return float64(v)
	case int32:
		return float64(v)
	case json.Number:
		f, _ := v.Float64()
		return f
	default:
		return 0
	}
}
