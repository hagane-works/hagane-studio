package exportengine

import (
	"context"
	"encoding/json"
	"fmt"
	"path"
	"sort"
	"strings"
	"unicode/utf8"

	"haganeworks/internal/cwtools"
	"haganeworks/internal/gameindex"
	"haganeworks/internal/hoi4/exportmsg"
	"haganeworks/internal/hoi4/paradox"
)

const (
	RegistryValidationCoverage = "registry_validation"
	RegistryValidationLogName  = "_CONTENT_SOURCE_REVIEW.log"
)

type registryCatalog struct {
	Modifiers        []string
	ModifierPatterns []string

	IdeaCategories []string
	IdeaSlots      []string
	ideaTagsKnown  bool

	Source struct {
		Repository string
		Commit     string
		License    string
		Dir        string
	}
}

type RegistryFinding struct {
	Code   string
	Status string
	Path   string
	Line   int
	Token  string
	Text   string
}

type RegistryReport struct {
	Findings []RegistryFinding
	Catalog  registryCatalog
	msg      exportmsg.Printer
}

type registryParsedArtifact struct {
	entry Entry
	rel   string
	ast   *paradox.Block
}

func (r RegistryReport) Count() int { return len(r.Findings) }

func (r RegistryReport) Headline() string {
	if len(r.Findings) == 0 {
		return ""
	}
	return r.msg.Text("registry.headline", exportmsg.Args{"count": len(r.Findings), "log": RegistryValidationLogName})
}

func RegistryValidationLogEntry(modDir string, report RegistryReport) Entry {
	if modDir == "" || len(report.Findings) == 0 {
		return Entry{}
	}
	source, msg := report.Catalog.Source, report.msg
	rules := msg.Text("registry.log.rules", exportmsg.Args{"repository": source.Repository, "commit": source.Commit, "license": source.License})
	if source.Dir == "" {
		rules = msg.Text("registry.log.no_rules", exportmsg.Args{
			"file": cwtools.ModifiersFile, "repository": cwtools.Repository, "license": cwtools.License, "folder": cwtools.Folder,
		})
	}
	lines := commentLines(msg.Text("registry.log.title", nil))
	lines = append(lines, commentLines(rules)...)
	lines = append(lines, commentLines(msg.Text("registry.log.notes", nil))...)
	lines = append(lines, "")
	for _, finding := range report.Findings {
		where := finding.Path
		if finding.Line > 0 {
			where += fmt.Sprintf(":%d", finding.Line)
		}
		lines = append(lines, fmt.Sprintf("%s · %s · %s · %s", finding.Status, finding.Code, where, finding.Text))
	}
	return Entry{Path: modDir + "/" + RegistryValidationLogName, Bytes: []byte(strings.Join(lines, "\n") + "\n"), Origin: "generated", SourceEntity: "registry_validation"}
}

func ScanRegistryValidation(entries []Entry, msg exportmsg.Printer) RegistryReport {
	catalog := loadRegistryCatalog()
	report := RegistryReport{Catalog: catalog, msg: msg}

	modifierCatalogReady := len(catalog.Modifiers) > 0
	knownModifiers := stringSet(catalog.Modifiers)
	knownCategories := stringSet(catalog.IdeaCategories)
	knownSlots := stringSet(catalog.IdeaSlots)
	root := artifactRootFromEntries(entries)

	parsed := make([]registryParsedArtifact, 0, len(entries))
	for _, entry := range entries {
		rel := artifactRelativePath(root, entry.Path)
		if !strings.HasSuffix(strings.ToLower(rel), ".txt") {
			continue
		}
		ast, recoveries, err := paradox.Parse(string(entry.Bytes))
		if err != nil || len(recoveries) > 0 {
			continue
		}
		parsed = append(parsed, registryParsedArtifact{entry: entry, rel: rel, ast: ast})
	}

	for _, file := range parsed {
		if under(file.rel, "common/modifier_definitions/") {
			for _, item := range file.ast.Entries {
				if item.Key != nil {
					knownModifiers[*item.Key] = true
				}
			}
		}
		if under(file.rel, "common/idea_tags/") {
			for _, item := range file.ast.Entries {
				if item.Key != nil && *item.Key == "idea_categories" && item.Block != nil {
					for _, category := range item.Block.Entries {
						if category.Key != nil {
							knownCategories[*category.Key] = true
						}
					}
				}
			}
			walkEntries(file.ast, func(item *paradox.Entry) {
				if item.Key != nil && (*item.Key == "slot" || *item.Key == "character_slot") && item.Block == nil {
					knownSlots[item.Scalar.StrValue()] = true
				}
			})
		}
	}

	for _, file := range parsed {
		if modifierCatalogReady {
			scanModifierNames(&report, file.rel, file.ast, knownModifiers, catalog.ModifierPatterns)
		}
		if catalog.ideaTagsKnown {
			scanIdeaSlots(&report, file.rel, file.ast, knownCategories, knownSlots)
		}
		scanScriptIDs(&report, file.rel, file.ast)
		scanCharacterInstances(&report, file.rel, file.ast)
		scanRawVanillaCharacterCollisions(&report, file)
	}
	scanFlagCoverage(&report, root, entries, parsed)

	report.Findings = dedupeRegistryFindings(report.Findings)
	sort.Slice(report.Findings, func(i, j int) bool {
		a, b := report.Findings[i], report.Findings[j]
		if a.Path != b.Path {
			return a.Path < b.Path
		}
		if a.Line != b.Line {
			return a.Line < b.Line
		}
		if a.Code != b.Code {
			return a.Code < b.Code
		}
		return a.Token < b.Token
	})
	return report
}

