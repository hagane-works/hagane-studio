package guirender

import (
	"sort"
	"strings"

	"haganeworks/internal/hoi4/gfx"
	"haganeworks/internal/hoi4/loc"
)

func (idx *Index) SetProjectFileSources(sources map[string]string) {
	if idx == nil {
		return
	}
	if len(sources) == 0 {
		idx.projectFileSources = nil
		return
	}
	idx.projectFileSources = make(map[string]string, len(sources))
	for name, source := range sources {
		key := strings.ToLower(strings.Trim(strings.ReplaceAll(name, "\\", "/"), "/"))
		if key != "" {
			idx.projectFileSources[key] = source
		}
	}
}

func (idx *Index) ProjectFileSource(name string) (string, bool) {
	if idx == nil {
		return "", false
	}
	key := strings.ToLower(strings.Trim(strings.ReplaceAll(name, "\\", "/"), "/"))
	source, ok := idx.projectFileSources[key]
	return source, ok
}

func (idx *Index) ProjectDataLocFile(key, header string) (string, bool) {
	if idx == nil || key == "" || len(idx.projectFileSources) == 0 {
		return "", false
	}
	names := make([]string, 0, 8)
	for name := range idx.files {
		lower := strings.ToLower(name)
		if !strings.HasPrefix(lower, "localisation/") || !strings.HasSuffix(lower, ".yml") || strings.Contains(lower, "/replace/") {
			continue
		}
		source, ok := idx.ProjectFileSource(name)
		if !ok || source == "project_loc_file" || strings.HasPrefix(source, "project_raw_files") {
			continue
		}
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		text, ok := gfx.DecodeScriptText(idx.files[name])
		if !ok {
			continue
		}
		language, entries, _ := loc.ParseLocText(text)
		if language != header {
			continue
		}
		if _, found := entries[key]; found {
			return name, true
		}
	}
	return "", false
}
