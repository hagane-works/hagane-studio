package exportengine

import (
	"encoding/json"
	"path"
	"regexp"
	"sort"
	"strings"

	"haganeworks/internal/hoi4/exportmsg"
	"haganeworks/internal/hoi4/paradox"
)

const ReferenceIntegrityLogBasename = "_REFERENCE_INTEGRITY.log"

type ReferenceIntegrityFinding struct {
	Code        string
	File        string
	Line        int
	Target      string
	Summary     string
	Consequence string
	EditAt      string
	Severity    string
}

func (f ReferenceIntegrityFinding) severity() string {
	if f.Severity == "" {
		return "review"
	}
	return f.Severity
}

func (f ReferenceIntegrityFinding) line(msg exportmsg.Printer) string {
	label := msg.Text("refs.label.review", nil)
	if f.severity() == "warning" {
		label = msg.Text("refs.label.broken", nil)
	}
	return msg.Text("refs.log.line", exportmsg.Args{
		"label": label, "code": f.Code, "file": f.File, "line": f.Line,
		"summary": f.Summary, "impact": f.Consequence, "edit": f.EditAt,
	})
}

type ReferenceIntegrityReport struct {
	Findings []ReferenceIntegrityFinding
	msg      exportmsg.Printer
}

func (r ReferenceIntegrityReport) Headline() string {
	if len(r.Findings) == 0 {
		return ""
	}
	broken := 0
	for _, f := range r.Findings {
		if f.severity() == "warning" {
			broken++
		}
	}
	review := len(r.Findings) - broken
	args := exportmsg.Args{"broken": broken, "review": review, "log": ReferenceIntegrityLogBasename}
	switch {
	case broken > 0 && review > 0:
		return r.msg.Text("refs.headline.broken_and_review", args)
	case broken > 0:
		return r.msg.Text("refs.headline.broken", args)
	default:
		return r.msg.Text("refs.headline.review", args)
	}
}

func (r ReferenceIntegrityReport) Entries(modRoot string) []Entry {
	if len(r.Findings) == 0 || modRoot == "" {
		return nil
	}
	lines := commentLines(r.msg.Text("refs.log.title", nil))
	lines = append(lines, commentLines(r.msg.Text("refs.log.notes", nil))...)
	lines = append(lines, "")
	for _, finding := range r.Findings {
		lines = append(lines, finding.line(r.msg))
	}
	return []Entry{{
		Path:         strings.TrimSuffix(modRoot, "/") + "/" + ReferenceIntegrityLogBasename,
		Bytes:        []byte(strings.Join(lines, "\n") + "\n"),
		Origin:       "generated",
		SourceEntity: "reference_integrity",
	}}
}

type parsedArtifactScript struct {
	entry   Entry
	relPath string
	text    string
	block   *paradox.Block
	locator *tokenLocator
}

type focusDefinition struct {
	id      string
	file    string
	line    int
	entry   *paradox.Entry
	locator *tokenLocator
}

var (
	tagTokenRE       = regexp.MustCompile(`^[A-Z][A-Z0-9]{2}$`)
	oobFilenameTagRE = regexp.MustCompile(`^([A-Za-z0-9]{3})(?:_|\b)`)

	createUnitTemplateValue = regexp.MustCompile(`(?i)\bdivision_template\s*=\s*\\?"([^"\\]+)\\?"`)
)

func BuildReferenceIntegrityReport(snapshot Snapshot, entries []Entry, msg exportmsg.Printer) ReferenceIntegrityReport {
	scripts := parseFinalArtifactScripts(entries)
	definitions, foldedFocus := collectFocusDefinitions(scripts)
	expectsUnregisteredSource := projectHasDependencies(snapshot.Project)

	var findings []ReferenceIntegrityFinding
	findings = append(findings, checkFocusRelationships(definitions, foldedFocus, expectsUnregisteredSource, msg)...)
	findings = append(findings, checkCompletedFocusCase(scripts, definitions, foldedFocus, msg)...)
	findings = append(findings, checkOOBTemplateReferences(scripts, expectsUnregisteredSource, msg)...)

	sort.SliceStable(findings, func(i, j int) bool {
		if findings[i].File != findings[j].File {
			return findings[i].File < findings[j].File
		}
		if findings[i].Line != findings[j].Line {
			return findings[i].Line < findings[j].Line
		}
		if findings[i].Code != findings[j].Code {
			return findings[i].Code < findings[j].Code
		}
		return findings[i].Target < findings[j].Target
	})
	return ReferenceIntegrityReport{Findings: findings, msg: msg}
}

