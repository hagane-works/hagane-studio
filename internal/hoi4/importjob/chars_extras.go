package importjob

import (
	"bytes"
	"encoding/json"
	"fmt"
	"haganeworks/internal/hoi4/importassets"
	"haganeworks/internal/project"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"haganeworks/internal/hoi4/importsvc/characters"
	"haganeworks/internal/hoi4/importsvc/ideas"
	"haganeworks/internal/hoi4/importsvc/specialprojects"
)

type CharsExtrasStageOptions struct {
	ImportCharacters bool
	ImportIdeas      bool
	CountryTag       string
}

type CharsExtrasMergeOptions struct{ ConflictPolicy string }

func characterExtraFiles(root string) ([]string, error) {
	var out []string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, _ := filepath.Rel(root, path)
		n := strings.ToLower(filepath.ToSlash(rel))
		if !strings.HasSuffix(n, ".txt") {
			return nil
		}
		prefixes := []string{"common/characters/", "common/country_leader/", "common/unit_leader/", "common/scientist_traits/", "common/ideas/", "common/military_industrial_organization/", "common/intelligence_agencies/", "common/doctrines/", "common/special_projects/specialization/", "common/special_projects/projects/"}
		for _, p := range prefixes {
			if strings.HasPrefix(n, p) {
				out = append(out, path)
				break
			}
		}
		return nil
	})
	return out, err
}

func rowsFromMaps(kind, key, rel string, maps []map[string]any) ([]entityRow, error) {
	out := make([]entityRow, 0, len(maps))
	for _, m := range maps {
		id, _ := m[key].(string)
		if id == "" {
			continue
		}
		b, e := marshalPGJSON(m)
		if e != nil {
			return nil, e
		}
		out = append(out, entityRow{kind: kind, naturalKey: id, sourceFile: rel, payload: b})
	}
	return out, nil
}

func parseCharacterExtra(rel, text string) ([]entityRow, error) {
	n := strings.ToLower(rel)
	switch {
	case strings.HasPrefix(n, "common/characters/"):
		r, _, e := characters.ImportCharacters(text)
		if e != nil {
			return nil, e
		}
		for _, m := range r {
			m["source_file"] = rel
		}
		return rowsFromMaps("character", "tag", rel, characters.NativeEntities(r))
	case strings.HasPrefix(n, "common/country_leader/"):
		r, _, e := characters.ImportCountryLeaderTraits(text)
		if e != nil {
			return nil, e
		}
		return rowsFromMaps("trait:country_leader", "trait_id", rel, characters.NativeEntities(r))
	case strings.HasPrefix(n, "common/unit_leader/"):
		r, _, e := characters.ImportUnitLeaderTraits(text)
		if e != nil {
			return nil, e
		}
		return rowsFromMaps("trait:unit_leader", "trait_id", rel, characters.NativeEntities(r))
	case strings.HasPrefix(n, "common/scientist_traits/"):
		r, _, e := characters.ImportScientistTraits(text)
		if e != nil {
			return nil, e
		}
		return rowsFromMaps("trait:scientist", "trait_id", rel, characters.NativeEntities(r))
	case strings.HasPrefix(n, "common/ideas/"):
		r, _, e := ideas.ImportIdeas(text)
		if e != nil {
			return nil, e
		}
		return rowsFromMaps("idea", "idea_id", rel, ideas.NativeEntities(r))
	case strings.HasPrefix(n, "common/military_industrial_organization/"):
		r, _, e := ideas.ImportMIO(text)
		if e != nil {
			return nil, e
		}
		return rowsFromMaps("mio", "mio_id", rel, ideas.NativeEntities(r))
	case strings.HasPrefix(n, "common/intelligence_agencies/"):
		r, _, e := ideas.ImportIntelligenceAgencies(text)
		if e != nil {
			return nil, e
		}
		return rowsFromMaps("intel_agency", "agency_id", rel, ideas.NativeEntities(r))
	case strings.HasPrefix(n, "common/doctrines/"):
		parts := strings.Split(n, "/")
		if len(parts) < 3 {
			return nil, nil
		}
		typ := map[string]string{"folders": "folder", "grand_doctrines": "grand_doctrine", "tracks": "track", "subdoctrines": "subdoctrine"}[parts[2]]
		if typ == "" {
			return nil, nil
		}
		src := strings.TrimPrefix(rel, "common/doctrines/")
		r, _, e := ideas.ImportDoctrines(text, typ, src)
		if e != nil {
			return nil, e
		}
		maps := ideas.NativeEntities(r)
		for _, m := range maps {

			m["_compound_key"] = typ + "\x1f" + fmt.Sprint(m["entity_key"])
		}
		return rowsFromMaps("doctrine", "_compound_key", rel, maps)
	case strings.HasPrefix(n, "common/special_projects/specialization/"):
		r, _, e := specialprojects.ImportSpecializations(text)
		if e != nil {
			return nil, e
		}
		maps := make([]map[string]any, 0, len(r))
		for _, x := range r {
			color := make([]any, len(x.Color))
			for i, v := range x.Color {
				if v.IsInt64() {
					color[i] = v.Int64()
				} else {
					color[i] = v.String()
				}
			}
			maps = append(maps, map[string]any{"specialization_id": x.SpecializationID, "color": color, "blueprint_image": x.BlueprintImage, "program_background": x.ProgramBackground, "facility_model": x.FacilityModel, "sub_specializations": x.SubSpecializations, "raw_extras": specialprojects.NativeRawExtras(x.RawExtras)})
		}
		return rowsFromMaps("specialization", "specialization_id", rel, maps)
	case strings.HasPrefix(n, "common/special_projects/projects/"):
		r, _, e := specialprojects.ImportSpecialProjects(text)
		if e != nil {
			return nil, e
		}
		maps := make([]map[string]any, 0, len(r))
		for _, x := range r {
			maps = append(maps, map[string]any{"project_token": x.ProjectToken, "specialization": x.Specialization, "icon": x.Icon, "blueprint_image": x.BlueprintImage, "complexity": x.Complexity, "prototype_time": x.PrototypeTime, "project_tags": x.ProjectTags, "special_project_parent": x.SpecialProjectParent, "allowed": x.Allowed, "visible": x.Visible, "available": x.Available, "resource_cost": x.ResourceCost, "project_output": x.ProjectOutput, "generic_prototype_rewards": x.GenericPrototypeRewards, "raw_extras": specialprojects.NativeRawExtras(x.RawExtras)})
		}
		return rowsFromMaps("special_project", "project_token", rel, maps)
	}
	return nil, nil
}

