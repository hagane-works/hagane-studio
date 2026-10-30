package exportcheck

import (
	_ "embed"
	"encoding/json"
	"haganeworks/internal/hoi4/emit/locout"
	"regexp"
	"sort"
	"strings"
	"sync"
)

//go:embed gendata/content_language_specs.json
var contentLanguageSpecsJSON []byte

type contentEntitySpec struct {
	LocaleFields       []string   `json:"locale_fields"`
	NestedLocaleFields []string   `json:"nested_locale_fields"`
	Children           [][]string `json:"children"`
	DisplayIDFields    []string   `json:"display_id_fields"`
}

type contentLanguageSpecs struct {
	ContentLocales    []string                     `json:"content_locales"`
	EntitySpecs       map[string]contentEntitySpec `json:"entity_specs"`
	GroupEntitySpecs  map[string]string            `json:"group_entity_specs"`
	GroupTables       map[string][]any             `json:"group_tables"`
	contentLocaleSet  map[string]bool
	contentLocaleRank map[string]int
}

var contentSpecsOnce sync.Once

var contentSpecs contentLanguageSpecs

func contentLanguageSpecsData() *contentLanguageSpecs {
	contentSpecsOnce.Do(func() {
		if err := json.Unmarshal(contentLanguageSpecsJSON, &contentSpecs); err != nil {
			panic("content_language_specs.json 解析失败: " + err.Error())
		}
		contentSpecs.contentLocaleSet = map[string]bool{}
		contentSpecs.contentLocaleRank = map[string]int{}
		for i, code := range contentSpecs.ContentLocales {
			contentSpecs.contentLocaleSet[code] = true
			contentSpecs.contentLocaleRank[code] = i
		}
	})
	return &contentSpecs
}

type ContentLanguageDiagnostic struct {
	Language string   `json:"language"`
	Fields   []string `json:"fields"`
}

type ContentLanguageClosureAudit struct {
	EnabledLanguages   []string                    `json:"enabled_languages"`
	PopulatedLanguages []string                    `json:"populated_languages"`
	MissingLanguages   []string                    `json:"missing_languages"`
	Diagnostics        []ContentLanguageDiagnostic `json:"diagnostics"`
}

func (a *ContentLanguageClosureAudit) normalize() {
	if a.EnabledLanguages == nil {
		a.EnabledLanguages = []string{}
	}
	if a.PopulatedLanguages == nil {
		a.PopulatedLanguages = []string{}
	}
	if a.MissingLanguages == nil {
		a.MissingLanguages = []string{}
	}
	if a.Diagnostics == nil {
		a.Diagnostics = []ContentLanguageDiagnostic{}
	}
	for i := range a.Diagnostics {
		if a.Diagnostics[i].Fields == nil {
			a.Diagnostics[i].Fields = []string{}
		}
	}
}

func member(entity map[string]any, field string) (any, bool) {
	if entity == nil {
		return nil, false
	}
	v, ok := entity[field]
	return v, ok
}

func iterGroup(value any) []map[string]any {
	switch v := value.(type) {
	case nil:
		return nil
	case map[string]any:
		return []map[string]any{v}
	case []map[string]any:
		return v
	case []any:
		out := make([]map[string]any, 0, len(v))
		for _, item := range v {
			if m, ok := item.(map[string]any); ok {
				out = append(out, m)
			}
		}
		return out
	}
	return nil
}

func nonblankMember(entity map[string]any, field string) bool {
	v, ok := member(entity, field)
	if !ok {
		return false
	}
	return strings.TrimSpace(anyToString(v)) != ""
}

var intelAgencyKeyRE = regexp.MustCompile(`^[A-Za-z0-9_.]+$`)

func intelAgencyHasLocKey(entity map[string]any) bool {
	var candidates []string
	names, ok := member(entity, "names")
	if !ok || !anyTruthy(names) {
		if agencyID, ok := member(entity, "agency_id"); ok && anyTruthy(agencyID) {
			candidates = []string{anyToString(agencyID)}
		}
	} else {
		candidates = anyToStringSlice(names)
	}
	for _, value := range candidates {
		if intelAgencyKeyRE.MatchString(strings.TrimSpace(value)) {
			return true
		}
	}
	return false
}

var closureRequiredNonblank = map[string]string{
	"specialization": "specialization_id", "special_project": "project_token",
	"dynamic_modifier": "dm_id", "autonomy_state": "autonomy_id",
	"mio": "mio_id", "bookmark": "bookmark_id", "panel": "token",
	"bookmark_country": "tag",
}

func entityCanEmitLocalisation(entity map[string]any, specName string) bool {
	if field, ok := closureRequiredNonblank[specName]; ok {
		return nonblankMember(entity, field)
	}
	if specName == "state" {
		v, ok := member(entity, "state_id")
		return ok && v != nil
	}
	if specName == "intel_agency" {
		return intelAgencyHasLocKey(entity)
	}
	return true
}

func fieldCanEmitLocalisation(entity map[string]any, specName, field string) bool {
	if specName == "doctrine" {

		keyField := "description"
		if field == "display_name_locs" {
			keyField = "name"
		}
		v, _ := member(entity, keyField)
		token := strings.TrimSpace(anyToString(v))
		return token != "" && !strings.Contains(token, " ")
	}
	if specName == "panel_control" {
		v, _ := member(entity, "kind")
		kind := strings.ToLower(strings.TrimSpace(anyToString(v)))
		switch field {
		case "text_locs":
			return kind == "text" || kind == "button"
		case "tooltip_locs":
			return kind == "button"
		}
	}
	return true
}

