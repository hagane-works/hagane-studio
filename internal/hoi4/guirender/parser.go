package guirender

import (
	"math"
	"strconv"
	"strings"

	"haganeworks/internal/hoi4/paradox"
)

var elementTypes = map[string]bool{
	"containerwindowtype": true, "windowtype": true, "icontype": true,
	"buttontype": true, "guibuttontype": true, "instanttextboxtype": true,
	"textboxtype": true, "editboxtype": true, "checkboxtype": true,
	"listboxtype": true, "smoothlistboxtype": true, "gridboxtype": true,
	"scrollbartype": true, "extendedscrollbartype": true,
	"overlappingelementsboxtype": true, "positiontype": true,
	"shieldtype": true, "progressbartype": true,

	"dropdownboxtype": true, "expandedwindow": true,
}

var pointKeys = map[string]bool{
	"position": true, "size": true, "maxsize": true, "slotsize": true,
	"bordersize": true, "origo": true,

	"show_position": true,
}

var spriteKeys = map[string]string{
	"quadtexturesprite": "quadTextureSprite", "spritetype": "spriteType",
	"texturefile": "textureFile", "buttonmesh": "buttonMesh",
}

func ParseGUI(text string) ([]map[string]any, error) {
	root, _, err := paradox.Parse(text)
	if err != nil {
		return nil, err
	}
	expandConstants(root)
	windows := make([]map[string]any, 0)
	var scan func([]*paradox.Entry)
	scan = func(entries []*paradox.Entry) {
		for _, entry := range entries {
			if entry == nil || entry.Key == nil || entry.Block == nil {
				continue
			}
			key := strings.ToLower(*entry.Key)
			switch key {
			case "guitypes":
				scan(entry.Block.Entries)
			case "containerwindowtype", "windowtype":
				windows = append(windows, buildNode(key, entry.Block.Entries))
			}
		}
	}
	scan(root.Entries)
	return windows, nil
}

func expandConstants(root *paradox.Block) {
	registry := map[string]paradox.Scalar{}
	var harvest func([]*paradox.Entry)
	harvest = func(entries []*paradox.Entry) {
		for _, entry := range entries {
			if entry == nil {
				continue
			}
			if entry.Block != nil {
				harvest(entry.Block.Entries)
				continue
			}
			if entry.Key != nil && strings.HasPrefix(*entry.Key, "@") {
				registry[*entry.Key] = entry.Scalar
			}
		}
	}
	harvest(root.Entries)
	if len(registry) == 0 {
		return
	}
	var walk func([]*paradox.Entry)
	walk = func(entries []*paradox.Entry) {
		for _, entry := range entries {
			if entry == nil {
				continue
			}
			if entry.Block != nil {
				walk(entry.Block.Entries)
				continue
			}
			if entry.Scalar.Kind == paradox.ScalarStr && strings.HasPrefix(entry.Scalar.Str, "@") {
				if value, ok := registry[entry.Scalar.Str]; ok {
					entry.Scalar = value
				}
			}
		}
	}
	walk(root.Entries)
}

func buildNode(elemType string, body []*paradox.Entry) map[string]any {
	node := map[string]any{
		"type": strings.ToLower(elemType), "name": nil,
		"position": map[string]any{"x": 0, "y": 0}, "size": nil,
		"orientation": nil, "origo": nil, "scale": nil, "sprite": nil,
		"sprite_key": nil, "frame": nil, "text": nil, "text_source": nil,
		"font": nil, "format": nil, "format_token": nil, "maxwidth": nil,
		"maxheight": nil, "text_color": nil, "centerposition": false,
		"alwaystransparent": false, "background": nil,
		"children": []map[string]any{}, "raw": map[string]any{},
	}
	raw := node["raw"].(map[string]any)
	for _, entry := range body {
		if entry == nil || entry.Key == nil {
			continue
		}
		key := strings.ToLower(*entry.Key)
		if entry.Block != nil && elementTypes[key] {
			node["children"] = append(node["children"].([]map[string]any), buildNode(key, entry.Block.Entries))
			continue
		}
		if key == "background" && entry.Block != nil {

			if node["background"] == nil {
				node["background"] = buildNode("background", entry.Block.Entries)
			}
			continue
		}
		if entry.Block != nil && pointKeys[key] {
			pt := point(entry.Block.Entries)
			switch key {
			case "position":
				node["position"] = map[string]any{"x": valueOr(pt, "x", 0), "y": valueOr(pt, "y", 0)}
			case "size":
				node["size"] = pt
			case "origo":
				raw["origo"] = pt
			default:
				raw[key] = pt
			}
			if _, ok := losslessPoint(entry.Block.Entries); !ok {
				tokens, _ := node["point_tokens"].(map[string]any)
				if tokens == nil {
					tokens = map[string]any{}
					node["point_tokens"] = tokens
				}
				tokens[key] = map[string]any{"__block__": rawBlockItems(entry.Block.Entries)}
			}
			continue
		}
		if entry.Block != nil {
			if pt, ok := losslessPoint(entry.Block.Entries); ok {
				raw[key] = pt
			} else {
				raw[key] = map[string]any{"__block__": rawBlockItems(entry.Block.Entries)}
			}
			continue
		}
		value := scalarValue(entry.Scalar)
		text := entry.Scalar.StrValue()
		switch key {
		case "name":
			node["name"] = text
		case "quadtexturesprite", "spritetype", "texturefile", "buttonmesh":

			replaces := key == "quadtexturesprite" && elemType == "buttontype" &&
				node["sprite_key"] == spriteKeys["spritetype"]
			if node["sprite"] == nil || replaces {
				node["sprite"], node["sprite_key"] = text, spriteKeys[key]
			}
		case "frame":
			node["frame"] = toInt(value, 1)
		case "scale":
			if number, ok := toFloat(value); ok {
				node["scale"] = number
			}
		case "orientation":
			node["orientation"] = strings.ToUpper(text)
		case "origo":
			node["origo"] = strings.ToUpper(text)
		case "text", "buttontext":
			node["text"] = text
			node["text_source"] = key
		case "font", "buttonfont":
			node["font"] = text
		case "format":
			node["format"], node["format_token"] = strings.ToLower(text), text
		case "maxwidth", "maxheight":
			coerced := toInt(value, 0)
			node[key] = coerced
			if strconv.Itoa(coerced) != strings.TrimSpace(pythonString(value)) {
				tokens, _ := node["scalar_tokens"].(map[string]any)
				if tokens == nil {
					tokens = map[string]any{}
					node["scalar_tokens"] = tokens
				}
				tokens[key] = value
			}
		case "text_color_code":
			node["text_color"] = text
		case "centerposition":
			node["centerposition"] = truthy(value)
		case "alwaystransparent":
			node["alwaystransparent"] = truthy(value)
		default:
			raw[key] = value
		}
	}
	return node
}

