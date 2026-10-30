package build

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"path"
	"regexp"
	"sort"
	"strings"
	"time"

	"haganeworks/internal/gameindex"
	"haganeworks/internal/hoi4/importsvc/trees"
)

var focusSpriteRe = regexp.MustCompile(`(?is)SpriteType\s*=\s*\{[^}]*?\bname\s*=\s*"(GFX_(?:goal|focus)_[A-Za-z0-9_-]+)"[^}]*?\btexturefile\s*=\s*"([^"]+)"`)

type focusIcon struct {
	Name        string  `json:"name"`
	Texturefile string  `json:"texturefile"`
	DLC         *string `json:"dlc"`
	Category    string  `json:"category"`
}

func focusIconsOf(text string) []focusIcon {
	text = stripComments(text)
	seen := map[string]bool{}
	var out []focusIcon
	for _, m := range focusSpriteRe.FindAllStringSubmatch(text, -1) {
		name := m[1]
		if strings.HasSuffix(name, "_shine") || seen[name] {
			continue
		}
		seen[name] = true
		category := "focus"
		if strings.HasPrefix(name, "GFX_goal_") {
			category = "goal"
		}
		out = append(out, focusIcon{Name: name, Texturefile: strings.ReplaceAll(m[2], "\\", "/"), Category: category})
	}
	return out
}

func buildFocusIcons(ctx context.Context, in *gameindex.Input) ([]byte, error) {
	all := map[string]focusIcon{}
	byFile := newOrdered()
	add := func(rel string, dlc *string) {
		text, ok := readBase(in, rel)
		if !ok {
			return
		}
		icons := focusIconsOf(text)
		if len(icons) == 0 && dlc != nil {
			return
		}
		key := "interface/" + path.Base(rel)
		if dlc != nil {
			key = path.Dir(path.Dir(rel)) + "/" + path.Base(rel)
		}
		byFile.set(key, len(icons))
		for _, icon := range icons {
			if _, seen := all[icon.Name]; !seen {
				icon.DLC = dlc
				all[icon.Name] = icon
			}
		}
	}
	add("interface/goals.gfx", nil)
	for _, group := range []string{"integrated_dlc", "dlc"} {
		for _, dir := range layerDirs(in.FS, group) {
			if !strings.HasPrefix(path.Base(dir), "dlc") {
				continue
			}
			name := path.Base(dir)
			for _, rel := range listFiles(in.FS, dir+"/interface", ".gfx") {
				if strings.Contains(path.Base(rel), "goals") {
					add(rel, &name)
				}
			}
		}
	}
	if len(all) == 0 {
		return nil, missingFile("interface/goals.gfx")
	}
	names := sortedKeys(all)
	icons := make([]focusIcon, 0, len(names))
	for _, name := range names {
		icons = append(icons, all[name])
	}
	out := newOrdered()
	out.set("source", "interface/goals.gfx + integrated_dlc/*/interface/*goals*.gfx + dlc/*/interface/*goals*.gfx")
	out.set("game_version", shortVersion(in.GameVersion))
	out.set("total", len(icons))
	out.set("by_file", byFile)
	out.set("names", names)
	out.set("icons", icons)
	return marshal(out)
}

const (
	focusRegistrySchema    = 2
	focusRegistryGenerator = "focus-path-registry-v2"
)