func recordLocaleMap(
	value any, path string, fieldsByLanguage map[string]map[string]bool,
	entity map[string]any, spec contentEntitySpec, field string,
) {
	locales, ok := value.(map[string]any)
	if !ok {
		return
	}
	idValue := ""
	for _, idField := range spec.DisplayIDFields {
		candidate, ok := member(entity, idField)
		if ok && strings.TrimSpace(anyToString(candidate)) != "" {
			idValue = strings.TrimSpace(anyToString(candidate))
			break
		}
	}

	if field == "display_name_locs" && len(spec.DisplayIDFields) == 1 &&
		spec.DisplayIDFields[0] == "state_id" {
		if idValue != "" {
			idValue = "STATE_" + idValue
		}
	}
	data := contentLanguageSpecsData()
	for rawLanguage, rawText := range locales {
		if !data.contentLocaleSet[rawLanguage] {
			continue
		}
		text, isString := rawText.(string)
		if !isString || strings.TrimSpace(text) == "" {
			continue
		}

		idArg := locout.Value{Kind: locout.KNull}
		if idValue != "" {
			idArg = locout.Str(idValue)
		}
		if field == "display_name_locs" &&
			locout.IsPlaceholderDisplayName(text, idArg) {
			continue
		}
		if fieldsByLanguage[rawLanguage] == nil {
			fieldsByLanguage[rawLanguage] = map[string]bool{}
		}
		fieldsByLanguage[rawLanguage][path] = true
	}
}

func scanEntity(
	entity map[string]any, path string,
	fieldsByLanguage map[string]map[string]bool, specName string,
) {
	if entity == nil {
		return
	}
	if !entityCanEmitLocalisation(entity, specName) {
		return
	}
	data := contentLanguageSpecsData()
	spec, ok := data.EntitySpecs[specName]
	if !ok {
		return
	}
	for _, field := range spec.LocaleFields {
		if !fieldCanEmitLocalisation(entity, specName, field) {
			continue
		}
		if value, present := member(entity, field); present {
			recordLocaleMap(value, path+"."+field, fieldsByLanguage,
				entity, spec, field)
		}
	}
	for _, field := range spec.NestedLocaleFields {
		value, _ := member(entity, field)
		nested, ok := value.(map[string]any)
		if !ok {
			continue
		}
		for _, localeMap := range nested {
			recordLocaleMap(localeMap, path+"."+field+"[]", fieldsByLanguage,
				entity, spec, field)
		}
	}
	for _, child := range spec.Children {
		if len(child) != 2 {
			continue
		}
		field, childSpecName := child[0], child[1]
		value, present := member(entity, field)
		if !present || value == nil {
			continue
		}
		childPath := path + "." + field + "[]"
		for _, item := range iterGroup(value) {
			scanEntity(item, childPath, fieldsByLanguage, childSpecName)
		}
	}
}

func normaliseEnabledLanguages(values []string) []string {
	data := contentLanguageSpecsData()
	enabled := map[string]bool{"en": true}
	for _, value := range values {
		code := strings.ToLower(strings.TrimSpace(value))
		if data.contentLocaleSet[code] {
			enabled[code] = true
		}
	}
	out := []string{}
	for _, code := range data.ContentLocales {
		if enabled[code] {
			out = append(out, code)
		}
	}
	return out
}

func AuditContentLanguageClosure(
	exportLanguages []string, groups map[string]any,
) ContentLanguageClosureAudit {
	data := contentLanguageSpecsData()
	fieldsByLanguage := map[string]map[string]bool{}

	names := make([]string, 0, len(groups))
	for name := range groups {
		if _, known := data.GroupEntitySpecs[name]; known {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	for _, name := range names {
		specName := data.GroupEntitySpecs[name]
		for _, entity := range iterGroup(groups[name]) {
			scanEntity(entity, name+"[]", fieldsByLanguage, specName)
		}
	}

	return finishClosureAudit(exportLanguages, fieldsByLanguage)
}

func finishClosureAudit(
	exportLanguages []string, fieldsByLanguage map[string]map[string]bool,
) ContentLanguageClosureAudit {
	data := contentLanguageSpecsData()
	enabled := normaliseEnabledLanguages(exportLanguages)
	enabledSet := map[string]bool{}
	for _, code := range enabled {
		enabledSet[code] = true
	}
	populated, missing := []string{}, []string{}
	for _, code := range data.ContentLocales {
		if len(fieldsByLanguage[code]) == 0 {
			continue
		}
		populated = append(populated, code)
		if !enabledSet[code] {
			missing = append(missing, code)
		}
	}
	diagnostics := []ContentLanguageDiagnostic{}
	for _, code := range missing {
		fields := make([]string, 0, len(fieldsByLanguage[code]))
		for path := range fieldsByLanguage[code] {
			fields = append(fields, path)
		}
		sort.Strings(fields)
		diagnostics = append(diagnostics, ContentLanguageDiagnostic{
			Language: code, Fields: fields,
		})
	}
	out := ContentLanguageClosureAudit{
		EnabledLanguages:   enabled,
		PopulatedLanguages: populated,
		MissingLanguages:   missing,
		Diagnostics:        diagnostics,
	}
	out.normalize()
	return out
}
