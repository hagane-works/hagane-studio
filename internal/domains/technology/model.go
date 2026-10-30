package technology

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"haganeworks/internal/project"
	"haganeworks/internal/workbench"
)

const table = "technologies"

var techLineFolders = map[string]string{
	"infantry": "infantry_folder", "support": "support_folder",
	"artillery": "artillery_folder", "armor": "armour_folder",
	"nsb_armor": "nsb_armour_folder", "air_techs": "air_techs_folder",
	"bba_air_techs": "bba_air_techs_folder", "naval": "naval_folder",
	"mtg_naval": "mtgnavalfolder", "naval_support": "mtgnavalsupportfolder",
	"industry":                          "industry_folder",
	"electronic_mechanical_engineering": "electronics_folder",
	"special_forces_doctrine":           "special_forces_doctrine_folder",
}

var dlcPrefixes = map[string]bool{
	"MTG": true, "MtG": true, "WTT": true, "WP": true, "LAR": true, "BBA": true,
	"NSB": true, "TFV": true, "BFTB": true, "AAT": true, "TOA": true, "GOT": true,
	"GOE": true, "GTD": true, "DOD": true, "AOA": true, "WUW": true, "MPP": true,
	"GW": true, "GFX": true, "TFR": true,
}

var modBrandPrefixes = map[string]bool{"r56": true, "tfr": true, "kr": true, "modusa": true, "tno": true}

var techTagRE = regexp.MustCompile(`^[A-Z]{3,5}$`)

func techCountrySignal(techID string) string {
	if techID == "" || !strings.Contains(techID, "_") {
		return ""
	}
	parts := strings.Split(techID, "_")
	limit := min(2, len(parts))
	for i := 0; i < limit; i++ {
		p := parts[i]
		if i == 0 && modBrandPrefixes[strings.ToLower(p)] {
			continue
		}
		if techTagRE.MatchString(p) {
			if dlcPrefixes[p] {
				return ""
			}
			return p
		}
	}
	return ""
}

func techVisibleForCountry(techID, activeCountry string) bool {
	if activeCountry == "" {
		return true
	}
	signal := techCountrySignal(techID)
	return signal == "" || signal == activeCountry
}

func vanillaTechCollisionWarning(vanilla map[string]bool, techID, countryTag, source string) *string {
	if techID == "" || source == "imported" || !vanilla[techID] {
		return nil
	}
	prefix := "MOD_"
	if countryTag != "" {
		prefix = strings.ToUpper(countryTag) + "_"
	}
	suggested := techID
	if !strings.HasPrefix(techID, prefix) {
		suggested = prefix + techID
	}
	w := fmt.Sprintf("'%s' 跟 vanilla technology 同名 · HOI4 mod 加载会 silent override vanilla · 建议改名 '%s'(同 country TAG prefix 防 collision)", techID, suggested)
	return &w
}

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

func decodeColumn(v any, dst any) {
	if v == nil {
		return
	}
	raw, err := json.Marshal(v)
	if err != nil {
		return
	}
	_ = json.Unmarshal(raw, dst)
}

func text(row project.Row, column, def string) string {
	if s, ok := row[column].(string); ok {
		return s
	}
	return def
}

func float(row project.Row, column string, def float64) float64 {
	switch v := row[column].(type) {
	case json.Number:
		if f, err := v.Float64(); err == nil {
			return f
		}
	case float64:
		return v
	case float32:
		return float64(v)
	case int:
		return float64(v)
	case int64:
		return float64(v)
	}
	return def
}

func integer(row project.Row, column string, def int) int {
	if row[column] == nil {
		return def
	}
	return int(float(row, column, float64(def)))
}

func boolean(row project.Row, column string) bool {
	b, _ := row[column].(bool)
	return b
}

