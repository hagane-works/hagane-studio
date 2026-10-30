package ideologysource

import (
	"fmt"
	"sort"
	"strings"
)

type prepared struct {
	baseLayers []Layer
	discovery  Discovery
	regions    map[string]int
}

func prepare(layers []Layer) (prepared, error) {
	baseLayers, regions, err := stripLayers(layers)
	if err != nil {
		return prepared{}, err
	}
	discovery, err := Discover(baseLayers)
	if err != nil {
		return prepared{}, err
	}
	return prepared{baseLayers: baseLayers, discovery: discovery, regions: regions}, nil
}

func Relock(layers []Layer, subtypes []Subtype) (Manifest, error) {
	state, err := prepare(layers)
	if err != nil {
		return Manifest{}, err
	}
	return Lock(state.discovery, subtypes)
}

func coreFindings(discovery Discovery, manifest Manifest) []Finding {
	findings := []Finding{}
	if manifest.SchemaVersion != SchemaVersion {
		findings = append(findings, finding("ideology_source_manifest_version", fmt.Sprintf("schema_version=%d, want %d", manifest.SchemaVersion, SchemaVersion)))
	}
	seenIDs := map[string]bool{}
	existing := map[string]string{}
	for _, owner := range discovery.Roots {
		existing[owner.RootID] = "ROOT"
		for _, id := range owner.SubtypeIDs {
			existing[id] = owner.RootID
		}
	}
	for _, raw := range manifest.Entries {
		entry := normalizedEntry(raw)
		if !validID(entry.ScriptID) {
			findings = append(findings, withEntry(finding("ideology_source_script_id_invalid", fmt.Sprintf("script_id %q is not a Paradox identifier", entry.ScriptID)), entry))
		}
		if !validID(entry.ParentRootID) {
			findings = append(findings, withEntry(finding("ideology_source_parent_id_invalid", fmt.Sprintf("parent_root_id %q is not a Paradox identifier", entry.ParentRootID)), entry))
		}
		if seenIDs[entry.ScriptID] {
			findings = append(findings, withEntry(finding("ideology_source_script_id_duplicate", fmt.Sprintf("script_id %q appears more than once in manifest", entry.ScriptID)), entry))
		}
		seenIDs[entry.ScriptID] = true
		if current, exists := existing[entry.ScriptID]; exists {
			findings = append(findings, withEntry(finding("ideology_source_script_id_conflict", fmt.Sprintf("script_id %q already belongs to %s", entry.ScriptID, current)), entry))
		}
		owner, ok := ownerByRoot(discovery, entry.ParentRootID)
		if !ok {
			code, detail := "ideology_source_owner_missing", fmt.Sprintf("ROOT %q has no effective owner", entry.ParentRootID)
			if suppressedByRoot(discovery, entry.ParentRootID) {
				code, detail = "ideology_source_owner_replaced", fmt.Sprintf("ROOT %q is hidden by dependency replace_path=%q", entry.ParentRootID, IdeologyDirectory)
			}
			findings = append(findings, withEntry(finding(code, detail), entry))
			continue
		}
		if owner.TypesCount != 1 {
			findings = append(findings, withEntry(finding("ideology_source_types_ambiguous", fmt.Sprintf("ROOT %q must have exactly one direct types block; got %d", entry.ParentRootID, owner.TypesCount)), entry))
		}
		if entry.SourceKind != owner.SourceKind || entry.DependencyKey != owner.DependencyKey || !strings.EqualFold(entry.OwnerPath, owner.OwnerPath) {
			findings = append(findings, withEntry(finding("ideology_source_owner_drift", fmt.Sprintf("ROOT %q owner changed from %s:%s:%s to %s:%s:%s", entry.ParentRootID, entry.SourceKind, entry.DependencyKey, entry.OwnerPath, owner.SourceKind, owner.DependencyKey, owner.OwnerPath)), entry))
		}
		if entry.FileSHA256 != owner.FileSHA256 {
			findings = append(findings, withEntry(finding("ideology_source_file_drift", fmt.Sprintf("owner file sha256 changed: locked=%s current=%s", entry.FileSHA256, owner.FileSHA256)), entry))
		}
		if entry.RootSHA256 != owner.RootSHA256 {
			findings = append(findings, withEntry(finding("ideology_source_root_drift", fmt.Sprintf("ROOT sha256 changed: locked=%s current=%s", entry.RootSHA256, owner.RootSHA256)), entry))
		}
	}
	sortFindings(findings)
	return findings
}

