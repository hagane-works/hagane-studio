package politicsrefs

import (
	"errors"
	"fmt"
	"sort"
	"strings"
)

type IdeologyKind string

const (
	KindRoot    IdeologyKind = "root"
	KindSubtype IdeologyKind = "subtype"
)

type SourceOwnership string

const (
	SourceProjectManaged SourceOwnership = "project_managed"
	SourceVanilla        SourceOwnership = "vanilla"
	SourceImported       SourceOwnership = "imported"
	SourceBaseMod        SourceOwnership = "base_mod"
)

type Action string

const (
	ActionEdit     Action = "edit"
	ActionDelete   Action = "delete"
	ActionTakeover Action = "takeover"
)

type Target struct {
	Kind     IdeologyKind `json:"kind"`
	ScriptID string       `json:"script_id"`
}

type Ideology struct {
	Kind           IdeologyKind    `json:"kind"`
	ScriptID       string          `json:"script_id"`
	RootID         string          `json:"root_id,omitempty"`
	DisplayName    string          `json:"display_name,omitempty"`
	Source         SourceOwnership `json:"source"`
	RandomDisabled bool            `json:"random_disabled,omitempty"`
}

type EntityRef struct {
	Kind string `json:"kind"`
	ID   string `json:"id"`
	Path string `json:"path"`
}

type Reference struct {
	Target  Target    `json:"target"`
	Entity  EntityRef `json:"entity"`
	Context string    `json:"context,omitempty"`
}

type DependencyDisposition string

const (
	DependencyRequired    DependencyDisposition = "required_compatibility"
	DependencyIntentional DependencyDisposition = "intentional_design"
)

type RootDependency struct {
	CustomRootID string                `json:"custom_root_id"`
	ExpectedRoot string                `json:"expected_root_id"`
	ReferenceKey string                `json:"reference_key"`
	Disposition  DependencyDisposition `json:"disposition"`
	Entity       EntityRef             `json:"entity"`
}

type Snapshot struct {
	ProjectID    string           `json:"project_id"`
	Ideologies   []Ideology       `json:"ideologies"`
	References   []Reference      `json:"references,omitempty"`
	Dependencies []RootDependency `json:"dependencies,omitempty"`

	Conflicts []IdeologyConflict `json:"conflicts,omitempty"`
}

type OwnershipDecision struct {
	Allowed bool            `json:"allowed"`
	Action  Action          `json:"action"`
	Target  Target          `json:"target"`
	Source  SourceOwnership `json:"source"`
	Reason  string          `json:"reason,omitempty"`
}

var (
	ErrInvalidSnapshot   = errors.New("invalid politics reference snapshot")
	ErrTargetNotFound    = errors.New("politics target not found")
	ErrImmutableSource   = errors.New("source-owned ideology is immutable")
	ErrDuplicateScriptID = errors.New("ideology script id must be unique")
	ErrStableScriptID    = errors.New("persisted ideology script id is stable")
	ErrUnknownPreview    = errors.New("preview token is unknown or already used")
	ErrStalePreview      = errors.New("preview token is stale")
	ErrDeleteBlocked     = errors.New("delete blocked by references")
	ErrAutomaticRewrite  = errors.New("automatic political reference rewrite is forbidden")
)

func normalizeTarget(target Target) Target {
	target.ScriptID = strings.TrimSpace(target.ScriptID)
	return target
}

func normalizeIdeology(ideology Ideology) Ideology {
	ideology.ScriptID = strings.TrimSpace(ideology.ScriptID)
	ideology.RootID = strings.TrimSpace(ideology.RootID)
	ideology.DisplayName = strings.TrimSpace(ideology.DisplayName)
	return ideology
}

func validateKind(kind IdeologyKind) bool {
	return kind == KindRoot || kind == KindSubtype
}

func validateSource(source SourceOwnership) bool {
	switch source {
	case SourceProjectManaged, SourceVanilla, SourceImported, SourceBaseMod:
		return true
	default:
		return false
	}
}

func ValidateSnapshot(snapshot Snapshot) error {
	ids := make(map[string]Ideology, len(snapshot.Ideologies))
	roots := make(map[string]bool, len(snapshot.Ideologies))
	for _, raw := range snapshot.Ideologies {
		ideology := normalizeIdeology(raw)
		if !validateKind(ideology.Kind) || !validateSource(ideology.Source) || ideology.ScriptID == "" {
			return fmt.Errorf("%w: invalid ideology definition %#v", ErrInvalidSnapshot, raw)
		}
		if previous, exists := ids[ideology.ScriptID]; exists {
			return fmt.Errorf("%w: %q is used by %s and %s", ErrDuplicateScriptID,
				ideology.ScriptID, previous.Kind, ideology.Kind)
		}
		ids[ideology.ScriptID] = ideology
		if ideology.Kind == KindRoot {
			if ideology.RootID != "" {
				return fmt.Errorf("%w: ROOT %q must not have root_id", ErrInvalidSnapshot, ideology.ScriptID)
			}
			roots[ideology.ScriptID] = true
		} else if ideology.RootID == "" {
			return fmt.Errorf("%w: subtype %q has no ROOT", ErrInvalidSnapshot, ideology.ScriptID)
		}
	}
	for _, raw := range snapshot.Ideologies {
		ideology := normalizeIdeology(raw)
		if ideology.Kind == KindSubtype && !roots[ideology.RootID] {
			return fmt.Errorf("%w: subtype %q refers to missing ROOT %q",
				ErrInvalidSnapshot, ideology.ScriptID, ideology.RootID)
		}
	}
	for _, ref := range snapshot.References {
		target := normalizeTarget(ref.Target)
		if !validateKind(target.Kind) || target.ScriptID == "" || strings.TrimSpace(ref.Entity.Kind) == "" ||
			strings.TrimSpace(ref.Entity.ID) == "" || strings.TrimSpace(ref.Entity.Path) == "" {
			return fmt.Errorf("%w: invalid reference %#v", ErrInvalidSnapshot, ref)
		}
	}
	for _, dependency := range snapshot.Dependencies {
		if strings.TrimSpace(dependency.CustomRootID) == "" || strings.TrimSpace(dependency.ExpectedRoot) == "" ||
			strings.TrimSpace(dependency.ReferenceKey) == "" || strings.TrimSpace(dependency.Entity.Kind) == "" ||
			strings.TrimSpace(dependency.Entity.ID) == "" || strings.TrimSpace(dependency.Entity.Path) == "" {
			return fmt.Errorf("%w: invalid ROOT dependency %#v", ErrInvalidSnapshot, dependency)
		}
		if dependency.Disposition != DependencyRequired && dependency.Disposition != DependencyIntentional {
			return fmt.Errorf("%w: invalid dependency disposition %q", ErrInvalidSnapshot, dependency.Disposition)
		}
	}
	return nil
}

