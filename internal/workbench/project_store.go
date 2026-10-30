package workbench

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"time"

	"haganeworks/internal/hoi4/techsurface"
	"haganeworks/internal/project"
	"haganeworks/internal/schema"
)

const techYearColumnsDefault = `[["@1918",0],["@1936",2],["@1938",4],["@1939",6],["@1940",8],["@1941",10],["@1942",12],["@1943",14],["@1944",16],["@1945",18],["@1946",20]]`

type ValidationError string

func (e ValidationError) Error() string { return string(e) }

var ErrNotFound = errors.New("not found")

func CreateProject(store *project.Store, in CreateInput) (project.Row, error) {
	if err := in.Normalize(); err != nil {
		return nil, ValidationError(err.Error())
	}
	if in.TemplateID != nil {

		return nil, ValidationError("base-mod templates are not available in the desktop program")
	}
	registry := normalizeRegistry(in.IdeologyRegistry)
	if err := validateRegistry(registry, in.Dependencies); err != nil {
		return nil, ValidationError(err.Error())
	}
	surface, err := techsurface.Parse(in.TechSurfaceManifest)
	if err != nil {
		return nil, ValidationError(err.Error())
	}
	if findings := techsurface.ValidateShape(surface, len(in.Dependencies) > 0); len(findings) > 0 {
		return nil, ValidationError(findings[0].Code + ": " + findings[0].Message)
	}
	primaryTag, seeds := countrySeedPlan(in)
	countryTags := []string{}
	countryName := ""
	if seeds != nil {
		countryName = in.CountryName
		for _, seed := range seeds {
			countryTags = append(countryTags, seed.Tag)
		}
	}
	var stageStateIDs any
	if in.stageJSON != nil {
		stageStateIDs = *in.stageJSON
	}
	var ownedDLCs any
	if in.OwnedDLCs != nil {
		ownedDLCs = *in.OwnedDLCs
	}
	var techYears any
	_ = json.Unmarshal([]byte(techYearColumnsDefault), &techYears)
	record := project.Row{
		"uuid":                              newUUID(),
		"name":                              in.Name,
		"country_tag":                       primaryTag,
		"country_name":                      countryName,
		"country_tags":                      countryTags,
		"mod_path":                          "",
		"ideology_icon_paths":               in.IdeologyIconPaths,
		"version":                           *in.Version,
		"picture":                           in.Picture,
		"tags":                              in.Tags,
		"dependencies":                      in.Dependencies,
		"supported_version":                 *in.SupportedVersion,
		"imported_descriptor_replace_paths": in.ImportedReplacePaths,
		"basemod_template_id":               nil,
		"basemod_pinned_version":            nil,
		"is_loaded_existing_mod":            in.Loaded,
		"start_date":                        in.StartDate,
		"end_date":                          in.EndDate,
		"scenario_history_policy":           *in.ScenarioHistoryPolicy,
		"stage_state_ids":                   stageStateIDs,
		"tech_year_columns":                 techYears,
		"export_languages":                  *in.ExportLanguages,
		"ideology_registry":                 registry,
		"tech_surface_manifest":             in.TechSurfaceManifest,
		"ideology_source_manifest":          map[string]any{},
		"owned_dlcs":                        ownedDLCs,
	}
	p, err := store.Create(record, in.Name)
	if err != nil {
		return nil, err
	}
	if len(seeds) > 0 {
		err := p.Write([]string{"countries"}, func(tx *project.Tx) error {
			for _, seed := range seeds {
				tx.Insert("countries", newCountryRow(seed.Tag, seed.Name, seed.NameEN, seed.NameZH))
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	return ProjectRecord(p), nil
}

func newCountryRow(tag, name, nameEN, nameZH string) project.Row {
	return project.Row{
		"tag": tag, "name": name, "color": []any{}, "color_ui": []any{},
		"ideology": "neutrality", "graphical_culture": "", "graphical_culture_2d": "",
		"name_en": nameEN, "name_zh": nameZH, "adjective_en": "", "adjective_zh": "",
		"display_name_locs": map[string]any{}, "adjective_locs": map[string]any{},
		"ideology_name_locs": map[string]any{},
		"name_democratic_en": "", "name_democratic_zh": "", "name_fascism_en": "",
		"name_fascism_zh": "", "name_communism_en": "", "name_communism_zh": "",
		"name_neutrality_en": "", "name_neutrality_zh": "", "flag_image_path": "",
		"flag_paths_by_ideology": map[string]any{}, "has_obs": false, "auto_added": false,
	}
}

func newUUID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

func nowTimestamp() string {
	return schema.Timestamp("timestamp without time zone", time.Now())
}

var countTables = map[string]string{
	"character": "characters", "custom_trait": "custom_traits", "opinion_modifier": "opinion_modifiers",
	"dynamic_modifier": "dynamic_modifiers", "modifier_definition": "modifier_definitions",
	"ai_strategy_plan": "ai_strategy_plans", "ai_template": "ai_templates", "wargoal": "wargoal_types",
	"focus_tree": "focus_trees", "technology": "technologies", "idea": "ideas",
	"event_namespace": "event_namespaces", "event": "events", "decision_category": "decision_categories",
	"decision": "decisions", "super_event": "super_events", "bookmark": "bookmarks",
	"history_file": "history_files", "mio": "military_industrial_organizations", "country": "countries",
	"state": "states", "focus_node": "focus_nodes",
}

func ProjectRecord(p *project.Project) project.Row {
	return projectResponse(p.Record(), p.Counts())
}

func ProjectListRecord(row project.Row) project.Row {
	counts := map[string]int64{}
	if raw, ok := row["_table_counts"].(map[string]any); ok {
		for table, n := range raw {
			counts[table] = project.Int64(n)
		}
	}
	delete(row, "_table_counts")
	return projectResponse(row, counts)
}

func projectResponse(row project.Row, tableCounts map[string]int64) project.Row {
	isNil := func(v any) bool { return v == nil }
	isBlank := func(v any) bool { s, _ := v.(string); return s == "" }
	isEmptyList := func(v any) bool { list, _ := v.([]any); return len(list) == 0 }
	setDefault := func(key string, empty func(any) bool, value any) {
		if empty(row[key]) {
			row[key] = value
		}
	}
	if row["country_tags"] == nil {
		if tag, _ := row["country_tag"].(string); tag != "" {
			row["country_tags"] = []any{tag}
		} else {
			row["country_tags"] = []any{}
		}
	}
	setDefault("ideology_icon_paths", isNil, map[string]any{})
	setDefault("version", isBlank, "1.0")
	setDefault("tags", isEmptyList, []any{"Alternative History"})
	setDefault("dependencies", isNil, []any{})
	setDefault("imported_descriptor_replace_paths", isNil, []any{})
	setDefault("supported_version", isBlank, "1.19.*")
	setDefault("scenario_history_policy", isBlank, ScenarioHistoryPolicyInheritBaseHistory)
	setDefault("export_languages", isEmptyList, []any{"en"})
	setDefault("tech_year_columns", isNil, []any{})
	setDefault("ideology_registry", isNil, map[string]any{})
	setDefault("tech_surface_manifest", isNil, map[string]any{})
	setDefault("ideology_source_manifest", isNil, map[string]any{})
	counts := map[string]int64{}
	for key, table := range countTables {
		counts[key] = tableCounts[table]
	}
	row["entity_counts"] = counts
	row["character_count"] = counts["character"]
	row["focus_node_count"] = counts["focus_node"]
	row["technology_count"] = counts["technology"]
	return row
}

func UpdateProject(p *project.Project, body []byte) (project.Row, error) {
	var patch UpdatePatch
	if err := json.Unmarshal(body, &patch); err != nil {
		return nil, ValidationError(err.Error())
	}
	if err := patch.Validate(); err != nil {
		return nil, ValidationError(err.Error())
	}
	if err := validateAgainst(&patch, p.Record()); err != nil {
		return nil, ValidationError(err.Error())
	}

	raw, err := json.Marshal(patchValues(patch))
	if err != nil {
		return nil, err
	}
	var values map[string]any
	if err := json.Unmarshal(raw, &values); err != nil {
		return nil, err
	}
	_, err = p.UpdateRecordAndBaseMods(func(record project.Row, baseMods map[string]string) {
		for key, value := range values {
			record[key] = value
		}
		if ids, ok := values["stage_state_ids"].([]any); ok {

			encoded, _ := json.Marshal(ids)
			record["stage_state_ids"] = string(encoded)
		}
		record["updated_at"] = nowTimestamp()
		forgetRemovedBaseMods(record, baseMods)
	})
	if err != nil {
		return nil, err
	}
	return ProjectRecord(p), nil
}

func patchValues(p UpdatePatch) map[string]any {
	out := map[string]any{}
	add := func(name string, set bool, value any) {
		if set {
			out[name] = value
		}
	}
	add("name", p.Name.Set, optionalValue(p.Name))
	add("country_tag", p.CountryTag.Set, optionalValue(p.CountryTag))
	add("country_name", p.CountryName.Set, optionalValue(p.CountryName))
	add("country_tags", p.CountryTags.Set, optionalValue(p.CountryTags))
	add("mod_path", p.ModPath.Set, optionalValue(p.ModPath))
	add("ideology_icon_paths", p.IdeologyIconPaths.Set, optionalValue(p.IdeologyIconPaths))
	add("version", p.Version.Set, optionalValue(p.Version))
	add("picture", p.Picture.Set, optionalValue(p.Picture))
	add("tags", p.Tags.Set, optionalValue(p.Tags))
	add("dependencies", p.Dependencies.Set, optionalValue(p.Dependencies))
	add("supported_version", p.SupportedVersion.Set, optionalValue(p.SupportedVersion))
	add("imported_descriptor_replace_paths", p.ImportedReplacePaths.Set, optionalValue(p.ImportedReplacePaths))
	add("is_loaded_existing_mod", p.Loaded.Set, optionalValue(p.Loaded))
	add("start_date", p.StartDate.Set, optionalValue(p.StartDate))
	add("end_date", p.EndDate.Set, optionalValue(p.EndDate))
	add("scenario_history_policy", p.ScenarioHistoryPolicy.Set, optionalValue(p.ScenarioHistoryPolicy))
	add("stage_state_ids", p.StageStateIDs.Set, optionalValue(p.StageStateIDs))
	add("export_languages", p.ExportLanguages.Set, optionalValue(p.ExportLanguages))
	add("owned_dlcs", p.OwnedDLCs.Set, optionalValue(p.OwnedDLCs))
	add("ideology_registry", p.IdeologyRegistry.Set, optionalValue(p.IdeologyRegistry))
	add("tech_surface_manifest", p.TechSurfaceManifest.Set, optionalValue(p.TechSurfaceManifest))
	return out
}

func validateAgainst(p *UpdatePatch, current project.Row) error {
	deps := stringList(current["dependencies"])
	if p.Dependencies.Set {
		if p.Dependencies.Value == nil {
			deps = nil
		} else {
			deps = *p.Dependencies.Value
		}
	}
	registry, _ := current["ideology_registry"].(map[string]any)
	if registry == nil {
		registry = map[string]any{}
	}
	if p.IdeologyRegistry.Set {
		if p.IdeologyRegistry.Value == nil {
			registry = map[string]any{}
		} else {
			registry = normalizeRegistry(*p.IdeologyRegistry.Value)
			p.IdeologyRegistry.Value = &registry
		}
	}
	if err := validateRegistry(registry, deps); err != nil {
		return err
	}
	surface, _ := current["tech_surface_manifest"].(map[string]any)
	if surface == nil {
		surface = map[string]any{}
	}
	if p.TechSurfaceManifest.Set {
		if p.TechSurfaceManifest.Value == nil {
			surface = map[string]any{}
		} else {
			surface = *p.TechSurfaceManifest.Value
		}
	}
	manifest, err := techsurface.Parse(surface)
	if err != nil {
		return err
	}
	if findings := techsurface.ValidateShape(manifest, len(deps) > 0); len(findings) > 0 {
		return fmt.Errorf("%s: %s", findings[0].Code, findings[0].Message)
	}
	return nil
}

func stringList(v any) []string {
	if names, ok := v.([]string); ok {
		return slices.Clone(names)
	}
	list, _ := v.([]any)
	out := make([]string, 0, len(list))
	for _, item := range list {
		if s, ok := item.(string); ok {
			out = append(out, s)
		}
	}
	return out
}