type registryIdeaTagsKey struct{}

func loadRegistryCatalog() registryCatalog {
	var out registryCatalog
	if rules, err := cwtools.Load(); err == nil {
		out.Modifiers, out.ModifierPatterns = rules.Modifiers, rules.ModifierPatterns
		out.Source.Repository, out.Source.Commit = cwtools.Repository, rules.Commit
		out.Source.License, out.Source.Dir = cwtools.License, rules.Dir
	}
	tags, err := gameindex.Load(context.Background(), registryIdeaTagsKey{}, gameindex.IdeaTags, func(raw []byte) ([2][]string, error) {
		var tags struct {
			Categories []string `json:"idea_categories"`
			Slots      []string `json:"idea_slots"`
		}
		err := json.Unmarshal(raw, &tags)
		return [2][]string{tags.Categories, tags.Slots}, err
	})
	if err == nil && len(tags[0]) > 0 {
		out.IdeaCategories, out.IdeaSlots, out.ideaTagsKnown = tags[0], tags[1], true
	}
	return out
}

func stringSet(values []string) map[string]bool {
	out := make(map[string]bool, len(values))
	for _, value := range values {
		out[value] = true
	}
	return out
}

func artifactRelativePath(root, full string) string {
	full = strings.ReplaceAll(full, "\\", "/")
	if root != "" && strings.HasPrefix(full, root) {
		return strings.TrimPrefix(full, root)
	}
	return full
}

func under(rel, prefix string) bool {
	return strings.HasPrefix(strings.ToLower(rel), strings.ToLower(prefix))
}

func walkEntries(block *paradox.Block, fn func(*paradox.Entry)) {
	if block == nil {
		return
	}
	for _, entry := range block.Entries {
		fn(entry)
		walkEntries(entry.Block, fn)
	}
}

func walkEntriesWithParent(block *paradox.Block, fn func(parent *paradox.Block, entry *paradox.Entry)) {
	if block == nil {
		return
	}
	for _, entry := range block.Entries {
		fn(block, entry)
		walkEntriesWithParent(entry.Block, fn)
	}
}

var modifierSelectorKeys = map[string]bool{
	"tag": true, "country": true, "state": true, "scope": true, "target": true,
}

var modifierDefinitionStructureKeys = map[string]bool{
	"icon": true, "enable": true, "remove_trigger": true, "attacker_modifier": true,
	"custom_modifier_tooltip": true, "valid_relation_trigger": true,
}

var modifierBlockStructureKeys = map[string]bool{
	"custom_modifier_tooltip": true,
}

var (
	weightParentScoreKeys = map[string]bool{"factor": true, "base": true}
	weightOperatorKeys    = map[string]bool{"factor": true, "add": true, "base": true}
)

func isWeightModifierBlock(parent, block *paradox.Block) bool {
	return hasScalarKeyIn(parent, weightParentScoreKeys) || hasScalarKeyIn(block, weightOperatorKeys)
}

func hasScalarKeyIn(block *paradox.Block, keys map[string]bool) bool {
	if block == nil {
		return false
	}
	for _, entry := range block.Entries {
		if entry.Key != nil && entry.Block == nil && keys[*entry.Key] {
			return true
		}
	}
	return false
}

