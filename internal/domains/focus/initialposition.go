package focus

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

type initialPositionAnalysis struct {
	Value    map[string]any
	Findings []map[string]any
}

func (a initialPositionAnalysis) blockers() []map[string]any {
	return a.Findings
}

func focusSourceSpan(text string) map[string]any {
	endLine := 1
	lastStart := 0
	for i, r := range text {
		if r == '\n' {
			endLine++
			lastStart = i + 1
		}
	}
	endColumn := len([]rune(text[lastStart:])) + 1
	return map[string]any{
		"start_offset": 0,
		"end_offset":   len([]rune(text)),
		"start_line":   1,
		"start_column": 1,
		"end_line":     endLine,
		"end_column":   endColumn,
	}
}

func initialPositionFinding(code string, raw any, rawText, sourceName, reason string) map[string]any {
	var file any
	if sourceName != "" {
		file = sourceName
	}
	return map[string]any{
		"code":     code,
		"severity": "blocker",
		"source": map[string]any{
			"file":       file,
			"field_path": []string{"focus_tree", "initial_show_position"},
			"span":       focusSourceSpan(rawText),
		},
		"evidence":     map[string]any{"reason": reason, "raw": raw},
		"repair_route": "focus.initial_position.choose_kind",
	}
}

func initialPositionInvalid(raw any, rawText, sourceName, code, reason string) initialPositionAnalysis {
	if code == "" {
		code = "focus.initial_position.invalid_shape"
	}
	return initialPositionAnalysis{
		Value: map[string]any{"kind": "invalid", "raw": raw, "raw_text": rawText},
		Findings: []map[string]any{
			initialPositionFinding(code, raw, rawText, sourceName, reason),
		},
	}
}