func parseFinalArtifactScripts(entries []Entry) []parsedArtifactScript {
	modRoot := referenceArtifactRoot(entries)
	var out []parsedArtifactScript
	for _, entry := range entries {
		if !isScriptEntry(entry) {
			continue
		}
		text := scriptText(entry)
		block, _, err := paradox.Parse(text)
		if err != nil || block == nil {

			continue
		}
		out = append(out, parsedArtifactScript{
			entry: entry, relPath: relativeArtifactPath(entry.Path, modRoot),
			text: text, block: block, locator: newTokenLocator(text),
		})
	}
	return out
}

func referenceArtifactRoot(entries []Entry) string {
	for _, entry := range entries {
		if !strings.Contains(entry.Path, "/") && strings.HasSuffix(strings.ToLower(entry.Path), ".mod") {
			return strings.TrimSuffix(entry.Path, path.Ext(entry.Path))
		}
	}
	for _, entry := range entries {
		if slash := strings.IndexByte(entry.Path, '/'); slash > 0 {
			return entry.Path[:slash]
		}
	}
	return ""
}

func relativeArtifactPath(entryPath, modRoot string) string {
	entryPath = strings.TrimPrefix(entryPath, "/")
	if modRoot != "" {
		entryPath = strings.TrimPrefix(entryPath, strings.TrimSuffix(modRoot, "/")+"/")
	}
	return entryPath
}

func collectFocusDefinitions(scripts []parsedArtifactScript) (map[string]focusDefinition, map[string][]string) {
	definitions := map[string]focusDefinition{}
	folded := map[string][]string{}
	for i := range scripts {
		script := &scripts[i]
		if !strings.HasPrefix(strings.ToLower(script.relPath), "common/national_focus/") {
			continue
		}
		walkBlock(script.block, func(entry *paradox.Entry) {
			if entry.Key == nil || entry.Block == nil ||
				(*entry.Key != "focus" && *entry.Key != "shared_focus") {
				return
			}
			ids := fieldsInBlock(entry, "id")
			if len(ids) == 0 || strings.TrimSpace(ids[0].id) == "" {
				return
			}
			id := strings.TrimSpace(ids[0].id)
			definitions[id] = focusDefinition{
				id: id, file: script.relPath, line: ids[0].line,
				entry: entry, locator: script.locator,
			}
			key := strings.ToLower(id)
			if !containsExact(folded[key], id) {
				folded[key] = append(folded[key], id)
			}
		})
	}
	return definitions, folded
}

func checkFocusRelationships(
	definitions map[string]focusDefinition,
	folded map[string][]string,
	expectsUnregisteredSource bool,
	msg exportmsg.Printer,
) []ReferenceIntegrityFinding {
	var findings []ReferenceIntegrityFinding
	for _, owner := range definitions {
		for _, relation := range owner.entry.Block.Entries {
			if relation.Key == nil || relation.Block == nil ||
				(*relation.Key != "prerequisite" && *relation.Key != "mutually_exclusive") {
				continue
			}
			for _, target := range fieldsInBlock(relation, "focus") {
				targetID := strings.TrimSpace(target.id)
				if targetID == "" {
					continue
				}
				line, _, ok := owner.locator.locate(targetID, target.line)
				if !ok {
					continue
				}
				if _, ok := definitions[targetID]; ok {
					continue
				}
				codeName, relationName := "focus_prerequisite", msg.Text("refs.relation.prerequisite", nil)
				if *relation.Key == "mutually_exclusive" {
					codeName, relationName = "focus_mutually_exclusive", msg.Text("refs.relation.mutually_exclusive", nil)
				}
				if matches := folded[strings.ToLower(targetID)]; len(matches) > 0 {
					findings = append(findings, focusCaseFinding(
						codeName+"_case", owner, targetID, matches[0], relationName, line, msg,
					))
					continue
				}
				findings = append(findings, focusMissingFinding(
					codeName, owner, targetID, relationName, line, expectsUnregisteredSource, msg,
				))
			}
		}
	}
	return findings
}

func focusCaseFinding(code string, owner focusDefinition, target, exact, relation string, line int, msg exportmsg.Printer) ReferenceIntegrityFinding {
	args := exportmsg.Args{"focus": owner.id, "relation": relation, "target": target, "exact": exact}
	return ReferenceIntegrityFinding{
		Code: code, File: owner.file, Line: line, Target: target,
		Summary:     msg.Text("refs.focus_case.summary", args),
		Consequence: msg.Text("refs.focus_case.impact", nil),
		EditAt:      msg.Text("refs.focus_case.edit", args),

		Severity: "warning",
	}
}

