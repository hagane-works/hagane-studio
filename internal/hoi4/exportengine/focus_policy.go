package exportengine

import (
	"context"
	"encoding/json"
	"strings"

	"haganeworks/internal/gameindex"
)

type focusVanillaIconsKey struct{}
type focusVanillaTreesKey struct{}

func focusVanillaIcons() (map[string]bool, bool) {
	icons, err := gameindex.Load(context.Background(), focusVanillaIconsKey{}, gameindex.FocusIcons, func(raw []byte) (map[string]bool, error) {
		var table struct {
			Names []string `json:"names"`
		}
		if err := json.Unmarshal(raw, &table); err != nil {
			return nil, err
		}
		out := map[string]bool{"GFX_goal_unknown": true}
		for _, name := range table.Names {
			if name != "" {
				out[name] = true
			}
		}
		return out, nil
	})
	return icons, err == nil
}

func focusVanillaTrees() map[string]string {
	trees, err := gameindex.Load(context.Background(), focusVanillaTreesKey{}, gameindex.FocusTrees, func(raw []byte) (map[string]string, error) {
		var table struct {
			PrimaryPaths map[string]string `json:"primary_paths"`
		}
		if err := json.Unmarshal(raw, &table); err != nil {
			return nil, err
		}
		out := map[string]string{}
		for treeID, canonical := range table.PrimaryPaths {
			if treeID != "" && canonical != "" {
				out[treeID] = canonical
			}
		}
		return out, nil
	})
	if err != nil {
		return map[string]string{}
	}
	return trees
}

func FocusIconGate(isSubmod bool, uploaded map[string]bool) map[string]bool {
	if isSubmod {

		return nil
	}
	vanilla, ok := focusVanillaIcons()
	if !ok {

		return nil
	}
	gate := make(map[string]bool, len(vanilla)+len(uploaded))
	for name := range vanilla {
		gate[name] = true
	}
	for name := range uploaded {
		if name != "" {
			gate[name] = true
		}
	}
	return gate
}

func FocusFallbackPath(sourceFile, treeID, tag string) string {
	if normalized := normaliseFocusPath(sourceFile); normalized != "" {

		if canonical := focusVanillaTrees()[strings.TrimSpace(treeID)]; canonical != "" {
			return canonical
		}

		return normalized
	}

	return focusNamespacedAdditionPath(treeID, tag)
}

func normaliseFocusPath(value string) string {
	text := strings.ReplaceAll(strings.TrimSpace(value), "\\", "/")
	if text == "" || strings.Contains(text, ":") {
		return ""
	}
	for _, char := range text {
		if char < 32 {
			return ""
		}
	}

	if strings.HasPrefix(text, "/") {
		return ""
	}
	var parts []string
	for _, part := range strings.Split(text, "/") {

		if part == "" || part == "." {
			continue
		}
		if part == ".." {
			return ""
		}
		parts = append(parts, part)
	}
	switch {
	case len(parts) == 0:
		return ""
	case len(parts) == 1:

		parts = []string{"common", "national_focus", parts[0]}
	case len(parts) != 3 || parts[0] != "common" || parts[1] != "national_focus":
		return ""
	}
	if !strings.EqualFold(focusPathExt(parts[2]), ".txt") {
		return ""
	}
	return strings.Join(parts, "/")
}

func focusPathExt(name string) string {
	if dot := strings.LastIndex(name, "."); dot > 0 {
		return name[dot:]
	}
	return ""
}

func focusNamespacedAdditionPath(treeID, tag string) string {
	namespacePart := focusSafeComponent(tag, "mod")
	treePart := focusSafeComponent(treeID, "focus_tree")
	namespacedID := treePart
	if !strings.HasPrefix(strings.ToLower(treePart), strings.ToLower(namespacePart)+"_") {
		namespacedID = namespacePart + "_" + treePart
	}
	return "common/national_focus/" + namespacedID + ".txt"
}

func focusSafeComponent(value, fallback string) string {
	var out strings.Builder
	substituted := false
	for _, char := range strings.TrimSpace(value) {
		switch {
		case char >= 'A' && char <= 'Z', char >= 'a' && char <= 'z',
			char >= '0' && char <= '9', char == '_':
			out.WriteRune(char)
			substituted = false
		case !substituted:
			out.WriteByte('_')
			substituted = true
		}
	}
	if cleaned := strings.Trim(out.String(), "_"); cleaned != "" {
		return cleaned
	}
	return fallback
}