func isNumericLegacy(raw string) bool {
	s := strings.TrimSpace(raw)
	if s == "" {
		return false
	}
	if s[0] == '+' || s[0] == '-' {
		s = s[1:]
	}
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func coerceUnionInt(v any) (int, bool) {
	switch t := v.(type) {
	case float64:
		if t == float64(int(t)) {
			return int(t), true
		}
	case string:
		if n, err := strconv.Atoi(strings.TrimSpace(t)); err == nil {
			return n, true
		}
	case int:
		return t, true
	case int64:
		return int(t), true
	case json.Number:
		if n, err := t.Int64(); err == nil {
			return int(n), true
		}
		if f, err := t.Float64(); err == nil && f == float64(int(f)) {
			return int(f), true
		}
	}
	return 0, false
}

func ValidateInitialShowPositionUnion(raw map[string]any) (map[string]any, error) {
	kind, _ := raw["kind"].(string)
	switch kind {
	case "focus":
		focusID, ok := raw["focus_id"].(string)
		if !ok {
			return nil, fmt.Errorf("initial_show_position.focus: focus_id must be a string")
		}
		return map[string]any{"kind": "focus", "focus_id": focusID}, nil
	case "coordinates":
		x, okX := coerceUnionInt(raw["x"])
		y, okY := coerceUnionInt(raw["y"])
		if raw["x"] == nil || raw["y"] == nil || !okX || !okY {
			return nil, fmt.Errorf("initial_show_position.coordinates: x and y must be integers")
		}
		offsets := []any{}
		if rawOffsets, present := raw["offsets"]; present && rawOffsets != nil {
			list, ok := rawOffsets.([]any)
			if !ok {
				return nil, fmt.Errorf("initial_show_position.coordinates: offsets must be a list")
			}
			for _, item := range list {
				m, ok := item.(map[string]any)
				if !ok {
					return nil, fmt.Errorf("initial_show_position.coordinates: each offset must be an object")
				}
				ox, oy := 0, 0
				if v, present := m["x"]; present && v != nil {
					if ox, ok = coerceUnionInt(v); !ok {
						return nil, fmt.Errorf("initial_show_position.offset: x must be an integer")
					}
				}
				if v, present := m["y"]; present && v != nil {
					if oy, ok = coerceUnionInt(v); !ok {
						return nil, fmt.Errorf("initial_show_position.offset: y must be an integer")
					}
				}
				trigger := ""
				if v, present := m["trigger"]; present && v != nil {
					if trigger, ok = v.(string); !ok {
						return nil, fmt.Errorf("initial_show_position.offset: trigger must be a string")
					}
				}
				offsets = append(offsets, map[string]any{"x": ox, "y": oy, "trigger": trigger})
			}
		}
		return map[string]any{"kind": "coordinates", "x": x, "y": y, "offsets": offsets}, nil
	case "invalid":
		rawValue, present := raw["raw"]
		if !present {
			return nil, fmt.Errorf("initial_show_position.invalid: raw is required")
		}
		rawText := ""
		if v, present := raw["raw_text"]; present && v != nil {
			s, ok := v.(string)
			if !ok {
				return nil, fmt.Errorf("initial_show_position.invalid: raw_text must be a string")
			}
			rawText = s
		}
		return map[string]any{"kind": "invalid", "raw": rawValue, "raw_text": rawText}, nil
	case "legacy_ambiguous":
		rawValue, ok := raw["raw"].(string)
		if !ok {
			return nil, fmt.Errorf("initial_show_position.legacy_ambiguous: raw must be a string")
		}
		return map[string]any{"kind": "legacy_ambiguous", "raw": rawValue}, nil
	default:
		return nil, fmt.Errorf(
			"initial_show_position: kind must be one of 'focus', 'coordinates', 'invalid', 'legacy_ambiguous'")
	}
}

func normalizeInitialShowPosition(value map[string]any, legacyRaw, rawText, sourceName string) initialPositionAnalysis {
	if value == nil {
		legacyText := strings.TrimSpace(legacyRaw)
		if legacyText == "" {
			return initialPositionAnalysis{}
		}
		if isNumericLegacy(legacyText) {
			effectiveRawText := rawText
			if effectiveRawText == "" {
				effectiveRawText = legacyText
			}
			return initialPositionAnalysis{
				Value: map[string]any{"kind": "legacy_ambiguous", "raw": legacyText},
				Findings: []map[string]any{
					initialPositionFinding(
						"focus.initial_position.legacy_ambiguous",
						legacyText, effectiveRawText, sourceName,
						"numeric legacy text could have been a focus id or an x "+
							"coordinate; migration cannot reconstruct the lost kind",
					),
				},
			}
		}
		value = map[string]any{"kind": "focus", "focus_id": legacyText}
	}

	kind, _ := value["kind"].(string)
	switch kind {
	case "legacy_ambiguous":
		raw := fmt.Sprintf("%v", value["raw"])
		if value["raw"] == nil {
			raw = ""
		}
		effectiveRawText := rawText
		if effectiveRawText == "" {
			effectiveRawText = raw
		}
		return initialPositionAnalysis{
			Value: map[string]any{"kind": "legacy_ambiguous", "raw": raw},
			Findings: []map[string]any{
				initialPositionFinding(
					"focus.initial_position.legacy_ambiguous",
					raw, effectiveRawText, sourceName,
					"legacy kind remains ambiguous and cannot be emitted safely",
				),
			},
		}
	case "invalid":
		rawValue, present := value["raw"]
		if !present {
			rawValue = value
		}
		effectiveRawText := rawText
		if v, present := value["raw_text"]; present {
			effectiveRawText = fmt.Sprintf("%v", v)
		}
		return initialPositionInvalid(rawValue, effectiveRawText, sourceName, "",
			"previous import preserved an illegal or unknown shape")
	case "focus":
		focusID, _ := value["focus_id"].(string)
		if strings.TrimSpace(focusID) == "" {
			return initialPositionInvalid(value, rawText, sourceName, "",
				"focus variant requires a non-empty focus_id")
		}
		return initialPositionAnalysis{
			Value: map[string]any{"kind": "focus", "focus_id": strings.TrimSpace(focusID)},
		}
	case "coordinates":
		offsets, _ := value["offsets"].([]any)
		normalized := []any{}
		for _, item := range offsets {
			m, _ := item.(map[string]any)
			trigger, _ := m["trigger"].(string)
			if strings.TrimSpace(trigger) == "" {
				return initialPositionInvalid(value, rawText, sourceName, "",
					"conditional offset requires a non-empty trigger block")
			}
			normalized = append(normalized, map[string]any{
				"x": m["x"], "y": m["y"], "trigger": trigger,
			})
		}
		return initialPositionAnalysis{
			Value: map[string]any{
				"kind": "coordinates", "x": value["x"], "y": value["y"],
				"offsets": normalized,
			},
		}
	}
	return initialPositionInvalid(value, rawText, sourceName,
		"focus.initial_position.unknown_kind",
		fmt.Sprintf("unknown initial-position kind: %s", pyRepr(kind)))
}

func materializeInitialPosition(value map[string]any, legacyRaw, source string) (map[string]any, error) {
	analysis := normalizeInitialShowPosition(value, legacyRaw, "", source)
	if blockers := analysis.blockers(); len(blockers) > 0 {
		return nil, DetailError{Status: 422, Detail: map[string]any{
			"error":    "focus_initial_position_blocked",
			"findings": blockers,
		}}
	}
	if analysis.Value == nil {
		return map[string]any{}, nil
	}
	return analysis.Value, nil
}