func focusMissingFinding(code string, owner focusDefinition, target, relation string, line int, unregistered bool, msg exportmsg.Printer) ReferenceIntegrityFinding {
	args := exportmsg.Args{"focus": owner.id, "relation": relation, "target": target}
	summary := msg.Text("refs.focus_missing.summary", args)
	severity := "warning"
	if unregistered {
		summary = msg.Text("refs.focus_unregistered.summary", args)
		severity = ""
	}
	return ReferenceIntegrityFinding{
		Code: code, File: owner.file, Line: line, Target: target, Summary: summary,
		Consequence: msg.Text("refs.focus_missing.impact", nil),
		EditAt:      msg.Text("refs.focus_missing.edit", nil),
		Severity:    severity,
	}
}

func checkCompletedFocusCase(
	scripts []parsedArtifactScript,
	definitions map[string]focusDefinition,
	folded map[string][]string,
	msg exportmsg.Printer,
) []ReferenceIntegrityFinding {
	var findings []ReferenceIntegrityFinding
	for i := range scripts {
		script := &scripts[i]
		walkBlock(script.block, func(entry *paradox.Entry) {
			if entry.Key == nil || !strings.EqualFold(*entry.Key, "has_completed_focus") || entry.Block != nil {
				return
			}
			target := strings.TrimSpace(entry.Scalar.StrValue())
			if target == "" {
				return
			}
			if _, ok := definitions[target]; ok {
				return
			}
			matches := folded[strings.ToLower(target)]
			if len(matches) == 0 {

				return
			}
			line, _, ok := script.locator.locate(target, entry.Line)
			if !ok {
				return
			}
			args := exportmsg.Args{"target": target, "exact": matches[0]}
			findings = append(findings, ReferenceIntegrityFinding{
				Code: "has_completed_focus_case", File: script.relPath, Line: line, Target: target,
				Summary:     msg.Text("refs.completed_case.summary", args),
				Consequence: msg.Text("refs.completed_case.impact", nil),
				EditAt:      msg.Text("refs.completed_case.edit", args),

				Severity: "warning",
			})
		})
	}
	return findings
}

type oobIndex struct {
	tagByFile      map[string]string
	templatesByTag map[string]map[string]bool
}

func checkOOBTemplateReferences(scripts []parsedArtifactScript, unregistered bool, msg exportmsg.Printer) []ReferenceIntegrityFinding {
	index := buildOOBIndex(scripts)
	var findings []ReferenceIntegrityFinding
	for i := range scripts {
		script := &scripts[i]
		lowerPath := strings.ToLower(script.relPath)
		if strings.HasPrefix(lowerPath, "history/units/") {
			tag := index.tagByFile[strings.ToLower(path.Base(script.relPath))]
			if tag != "" {
				findings = append(findings, checkOOBDeployments(*script, tag, index, unregistered, msg)...)
			}
		}
		walkWithCountryScope(script.block, "", func(entry *paradox.Entry, scopeTag string) {
			if entry.Key == nil || entry.Block == nil || !strings.EqualFold(*entry.Key, "create_unit") {
				return
			}
			owner := literalCountryOwner(entry)
			if owner == "" {
				owner = scopeTag
			}
			if owner == "" {
				return
			}
			for _, division := range fieldsInBlock(entry, "division") {
				match := createUnitTemplateValue.FindStringSubmatch(division.id)
				if len(match) != 2 {
					continue
				}
				template := strings.TrimSpace(match[1])
				if template == "" || index.templatesByTag[owner][template] {
					continue
				}
				line, _, ok := script.locator.locate(template, division.line)
				if !ok {
					continue
				}
				findings = append(findings, templateFinding(
					"create_unit_template", script.relPath, line, owner, template, unregistered, msg,
				))
			}
		})
	}
	return findings
}

