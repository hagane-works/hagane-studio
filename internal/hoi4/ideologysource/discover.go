package ideologysource

import (
	"bytes"
	"fmt"
	"sort"
	"strings"

	"haganeworks/internal/hoi4/paradox"
)

func layerKind(value string) string { return strings.ToLower(strings.TrimSpace(value)) }

func replacesIdeologies(paths []string) bool {
	for _, value := range paths {
		if normalizedDirectory(value) == IdeologyDirectory {
			return true
		}
	}
	return false
}

func validateLayers(layers []Layer) (Layer, *Layer, error) {
	var vanilla *Layer
	dependencies := []Layer{}
	orders := map[int]string{}
	for index := range layers {
		layer := layers[index]
		kind := layerKind(layer.SourceKind)
		if previous, exists := orders[layer.LoadOrder]; exists {
			return Layer{}, nil, findingsError(finding(CodeLayerAmbiguous, fmt.Sprintf("load order %d is shared by %s and %s", layer.LoadOrder, previous, kind)))
		}
		orders[layer.LoadOrder] = kind
		switch kind {
		case SourceVanilla:
			if vanilla != nil {
				return Layer{}, nil, findingsError(finding(CodeLayerAmbiguous, "more than one vanilla layer was supplied"))
			}
			copy := layer
			vanilla = &copy
		case SourceDependency:
			if strings.TrimSpace(layer.DependencyKey) == "" {
				return Layer{}, nil, findingsError(finding(CodeLayerAmbiguous, "dependency layer has no dependency_key"))
			}
			dependencies = append(dependencies, layer)
		default:
			return Layer{}, nil, findingsError(finding(CodeLayerAmbiguous, fmt.Sprintf("unsupported source_kind %q", layer.SourceKind)))
		}
	}
	if vanilla == nil {
		return Layer{}, nil, findingsError(finding(CodeLayerAmbiguous, "a vanilla layer is required"))
	}
	if len(dependencies) > 1 {
		return Layer{}, nil, findingsError(finding(CodeLayerAmbiguous, "more than one dependency layer makes ROOT ownership ambiguous"))
	}
	if len(dependencies) == 1 {
		if dependencies[0].LoadOrder <= vanilla.LoadOrder {
			return Layer{}, nil, findingsError(finding(CodeLayerAmbiguous, "dependency load_order must follow vanilla"))
		}
		dependency := dependencies[0]
		return *vanilla, &dependency, nil
	}
	return *vanilla, nil, nil
}

func discoverLayer(layer Layer) ([]discoveredFile, error) {
	files := []discoveredFile{}
	seenPaths := map[string]bool{}
	for _, source := range layer.Files {
		if !isIdeologyFile(source.Path) {
			continue
		}
		pathKey := strings.ToLower(normalizePath(source.Path))
		if seenPaths[pathKey] {
			return nil, findingsError(finding("ideology_source_file_ambiguous", fmt.Sprintf("layer %q contains duplicate path %q", layer.SourceKind, normalizePath(source.Path))))
		}
		seenPaths[pathKey] = true
		parseBytes := source.Bytes
		if bytes.HasPrefix(parseBytes, []byte{0xef, 0xbb, 0xbf}) {
			parseBytes = parseBytes[3:]
		}
		if _, recoveries, parseErr := paradox.Parse(string(parseBytes)); parseErr != nil || len(recoveries) > 0 {
			detail := "source is not a strict Paradox document"
			if parseErr != nil {
				detail = parseErr.Error()
			}
			return nil, findingsError(Finding{Code: "ideology_source_malformed", OwnerPath: normalizePath(source.Path), Detail: detail})
		}
		file, err := scanSource(source.Path, source.Bytes, layerKind(layer.SourceKind), strings.TrimSpace(layer.DependencyKey))
		if err != nil {
			return nil, findingsError(Finding{Code: "ideology_source_malformed", OwnerPath: normalizePath(source.Path), Detail: err.Error()})
		}
		files = append(files, file)
	}
	sort.Slice(files, func(i, j int) bool { return strings.ToLower(files[i].Path) < strings.ToLower(files[j].Path) })
	return files, nil
}