func (s *session) stageCharsExtras(options CharsExtrasStageOptions, progress func(EntityProgress)) (int64, error) {
	files, err := characterExtraFiles(s.root)
	if err != nil {
		return 0, err
	}
	tag := strings.ToUpper(strings.TrimSpace(options.CountryTag))
	var total int64
	err = inFileOrder(s.ctx, len(files), func(i int) (fileRows[entityRow], error) {
		rel, _ := filepath.Rel(s.root, files[i])
		rel = filepath.ToSlash(rel)
		body, err := os.ReadFile(files[i])
		if err != nil {
			return fileRows[entityRow]{}, fmt.Errorf("read %s: %w", rel, err)
		}
		rows, err := parseCharacterExtra(rel, decodeScriptBytes(body))
		if err != nil {
			return fileRows[entityRow]{rel: rel, invalid: err}, nil
		}
		kept := rows[:0]
		for _, row := range rows {
			isCharacter := row.kind == "character" || strings.HasPrefix(row.kind, "trait:")
			if (isCharacter && !options.ImportCharacters) || (!isCharacter && !options.ImportIdeas) {
				continue
			}
			if tag != "" {
				if row.payload, err = charsExtrasWithOverride(row.payload, tag); err != nil {
					return fileRows[entityRow]{}, fmt.Errorf("copy %s: %w", rel, err)
				}
			}
			kept = append(kept, row)
		}
		return fileRows[entityRow]{rel: rel, rows: kept}, nil
	}, func(i int, file fileRows[entityRow]) error {
		if file.invalid != nil {

			s.warnParse(file.rel, file.invalid)
			return nil
		}
		s.entities = append(s.entities, file.rows...)
		total += int64(len(file.rows))
		if progress != nil {
			progress(EntityProgress{FilesDone: i + 1, FilesTotal: len(files), Rows: total})
		}
		return nil
	})
	if err != nil {
		return total, err
	}

	s.entities = lastWins(s.entities, entityKey)
	return total, nil
}

