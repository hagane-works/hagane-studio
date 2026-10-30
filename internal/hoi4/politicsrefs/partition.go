package politicsrefs

import (
	"fmt"
	"strings"
)

type ConflictReason string

const (
	ConflictDuplicateID ConflictReason = "duplicate_script_id"

	ConflictMissingRoot ConflictReason = "missing_root"

	ConflictUnknownRoot ConflictReason = "unknown_root"

	ConflictMalformed ConflictReason = "malformed_definition"
)

type IdeologyConflict struct {
	ScriptID string         `json:"script_id"`
	Kind     IdeologyKind   `json:"kind,omitempty"`
	RootID   string         `json:"root_id,omitempty"`
	Reason   ConflictReason `json:"reason"`
	Detail   string         `json:"detail"`

	UsageCount int `json:"usage_count"`
}

func PartitionValid(snapshot Snapshot) Snapshot {
	clean := Snapshot{
		ProjectID: snapshot.ProjectID,
		Conflicts: append([]IdeologyConflict(nil), snapshot.Conflicts...),
	}
	seenConflict := make(map[string]bool, len(clean.Conflicts))
	for _, existing := range clean.Conflicts {
		seenConflict[conflictKey(existing.ScriptID, existing.Reason)] = true
	}
	reject := func(ideology Ideology, reason ConflictReason, detail string) {
		key := conflictKey(ideology.ScriptID, reason)
		if seenConflict[key] {
			return
		}
		seenConflict[key] = true
		clean.Conflicts = append(clean.Conflicts, IdeologyConflict{
			ScriptID: ideology.ScriptID, Kind: ideology.Kind,
			RootID: ideology.RootID, Reason: reason, Detail: detail,
		})
	}

	kept := make([]Ideology, 0, len(snapshot.Ideologies))
	byID := make(map[string]Ideology, len(snapshot.Ideologies))
	roots := make(map[string]bool, len(snapshot.Ideologies))
	for _, raw := range snapshot.Ideologies {
		ideology := normalizeIdeology(raw)
		if ideology.ScriptID == "" {

			continue
		}
		if !validateKind(ideology.Kind) || !validateSource(ideology.Source) {
			reject(ideology, ConflictMalformed,
				fmt.Sprintf("invalid kind %q or source %q", ideology.Kind, ideology.Source))
			continue
		}
		if previous, exists := byID[ideology.ScriptID]; exists {

			reject(ideology, ConflictDuplicateID,
				fmt.Sprintf("%q already declared as %s", ideology.ScriptID, previous.Kind))
			continue
		}
		if ideology.Kind == KindRoot {
			if ideology.RootID != "" {
				reject(ideology, ConflictMalformed, "ROOT must not have root_id")
				continue
			}
			roots[ideology.ScriptID] = true
		} else if ideology.RootID == "" {
			reject(ideology, ConflictMissingRoot,
				fmt.Sprintf("subtype %q has no parent group", ideology.ScriptID))
			continue
		}
		byID[ideology.ScriptID] = ideology
		kept = append(kept, ideology)
	}

	survivors := make([]Ideology, 0, len(kept))
	for _, ideology := range kept {
		if ideology.Kind == KindSubtype && !roots[ideology.RootID] {
			reject(ideology, ConflictUnknownRoot,
				fmt.Sprintf("parent group %q does not exist", ideology.RootID))
			delete(byID, ideology.ScriptID)
			continue
		}
		survivors = append(survivors, ideology)
	}
	clean.Ideologies = survivors

	for _, ref := range snapshot.References {
		target := normalizeTarget(ref.Target)
		if !validateKind(target.Kind) || target.ScriptID == "" ||
			strings.TrimSpace(ref.Entity.Kind) == "" ||
			strings.TrimSpace(ref.Entity.ID) == "" ||
			strings.TrimSpace(ref.Entity.Path) == "" {
			continue
		}
		if ideology, ok := byID[target.ScriptID]; !ok || ideology.Kind != target.Kind {
			continue
		}
		clean.References = append(clean.References, ref)
	}

	for _, dependency := range snapshot.Dependencies {
		if strings.TrimSpace(dependency.CustomRootID) == "" ||
			strings.TrimSpace(dependency.ExpectedRoot) == "" ||
			strings.TrimSpace(dependency.ReferenceKey) == "" ||
			strings.TrimSpace(dependency.Entity.Kind) == "" ||
			strings.TrimSpace(dependency.Entity.ID) == "" ||
			strings.TrimSpace(dependency.Entity.Path) == "" {
			continue
		}
		if dependency.Disposition != DependencyRequired && dependency.Disposition != DependencyIntentional {
			continue
		}
		clean.Dependencies = append(clean.Dependencies, dependency)
	}

	return clean
}

func conflictKey(scriptID string, reason ConflictReason) string {
	return string(reason) + "\x00" + scriptID
}
