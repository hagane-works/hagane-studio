package exportcheck

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"haganeworks/internal/hoi4/paradox"
	"haganeworks/internal/hoi4/politicsrefs"
	"sort"
	"strconv"
	"strings"
)

var ErrPoliticsReferenceCoverage = errors.New("politics reference coverage incomplete")

type PoliticsCoverageError struct {
	Path   string `json:"path,omitempty"`
	Detail string `json:"detail"`
}

func (e PoliticsCoverageError) Error() string {
	if e.Path == "" {
		return e.Detail
	}
	return e.Path + ": " + e.Detail
}

func (e PoliticsCoverageError) Unwrap() error { return ErrPoliticsReferenceCoverage }

type politicsRegistry struct {
	Raw    map[string]any
	Groups []string
	Roots  []map[string]any
	Subs   []map[string]any
}

func decodePoliticsRegistry(raw []byte) politicsRegistry {
	doc := politicsRegistry{Raw: map[string]any{}}
	if len(raw) == 0 || json.Unmarshal(raw, &doc.Raw) != nil || doc.Raw == nil {
		doc.Raw = map[string]any{}
		return doc
	}
	if values, ok := doc.Raw["groups"].([]any); ok {
		for _, value := range values {
			if id := strings.TrimSpace(stringFrom(value)); id != "" {
				doc.Groups = append(doc.Groups, id)
			}
		}
	}
	if values, ok := doc.Raw["root_definitions"].([]any); ok {
		for _, value := range values {
			if item, ok := value.(map[string]any); ok && strings.TrimSpace(stringFrom(item["id"])) != "" {
				doc.Roots = append(doc.Roots, item)
			}
		}
	}
	if values, ok := doc.Raw["sub_ideologies"].([]any); ok {
		for _, value := range values {
			switch item := value.(type) {
			case string:
				if id := strings.TrimSpace(item); id != "" {
					doc.Subs = append(doc.Subs, map[string]any{"id": id})
				}
			case map[string]any:
				if strings.TrimSpace(stringFrom(item["id"])) != "" {
					doc.Subs = append(doc.Subs, item)
				}
			}
		}
	}
	return doc
}

func politicsSource(value any, fallback politicsrefs.SourceOwnership) politicsrefs.SourceOwnership {
	switch strings.TrimSpace(stringFrom(value)) {
	case string(politicsrefs.SourceProjectManaged), "created":
		return politicsrefs.SourceProjectManaged
	case string(politicsrefs.SourceVanilla):
		return politicsrefs.SourceVanilla
	case string(politicsrefs.SourceImported):
		return politicsrefs.SourceImported
	case string(politicsrefs.SourceBaseMod), "basemod":
		return politicsrefs.SourceBaseMod
	default:
		return fallback
	}
}

func politicsDisplayName(item map[string]any, id string) string {
	for _, key := range []string{"name", "display_name", "name_en", "name_zh"} {
		if value := strings.TrimSpace(stringFrom(item[key])); value != "" {
			return value
		}
	}
	return id
}

type ideologyDefinition struct {
	Kind           politicsrefs.IdeologyKind
	ScriptID       string
	RootID         string
	RandomDisabled bool
}

func parseIdeologyDefinitions(content string) ([]ideologyDefinition, error) {
	block, recoveries, err := paradox.Parse(content)
	if err != nil || block == nil {
		return nil, fmt.Errorf("parse ideology definitions: %w", err)
	}
	for _, recovery := range recoveries {
		if recovery.Code == paradox.RecoveryUnclosedBlock {
			return nil, fmt.Errorf("parse ideology definitions: unclosed block")
		}
	}
	outer := paradoxBlock(block.Entries, "ideologies")
	if outer == nil {
		return nil, nil
	}
	definitions := make([]ideologyDefinition, 0)
	for _, root := range outer.Entries {
		if root.Key == nil || root.Block == nil {
			continue
		}
		rootID := strings.TrimSpace(*root.Key)
		if rootID == "" {
			continue
		}
		definitions = append(definitions, ideologyDefinition{Kind: politicsrefs.KindRoot, ScriptID: rootID})
		types := paradoxBlock(root.Block.Entries, "types")
		if types == nil {
			continue
		}
		for _, subtype := range types.Entries {
			if subtype.Key == nil {
				continue
			}
			if id := strings.TrimSpace(*subtype.Key); id != "" {
				disabled := false
				if subtype.Block != nil {
					for _, field := range subtype.Block.Entries {
						if field.Key != nil && *field.Key == "can_be_randomly_selected" && field.Block == nil &&
							strings.EqualFold(strings.TrimSpace(field.Scalar.StrValue()), "no") {
							disabled = true
						}
					}
				}
				definitions = append(definitions, ideologyDefinition{Kind: politicsrefs.KindSubtype, ScriptID: id, RootID: rootID, RandomDisabled: disabled})
			}
		}
	}
	return definitions, nil
}