func charsExtrasWithOverride(raw []byte, tag string) ([]byte, error) {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var data map[string]any
	if decoder.Decode(&data) != nil || data == nil {
		return raw, nil
	}
	data["_country_tag_override"] = tag
	return marshalPGJSON(data)
}

var charsExtrasImageFields = []string{"large_portrait", "small_portrait", "picture", "icon", "blueprint_image", "program_background"}

func (s *session) stageCharsExtrasNeededImages() {
	for i := range s.entities {
		e := &s.entities[i]
		switch e.kind {
		case "character", "idea", "mio", "intel_agency", "specialization", "special_project":
		default:
			continue
		}
		p := e.data()
		for _, field := range charsExtrasImageFields {
			ref := p.str(field)
			if ref == "" {
				continue
			}
			if e.kind == "idea" && field == "picture" {

				s.needImage(importassets.ResolveSprite("idea_picture", ref, s.gfx))
				continue
			}
			if texture, ok := s.gfx[ref]; ok {
				s.needImage(texture)
			} else if charsExtrasIsImagePath(ref) {

				s.needImage(strings.ReplaceAll(ref, `\\`, "/"))
			}
		}
	}
}

func charsExtrasIsImagePath(ref string) bool {
	for _, prefix := range []string{"gfx/", "interface/"} {
		if len(ref) >= len(prefix) && strings.EqualFold(ref[:len(prefix)], prefix) {
			return true
		}
	}
	return false
}

func (s *session) mergeCharsExtras(tx *project.Tx, options CharsExtrasMergeOptions) error {
	characterTable := newKeyedTable(tx, "characters", "tag")
	traitTable := newKeyedTable(tx, "custom_traits", "trait_type", "trait_id")
	specializationTable := newKeyedTable(tx, "special_project_specializations", "specialization_id")
	specialProjectTable := newKeyedTable(tx, "special_projects", "project_token")
	mioTable := newKeyedTable(tx, "military_industrial_organizations", "mio_id")
	agencyTable := newKeyedTable(tx, "intelligence_agencies", "agency_id")
	ideaTable := newKeyedTable(tx, "ideas", "idea_id")
	doctrineTable := newKeyedTable(tx, "doctrines", "entity_type", "entity_key")

	rows := s.entities
	if strings.EqualFold(options.ConflictPolicy, "skip") {

		kept := make([]entityRow, 0, len(rows))
		for _, e := range rows {
			var exists bool
			switch {
			case e.kind == "character":
				_, exists = characterTable.find(e.naturalKey)
			case strings.HasPrefix(e.kind, "trait:"):

				_, exists = traitTable.find(strings.TrimPrefix(e.kind, "trait:"), e.naturalKey)
			case e.kind == "idea":
				_, exists = ideaTable.find(e.naturalKey)
			case e.kind == "mio":
				_, exists = mioTable.find(e.naturalKey)
			case e.kind == "intel_agency":
				_, exists = agencyTable.find(e.naturalKey)
			case e.kind == "doctrine":

				entityType, okType := e.data().text("entity_type")
				entityKey, okKey := e.data().text("entity_key")
				if okType && okKey {
					_, exists = doctrineTable.find(entityType, entityKey)
				}
			case e.kind == "specialization":
				_, exists = specializationTable.find(e.naturalKey)
			case e.kind == "special_project":
				_, exists = specialProjectTable.find(e.naturalKey)
			}
			if !exists {
				kept = append(kept, e)
			}
		}
		rows = kept
	}

	proj := charsExtrasProjectOf(tx)
	s.charsExtrasMergeCharacters(rows, characterTable, proj)
	charsExtrasFillCharacterText(tx)
	s.charsExtrasMergeTraits(rows, traitTable, proj)
	s.charsExtrasMergeSpecializations(rows, specializationTable, proj)
	s.charsExtrasMergeSpecialProjects(rows, specialProjectTable, proj)
	s.charsExtrasMergeMIOs(rows, mioTable, proj)
	s.charsExtrasMergeAgencies(rows, agencyTable, proj)
	s.charsExtrasMergeIdeas(rows, ideaTable, proj)
	charsExtrasMergeDoctrines(rows, doctrineTable)
	return nil
}

