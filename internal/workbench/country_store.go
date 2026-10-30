package workbench

import (
	"fmt"

	"haganeworks/internal/project"
)

var countryTextFields = []string{
	"tag", "name", "ideology", "graphical_culture", "graphical_culture_2d",
	"name_en", "name_zh", "adjective_en", "adjective_zh", "flag_image_path",
	"name_democratic_en", "name_democratic_zh", "name_fascism_en", "name_fascism_zh",
	"name_communism_en", "name_communism_zh", "name_neutrality_en", "name_neutrality_zh",
}

func CountryResponse(row project.Row) project.Row {
	out := project.Row{
		"id":                     row["id"],
		"project_id":             row["project_id"],
		"color":                  listOrEmpty(row["color"]),
		"color_ui":               listOrEmpty(row["color_ui"]),
		"use_legacy_ai_pp_spend": row["use_legacy_ai_pp_spend"],
		"flag_paths_by_ideology": mapOrEmpty(row["flag_paths_by_ideology"]),
		"display_name_locs":      mapOrEmpty(row["display_name_locs"]),
		"adjective_locs":         mapOrEmpty(row["adjective_locs"]),
		"ideology_name_locs":     mapOrEmpty(row["ideology_name_locs"]),
		"auto_added":             row["auto_added"] == true,
		"warning":                nil,
	}
	for _, field := range countryTextFields {
		s, _ := row[field].(string)
		out[field] = s
	}
	return out
}

func listOrEmpty(v any) any {
	if v == nil {
		return []any{}
	}
	return v
}

func mapOrEmpty(v any) any {
	if v == nil {
		return map[string]any{}
	}
	return v
}

func ListCountries(p *project.Project) ([]project.Row, error) {
	rows, err := p.Rows("countries")
	if err != nil {
		return nil, err
	}
	out := make([]project.Row, len(rows))
	for i, row := range rows {
		out[i] = CountryResponse(row)
	}
	return out, nil
}

func GetCountry(p *project.Project, id int64) (project.Row, error) {
	rows, err := p.Rows("countries")
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		if project.Int64(row["id"]) == id {
			return CountryResponse(row), nil
		}
	}
	return nil, ErrCountryMissing
}

