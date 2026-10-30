package exportengine

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"path"
	"sort"
	"strings"

	"haganeworks/internal/hoi4/importsvc"
	"haganeworks/internal/hoi4/paradox"
)

func ReadGUISourceSnapshot(raw []byte) (map[string][]byte, error) {
	return readSafeZip(raw, guiSnapshotLimit)
}

const (
	guiPreviewFileLimit   = 100_000
	guiPreviewScriptLimit = int64(16 << 20)

	guiPreviewTextTotal   = int64(256 << 20)
	guiPreviewImagePrefix = int64(4 << 10)
	guiPreviewImageTotal  = int64(64 << 20)
	guiPreviewFontLimit   = int64(32 << 20)
	guiPreviewFontTotal   = int64(64 << 20)
)

func ReadGUIPreviewSnapshot(raw []byte) (map[string][]byte, error) {
	zr, err := zip.NewReader(bytes.NewReader(raw), int64(len(raw)))
	if err != nil {
		return nil, err
	}
	if len(zr.File) > guiPreviewFileLimit {
		return nil, fmt.Errorf("GUI snapshot has too many members")
	}
	out := map[string][]byte{}
	var expanded, retained, imageRetained, fontRetained int64
	for _, file := range zr.File {
		name := strings.ReplaceAll(file.Name, "\\", "/")
		if _, err := SafePath(name); err != nil {
			return nil, err
		}
		if file.FileInfo().IsDir() {
			continue
		}
		expanded += int64(file.UncompressedSize64)
		if expanded > guiSnapshotLimit {
			return nil, fmt.Errorf("expanded archive exceeds limit")
		}
		ext := strings.ToLower(path.Ext(name))
		limit := int64(0)
		switch ext {
		case ".gui", ".gfx", ".yml", ".fnt":
			if int64(file.UncompressedSize64) > guiPreviewScriptLimit {
				return nil, fmt.Errorf("GUI snapshot member %q exceeds text limit", name)
			}
			limit = int64(file.UncompressedSize64)
			retained += limit
			if retained > guiPreviewTextTotal {
				return nil, fmt.Errorf("GUI snapshot text exceeds limit")
			}
		case ".txt", ".csv":
			if int64(file.UncompressedSize64) > guiPreviewScriptLimit {
				continue
			}
			limit = int64(file.UncompressedSize64)
			retained += limit
			if retained > guiPreviewTextTotal {
				return nil, fmt.Errorf("GUI snapshot text exceeds limit")
			}
		case ".dds", ".png", ".tga":
			limit = minInt64(int64(file.UncompressedSize64), guiPreviewImagePrefix)
			imageRetained += limit
			if imageRetained > guiPreviewImageTotal {
				return nil, fmt.Errorf("GUI snapshot image index exceeds limit")
			}
		case ".ttf", ".otf":
			if int64(file.UncompressedSize64) > guiPreviewFontLimit {
				return nil, fmt.Errorf("GUI snapshot font %q exceeds font limit", name)
			}
			limit = int64(file.UncompressedSize64)
			fontRetained += limit
			if fontRetained > guiPreviewFontTotal {
				return nil, fmt.Errorf("GUI snapshot fonts exceed limit")
			}
		default:
			continue
		}
		rc, err := file.Open()
		if err != nil {
			return nil, err
		}
		body, readErr := io.ReadAll(io.LimitReader(rc, limit))
		closeErr := rc.Close()
		if readErr != nil {
			return nil, readErr
		}
		if closeErr != nil {
			return nil, closeErr
		}
		out[name] = body
	}
	return out, nil
}

func minInt64(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}