var charsExtrasCharacterUpdate = []string{
	"country_tag", "name", "localisation_key", "name_en", "name_zh", "display_name_locs",
	"role_type", "has_advisor_role", "is_replacement", "replaces_vanilla_id",
	"skill", "attack_skill", "defense_skill", "planning_skill", "logistics_skill", "maneuvering_skill", "coordination_skill",
	"military_visible", "traits", "advisor_traits", "advisor_slot", "advisor_idea_token", "advisor_cost",
	"advisor_available", "advisor_visible", "advisor_removal_cost", "advisor_modifier", "advisor_on_add", "advisor_on_remove",
	"gender", "can_be_captured", "always_show_on_actions_tooltip", "ideology", "expire",
	"scientist_data", "scientist_visible", "operative_skill_level", "operative_traits", "operative_nationalities",
	"operative_available", "operative_bg", "raw_extras", "source", "source_file", "large_portrait", "small_portrait",
}

func (s *session) charsExtrasMergeCharacters(rows []entityRow, table *keyedTable, proj charsExtrasProject) {
	for i := range rows {
		e := &rows[i]
		if e.kind != "character" {
			continue
		}
		p := e.data()
		if !proj.inScope(e.naturalKey, p) {
			continue
		}
		countryTag := proj.ownTag(p)
		if prefix, isTag := charsExtrasTagPrefix(e.naturalKey); isTag {
			countryTag = prefix
		}
		locKey := p.str("name")
		if locKey == "" {
			locKey = e.naturalKey
		}
		localisationKey := p.str("localisation_key")
		if localisationKey == "" {
			localisationKey = p.str("name")
		}
		if localisationKey == "" {
			localisationKey = e.naturalKey
		}
		nameEn, ok := s.locText(locKey, "en")
		if !ok {
			nameEn = p.strOr("name_en", e.naturalKey)
		}
		nameZh, ok := s.locText(locKey, "zh")
		if !ok {
			nameZh = p.str("name_zh")
		}
		table.upsert(project.Row{
			"country_tag": countryTag, "tag": e.naturalKey, "name": p.strOr("name", e.naturalKey),
			"localisation_key": localisationKey, "name_en": nameEn, "name_zh": nameZh,
			"display_name_locs": s.locNamesOrEmpty(locKey), "description_locs": map[string]any{},
			"role_type": p.strOr("role_type", "advisor"), "has_advisor_role": p.boolOr("has_advisor_role", false),
			"is_replacement": true, "replaces_vanilla_id": true, "is_portrait_modified": false,
			"skill": p.intOr("skill", 1), "attack_skill": p.intOr("attack_skill", 1),
			"defense_skill": p.intOr("defense_skill", 1), "planning_skill": p.intOr("planning_skill", 1),
			"logistics_skill": p.intOr("logistics_skill", 1), "maneuvering_skill": p.intOr("maneuvering_skill", 1),
			"coordination_skill": p.intOr("coordination_skill", 1),
			"military_visible":   p.str("military_visible"),
			"traits":             p.jsonOr("traits", []any{}), "advisor_traits": p.jsonOr("advisor_traits", []any{}),
			"advisor_slot": p.str("advisor_slot"), "advisor_idea_token": p.str("advisor_idea_token"),
			"advisor_cost":      p.intOr("advisor_cost", 150),
			"advisor_available": p.str("advisor_available"), "advisor_visible": p.str("advisor_visible"),
			"advisor_removal_cost": p.intOr("advisor_removal_cost", -1),
			"advisor_modifier":     p.str("advisor_modifier"), "advisor_on_add": p.str("advisor_on_add"),
			"advisor_on_remove": p.str("advisor_on_remove"), "gender": p.str("gender"),
			"can_be_captured":                p.boolOr("can_be_captured", true),
			"always_show_on_actions_tooltip": charsExtrasNullableBool(p, "always_show_on_actions_tooltip"),
			"ideology":                       p.str("ideology"), "expire": p.str("expire"),
			"scientist_data": p.jsonOr("scientist_data", map[string]any{}), "scientist_visible": p.str("scientist_visible"),
			"operative_skill_level":   p.intOr("operative_skill_level", 1),
			"operative_traits":        p.jsonOr("operative_traits", []any{}),
			"operative_nationalities": p.jsonOr("operative_nationalities", []any{}),
			"operative_available":     p.str("operative_available"), "operative_bg": p.str("operative_bg"),
			"raw_extras": p.jsonOr("raw_extras", map[string]any{}), "source": "imported", "source_file": e.sourceFile,
			"large_portrait": p.str("large_portrait"), "small_portrait": p.str("small_portrait"), "has_obs": false,
		}, charsExtrasCharacterUpdate)
	}
}

