package guirender

import (
	"path"
	"sort"
	"strings"
	"sync"
)

type lowerFileMask struct {
	exact   map[string]bool
	folders map[string]bool
}

func (idx *Index) WithoutFiles(names []string) (*Index, error) {
	mask := lowerFileMask{exact: normalisedMaskNames(names)}
	if len(mask.exact) == 0 {
		return idx, nil
	}
	return idx.withoutLowerFilesCached(mask)
}

func (idx *Index) WithoutFolders(folders []string) (*Index, error) {
	mask := lowerFileMask{folders: normalisedMaskNames(folders)}
	if len(mask.folders) == 0 {
		return idx, nil
	}
	return idx.withoutLowerFilesCached(mask)
}

func normalisedMaskNames(values []string) map[string]bool {
	out := map[string]bool{}
	for _, value := range values {
		clean := strings.ToLower(strings.Trim(strings.ReplaceAll(strings.TrimSpace(value), "\\", "/"), "/"))
		if clean != "" {
			out[clean] = true
		}
	}
	return out
}

func (idx *Index) withoutLowerFiles(mask lowerFileMask) (*Index, error) {
	if idx == nil {
		return nil, nil
	}
	below, err := idx.fallback.withoutLowerFiles(mask)
	if err != nil {
		return nil, err
	}
	kept := make(map[string][]byte, len(idx.files))
	filesChanged := false
	for name, body := range idx.files {
		if mask.removes(name) {
			filesChanged = true
			continue
		}
		kept[name] = body
	}
	external, externalChanged := externalWithoutMask(idx.external, mask)
	projectFileSources, projectFileSourcesChanged := projectFileSourcesWithoutMask(idx.projectFileSources, mask)
	if !filesChanged {
		if below == idx.fallback && !externalChanged && !projectFileSourcesChanged {
			return idx, nil
		}
		copy := *idx
		copy.fallback = below
		if externalChanged {
			copy.external = external
			copy.externalLoose = buildLooseTextureIndex(external)
		}
		if projectFileSourcesChanged {
			copy.projectFileSources = projectFileSources
		}
		return &copy, nil
	}
	rebuilt, partial, err := idx.withoutOwnFiles(mask)
	if !partial && err == nil {
		rebuilt, err = newIndex(kept, idx.source)
	}
	if err != nil {
		return nil, err
	}
	rebuilt.fallback = below
	rebuilt.external = external
	rebuilt.externalLoose = buildLooseTextureIndex(external)
	rebuilt.version = idx.version
	rebuilt.baseURL = idx.baseURL
	rebuilt.textureURL = idx.textureURL
	rebuilt.sourceArchiveKey = idx.sourceArchiveKey
	rebuilt.sourceArchiveRevision = idx.sourceArchiveRevision
	rebuilt.projectFlags = idx.projectFlags
	rebuilt.projectSprites = idx.projectSprites
	rebuilt.projectSpriteVersions = idx.projectSpriteVersions
	rebuilt.projectFileSources = projectFileSources
	rebuilt.spriteOverrides = idx.spriteOverrides
	rebuilt.projectRawTextures = idx.projectRawTextures
	rebuilt.expandDuplicateArchetypes()
	return rebuilt, nil
}

func projectFileSourcesWithoutMask(sources map[string]string, mask lowerFileMask) (map[string]string, bool) {
	kept := make(map[string]string, len(sources))
	changed := false
	for name, source := range sources {
		if mask.removes(name) {
			changed = true
			continue
		}
		kept[name] = source
	}
	if !changed {
		return sources, false
	}
	return kept, true
}

func (mask lowerFileMask) removes(name string) bool {
	folded := strings.ToLower(strings.TrimPrefix(strings.ReplaceAll(name, "\\", "/"), "/"))
	return mask.exact[folded] || mask.folders[strings.ToLower(path.Dir(folded))]
}

func externalWithoutMask(external map[string]ExternalTexture, mask lowerFileMask) (map[string]ExternalTexture, bool) {
	kept := make(map[string]ExternalTexture, len(external))
	changed := false
	for name, texture := range external {
		if mask.removes(name) {
			changed = true
			continue
		}
		kept[name] = texture
	}
	if !changed {
		return external, false
	}
	return kept, true
}

const maskedIndexCacheEntries = 4

type maskedIndexEntry struct {
	source *Index
	key    string
	out    *Index
}

var maskedIndexes = struct {
	sync.Mutex
	entries []maskedIndexEntry
}{}

func (idx *Index) withoutLowerFilesCached(mask lowerFileMask) (*Index, error) {
	if idx == nil {
		return nil, nil
	}
	key := idx.removedFilesKey(mask)
	if key == "" {
		return idx, nil
	}
	maskedIndexes.Lock()
	for i, entry := range maskedIndexes.entries {
		if entry.source == idx && entry.key == key {
			maskedIndexes.entries = append(append(maskedIndexes.entries[:i:i], maskedIndexes.entries[i+1:]...), entry)
			maskedIndexes.Unlock()
			return entry.out, nil
		}
	}
	maskedIndexes.Unlock()
	out, err := idx.withoutLowerFiles(mask)
	if err != nil {
		return nil, err
	}
	maskedIndexes.Lock()
	maskedIndexes.entries = append(maskedIndexes.entries, maskedIndexEntry{source: idx, key: key, out: out})
	if over := len(maskedIndexes.entries) - maskedIndexCacheEntries; over > 0 {
		maskedIndexes.entries = append([]maskedIndexEntry(nil), maskedIndexes.entries[over:]...)
	}
	maskedIndexes.Unlock()
	return out, nil
}

func (idx *Index) removedFilesKey(mask lowerFileMask) string {
	removed := []string{}
	depth := 0
	for layer := idx; layer != nil; layer = layer.fallback {
		prefix := string(rune('a'+depth%26)) + ":"
		for name := range layer.files {
			if mask.removes(name) {
				removed = append(removed, prefix+"f:"+name)
			}
		}
		for name := range layer.external {
			if mask.removes(name) {
				removed = append(removed, prefix+"x:"+name)
			}
		}
		for name := range layer.projectFileSources {
			if mask.removes(name) {
				removed = append(removed, prefix+"p:"+name)
			}
		}
		depth++
	}
	if len(removed) == 0 {
		return ""
	}
	sort.Strings(removed)
	return strings.Join(removed, "|")
}

func (idx *Index) WithoutFilesAndFolders(files, folders []string) (*Index, error) {
	mask := lowerFileMask{exact: normalisedMaskNames(files), folders: normalisedMaskNames(folders)}
	if len(mask.exact) == 0 && len(mask.folders) == 0 {
		return idx, nil
	}
	return idx.withoutLowerFilesCached(mask)
}

func (idx *Index) OwnFiles() []string {
	if idx == nil {
		return nil
	}
	out := make([]string, 0, len(idx.files))
	for name := range idx.files {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

func (idx *Index) Over(below *Index) *Index {
	if idx == nil {
		return below
	}
	linked := *idx
	linked.fallback = below
	if len(idx.equipmentDuplicates) > 0 {
		linked.equipment = make(map[string]*equipmentRecord, len(idx.equipment))
		for name, record := range idx.equipment {
			linked.equipment[name] = record
		}
		linked.expandDuplicateArchetypes()
	}
	return &linked
}

func ReleaseCaches() {
	maskedIndexes.Lock()
	maskedIndexes.entries = nil
	maskedIndexes.Unlock()
	vanillaWithoutCache.Lock()
	vanillaWithoutCache.built = map[vanillaWithoutKey]*Index{}
	vanillaWithoutCache.order = nil
	vanillaWithoutCache.Unlock()
}