func buildOOBIndex(scripts []parsedArtifactScript) oobIndex {
	index := oobIndex{tagByFile: map[string]string{}, templatesByTag: map[string]map[string]bool{}}

	for i := range scripts {
		script := &scripts[i]
		lowerPath := strings.ToLower(script.relPath)
		if !strings.HasPrefix(lowerPath, "history/countries/") {
			continue
		}
		tag := countryTagFromHistoryFilename(path.Base(script.relPath))
		if tag == "" {
			continue
		}
		walkBlock(script.block, func(entry *paradox.Entry) {
			if entry.Key == nil || entry.Block != nil || !strings.EqualFold(*entry.Key, "oob") {
				return
			}
			name := strings.TrimSpace(entry.Scalar.StrValue())
			if name != "" {
				index.tagByFile[strings.ToLower(name+".txt")] = tag
			}
		})
	}
	for i := range scripts {
		script := &scripts[i]
		if !strings.HasPrefix(strings.ToLower(script.relPath), "history/units/") {
			continue
		}
		base := path.Base(script.relPath)
		tag := index.tagByFile[strings.ToLower(base)]
		if tag == "" {
			tag = countryTagFromOOBFilename(base)
			if tag != "" {
				index.tagByFile[strings.ToLower(base)] = tag
			}
		}
		if tag == "" {
			continue
		}
		if index.templatesByTag[tag] == nil {
			index.templatesByTag[tag] = map[string]bool{}
		}
		walkBlock(script.block, func(entry *paradox.Entry) {
			if entry.Key == nil || entry.Block == nil || !strings.EqualFold(*entry.Key, "division_template") {
				return
			}
			for _, name := range fieldsInBlock(entry, "name") {
				if strings.TrimSpace(name.id) != "" {
					index.templatesByTag[tag][strings.TrimSpace(name.id)] = true
				}
			}
		})
	}
	return index
}

func checkOOBDeployments(script parsedArtifactScript, tag string, index oobIndex, unregistered bool, msg exportmsg.Printer) []ReferenceIntegrityFinding {
	var findings []ReferenceIntegrityFinding
	walkBlock(script.block, func(entry *paradox.Entry) {
		if entry.Key == nil || entry.Block == nil || !strings.EqualFold(*entry.Key, "division") {
			return
		}
		for _, target := range fieldsInBlock(entry, "division_template") {
			template := strings.TrimSpace(target.id)
			if template == "" || index.templatesByTag[tag][template] {
				continue
			}
			line, _, ok := script.locator.locate(template, target.line)
			if !ok {
				continue
			}
			findings = append(findings, templateFinding(
				"oob_deployment_template", script.relPath, line, tag, template, unregistered, msg,
			))
		}
	})
	return findings
}

func templateFinding(code, file string, line int, tag, template string, unregistered bool, msg exportmsg.Printer) ReferenceIntegrityFinding {
	args := exportmsg.Args{"tag": tag, "template": template}
	summary := msg.Text("refs.template_missing.summary", args)
	severity := "warning"
	if unregistered {
		summary = msg.Text("refs.template_unregistered.summary", args)
		severity = ""
	}
	return ReferenceIntegrityFinding{
		Code: code, File: file, Line: line, Target: tag + ":" + template, Summary: summary,
		Consequence: msg.Text("refs.template.impact", nil),
		EditAt:      msg.Text("refs.template.edit", args),
		Severity:    severity,
	}
}

func literalCountryOwner(entry *paradox.Entry) string {
	owners := fieldsInBlock(entry, "owner")
	if len(owners) == 0 {
		return ""
	}
	owner := strings.ToUpper(strings.TrimSpace(owners[0].id))
	if !tagTokenRE.MatchString(owner) || owner == "ROOT" || owner == "PRE" {
		return ""
	}
	return owner
}

func walkWithCountryScope(block *paradox.Block, scopeTag string, visit func(*paradox.Entry, string)) {
	if block == nil {
		return
	}
	for _, entry := range block.Entries {
		visit(entry, scopeTag)
		if entry.Block == nil {
			continue
		}
		next := scopeTag
		if entry.Key != nil && tagTokenRE.MatchString(*entry.Key) {
			next = *entry.Key
		}
		walkWithCountryScope(entry.Block, next, visit)
	}
}

func countryTagFromHistoryFilename(filename string) string {
	if len(filename) < 3 {
		return ""
	}
	tag := strings.ToUpper(filename[:3])
	if !tagTokenRE.MatchString(tag) {
		return ""
	}
	return tag
}

func countryTagFromOOBFilename(filename string) string {
	match := oobFilenameTagRE.FindStringSubmatch(strings.TrimSuffix(filename, path.Ext(filename)))
	if len(match) != 2 {
		return ""
	}
	tag := strings.ToUpper(match[1])
	if !tagTokenRE.MatchString(tag) {
		return ""
	}
	return tag
}

func projectHasDependencies(raw json.RawMessage) bool {
	var project struct {
		Dependencies []json.RawMessage `json:"dependencies"`
	}
	return json.Unmarshal(raw, &project) == nil && len(project.Dependencies) > 0
}

func containsExact(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