func charsExtrasFillCharacterText(tx *project.Tx) {
	current := tx.Rows("characters")
	var next []project.Row
	for i, row := range current {
		if row["description"] != nil && row["desc_en"] != nil && row["desc_zh"] != nil {
			continue
		}
		if next == nil {
			next = append([]project.Row(nil), current...)
		}
		filled := make(project.Row, len(row))
		for column, value := range row {
			filled[column] = value
		}
		for _, column := range []string{"description", "desc_en", "desc_zh"} {
			if filled[column] == nil {
				filled[column] = ""
			}
		}
		next[i] = filled
	}
	if next != nil {
		tx.Replace("characters", next)
	}
}

func (s *session) charsExtrasMergeTraits(rows []entityRow, table *keyedTable, proj charsExtrasProject) {
	for i := range rows {
		e := &rows[i]
		if !strings.HasPrefix(e.kind, "trait:") {
			continue
		}
		p := e.data()
		descKey := e.naturalKey + "_desc"
		table.upsert(project.Row{
			"country_tag": proj.ownTag(p), "trait_type": strings.TrimPrefix(e.kind, "trait:"), "trait_id": e.naturalKey,
			"name_en":           s.charsExtrasTextOrLoc(p, "name_en", e.naturalKey, "en"),
			"name_zh":           s.charsExtrasTextOrLoc(p, "name_zh", e.naturalKey, "zh"),
			"desc_en":           s.charsExtrasTextOrLoc(p, "desc_en", descKey, "en"),
			"desc_zh":           s.charsExtrasTextOrLoc(p, "desc_zh", descKey, "zh"),
			"display_name_locs": s.locNamesOrEmpty(e.naturalKey), "description_locs": s.locNamesOrEmpty(descKey),
			"modifier": p.jsonOr("modifier", map[string]any{}), "ai_will_do": p.str("ai_will_do"),
			"targeted_modifier": p.jsonOr("targeted_modifier", []any{}), "random": p.boolOr("random", true),
			"extra_fields": p.jsonOr("extra_fields", map[string]any{}), "raw_extras": p.jsonOr("raw_extras", map[string]any{}),
		}, []string{"country_tag", "name_en", "name_zh", "desc_en", "desc_zh", "display_name_locs", "description_locs",
			"modifier", "ai_will_do", "targeted_modifier", "random", "extra_fields", "raw_extras"})
	}
}

func (s *session) charsExtrasMergeSpecializations(rows []entityRow, table *keyedTable, proj charsExtrasProject) {
	for i := range rows {
		e := &rows[i]
		if e.kind != "specialization" {
			continue
		}
		p := e.data()
		descKey := e.naturalKey + "_desc"
		table.upsert(project.Row{
			"country_tag": proj.ownTag(p), "specialization_id": e.naturalKey,
			"name_en": s.locStr(e.naturalKey, "en"), "name_zh": s.locStr(e.naturalKey, "zh"),
			"desc_en": s.locStr(descKey, "en"), "desc_zh": s.locStr(descKey, "zh"),
			"display_name_locs": s.locNamesOrEmpty(e.naturalKey), "description_locs": s.locNamesOrEmpty(descKey),
			"color": p.jsonOr("color", []any{}), "blueprint_image": p.str("blueprint_image"),
			"program_background": p.str("program_background"), "facility_model": p.str("facility_model"),
			"sub_specializations": p.jsonOr("sub_specializations", []any{}),
			"raw_extras":          p.jsonOr("raw_extras", map[string]any{}), "source": "imported",
		}, []string{"country_tag", "name_en", "name_zh", "desc_en", "desc_zh", "display_name_locs", "description_locs",
			"color", "blueprint_image", "program_background", "facility_model", "sub_specializations", "raw_extras", "source"})
	}
}