func techFromRow(row project.Row) Technology {
	t := Technology{
		ID: project.Int64(row["id"]), ProjectID: project.Int64(row["project_id"]),
		TechID: text(row, "tech_id", ""), CountryTag: text(row, "country_tag", ""),
		Name: text(row, "name", ""), TechLine: text(row, "tech_line", ""),
		ResearchCost: float(row, "research_cost", 1.0), StartYear: integer(row, "start_year", 1936),
		Folder: text(row, "folder", ""), PositionX: float(row, "position_x", 0), PositionY: float(row, "position_y", 0),
		AIWillDo: text(row, "ai_will_do", "factor = 1"), Source: text(row, "source", "created"),
		IsBanner: boolean(row, "is_banner"), ForceUseSmallTechLayout: boolean(row, "force_use_small_tech_layout"),
		NameEn: text(row, "name_en", ""), NameZh: text(row, "name_zh", ""),
		DescEn: text(row, "desc_en", ""), DescZh: text(row, "desc_zh", ""),
		EnableBuilding: text(row, "enable_building", ""), EnableTactic: text(row, "enable_tactic", ""),
		Doctrine: boolean(row, "doctrine"), Allow: text(row, "allow", ""), AllowBranch: text(row, "allow_branch", ""),
		OnResearchComplete:      text(row, "on_research_complete", ""),
		OnResearchCompleteLimit: text(row, "on_research_complete_limit", ""),
		XpResearchType:          text(row, "xp_research_type", ""), XpBoostCost: integer(row, "xp_boost_cost", 0),
	}
	if b, ok := row["position_y_is_explicit"].(bool); ok {
		t.PositionYIsExplicit = &b
	}
	decodeColumn(row["folder_placements"], &t.FolderPlacements)
	decodeColumn(row["prerequisites"], &t.Prerequisites)
	decodeColumn(row["categories"], &t.Categories)
	decodeColumn(row["modifiers"], &t.Modifiers)
	decodeColumn(row["raw_extras"], &t.RawExtras)
	decodeColumn(row["display_name_locs"], &t.DisplayNameLocs)
	decodeColumn(row["description_locs"], &t.DescriptionLocs)
	decodeColumn(row["enable_equipments"], &t.EnableEquipments)
	decodeColumn(row["enable_subunits"], &t.EnableSubunits)
	decodeColumn(row["enable_equipment_modules"], &t.EnableEquipmentModules)
	decodeColumn(row["xor"], &t.Xor)
	decodeColumn(row["dependencies"], &t.Dependencies)
	decodeColumn(row["sub_technologies"], &t.SubTechnologies)
	decodeColumn(row["path"], &t.Path)
	decodeColumn(row["xp_research_bonus"], &t.XpResearchBonus)
	for _, p := range []*[]string{&t.Prerequisites, &t.Categories, &t.EnableEquipments, &t.EnableSubunits, &t.EnableEquipmentModules, &t.Xor, &t.SubTechnologies} {
		if *p == nil {
			*p = []string{}
		}
	}
	for _, m := range []*map[string]any{&t.Modifiers, &t.RawExtras, &t.Dependencies, &t.XpResearchBonus} {
		if *m == nil {
			*m = map[string]any{}
		}
	}
	if t.DisplayNameLocs == nil {
		t.DisplayNameLocs = map[string]string{}
	}
	if t.DescriptionLocs == nil {
		t.DescriptionLocs = map[string]string{}
	}
	if t.Path == nil {
		t.Path = []map[string]any{}
	}
	t.Warnings = []string{}
	return t
}

func sortForList(rows []project.Row) {
	key := func(row project.Row, column string) (float64, bool) {
		if row[column] == nil {
			return 0, false
		}
		return float(row, column, 0), true
	}
	less := func(a, b project.Row, column string) (bool, bool) {
		av, aok := key(a, column)
		bv, bok := key(b, column)
		switch {
		case aok && bok && av != bv:
			return av < bv, true
		case aok != bok:
			return aok, true
		}
		return false, false
	}
	sort.SliceStable(rows, func(i, j int) bool {
		if result, decided := less(rows[i], rows[j], "start_year"); decided {
			return result
		}
		result, _ := less(rows[i], rows[j], "position_x")
		return result
	})
}

type CreateInput struct {
	TechID                  string            `json:"tech_id"`
	CountryTag              string            `json:"country_tag"`
	Name                    *string           `json:"name"`
	TechLine                *string           `json:"tech_line"`
	ResearchCost            *float64          `json:"research_cost"`
	StartYear               *int              `json:"start_year"`
	Folder                  *string           `json:"folder"`
	PositionX               float64           `json:"position_x"`
	PositionY               float64           `json:"position_y"`
	FolderPlacements        []map[string]any  `json:"folder_placements"`
	PositionYIsExplicit     bool              `json:"position_y_is_explicit"`
	Prerequisites           []string          `json:"prerequisites"`
	Categories              []string          `json:"categories"`
	Modifiers               map[string]any    `json:"modifiers"`
	AIWillDo                *string           `json:"ai_will_do"`
	NameEn                  string            `json:"name_en"`
	NameZh                  string            `json:"name_zh"`
	DescEn                  string            `json:"desc_en"`
	DescZh                  string            `json:"desc_zh"`
	DisplayNameLocs         map[string]string `json:"display_name_locs"`
	DescriptionLocs         map[string]string `json:"description_locs"`
	Source                  *string           `json:"source"`
	IsBanner                bool              `json:"is_banner"`
	ForceUseSmallTechLayout bool              `json:"force_use_small_tech_layout"`
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
}

func (in *CreateInput) Normalize() {
	if in.Prerequisites == nil {
		in.Prerequisites = []string{}
	}
	if in.Categories == nil {
		in.Categories = []string{}
	}
	if in.Modifiers == nil {
		in.Modifiers = map[string]any{}
	}
	if in.DisplayNameLocs == nil {
		in.DisplayNameLocs = map[string]string{}
	}
	if in.DescriptionLocs == nil {
		in.DescriptionLocs = map[string]string{}
	}
	if in.EnableEquipments == nil {
		in.EnableEquipments = []string{}
	}
	if in.EnableSubunits == nil {
		in.EnableSubunits = []string{}
	}
	if in.EnableEquipmentModules == nil {
		in.EnableEquipmentModules = []string{}
	}
	if in.Xor == nil {
		in.Xor = []string{}
	}
	if in.Dependencies == nil {
		in.Dependencies = map[string]any{}
	}
	if in.SubTechnologies == nil {
		in.SubTechnologies = []string{}
	}
	if in.Path == nil {
		in.Path = []map[string]any{}
	}
	if in.XpResearchBonus == nil {
		in.XpResearchBonus = map[string]any{}
	}
}

