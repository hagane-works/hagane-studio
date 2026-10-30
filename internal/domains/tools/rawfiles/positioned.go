package rawfiles

import (
	"strings"

	"haganeworks/internal/hoi4/clausewitz"
	"haganeworks/internal/hoi4/paradox"
)

func isReferenceTokenEdge(r rune) bool {
	switch {
	case r >= 'A' && r <= 'Z', r >= 'a' && r <= 'z', r >= '0' && r <= '9':
		return true
	case r == '_' || r == ':' || r == '.' || r == '@' || r == '-':
		return true
	}
	return false
}

var booleanBlockKeys = map[string]bool{
	"and": true, "or": true, "not": true, "nor": true, "nand": true,
}

type usedRange struct{ start, end int }

func CollectPositionedReferenceCandidates(text string, rootPath []string) []clausewitz.ReferenceCandidate {
	if strings.TrimSpace(text) == "" {
		return nil
	}
	block, recoveries, err := paradox.Parse(text)
	if err != nil || block == nil {
		return nil
	}
	for _, rec := range recoveries {
		if rec.Code == paradox.RecoveryUnclosedBlock {
			return nil
		}
	}

	analysis := clausewitz.AnalyzeAssignments(text, rootPath, nil)
	assignments := analysis.Assignments
	runes := []rune(text)
	used := map[usedRange]bool{}
	var candidates []clausewitz.ReferenceCandidate

	pathEqual := func(a, b []string) bool {
		if len(a) != len(b) {
			return false
		}
		for i := range a {
			if a[i] != b[i] {
				return false
			}
		}
		return true
	}

	assignmentFor := func(effectName string, path []string) []clausewitz.AssignmentIR {
		expected := append(append([]string{}, rootPath...), path...)
		var exact []clausewitz.AssignmentIR
		for _, a := range assignments {
			if a.Key == effectName && pathEqual(a.ParentPath, expected) {
				exact = append(exact, a)
			}
		}
		if len(exact) > 0 {
			return exact
		}
		var any []clausewitz.AssignmentIR
		for _, a := range assignments {
			if a.Key == effectName {
				any = append(any, a)
			}
		}
		return any
	}

	locateTarget := func(a clausewitz.AssignmentIR, targetID string) *clausewitz.SourceSpan {
		start, end := a.Span.StartOffset, a.Span.EndOffset
		if start < 0 || end > len(runes) || end < start {
			return nil
		}
		valueStart := start
		for i := start; i < end; i++ {
			if runes[i] == '=' {
				valueStart = i + 1
				break
			}
		}
		target := []rune(targetID)
		if len(target) == 0 {
			return nil
		}
		for i := valueStart; i+len(target) <= end; i++ {
			match := true
			for j, c := range target {
				if runes[i+j] != c {
					match = false
					break
				}
			}
			if !match {
				continue
			}
			if i > 0 && isReferenceTokenEdge(runes[i-1]) {
				continue
			}
			if i+len(target) < end && isReferenceTokenEdge(runes[i+len(target)]) {
				continue
			}
			key := usedRange{i, i + len(target)}
			if used[key] {
				continue
			}
			used[key] = true
			sp := clausewitz.Span(runes, key.start, key.end)
			return &sp
		}
		return nil
	}

	var walk func(entries []*paradox.Entry, path []string)
	walk = func(entries []*paradox.Entry, path []string) {
		for _, entry := range entries {
			name := ""
			if entry.Key != nil {
				name = *entry.Key
			}
			if specs, ok := activeCrossRefs[name]; ok {
				containers := assignmentFor(name, path)
				for _, spec := range specs {
					for _, targetID := range spec.extract(entry) {
						var located *clausewitz.SourceSpan
						var container *clausewitz.AssignmentIR
						for i := range containers {
							if sp := locateTarget(containers[i], targetID); sp != nil {
								located = sp
								container = &containers[i]
								break
							}
						}
						if located == nil || container == nil {
							continue
						}
						var ancestry []string
						for _, part := range container.ParentPath {
							if booleanBlockKeys[strings.ToLower(part)] {
								ancestry = append(ancestry, part)
							}
						}
						assignSpan := container.Span
						candidates = append(candidates, clausewitz.ReferenceCandidate{
							EffectName: name, TargetType: spec.targetType,
							TargetID: targetID, SourceSpan: *located,
							ParentPath:      append(append([]string{}, container.ParentPath...), name),
							AssignmentSpan:  &assignSpan,
							BooleanAncestry: ancestry,
						})
					}
				}
			}
			if entry.Block != nil {
				childKey := name
				if childKey == "" {
					childKey = "{anonymous}"
				}
				walk(entry.Block.Entries, append(append([]string{}, path...), childKey))
			}
		}
	}
	walk(block.Entries, nil)
	return candidates
}