func rootsFromFiles(files []discoveredFile) []rootLocation {
	roots := []rootLocation{}
	for _, file := range files {
		roots = append(roots, file.Roots...)
	}
	return roots
}

func appendOwner(discovery *Discovery, root rootLocation, suppressed bool) {
	owner := root.Owner
	owner.SubtypeIDs = append([]string(nil), owner.SubtypeIDs...)
	sort.Strings(owner.SubtypeIDs)
	if suppressed {
		discovery.Suppressed = append(discovery.Suppressed, owner)
		return
	}
	discovery.Roots = append(discovery.Roots, owner)
	discovery.locations[keyForOwner(owner.RootID, owner.OwnerPath)] = root
}

func Discover(layers []Layer) (Discovery, error) {
	vanilla, dependency, err := validateLayers(layers)
	if err != nil {
		return Discovery{}, err
	}
	vanillaFiles, err := discoverLayer(vanilla)
	if err != nil {
		return Discovery{}, err
	}
	dependencyFiles := []discoveredFile{}
	if dependency != nil {
		dependencyFiles, err = discoverLayer(*dependency)
		if err != nil {
			return Discovery{}, err
		}
	}
	vanillaSuppressed := dependency != nil && replacesIdeologies(dependency.ReplacePaths)
	discovery := Discovery{locations: map[string]rootLocation{}, files: map[string]discoveredFile{}}
	if !vanillaSuppressed {
		for _, file := range vanillaFiles {
			discovery.files[strings.ToLower(file.Path)] = file
		}
	}
	for _, file := range dependencyFiles {
		pathKey := strings.ToLower(file.Path)
		if previous, exists := discovery.files[pathKey]; exists {
			return Discovery{}, findingsError(Finding{Code: "ideology_source_owner_path_ambiguous", OwnerPath: file.Path, Detail: fmt.Sprintf("effective layers %s and %s both own the same output path", previous.SourceKind, file.SourceKind)})
		}
		discovery.files[pathKey] = file
	}

	for _, root := range rootsFromFiles(vanillaFiles) {
		appendOwner(&discovery, root, vanillaSuppressed)
	}
	for _, root := range rootsFromFiles(dependencyFiles) {
		appendOwner(&discovery, root, false)
	}

	byID := map[string][]RootOwner{}
	for _, owner := range discovery.Roots {
		byID[owner.RootID] = append(byID[owner.RootID], owner)
	}
	findings := []Finding{}
	for rootID, owners := range byID {
		if len(owners) != 1 {
			paths := make([]string, len(owners))
			for i := range owners {
				paths[i] = owners[i].OwnerPath
			}
			sort.Strings(paths)
			findings = append(findings, Finding{Code: "ideology_source_owner_ambiguous", ParentRootID: rootID, Detail: fmt.Sprintf("ROOT has %d effective owners: %s", len(owners), strings.Join(paths, ", "))})
		}
	}
	if len(findings) > 0 {
		return Discovery{}, findingsError(findings...)
	}
	sort.Slice(discovery.Roots, func(i, j int) bool {
		if discovery.Roots[i].RootID != discovery.Roots[j].RootID {
			return discovery.Roots[i].RootID < discovery.Roots[j].RootID
		}
		return discovery.Roots[i].OwnerPath < discovery.Roots[j].OwnerPath
	})
	sort.Slice(discovery.Suppressed, func(i, j int) bool { return discovery.Suppressed[i].RootID < discovery.Suppressed[j].RootID })
	return discovery, nil
}

func ownerByRoot(discovery Discovery, rootID string) (RootOwner, bool) {
	for _, owner := range discovery.Roots {
		if owner.RootID == rootID {
			return owner, true
		}
	}
	return RootOwner{}, false
}

