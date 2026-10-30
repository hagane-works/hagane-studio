package countries

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"

	"haganeworks/internal/apiutil"
	"haganeworks/internal/hoi4/emit/worldout"
	"haganeworks/internal/hoi4/ideologysource"
	"haganeworks/internal/hoi4/paradox"
	"haganeworks/internal/hoi4/politicsrefs"
	"haganeworks/internal/project"
	"haganeworks/internal/reftables"
	"haganeworks/internal/workbench"
)

var politicsDeleteManager = politicsrefs.NewManager()

var errPoliticsReferenceCoverage = errors.New("politics reference coverage incomplete")

type politicsCoverageError struct {
	Path   string
	Detail string
}

func (e politicsCoverageError) Error() string {
	if e.Path == "" {
		return e.Detail
	}
	return e.Path + ": " + e.Detail
}

func (e politicsCoverageError) Unwrap() error { return errPoliticsReferenceCoverage }

type politicsCommandError struct {
	Code   string `json:"code"`
	Detail string `json:"detail"`
}

func (e politicsCommandError) Error() string { return e.Detail }

var politicsTables = []string{
	"project_raw_files", "countries", "characters", "history_files", "bookmarks",
	"focus_trees", "focus_nodes", "events", "decisions", "ideas", "division_templates",
	"project_obs_assets",
}

type politicsRegistry struct {
	Raw    map[string]any
	Groups []string
	Roots  []map[string]any
	Subs   []map[string]any
}

func decodePoliticsRegistry(value any) politicsRegistry {
	doc := politicsRegistry{Raw: map[string]any{}}
	raw, err := json.Marshal(value)
	if err != nil || json.Unmarshal(raw, &doc.Raw) != nil || doc.Raw == nil {
		doc.Raw = map[string]any{}
		return doc
	}
	if values, ok := doc.Raw["groups"].([]any); ok {
		for _, value := range values {
			if id := strings.TrimSpace(str(value)); id != "" {
				doc.Groups = append(doc.Groups, id)
			}
		}
	}
	if values, ok := doc.Raw["root_definitions"].([]any); ok {
		for _, value := range values {
			if item, ok := value.(map[string]any); ok && strings.TrimSpace(str(item["id"])) != "" {
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
				if strings.TrimSpace(str(item["id"])) != "" {
					doc.Subs = append(doc.Subs, item)
				}
			}
		}
	}
	return doc
}