func scanModifierNames(report *RegistryReport, rel string, ast *paradox.Block, known map[string]bool, patterns []string) {
	if !under(rel, "common/") && !under(rel, "history/") && !under(rel, "events/") {
		return
	}
	walkEntriesWithParent(ast, func(parent *paradox.Block, entry *paradox.Entry) {
		if entry.Key == nil || entry.Block == nil || (*entry.Key != "modifier" && *entry.Key != "targeted_modifier") {
			return
		}
		if isWeightModifierBlock(parent, entry.Block) {
			return
		}
		for _, child := range entry.Block.Entries {
			if child.Key == nil || child.Block != nil || modifierSelectorKeys[*child.Key] ||
				modifierBlockStructureKeys[*child.Key] || strings.HasPrefix(*child.Key, "@") {
				continue
			}
			if !known[*child.Key] && !matchesCWTModifierPattern(*child.Key, patterns) {
				addRegistryFinding(report, RegistryFinding{Code: "modifier.unknown", Status: report.msg.Text("registry.status.review", nil), Path: rel, Line: child.Line, Token: *child.Key,
					Text: report.msg.Text("registry.modifier_unknown", exportmsg.Args{"name": *child.Key})})
			}
		}
	})
	if under(rel, "common/dynamic_modifiers/") || under(rel, "common/modifiers/") {
		for _, definition := range ast.Entries {
			if definition.Key == nil || definition.Block == nil {
				continue
			}
			for _, child := range definition.Block.Entries {
				if child.Key == nil || child.Block != nil || modifierDefinitionStructureKeys[*child.Key] || strings.HasPrefix(*child.Key, "@") {
					continue
				}
				if !known[*child.Key] && !matchesCWTModifierPattern(*child.Key, patterns) {
					addRegistryFinding(report, RegistryFinding{Code: "modifier.unknown", Status: report.msg.Text("registry.status.review", nil), Path: rel, Line: child.Line, Token: *child.Key,
						Text: report.msg.Text("registry.modifier_unknown", exportmsg.Args{"name": *child.Key})})
				}
			}
		}
	}
}

func matchesCWTModifierPattern(value string, patterns []string) bool {
	for _, pattern := range patterns {
		open, close := strings.Index(pattern, "<"), strings.Index(pattern, ">")
		if open < 0 || close <= open {
			continue
		}
		prefix, suffix := pattern[:open], pattern[close+1:]
		if strings.HasPrefix(value, prefix) && strings.HasSuffix(value, suffix) && len(value) > len(prefix)+len(suffix) {
			return true
		}
	}
	return false
}

func scanIdeaSlots(report *RegistryReport, rel string, ast *paradox.Block, categories, slots map[string]bool) {
	if !under(rel, "common/ideas/") && !under(rel, "common/characters/") {
		return
	}
	if under(rel, "common/ideas/") {
		for _, wrapper := range ast.Entries {
			if wrapper.Key == nil || *wrapper.Key != "ideas" || wrapper.Block == nil {
				continue
			}
			for _, category := range wrapper.Block.Entries {
				if category.Key != nil && !categories[*category.Key] {
					addRegistryFinding(report, RegistryFinding{Code: "idea_category.source_unregistered", Status: report.msg.Text("registry.status.unregistered", nil), Path: rel, Line: category.Line, Token: *category.Key,
						Text: report.msg.Text("registry.idea_category_unregistered", exportmsg.Args{"name": *category.Key})})
				}
			}
		}
	}
	walkEntries(ast, func(entry *paradox.Entry) {
		if entry.Key == nil || (*entry.Key != "slot" && *entry.Key != "character_slot") || entry.Block != nil {
			return
		}
		slot := entry.Scalar.StrValue()
		if slot != "" && !slots[slot] {
			addRegistryFinding(report, RegistryFinding{Code: "idea_slot.source_unregistered", Status: report.msg.Text("registry.status.unregistered", nil), Path: rel, Line: entry.Line, Token: slot,
				Text: report.msg.Text("registry.idea_slot_unregistered", exportmsg.Args{"name": slot})})
		}
	})
}

var identifierValueKeys = map[string]bool{
	"id": true, "add_namespace": true, "namespace": true,
}

func scanScriptIDs(report *RegistryReport, rel string, ast *paradox.Block) {
	if !under(rel, "common/") && !under(rel, "events/") && !under(rel, "history/") {
		return
	}
	walkEntries(ast, func(entry *paradox.Entry) {
		if entry.Key != nil && !isASCIIIdentifier(*entry.Key) {
			addRegistryFinding(report, RegistryFinding{Code: "script_id.non_ascii", Status: report.msg.Text("registry.status.review", nil), Path: rel, Line: entry.Line, Token: *entry.Key,
				Text: report.msg.Text("registry.script_id_non_ascii", exportmsg.Args{"id": *entry.Key})})
		}
		if entry.Key != nil && identifierValueKeys[*entry.Key] && entry.Block == nil {
			value := entry.Scalar.StrValue()
			if !isASCIIIdentifier(value) {
				addRegistryFinding(report, RegistryFinding{Code: "script_id.non_ascii", Status: report.msg.Text("registry.status.review", nil), Path: rel, Line: entry.Line, Token: value,
					Text: report.msg.Text("registry.script_id_non_ascii", exportmsg.Args{"id": value})})
			}
		}
	})
}

func isASCIIIdentifier(value string) bool {
	if value == "" || !utf8.ValidString(value) {
		return false
	}
	for _, r := range value {
		if r > 0x7f {
			return false
		}
	}
	return true
}

