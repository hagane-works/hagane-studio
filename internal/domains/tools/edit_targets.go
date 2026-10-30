package tools

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"sync"

	"haganeworks/internal/export"
	"haganeworks/internal/hoi4/exportengine"
	"haganeworks/internal/project"
)

type editTargetKind struct {
	Label string         `json:"label"`
	Count int            `json:"count"`
	Modes map[string]int `json:"modes"`
	Paths []string       `json:"paths"`
}

type editTargetsResponse struct {
	IsLoadedExistingMod bool                       `json:"is_loaded_existing_mod"`
	PrimaryTag          string                     `json:"primary_tag"`
	ModName             string                     `json:"mod_name"`
	ByKind              map[string]*editTargetKind `json:"by_kind"`
	TotalFiles          int                        `json:"total_files"`
	Error               string                     `json:"error,omitempty"`
}

var editTargetPathRules = []struct{ prefix, kind string }{
	{"common/country_leader/", "custom_trait_country_leader"},
	{"common/unit_leader/", "custom_trait_unit_leader"},
	{"common/scientist_traits/", "custom_trait_scientist"},
	{"common/decisions/categories/", "decision_category"},
	{"common/decisions/", "decision"},
	{"common/national_focus/", "focus_tree"},
	{"common/technologies/", "technology"},
	{"common/ideas/", "idea"},
	{"common/opinion_modifiers/", "opinion_modifier"},
	{"common/characters/", "character"},
	{"common/bookmarks/", "bookmark"},
	{"common/on_actions/", "on_action"},
	{"history/countries/", "history_file"},
	{"events/", "event"},
	{"localisation/", "localisation"},
	{"interface/", "gfx"},
	{"portraits/", "portrait"},
	{"gfx/", "asset"},
	{"descriptor.mod", "descriptor"},
	{"thumbnail", "descriptor"},
}

var editTargetLabels = map[string]string{
	"character":                   "角色",
	"custom_trait_country_leader": "领袖 trait",
	"custom_trait_unit_leader":    "将领 trait",
	"custom_trait_scientist":      "科学家 trait",
	"focus_tree":                  "国策树",
	"technology":                  "科技",
	"idea":                        "Idea / 国家精神",
	"event":                       "事件",
	"decision":                    "决议",
	"decision_category":           "决议分类",
	"opinion_modifier":            "外交立场 modifier",
	"bookmark":                    "剧本",
	"history_file":                "国家起点 history",
	"on_action":                   "on_actions",
	"localisation":                "本地化 yml",
	"gfx":                         "GFX 映射",
	"portrait":                    "肖像 PNG",
	"asset":                       "GFX 资源",
	"descriptor":                  "Mod descriptor",
}

var editTargetCache = struct {
	sync.Mutex
	order []string
	paths map[string][]string
}{paths: map[string][]string{}}

const editTargetCacheSize = 16

func editTargets(p *project.Project, w http.ResponseWriter, r *http.Request) error {
	includeImported, err := parseEditTargetsBool(r.URL.Query().Get("include_imported"))
	if err != nil {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]any{"detail": []map[string]any{{
			"type": "bool_parsing", "loc": []string{"query", "include_imported"}, "msg": "Input should be a valid boolean",
		}}})
		return nil
	}
	snapshot, err := export.Snapshot(p)
	if err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"error": "export snapshot unavailable"})
		return nil
	}
	response := editTargetProjectMetadata(snapshot)
	key := fmt.Sprintf("%s|%t", snapshot.Fingerprint, includeImported)
	paths, ok := cachedEditTargetPaths(key)
	if !ok {
		index, err := exportengine.BuildNativeFileIndex(snapshot, nil, includeImported)
		if err != nil {
			response.Error = "compute failed: " + err.Error()
			writeJSON(w, http.StatusOK, response)
			return nil
		}
		paths = index.Paths
		storeEditTargetPaths(key, paths)
	}
	writeJSON(w, http.StatusOK, classifyEditTargets(response, paths))
	return nil
}

