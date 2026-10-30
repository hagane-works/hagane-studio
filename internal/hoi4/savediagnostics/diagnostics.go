package savediagnostics

import (
	"bytes"
	"encoding/json"
	"math/big"
	"sort"
)

type Change string

const (
	Introduced Change = "introduced"
	Unchanged  Change = "unchanged"
	Improved   Change = "improved"
	Worsened   Change = "worsened"
	Changed    Change = "changed"
	Resolved   Change = "resolved"
)

type EntityRef struct {
	Type string `json:"type"`
	ID   string `json:"id"`
}

type Diagnostic struct {
	Key         string         `json:"key"`
	Code        string         `json:"code"`
	Severity    string         `json:"severity"`
	EntityRef   EntityRef      `json:"entity_ref"`
	Evidence    map[string]any `json:"evidence"`
	RepairRoute string         `json:"repair_route"`
	Affected    []string       `json:"affected"`
	Change      Change         `json:"change,omitempty"`
	Source      any            `json:"source,omitempty"`
}

type Envelope struct {
	Diagnostics *[]Diagnostic `json:"diagnostics,omitempty"`
}

func Evaluated(items []Diagnostic) *[]Diagnostic {
	out := normalizedSet(items, false)
	return &out
}

func Current(items []Diagnostic) []Diagnostic {
	return normalizedSet(items, true)
}

func Compare(before, current []Diagnostic) []Diagnostic {
	previous := Current(before)
	next := Current(current)
	byKey := make(map[string]Diagnostic, len(previous))
	for _, diagnostic := range previous {
		byKey[diagnostic.Key] = diagnostic
	}
	result := make([]Diagnostic, 0, len(previous)+len(next))
	for _, diagnostic := range next {
		prior, exists := byKey[diagnostic.Key]
		if !exists {
			diagnostic.Change = Introduced
		} else {
			delete(byKey, diagnostic.Key)
			diagnostic.Change = compareFinding(prior, diagnostic)
		}
		result = append(result, diagnostic)
	}
	for _, diagnostic := range byKey {
		diagnostic.Change = Resolved
		result = append(result, diagnostic)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Key < result[j].Key })
	return result
}

func compareFinding(before, current Diagnostic) Change {
	oldInNew := subset(before.Affected, current.Affected)
	newInOld := subset(current.Affected, before.Affected)
	switch {
	case oldInNew && !newInOld:
		return Worsened
	case newInOld && !oldInNew:
		return Improved
	case !oldInNew || !newInOld:
		return Changed
	case payloadFingerprint(before) != payloadFingerprint(current):
		return Changed
	default:
		return Unchanged
	}
}

func subset(left, right []string) bool {
	members := make(map[string]bool, len(right))
	for _, value := range right {
		members[value] = true
	}
	for _, value := range left {
		if !members[value] {
			return false
		}
	}
	return true
}

func normalizedSet(items []Diagnostic, currentOnly bool) []Diagnostic {
	byKey := make(map[string]Diagnostic, len(items))
	for _, diagnostic := range items {
		if currentOnly && diagnostic.Change == Resolved {
			continue
		}
		diagnostic = cloneDiagnostic(diagnostic)
		if currentOnly {
			diagnostic.Change = ""
		}
		prior, exists := byKey[diagnostic.Key]
		if !exists || prefer(diagnostic, prior) {
			byKey[diagnostic.Key] = diagnostic
		}
	}
	out := make([]Diagnostic, 0, len(byKey))
	for _, diagnostic := range byKey {
		out = append(out, diagnostic)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Key < out[j].Key })
	return out
}

func prefer(candidate, prior Diagnostic) bool {

	if (candidate.Change == Resolved) != (prior.Change == Resolved) {
		return candidate.Change != Resolved
	}

	return jsonText(candidate) < jsonText(prior)
}

func cloneDiagnostic(diagnostic Diagnostic) Diagnostic {
	affected := make(map[string]bool, len(diagnostic.Affected))
	for _, subject := range diagnostic.Affected {
		affected[subject] = true
	}
	diagnostic.Affected = make([]string, 0, len(affected))
	for subject := range affected {
		diagnostic.Affected = append(diagnostic.Affected, subject)
	}
	sort.Strings(diagnostic.Affected)
	if diagnostic.Evidence == nil {
		diagnostic.Evidence = map[string]any{}
	} else if evidence, ok := normalizedJSON(diagnostic.Evidence, true).(map[string]any); ok {
		diagnostic.Evidence = evidence
	}
	if diagnostic.Source != nil {
		diagnostic.Source = normalizedJSON(diagnostic.Source, false)
	}
	return diagnostic
}

func payloadFingerprint(diagnostic Diagnostic) string {

	return semanticFingerprint(map[string]any{
		"code": diagnostic.Code, "severity": diagnostic.Severity,
		"entity_ref": map[string]any{"type": diagnostic.EntityRef.Type, "id": diagnostic.EntityRef.ID},
		"evidence":   diagnostic.Evidence, "repair_route": diagnostic.RepairRoute,
		"source": diagnostic.Source,
	})
}

func isEvidenceSet(field string) bool {
	switch field {
	case "focus_ids", "bookmark_ids", "default_bookmarks", "paths", "interesting_countries":
		return true
	default:
		return false
	}
}

func normalizedJSON(value any, evidence bool) any {
	raw, err := json.Marshal(value)
	if err != nil {

		return value
	}
	var decoded any
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if decoder.Decode(&decoded) != nil {
		return value
	}
	return normalizeValue(decoded, "", evidence)
}

func normalizeValue(value any, field string, evidence bool) any {
	switch value := value.(type) {
	case map[string]any:
		out := make(map[string]any, len(value))
		for key, child := range value {
			out[key] = normalizeValue(child, key, evidence)
		}
		return out
	case []any:
		out := make([]any, len(value))
		for index, child := range value {
			out[index] = normalizeValue(child, "", evidence)
		}
		if evidence && isEvidenceSet(field) {
			sort.Slice(out, func(i, j int) bool {
				left, right := semanticFingerprint(out[i]), semanticFingerprint(out[j])
				if left == right {
					return jsonText(out[i]) < jsonText(out[j])
				}
				return left < right
			})
			unique := make([]any, 0, len(out))
			for _, item := range out {
				if len(unique) == 0 || semanticFingerprint(item) != semanticFingerprint(unique[len(unique)-1]) {
					unique = append(unique, item)
				}
			}
			return unique
		}
		return out
	default:
		return value
	}
}

func semanticFingerprint(value any) string {

	switch value := value.(type) {
	case json.Number:
		if rational, ok := new(big.Rat).SetString(value.String()); ok {
			return "number:" + rational.RatString()
		}
	case map[string]any:
		keys := make([]string, 0, len(value))
		for key := range value {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		pairs := make([][2]string, 0, len(keys))
		for _, key := range keys {
			pairs = append(pairs, [2]string{key, semanticFingerprint(value[key])})
		}
		return "object:" + jsonText(pairs)
	case []any:
		items := make([]string, 0, len(value))
		for _, item := range value {
			items = append(items, semanticFingerprint(item))
		}
		return "array:" + jsonText(items)
	}
	return jsonText(value)
}

func jsonText(value any) string {
	raw, err := json.Marshal(value)
	if err != nil {
		return "invalid JSON:" + err.Error()
	}
	return string(raw)
}