func scanCharacterInstances(report *RegistryReport, rel string, ast *paradox.Block) {
	if !under(rel, "common/characters/") {
		return
	}
	walkEntries(ast, func(entry *paradox.Entry) {
		if entry.Key == nil || *entry.Key != "instance" || entry.Block == nil {
			return
		}
		if len(entry.Block.Entries) == 0 || entry.Block.Entries[0].Key == nil || *entry.Block.Entries[0].Key != "allowed" {
			addRegistryFinding(report, RegistryFinding{Code: "character_instance.allowed_not_first", Status: report.msg.Text("registry.status.review", nil), Path: rel, Line: entry.Line, Token: "instance",
				Text: report.msg.Text("registry.instance_allowed_first", nil)})
		}
	})
}

func scanRawVanillaCharacterCollisions(report *RegistryReport, file registryParsedArtifact) {
	if !under(file.rel, "common/characters/") || (file.entry.Origin == "generated" && file.entry.SourceEntity == "characters") {
		return
	}
	vanilla := loadCrossRefVanillaIDs()["character"]
	for _, wrapper := range file.ast.Entries {
		if wrapper.Key == nil || wrapper.Block == nil {
			continue
		}
		blocks := []*paradox.Entry{wrapper}
		if *wrapper.Key == "characters" {
			blocks = wrapper.Block.Entries
		}
		for _, definition := range blocks {
			if definition.Key == nil || !vanilla[*definition.Key] {
				continue
			}
			addRegistryFinding(report, RegistryFinding{Code: "character_id.vanilla_collision", Status: report.msg.Text("registry.status.review", nil), Path: file.rel, Line: definition.Line, Token: *definition.Key,
				Text: report.msg.Text("registry.character_vanilla_id", exportmsg.Args{"id": *definition.Key})})
		}
	}
}

func scanFlagCoverage(report *RegistryReport, root string, entries []Entry, parsed []registryParsedArtifact) {
	tags := map[string]bool{}
	for _, file := range parsed {
		if under(file.rel, "common/country_tags/") {
			for _, item := range file.ast.Entries {
				if item.Key != nil && item.Block == nil && strings.HasPrefix(strings.ToLower(item.Scalar.StrValue()), "countries/") && plausibleCountryTag(*item.Key) {
					tags[strings.ToUpper(*item.Key)] = true
				}
			}
		}
		if under(file.rel, "history/countries/") {
			name := strings.TrimSuffix(path.Base(file.rel), path.Ext(file.rel))
			if cut := strings.Index(name, " -"); cut >= 0 {
				name = name[:cut]
			}
			if plausibleCountryTag(name) {
				tags[strings.ToUpper(name)] = true
			}
		}
	}
	if len(tags) == 0 {
		return
	}

	present := map[string]bool{}
	for _, entry := range entries {
		rel := strings.ToLower(artifactRelativePath(root, entry.Path))
		present[rel] = true
	}
	ordered := make([]string, 0, len(tags))
	for tag := range tags {
		ordered = append(ordered, tag)
	}
	sort.Strings(ordered)
	for _, tag := range ordered {
		low := strings.ToLower(tag)
		var missing []string
		for _, candidate := range []string{"gfx/flags/" + low + ".tga", "gfx/flags/medium/" + low + ".tga", "gfx/flags/small/" + low + ".tga"} {
			if !present[candidate] {
				missing = append(missing, candidate)
			}
		}
		if len(missing) > 0 {
			addRegistryFinding(report, RegistryFinding{Code: "country_flag.source_unregistered", Status: report.msg.Text("registry.status.unregistered", nil), Path: "gfx/flags", Token: tag,
				Text: report.msg.Text("registry.flags_unregistered", exportmsg.Args{"tag": tag, "files": strings.Join(missing, ", ")})})
		}
	}
}

func plausibleCountryTag(value string) bool {
	value = strings.TrimSpace(value)
	if len(value) < 2 || len(value) > 32 || !isASCIIIdentifier(value) {
		return false
	}
	for _, r := range value {
		if !(r == '_' || r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z' || r >= '0' && r <= '9') {
			return false
		}
	}
	return true
}

func addRegistryFinding(report *RegistryReport, finding RegistryFinding) {
	report.Findings = append(report.Findings, finding)
}

func dedupeRegistryFindings(in []RegistryFinding) []RegistryFinding {
	seen := map[string]bool{}
	out := make([]RegistryFinding, 0, len(in))
	for _, finding := range in {
		key := fmt.Sprintf("%s\x00%s\x00%d\x00%s", finding.Code, finding.Path, finding.Line, finding.Token)
		if !seen[key] {
			seen[key] = true
			out = append(out, finding)
		}
	}
	return out
}