func (s *session) charsExtrasMergeSpecialProjects(rows []entityRow, table *keyedTable, proj charsExtrasProject) {
	for i := range rows {
		e := &rows[i]
		if e.kind != "special_project" {
			continue
		}
		p := e.data()
		descKey := e.naturalKey + "_desc"
		table.upsert(project.Row{
			"country_tag": proj.ownTag(p), "project_token": e.naturalKey, "specialization": p.str("specialization"),
			"name_en": s.locStr(e.naturalKey, "en"), "name_zh": s.locStr(e.naturalKey, "zh"),
			"desc_en": s.locStr(descKey, "en"), "desc_zh": s.locStr(descKey, "zh"),
			"display_name_locs": s.locNamesOrEmpty(e.naturalKey), "description_locs": s.locNamesOrEmpty(descKey),
			"icon": p.str("icon"), "blueprint_image": p.str("blueprint_image"),
			"complexity": p.str("complexity"), "prototype_time": p.str("prototype_time"),
			"project_tags": p.jsonOr("project_tags", []any{}), "special_project_parent": p.jsonOr("special_project_parent", []any{}),
			"allowed": p.str("allowed"), "visible": p.str("visible"), "available": p.str("available"),
			"resource_cost": p.str("resource_cost"), "project_output": p.str("project_output"),
			"generic_prototype_rewards": p.jsonOr("generic_prototype_rewards", []any{}),
			"raw_extras":                p.jsonOr("raw_extras", map[string]any{}), "source": "imported",
		}, []string{"specialization", "name_en", "name_zh", "desc_en", "desc_zh", "display_name_locs", "description_locs",
			"icon", "blueprint_image", "complexity", "prototype_time", "project_tags", "special_project_parent",
			"allowed", "visible", "available", "resource_cost", "project_output", "generic_prototype_rewards", "raw_extras", "source"})
	}
}

func (s *session) charsExtrasMergeMIOs(rows []entityRow, table *keyedTable, proj charsExtrasProject) {
	for i := range rows {
		e := &rows[i]
		if e.kind != "mio" {
			continue
		}
		p := e.data()
		countryTag := proj.ownTag(p)
		if tag, ok := charsExtrasPathText(p, "_scope_verdict", "country_tag"); ok {
			countryTag = tag
		}
		scopeKind := "own"
		if kind, ok := charsExtrasPathText(p, "_scope_verdict", "kind"); ok {
			scopeKind = kind
		}
		shared, _ := charsExtrasPath(p, "_scope_verdict", "shared_country_tags")
		descKey := e.naturalKey + "_desc"
		table.upsert(project.Row{
			"mio_id": e.naturalKey, "include": p.str("include"), "icon": p.str("icon"),
			"equipment_type": p.jsonOr("equipment_type", []any{}), "research_categories": p.jsonOr("research_categories", []any{}),
			"allowed": p.str("allowed"), "available": p.str("available"), "visible": p.str("visible"),
			"traits": p.jsonOr("traits", []any{}), "raw_extras": p.jsonOr("raw_extras", map[string]any{}),
			"country_tag": countryTag, "scope_kind": scopeKind, "shared_country_tags": shared, "source": "imported",
			"name_en": s.locStr(e.naturalKey, "en"), "name_zh": s.locStr(e.naturalKey, "zh"),
			"desc_en": s.locStr(descKey, "en"), "desc_zh": s.locStr(descKey, "zh"),
			"display_name_locs": s.locNamesOrEmpty(e.naturalKey), "description_locs": s.locNamesOrEmpty(descKey),
		}, []string{"include", "icon", "equipment_type", "research_categories", "allowed", "available", "visible",
			"traits", "raw_extras", "country_tag", "scope_kind", "shared_country_tags", "source",
			"name_en", "name_zh", "desc_en", "desc_zh", "display_name_locs", "description_locs"})
	}
}

func (s *session) charsExtrasMergeAgencies(rows []entityRow, table *keyedTable, proj charsExtrasProject) {
	for i := range rows {
		e := &rows[i]
		if e.kind != "intel_agency" {
			continue
		}
		p := e.data()
		locKey := charsExtrasAgencyLocKey(p, e.naturalKey)
		table.upsert(project.Row{
			"agency_id": e.naturalKey, "picture": p.str("picture"), "names": p.jsonOr("names", []any{}),
			"default_trigger": p.str("default_trigger"), "available": p.str("available"),
			"upgrade_branches": p.jsonOr("upgrade_branches", []any{}), "raw_extras": p.jsonOr("raw_extras", map[string]any{}),
			"country_tag": proj.ownTag(p), "scope_kind": "own", "shared_country_tags": nil, "source": "imported",
			"name_en": s.locStr(locKey, "en"), "name_zh": s.locStr(locKey, "zh"),
			"display_name_locs": s.locNamesOrEmpty(locKey), "description_locs": map[string]any{},
		}, []string{"picture", "names", "default_trigger", "available", "upgrade_branches", "raw_extras",
			"country_tag", "source", "name_en", "name_zh", "display_name_locs"})
	}
}

