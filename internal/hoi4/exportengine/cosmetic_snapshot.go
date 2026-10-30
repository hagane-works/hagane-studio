package exportengine

import (
	"encoding/json"
	"fmt"
	"strings"

	"haganeworks/internal/hoi4/cosmetic"
)

func nativeCosmeticDefinitions(snapshot Snapshot, includeImported bool) (map[string]cosmetic.Definition, error) {
	type parent struct {
		ID     int64  `json:"id"`
		Source string `json:"source"`
		cosmetic.Definition
	}
	byID := map[int64]*parent{}
	for _, raw := range snapshot.Tables["project_cosmetic_tags"] {
		var row parent
		if err := json.Unmarshal(raw, &row); err != nil {
			return nil, fmt.Errorf("decode cosmetic tag: %w", err)
		}
		if err := cosmetic.ValidateTag(row.Tag); err != nil {
			return nil, err
		}
		row.NameLocs, row.ImportedNameLocs = cosmetic.Names{}, cosmetic.Names{}
		if !includeImported {
			row.ImportedColor, row.ImportedColorUI = nil, nil
		}
		byID[row.ID] = &row
	}
	for _, raw := range snapshot.Tables["project_cosmetic_tag_names"] {
		var row struct {
			ParentID                      int64 `json:"cosmetic_tag_id"`
			Locale, Suffix, Value, Source string
		}
		if err := json.Unmarshal(raw, &row); err != nil {
			return nil, fmt.Errorf("decode cosmetic name: %w", err)
		}
		p := byID[row.ParentID]
		if p == nil {
			continue
		}
		imported := strings.EqualFold(row.Source, "imported")
		if imported && !includeImported {
			continue
		}

		if err := cosmetic.ValidateSuffix(row.Suffix); err != nil {
			continue
		}
		names := p.NameLocs
		if imported {
			names = p.ImportedNameLocs
		} else if err := cosmetic.ValidateName(row.Locale, row.Suffix, row.Value); err != nil {
			return nil, err
		}
		if names[row.Locale] == nil {
			names[row.Locale] = map[string]string{}
		}
		names[row.Locale][row.Suffix] = row.Value
	}
	out := map[string]cosmetic.Definition{}
	for _, row := range byID {
		if !includeImported && strings.EqualFold(row.Source, "imported") && len(row.NameLocs) == 0 && row.Color == nil && row.ColorUI == nil {
			continue
		}
		out[row.Tag] = row.Definition
	}
	return out, nil
}

func nativeCosmeticOwners(snapshot Snapshot) (map[string]string, error) {
	out := map[string]string{}
	ids := map[string]int64{}
	for _, raw := range snapshot.Tables["history_files"] {
		var row struct {
			ID      int64
			Tag     string `json:"set_cosmetic_tag"`
			Country string `json:"country_tag"`
		}
		if err := json.Unmarshal(raw, &row); err != nil {
			return nil, err
		}
		if row.Tag == "" {
			continue
		}
		if id, ok := ids[row.Tag]; !ok || row.ID < id {
			ids[row.Tag] = row.ID
			out[row.Tag] = strings.ToUpper(strings.TrimSpace(row.Country))
		}
	}
	return out, nil
}