func ApplyGUIPreviewEdits(source []byte, edits []map[string]any) ([]byte, error) {
	ast, _, err := paradox.Parse(string(source))
	if err != nil {
		return nil, err
	}
	for _, raw := range edits {
		target, _ := raw["target"].(string)
		op, _ := raw["op"].(string)
		payload, _ := raw["payload"].(map[string]any)
		if payload == nil {
			payload = map[string]any{}
		}
		node, parent, index := resolveGUINode(ast, target)
		switch op {
		case "add_node":
			if node == nil {
				continue
			}
			rawNode, ok := payload["node"].(map[string]any)
			if !ok {
				continue
			}
			child := guiEntryFromJSON(rawNode)
			ensureGUIPreviewNodeName(node, child)
			insert := len(node.Block.Entries)
			if wanted, ok := integerPayload(payload["index"]); ok {
				if wanted < 0 {
					wanted = 0
				}
				seen := 0
				for i, entry := range node.Block.Entries {
					if entry.Block != nil && entry.Key != nil && strings.HasSuffix(strings.ToLower(*entry.Key), "type") {
						if seen == wanted {
							insert = i
							break
						}
						seen++
					}
				}
			}
			node.Block.Entries = append(node.Block.Entries, nil)
			copy(node.Block.Entries[insert+1:], node.Block.Entries[insert:])
			node.Block.Entries[insert] = child
			continue
		case "delete_node":
			if node != nil && parent != nil {
				parent.Entries = append(parent.Entries[:index], parent.Entries[index+1:]...)
			}
			continue
		}
		if node == nil {
			continue
		}
		switch op {
		case "set_text":
			if text := guiOverlayTextValue(payload); guiOverlayTextSupplied(payload) {
				setGUIString(node, "text", text)
			}
		case "swap_sprite":
			if ref := guiPayloadString(payload, "gfx_ref"); ref != "" && !engineSwappedGUITarget(target) {
				setGUIImage(node, ref)
			}
		case "set_layout":
			setGUIPair(node, "position", "x", payload["x"], "y", payload["y"])
			setGUIPair(node, "size", "width", payload["width"], "height", payload["height"])
		case "hide_node":

			setGUIWord(node, "gameview_hidden", "yes")
		case "set_orientation":
			if value, exists := payload["value"]; exists && value != nil {
				setGUIString(node, "orientation", fmt.Sprint(value))
			}
		case "set_origo":
			value, exists := payload["value"]
			if !exists || value == nil {
				continue
			}
			text := fmt.Sprint(value)
			if text == "" {
				deleteGUIKey(node, "origo")
			} else {
				setGUIString(node, "origo", text)
			}
		}
	}
	return []byte(importsvc.ToScript(ast, 0) + "\n"), nil
}

func GUIOverlayPreviewLocalisations(edits []map[string]any) map[string]string {
	return GUIOverlayPreviewLocalisationsIn(edits, "en")
}

func GUIOverlayPreviewLocalisationsIn(edits []map[string]any, lang string) map[string]string {
	lang = strings.TrimSpace(lang)
	if lang == "" {
		lang = "en"
	}
	out := map[string]string{}
	for _, raw := range edits {
		if strings.ToLower(strings.TrimSpace(guiPayloadString(raw, "op"))) != "set_text" {
			continue
		}
		payload, _ := raw["payload"].(map[string]any)
		key := strings.TrimSpace(guiPayloadString(payload, "key"))
		if key == "" {
			continue
		}
		value := guiOverlayLocalizationValue(payload, lang)
		if value == "" {
			continue
		}
		emitKey := key
		if isNew, supplied := payload["is_new"].(bool); !supplied || isNew {
			emitKey = "GUISUB_" + key
		}
		out[emitKey] = value
	}
	return out
}

func guiPreviewLangText(payload map[string]any) (string, bool) {
	texts, ok := payload["lang_texts"].(map[string]any)
	if !ok || len(texts) == 0 {
		return "", false
	}
	if value, exists := texts["en"]; exists && value != nil {
		return fmt.Sprint(value), true
	}

	keys := make([]string, 0, len(texts))
	for key := range texts {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		if texts[key] != nil {
			return fmt.Sprint(texts[key]), true
		}
	}
	return "", false
}

func integerPayload(value any) (int, bool) {
	switch v := value.(type) {
	case int:
		return v, true
	case int64:
		return int(v), true
	case float64:
		return int(v), true
	default:
		return 0, false
	}
}

func ensureGUIPreviewNodeName(parent, child *paradox.Entry) {
	if guiName(child) != "" {
		return
	}
	base := "node"
	if child != nil && child.Key != nil {
		base = strings.TrimSpace(*child.Key)
		if strings.HasSuffix(strings.ToLower(base), "type") {
			base = base[:len(base)-4]
		}
		if base == "" {
			base = "node"
		}
	}
	used := map[string]bool{}
	for _, sibling := range guiChildren(parent) {
		used[guiName(sibling)] = true
	}
	candidate := base
	for i := 1; used[candidate]; i++ {
		candidate = fmt.Sprintf("%s_%d", base, i)
	}
	setGUIString(child, "name", candidate)
}

func NormalizeGUISourcePath(value string) (string, error) {
	v := strings.TrimSpace(strings.ReplaceAll(value, "\\", "/"))
	for strings.Contains(v, "//") {
		v = strings.ReplaceAll(v, "//", "/")
	}
	if v == "" || strings.HasPrefix(v, "/") {
		return "", fmt.Errorf("invalid GUI source path")
	}
	if !strings.HasPrefix(strings.ToLower(v), "interface/") {
		v = "interface/" + v
	}
	if _, err := SafePath(v); err != nil {
		return "", fmt.Errorf("invalid GUI source path")
	}
	if strings.ToLower(path.Ext(v)) != ".gui" {
		return "", fmt.Errorf("GUI source path must end in .gui")
	}
	return v, nil
}