func point(entries []*paradox.Entry) map[string]any {
	out := map[string]any{}
	for _, entry := range entries {
		if entry == nil || entry.Key == nil || entry.Block != nil {
			continue
		}
		key := strings.ToLower(*entry.Key)
		value := scalarValue(entry.Scalar)
		if text, ok := value.(string); ok && strings.HasSuffix(text, "%") {
			out[key] = text
		} else {
			out[key] = toInt(value, 0)
		}
	}
	return out
}

func losslessPoint(entries []*paradox.Entry) (map[string]any, bool) {
	out := map[string]any{}
	for _, entry := range entries {
		if entry == nil || entry.Key == nil || entry.Block != nil {
			return nil, false
		}
		key := strings.ToLower(*entry.Key)
		value := scalarValue(entry.Scalar)
		if text, ok := value.(string); ok && strings.HasSuffix(text, "%") {
			out[key] = text
			continue
		}
		coerced, ok := strictInt(value)
		if !ok || strconv.Itoa(coerced) != strings.TrimSpace(pythonString(value)) {
			return nil, false
		}
		out[key] = coerced
	}
	return out, true
}

func rawBlockItems(entries []*paradox.Entry) []map[string]any {
	out := make([]map[string]any, 0, len(entries))
	for _, entry := range entries {
		if entry == nil {
			continue
		}
		var key any
		if entry.Key != nil {
			key = *entry.Key
		}
		var value any
		if entry.Block != nil {
			value = map[string]any{"__block__": rawBlockItems(entry.Block.Entries)}
		} else {
			value = scalarValue(entry.Scalar)
		}
		item := map[string]any{"k": key, "v": value}
		if entry.Quoted {
			item["q"] = true
		}
		if entry.Op != nil && *entry.Op != "=" {
			item["op"] = *entry.Op
		}
		out = append(out, item)
	}
	return out
}

func scalarValue(value paradox.Scalar) any {
	switch value.Kind {
	case paradox.ScalarInt:
		if value.Int != nil && value.Int.IsInt64() {
			return int(value.Int.Int64())
		}
		return value.StrValue()
	case paradox.ScalarFloat:
		return value.Float
	default:
		return value.Str
	}
}

func strictInt(value any) (int, bool) {
	f, ok := toFloat(value)
	if !ok || math.IsNaN(f) || math.IsInf(f, 0) {
		return 0, false
	}
	return int(f), true
}

func toInt(value any, fallback int) int {
	if n, ok := strictInt(value); ok {
		return n
	}
	return fallback
}

func toFloat(value any) (float64, bool) {
	switch v := value.(type) {
	case int:
		return float64(v), true
	case int64:
		return float64(v), true
	case float64:
		return v, true
	case string:
		f, err := strconv.ParseFloat(strings.TrimSpace(v), 64)
		return f, err == nil
	default:
		return 0, false
	}
}

func pythonString(value any) string {
	switch v := value.(type) {
	case float64:
		return strconv.FormatFloat(v, 'g', -1, 64)
	default:
		return strings.TrimSpace(strings.TrimSpace(toString(v)))
	}
}

func toString(value any) string {
	switch v := value.(type) {
	case string:
		return v
	case int:
		return strconv.Itoa(v)
	case int64:
		return strconv.FormatInt(v, 10)
	case float64:
		return strconv.FormatFloat(v, 'g', -1, 64)
	default:
		return ""
	}
}

func truthy(value any) bool {
	switch strings.ToLower(strings.TrimSpace(pythonString(value))) {
	case "yes", "true", "1":
		return true
	default:
		return false
	}
}

func valueOr(values map[string]any, key string, fallback any) any {
	if value, ok := values[key]; ok {
		return value
	}
	return fallback
}