func CreateCountry(p *project.Project, in CountryCreateInput) (project.Row, error) {
	if err := in.Normalize(); err != nil {
		return nil, ValidationError(err.Error())
	}
	locs := &CountryLocsFields{
		AdjectiveEn: in.AdjectiveEn, AdjectiveZh: in.AdjectiveZh,
		AdjectiveLocs:    in.AdjectiveLocs,
		IdeologyNameLocs: in.IdeologyNameLocs,
		IdeologyScalars: map[string]*[2]string{
			"democratic": {in.NameDemocraticEn, in.NameDemocraticZh},
			"fascism":    {in.NameFascismEn, in.NameFascismZh},
			"communism":  {in.NameCommunismEn, in.NameCommunismZh},
			"neutrality": {in.NameNeutralityEn, in.NameNeutralityZh},
		},
	}
	normalizeCountryLocs(locs)
	autoEn, autoZh := splitLocName(*in.Name, in.Tag)
	var created project.Row
	err := p.Write([]string{"countries"}, func(tx *project.Tx) error {
		for _, row := range tx.Rows("countries") {
			if row["tag"] == in.Tag {
				return ErrCountryConflict{Tag: in.Tag}
			}
		}
		created = tx.Insert("countries", project.Row{
			"tag": in.Tag, "name": *in.Name, "color": int64List(in.Color), "color_ui": int64List(in.ColorUI),
			"ideology": strDefault(in.Ideology, "neutrality"), "graphical_culture": in.GraphicalCulture,
			"graphical_culture_2d": in.GraphicalCulture2D, "use_legacy_ai_pp_spend": boolPointer(in.UseLegacyAIPPSpend),
			"name_en": firstNonBlank(in.NameEn, autoEn), "name_zh": firstNonBlank(in.NameZh, autoZh),
			"adjective_en": locs.AdjectiveEn, "adjective_zh": locs.AdjectiveZh,
			"adjective_locs": stringMap(locs.AdjectiveLocs), "display_name_locs": stringMap(in.DisplayNameLocs),
			"ideology_name_locs": nestedStringMap(locs.IdeologyNameLocs), "flag_image_path": in.FlagImagePath,
			"name_democratic_en": locs.IdeologyScalars["democratic"][0], "name_democratic_zh": locs.IdeologyScalars["democratic"][1],
			"name_fascism_en": locs.IdeologyScalars["fascism"][0], "name_fascism_zh": locs.IdeologyScalars["fascism"][1],
			"name_communism_en": locs.IdeologyScalars["communism"][0], "name_communism_zh": locs.IdeologyScalars["communism"][1],
			"name_neutrality_en": locs.IdeologyScalars["neutrality"][0], "name_neutrality_zh": locs.IdeologyScalars["neutrality"][1],
			"flag_paths_by_ideology": map[string]any{}, "has_obs": false, "auto_added": false,
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	if err := syncProjectCountryTags(p); err != nil {
		return nil, err
	}
	response := CountryResponse(created)
	if warning := vanillaTagWarning(in.Tag); warning != nil {
		response["warning"] = *warning
	}
	return response, nil
}

func TakeInCountry(p *project.Project, tag string) (project.Row, bool, error) {
	if !countryTagRE.MatchString(tag) {
		return nil, false, ValidationError(ErrCountryTagInvalid{Tag: tag}.Error())
	}
	var row project.Row
	created := false
	err := p.Write([]string{"countries"}, func(tx *project.Tx) error {
		for _, existing := range tx.Rows("countries") {
			if existing["tag"] == tag {
				row = existing
				return nil
			}
		}
		created = true
		row = tx.Insert("countries", project.Row{
			"tag": tag, "name": tag, "color": []any{}, "color_ui": []any{}, "ideology": "",
			"graphical_culture": "", "graphical_culture_2d": "", "name_en": "", "name_zh": "",
			"adjective_en": "", "adjective_zh": "", "adjective_locs": map[string]any{},
			"display_name_locs": map[string]any{}, "ideology_name_locs": map[string]any{},
			"flag_image_path": "", "flag_paths_by_ideology": map[string]any{}, "has_obs": false, "auto_added": false,
		})
		return nil
	})
	if err != nil {
		return nil, false, err
	}
	if created {
		if err := syncProjectCountryTags(p); err != nil {
			return nil, false, err
		}
	}
	return CountryResponse(row), created, nil
}

func GroupedCountries(p *project.Project) (map[string]any, error) {
	activeTag, _ := p.Record()["country_tag"].(string)
	perBadge := map[string]map[string]int64{}
	for _, src := range groupedContentTables {
		counts := map[string]int64{}
		rows, err := p.Rows(src.Table)
		if err != nil {
			return nil, err
		}
		for _, row := range rows {
			tag, _ := row["country_tag"].(string)
			counts[tag]++
		}
		perBadge[src.Badge] = counts
	}
	countries, err := ListCountries(p)
	if err != nil {
		return nil, err
	}
	authored := []map[string]any{}
	reference := []map[string]any{}
	for _, c := range countries {
		tag := c["tag"].(string)
		counts := make(OrderedCounts, 0, len(groupedContentTables))
		var total int64
		for _, src := range groupedContentTables {
			n := perBadge[src.Badge][tag]
			counts = append(counts, struct {
				Label string
				N     int64
			}{src.Badge, n})
			total += n
		}
		autoAdded := c["auto_added"] == true
		isActive := activeTag != "" && tag == activeTag
		isAuthored := !autoAdded || isActive || total > 0
		item := map[string]any{
			"id": c["id"], "tag": tag, "name": c["name"],
			"name_en": c["name_en"], "name_zh": c["name_zh"], "color": c["color"],
			"auto_added": autoAdded, "is_active": isActive,
			"is_authored": isAuthored, "counts": counts, "total_content": total,
		}
		if isAuthored {
			authored = append(authored, item)
		} else {
			reference = append(reference, item)
		}
	}
	return map[string]any{
		"project_id": p.ID(), "active_tag": activeTag,
		"authored": authored, "reference": reference,
		"authored_total": len(authored), "reference_total": len(reference),
	}, nil
}

func syncProjectCountryTags(p *project.Project) error {
	rows, err := p.Rows("countries")
	if err != nil {
		return err
	}
	tags := []string{}
	names := map[string]string{}
	for _, row := range rows {
		tag, _ := row["tag"].(string)
		name, _ := row["name"].(string)
		tags = append(tags, tag)
		names[tag] = name
	}
	record := p.Record()
	curTag, _ := record["country_tag"].(string)
	curName, _ := record["country_name"].(string)
	curTags := stringList(record["country_tags"])
	newTag, newName := curTag, curName
	if len(tags) > 0 {
		if !containsString(tags, curTag) {
			newTag = tags[0]
		}
		if n, ok := names[newTag]; ok {
			newName = n
		}
	}
	if fmt.Sprint(curTags) == fmt.Sprint(tags) && newTag == curTag && newName == curName {
		return nil
	}
	_, err = p.UpdateRecord(func(r project.Row) {
		r["country_tags"] = tags
		r["country_tag"] = newTag
		r["country_name"] = newName
		r["updated_at"] = nowTimestamp()
	})
	return err
}

func containsString(xs []string, want string) bool {
	for _, x := range xs {
		if x == want {
			return true
		}
	}
	return false
}

func int64List(values []int64) []any {
	out := make([]any, len(values))
	for i, v := range values {
		out[i] = v
	}
	return out
}

func stringMap(values map[string]string) map[string]any {
	out := make(map[string]any, len(values))
	for k, v := range values {
		out[k] = v
	}
	return out
}

func nestedStringMap(values map[string]map[string]string) map[string]any {
	out := make(map[string]any, len(values))
	for k, v := range values {
		out[k] = stringMap(v)
	}
	return out
}

func boolPointer(v *bool) any {
	if v == nil {
		return nil
	}
	return *v
}

func ExistingCountryTags(p *project.Project) (map[string]bool, error) {
	rows, err := p.Rows("countries")
	if err != nil {
		return nil, err
	}
	out := map[string]bool{}
	for _, row := range rows {
		if tag, ok := row["tag"].(string); ok {
			out[tag] = true
		}
	}
	return out, nil
}
