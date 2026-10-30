package guirender

import (
	"fmt"
	"path"
	"sort"
	"strings"
	"sync"
)

func (idx *Index) WithoutVanillaFolders(folders []string) (*Index, error) {
	if idx == nil || len(folders) == 0 {
		return idx, nil
	}
	set := map[string]bool{}
	for _, folder := range folders {
		clean := strings.ToLower(strings.Trim(strings.ReplaceAll(strings.TrimSpace(folder), "\\", "/"), "/"))
		if clean != "" {
			set[clean] = true
		}
	}
	if len(set) == 0 {
		return idx, nil
	}
	return idx.withoutVanilla(set)
}

func (idx *Index) withoutVanilla(folders map[string]bool) (*Index, error) {
	if idx.source == "vanilla" {
		return vanillaWithout(idx, folders)
	}
	if idx.fallback == nil {
		return idx, nil
	}
	below, err := idx.fallback.withoutVanilla(folders)
	if err != nil || below == idx.fallback {
		return idx, err
	}

	layer := *idx
	layer.fallback = below
	return &layer, nil
}

type vanillaWithoutKey struct {
	layer   *Index
	folders string
}

var vanillaWithoutCache = struct {
	sync.Mutex
	built map[vanillaWithoutKey]*Index
	order []vanillaWithoutKey
}{built: map[vanillaWithoutKey]*Index{}}

func vanillaWithout(layer *Index, folders map[string]bool) (*Index, error) {
	names := make([]string, 0, len(folders))
	for folder := range folders {
		names = append(names, folder)
	}
	sort.Strings(names)
	k := vanillaWithoutKey{layer, strings.Join(names, "|")}
	cache := &vanillaWithoutCache
	cache.Lock()
	defer cache.Unlock()
	if out, ok := cache.built[k]; ok {
		return out, nil
	}
	files := make(map[string][]byte, len(layer.files))
	for name, body := range layer.files {
		if !folders[strings.ToLower(path.Dir(name))] {
			files[name] = body
		}
	}
	external, externalChanged := externalWithoutMask(layer.external, lowerFileMask{folders: folders})
	out := layer
	if len(files) != len(layer.files) {
		rebuilt, partial, err := layer.withoutOwnFiles(lowerFileMask{folders: folders})
		if !partial && err == nil {
			rebuilt, err = newIndex(files, layer.source)
		}
		if err != nil {
			return nil, fmt.Errorf("base game without %v: %w", names, err)
		}
		rebuilt.version, rebuilt.baseURL, rebuilt.textureURL = layer.version, layer.baseURL, layer.textureURL
		rebuilt.external = external
		rebuilt.externalLoose = buildLooseTextureIndex(external)
		rebuilt.projectFlags = layer.projectFlags
		rebuilt.fallback = layer.fallback
		out = rebuilt
	} else if externalChanged {
		copy := *layer
		copy.external = external
		copy.externalLoose = buildLooseTextureIndex(external)
		out = &copy
	}
	cache.built[k] = out
	cache.order = append(cache.order, k)
	if len(cache.order) > 4 {
		delete(cache.built, cache.order[0])
		cache.order = cache.order[1:]
	}
	return out, nil
}

func (idx *Index) HasOwnFilesIn(folder string) bool {
	if idx == nil {
		return false
	}
	folder = strings.ToLower(strings.Trim(folder, "/"))
	for name := range idx.files {
		if strings.ToLower(path.Dir(name)) == folder {
			return true
		}
	}
	return false
}