func charsExtrasAgencyLocKey(p payload, naturalKey string) string {
	names, _ := p["names"].([]any)
	for _, item := range names {
		if item == nil {
			continue
		}
		if value := jsonText(item); charsExtrasIsLocToken(value) {
			return value
		}
	}
	return naturalKey
}

func charsExtrasIsLocToken(value string) bool {
	if value == "" {
		return false
	}
	for _, r := range value {
		if r != '_' && r != '.' && !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}

func (s *session) charsExtrasMergeIdeas(rows []entityRow, table *keyedTable, proj charsExtrasProject) {
	for i := range rows {
		e := &rows[i]
		if e.kind != "idea" {
			continue
		}
		p := e.data()
		if !proj.inScope(e.naturalKey, p) {
			continue
		}
		countryTag, scopeKind := "", "global"
		if prefix, isTag := charsExtrasTagPrefix(e.naturalKey); isTag {
			countryTag, scopeKind = prefix, "own"
		}
		if tag, ok := charsExtrasPathText(p, "_scope_verdict", "country_tag"); ok {
			countryTag = tag
		}
		if kind, ok := charsExtrasPathText(p, "_scope_verdict", "kind"); ok {
			scopeKind = kind
		}
		shared, _ := charsExtrasPath(p, "_scope_verdict", "shared_country_tags")
		categoryFlags, _ := p.json("category_flags")
		descKey := e.naturalKey + "_desc"
		table.upsert(project.Row{
			"country_tag": countryTag, "scope_kind": scopeKind, "shared_country_tags": shared,
			"idea_id": e.naturalKey, "idea_type": p.strOr("idea_type", "country"), "name": p.strOr("name", e.naturalKey),
			"picture": p.str("picture"), "modifier": p.jsonOr("modifier", map[string]any{}), "ai_will_do": p.str("ai_will_do"),
			"removal_cost": p.intOr("removal_cost", -1), "auto_attach_on_start": true,
			"allowed": p.str("allowed"), "allowed_civil_war": p.str("allowed_civil_war"),
			"is_designer": p.boolOr("is_designer", false), "research_bonus": p.jsonOr("research_bonus", map[string]any{}),
			"designer_traits": p.jsonOr("designer_traits", []any{}),
			"available":       p.str("available"), "visible": p.str("visible"), "allowed_to_remove": p.str("allowed_to_remove"),
			"cancel": p.str("cancel"), "do_effect": p.str("do_effect"), "on_add": p.str("on_add"), "on_remove": p.str("on_remove"),
			"cancel_if_invalid": p.boolOr("cancel_if_invalid", true), "targeted_modifier": p.jsonOr("targeted_modifier", []any{}),
			"equipment_bonus": p.jsonOr("equipment_bonus", map[string]any{}), "rule": p.jsonOr("rule", map[string]any{}),
			"level": p.intOr("level", 0), "ledger": p.str("ledger"), "cost": p.intOr("cost", 0),
			"raw_extras": p.jsonOr("raw_extras", map[string]any{}), "category_flags": categoryFlags,
			"source": "imported", "source_file": e.sourceFile,
			"name_en": s.locStr(e.naturalKey, "en"), "name_zh": s.locStr(e.naturalKey, "zh"),
			"desc_en": s.locStr(descKey, "en"), "desc_zh": s.locStr(descKey, "zh"),
			"display_name_locs": s.locNamesOrEmpty(e.naturalKey), "description_locs": s.locNamesOrEmpty(descKey),
			"has_obs": false,
		}, []string{"country_tag", "scope_kind", "shared_country_tags", "idea_type", "name", "picture", "modifier",
			"ai_will_do", "removal_cost", "allowed", "allowed_civil_war", "is_designer", "research_bonus", "designer_traits",
			"available", "visible", "allowed_to_remove", "cancel", "do_effect", "on_add", "on_remove", "cancel_if_invalid",
			"targeted_modifier", "equipment_bonus", "rule", "level", "ledger", "cost", "raw_extras", "category_flags",
			"source", "source_file", "name_en", "name_zh", "desc_en", "desc_zh", "display_name_locs", "description_locs"})
	}
}

func charsExtrasMergeDoctrines(rows []entityRow, table *keyedTable) {
	for i := range rows {
		e := &rows[i]
		if e.kind != "doctrine" {
			continue
		}
		p := e.data()
		table.upsert(project.Row{
			"entity_type": p.strOr("entity_type", "grand_doctrine"), "entity_key": p.nullableText("entity_key"),
			"folder_key": p.str("folder_key"), "track_key": p.str("track_key"), "name": p.str("name"),
			"description": p.str("description"), "icon": p.str("icon"), "xp_cost": p.intOr("xp_cost", 0),
			"xp_type": p.str("xp_type"), "xor": p.nullableText("xor"),
			"allow_in_multiple_tracks": charsExtrasNullableBool(p, "allow_in_multiple_tracks"),
			"raw_extras":               p.jsonOr("raw_extras", map[string]any{}), "source_file": p.strOr("source_file", e.sourceFile),
			"source": "imported", "display_name_locs": p.jsonOr("display_name_locs", map[string]any{}),
			"description_locs": p.jsonOr("description_locs", map[string]any{}),
		}, []string{"folder_key", "track_key", "name", "description", "icon", "xp_cost", "xp_type", "xor",
			"allow_in_multiple_tracks", "raw_extras", "source_file", "source"})
	}
}

type charsExtrasProject struct {
	countryTag  any
	countryTags any
}

func charsExtrasProjectOf(tx *project.Tx) charsExtrasProject {
	record := tx.Record()
	var tag any
	if text, ok := record["country_tag"].(string); ok {
		tag = text
	}
	return charsExtrasProject{countryTag: tag, countryTags: record["country_tags"]}
}

func (proj charsExtrasProject) ownTag(p payload) any {
	if tag, ok := p.text("_country_tag_override"); ok {
		return tag
	}
	return proj.countryTag
}

func (proj charsExtrasProject) inScope(naturalKey string, p payload) bool {
	prefix, isTag := charsExtrasTagPrefix(naturalKey)
	if !isTag {
		return true
	}
	if own, ok := proj.ownTag(p).(string); ok && own == prefix {
		return true
	}
	return charsExtrasJSONHas(proj.countryTags, prefix)
}

func charsExtrasTagPrefix(key string) (string, bool) {
	prefix, _, _ := strings.Cut(key, "_")
	if len(prefix) < 2 || len(prefix) > 5 {
		return prefix, false
	}
	for i := 0; i < len(prefix); i++ {
		if prefix[i] < 'A' || prefix[i] > 'Z' {
			return prefix, false
		}
	}
	return prefix, true
}

func charsExtrasJSONHas(value any, text string) bool {
	switch v := value.(type) {
	case []any:
		for _, item := range v {
			if s, ok := item.(string); ok && s == text {
				return true
			}
		}
	case []string:
		for _, item := range v {
			if item == text {
				return true
			}
		}
	case map[string]any:
		_, ok := v[text]
		return ok
	case string:
		return v == text
	}
	return false
}

func charsExtrasPath(p payload, keys ...string) (any, bool) {
	var current any = map[string]any(p)
	for _, key := range keys {
		object, ok := current.(map[string]any)
		if !ok {
			return nil, false
		}
		if current, ok = object[key]; !ok {
			return nil, false
		}
	}
	return current, true
}

func charsExtrasPathText(p payload, keys ...string) (string, bool) {
	value, ok := charsExtrasPath(p, keys...)
	if !ok || value == nil {
		return "", false
	}
	return jsonText(value), true
}

func (s *session) charsExtrasTextOrLoc(p payload, field, key, lang string) string {
	if text, ok := p.text(field); ok {
		return text
	}
	return s.locStr(key, lang)
}

func charsExtrasNullableBool(p payload, key string) any {
	if text, ok := p.text(key); !ok || text == "" {
		return nil
	}
	if yes := p.boolOr(key, true); yes == p.boolOr(key, false) {
		return yes
	}
	return nil
}
