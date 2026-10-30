package imagebatch

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

var VanillaAssetDirs = map[string]string{
	"idea_pictures":     "gfx/interface/ideas",
	"event_pictures":    "gfx/event_pictures",
	"decision_pictures": "gfx/interface/decisions",
	"focus_icons":       "gfx/interface/goals",
}

var VanillaAssetTypes = []string{"idea_pictures", "event_pictures", "decision_pictures", "focus_icons"}

type VanillaAsset struct {
	Name         string `json:"name"`
	Filename     string `json:"filename"`
	RelativePath string `json:"relative_path"`
}

func VanillaRoot() string {
	if configured := strings.TrimSpace(os.Getenv("VANILLA_ROOT")); configured != "" {
		return filepath.Clean(configured)
	}
	return filepath.Clean("/app/vanilla")
}

func ListVanillaAssets(root, assetType string) ([]VanillaAsset, error) {
	subdir, ok := VanillaAssetDirs[assetType]
	if !ok {
		keys := make([]string, 0, len(VanillaAssetDirs))
		for key := range VanillaAssetDirs {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		return nil, fmt.Errorf("Unknown asset_type %q. Expected one of: %v", assetType, keys)
	}
	dir := filepath.Join(root, filepath.FromSlash(subdir))
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return []VanillaAsset{}, nil
	}
	if err != nil {
		return nil, err
	}
	assets := make([]VanillaAsset, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.EqualFold(filepath.Ext(entry.Name()), ".dds") {
			continue
		}
		assets = append(assets, VanillaAsset{
			Name: strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name())), Filename: entry.Name(),
			RelativePath: filepath.ToSlash(filepath.Join(subdir, entry.Name())),
		})
	}
	sort.Slice(assets, func(i, j int) bool { return assets[i].Filename < assets[j].Filename })
	return assets, nil
}
