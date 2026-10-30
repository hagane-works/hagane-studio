package clausewitz

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
)

var swapIdeaChildKeys = map[string]bool{"add_idea": true, "remove_idea": true}

type ScriptPositionFinding struct {
	Code           string
	Severity       string
	Key            string
	ExpectedParent string
	ActualParent   *string
	ParentPath     []string
	SourceSpan     SourceSpan
	Excerpt        string
	RepairRoute    string
}

func (f ScriptPositionFinding) ToMap() map[string]any {
	var actual any
	if f.ActualParent != nil {
		actual = *f.ActualParent
	}
	pp := f.ParentPath
	if pp == nil {
		pp = []string{}
	}
	return map[string]any{
		"code": f.Code, "severity": f.Severity, "key": f.Key,
		"expected_parent": f.ExpectedParent, "actual_parent": actual,
		"parent_path": pp, "source_span": f.SourceSpan,
		"excerpt": f.Excerpt, "repair_route": f.RepairRoute,
	}
}

func (f ScriptPositionFinding) ToLintIssue(entityKind, entityID, field string) map[string]any {
	actual := "<root>"
	if f.ActualParent != nil {
		actual = *f.ActualParent
	}
	pp := f.ParentPath
	if pp == nil {
		pp = []string{}
	}
	return map[string]any{
		"level": f.Severity, "code": f.Code,
		"msg": fmt.Sprintf("[%s] %s: %s is only legal directly inside %s; actual parent is %s",
			entityID, field, f.Key, f.ExpectedParent, actual),
		"suggested_fix": fmt.Sprintf("Move %s directly inside a %s block",
			f.Key, f.ExpectedParent),
		"entity_kind": entityKind, "entity_id": entityID, "field": field,
		"source_span": f.SourceSpan, "parent_path": pp,
		"repair_route": f.RepairRoute,
	}
}

type ScriptPositionReport struct {
	Findings       []ScriptPositionFinding
	ParserFindings []AnalysisFinding
	SourceSHA256   string
}

func (r ScriptPositionReport) IsValid() bool { return len(r.Findings) == 0 }

func (r ScriptPositionReport) ToMap() map[string]any {
	fs := make([]map[string]any, 0, len(r.Findings))
	for _, f := range r.Findings {
		fs = append(fs, f.ToMap())
	}
	pf := make([]map[string]any, 0, len(r.ParserFindings))
	for _, f := range r.ParserFindings {
		src := f.Source
		if src == nil {
			src = map[string]any{}
		}
		ev := f.Evidence
		if ev == nil {
			ev = map[string]any{}
		}
		pf = append(pf, map[string]any{
			"code": f.Code, "severity": f.Severity, "source": src,
			"evidence": ev, "repair_route": f.RepairRoute,
		})
	}
	return map[string]any{
		"findings": fs, "parser_findings": pf,
		"source_sha256": r.SourceSHA256, "is_valid": r.IsValid(),
	}
}

func AnalyzeScriptPositions(text string, rootPath []string, sourceName *string,
	analysis *Analysis) (ScriptPositionReport, error) {

	expected := sha256Hex(text)
	var ir Analysis
	if analysis != nil {
		ir = *analysis
	} else {
		ir = AnalyzeAssignments(text, rootPath, sourceName)
	}
	if ir.SourceSHA256 != expected {
		return ScriptPositionReport{}, errors.New(
			"analysis source hash does not match the supplied text")
	}

	rootDepth := len(rootPath)
	findings := []ScriptPositionFinding{}
	for _, a := range ir.Assignments {
		if !swapIdeaChildKeys[a.Key] {
			continue
		}

		var actual *string
		if len(a.ParentPath) > rootDepth {
			parsed := a.ParentPath[rootDepth:]
			if len(parsed) > 0 {
				last := parsed[len(parsed)-1]
				actual = &last
			}
		}
		if actual != nil && *actual == "swap_ideas" {
			continue
		}
		findings = append(findings, ScriptPositionFinding{
			Code:     "script.position.swap_ideas_child_outside_parent",
			Severity: "warning", Key: a.Key, ExpectedParent: "swap_ideas",
			ActualParent: actual, ParentPath: a.ParentPath,
			SourceSpan: a.Span, Excerpt: a.Excerpt,
			RepairRoute: "script.position.move_into_swap_ideas",
		})
	}
	return ScriptPositionReport{
		Findings: findings, ParserFindings: ir.Findings, SourceSHA256: ir.SourceSHA256,
	}, nil
}

