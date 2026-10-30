package exportcheck

import (
	"fmt"
	"regexp"
	"strings"
)

var subUnitInternalTypes = map[string]bool{
	"infantry": true, "support": true, "artillery": true, "anti_tank": true,
	"anti_air": true, "armor": true, "fighter": true, "cas": true,
	"naval_bomber": true, "interceptor": true, "suicide": true,
	"tactical_bomber": true, "strategic_bomber": true, "air_transport": true,
	"missile": true, "submarine": true, "screen_ship": true,
	"capital_ship": true, "carrier": true, "motorized": true,
	"mechanized": true, "flame": true, "amphibious": true,
}

var subUnitBooleanFields = map[string]bool{
	"active": true, "cavalry": true, "special_forces": true, "marines": true,
	"can_exfiltrate_from_coast": true, "mountaineers": true,
	"can_be_parachuted": true, "affects_speed": true,
}

var subUnitCategoryTokenRE = regexp.MustCompile(`^[A-Za-z0-9_@.$:-]+$`)

func validateSubUnitTypes(values []string) ([]string, error) {
	normalized := make([]string, 0, len(values))
	for _, value := range values {
		token := strings.TrimSpace(value)
		if subUnitBooleanFields[token] {
			return nil, fmt.Errorf(
				"unit_types contains boolean field '%s'; set %s=yes instead", token, token)
		}
		if !subUnitInternalTypes[token] {
			return nil, fmt.Errorf("unsupported sub-unit internal type '%s'", token)
		}
		normalized = append(normalized, token)
	}
	return normalized, nil
}

func validateSubUnitCategories(values []string) ([]string, error) {
	normalized := make([]string, 0, len(values))
	for _, value := range values {
		token := strings.TrimSpace(value)
		if token == "" || !subUnitCategoryTokenRE.MatchString(token) {
			return nil, fmt.Errorf("invalid sub-unit category token '%s'", token)
		}
		normalized = append(normalized, token)
	}
	return normalized, nil
}

type SubUnit struct {
	ID                     int64             `json:"id"`
	ProjectID              int64             `json:"project_id"`
	SubUnitID              string            `json:"sub_unit_id"`
	CountryTag             string            `json:"country_tag"`
	SourceFile             string            `json:"source_file"`
	Source                 string            `json:"source"`
	Ordering               int               `json:"ordering"`
	Name                   string            `json:"name"`
	NameEn                 string            `json:"name_en"`
	NameZh                 string            `json:"name_zh"`
	DescEn                 string            `json:"desc_en"`
	DescZh                 string            `json:"desc_zh"`
	DisplayNameLocs        map[string]string `json:"display_name_locs"`
	DescriptionLocs        map[string]string `json:"description_locs"`
	Abbreviation           string            `json:"abbreviation"`
	Sprite                 string            `json:"sprite"`
	MapIconCategory        string            `json:"map_icon_category"`
	Priority               *int              `json:"priority"`
	AiPriority             *int              `json:"ai_priority"`
	Transport              string            `json:"transport"`
	Group                  string            `json:"group"`
	Active                 *bool             `json:"active"`
	Cavalry                *bool             `json:"cavalry"`
	SpecialForces          *bool             `json:"special_forces"`
	Marines                *bool             `json:"marines"`
	CanExfiltrateFromCoast *bool             `json:"can_exfiltrate_from_coast"`
	Mountaineers           *bool             `json:"mountaineers"`
	CanBeParachuted        *bool             `json:"can_be_parachuted"`
	AffectsSpeed           *bool             `json:"affects_speed"`
	UnitTypes              []string          `json:"unit_types"`
	Categories             []string          `json:"categories"`
	Essential              []string          `json:"essential"`
	Need                   map[string]any    `json:"need"`
	Stats                  map[string]any    `json:"stats"`
	TerrainModifiers       map[string]any    `json:"terrain_modifiers"`
	RawExtras              map[string]any    `json:"raw_extras"`
}