func findIdeology(snapshot Snapshot, target Target) (Ideology, bool) {
	target = normalizeTarget(target)
	for _, raw := range snapshot.Ideologies {
		ideology := normalizeIdeology(raw)
		if ideology.Kind == target.Kind && ideology.ScriptID == target.ScriptID {
			return ideology, true
		}
	}
	return Ideology{}, false
}

func OwnershipFor(ideology Ideology, action Action) OwnershipDecision {
	ideology = normalizeIdeology(ideology)
	decision := OwnershipDecision{
		Action: action,
		Target: Target{Kind: ideology.Kind, ScriptID: ideology.ScriptID},
		Source: ideology.Source,
	}
	if action == ActionTakeover {
		decision.Reason = "P0 does not take over ROOT or subtype definitions; base-Mod takeover is a later migration capability"
		return decision
	}
	if ideology.Source != SourceProjectManaged {
		decision.Reason = "vanilla, imported, and base-Mod definitions are source-owned and read-only"
		return decision
	}
	decision.Allowed = action == ActionEdit || action == ActionDelete
	if !decision.Allowed {
		decision.Reason = "unsupported ownership action"
	}
	return decision
}

func ValidateEdit(snapshot Snapshot, target Target, proposed Ideology) error {
	if err := ValidateSnapshot(snapshot); err != nil {
		return err
	}
	current, found := findIdeology(snapshot, target)
	if !found {
		return ErrTargetNotFound
	}
	if !OwnershipFor(current, ActionEdit).Allowed {
		return ErrImmutableSource
	}
	proposed = normalizeIdeology(proposed)
	if proposed.Kind != current.Kind || proposed.Source != SourceProjectManaged || proposed.ScriptID == "" {
		return fmt.Errorf("%w: invalid edit proposal", ErrInvalidSnapshot)
	}
	if proposed.ScriptID != current.ScriptID {
		return fmt.Errorf("%w: %q cannot be changed to %q after creation",
			ErrStableScriptID, current.ScriptID, proposed.ScriptID)
	}
	if current.Kind == KindRoot && proposed.RootID != "" {
		return fmt.Errorf("%w: ROOT cannot have a parent", ErrInvalidSnapshot)
	}
	if current.Kind == KindSubtype {
		if proposed.RootID == "" {
			return fmt.Errorf("%w: subtype must retain an explicit ROOT", ErrInvalidSnapshot)
		}
		root, found := findIdeology(snapshot, Target{Kind: KindRoot, ScriptID: proposed.RootID})
		if !found || root.Kind != KindRoot {
			return fmt.Errorf("%w: subtype parent ROOT %q is missing", ErrInvalidSnapshot, proposed.RootID)
		}
	}
	return nil
}

func entityLess(left, right EntityRef) bool {
	if left.Kind != right.Kind {
		return left.Kind < right.Kind
	}
	if left.ID != right.ID {
		return left.ID < right.ID
	}
	return left.Path < right.Path
}

func referenceLess(left, right Reference) bool {
	if left.Target.Kind != right.Target.Kind {
		return left.Target.Kind < right.Target.Kind
	}
	if left.Target.ScriptID != right.Target.ScriptID {
		return left.Target.ScriptID < right.Target.ScriptID
	}
	if entityLess(left.Entity, right.Entity) {
		return true
	}
	if entityLess(right.Entity, left.Entity) {
		return false
	}
	return left.Context < right.Context
}

func normalizeReference(ref Reference) Reference {
	ref.Target = normalizeTarget(ref.Target)
	ref.Entity.Kind = strings.TrimSpace(ref.Entity.Kind)
	ref.Entity.ID = strings.TrimSpace(ref.Entity.ID)
	ref.Entity.Path = strings.TrimSpace(ref.Entity.Path)
	ref.Context = strings.TrimSpace(ref.Context)
	return ref
}

func ListReferences(snapshot Snapshot) ([]Reference, error) {
	if err := ValidateSnapshot(snapshot); err != nil {
		return nil, err
	}
	seen := map[Reference]bool{}
	refs := make([]Reference, 0, len(snapshot.References))
	for _, raw := range snapshot.References {
		ref := normalizeReference(raw)
		if seen[ref] {
			continue
		}
		seen[ref] = true
		refs = append(refs, ref)
	}
	sort.Slice(refs, func(i, j int) bool { return referenceLess(refs[i], refs[j]) })
	return refs, nil
}