func normalizedEntry(entry ManifestEntry) ManifestEntry {
	entry.SourceKind = layerKind(entry.SourceKind)
	entry.DependencyKey = strings.TrimSpace(entry.DependencyKey)
	entry.OwnerPath = normalizePath(entry.OwnerPath)
	entry.FileSHA256 = strings.ToLower(strings.TrimSpace(entry.FileSHA256))
	entry.RootSHA256 = strings.ToLower(strings.TrimSpace(entry.RootSHA256))
	entry.ScriptID = strings.TrimSpace(entry.ScriptID)
	entry.ParentRootID = strings.TrimSpace(entry.ParentRootID)
	return entry
}

func subtypeSetFindings(manifest Manifest, subtypes []Subtype) []Finding {
	findings := []Finding{}
	manifestByID := map[string]ManifestEntry{}
	for _, raw := range manifest.Entries {
		entry := normalizedEntry(raw)
		manifestByID[entry.ScriptID] = entry
	}
	seen := map[string]bool{}
	for _, subtype := range subtypes {
		id, parent := strings.TrimSpace(subtype.ScriptID), strings.TrimSpace(subtype.ParentRootID)
		entry, exists := manifestByID[id]
		if seen[id] {
			findings = append(findings, Finding{Code: "ideology_source_script_id_duplicate", ScriptID: id, ParentRootID: parent, Detail: fmt.Sprintf("script_id %q is supplied more than once", id)})
		}
		seen[id] = true
		if !exists {
			findings = append(findings, Finding{Code: "ideology_source_manifest_entry_missing", ScriptID: id, ParentRootID: parent, Detail: "subtype has no locked manifest entry"})
			continue
		}
		if entry.ParentRootID != parent {
			findings = append(findings, withEntry(finding("ideology_source_parent_drift", fmt.Sprintf("subtype parent=%q, locked parent=%q", parent, entry.ParentRootID)), entry))
		}
	}
	for _, raw := range manifest.Entries {
		entry := normalizedEntry(raw)
		if !seen[entry.ScriptID] {
			findings = append(findings, withEntry(finding("ideology_source_subtype_missing", "locked manifest entry has no subtype payload"), entry))
		}
	}
	sortFindings(findings)
	return findings
}

func Validate(layers []Layer, manifest Manifest, subtypes []Subtype) []Finding {
	state, err := prepare(layers)
	if err != nil {
		if typed, ok := err.(FindingsError); ok {
			return typed.Findings
		}
		return []Finding{finding("ideology_source_discovery_failed", err.Error())}
	}
	findings := coreFindings(state.discovery, manifest)
	findings = append(findings, subtypeSetFindings(manifest, subtypes)...)
	sortFindings(findings)
	return findings
}

func CertifiedIDs(layers []Layer, manifest Manifest) ([]string, error) {
	state, err := prepare(layers)
	if err != nil {
		return nil, err
	}
	if findings := coreFindings(state.discovery, manifest); len(findings) > 0 {
		return nil, findingsError(findings...)
	}
	ids := make([]string, 0, len(manifest.Entries))
	for _, entry := range manifest.Entries {
		ids = append(ids, strings.TrimSpace(entry.ScriptID))
	}
	sort.Strings(ids)
	return ids, nil
}
