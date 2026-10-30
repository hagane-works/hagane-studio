package guirender

import (
	"strings"

	"haganeworks/internal/reftables"
)

const spriteUsageReferenceLimit = 24

func (idx *Index) SpriteUsagePayload(names []string) map[string]any {
	out := map[string]any{}
	for _, name := range names {
		name = strings.TrimSpace(name)
		if name != "" {
			out[name] = map[string]any{
				"sprite": name, "frame_base": 0,
				"references": []map[string]any{}, "frame_usages": []map[string]any{},
				"exact_runtime_condition_available": false,
				"note":                              "Indexed declarations are evidence, not complete runtime conditions. Read the effective get_gui_window or structured panel/entity and its bindings before redesigning. Mark unavailable conditions unknown; never infer them from pixels or a frame count.",
			}
		}
	}
	if len(out) == 0 {
		return out
	}

	for _, window := range idx.ListWindows() {
		name, _ := window["name"].(string)
		var record windowRecord
		for layer := idx; layer != nil; layer = layer.fallback {
			if found, ok := layer.windows[name]; ok {
				record = found
				break
			}
		}
		var walk func(map[string]any, string, int)
		walk = func(node map[string]any, parent string, depth int) {
			if node == nil || depth > 64 {
				return
			}
			nodeName, _ := node["name"].(string)
			namePath := parent
			if nodeName != "" {
				if namePath != "" {
					namePath += "/"
				}
				namePath += nodeName
			}
			sprite, _ := node["sprite"].(string)
			if usage, ok := out[sprite].(map[string]any); ok {
				refs := usage["references"].([]map[string]any)
				if len(refs) >= spriteUsageReferenceLimit {
					usage["references_truncated"] = true
				} else {
					ref := map[string]any{"file": record.File, "window": name, "name_path": namePath,
						"node_type": node["type"], "source": record.Source,
						"scope": "indexed_gui_declaration", "gui_frame": node["frame"]}
					texts := map[string]any{}
					keys := tooltipKeys(node)
					if key, _ := node["text"].(string); key != "" {
						keys = append(keys, key)
					}
					for _, key := range keys {
						value, resolved := idx.LocaliseMarkup(key)
						texts[key] = map[string]any{"text": value, "resolved": resolved}
					}
					ref["texts"] = texts
					usage["references"] = append(refs, ref)
				}
			}
			if bg, _ := node["background"].(map[string]any); bg != nil {
				walk(bg, namePath, depth+1)
			}
			for _, child := range nodeChildren(node) {
				walk(child, namePath, depth+1)
			}
		}
		walk(record.Node, "", 0)
	}

	if usage, ok := out[idx.alertTemplateSprite()].(map[string]any); ok {
		geometry, declared := idx.SpriteTextureGeometry(idx.alertTemplateSprite())
		if declared && geometry.Frames == 70 {
			frames := []map[string]any{}
			for _, alert := range reftables.Alerts().Engine {
				title, detail := idx.alertTexts(alert.Name)
				confidence := "engine_mapping"
				if alert.Type == 68 {
					confidence = "inferred_alert_name"
				}
				frames = append(frames, map[string]any{
					"frame": alert.Frame - 1, "game_frame": alert.Frame,
					"alert_type": alert.Type, "key": alert.Name, "title": title,
					"description": detail, "category": idx.alertCategory(alert.Name),
					"evidence":   "engine alert mapping for HOI4 1.19.3 and effective project localisation; not live trigger evaluation",
					"confidence": confidence, "exact_runtime_condition_available": false,
				})
			}
			usage["frame_usages"] = frames
			usage["mapping_game_version"] = "1.19.3"
		}
	}
	for name, value := range out {
		usage := value.(map[string]any)
		if file, ok := idx.ScreenPictureFile(name); ok {
			usage["screen_picture"] = map[string]any{"file": file, "role": "loading/main-menu background", "states": "no interaction states inferred"}
		}

		usage["usage_known"] = len(usage["frame_usages"].([]map[string]any)) > 0 || usage["screen_picture"] != nil
		usage["evidence_available"] = usage["usage_known"] == true || len(usage["references"].([]map[string]any)) > 0
	}
	return out
}