func buildFocusTrees(ctx context.Context, in *gameindex.Input) ([]byte, error) {
	files := baseFiles(in, "common/national_focus", ".txt")
	if len(files) == 0 {
		return nil, missingFile("common/national_focus")
	}
	branch := ""
	if raw, err := fs.ReadFile(in.FS, "clausewitz_branch.txt"); err == nil {
		branch = strings.TrimSpace(pyText(raw))
	}
	type record = map[string]any
	var primary, shared, ignored []record
	hashes := newOrdered()
	var hashKeys []string
	hashOf := map[string]string{}
	for _, rel := range files {
		if err := checkCtx(ctx); err != nil {
			return nil, err
		}
		raw, err := fs.ReadFile(in.FS, rel)
		if err != nil {
			continue
		}
		sum := sha256.Sum256(raw)
		hash := hex.EncodeToString(sum[:])
		name := path.Base(rel)
		canonical := "common/national_focus/" + name
		hashKeys = append(hashKeys, canonical)
		hashOf[canonical] = hash
		bundles, summary, _ := trees.ImportFocusFile(pyText(raw), name)
		var primaryBundles, sharedBundles []map[string]any
		focusCount := 0
		var poolIDs []string
		for _, bundle := range bundles {
			switch bundle.TreeKind {
			case "primary":
				primaryBundles = append(primaryBundles, bundle.TreeData)
			case "shared_pool":
				sharedBundles = append(sharedBundles, bundle.TreeData)
				focusCount += len(bundle.NodesData)
				poolIDs = append(poolIDs, strings.TrimSpace(fmt.Sprint(orEmptyAny(bundle.TreeData["tree_id"]))))
			}
		}
		if len(primaryBundles) > 1 {
			return nil, fmt.Errorf("%s contains %d primary trees", name, len(primaryBundles))
		}
		for _, tree := range primaryBundles {
			id := strings.TrimSpace(fmt.Sprint(orEmptyAny(tree["tree_id"])))
			if id == "" {
				return nil, fmt.Errorf("primary focus tree without tree_id: %s", name)
			}
			primary = append(primary, record{"tree_id": id, "canonical_path": canonical, "source_file": name, "content_sha256": hash})
		}
		if len(sharedBundles) > 0 {
			sort.Strings(poolIDs)
			shared = append(shared, record{"canonical_path": canonical, "source_file": name, "content_sha256": hash,
				"pool_ids": poolIDs, "focus_count": focusCount})
		}
		if len(bundles) == 0 {
			warnings := []string{}
			if summary != nil && summary.Warnings != nil {
				warnings = append(warnings, summary.Warnings...)
			}
			ignored = append(ignored, record{"canonical_path": canonical, "source_file": name, "content_sha256": hash,
				"reason": "no_primary_or_shared_focus_bundle", "parser_warnings": warnings})
		}
	}
	sort.SliceStable(primary, func(i, j int) bool {
		a, b := primary[i]["tree_id"].(string), primary[j]["tree_id"].(string)
		if a != b {
			return a < b
		}
		return primary[i]["canonical_path"].(string) < primary[j]["canonical_path"].(string)
	})
	sort.SliceStable(shared, func(i, j int) bool {
		return shared[i]["canonical_path"].(string) < shared[j]["canonical_path"].(string)
	})
	sort.SliceStable(ignored, func(i, j int) bool {
		return ignored[i]["canonical_path"].(string) < ignored[j]["canonical_path"].(string)
	})
	sort.Strings(hashKeys)
	for _, key := range hashKeys {
		hashes.set(key, hashOf[key])
	}
	ids := []string{}
	byFile, pathByTree := map[string]string{}, map[string]string{}
	seen := map[string]bool{}
	for _, r := range primary {
		id := r["tree_id"].(string)
		if seen[id] {
			return nil, fmt.Errorf("duplicate primary focus tree id %s", id)
		}
		seen[id] = true
		ids = append(ids, id)
		byFile[r["source_file"].(string)] = id
		pathByTree[id] = r["canonical_path"].(string)
	}
	sort.Strings(ids)
	orEmptyList := func(list []record) []record {
		if list == nil {
			return []record{}
		}
		return list
	}
	now := time.Now().UTC()
	return marshal(map[string]any{
		"schema_version":         focusRegistrySchema,
		"game_version":           "hoi4-" + shortVersion(in.GameVersion),
		"game_branch":            branch,
		"generator_version":      focusRegistryGenerator,
		"source":                 "live HOI4 installation parsed by app.services.import_service.import_focus_file",
		"source_directory":       "Hearts of Iron IV/common/national_focus/*.txt",
		"file_count":             len(files),
		"primary_count":          len(primary),
		"shared_pool_file_count": len(shared),
		"ignored_file_count":     len(ignored),
		"primary_records":        orEmptyList(primary),
		"shared_pool_files":      orEmptyList(shared),
		"ignored_files":          orEmptyList(ignored),
		"source_hashes":          hashes,
		"total":                  len(ids),
		"ids":                    ids,
		"by_file":                byFile,
		"path_by_tree_id":        pathByTree,
		"generated_at":           now.Format(time.RFC3339),
		"extracted_at":           now.Format("2006-01-02"),
		"primary_paths":          pathByTree,
	})
}

func orEmptyAny(v any) any {
	if v == nil {
		return ""
	}
	return v
}
