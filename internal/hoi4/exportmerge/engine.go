package exportmerge

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"haganeworks/internal/hoi4/exportengine"
)

var diffableEntities = []struct {
	Kind  string
	Table string
	ID    string
}{
	{Kind: "character", Table: "characters", ID: "tag"},
	{Kind: "event", Table: "events", ID: "event_id"},
	{Kind: "decision", Table: "decisions", ID: "decision_id"},
	{Kind: "focus_node", Table: "focus_nodes", ID: "focus_id"},
	{Kind: "idea", Table: "ideas", ID: "idea_id"},
}

type PreviewTotals struct {
	NoChange           int `json:"no_change"`
	CleanMerge         int `json:"clean_merge"`
	AutoMergeable      int `json:"auto_mergeable"`
	Conflict           int `json:"conflict"`
	TotalConflictPaths int `json:"total_conflict_paths"`
}

type Preview struct {
	Reports  map[string][]ConflictReport `json:"reports"`
	Totals   PreviewTotals               `json:"totals"`
	Warnings []string                    `json:"warnings"`
}

func BuildPreview(snapshot exportengine.Snapshot, target *TargetMod) (Preview, error) {
	if target == nil {
		target = NewTargetMod()
	}
	out := Preview{Reports: make(map[string][]ConflictReport, len(diffableEntities)), Warnings: append([]string(nil), target.Warnings...)}
	for _, spec := range diffableEntities {
		reports := make([]ConflictReport, 0)
		for _, raw := range snapshot.Tables[spec.Table] {
			var row map[string]any
			if err := decodeJSON(raw, &row); err != nil {
				return Preview{}, fmt.Errorf("decode %s snapshot row: %w", spec.Table, err)
			}
			id := entityID(row, spec.ID)
			rawTarget := target.ForType(spec.Kind)[id]
			baseline, _ := SourceSnapshot(row)
			targetValue := NormalizeTarget(baseline, rawTarget)
			report, err := MergeEntityDiff(spec.Kind, id, row, targetValue)
			if err != nil {
				return Preview{}, err
			}
			switch report.Status {
			case "no_change":
				out.Totals.NoChange++
				continue
			case "clean_merge":
				out.Totals.CleanMerge++
			case "auto_mergeable":
				out.Totals.AutoMergeable++
			case "conflict":
				out.Totals.Conflict++
				out.Totals.TotalConflictPaths += len(report.Conflicts)
			default:
				return Preview{}, fmt.Errorf("unexpected merge status %q", report.Status)
			}
			reports = append(reports, report)
		}
		out.Reports[spec.Kind] = reports
	}
	return out, nil
}

func ComputeFinalOps(snapshot exportengine.Snapshot, target *TargetMod, payload ResolutionRequest) (map[string][]DiffOp, []string, error) {
	if err := ValidateResolutionRequest(payload); err != nil {
		return nil, nil, err
	}
	if target == nil {
		target = NewTargetMod()
	}
	final := map[string][]DiffOp{}
	for _, spec := range diffableEntities {
		for _, raw := range snapshot.Tables[spec.Table] {
			var row map[string]any
			if err := decodeJSON(raw, &row); err != nil {
				return nil, nil, fmt.Errorf("decode %s snapshot row: %w", spec.Table, err)
			}
			id := entityID(row, spec.ID)
			rawTarget := target.ForType(spec.Kind)[id]
			baseline, _ := SourceSnapshot(row)
			report, err := MergeEntityDiff(spec.Kind, id, row, NormalizeTarget(baseline, rawTarget))
			if err != nil {
				return nil, nil, err
			}
			if report.Status == "no_change" {
				continue
			}
			prefix := spec.Kind + "/" + id + "/"
			resolutions := map[string]string{}
			manual := map[string]any{}
			for key, choice := range payload.Resolutions {
				if strings.HasPrefix(key, prefix) {
					resolutions[strings.TrimPrefix(key, prefix)] = choice
				}
			}
			for key, value := range payload.ManualValues {
				if strings.HasPrefix(key, prefix) {
					manual[strings.TrimPrefix(key, prefix)] = value
				}
			}
			ops, err := ApplyResolutions(report, resolutions, manual)
			if err != nil {
				return nil, nil, err
			}
			if len(ops) > 0 {
				final[spec.Kind+"/"+id] = ops
			}
		}
	}
	return final, append([]string(nil), target.Warnings...), nil
}

var resolutionKeyRE = regexp.MustCompile(`^(character|event|decision|focus_node|idea)/[^/]+/[A-Za-z0-9_.]+$`)

func ValidateResolutionRequest(payload ResolutionRequest) error {
	for key, choice := range payload.Resolutions {
		if !resolutionKeyRE.MatchString(key) {
			return fmt.Errorf("resolution key malformed: %q", key)
		}
		switch choice {
		case "keep_user", "keep_target":
		case "manual":
			if _, exists := payload.ManualValues[key]; !exists {
				return fmt.Errorf("resolution %q chose 'manual' but no value in manual_values", key)
			}
		default:
			return fmt.Errorf("resolution %q has unsupported choice %q", key, choice)
		}
	}
	return nil
}

func entityID(row map[string]any, key string) string {
	if value, ok := row[key].(string); ok {
		return value
	}
	return fmt.Sprint(row[key])
}

func sortedFinalKeys(final map[string][]DiffOp) []string {
	keys := make([]string, 0, len(final))
	for key := range final {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