type rawPoliticsFile struct {
	ID       int64
	Path     string
	Content  string
	Source   string
	Imported bool
}

func loadRawPoliticsFiles(_ context.Context, db *Store, _ int64) ([]rawPoliticsFile, error) {
	rows, err := db.rows("project_raw_files")
	if err != nil {
		return nil, err
	}
	var out []rawPoliticsFile
	for _, row := range rows {
		if colBool(row, "is_binary") {
			continue
		}
		file := rawPoliticsFile{ID: colInt(row, "id"), Path: colStr(row, "dest_path"), Content: colStr(row, "content"), Source: colStr(row, "source")}
		if file.Source == "" {
			file.Source = "created"
		}
		file.Path = strings.ReplaceAll(strings.TrimSpace(file.Path), "\\", "/")
		file.Imported = file.Source == "imported"
		out = append(out, file)
	}
	return out, nil
}

func addPoliticsIdeology(
	target map[string]politicsrefs.Ideology,
	conflicts *[]politicsrefs.IdeologyConflict,
	ideology politicsrefs.Ideology,
) {
	ideology.ScriptID = strings.TrimSpace(ideology.ScriptID)
	ideology.RootID = strings.TrimSpace(ideology.RootID)
	if ideology.ScriptID == "" {
		return
	}
	if previous, found := target[ideology.ScriptID]; found {
		if previous.Kind != ideology.Kind {
			*conflicts = append(*conflicts, politicsrefs.IdeologyConflict{
				ScriptID: ideology.ScriptID,
				Kind:     ideology.Kind,
				RootID:   ideology.RootID,
				Reason:   politicsrefs.ConflictDuplicateID,
				Detail: fmt.Sprintf("%q is declared both as %s and as %s",
					ideology.ScriptID, previous.Kind, ideology.Kind),
			})
			return
		}

		if previous.Source == politicsrefs.SourceProjectManaged && ideology.Source != politicsrefs.SourceProjectManaged {
			target[ideology.ScriptID] = ideology
		}
		return
	}
	target[ideology.ScriptID] = ideology
}

func ideologyTarget(known map[string]politicsrefs.Ideology, id string) (politicsrefs.Target, bool) {
	ideology, found := known[strings.TrimSpace(id)]
	if !found {
		return politicsrefs.Target{}, false
	}
	return politicsrefs.Target{Kind: ideology.Kind, ScriptID: ideology.ScriptID}, true
}

func appendPoliticsRef(refs *[]politicsrefs.Reference, known map[string]politicsrefs.Ideology,
	id string, entity politicsrefs.EntityRef, context string) {
	target, found := ideologyTarget(known, id)
	if !found {
		return
	}
	*refs = append(*refs, politicsrefs.Reference{Target: target, Entity: entity, Context: context})
}

func jsonObjectKeys(raw []byte) []string {
	var value map[string]any
	if len(raw) == 0 || json.Unmarshal(raw, &value) != nil {
		return nil
	}
	out := make([]string, 0, len(value))
	for key := range value {
		out = append(out, key)
	}
	sort.Strings(out)
	return out
}

func scriptIdeologyTokens(content string, known map[string]politicsrefs.Ideology) []struct{ ID, Context string } {
	block, recoveries, err := paradox.Parse(content)
	failed := err != nil || block == nil
	for _, recovery := range recoveries {
		failed = failed || recovery.Code == paradox.RecoveryUnclosedBlock
	}
	if failed {

		var out []struct{ ID, Context string }
		for id := range known {
			if containsScriptToken(content, id) {
				out = append(out, struct{ ID, Context string }{id, "conservative token match in unparsed script"})
			}
		}
		sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
		return out
	}
	seen := map[string]bool{}
	var out []struct{ ID, Context string }
	var walk func([]*paradox.Entry, string)
	walk = func(entries []*paradox.Entry, path string) {
		for _, entry := range entries {
			name := "value"
			if entry.Key != nil {
				name = *entry.Key
			}
			current := name
			if path != "" {
				current = path + "." + name
			}
			if entry.Block != nil {
				walk(entry.Block.Entries, current)
				continue
			}
			id := strings.TrimSpace(entry.Scalar.StrValue())
			if _, found := known[id]; !found {
				continue
			}
			key := id + "\x00" + current
			if !seen[key] {
				seen[key] = true
				out = append(out, struct{ ID, Context string }{id, current})
			}
		}
	}
	walk(block.Entries, "")
	return out
}

