package politicsrefs

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
)

type DeletePreview struct {
	Token             string            `json:"token"`
	SnapshotHash      string            `json:"snapshot_hash"`
	Target            Target            `json:"target"`
	Allowed           bool              `json:"allowed"`
	Ownership         OwnershipDecision `json:"ownership"`
	Impacts           []Reference       `json:"impacts"`
	AutomaticRewrites bool              `json:"automatic_rewrites"`
}

type DeleteAuthorization struct {
	Authorized        bool        `json:"authorized"`
	SnapshotHash      string      `json:"snapshot_hash"`
	Target            Target      `json:"target"`
	Impacts           []Reference `json:"impacts"`
	AutomaticRewrites bool        `json:"automatic_rewrites"`
}

type previewRecord struct {
	SnapshotHash string
	Target       Target
}

type Manager struct {
	mu     sync.Mutex
	tokens map[string]previewRecord
}

func NewManager() *Manager {
	return &Manager{tokens: map[string]previewRecord{}}
}

func canonicalSnapshot(snapshot Snapshot) (Snapshot, error) {
	if err := ValidateSnapshot(snapshot); err != nil {
		return Snapshot{}, err
	}
	canonical := Snapshot{ProjectID: strings.TrimSpace(snapshot.ProjectID)}
	canonical.Ideologies = make([]Ideology, 0, len(snapshot.Ideologies))
	for _, ideology := range snapshot.Ideologies {
		canonical.Ideologies = append(canonical.Ideologies, normalizeIdeology(ideology))
	}
	sort.Slice(canonical.Ideologies, func(i, j int) bool {
		if canonical.Ideologies[i].Kind != canonical.Ideologies[j].Kind {
			return canonical.Ideologies[i].Kind < canonical.Ideologies[j].Kind
		}
		return canonical.Ideologies[i].ScriptID < canonical.Ideologies[j].ScriptID
	})
	refs, err := ListReferences(snapshot)
	if err != nil {
		return Snapshot{}, err
	}
	canonical.References = refs
	dependencySeen := map[RootDependency]bool{}
	for _, raw := range snapshot.Dependencies {
		dependency := normalizeDependency(raw)
		if dependencySeen[dependency] {
			continue
		}
		dependencySeen[dependency] = true
		canonical.Dependencies = append(canonical.Dependencies, dependency)
	}
	sort.Slice(canonical.Dependencies, func(i, j int) bool {
		return dependencyLess(canonical.Dependencies[i], canonical.Dependencies[j])
	})
	return canonical, nil
}

func SnapshotHash(snapshot Snapshot) (string, error) {
	canonical, err := canonicalSnapshot(snapshot)
	if err != nil {
		return "", err
	}
	raw, err := json.Marshal(canonical)
	if err != nil {
		return "", fmt.Errorf("%w: canonical JSON: %v", ErrInvalidSnapshot, err)
	}
	digest := sha256.Sum256(raw)
	return hex.EncodeToString(digest[:]), nil
}

func aggregateReferences(snapshot Snapshot, target Target) ([]Reference, error) {
	target = normalizeTarget(target)
	if _, found := findIdeology(snapshot, target); !found {
		return nil, ErrTargetNotFound
	}
	all, err := ListReferences(snapshot)
	if err != nil {
		return nil, err
	}
	childIDs := map[string]bool{}
	if target.Kind == KindRoot {
		for _, ideology := range snapshot.Ideologies {
			ideology = normalizeIdeology(ideology)
			if ideology.Kind != KindSubtype || ideology.RootID != target.ScriptID {
				continue
			}
			childIDs[ideology.ScriptID] = true
			all = append(all, Reference{
				Target: target,
				Entity: EntityRef{
					Kind: "ideology_subtype",
					ID:   ideology.ScriptID,
					Path: "ideology_registry.sub_ideologies[" + ideology.ScriptID + "].group",
				},
				Context: "child subtype requires this ROOT",
			})
		}
	}
	seen := map[Reference]bool{}
	impacts := make([]Reference, 0)
	for _, raw := range all {
		ref := normalizeReference(raw)
		direct := ref.Target == target
		child := target.Kind == KindRoot && ref.Target.Kind == KindSubtype && childIDs[ref.Target.ScriptID]
		if (!direct && !child) || seen[ref] {
			continue
		}
		seen[ref] = true
		impacts = append(impacts, ref)
	}
	sort.Slice(impacts, func(i, j int) bool { return referenceLess(impacts[i], impacts[j]) })
	return impacts, nil
}

func AggregateReferences(snapshot Snapshot, target Target) ([]Reference, error) {
	if err := ValidateSnapshot(snapshot); err != nil {
		return nil, err
	}
	return aggregateReferences(snapshot, target)
}

func buildDeletePreview(snapshot Snapshot, target Target) (DeletePreview, error) {
	if err := ValidateSnapshot(snapshot); err != nil {
		return DeletePreview{}, err
	}
	target = normalizeTarget(target)
	ideology, found := findIdeology(snapshot, target)
	if !found {
		return DeletePreview{}, ErrTargetNotFound
	}
	ownership := OwnershipFor(ideology, ActionDelete)
	if !ownership.Allowed {
		return DeletePreview{Target: target, Ownership: ownership}, ErrImmutableSource
	}
	hash, err := SnapshotHash(snapshot)
	if err != nil {
		return DeletePreview{}, err
	}
	impacts, err := aggregateReferences(snapshot, target)
	if err != nil {
		return DeletePreview{}, err
	}
	return DeletePreview{
		SnapshotHash: hash,
		Target:       target,
		Allowed:      len(impacts) == 0,
		Ownership:    ownership,
		Impacts:      impacts,

		AutomaticRewrites: false,
	}, nil
}

func newOpaqueToken() (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}

func (manager *Manager) PreviewDelete(snapshot Snapshot, target Target) (DeletePreview, error) {
	preview, err := buildDeletePreview(snapshot, target)
	if err != nil {
		return preview, err
	}
	token, err := newOpaqueToken()
	if err != nil {
		return DeletePreview{}, fmt.Errorf("create preview token: %w", err)
	}
	manager.mu.Lock()
	manager.tokens[token] = previewRecord{SnapshotHash: preview.SnapshotHash, Target: preview.Target}
	manager.mu.Unlock()
	preview.Token = token
	return preview, nil
}

func (manager *Manager) ApplyDelete(snapshot Snapshot, token string) (DeleteAuthorization, error) {
	manager.mu.Lock()
	record, found := manager.tokens[token]
	if found {
		delete(manager.tokens, token)
	}
	manager.mu.Unlock()
	if !found {
		return DeleteAuthorization{}, ErrUnknownPreview
	}
	hash, err := SnapshotHash(snapshot)
	if err != nil {
		return DeleteAuthorization{}, err
	}
	if hash != record.SnapshotHash {
		return DeleteAuthorization{SnapshotHash: hash, Target: record.Target}, ErrStalePreview
	}
	preview, err := buildDeletePreview(snapshot, record.Target)
	if err != nil {
		return DeleteAuthorization{}, err
	}
	authorization := DeleteAuthorization{
		SnapshotHash:      preview.SnapshotHash,
		Target:            preview.Target,
		Impacts:           preview.Impacts,
		AutomaticRewrites: false,
	}
	if !preview.Allowed {
		return authorization, ErrDeleteBlocked
	}
	authorization.Authorized = true
	return authorization, nil
}

func RewriteReferences(_ Snapshot, _ Target, _ []Reference) error {
	return ErrAutomaticRewrite
}
