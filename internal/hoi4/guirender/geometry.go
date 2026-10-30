package guirender

import (
	"regexp"
	"sort"
	"strings"
)

var yearLabelRE = regexp.MustCompile(`(?i)_year\d+$`)

func FindNamedNode(root map[string]any, name string) map[string]any {
	wanted := strings.ToLower(strings.TrimSpace(name))
	if root == nil || wanted == "" {
		return nil
	}
	stack := []map[string]any{root}
	for len(stack) > 0 {
		last := len(stack) - 1
		node := stack[last]
		stack = stack[:last]
		nodeName, _ := node["name"].(string)
		if strings.ToLower(strings.TrimSpace(nodeName)) == wanted {
			return node
		}
		stack = append(stack, nodeChildren(node)...)
	}
	return nil
}

func FolderGeometry(container map[string]any, folder string) map[string]any {
	if container == nil || folder == "" {
		return nil
	}
	grids := make([]map[string]any, 0)
	years := make([]map[string]any, 0)
	var walk func(map[string]any, int, int)
	walk = func(node map[string]any, parentX, parentY int) {
		position, _ := node["position"].(map[string]any)
		x, y := parentX, parentY
		if position != nil {
			x += numeric(position["x"])
			y += numeric(position["y"])
		}
		nodeType, _ := node["type"].(string)
		name, _ := node["name"].(string)
		switch strings.ToLower(nodeType) {
		case "gridboxtype":
			width, height := 70, 70
			if raw, ok := node["raw"].(map[string]any); ok {
				if slot, ok := raw["slotsize"].(map[string]any); ok {
					if _, isBool := slot["width"].(bool); !isBool {
						if _, ok := slot["width"].(int); ok {
							width = numeric(slot["width"])
						}
					}
					if _, isBool := slot["height"].(bool); !isBool {
						if _, ok := slot["height"].(int); ok {
							height = numeric(slot["height"])
						}
					}
				}
			}
			format, _ := node["format"].(string)
			grids = append(grids, map[string]any{
				"name": name, "originX": x, "originY": y,
				"slotWidth": width, "slotHeight": height, "format": format,
			})
		case "instanttextboxtype":
			if yearLabelRE.MatchString(name) {
				text, _ := node["text"].(string)
				years = append(years, map[string]any{"name": name, "text": text, "x": x, "y": y})
			}
		}
		for _, child := range nodeChildren(node) {
			walk(child, x, y)
		}
	}
	for _, child := range nodeChildren(container) {
		walk(child, 0, 0)
	}
	if len(grids) == 0 {
		return nil
	}
	columnMajor := strings.ToLower(strings.TrimSpace(grids[0]["format"].(string))) == "left"
	sort.SliceStable(years, func(i, j int) bool {
		if columnMajor {
			return numeric(years[i]["x"]) < numeric(years[j]["x"])
		}
		return numeric(years[i]["y"]) < numeric(years[j]["y"])
	})
	return map[string]any{"folder": folder, "grids": grids, "yearLabels": years}
}

func numeric(value any) int {
	switch value.(type) {
	case int, int64, float64:
		return toInt(value, 0)
	default:
		return 0
	}
}