const fatalRegistrySchemaVersion = "hagane.cross-ref-fatal-registry.v1"

type FatalRegistry struct {
	GameVersion        string
	Verified           bool
	VerificationSource *string
	FatalKinds         map[[2]string]bool
}

func ParseFatalRegistry(raw []byte) (FatalRegistry, error) {
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil || payload == nil {
		return FatalRegistry{}, errors.New("cross-ref fatal registry root must be an object")
	}
	if s, _ := payload["schema_version"].(string); s != fatalRegistrySchemaVersion {
		return FatalRegistry{}, errors.New("unsupported cross-ref fatal registry schema")
	}
	gv := strings.TrimSpace(anyToStr(payload["game_version"]))
	if gv == "" {
		return FatalRegistry{}, errors.New("cross-ref fatal registry requires game_version")
	}
	verified, ok := payload["verified"].(bool)
	if !ok {
		return FatalRegistry{}, errors.New("cross-ref fatal registry verified must be boolean")
	}
	var source *string
	if v, present := payload["verification_source"]; present && v != nil {
		s := strings.TrimSpace(anyToStr(v))
		source = &s
	}
	rows, ok := payload["entries"].([]any)
	if !ok {
		return FatalRegistry{}, errors.New("cross-ref fatal registry entries must be an array")
	}
	if verified && (source == nil || *source == "") {
		return FatalRegistry{}, errors.New("verified cross-ref fatal registry requires evidence source")
	}
	if !verified && len(rows) > 0 {
		return FatalRegistry{}, errors.New("unverified cross-ref fatal registry must remain empty")
	}
	kinds := map[[2]string]bool{}
	for _, row := range rows {
		obj, ok := row.(map[string]any)
		if !ok {
			return FatalRegistry{}, errors.New("cross-ref fatal registry entry must be an object")
		}
		effect := strings.TrimSpace(anyToStr(obj["effect_name"]))
		target := strings.TrimSpace(anyToStr(obj["target_type"]))
		if effect == "" || target == "" {
			return FatalRegistry{}, errors.New("fatal registry entry requires effect_name and target_type")
		}
		kinds[[2]string{effect, target}] = true
	}
	return FatalRegistry{GameVersion: gv, Verified: verified,
		VerificationSource: source, FatalKinds: kinds}, nil
}

func anyToStr(v any) string {
	switch t := v.(type) {
	case nil:
		return ""
	case string:
		return t
	}
	b, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	return string(b)
}

func RegistryEvidenceForVersion(gameVersion string, raw []byte, loadErr error) map[string]any {
	requested := strings.TrimSpace(gameVersion)
	if loadErr != nil {
		return map[string]any{
			"status": "unavailable", "requested_game_version": requested,
			"fatal_reference_kinds": []any{},
			"reason":                "registry load failed: OSError",
		}
	}
	reg, err := ParseFatalRegistry(raw)
	if err != nil {
		return map[string]any{
			"status": "unavailable", "requested_game_version": requested,
			"fatal_reference_kinds": []any{},
			"reason":                "registry load failed: ValueError",
		}
	}
	eligible := requested == reg.GameVersion && reg.Verified
	kinds := []any{}
	if eligible {
		pairs := make([][2]string, 0, len(reg.FatalKinds))
		for k := range reg.FatalKinds {
			pairs = append(pairs, k)
		}
		sort.Slice(pairs, func(i, j int) bool {
			if pairs[i][0] != pairs[j][0] {
				return pairs[i][0] < pairs[j][0]
			}
			return pairs[i][1] < pairs[j][1]
		})
		for _, p := range pairs {
			kinds = append(kinds, []string{p[0], p[1]})
		}
	}
	var src any
	if reg.VerificationSource != nil {
		src = *reg.VerificationSource
	}
	var reason any
	if !eligible {
		reason = "exact verified engine evidence is not available"
	}
	return map[string]any{
		"status":                 map[bool]string{true: "verified", false: "unverified"}[eligible],
		"requested_game_version": requested,
		"registry_game_version":  reg.GameVersion,
		"verification_source":    src,
		"fatal_reference_kinds":  kinds,
		"reason":                 reason,
	}
}