func containsScriptToken(content, token string) bool {
	if token == "" {
		return false
	}
	for start := 0; ; {
		index := strings.Index(content[start:], token)
		if index < 0 {
			return false
		}
		index += start
		leftOK := index == 0 || !isScriptIDByte(content[index-1])
		right := index + len(token)
		rightOK := right == len(content) || !isScriptIDByte(content[right])
		if leftOK && rightOK {
			return true
		}
		start = index + len(token)
	}
}

func isScriptIDByte(value byte) bool {
	return value == '_' || value >= '0' && value <= '9' || value >= 'A' && value <= 'Z' || value >= 'a' && value <= 'z'
}

func scriptHasGovernment(content string) []string {
	block, _, err := paradox.Parse(content)
	if err != nil || block == nil {
		return nil
	}
	seen := map[string]bool{}
	var out []string
	var walk func([]*paradox.Entry)
	walk = func(entries []*paradox.Entry) {
		for _, entry := range entries {
			if entry.Key != nil && *entry.Key == "has_government" && entry.Block == nil {
				if id := strings.TrimSpace(entry.Scalar.StrValue()); id != "" && !seen[id] {
					seen[id] = true
					out = append(out, id)
				}
			}
			if entry.Block != nil {
				walk(entry.Block.Entries)
			}
		}
	}
	walk(block.Entries)
	sort.Strings(out)
	return out
}

func nameGroupGovernmentRequirements(content string) map[string][]string {
	block, _, err := paradox.Parse(content)
	if err != nil || block == nil {
		return nil
	}
	out := map[string][]string{}
	for _, entry := range block.Entries {
		if entry.Key == nil || entry.Block == nil {
			continue
		}
		if values := scriptHasGovernment(renderParadoxEntries(entry.Block.Entries)); len(values) > 0 {
			out[strings.TrimSpace(*entry.Key)] = values
		}
	}
	return out
}

func renderParadoxEntries(entries []*paradox.Entry) string {
	var builder strings.Builder
	var writeEntries func([]*paradox.Entry)
	writeEntries = func(values []*paradox.Entry) {
		for _, entry := range values {
			if entry.Key != nil {
				builder.WriteString(*entry.Key)
				builder.WriteString(" = ")
			}
			if entry.Block != nil {
				builder.WriteString("{ ")
				writeEntries(entry.Block.Entries)
				builder.WriteString("} ")
			} else {
				builder.WriteString(entry.Scalar.StrValue())
				builder.WriteByte(' ')
			}
		}
	}
	writeEntries(entries)
	return builder.String()
}