func politicsSource(value any, fallback politicsrefs.SourceOwnership) politicsrefs.SourceOwnership {
	switch strings.TrimSpace(str(value)) {
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
		if value := strings.TrimSpace(str(item[key])); value != "" {
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

func paradoxBlock(entries []*paradox.Entry, key string) *paradox.Block {
	for _, e := range entries {
		if e.Key != nil && *e.Key == key && e.Block != nil {
			return e.Block
		}
	}
	return nil
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
	Imported bool
}

func loadRawPoliticsFiles(rr rowReader) []rawPoliticsFile {
	var out []rawPoliticsFile
	for _, row := range rr.rows("project_raw_files") {
		if row["is_binary"] == true {
			continue
		}
		out = append(out, rawPoliticsFile{
			ID:       project.Int64(row["id"]),
			Path:     strings.ReplaceAll(strings.TrimSpace(str(row["dest_path"])), "\\", "/"),
			Content:  str(row["content"]),
			Imported: str(row["source"]) == "imported",
		})
	}
	return out
}

func addPoliticsIdeology(target map[string]politicsrefs.Ideology, conflicts *[]politicsrefs.IdeologyConflict, ideology politicsrefs.Ideology) {
	ideology.ScriptID = strings.TrimSpace(ideology.ScriptID)
	ideology.RootID = strings.TrimSpace(ideology.RootID)
	if ideology.ScriptID == "" {
		return
	}
	if previous, found := target[ideology.ScriptID]; found {
		if previous.Kind != ideology.Kind {
			*conflicts = append(*conflicts, politicsrefs.IdeologyConflict{
				ScriptID: ideology.ScriptID, Kind: ideology.Kind, RootID: ideology.RootID,
				Reason: politicsrefs.ConflictDuplicateID,
				Detail: fmt.Sprintf("%q is declared both as %s and as %s", ideology.ScriptID, previous.Kind, ideology.Kind),
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

func appendPoliticsRef(refs *[]politicsrefs.Reference, known map[string]politicsrefs.Ideology, id string, entity politicsrefs.EntityRef, context string) {
	ideology, found := known[strings.TrimSpace(id)]
	if !found {
		return
	}
	*refs = append(*refs, politicsrefs.Reference{
		Target: politicsrefs.Target{Kind: ideology.Kind, ScriptID: ideology.ScriptID}, Entity: entity, Context: context,
	})
}

func objectKeys(value any) []string {
	m, _ := value.(map[string]any)
	out := make([]string, 0, len(m))
	for key := range m {
		out = append(out, key)
	}
	sort.Strings(out)
	return out
}

type ideologyToken struct{ ID, Context string }

func scriptIdeologyTokens(content string, known map[string]politicsrefs.Ideology) []ideologyToken {
	block, recoveries, err := paradox.Parse(content)
	failed := err != nil || block == nil
	for _, recovery := range recoveries {
		failed = failed || recovery.Code == paradox.RecoveryUnclosedBlock
	}
	if failed {
		var out []ideologyToken
		for id := range known {
			if containsScriptToken(content, id) {
				out = append(out, ideologyToken{id, "conservative token match in unparsed script"})
			}
		}
		sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
		return out
	}
	seen := map[string]bool{}
	var out []ideologyToken
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
			if key := id + "\x00" + current; !seen[key] {
				seen[key] = true
				out = append(out, ideologyToken{id, current})
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

func jsonText(value any) string {
	if value == nil {
		return ""
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return ""
	}
	return string(raw)
}

func projectKey(record project.Row) string {
	if uuid := str(record["uuid"]); uuid != "" {
		return uuid
	}
	return strconv.FormatInt(project.Int64(record["id"]), 10)
}

func loadPoliticsSnapshot(rr rowReader) (politicsrefs.Snapshot, politicsRegistry, error) {
	record := rr.record()
	key := projectKey(record)
	registry := decodePoliticsRegistry(record["ideology_registry"])
	rawFiles := loadRawPoliticsFiles(rr)
	known := map[string]politicsrefs.Ideology{}
	conflicts := []politicsrefs.IdeologyConflict{}
	ideologyUsage := map[string]int{}
	for _, root := range reftables.Ideologies().Groups {
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
			return politicsrefs.Snapshot{}, registry, politicsCoverageError{Path: file.Path, Detail: "cannot classify imported ideology ownership safely"}
		}
		for _, definition := range definitions {
			importedEvidence[definition.ScriptID] = definition
			addPoliticsIdeology(known, &conflicts, politicsrefs.Ideology{Kind: definition.Kind, ScriptID: definition.ScriptID,
				RootID: definition.RootID, DisplayName: definition.ScriptID, Source: politicsrefs.SourceImported,
				RandomDisabled: definition.RandomDisabled})
		}
	}
	deps, _ := record["dependencies"].([]any)
	hasBaseMod := len(deps) > 0
	rootDefinitionIDs := map[string]bool{}
	for _, item := range registry.Roots {
		id := strings.TrimSpace(str(item["id"]))
		rootDefinitionIDs[id] = true
		source := politicsSource(item["source"], politicsrefs.SourceProjectManaged)
		if reftables.Ideologies().IsGroup(id) {
			source = politicsrefs.SourceVanilla
		} else if _, imported := importedEvidence[id]; imported {
			source = politicsrefs.SourceImported
		}
		addPoliticsIdeology(known, &conflicts, politicsrefs.Ideology{Kind: politicsrefs.KindRoot, ScriptID: id,
			DisplayName: politicsDisplayName(item, id), Source: source})
		body := strings.TrimSpace(str(item["body"]))
		if source == politicsrefs.SourceProjectManaged && body != "" {
			definitions, parseErr := parseIdeologyDefinitions("ideologies = {\n" + id + " = {\n" + body + "\n}\n}\n")
			if parseErr != nil {
				return politicsrefs.Snapshot{}, registry, politicsCoverageError{Path: "ideology_registry.root_definitions." + id, Detail: "cannot classify project ROOT subtypes safely"}
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
		if reftables.Ideologies().IsGroup(id) {
			source = politicsrefs.SourceVanilla
		}
		addPoliticsIdeology(known, &conflicts, politicsrefs.Ideology{Kind: politicsrefs.KindRoot, ScriptID: id, DisplayName: id, Source: source})
	}
	for _, item := range registry.Subs {
		id := strings.TrimSpace(str(item["id"]))
		if reftables.Ideologies().IsSub(id) {
			continue
		}
		source := politicsSource(item["source"], politicsrefs.SourceProjectManaged)
		if _, imported := importedEvidence[id]; imported {
			source = politicsrefs.SourceImported
		}
		randomDisabled, _ := item["random_disabled"].(bool)
		addPoliticsIdeology(known, &conflicts, politicsrefs.Ideology{Kind: politicsrefs.KindSubtype, ScriptID: id,
			RootID: strings.TrimSpace(str(item["group"])), DisplayName: politicsDisplayName(item, id), Source: source,
			RandomDisabled: randomDisabled})
	}

	snapshot := politicsrefs.Snapshot{ProjectID: key}
	for _, ideology := range known {
		snapshot.Ideologies = append(snapshot.Ideologies, ideology)
	}
	sort.Slice(snapshot.Ideologies, func(i, j int) bool {
		if snapshot.Ideologies[i].Kind != snapshot.Ideologies[j].Kind {
			return snapshot.Ideologies[i].Kind < snapshot.Ideologies[j].Kind
		}
		return snapshot.Ideologies[i].ScriptID < snapshot.Ideologies[j].ScriptID
	})
	for _, id := range objectKeys(record["ideology_icon_paths"]) {
		appendPoliticsRef(&snapshot.References, known, id,
			politicsrefs.EntityRef{Kind: "project", ID: key, Path: "ideology_icon_paths." + id}, "project ideology icon")
	}
	countryRoots := map[string]string{}
	for _, row := range rr.rows("countries") {
		tag := str(row["tag"])
		ideology := str(row["ideology"])
		countryRoots[strings.ToUpper(tag)] = strings.TrimSpace(ideology)
		entity := politicsrefs.EntityRef{Kind: "country", ID: strconv.FormatInt(project.Int64(row["id"]), 10), Path: "countries[" + tag + "]"}
		appendPoliticsRef(&snapshot.References, known, ideology, entity, "country ideology")
		for _, k := range append(objectKeys(row["ideology_name_locs"]), objectKeys(row["flag_paths_by_ideology"])...) {
			appendPoliticsRef(&snapshot.References, known, k, entity, "country ideology-specific localisation or flag")
		}
	}
	for _, row := range rr.rows("characters") {
		tag, ideology := str(row["tag"]), str(row["ideology"])
		appendPoliticsRef(&snapshot.References, known, ideology,
			politicsrefs.EntityRef{Kind: "character", ID: strconv.FormatInt(project.Int64(row["id"]), 10), Path: "characters[" + tag + "].ideology"},
			"country leader ideology")

		if trimmed := strings.TrimSpace(ideology); trimmed != "" {
			ideologyUsage[trimmed]++
		}
	}
	for _, row := range rr.rows("history_files") {
		tag, rulingParty := str(row["country_tag"]), str(row["set_politics_ruling_party"])
		if strings.TrimSpace(rulingParty) != "" {
			countryRoots[strings.ToUpper(tag)] = strings.TrimSpace(rulingParty)
		}
		entity := politicsrefs.EntityRef{Kind: "history_file", ID: strconv.FormatInt(project.Int64(row["id"]), 10), Path: "history/countries/" + tag}
		appendPoliticsRef(&snapshot.References, known, rulingParty, entity, "set_politics.ruling_party")
		for _, k := range objectKeys(row["set_popularities"]) {
			appendPoliticsRef(&snapshot.References, known, k, entity, "set_popularities")
		}
		for _, token := range scriptIdeologyTokens(str(row["raw_script"]), known) {
			appendPoliticsRef(&snapshot.References, known, token.ID, entity, "raw_script."+token.Context)
		}
	}
	for _, row := range rr.rows("bookmarks") {
		entity := politicsrefs.EntityRef{Kind: "bookmark", ID: strconv.FormatInt(project.Int64(row["id"]), 10), Path: "common/bookmarks/" + str(row["bookmark_id"])}
		countries, _ := row["selectable_countries"].([]any)
		for index, item := range countries {
			country, _ := item.(map[string]any)
			appendPoliticsRef(&snapshot.References, known, str(country["ideology"]), entity,
				"selectable_countries["+strconv.Itoa(index)+"].ideology")
		}
		for _, token := range scriptIdeologyTokens(jsonText(row["raw_extras"]), known) {
			appendPoliticsRef(&snapshot.References, known, token.ID, entity, "raw_extras."+token.Context)
		}
	}
	proj := relProject{countryTag: str(record["country_tag"]), countryTags: jsonStrings(record["country_tags"]), isSubmod: hasBaseMod}
	for _, source := range relSources(rr, proj) {
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
	for _, row := range rr.rows("division_templates") {
		group := str(row["division_names_group"])
		if group == "" {
			continue
		}
		customRoot := countryRoots[strings.ToUpper(str(row["country_tag"]))]
		ideology, custom := known[customRoot]
		if !custom || ideology.Kind != politicsrefs.KindRoot || ideology.Source != politicsrefs.SourceProjectManaged {
			continue
		}
		entity := politicsrefs.EntityRef{Kind: "division_template", ID: strconv.FormatInt(project.Int64(row["id"]), 10),
			Path: "history/units/" + str(row["oob_file"]) + ".txt.division_names_group"}
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

func politicsReferences(p *project.Project) (politicsrefs.Snapshot, error) {
	rr := &projectReader{p: p}
	snapshot, _, err := loadPoliticsSnapshot(rr)
	if rr.err != nil {
		return politicsrefs.Snapshot{}, rr.err
	}
	return snapshot, err
}

func politicsError(err error) error {
	if err == nil {
		return nil
	}
	var status *apiutil.StatusError
	if errors.As(err, &status) {
		return err
	}
	var commandError politicsCommandError
	if errors.As(err, &commandError) {
		code := http.StatusUnprocessableEntity
		if commandError.Code == "politics_script_id_conflict" || commandError.Code == "politics_root_takeover_out_of_scope" ||
			commandError.Code == "politics_parent_stable" || strings.HasPrefix(commandError.Code, "politics_source_") {
			code = http.StatusConflict
		}
		return &apiutil.StatusError{Status: code, Body: map[string]any{"detail": commandError.Detail, "code": commandError.Code}}
	}
	var sourceFindings ideologysource.FindingsError
	if errors.As(err, &sourceFindings) {
		return &apiutil.StatusError{Status: http.StatusConflict, Body: map[string]any{
			"detail": "The inherited ideology source could not be locked safely.",
			"code":   "politics_source_validation_failed", "findings": sourceFindings.Findings,
		}}
	}
	code := http.StatusUnprocessableEntity
	errCode := "politics_contract_invalid"
	switch {
	case errors.Is(err, politicsrefs.ErrTargetNotFound):
		code, errCode = http.StatusNotFound, "politics_target_not_found"
	case errors.Is(err, politicsrefs.ErrDeleteBlocked):
		code, errCode = http.StatusConflict, "politics_delete_blocked"
	case errors.Is(err, politicsrefs.ErrUnknownPreview):
		code, errCode = http.StatusConflict, "politics_delete_preview_unknown"
	case errors.Is(err, politicsrefs.ErrStalePreview):
		code, errCode = http.StatusConflict, "politics_delete_preview_stale"
	case errors.Is(err, errPoliticsReferenceCoverage):
		code, errCode = http.StatusConflict, "politics_reference_coverage_incomplete"
	case errors.Is(err, politicsrefs.ErrImmutableSource):
		code, errCode = http.StatusConflict, "politics_source_owned_read_only"
	case errors.Is(err, politicsrefs.ErrStableScriptID):
		code = http.StatusConflict
	case errors.Is(err, politicsrefs.ErrInvalidSnapshot), errors.Is(err, politicsrefs.ErrDuplicateScriptID):
	default:
		logf("politics: %v", err)
		return detail(http.StatusServiceUnavailable, "politics reference service unavailable")
	}
	return &apiutil.StatusError{Status: code, Body: map[string]any{"detail": err.Error(), "code": errCode}}
}

func politicsReferencesHandler(p *project.Project, w http.ResponseWriter, r *http.Request) error {
	ensureImportedPoliticsSeeded(p)
	snapshot, err := politicsReferences(p)
	if err != nil {
		return politicsError(err)
	}
	apiutil.JSON(w, http.StatusOK, snapshot)
	return nil
}

func politicsCreateRootHandler(p *project.Project, w http.ResponseWriter, r *http.Request) error {
	var input politicsRootCreateInput
	if err := decodeStrict(r, &input, 1<<20); err != nil {
		return detail(http.StatusUnprocessableEntity, "Invalid politics ROOT payload")
	}
	ensureImportedPoliticsSeeded(p)
	result, err := runPoliticsRegistryCommand(p, func(snapshot politicsrefs.Snapshot, registry map[string]any, deps []string) (map[string]any, politicsDefinitionView, error) {
		return preparePoliticsRootCreate(snapshot, registry, deps, input)
	})
	if err != nil {
		return politicsError(err)
	}
	apiutil.JSON(w, http.StatusCreated, result)
	return nil
}

func politicsCreateSubtypeHandler(p *project.Project, w http.ResponseWriter, r *http.Request) error {
	var input politicsSubtypeCreateInput
	if err := decodeStrict(r, &input, 1<<20); err != nil {
		return detail(http.StatusUnprocessableEntity, "Invalid politics subtype payload")
	}
	ensureImportedPoliticsSeeded(p)
	result, err := runPoliticsRegistryCommand(p, func(snapshot politicsrefs.Snapshot, registry map[string]any, deps []string) (map[string]any, politicsDefinitionView, error) {
		return preparePoliticsSubtypeCreate(snapshot, registry, input, false)
	})
	var lock politicsCommandError
	if errors.As(err, &lock) && lock.Code == "politics_source_lock_required" {
		result, err = createSubtypeFromInheritedSource(p, input)
	}
	if err != nil {
		return politicsError(err)
	}
	apiutil.JSON(w, http.StatusCreated, result)
	return nil
}

func politicsUpdateHandler(p *project.Project, w http.ResponseWriter, r *http.Request) error {
	kind := politicsrefs.IdeologyKind(r.PathValue("kind"))
	if kind != politicsrefs.KindRoot && kind != politicsrefs.KindSubtype {
		return detail(http.StatusNotFound, "Not Found")
	}
	var patch politicsDefinitionPatch
	if err := decodeStrict(r, &patch, 1<<20); err != nil {
		return detail(http.StatusUnprocessableEntity, "Invalid politics definition patch")
	}
	ensureImportedPoliticsSeeded(p)
	target := politicsrefs.Target{Kind: kind, ScriptID: r.PathValue("script")}
	result, err := runPoliticsRegistryCommand(p, func(snapshot politicsrefs.Snapshot, registry map[string]any, deps []string) (map[string]any, politicsDefinitionView, error) {
		return preparePoliticsDefinitionPatch(snapshot, registry, target, patch)
	})
	if err != nil {
		return politicsError(err)
	}
	apiutil.JSON(w, http.StatusOK, result)
	return nil
}

func politicsDeletePreviewHandler(p *project.Project, w http.ResponseWriter, r *http.Request) error {
	var target politicsrefs.Target
	if err := decodeStrict(r, &target, 1<<20); err != nil {
		return detail(http.StatusUnprocessableEntity, "Invalid politics delete preview payload")
	}
	ensureImportedPoliticsSeeded(p)
	snapshot, err := politicsReferences(p)
	if err != nil {
		return politicsError(err)
	}
	preview, err := politicsDeleteManager.PreviewDelete(snapshot, target)
	if err != nil {
		return politicsError(err)
	}
	apiutil.JSON(w, http.StatusOK, preview)
	return nil
}

func politicsDeleteApplyHandler(p *project.Project, w http.ResponseWriter, r *http.Request) error {
	var input struct {
		PreviewToken string `json:"preview_token"`
	}
	if err := decodeStrict(r, &input, 1<<20); err != nil || strings.TrimSpace(input.PreviewToken) == "" {
		return detail(http.StatusUnprocessableEntity, "Invalid politics delete apply payload")
	}
	result, err := applyPoliticsDelete(p, strings.TrimSpace(input.PreviewToken))
	if errors.Is(err, politicsrefs.ErrDeleteBlocked) {
		return &apiutil.StatusError{Status: http.StatusConflict, Body: map[string]any{
			"code": "politics_delete_blocked", "detail": err.Error(), "authorization": result,
		}}
	}
	if err != nil {
		return politicsError(err)
	}
	apiutil.JSON(w, http.StatusOK, result)
	return nil
}

func politicsDiscardConflictHandler(p *project.Project, w http.ResponseWriter, r *http.Request) error {
	result, err := discardPoliticsConflict(p, r.PathValue("script"))
	if err != nil {
		return politicsError(err)
	}
	apiutil.JSON(w, http.StatusOK, result)
	return nil
}

func removePoliticsRegistryTarget(registry politicsRegistry, target politicsrefs.Target) (map[string]any, error) {
	key := "root_definitions"
	if target.Kind == politicsrefs.KindSubtype {
		key = "sub_ideologies"
	}
	values, _ := registry.Raw[key].([]any)
	filtered := make([]any, 0, len(values))
	removed := 0
	for _, value := range values {
		id := ""
		switch item := value.(type) {
		case string:
			id = strings.TrimSpace(item)
		case map[string]any:
			id = strings.TrimSpace(str(item["id"]))
		}
		if id == target.ScriptID {
			removed++
			continue
		}
		filtered = append(filtered, value)
	}
	if removed != 1 {
		return nil, fmt.Errorf("%w: expected one registry definition for %s, found %d", politicsrefs.ErrInvalidSnapshot, target.ScriptID, removed)
	}
	registry.Raw[key] = filtered
	if target.Kind == politicsrefs.KindRoot {
		if groups, ok := registry.Raw["groups"].([]any); ok {
			kept := make([]any, 0, len(groups))
			for _, value := range groups {
				if strings.TrimSpace(str(value)) != target.ScriptID {
					kept = append(kept, value)
				}
			}
			registry.Raw["groups"] = kept
		}
	}
	return registry.Raw, nil
}

func removePoliticsManifestTarget(value any, target politicsrefs.Target) (map[string]any, bool, error) {
	if target.Kind != politicsrefs.KindSubtype || strings.TrimSpace(target.ScriptID) == "" || value == nil {
		return nil, false, nil
	}
	doc := map[string]any{}
	raw, _ := json.Marshal(value)
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, false, fmt.Errorf("%w: malformed ideology source manifest", politicsrefs.ErrInvalidSnapshot)
	}
	values, ok := doc["entries"].([]any)
	if !ok {
		return nil, false, nil
	}
	filtered := make([]any, 0, len(values))
	removed := 0
	for _, value := range values {
		entry, _ := value.(map[string]any)
		if strings.TrimSpace(str(entry["script_id"])) == target.ScriptID {
			removed++
			continue
		}
		filtered = append(filtered, value)
	}
	if removed == 0 {
		return nil, false, nil
	}
	if removed != 1 {
		return nil, false, fmt.Errorf("%w: expected one source manifest entry for %s, found %d", politicsrefs.ErrInvalidSnapshot, target.ScriptID, removed)
	}
	doc["entries"] = filtered
	if len(filtered) == 0 {
		for _, key := range []string{"ready", "ready_at", "compiled_at", "source_key", "state", "snapshot_updated_at"} {
			delete(doc, key)
		}
	}
	return doc, true, nil
}

func applyPoliticsDelete(p *project.Project, token string) (politicsrefs.DeleteAuthorization, error) {
	var authorization politicsrefs.DeleteAuthorization
	err := p.Write(politicsTables, func(tx *project.Tx) error {
		snapshot, registry, err := loadPoliticsSnapshot(txReader{tx})
		if err != nil {
			return err
		}
		authorization, err = politicsDeleteManager.ApplyDelete(snapshot, token)
		if err != nil {
			return err
		}
		updated, err := removePoliticsRegistryTarget(registry, authorization.Target)
		if err != nil {
			return err
		}
		manifest, manifestChanged, err := removePoliticsManifestTarget(tx.Record()["ideology_source_manifest"], authorization.Target)
		if err != nil {
			return err
		}
		tx.SetRecord(func(record project.Row) {
			record["ideology_registry"] = updated
			if manifestChanged {
				record["ideology_source_manifest"] = manifest
			}
			record["updated_at"] = workbench.NowTimestamp()
		})
		return nil
	})
	return authorization, err
}

func discardPoliticsConflict(p *project.Project, scriptID string) (politicsrefs.IdeologyConflict, error) {
	scriptID = strings.TrimSpace(scriptID)
	if scriptID == "" {
		return politicsrefs.IdeologyConflict{}, politicsrefs.ErrTargetNotFound
	}
	var conflict politicsrefs.IdeologyConflict
	err := p.Write(politicsTables, func(tx *project.Tx) error {
		snapshot, registry, err := loadPoliticsSnapshot(txReader{tx})
		if err != nil {
			return err
		}
		found := false
		for _, candidate := range snapshot.Conflicts {
			if candidate.ScriptID == scriptID {
				conflict, found = candidate, true
				break
			}
		}
		if !found {
			return politicsrefs.ErrTargetNotFound
		}
		target := politicsrefs.Target{Kind: conflict.Kind, ScriptID: scriptID}
		if target.Kind == "" {
			target.Kind = politicsrefs.KindSubtype
		}
		updated, err := removePoliticsRegistryTarget(registry, target)
		if err != nil {
			return err
		}
		tx.SetRecord(func(record project.Row) {
			record["ideology_registry"] = updated
			record["updated_at"] = workbench.NowTimestamp()
		})
		return nil
	})
	return conflict, err
}

type politicsRootCreateInput struct {
	ScriptID string            `json:"script_id"`
	Body     string            `json:"body"`
	Name     map[string]string `json:"name"`
	Desc     map[string]string `json:"desc,omitempty"`
}

type politicsSubtypeCreateInput struct {
	ScriptID       string            `json:"script_id"`
	RootID         string            `json:"root_id"`
	Name           map[string]string `json:"name"`
	Desc           map[string]string `json:"desc,omitempty"`
	RandomDisabled bool              `json:"random_disabled"`
}

type politicsDefinitionPatch struct {
	Name           workbench.Optional[map[string]string] `json:"name"`
	Desc           workbench.Optional[map[string]string] `json:"desc"`
	Body           workbench.Optional[string]            `json:"body"`
	RootID         workbench.Optional[string]            `json:"root_id"`
	RandomDisabled workbench.Optional[bool]              `json:"random_disabled"`
}

type politicsDefinitionView struct {
	Kind           politicsrefs.IdeologyKind    `json:"kind"`
	ScriptID       string                       `json:"script_id"`
	RootID         string                       `json:"root_id,omitempty"`
	Body           string                       `json:"body,omitempty"`
	Name           map[string]string            `json:"name"`
	Desc           map[string]string            `json:"desc"`
	RandomDisabled bool                         `json:"random_disabled,omitempty"`
	Source         politicsrefs.SourceOwnership `json:"source"`
}

func clonePoliticsRegistry(raw map[string]any) (map[string]any, error) {
	encoded, err := json.Marshal(raw)
	if err != nil {
		return nil, err
	}
	var cloned map[string]any
	if err := json.Unmarshal(encoded, &cloned); err != nil {
		return nil, err
	}
	if cloned == nil {
		cloned = map[string]any{}
	}
	for _, key := range []string{"groups", "root_definitions", "sub_ideologies"} {
		if _, ok := cloned[key].([]any); !ok {
			cloned[key] = []any{}
		}
	}
	return cloned, nil
}

func normalizePoliticsLocs(raw map[string]string, required bool) (map[string]any, error) {
	out := map[string]any{}
	for locale, value := range raw {
		code := strings.ToLower(strings.TrimSpace(locale))
		text := strings.TrimSpace(value)
		if code == "" || text == "" {
			continue
		}
		if !workbench.ContentLanguages[code] {
			return nil, politicsCommandError{Code: "politics_locale_invalid", Detail: fmt.Sprintf("Unsupported politics content locale %q.", locale)}
		}
		out[code] = text
	}
	if required && len(out) == 0 {
		return nil, politicsCommandError{Code: "politics_display_name_required", Detail: "Give the political definition a visible name before choosing its script ID."}
	}
	return out, nil
}

func politicsDefinitionByTarget(snapshot politicsrefs.Snapshot, target politicsrefs.Target) (politicsrefs.Ideology, bool) {
	for _, ideology := range snapshot.Ideologies {
		if ideology.Kind == target.Kind && ideology.ScriptID == target.ScriptID {
			return ideology, true
		}
	}
	return politicsrefs.Ideology{}, false
}

func politicsIDExists(snapshot politicsrefs.Snapshot, scriptID string) bool {
	for _, ideology := range snapshot.Ideologies {
		if ideology.ScriptID == scriptID {
			return true
		}
	}
	return false
}

func politicsEntryDefinition(kind politicsrefs.IdeologyKind, entry map[string]any) politicsDefinitionView {
	definition := politicsDefinitionView{
		Kind: kind, ScriptID: strings.TrimSpace(str(entry["id"])), RootID: strings.TrimSpace(str(entry["group"])),
		Body: str(entry["body"]), Name: map[string]string{}, Desc: map[string]string{},
		Source: politicsrefs.SourceProjectManaged,
	}
	for key, value := range mapFrom(entry["name"]) {
		if text := strings.TrimSpace(str(value)); text != "" {
			definition.Name[key] = text
		}
	}
	for key, value := range mapFrom(entry["desc"]) {
		if text := strings.TrimSpace(str(value)); text != "" {
			definition.Desc[key] = text
		}
	}
	definition.RandomDisabled, _ = entry["random_disabled"].(bool)
	return definition
}

func mapFrom(value any) map[string]any {
	if result, ok := value.(map[string]any); ok {
		return result
	}
	return map[string]any{}
}

func preparePoliticsRootCreate(snapshot politicsrefs.Snapshot, registry map[string]any, deps []string, input politicsRootCreateInput) (map[string]any, politicsDefinitionView, error) {
	input.ScriptID = strings.TrimSpace(input.ScriptID)
	input.Body = strings.TrimSpace(input.Body)
	if len(deps) > 0 {
		return nil, politicsDefinitionView{}, politicsCommandError{Code: "politics_root_takeover_out_of_scope", Detail: "A submod inherits source ROOT groups. P0 does not create or take over a ROOT in a base Mod."}
	}
	if reason := workbench.ValidateRoot(input.ScriptID, input.Body); reason != "" {
		return nil, politicsDefinitionView{}, politicsCommandError{Code: "politics_root_invalid", Detail: reason}
	}
	if politicsIDExists(snapshot, input.ScriptID) {
		return nil, politicsDefinitionView{}, politicsCommandError{Code: "politics_script_id_conflict", Detail: fmt.Sprintf("Political script ID %q already exists.", input.ScriptID)}
	}
	name, err := normalizePoliticsLocs(input.Name, true)
	if err != nil {
		return nil, politicsDefinitionView{}, err
	}
	desc, err := normalizePoliticsLocs(input.Desc, false)
	if err != nil {
		return nil, politicsDefinitionView{}, err
	}
	updated, err := clonePoliticsRegistry(registry)
	if err != nil {
		return nil, politicsDefinitionView{}, err
	}
	entry := map[string]any{"id": input.ScriptID, "body": input.Body, "name": name, "desc": desc, "source": string(politicsrefs.SourceProjectManaged)}
	updated["root_definitions"] = append(updated["root_definitions"].([]any), entry)
	groups := append(updated["groups"].([]any), input.ScriptID)
	sort.SliceStable(groups, func(i, j int) bool { return str(groups[i]) < str(groups[j]) })
	updated["groups"] = groups
	return updated, politicsEntryDefinition(politicsrefs.KindRoot, entry), nil
}

func preparePoliticsSubtypeCreate(snapshot politicsrefs.Snapshot, registry map[string]any, input politicsSubtypeCreateInput, inheritedSourceLocked bool) (map[string]any, politicsDefinitionView, error) {
	input.ScriptID = strings.TrimSpace(input.ScriptID)
	input.RootID = strings.TrimSpace(input.RootID)
	if !workbench.RootIDRE.MatchString(input.ScriptID) {
		return nil, politicsDefinitionView{}, politicsCommandError{Code: "politics_script_id_invalid", Detail: "Script ID must start with a letter or underscore and use only letters, numbers, and underscores."}
	}
	if politicsIDExists(snapshot, input.ScriptID) {
		return nil, politicsDefinitionView{}, politicsCommandError{Code: "politics_script_id_conflict", Detail: fmt.Sprintf("Political script ID %q already exists.", input.ScriptID)}
	}
	parent, found := politicsDefinitionByTarget(snapshot, politicsrefs.Target{Kind: politicsrefs.KindRoot, ScriptID: input.RootID})
	if !found {
		return nil, politicsDefinitionView{}, politicsrefs.ErrTargetNotFound
	}
	if parent.Source != politicsrefs.SourceProjectManaged && !inheritedSourceLocked {
		return nil, politicsDefinitionView{}, politicsCommandError{Code: "politics_source_lock_required", Detail: "Import and confirm the exact ideology source before adding a subtype to an inherited ROOT."}
	}
	name, err := normalizePoliticsLocs(input.Name, true)
	if err != nil {
		return nil, politicsDefinitionView{}, err
	}
	desc, err := normalizePoliticsLocs(input.Desc, false)
	if err != nil {
		return nil, politicsDefinitionView{}, err
	}
	updated, err := clonePoliticsRegistry(registry)
	if err != nil {
		return nil, politicsDefinitionView{}, err
	}
	entry := map[string]any{"id": input.ScriptID, "group": input.RootID, "name": name, "desc": desc, "random_disabled": input.RandomDisabled, "source": string(politicsrefs.SourceProjectManaged)}
	updated["sub_ideologies"] = append(updated["sub_ideologies"].([]any), entry)
	return updated, politicsEntryDefinition(politicsrefs.KindSubtype, entry), nil
}

func preparePoliticsDefinitionPatch(snapshot politicsrefs.Snapshot, registry map[string]any, target politicsrefs.Target, patch politicsDefinitionPatch) (map[string]any, politicsDefinitionView, error) {
	target.ScriptID = strings.TrimSpace(target.ScriptID)
	current, found := politicsDefinitionByTarget(snapshot, target)
	if !found {
		return nil, politicsDefinitionView{}, politicsrefs.ErrTargetNotFound
	}
	if !politicsrefs.OwnershipFor(current, politicsrefs.ActionEdit).Allowed {
		return nil, politicsDefinitionView{}, politicsrefs.ErrImmutableSource
	}

	if patch.RootID.Set {
		return nil, politicsDefinitionView{}, politicsCommandError{
			Code:   "politics_parent_stable",
			Detail: "A political subtype's parent ROOT is stable after creation. Create the subtype under the intended ROOT, then preview and delete the old definition.",
		}
	}
	updated, err := clonePoliticsRegistry(registry)
	if err != nil {
		return nil, politicsDefinitionView{}, err
	}
	key := "root_definitions"
	if target.Kind == politicsrefs.KindSubtype {
		key = "sub_ideologies"
	}
	values := updated[key].([]any)
	matched := 0
	var definition politicsDefinitionView
	for index, value := range values {
		entry, ok := value.(map[string]any)
		if !ok || strings.TrimSpace(str(entry["id"])) != target.ScriptID {
			continue
		}
		matched++
		if patch.Name.Set {
			if patch.Name.Value == nil {
				return nil, politicsDefinitionView{}, politicsCommandError{Code: "politics_display_name_required", Detail: "A visible political name cannot be cleared."}
			}
			locs, err := normalizePoliticsLocs(*patch.Name.Value, true)
			if err != nil {
				return nil, politicsDefinitionView{}, err
			}
			entry["name"] = locs
		}
		if patch.Desc.Set {
			locs := map[string]any{}
			if patch.Desc.Value != nil {
				if locs, err = normalizePoliticsLocs(*patch.Desc.Value, false); err != nil {
					return nil, politicsDefinitionView{}, err
				}
			}
			entry["desc"] = locs
		}
		if patch.Body.Set {
			if target.Kind != politicsrefs.KindRoot || patch.Body.Value == nil {
				return nil, politicsDefinitionView{}, politicsCommandError{Code: "politics_root_body_invalid", Detail: "Advanced political-group script can only be edited for a project-managed political group."}
			}
			body := strings.TrimSpace(*patch.Body.Value)
			if reason := workbench.ValidateRoot(target.ScriptID, body); reason != "" {
				return nil, politicsDefinitionView{}, politicsCommandError{Code: "politics_root_invalid", Detail: reason}
			}
			entry["body"] = body
		}
		if patch.RandomDisabled.Set {
			if target.Kind != politicsrefs.KindSubtype || patch.RandomDisabled.Value == nil {
				return nil, politicsDefinitionView{}, politicsCommandError{Code: "politics_random_selection_invalid", Detail: "Random-selection control requires a subtype boolean."}
			}
			entry["random_disabled"] = *patch.RandomDisabled.Value
		}
		values[index] = entry
		definition = politicsEntryDefinition(target.Kind, entry)
	}
	if matched != 1 {
		return nil, politicsDefinitionView{}, fmt.Errorf("%w: expected one project definition for %s", politicsrefs.ErrInvalidSnapshot, target.ScriptID)
	}
	updated[key] = values
	return updated, definition, nil
}

func persistPoliticsRegistry(tx *project.Tx, registry map[string]any, deps []string) error {
	registry = workbench.NormalizeRegistry(registry)
	if err := workbench.ValidateRegistry(registry, deps); err != nil {
		return politicsCommandError{Code: "politics_registry_invalid", Detail: err.Error()}
	}
	tx.SetRecord(func(record project.Row) {
		record["ideology_registry"] = registry
		record["updated_at"] = workbench.NowTimestamp()
	})
	return nil
}

type politicsCommand func(snapshot politicsrefs.Snapshot, registry map[string]any, deps []string) (map[string]any, politicsDefinitionView, error)

func runPoliticsRegistryCommand(p *project.Project, command politicsCommand) (politicsDefinitionView, error) {
	var definition politicsDefinitionView
	err := p.Write(politicsTables, func(tx *project.Tx) error {
		snapshot, registry, err := loadPoliticsSnapshot(txReader{tx})
		if err != nil {
			return err
		}
		deps := jsonStrings(tx.Record()["dependencies"])
		updated, def, err := command(snapshot, registry.Raw, deps)
		if err != nil {
			return err
		}
		definition = def
		return persistPoliticsRegistry(tx, updated, deps)
	})
	return definition, err
}

const ideologySourceArchiveLimit int64 = 520 << 20

type ideologySnapshotRef struct {
	SourceKey  string
	OBSKey     string
	Provenance struct {
		SchemaVersion          int      `json:"schema_version"`
		SourceKey              string   `json:"source_key"`
		DependencyKey          string   `json:"dependency_key,omitempty"`
		DescriptorReplacePaths []string `json:"descriptor_replace_paths"`
		Files                  []struct {
			Path   string `json:"path"`
			SHA256 string `json:"sha256"`
		} `json:"files"`
	}
}

func ideologySourceSnapshot(rows []project.Row, sourceKey string) (ideologySnapshotRef, bool, error) {
	var out ideologySnapshotRef
	for _, row := range rows {
		if str(row["asset_type"]) != "ideology_source" || str(row["gfx_name"]) != sourceKey {
			continue
		}
		out.SourceKey, out.OBSKey = str(row["gfx_name"]), str(row["obs_key"])
		raw, err := json.Marshal(row["provenance"])
		if err != nil {
			return out, false, err
		}
		if err := json.Unmarshal(raw, &out.Provenance); err != nil {
			return out, false, err
		}
		if out.Provenance.SourceKey != out.SourceKey || out.Provenance.SchemaVersion != 1 {
			return ideologySnapshotRef{}, false, errors.New("ideology source snapshot provenance mismatch")
		}
		return out, true, nil
	}
	return out, false, nil
}

func bundledVanillaIdeologyLayers() ([]ideologysource.Layer, error) {
	tables, err := worldout.LoadVanillaTables(context.Background())
	if err != nil || strings.TrimSpace(tables.IdeologyBaseText) == "" {
		return nil, politicsCommandError{Code: "politics_source_snapshot_unavailable", Detail: "The base game's ideology file is unavailable; set the Hearts of Iron IV folder in settings."}
	}
	return []ideologysource.Layer{{
		SourceKind: ideologysource.SourceVanilla, LoadOrder: 0,
		Files: []ideologysource.SourceFile{{Path: "common/ideologies/00_ideologies.txt", Bytes: []byte(tables.IdeologyBaseText)}},
	}}, nil
}

func safeIdeologyArchivePath(raw string) (string, bool) {
	clean := filepath.ToSlash(filepath.Clean(strings.ReplaceAll(raw, `\`, "/")))
	if clean == "." || clean == ".." || strings.HasPrefix(clean, "../") || strings.HasPrefix(clean, "/") {
		return "", false
	}
	lower := strings.ToLower(clean)
	return clean, strings.HasPrefix(lower, ideologysource.IdeologyDirectory+"/") && strings.HasSuffix(lower, ".txt")
}

func ideologyLayersFromSnapshot(ref ideologySnapshotRef, archive []byte) ([]ideologysource.Layer, error) {
	reader, err := zip.NewReader(bytes.NewReader(archive), int64(len(archive)))
	if err != nil {
		return nil, politicsCommandError{Code: "politics_source_archive_invalid", Detail: "The private ideology source snapshot is not a readable ZIP archive."}
	}
	expected := map[string]string{}
	for _, file := range ref.Provenance.Files {
		path, ok := safeIdeologyArchivePath(file.Path)
		if !ok || expected[strings.ToLower(path)] != "" {
			return nil, politicsCommandError{Code: "politics_source_archive_invalid", Detail: "The ideology source provenance contains an unsafe or duplicate path."}
		}
		expected[strings.ToLower(path)] = strings.ToLower(strings.TrimSpace(file.SHA256))
	}
	files := make([]ideologysource.SourceFile, 0, len(expected))
	seen := map[string]bool{}
	var total int64
	for _, member := range reader.File {
		if member.FileInfo().IsDir() {
			continue
		}
		if member.Mode()&os.ModeSymlink != 0 || !member.Mode().IsRegular() {
			return nil, politicsCommandError{Code: "politics_source_archive_invalid", Detail: "The ideology source snapshot contains a link or non-regular member."}
		}
		path, ok := safeIdeologyArchivePath(member.Name)
		k := strings.ToLower(path)
		want, known := expected[k]
		if !ok || !known || seen[k] {
			return nil, politicsCommandError{Code: "politics_source_archive_invalid", Detail: fmt.Sprintf("Unexpected ideology source member %q.", member.Name)}
		}
		if member.UncompressedSize64 > uint64(ideologySourceArchiveLimit) || total > ideologySourceArchiveLimit-int64(member.UncompressedSize64) {
			return nil, politicsCommandError{Code: "politics_source_archive_invalid", Detail: "The ideology source snapshot exceeds its read limit."}
		}
		input, err := member.Open()
		if err != nil {
			return nil, err
		}
		data, readErr := io.ReadAll(io.LimitReader(input, int64(member.UncompressedSize64)+1))
		closeErr := input.Close()
		if readErr != nil {
			return nil, readErr
		}
		if closeErr != nil {
			return nil, closeErr
		}
		if uint64(len(data)) != member.UncompressedSize64 || fmt.Sprintf("%x", sha256.Sum256(data)) != want {
			return nil, politicsCommandError{Code: "politics_source_archive_drift", Detail: fmt.Sprintf("Ideology source member %q no longer matches its recorded source.", path)}
		}
		total += int64(len(data))
		seen[k] = true
		files = append(files, ideologysource.SourceFile{Path: path, Bytes: data})
	}
	if len(seen) != len(expected) {
		return nil, politicsCommandError{Code: "politics_source_archive_drift", Detail: "The ideology source snapshot is missing one or more recorded files."}
	}
	if strings.EqualFold(ref.SourceKey, ideologysource.SourceVanilla) {
		return []ideologysource.Layer{{SourceKind: ideologysource.SourceVanilla, LoadOrder: 0, Files: files}}, nil
	}
	dependencyKey := strings.TrimSpace(ref.Provenance.DependencyKey)
	if dependencyKey == "" {
		dependencyKey = ref.SourceKey
	}
	return []ideologysource.Layer{
		{SourceKind: ideologysource.SourceVanilla, LoadOrder: 0, Files: []ideologysource.SourceFile{}},
		{SourceKind: ideologysource.SourceDependency, DependencyKey: dependencyKey, LoadOrder: 1, ReplacePaths: ref.Provenance.DescriptorReplacePaths, Files: files},
	}, nil
}

func importedIdeologyLayers(p *project.Project) (ideologySnapshotRef, []ideologysource.Layer, bool, error) {
	rows, err := p.Rows("project_obs_assets")
	if err != nil {
		return ideologySnapshotRef{}, nil, false, err
	}
	ref, found, err := ideologySourceSnapshot(rows, "imported")
	if err != nil || !found {
		return ref, nil, found, err
	}
	archive, err := p.Blobs().Get(ref.OBSKey, ideologySourceArchiveLimit)
	if err != nil {
		return ref, nil, true, politicsCommandError{Code: "politics_source_snapshot_unavailable", Detail: "The private inherited ideology snapshot could not be read. Re-import the Mod and retry."}
	}
	layers, err := ideologyLayersFromSnapshot(ref, archive)
	return ref, layers, true, err
}

func createSubtypeFromInheritedSource(p *project.Project, input politicsSubtypeCreateInput) (politicsDefinitionView, error) {
	snapshot, err := politicsReferences(p)
	if err != nil {
		return politicsDefinitionView{}, err
	}
	parent, found := politicsDefinitionByTarget(snapshot, politicsrefs.Target{Kind: politicsrefs.KindRoot, ScriptID: strings.TrimSpace(input.RootID)})
	if !found {
		return politicsDefinitionView{}, politicsCommandError{Code: "politics_parent_missing", Detail: "The selected ideology ROOT no longer exists."}
	}
	if parent.Source == politicsrefs.SourceVanilla {
		layers, err := bundledVanillaIdeologyLayers()
		if err != nil {
			return politicsDefinitionView{}, err
		}
		return createSubtypeFromSource(p, input, ideologysource.SourceVanilla, "", layers)
	}
	ref, layers, found, err := importedIdeologyLayers(p)
	if err != nil {
		return politicsDefinitionView{}, err
	}
	if !found {
		return politicsDefinitionView{}, politicsCommandError{Code: "politics_source_lock_required", Detail: "Re-import the Mod to capture the exact inherited ideology source before adding a subtype."}
	}
	return createSubtypeFromSource(p, input, ref.SourceKey, ref.OBSKey, layers)
}

func inheritedSubtypePayloads(registry map[string]any, wanted map[string]bool) ([]ideologysource.Subtype, error) {
	rows, _ := registry["sub_ideologies"].([]any)
	result := make([]ideologysource.Subtype, 0, len(wanted))
	seen := map[string]bool{}
	for _, value := range rows {
		entry, ok := value.(map[string]any)
		if !ok {
			continue
		}
		id := strings.TrimSpace(str(entry["id"]))
		if !wanted[id] {
			continue
		}
		if seen[id] {
			return nil, politicsrefs.ErrDuplicateScriptID
		}
		seen[id] = true
		names := map[string]string{}
		for locale, raw := range mapFrom(entry["name"]) {
			names[locale] = str(raw)
		}
		desc := map[string]string{}
		for locale, raw := range mapFrom(entry["desc"]) {
			desc[locale] = str(raw)
		}
		result = append(result, ideologysource.Subtype{
			ScriptID: id, ParentRootID: strings.TrimSpace(str(entry["group"])), Names: names, Descriptions: desc,
		})
	}
	for id := range wanted {
		if !seen[id] {
			return nil, politicsCommandError{Code: "politics_source_manifest_registry_drift", Detail: fmt.Sprintf("Locked subtype %q is missing from the project registry.", id)}
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ScriptID < result[j].ScriptID })
	return result, nil
}

func rejectInheritedSubtypeOverOwnIdeologies(tx *project.Tx) error {
	owner := ""
	for _, row := range tx.Rows("project_raw_files") {
		path := str(row["dest_path"])
		if strings.HasPrefix(strings.ToLower(strings.ReplaceAll(path, `\`, "/")), "common/ideologies/") && (owner == "" || path < owner) {
			owner = path
		}
	}
	if owner != "" {
		return politicsCommandError{Code: "politics_source_raw_owned", Detail: fmt.Sprintf(
			"This project has its own ideology file %s, which the game loads instead of the inherited one. Add the subtype to that file instead.", owner)}
	}
	for _, path := range jsonStrings(tx.Record()["imported_descriptor_replace_paths"]) {
		if strings.Trim(strings.ToLower(strings.ReplaceAll(strings.TrimSpace(path), `\`, "/")), "/") == ideologysource.IdeologyDirectory {
			return politicsCommandError{Code: "politics_source_raw_owned", Detail: "This project replaces common/ideologies (replace_path), so the inherited ideologies are not loaded. Add the subtype to the project's own ideology file instead."}
		}
	}
	return nil
}

func createSubtypeFromSource(p *project.Project, input politicsSubtypeCreateInput, sourceKey, expectedOBSKey string, layers []ideologysource.Layer) (politicsDefinitionView, error) {
	sourceKey = strings.TrimSpace(sourceKey)
	var definition politicsDefinitionView
	err := p.Write(politicsTables, func(tx *project.Tx) error {
		snapshot, registry, err := loadPoliticsSnapshot(txReader{tx})
		if err != nil {
			return err
		}
		deps := jsonStrings(tx.Record()["dependencies"])
		if err := rejectInheritedSubtypeOverOwnIdeologies(tx); err != nil {
			return err
		}

		if sourceKey != ideologysource.SourceVanilla || strings.TrimSpace(expectedOBSKey) != "" {
			ref, found, err := ideologySourceSnapshot(tx.Rows("project_obs_assets"), sourceKey)
			if err != nil {
				return err
			}
			if !found {
				return politicsCommandError{Code: "politics_source_lock_required", Detail: "The inherited ideology source snapshot is missing. Re-import the Mod before adding this subtype."}
			}
			if strings.TrimSpace(expectedOBSKey) == "" || ref.OBSKey != expectedOBSKey {
				return politicsCommandError{Code: "politics_source_snapshot_stale", Detail: "The inherited ideology source changed while it was being checked. Retry after the import finishes."}
			}
		}
		var current ideologysource.Manifest
		if raw, err := json.Marshal(tx.Record()["ideology_source_manifest"]); err == nil {
			_ = json.Unmarshal(raw, &current)
		}
		updated, def, err := preparePoliticsSubtypeCreate(snapshot, registry.Raw, input, true)
		if err != nil {
			return err
		}
		wanted := map[string]bool{def.ScriptID: true}
		for _, entry := range current.Entries {
			wanted[strings.TrimSpace(entry.ScriptID)] = true
		}
		subtypes, err := inheritedSubtypePayloads(updated, wanted)
		if err != nil {
			return err
		}
		discovery, err := ideologysource.Discover(layers)
		if err != nil {
			return err
		}
		manifest, err := ideologysource.Lock(discovery, subtypes)
		if err != nil {
			return err
		}
		if _, err := ideologysource.Compile(layers, manifest, subtypes); err != nil {
			return err
		}
		if err := persistPoliticsRegistry(tx, updated, deps); err != nil {
			return err
		}

		var stored map[string]any
		raw, err := json.Marshal(manifest)
		if err != nil {
			return err
		}
		if err := json.Unmarshal(raw, &stored); err != nil {
			return err
		}
		stored["state"] = "ready"
		stored["source_key"] = sourceKey
		stored["ready_at"] = time.Now().UTC().Format(time.RFC3339Nano)
		tx.SetRecord(func(record project.Row) { record["ideology_source_manifest"] = stored })
		definition = def
		return nil
	})
	return definition, err
}

func mergeImportedPoliticsSource(registry map[string]any, discovery ideologysource.Discovery) (map[string]any, error) {
	updated, err := clonePoliticsRegistry(registry)
	if err != nil {
		return nil, err
	}
	importedRoots := map[string]bool{}
	roots := make([]any, 0)
	for _, raw := range updated["root_definitions"].([]any) {
		entry, _ := raw.(map[string]any)
		if politicsSource(entry["source"], politicsrefs.SourceProjectManaged) == politicsrefs.SourceImported {
			importedRoots[strings.TrimSpace(str(entry["id"]))] = true
			continue
		}
		roots = append(roots, raw)
	}
	subs := make([]any, 0)
	for _, raw := range updated["sub_ideologies"].([]any) {
		entry, _ := raw.(map[string]any)
		if politicsSource(entry["source"], politicsrefs.SourceProjectManaged) == politicsrefs.SourceImported {
			continue
		}
		subs = append(subs, raw)
	}
	groups := make([]any, 0)
	seenIDs := map[string]bool{}
	for _, raw := range updated["groups"].([]any) {
		id := strings.TrimSpace(str(raw))
		if id == "" || importedRoots[id] {
			continue
		}
		groups = append(groups, id)
		seenIDs[id] = true
	}
	for _, list := range [][]any{roots, subs} {
		for _, raw := range list {
			entry, _ := raw.(map[string]any)
			if id := strings.TrimSpace(str(entry["id"])); id != "" {
				seenIDs[id] = true
			}
		}
	}
	conflict := func(id string) error {
		return politicsCommandError{Code: "politics_script_id_conflict", Detail: fmt.Sprintf("Imported political script ID %q conflicts with an existing project definition.", id)}
	}
	for _, owner := range discovery.Roots {
		rootID := strings.TrimSpace(owner.RootID)

		if reftables.Ideologies().IsGroup(rootID) {
			continue
		}
		if rootID == "" || seenIDs[rootID] {
			return nil, conflict(rootID)
		}
		seenIDs[rootID] = true
		groups = append(groups, rootID)
		roots = append(roots, map[string]any{
			"id": rootID, "body": strings.TrimSpace(owner.Body), "name": map[string]any{}, "desc": map[string]any{},
			"source": string(politicsrefs.SourceImported),
		})
		for _, rawSubtypeID := range owner.SubtypeIDs {
			subtypeID := strings.TrimSpace(rawSubtypeID)
			if subtypeID == "" || seenIDs[subtypeID] {
				return nil, conflict(subtypeID)
			}
			seenIDs[subtypeID] = true
			subs = append(subs, map[string]any{
				"id": subtypeID, "group": rootID, "name": map[string]any{}, "desc": map[string]any{},
				"source": string(politicsrefs.SourceImported),
			})
		}
	}
	byID := func(list []any) func(i, j int) bool {
		return func(i, j int) bool {
			left, _ := list[i].(map[string]any)
			right, _ := list[j].(map[string]any)
			return str(left["id"]) < str(right["id"])
		}
	}
	sort.SliceStable(groups, func(i, j int) bool { return str(groups[i]) < str(groups[j]) })
	sort.SliceStable(roots, byID(roots))
	sort.SliceStable(subs, byID(subs))
	updated["groups"], updated["root_definitions"], updated["sub_ideologies"] = groups, roots, subs
	return updated, nil
}

func SeedImportedPolitics(p *project.Project) error {
	_, layers, found, err := importedIdeologyLayers(p)
	if err != nil || !found {
		return err
	}
	discovery, err := ideologysource.Discover(layers)
	if err != nil {
		return err
	}
	return p.Write(politicsTables, func(tx *project.Tx) error {
		_, registry, err := loadPoliticsSnapshot(txReader{tx})
		if err != nil {
			return err
		}
		updated, err := mergeImportedPoliticsSource(registry.Raw, discovery)
		if err != nil {
			return err
		}
		deps := jsonStrings(tx.Record()["dependencies"])
		normalized := workbench.NormalizeRegistry(updated)
		current, _ := clonePoliticsRegistry(registry.Raw)
		if reflect.DeepEqual(jsonRoundTrip(normalized), jsonRoundTrip(workbench.NormalizeRegistry(current))) &&
			tx.Record()["ideology_registry"] != nil {
			return nil
		}
		return persistPoliticsRegistry(tx, updated, deps)
	})
}

func jsonRoundTrip(value any) any {
	raw, err := json.Marshal(value)
	if err != nil {
		return nil
	}
	var out any
	_ = json.Unmarshal(raw, &out)
	return out
}

var seededSnapshots syncMap

func ensureImportedPoliticsSeeded(p *project.Project) {
	rows, err := p.Rows("project_obs_assets")
	if err != nil {
		return
	}
	key := ""
	for _, row := range rows {
		if str(row["asset_type"]) == "ideology_source" && str(row["gfx_name"]) == "imported" {
			key = str(row["obs_key"])
		}
	}
	if key == "" || seededSnapshots.get(p.Dir()) == key {
		return
	}
	if err := SeedImportedPolitics(p); err != nil {
		logf("seed imported politics %s: %v", p.Dir(), err)
	}
	seededSnapshots.set(p.Dir(), key)
}