type Patch struct {
	TechID                  workbench.Optional[string]            `json:"tech_id"`
	CountryTag              workbench.Optional[string]            `json:"country_tag"`
	Name                    workbench.Optional[string]            `json:"name"`
	TechLine                workbench.Optional[string]            `json:"tech_line"`
	ResearchCost            workbench.Optional[float64]           `json:"research_cost"`
	StartYear               workbench.Optional[int]               `json:"start_year"`
	Folder                  workbench.Optional[string]            `json:"folder"`
	PositionX               workbench.Optional[float64]           `json:"position_x"`
	PositionY               workbench.Optional[float64]           `json:"position_y"`
	FolderPlacements        workbench.Optional[[]map[string]any]  `json:"folder_placements"`
	PositionYIsExplicit     workbench.Optional[bool]              `json:"position_y_is_explicit"`
	Prerequisites           workbench.Optional[[]string]          `json:"prerequisites"`
	Categories              workbench.Optional[[]string]          `json:"categories"`
	Modifiers               workbench.Optional[map[string]any]    `json:"modifiers"`
	AIWillDo                workbench.Optional[string]            `json:"ai_will_do"`
	RawExtras               workbench.Optional[map[string]any]    `json:"raw_extras"`
	NameEn                  workbench.Optional[string]            `json:"name_en"`
	NameZh                  workbench.Optional[string]            `json:"name_zh"`
	DescEn                  workbench.Optional[string]            `json:"desc_en"`
	DescZh                  workbench.Optional[string]            `json:"desc_zh"`
	DisplayNameLocs         workbench.Optional[map[string]string] `json:"display_name_locs"`
	DescriptionLocs         workbench.Optional[map[string]string] `json:"description_locs"`
	Source                  workbench.Optional[string]            `json:"source"`
	IsBanner                workbench.Optional[bool]              `json:"is_banner"`
	ForceUseSmallTechLayout workbench.Optional[bool]              `json:"force_use_small_tech_layout"`
	EnableEquipments        workbench.Optional[[]string]          `json:"enable_equipments"`
	EnableSubunits          workbench.Optional[[]string]          `json:"enable_subunits"`
	EnableEquipmentModules  workbench.Optional[[]string]          `json:"enable_equipment_modules"`
	EnableBuilding          workbench.Optional[string]            `json:"enable_building"`
	EnableTactic            workbench.Optional[string]            `json:"enable_tactic"`
	Xor                     workbench.Optional[[]string]          `json:"xor"`
	Dependencies            workbench.Optional[map[string]any]    `json:"dependencies"`
	SubTechnologies         workbench.Optional[[]string]          `json:"sub_technologies"`
	Doctrine                workbench.Optional[bool]              `json:"doctrine"`
	Allow                   workbench.Optional[string]            `json:"allow"`
	AllowBranch             workbench.Optional[string]            `json:"allow_branch"`
	OnResearchComplete      workbench.Optional[string]            `json:"on_research_complete"`
	OnResearchCompleteLimit workbench.Optional[string]            `json:"on_research_complete_limit"`
	Path                    workbench.Optional[[]map[string]any]  `json:"path"`
	XpResearchType          workbench.Optional[string]            `json:"xp_research_type"`
	XpBoostCost             workbench.Optional[int]               `json:"xp_boost_cost"`
	XpResearchBonus         workbench.Optional[map[string]any]    `json:"xp_research_bonus"`
}

func normalizedFolderPlacements(placements []map[string]any, fallback string) []map[string]any {
	if placements == nil {
		return nil
	}
	out := make([]map[string]any, 0, len(placements))
	for _, placement := range placements {
		copy := make(map[string]any, len(placement)+1)
		for key, value := range placement {
			copy[key] = value
		}
		folder, _ := copy["folder"].(string)
		folder = strings.TrimSpace(folder)
		if folder == "" {
			folder = fallback
		}
		copy["folder"] = folder
		out = append(out, copy)
	}
	return out
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

func firstFolderPlacement(placements []map[string]any) (folder string, x, y float64, ok bool) {
	if len(placements) == 0 {
		return "", 0, 0, false
	}
	first := placements[0]
	folder, _ = first["folder"].(string)
	return folder, folderPlacementFloat(first["x"]), folderPlacementFloat(first["y"]), true
}

func jsonValue(v any) any {
	if v == nil {
		return nil
	}
	raw, err := json.Marshal(v)
	if err != nil {
		return nil
	}
	var out any
	_ = json.Unmarshal(raw, &out)
	return out
}

func stringFrom(value any) string {
	if text, ok := value.(string); ok {
		return text
	}
	return ""
}