func loadPoliticsSnapshot(ctx context.Context, db *Store, projectID int64, _ bool) (politicsrefs.Snapshot, politicsRegistry, error) {
	record := db.p.Record()
	if record == nil {
		return politicsrefs.Snapshot{}, politicsRegistry{}, ErrProjectMissing
	}
	projectKey := colStr(record, "uuid")
	if projectKey == "" {
		projectKey = strconv.FormatInt(db.p.ID(), 10)
	}
	primaryTag := colStr(record, "country_tag")
	registryRaw := colBytes(record, "ideology_registry")
	iconRaw := colBytes(record, "ideology_icon_paths")
	dependenciesRaw := colBytes(record, "dependencies")
	countryTagsRaw := colBytes(record, "country_tags")
	registry := decodePoliticsRegistry(registryRaw)
	rawFiles, err := loadRawPoliticsFiles(ctx, db, projectID)
	if err != nil {
		return politicsrefs.Snapshot{}, registry, err
	}
	known := map[string]politicsrefs.Ideology{}

	conflicts := []politicsrefs.IdeologyConflict{}

	ideologyUsage := map[string]int{}
	for root := range vanillaIdeologyGroupIDs() {
		known[root] = politicsrefs.Ideology{Kind: politicsrefs.KindRoot, ScriptID: root, DisplayName: root, Source: politicsrefs.SourceVanilla}
	}
	importedEvidence := map[string]ideologyDefinition{}
	for _, file := range rawFiles {
		path := strings.ToLower(file.Path)
		if !file.Imported || !strings.HasPrefix(path, "common/ideologies/") || !strings.HasSuffix(path, ".txt") {
			continue
		}
		definitions, parseErr := parseIdeologyDefinitions(file.Content)
		if parseErr != nil {
			return politicsrefs.Snapshot{}, registry, PoliticsCoverageError{Path: file.Path, Detail: "cannot classify imported ideology ownership safely"}
		}
		for _, definition := range definitions {
			importedEvidence[definition.ScriptID] = definition
			addPoliticsIdeology(known, &conflicts, politicsrefs.Ideology{Kind: definition.Kind, ScriptID: definition.ScriptID,
				RootID: definition.RootID, DisplayName: definition.ScriptID, Source: politicsrefs.SourceImported,
				RandomDisabled: definition.RandomDisabled})
		}
	}
	var deps []any
	_ = json.Unmarshal(dependenciesRaw, &deps)
	hasBaseMod := len(deps) > 0
	rootDefinitionIDs := map[string]bool{}
	for _, item := range registry.Roots {
		id := strings.TrimSpace(stringFrom(item["id"]))
		rootDefinitionIDs[id] = true
		source := politicsSource(item["source"], politicsrefs.SourceProjectManaged)
		if vanillaIdeologyGroupIDs()[id] {
			source = politicsrefs.SourceVanilla
		} else if _, imported := importedEvidence[id]; imported {
			source = politicsrefs.SourceImported
		}
		addPoliticsIdeology(known, &conflicts, politicsrefs.Ideology{Kind: politicsrefs.KindRoot, ScriptID: id,
			DisplayName: politicsDisplayName(item, id), Source: source})
		body := strings.TrimSpace(stringFrom(item["body"]))
		if source == politicsrefs.SourceProjectManaged && body != "" {
			definitions, parseErr := parseIdeologyDefinitions("ideologies = {\n" + id + " = {\n" + body + "\n}\n}\n")
			if parseErr != nil {
				return politicsrefs.Snapshot{}, registry, PoliticsCoverageError{Path: "ideology_registry.root_definitions." + id, Detail: "cannot classify project ROOT subtypes safely"}
			}
			for _, definition := range definitions {
				if definition.Kind != politicsrefs.KindSubtype {
					continue
				}
				addPoliticsIdeology(known, &conflicts, politicsrefs.Ideology{Kind: definition.Kind, ScriptID: definition.ScriptID,
					RootID: definition.RootID, DisplayName: definition.ScriptID, Source: source,
					RandomDisabled: definition.RandomDisabled})
			}
		}
	}
	for _, id := range registry.Groups {
		if id == "" || rootDefinitionIDs[id] {
			continue
		}
		source := politicsrefs.SourceImported
		if hasBaseMod {
			source = politicsrefs.SourceBaseMod
		}
		if vanillaIdeologyGroupIDs()[id] {
			source = politicsrefs.SourceVanilla
		}
		addPoliticsIdeology(known, &conflicts, politicsrefs.Ideology{Kind: politicsrefs.KindRoot, ScriptID: id,
			DisplayName: id, Source: source})
	}
	for _, item := range registry.Subs {
		id := strings.TrimSpace(stringFrom(item["id"]))
		if vanillaSubIdeologyIDs()[id] {
			continue
		}
		rootID := strings.TrimSpace(stringFrom(item["group"]))
		source := politicsSource(item["source"], politicsrefs.SourceProjectManaged)
		if _, imported := importedEvidence[id]; imported {
			source = politicsrefs.SourceImported
		}
		randomDisabled, _ := item["random_disabled"].(bool)
		addPoliticsIdeology(known, &conflicts, politicsrefs.Ideology{Kind: politicsrefs.KindSubtype, ScriptID: id,
			RootID: rootID, DisplayName: politicsDisplayName(item, id), Source: source,
			RandomDisabled: randomDisabled})
	}

	snapshot := politicsrefs.Snapshot{ProjectID: projectKey}
	for _, ideology := range known {
		snapshot.Ideologies = append(snapshot.Ideologies, ideology)
	}
	sort.Slice(snapshot.Ideologies, func(i, j int) bool {
		if snapshot.Ideologies[i].Kind != snapshot.Ideologies[j].Kind {
			return snapshot.Ideologies[i].Kind < snapshot.Ideologies[j].Kind
		}
		return snapshot.Ideologies[i].ScriptID < snapshot.Ideologies[j].ScriptID
	})
	for _, id := range jsonObjectKeys(iconRaw) {
		appendPoliticsRef(&snapshot.References, known, id,
			politicsrefs.EntityRef{Kind: "project", ID: projectKey, Path: "ideology_icon_paths." + id}, "project ideology icon")
	}

	countryRoots := map[string]string{}
	countryRows, err := db.rows("countries")
	if err != nil {
		return politicsrefs.Snapshot{}, registry, err
	}
	for _, row := range countryRows {
		id, tag, ideology := colInt(row, "id"), colStr(row, "tag"), colStr(row, "ideology")
		namesRaw, flagsRaw := colBytes(row, "ideology_name_locs"), colBytes(row, "flag_paths_by_ideology")
		countryRoots[strings.ToUpper(tag)] = strings.TrimSpace(ideology)
		entity := politicsrefs.EntityRef{Kind: "country", ID: strconv.FormatInt(id, 10), Path: "countries[" + tag + "]"}
		appendPoliticsRef(&snapshot.References, known, ideology, entity, "country ideology")
		for _, key := range append(jsonObjectKeys(namesRaw), jsonObjectKeys(flagsRaw)...) {
			appendPoliticsRef(&snapshot.References, known, key, entity, "country ideology-specific localisation or flag")
		}
	}

	characterRows, err := db.rows("characters")
	if err != nil {
		return politicsrefs.Snapshot{}, registry, err
	}
	for _, row := range characterRows {
		id, tag, ideology := colInt(row, "id"), colStr(row, "tag"), colStr(row, "ideology")
		appendPoliticsRef(&snapshot.References, known, ideology,
			politicsrefs.EntityRef{Kind: "character", ID: strconv.FormatInt(id, 10), Path: "characters[" + tag + "].ideology"},
			"country leader ideology")

		if trimmed := strings.TrimSpace(ideology); trimmed != "" {
			ideologyUsage[trimmed]++
		}
	}

	historyRows, err := db.rows("history_files")
	if err != nil {
		return politicsrefs.Snapshot{}, registry, err
	}
	for _, row := range historyRows {
		id, tag := colInt(row, "id"), colStr(row, "country_tag")
		rulingParty, rawScript := colStr(row, "set_politics_ruling_party"), colStr(row, "raw_script")
		popularities := colBytes(row, "set_popularities")
		if strings.TrimSpace(rulingParty) != "" {
			countryRoots[strings.ToUpper(tag)] = strings.TrimSpace(rulingParty)
		}
		entity := politicsrefs.EntityRef{Kind: "history_file", ID: strconv.FormatInt(id, 10), Path: "history/countries/" + tag}
		appendPoliticsRef(&snapshot.References, known, rulingParty, entity, "set_politics.ruling_party")
		for _, key := range jsonObjectKeys(popularities) {
			appendPoliticsRef(&snapshot.References, known, key, entity, "set_popularities")
		}
		for _, token := range scriptIdeologyTokens(rawScript, known) {
			appendPoliticsRef(&snapshot.References, known, token.ID, entity, "raw_script."+token.Context)
		}
	}

	bookmarkRows, err := db.rows("bookmarks")
	if err != nil {
		return politicsrefs.Snapshot{}, registry, err
	}
	for _, row := range bookmarkRows {
		id, bookmarkID := colInt(row, "id"), colStr(row, "bookmark_id")
		countriesRaw, extrasRaw := colBytes(row, "selectable_countries"), colBytes(row, "raw_extras")
		entity := politicsrefs.EntityRef{Kind: "bookmark", ID: strconv.FormatInt(id, 10), Path: "common/bookmarks/" + bookmarkID}
		var countries []map[string]any
		_ = json.Unmarshal(countriesRaw, &countries)
		for index, country := range countries {
			appendPoliticsRef(&snapshot.References, known, stringFrom(country["ideology"]), entity,
				"selectable_countries["+strconv.Itoa(index)+"].ideology")
		}
		for _, token := range scriptIdeologyTokens(string(extrasRaw), known) {
			appendPoliticsRef(&snapshot.References, known, token.ID, entity, "raw_extras."+token.Context)
		}
	}

	proj := relProject{
		countryTag:  primaryTag,
		countryTags: scanJSONStrings(countryTagsRaw),
		isSubmod:    hasBaseMod,
	}
	for _, source := range mustPoliticsRelationSources(ctx, db, projectID, proj, &err) {
		entity := politicsrefs.EntityRef{Kind: source.kind, ID: strconv.FormatInt(source.sourceID, 10),
			Path: source.kind + "[" + source.sourceName + "]." + source.fieldName}
		for _, token := range scriptIdeologyTokens(source.text, known) {
			appendPoliticsRef(&snapshot.References, known, token.ID, entity, token.Context)
		}
		if source.tag == nil {
			continue
		}
		customRoot := countryRoots[strings.ToUpper(*source.tag)]
		ideology, custom := known[customRoot]
		if !custom || ideology.Kind != politicsrefs.KindRoot || ideology.Source != politicsrefs.SourceProjectManaged {
			continue
		}
		for _, expected := range scriptHasGovernment(source.text) {
			if expected != customRoot {
				snapshot.Dependencies = append(snapshot.Dependencies, politicsrefs.RootDependency{
					CustomRootID: customRoot, ExpectedRoot: expected, ReferenceKey: "has_government",
					Disposition: politicsrefs.DependencyIntentional, Entity: entity,
				})
			}
		}
	}
	if err != nil {
		return politicsrefs.Snapshot{}, registry, err
	}

	nameGroups := map[string][]string{}
	for _, file := range rawFiles {
		path := strings.ToLower(file.Path)
		if strings.HasPrefix(path, "common/units/names_divisions/") && strings.HasSuffix(path, ".txt") {
			for group, expected := range nameGroupGovernmentRequirements(file.Content) {
				nameGroups[group] = append(nameGroups[group], expected...)
			}
		}
		if strings.HasPrefix(path, "common/ideologies/") {
			continue
		}
		entity := politicsrefs.EntityRef{Kind: "raw_file", ID: strconv.FormatInt(file.ID, 10), Path: file.Path}
		for _, token := range scriptIdeologyTokens(file.Content, known) {
			appendPoliticsRef(&snapshot.References, known, token.ID, entity, token.Context)
		}
	}
	divisionRows, err := db.rows("division_templates")
	if err != nil {
		return politicsrefs.Snapshot{}, registry, err
	}
	for _, row := range divisionRows {
		id, tag, file, group := colInt(row, "id"), colStr(row, "country_tag"), colStr(row, "oob_file"), colStr(row, "division_names_group")
		if group == "" {
			continue
		}
		customRoot := countryRoots[strings.ToUpper(tag)]
		ideology, custom := known[customRoot]
		if !custom || ideology.Kind != politicsrefs.KindRoot || ideology.Source != politicsrefs.SourceProjectManaged {
			continue
		}
		entity := politicsrefs.EntityRef{Kind: "division_template", ID: strconv.FormatInt(id, 10),
			Path: "history/units/" + file + ".txt.division_names_group"}
		expectedRoots := nameGroups[group]
		if len(expectedRoots) == 0 {
			expectedRoots = []string{"unresolved_name_group_requirement"}
		}
		for _, expected := range expectedRoots {
			if expected != customRoot {
				snapshot.Dependencies = append(snapshot.Dependencies, politicsrefs.RootDependency{
					CustomRootID: customRoot, ExpectedRoot: expected, ReferenceKey: "military_name_group:" + group,
					Disposition: politicsrefs.DependencyRequired, Entity: entity,
				})
			}
		}
	}

	snapshot.Conflicts = conflicts
	snapshot = politicsrefs.PartitionValid(snapshot)

	for i := range snapshot.Conflicts {
		snapshot.Conflicts[i].UsageCount = ideologyUsage[snapshot.Conflicts[i].ScriptID]
	}
	if err := politicsrefs.ValidateSnapshot(snapshot); err != nil {
		return politicsrefs.Snapshot{}, registry, err
	}
	return snapshot, registry, nil
}

func mustPoliticsRelationSources(ctx context.Context, db *Store, projectID int64, proj relProject, errp *error) []relScriptSource {
	values, err := relQuerySourcesWith(ctx, db, projectID, proj)
	if err != nil {
		*errp = err
		return nil
	}
	return values
}

func (s *Store) PoliticsReferences(ctx context.Context, projectID int64) (politicsrefs.Snapshot, error) {
	snapshot, _, err := loadPoliticsSnapshot(ctx, s, projectID, false)
	return snapshot, err
}