func cachedEditTargetPaths(key string) ([]string, bool) {
	editTargetCache.Lock()
	defer editTargetCache.Unlock()
	paths, ok := editTargetCache.paths[key]
	return paths, ok
}

func storeEditTargetPaths(key string, paths []string) {
	editTargetCache.Lock()
	defer editTargetCache.Unlock()
	if _, ok := editTargetCache.paths[key]; ok {
		return
	}
	editTargetCache.paths[key] = paths
	editTargetCache.order = append(editTargetCache.order, key)
	if len(editTargetCache.order) > editTargetCacheSize {
		delete(editTargetCache.paths, editTargetCache.order[0])
		editTargetCache.order = editTargetCache.order[1:]
	}
}

func editTargetProjectMetadata(snapshot exportengine.Snapshot) editTargetsResponse {
	var record struct {
		Name                string `json:"name"`
		CountryTag          string `json:"country_tag"`
		IsLoadedExistingMod bool   `json:"is_loaded_existing_mod"`
	}
	_ = json.Unmarshal(snapshot.Project, &record)
	name := record.Name
	if name == "" {
		name = "mod"
	}
	return editTargetsResponse{
		IsLoadedExistingMod: record.IsLoadedExistingMod,
		PrimaryTag:          strings.ToUpper(record.CountryTag),
		ModName:             name,
		ByKind:              map[string]*editTargetKind{},
	}
}

func classifyEditTargets(response editTargetsResponse, paths []string) editTargetsResponse {
	ordered := append([]string(nil), paths...)
	sort.Strings(ordered)
	for _, zipPath := range ordered {
		if strings.HasSuffix(zipPath, "/") || strings.HasSuffix(zipPath, "_CROSS_REF_SKIPPED.log") {
			continue
		}
		response.TotalFiles++
		kind := classifyEditTargetPath(zipPath)
		rel := stripEditTargetModDir(zipPath)
		mode := editTargetMode(rel, response.PrimaryTag)
		bucket := response.ByKind[kind]
		if bucket == nil {
			label := editTargetLabels[kind]
			if label == "" {
				label = kind
			}
			bucket = &editTargetKind{Label: label, Modes: map[string]int{}}
			response.ByKind[kind] = bucket
		}
		bucket.Count++
		bucket.Modes[mode]++
		if len(bucket.Paths) < 30 {
			bucket.Paths = append(bucket.Paths, rel)
		}
	}
	return response
}

func classifyEditTargetPath(zipPath string) string {
	rel := stripEditTargetModDir(zipPath)
	for _, rule := range editTargetPathRules {
		if strings.HasPrefix(rel, rule.prefix) || strings.HasSuffix(zipPath, strings.TrimSuffix(rule.prefix, "/")) {
			return rule.kind
		}
	}
	return "other"
}

func stripEditTargetModDir(zipPath string) string {
	if at := strings.IndexByte(zipPath, '/'); at >= 0 {
		return zipPath[at+1:]
	}
	return zipPath
}

func editTargetMode(relPath, primaryTag string) string {
	filename := relPath
	if at := strings.LastIndexByte(filename, '/'); at >= 0 {
		filename = filename[at+1:]
	}
	upperTag := strings.ToUpper(primaryTag)
	if upperTag != "" && (strings.HasPrefix(filename, upperTag+"_") || strings.HasPrefix(filename, "zz_"+upperTag+"_")) {
		return "append"
	}
	if strings.HasPrefix(filename, "descriptor.mod") || strings.HasPrefix(filename, "thumbnail") {
		return "overwrite"
	}
	return "shared"
}

func parseEditTargetsBool(raw string) (bool, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "", "0", "false", "off", "no":
		return false, nil
	case "1", "true", "on", "yes":
		return true, nil
	default:
		return false, fmt.Errorf("invalid boolean value %q", raw)
	}
}