func suppressedByRoot(discovery Discovery, rootID string) bool {
	for _, owner := range discovery.Suppressed {
		if owner.RootID == rootID {
			return true
		}
	}
	return false
}

func Lock(discovery Discovery, subtypes []Subtype) (Manifest, error) {
	findings := validateSubtypeRequests(discovery, subtypes)
	if len(findings) > 0 {
		return Manifest{}, findingsError(findings...)
	}
	manifest := Manifest{SchemaVersion: SchemaVersion, Entries: make([]ManifestEntry, 0, len(subtypes))}
	for _, subtype := range subtypes {
		owner, _ := ownerByRoot(discovery, strings.TrimSpace(subtype.ParentRootID))
		manifest.Entries = append(manifest.Entries, ManifestEntry{
			SourceKind: owner.SourceKind, DependencyKey: owner.DependencyKey, OwnerPath: owner.OwnerPath,
			FileSHA256: owner.FileSHA256, RootSHA256: owner.RootSHA256,
			ScriptID: strings.TrimSpace(subtype.ScriptID), ParentRootID: strings.TrimSpace(subtype.ParentRootID),
		})
	}
	sortManifestEntries(manifest.Entries)
	return manifest, nil
}

func validateSubtypeRequests(discovery Discovery, subtypes []Subtype) []Finding {
	findings := []Finding{}
	seen := map[string]bool{}
	existing := map[string]string{}
	for _, owner := range discovery.Roots {
		existing[owner.RootID] = "ROOT"
		for _, id := range owner.SubtypeIDs {
			existing[id] = owner.RootID
		}
	}
	for _, subtype := range subtypes {
		id, parent := strings.TrimSpace(subtype.ScriptID), strings.TrimSpace(subtype.ParentRootID)
		entry := ManifestEntry{ScriptID: id, ParentRootID: parent}
		if !validID(id) {
			findings = append(findings, withEntry(finding("ideology_source_script_id_invalid", fmt.Sprintf("script_id %q is not a Paradox identifier", id)), entry))
		}
		if !validID(parent) {
			findings = append(findings, withEntry(finding("ideology_source_parent_id_invalid", fmt.Sprintf("parent_root_id %q is not a Paradox identifier", parent)), entry))
		}
		if seen[id] {
			findings = append(findings, withEntry(finding("ideology_source_script_id_duplicate", fmt.Sprintf("script_id %q is requested more than once", id)), entry))
		}
		seen[id] = true
		if current, exists := existing[id]; exists {
			findings = append(findings, withEntry(finding("ideology_source_script_id_conflict", fmt.Sprintf("script_id %q already belongs to %s", id, current)), entry))
		}
		owner, ok := ownerByRoot(discovery, parent)
		if !ok {
			code := "ideology_source_owner_missing"
			detail := fmt.Sprintf("ROOT %q has no effective owner", parent)
			if suppressedByRoot(discovery, parent) {
				code = "ideology_source_owner_replaced"
				detail = fmt.Sprintf("ROOT %q is hidden by dependency replace_path=%q", parent, IdeologyDirectory)
			}
			findings = append(findings, withEntry(finding(code, detail), entry))
			continue
		}
		if owner.TypesCount != 1 {
			findings = append(findings, withEntry(finding("ideology_source_types_ambiguous", fmt.Sprintf("ROOT %q must have exactly one direct types block; got %d", parent, owner.TypesCount)), entry))
		}
	}
	sortFindings(findings)
	return findings
}

func sortManifestEntries(entries []ManifestEntry) {
	sort.Slice(entries, func(i, j int) bool {
		if ownerKey(entries[i]) != ownerKey(entries[j]) {
			return ownerKey(entries[i]) < ownerKey(entries[j])
		}
		return entries[i].ScriptID < entries[j].ScriptID
	})
}
